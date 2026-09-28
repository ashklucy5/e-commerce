package queue

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"

	platformsecurity "project.local/commerce-api/internal/platform/security"
)

type Handler func(
	ctx context.Context,
	message Message,
) error

type ConsumerConfig struct {
	Queue Config

	ConsumerName string
	Concurrency  int
	Prefetch     int64

	Block             time.Duration
	HandlerTimeout    time.Duration
	ClaimMinIdle      time.Duration
	ClaimInterval     time.Duration
	RetryPollInterval time.Duration

	Logger *log.Logger
}

type Consumer struct {
	redis *redis.Client

	producer *Producer

	config ConsumerConfig
}

var ackDeleteScript = redis.NewScript(
	`
local acknowledged = redis.call(
  'XACK',
  KEYS[1],
  ARGV[1],
  ARGV[2]
)

if acknowledged > 0 then
  redis.call(
    'XDEL',
    KEYS[1],
    ARGV[2]
  )
end

return acknowledged
`,
)

func DefaultConsumerConfig(
	queueConfig Config,
) ConsumerConfig {
	return ConsumerConfig{
		Queue: queueConfig,

		Concurrency: 4,

		Prefetch: 16,

		Block: 2 * time.Second,

		HandlerTimeout: 2 * time.Minute,

		ClaimMinIdle: 5 * time.Minute,

		ClaimInterval: 30 * time.Second,

		RetryPollInterval: time.Second,

		Logger: log.Default(),
	}
}

func NewConsumer(
	client *redis.Client,
	producer *Producer,
	cfg ConsumerConfig,
) (
	*Consumer,
	error,
) {
	if client == nil {
		return nil,
			fmt.Errorf(
				"queue consumer requires Redis",
			)
	}

	if producer == nil {
		return nil,
			fmt.Errorf(
				"queue consumer requires producer",
			)
	}

	cfg.Queue =
		normalizeConfig(
			cfg.Queue,
		)

	if cfg.Concurrency <= 0 {
		cfg.Concurrency =
			4
	}

	if cfg.Concurrency > 64 {
		return nil,
			fmt.Errorf(
				"queue consumer concurrency cannot exceed 64",
			)
	}

	if cfg.Prefetch <= 0 {
		cfg.Prefetch =
			16
	}

	if cfg.Prefetch > 256 {
		return nil,
			fmt.Errorf(
				"queue consumer prefetch cannot exceed 256",
			)
	}

	if cfg.Block <= 0 {
		cfg.Block =
			2 * time.Second
	}

	if cfg.HandlerTimeout <= 0 {
		cfg.HandlerTimeout =
			2 * time.Minute
	}

	if cfg.ClaimMinIdle <=
		cfg.HandlerTimeout {

		cfg.ClaimMinIdle =
			cfg.HandlerTimeout +
				time.Minute
	}

	if cfg.ClaimInterval <= 0 {
		cfg.ClaimInterval =
			30 * time.Second
	}

	if cfg.RetryPollInterval <=
		0 {

		cfg.RetryPollInterval =
			time.Second
	}

	if cfg.Logger == nil {
		cfg.Logger =
			log.Default()
	}

	cfg.ConsumerName =
		strings.TrimSpace(
			cfg.ConsumerName,
		)

	if cfg.ConsumerName == "" {
		name, err :=
			defaultConsumerName()
		if err != nil {
			return nil, err
		}

		cfg.ConsumerName =
			name
	}

	return &Consumer{
			redis: client,

			producer: producer,

			config: cfg,
		},
		nil
}

func (c *Consumer) Run(
	ctx context.Context,
	handler Handler,
) error {
	if handler == nil {
		return fmt.Errorf(
			"queue handler is required",
		)
	}

	if err :=
		c.ensureGroup(
			ctx,
		); err != nil {

		return err
	}

	bufferSize :=
		int(
			c.config.Prefetch,
		) *
			c.config.Concurrency

	if bufferSize <
		c.config.Concurrency {

		bufferSize =
			c.config.Concurrency
	}

	deliveries :=
		make(
			chan redis.XMessage,
			bufferSize,
		)

	var feederWG sync.WaitGroup

	feederWG.Add(
		2,
	)

	go func() {
		defer feederWG.Done()

		c.readLoop(
			ctx,
			deliveries,
		)
	}()

	go func() {
		defer feederWG.Done()

		c.claimLoop(
			ctx,
			deliveries,
		)
	}()

	var retryWG sync.WaitGroup

	retryWG.Add(
		1,
	)

	go func() {
		defer retryWG.Done()

		c.retryLoop(
			ctx,
		)
	}()

	var workerWG sync.WaitGroup

	workerWG.Add(
		c.config.Concurrency,
	)

	for index :=
		0; index <
		c.config.Concurrency; index++ {

		go func() {
			defer workerWG.Done()

			c.workerLoop(
				ctx,
				deliveries,
				handler,
			)
		}()
	}

	<-ctx.Done()

	feederWG.Wait()

	close(
		deliveries,
	)

	workerWG.Wait()
	retryWG.Wait()

	return nil
}

