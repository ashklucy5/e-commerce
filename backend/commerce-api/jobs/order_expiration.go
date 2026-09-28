package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"project.local/commerce-api/internal/platform/queue"
)

const OrderExpirationJobType = "order.payment_expiration"

const (
	defaultOrderExpirationBatchSize = 100

	maxOrderExpirationBatchSize = 1000

	// A single queue delivery is intentionally prevented from
	// monopolizing the database indefinitely.
	//
	// At the default batch size this drains up to 400 overdue
	// orders before yielding back to the worker pool.
	maxOrderExpirationBatchesPerJob = 4
)

type PaymentExpirer interface {
	ExpirePendingPayments(
		ctx context.Context,
		now time.Time,
		limit int,
	) (
		int,
		error,
	)
}

type OrderExpirationPayload struct {
	BatchSize int `json:"batch_size"`
}

type OrderExpirationEnqueuer struct {
	producer *queue.Producer

	dedupeTTL time.Duration
}

type OrderExpirationHandler struct {
	expirer PaymentExpirer
}

func NewOrderExpirationEnqueuer(
	producer *queue.Producer,
	dedupeTTL time.Duration,
) *OrderExpirationEnqueuer {
	if dedupeTTL <= 0 {
		dedupeTTL =
			20 * time.Second
	}

	return &OrderExpirationEnqueuer{
		producer: producer,

		dedupeTTL: dedupeTTL,
	}
}

func (e *OrderExpirationEnqueuer) EnqueuePaymentExpiry(
	ctx context.Context,
	batchSize int,
) error {
	if e == nil ||
		e.producer == nil {

		return fmt.Errorf(
			"order expiration queue producer is not configured",
		)
	}

	batchSize =
		normalizeOrderExpirationBatchSize(
			batchSize,
		)

	_, _, err :=
		e.producer.Enqueue(
			ctx,
			OrderExpirationJobType,
			OrderExpirationPayload{
				BatchSize: batchSize,
			},
			queue.EnqueueOptions{
				MaxAttempts: 5,

				DedupeKey: "scheduler:order-payment-expiration",

				DedupeTTL: e.dedupeTTL,
			},
		)
	if err != nil {
		return fmt.Errorf(
			"enqueue order payment expiration: %w",
			err,
		)
	}

	return nil
}

func NewOrderExpirationHandler(
	expirer PaymentExpirer,
) *OrderExpirationHandler {
	return &OrderExpirationHandler{
		expirer: expirer,
	}
}

func (h *OrderExpirationHandler) Handle(
	ctx context.Context,
	message queue.Message,
) error {
	if h == nil ||
		h.expirer == nil {

		return fmt.Errorf(
			"order expiration handler is not configured",
		)
	}

	var payload OrderExpirationPayload

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
					"decode order expiration payload: %w",
					err,
				),
			)
		}
	}

	if payload.BatchSize < 0 {
		return queue.Permanent(
			fmt.Errorf(
				"order expiration batch size cannot be negative",
			),
		)
	}

	batchSize :=
		normalizeOrderExpirationBatchSize(
			payload.BatchSize,
		)

	for batch :=
		0; batch <
		maxOrderExpirationBatchesPerJob; batch++ {

		expired, err :=
			h.expirer.
				ExpirePendingPayments(
					ctx,
					time.Now().
						UTC(),
					batchSize,
				)
		if err != nil {
			return fmt.Errorf(
				"expire pending payment orders: %w",
				err,
			)
		}

		if expired <
			batchSize {

			break
		}
	}

	return nil
}

func normalizeOrderExpirationBatchSize(
	batchSize int,
) int {
	if batchSize <= 0 {
		return defaultOrderExpirationBatchSize
	}

	if batchSize >
		maxOrderExpirationBatchSize {

		return maxOrderExpirationBatchSize
	}

	return batchSize
}
