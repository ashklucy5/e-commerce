package scheduler

import (
	"context"
	"fmt"
	"log"
	"time"
)

type NotificationOutboxEnqueuer interface {
	Enqueue(
		ctx context.Context,
		batchSize int,
	) error
}

type NotificationOutboxQueue struct {
	enqueuer NotificationOutboxEnqueuer

	interval time.Duration

	batchSize int

	logger *log.Logger
}

func NewNotificationOutboxQueue(
	enqueuer NotificationOutboxEnqueuer,
	interval time.Duration,
	batchSize int,
	logger *log.Logger,
) *NotificationOutboxQueue {
	if logger == nil {
		logger =
			log.Default()
	}

	return &NotificationOutboxQueue{
		enqueuer: enqueuer,

		interval: interval,

		batchSize: batchSize,

		logger: logger,
	}
}

// RunOnce performs one notification-outbox scheduling pass.
//
// It is used by finite/serverless runtimes. The normal scheduler
// continues to use Run, which calls this method repeatedly.
func (q *NotificationOutboxQueue) RunOnce(
	ctx context.Context,
) error {
	if q == nil {
		return fmt.Errorf(
			"notification outbox scheduler is required",
		)
	}

	if q.enqueuer == nil {
		return fmt.Errorf(
			"notification outbox scheduler has no queue enqueuer",
		)
	}

	if q.batchSize <= 0 {
		return fmt.Errorf(
			"notification outbox scheduler batch size must be greater than zero",
		)
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	return q.enqueuer.Enqueue(
		ctx,
		q.batchSize,
	)
}

func (q *NotificationOutboxQueue) Run(
	ctx context.Context,
) {
	if q == nil ||
		q.enqueuer == nil ||
		q.interval <= 0 ||
		q.batchSize <= 0 {

		if q != nil &&
			q.logger != nil {

			q.logger.Printf(
				"notification outbox scheduler is not configured",
			)
		}

		return
	}

	if ctx.Err() != nil {
		return
	}

	/*
		Run once immediately.

		This avoids waiting for the first interval after a
		scheduler restart while notifications may already be
		pending in PostgreSQL.
	*/
	q.enqueue(
		ctx,
	)

	ticker :=
		time.NewTicker(
			q.interval,
		)

	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			q.enqueue(
				ctx,
			)
		}
	}
}

func (q *NotificationOutboxQueue) enqueue(
	ctx context.Context,
) {
	if err :=
		q.RunOnce(
			ctx,
		); err != nil {

		if ctx.Err() != nil {
			return
		}

		q.logger.Printf(
			"enqueue notification outbox job failed: %v",
			err,
		)
	}
}