func (c *Consumer) ensureGroup(
	ctx context.Context,
) error {
	err :=
		c.redis.XGroupCreateMkStream(
			ctx,
			c.config.Queue.Stream,
			c.config.Queue.Group,
			"0",
		).Err()

	if err == nil {
		return nil
	}

	if strings.Contains(
		strings.ToUpper(
			err.Error(),
		),
		"BUSYGROUP",
	) {
		return nil
	}

	return fmt.Errorf(
		"create queue consumer group: %w",
		err,
	)
}

func (c *Consumer) readLoop(
	ctx context.Context,
	deliveries chan<- redis.XMessage,
) {
	for {
		if ctx.Err() != nil {
			return
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

					Count: c.config.
						Prefetch,

					Block: c.config.
						Block,
				},
			).Result()

		if err != nil {
			if errors.Is(
				err,
				redis.Nil,
			) {
				continue
			}

			if ctx.Err() != nil {
				return
			}

			c.config.Logger.Printf(
				"queue read error: %v",
				err,
			)

			if !waitContext(
				ctx,
				time.Second,
			) {
				return
			}

			continue
		}

		for _, stream := range streams {

			for _, message := range stream.Messages {

				select {
				case deliveries <- message:

				case <-ctx.Done():
					return
				}
			}
		}
	}
}

func (c *Consumer) claimLoop(
	ctx context.Context,
	deliveries chan<- redis.XMessage,
) {
	c.claimStale(
		ctx,
		deliveries,
	)

	ticker :=
		time.NewTicker(
			c.config.
				ClaimInterval,
		)

	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			c.claimStale(
				ctx,
				deliveries,
			)
		}
	}
}

func (c *Consumer) claimStale(
	ctx context.Context,
	deliveries chan<- redis.XMessage,
) {
	count :=
		c.config.
			Prefetch *
			4

	if count < 64 {
		count = 64
	}

	if count > 512 {
		count = 512
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

				Count: count,
			},
		).Result()
	if err != nil {
		if !errors.Is(
			err,
			redis.Nil,
		) &&
			ctx.Err() == nil {

			c.config.Logger.Printf(
				"queue pending scan error: %v",
				err,
			)
		}

		return
	}

	ids :=
		make(
			[]string,
			0,
			len(
				pending,
			),
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
		return
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
		if ctx.Err() == nil {
			c.config.Logger.Printf(
				"queue stale-claim error: %v",
				err,
			)
		}

		return
	}

	for _, message := range messages {

		select {
		case deliveries <- message:

		case <-ctx.Done():
			return
		}
	}
}

func (c *Consumer) retryLoop(
	ctx context.Context,
) {
	c.promoteRetries(
		ctx,
	)

	ticker :=
		time.NewTicker(
			c.config.
				RetryPollInterval,
		)

	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			c.promoteRetries(
				ctx,
			)
		}
	}
}

func (c *Consumer) promoteRetries(
	ctx context.Context,
) {
	_, err :=
		c.producer.
			PromoteDueRetries(
				ctx,
				c.config.
					Prefetch*4,
			)

	if err != nil &&
		ctx.Err() == nil {

		c.config.Logger.Printf(
			"queue retry promotion error: %v",
			err,
		)
	}
}

func (c *Consumer) workerLoop(
	ctx context.Context,
	deliveries <-chan redis.XMessage,
	handler Handler,
) {
	for {
		select {
		case <-ctx.Done():
			return

		case input, ok :=
			<-deliveries:

			if !ok {
				return
			}

			c.processMessage(
				ctx,
				input,
				handler,
			)
		}
	}
}

