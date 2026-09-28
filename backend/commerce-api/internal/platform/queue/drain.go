package queue

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	defaultDrainBlock = 10 * time.Millisecond
	maxDrainMessages  = 256
)

// Drain processes a bounded number of currently available queue messages
// and then returns.
//
// Consumer.Run remains the normal continuous-worker path.
// Drain exists for finite/serverless execution environments.
func (c *Consumer) Drain(
	ctx context.Context,
	handler Handler,
	maxMessages int,
) (int, error) {
	if c == nil {
		return 0, fmt.Errorf("queue consumer is required")
	}

	if handler == nil {
		return 0, fmt.Errorf("queue handler is required")
	}

	if maxMessages <= 0 {
		return 0, fmt.Errorf(
			"queue drain max messages must be greater than zero",
		)
	}

	if maxMessages > maxDrainMessages {
		return 0, fmt.Errorf(
			"queue drain max messages cannot exceed %d",
			maxDrainMessages,
		)
	}

	if err := ctx.Err(); err != nil {
		return 0, err
	}

	if err := c.ensureGroup(ctx); err != nil {
		return 0, err
	}

	// Promote due retries once for this finite execution.
	c.promoteRetries(ctx)

	deliveries := make(
		[]redis.XMessage,
		0,
		maxMessages,
	)

	/*
		Recover stale pending messages first.

		This preserves the continuous worker's recovery behavior when
		a previous finite invocation received a message but terminated
		before acknowledging it.
	*/
	stale, err := c.claimStaleBatch(
		ctx,
		int64(maxMessages),
	)
	if err != nil {
		if ctx.Err() != nil {
			return 0, ctx.Err()
		}

		c.config.Logger.Printf(
			"queue bounded stale-claim error: %v",
			err,
		)
	} else {
		deliveries = append(
			deliveries,
			stale...,
		)
	}

	remaining :=
		maxMessages -
			len(deliveries)

	if remaining > 0 {
		fresh, err := c.readAvailableBatch(
			ctx,
			int64(remaining),
		)
		if err != nil {
			if ctx.Err() != nil {
				return 0, ctx.Err()
			}

			return 0, err
		}

		deliveries = append(
			deliveries,
			fresh...,
		)
	}

	if len(deliveries) == 0 {
		return 0, nil
	}

	return c.processBoundedBatch(
		ctx,
		deliveries,
		handler,
	)
}

func (c *Consumer) readAvailableBatch(
	ctx context.Context,
	limit int64,
) ([]redis.XMessage, error) {
	if limit <= 0 {
		return nil, nil
	}

	streams, err :=
		c.redis.XReadGroup(
			ctx,
			&redis.XReadGroupArgs{
				Group: c.config.
					Queue.
					Group,

				Consumer: c.config.
					ConsumerName,

				Streams: []string{
					c.config.
						Queue.
						Stream,
					">",
				},

				Count: limit,

				/*
					Do not wait like the continuous worker does.

					A very short block allows the Redis Stream request to
					complete cleanly while keeping the serverless execution
					bounded.
				*/
				Block: defaultDrainBlock,
			},
		).Result()

	if err != nil {
		if errors.Is(
			err,
			redis.Nil,
		) {
			return nil, nil
		}

		return nil,
			fmt.Errorf(
				"read bounded queue batch: %w",
				err,
			)
	}

	messages :=
		make(
			[]redis.XMessage,
			0,
			limit,
		)

	for _, stream := range streams {
		for _, message := range stream.Messages {
			messages =
				append(
					messages,
					message,
				)

			if int64(len(messages)) >=
				limit {

				return messages, nil
			}
		}
	}

	return messages, nil
}

func (c *Consumer) claimStaleBatch(
	ctx context.Context,
	limit int64,
) ([]redis.XMessage, error) {
	if limit <= 0 {
		return nil, nil
	}

	pending, err :=
		c.redis.XPendingExt(
			ctx,
			&redis.XPendingExtArgs{
				Stream: c.config.
					Queue.
					Stream,

				Group: c.config.
					Queue.
					Group,

				Idle: c.config.
					ClaimMinIdle,

				Start: "-",

				End: "+",

				Count: limit,
			},
		).Result()

	if err != nil {
		if errors.Is(
			err,
			redis.Nil,
		) {
			return nil, nil
		}

		return nil,
			fmt.Errorf(
				"scan stale queue deliveries: %w",
				err,
			)
	}

	ids :=
		make(
			[]string,
			0,
			len(pending),
		)

	for _, item := range pending {
		if item.Idle <
			c.config.
				ClaimMinIdle {

			continue
		}

		ids =
			append(
				ids,
				item.ID,
			)
	}

	if len(ids) == 0 {
		return nil, nil
	}

	messages, err :=
		c.redis.XClaim(
			ctx,
			&redis.XClaimArgs{
				Stream: c.config.
					Queue.
					Stream,

				Group: c.config.
					Queue.
					Group,

				Consumer: c.config.
					ConsumerName,

				MinIdle: c.config.
					ClaimMinIdle,

				Messages: ids,
			},
		).Result()

	if err != nil {
		return nil,
			fmt.Errorf(
				"claim stale queue deliveries: %w",
				err,
			)
	}

	return messages, nil
}

func (c *Consumer) processBoundedBatch(
	ctx context.Context,
	deliveries []redis.XMessage,
	handler Handler,
) (int, error) {
	if len(deliveries) == 0 {
		return 0, nil
	}

	concurrency :=
		c.config.
			Concurrency

	if concurrency <= 0 {
		concurrency = 1
	}

	if concurrency >
		len(deliveries) {

		concurrency =
			len(deliveries)
	}

	inputs :=
		make(
			chan redis.XMessage,
		)

	var workers sync.WaitGroup

	workers.Add(
		concurrency,
	)

	for index := 0; index < concurrency; index++ {
		go func() {
			defer workers.Done()

			for input := range inputs {
				c.processMessage(
					ctx,
					input,
					handler,
				)
			}
		}()
	}

	submitted := 0

	for _, input := range deliveries {
		select {
		case <-ctx.Done():
			close(inputs)

			workers.Wait()

			return submitted,
				ctx.Err()

		case inputs <- input:
			submitted++
		}
	}

	close(inputs)

	workers.Wait()

	if err := ctx.Err(); err != nil {
		return submitted, err
	}

	return submitted, nil
}
