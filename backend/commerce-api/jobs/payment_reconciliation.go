package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"project.local/commerce-api/internal/payment"
	"project.local/commerce-api/internal/platform/queue"
)

const PaymentReconciliationJobType = "payment.reconciliation"

const (
	defaultPaymentReconciliationBatchSize = 100

	maxPaymentReconciliationBatchSize = 1000

	// Prevent one queue delivery from monopolizing a worker
	// indefinitely when a large payment backlog exists.
	maxPaymentReconciliationBatchesPerJob = 4
)

type PaymentReconciler interface {
	ReconcilePending(
		ctx context.Context,
		now time.Time,
		limit int,
	) (
		payment.ReconciliationBatchResult,
		error,
	)
}

type PaymentReconciliationPayload struct {
	BatchSize int `json:"batch_size"`
}

type PaymentReconciliationEnqueuer struct {
	producer *queue.Producer

	dedupeTTL time.Duration
}

type PaymentReconciliationHandler struct {
	reconciler PaymentReconciler
}

func NewPaymentReconciliationEnqueuer(
	producer *queue.Producer,
	dedupeTTL time.Duration,
) *PaymentReconciliationEnqueuer {
	if dedupeTTL <= 0 {
		dedupeTTL =
			20 * time.Second
	}

	return &PaymentReconciliationEnqueuer{
		producer: producer,

		dedupeTTL: dedupeTTL,
	}
}

func (e *PaymentReconciliationEnqueuer) Enqueue(
	ctx context.Context,
	batchSize int,
) error {
	if e == nil ||
		e.producer == nil {

		return fmt.Errorf(
			"payment reconciliation queue producer is not configured",
		)
	}

	batchSize =
		normalizePaymentReconciliationBatchSize(
			batchSize,
		)

	_, _, err :=
		e.producer.Enqueue(
			ctx,
			PaymentReconciliationJobType,
			PaymentReconciliationPayload{
				BatchSize: batchSize,
			},
			queue.EnqueueOptions{
				MaxAttempts: 5,

				DedupeKey: "scheduler:payment-reconciliation",

				DedupeTTL: e.dedupeTTL,
			},
		)
	if err != nil {
		return fmt.Errorf(
			"enqueue payment reconciliation: %w",
			err,
		)
	}

	return nil
}

func NewPaymentReconciliationHandler(
	reconciler PaymentReconciler,
) *PaymentReconciliationHandler {
	return &PaymentReconciliationHandler{
		reconciler: reconciler,
	}
}

func (h *PaymentReconciliationHandler) Handle(
	ctx context.Context,
	message queue.Message,
) error {
	if h == nil ||
		h.reconciler == nil {

		return fmt.Errorf(
			"payment reconciliation handler is not configured",
		)
	}

	var payload PaymentReconciliationPayload

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
					"decode payment reconciliation payload: %w",
					err,
				),
			)
		}
	}

	if payload.BatchSize < 0 {
		return queue.Permanent(
			fmt.Errorf(
				"payment reconciliation batch size cannot be negative",
			),
		)
	}

	batchSize :=
		normalizePaymentReconciliationBatchSize(
			payload.BatchSize,
		)

	for batch :=
		0; batch <
		maxPaymentReconciliationBatchesPerJob; batch++ {

		result, err :=
			h.reconciler.
				ReconcilePending(
					ctx,
					time.Now().
						UTC(),
					batchSize,
				)
		if err != nil {
			return fmt.Errorf(
				"reconcile pending payments: %w",
				err,
			)
		}

		if result.Scanned <
			batchSize {

			break
		}
	}

	return nil
}

func normalizePaymentReconciliationBatchSize(
	batchSize int,
) int {
	if batchSize <= 0 {
		return defaultPaymentReconciliationBatchSize
	}

	if batchSize >
		maxPaymentReconciliationBatchSize {

		return maxPaymentReconciliationBatchSize
	}

	return batchSize
}
