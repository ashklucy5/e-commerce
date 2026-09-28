package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"project.local/commerce-api/internal/platform/queue"
	"project.local/commerce-api/internal/supportattachment"
)

const SupportAttachmentRetentionJobType = "support.attachment_retention"

const maxSupportAttachmentRetentionBatchesPerJob = 4

type SupportAttachmentRetentionRunner interface {
	Run(
		ctx context.Context,
		now time.Time,
		limit int,
	) (
		supportattachment.RetentionResult,
		error,
	)
}

type SupportAttachmentRetentionPayload struct {
	BatchSize int `json:"batch_size"`
}

type SupportAttachmentRetentionEnqueuer struct {
	producer *queue.Producer

	dedupeTTL time.Duration
}

type SupportAttachmentRetentionHandler struct {
	retention SupportAttachmentRetentionRunner
}

func NewSupportAttachmentRetentionEnqueuer(
	producer *queue.Producer,
	dedupeTTL time.Duration,
) *SupportAttachmentRetentionEnqueuer {
	if dedupeTTL <= 0 {
		dedupeTTL =
			45 * time.Minute
	}

	return &SupportAttachmentRetentionEnqueuer{
		producer: producer,

		dedupeTTL: dedupeTTL,
	}
}

func (e *SupportAttachmentRetentionEnqueuer) Enqueue(
	ctx context.Context,
	batchSize int,
) error {
	if e == nil ||
		e.producer == nil {

		return fmt.Errorf(
			"support attachment retention queue producer is not configured",
		)
	}

	batchSize =
		normalizeSupportAttachmentRetentionBatchSize(
			batchSize,
		)

	_, _, err :=
		e.producer.Enqueue(
			ctx,
			SupportAttachmentRetentionJobType,
			SupportAttachmentRetentionPayload{
				BatchSize: batchSize,
			},
			queue.EnqueueOptions{
				MaxAttempts: 5,

				DedupeKey: "scheduler:support-attachment-retention",

				DedupeTTL: e.dedupeTTL,
			},
		)
	if err != nil {
		return fmt.Errorf(
			"enqueue support attachment retention: %w",
			err,
		)
	}

	return nil
}

func NewSupportAttachmentRetentionHandler(
	retention SupportAttachmentRetentionRunner,
) *SupportAttachmentRetentionHandler {
	return &SupportAttachmentRetentionHandler{
		retention: retention,
	}
}

func (h *SupportAttachmentRetentionHandler) Handle(
	ctx context.Context,
	message queue.Message,
) error {
	if h == nil ||
		h.retention == nil {

		return fmt.Errorf(
			"support attachment retention handler is not configured",
		)
	}

	var payload SupportAttachmentRetentionPayload

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
					"decode support attachment retention payload: %w",
					err,
				),
			)
		}
	}

	if payload.BatchSize < 0 {
		return queue.Permanent(
			fmt.Errorf(
				"support attachment retention batch size cannot be negative",
			),
		)
	}

	batchSize :=
		normalizeSupportAttachmentRetentionBatchSize(
			payload.BatchSize,
		)

	for batch :=
		0; batch <
		maxSupportAttachmentRetentionBatchesPerJob; batch++ {

		result, err :=
			h.retention.Run(
				ctx,
				time.Now().
					UTC(),
				batchSize,
			)
		if err != nil {
			return fmt.Errorf(
				"run support attachment retention: %w",
				err,
			)
		}

		/*
			Each retention pass independently scans the
			closed-case set and the orphan set.

			If both scans returned fewer than the requested
			batch size, there is no immediate backlog left.
		*/
		if result.ClosedCandidates <
			batchSize &&
			result.OrphanCandidates <
				batchSize {

			break
		}
	}

	return nil
}

func normalizeSupportAttachmentRetentionBatchSize(
	batchSize int,
) int {
	if batchSize <= 0 {
		return supportattachment.
			DefaultRetentionBatchSize
	}

	if batchSize >
		supportattachment.
			MaxRetentionBatchSize {

		return supportattachment.
			MaxRetentionBatchSize
	}

	return batchSize
}