func (c *Consumer) processMessage(
	ctx context.Context,
	input redis.XMessage,
	handler Handler,
) {
	message, err :=
		decodeMessage(
			input,
		)
	if err != nil {
		finalizeCtx, cancel :=
			context.WithTimeout(
				context.Background(),
				5*time.Second,
			)

		defer cancel()

		if deadErr :=
			c.producer.
				DeadLetterRaw(
					finalizeCtx,
					input.ID,
					input.Values,
					err,
				); deadErr != nil {

			c.config.Logger.Printf(
				"queue invalid-message dead-letter error: %v",
				deadErr,
			)

			return
		}

		if ackErr :=
			c.ackAndDelete(
				finalizeCtx,
				input.ID,
			); ackErr != nil {

			c.config.Logger.Printf(
				"queue invalid-message acknowledgement error: %v",
				ackErr,
			)
		}

		return
	}

	handlerCtx, cancel :=
		context.WithTimeout(
			ctx,
			c.config.
				HandlerTimeout,
		)

	handlerErr :=
		invokeHandler(
			handlerCtx,
			handler,
			message,
		)

	cancel()

	finalizeCtx,
		finalizeCancel :=
		context.WithTimeout(
			context.Background(),
			5*time.Second,
		)

	defer finalizeCancel()

	if handlerErr == nil {
		if err :=
			c.ackAndDelete(
				finalizeCtx,
				input.ID,
			); err != nil {

			c.config.Logger.Printf(
				"queue acknowledgement error job_id=%s type=%s: %v",
				message.ID,
				message.Type,
				err,
			)
		}

		return
	}

	if IsPermanent(
		handlerErr,
	) ||
		message.Attempt >=
			message.MaxAttempts {

		if err :=
			c.producer.
				DeadLetter(
					finalizeCtx,
					message,
					handlerErr,
				); err != nil {

			c.config.Logger.Printf(
				"queue dead-letter error job_id=%s type=%s: %v",
				message.ID,
				message.Type,
				err,
			)

			return
		}
	} else {
		if err :=
			c.producer.
				ScheduleRetry(
					finalizeCtx,
					message,
				); err != nil {

			c.config.Logger.Printf(
				"queue retry scheduling error job_id=%s type=%s attempt=%d: %v",
				message.ID,
				message.Type,
				message.Attempt,
				err,
			)

			return
		}
	}

	if err :=
		c.ackAndDelete(
			finalizeCtx,
			input.ID,
		); err != nil {

		c.config.Logger.Printf(
			"queue failed-job acknowledgement error job_id=%s type=%s: %v",
			message.ID,
			message.Type,
			err,
		)
	}
}

func (c *Consumer) ackAndDelete(
	ctx context.Context,
	streamID string,
) error {
	_, err :=
		ackDeleteScript.Run(
			ctx,
			c.redis,
			[]string{
				c.config.
					Queue.
					Stream,
			},
			c.config.
				Queue.
				Group,
			streamID,
		).Int64()
	if err != nil {
		return fmt.Errorf(
			"acknowledge queue message: %w",
			err,
		)
	}

	return nil
}

func invokeHandler(
	ctx context.Context,
	handler Handler,
	message Message,
) (
	err error,
) {
	defer func() {
		if recovered :=
			recover(); recovered != nil {

			err =
				fmt.Errorf(
					"queue handler panic: %v",
					recovered,
				)
		}
	}()

	return handler(
		ctx,
		message,
	)
}

func defaultConsumerName() (
	string,
	error,
) {
	hostname, err :=
		os.Hostname()
	if err != nil {
		hostname =
			"worker"
	}

	hostname =
		strings.TrimSpace(
			hostname,
		)

	if hostname == "" {
		hostname =
			"worker"
	}

	if len(hostname) > 48 {
		hostname =
			hostname[:48]
	}

	randomPart, err :=
		platformsecurity.RandomHex(
			4,
		)
	if err != nil {
		return "",
			fmt.Errorf(
				"generate queue consumer name: %w",
				err,
			)
	}

	return fmt.Sprintf(
			"%s-%d-%s",
			hostname,
			os.Getpid(),
			randomPart,
		),
		nil
}

func waitContext(
	ctx context.Context,
	delay time.Duration,
) bool {
	timer :=
		time.NewTimer(
			delay,
		)

	defer timer.Stop()

	select {
	case <-timer.C:
		return true

	case <-ctx.Done():
		return false
	}
}
