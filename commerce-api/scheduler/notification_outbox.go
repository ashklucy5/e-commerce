package scheduler

import (
	"context"
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
	if ctx.Err() != nil {
		return
	}

	if err :=
		q.enqueuer.Enqueue(
			ctx,
			q.batchSize,
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
