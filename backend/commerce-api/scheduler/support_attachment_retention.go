package scheduler

import (
	"context"
	"log"
	"time"
)

type SupportAttachmentRetentionEnqueuer interface {
	Enqueue(
		ctx context.Context,
		batchSize int,
	) error
}

type SupportAttachmentRetentionQueue struct {
	enqueuer SupportAttachmentRetentionEnqueuer

	interval time.Duration

	batchSize int

	logger *log.Logger
}

func NewSupportAttachmentRetentionQueue(
	enqueuer SupportAttachmentRetentionEnqueuer,
	interval time.Duration,
	batchSize int,
	logger *log.Logger,
) *SupportAttachmentRetentionQueue {
	if interval <= 0 {
		interval =
			time.Hour
	}

	if batchSize <= 0 {
		batchSize =
			100
	}

	if logger == nil {
		logger =
			log.Default()
	}

	return &SupportAttachmentRetentionQueue{
		enqueuer: enqueuer,

		interval: interval,

		batchSize: batchSize,

		logger: logger,
	}
}

func (q *SupportAttachmentRetentionQueue) Run(
	ctx context.Context,
) {
	if q == nil ||
		q.enqueuer == nil {

		if q != nil &&
			q.logger != nil {

			q.logger.Printf(
				"support attachment retention scheduler is not configured",
			)
		}

		return
	}

	if ctx.Err() != nil {
		return
	}

	/*
		Run once at scheduler startup.

		The queue dedupe key prevents scheduler replicas or rapid
		restarts from producing duplicate retention work.
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

func (q *SupportAttachmentRetentionQueue) enqueue(
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
			"enqueue support attachment retention job failed: %v",
			err,
		)
	}
}
