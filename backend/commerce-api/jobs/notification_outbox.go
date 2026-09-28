package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"project.local/commerce-api/internal/notification"
	"project.local/commerce-api/internal/platform/queue"
)

const NotificationOutboxJobType = "notification.outbox"

const (
	defaultNotificationOutboxBatchSize = 100

	maxNotificationOutboxBatchSize = 1000

	// Prevent one queue delivery from monopolizing a worker
	// when a large notification backlog exists.
	maxNotificationOutboxBatchesPerJob = 4
)

type NotificationOutboxDispatcher interface {
	DispatchDue(
		ctx context.Context,
		now time.Time,
		limit int,
	) (
		notification.DispatchBatchResult,
		error,
	)
}

type NotificationOutboxPayload struct {
	BatchSize int `json:"batch_size"`
}

type NotificationOutboxEnqueuer struct {
	producer *queue.Producer

	dedupeTTL time.Duration
}

type NotificationOutboxHandler struct {
	dispatcher NotificationOutboxDispatcher
}

func NewNotificationOutboxEnqueuer(
	producer *queue.Producer,
	dedupeTTL time.Duration,
) *NotificationOutboxEnqueuer {
	if dedupeTTL <= 0 {
		dedupeTTL =
			10 * time.Second
	}

	return &NotificationOutboxEnqueuer{
		producer: producer,

		dedupeTTL: dedupeTTL,
	}
}

func (e *NotificationOutboxEnqueuer) Enqueue(
	ctx context.Context,
	batchSize int,
) error {
	if e == nil ||
		e.producer == nil {

		return fmt.Errorf(
			"notification outbox queue producer is not configured",
		)
	}

	batchSize =
		normalizeNotificationOutboxBatchSize(
			batchSize,
		)

	_, _, err :=
		e.producer.Enqueue(
			ctx,
			NotificationOutboxJobType,
			NotificationOutboxPayload{
				BatchSize: batchSize,
			},
			queue.EnqueueOptions{
				MaxAttempts: 5,

				DedupeKey: "scheduler:notification-outbox",

				DedupeTTL: e.dedupeTTL,
			},
		)
	if err != nil {
		return fmt.Errorf(
			"enqueue notification outbox dispatch: %w",
			err,
		)
	}

	return nil
}

func NewNotificationOutboxHandler(
	dispatcher NotificationOutboxDispatcher,
) *NotificationOutboxHandler {
	return &NotificationOutboxHandler{
		dispatcher: dispatcher,
	}
}

func (h *NotificationOutboxHandler) Handle(
	ctx context.Context,
	message queue.Message,
) error {
	if h == nil ||
		h.dispatcher == nil {

		return fmt.Errorf(
			"notification outbox handler is not configured",
		)
	}

	var payload NotificationOutboxPayload

	if len(
		message.Payload,
	) > 0 {

		if err :=
			json.Unmarshal(
				message.Payload,
				&payload,
			); err != nil {

			return queue.Permanent(
				fmt.Errorf(
					"decode notification outbox payload: %w",
					err,
				),
			)
		}
	}

	if payload.BatchSize < 0 {
		return queue.Permanent(
			fmt.Errorf(
				"notification outbox batch size cannot be negative",
			),
		)
	}

	batchSize :=
		normalizeNotificationOutboxBatchSize(
			payload.BatchSize,
		)

	for batch :=
		0; batch <
		maxNotificationOutboxBatchesPerJob; batch++ {

		result, err :=
			h.dispatcher.DispatchDue(
				ctx,
				time.Now().
					UTC(),
				batchSize,
			)
		if err != nil {
			return fmt.Errorf(
				"dispatch notification outbox: %w",
				err,
			)
		}

		// If we claimed fewer rows than the batch limit,
		// there is no immediately-due backlog left.
		if result.Scanned <
			batchSize {

			break
		}
	}

	return nil
}

func normalizeNotificationOutboxBatchSize(
	batchSize int,
) int {
	if batchSize <= 0 {
		return defaultNotificationOutboxBatchSize
	}

	if batchSize >
		maxNotificationOutboxBatchSize {

		return maxNotificationOutboxBatchSize
	}

	return batchSize
}
