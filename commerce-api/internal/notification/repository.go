package notification

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(
	db *pgxpool.Pool,
) *Repository {
	return &Repository{
		db: db,
	}
}

const outboxColumns = `
	id::text,
	dedupe_key,
	category,
	event_type,
	channel,
	COALESCE(customer_id::text, ''),
	COALESCE(order_id::text, ''),
	COALESCE(case_id::text, ''),
	COALESCE(recipient, ''),
	template_key,
	payload,
	status,
	attempt_count,
	max_attempts,
	available_at,
	locked_at,
	processed_at,
	COALESCE(provider_message_id, ''),
	COALESCE(last_error, ''),
	COALESCE(skip_reason, ''),
	created_at,
	updated_at
`

type rowScanner interface {
	Scan(dest ...any) error
}

func scanOutboxItem(
	row rowScanner,
) (
	OutboxItem,
	error,
) {
	var result OutboxItem

	var payload []byte

	var lockedAt pgtype.Timestamptz
	var processedAt pgtype.Timestamptz

	err :=
		row.Scan(
			&result.ID,
			&result.DedupeKey,
			&result.Category,
			&result.EventType,
			&result.Channel,
			&result.CustomerID,
			&result.OrderID,
			&result.CaseID,
			&result.Recipient,
			&result.TemplateKey,
			&payload,
			&result.Status,
			&result.AttemptCount,
			&result.MaxAttempts,
			&result.AvailableAt,
			&lockedAt,
			&processedAt,
			&result.ProviderMessageID,
			&result.LastError,
			&result.SkipReason,
			&result.CreatedAt,
			&result.UpdatedAt,
		)
	if err != nil {
		return OutboxItem{},
			err
	}

	result.Payload =
		append(
			result.Payload[:0],
			payload...,
		)

	if lockedAt.Valid {
		value :=
			lockedAt.Time.UTC()

		result.LockedAt =
			&value
	}

	if processedAt.Valid {
		value :=
			processedAt.Time.UTC()

		result.ProcessedAt =
			&value
	}

	return result,
		nil
}

func (r *Repository) Enqueue(
	ctx context.Context,
	request EnqueueRequest,
	payload []byte,
) (
	OutboxItem,
	bool,
	error,
) {
	tx, err :=
		r.db.Begin(
			ctx,
		)
	if err != nil {
		return OutboxItem{},
			false,
			fmt.Errorf(
				"begin notification outbox transaction: %w",
				err,
			)
	}

	defer func() {
		_ = tx.Rollback(
			ctx,
		)
	}()

	item, inserted, err :=
		r.EnqueueTx(
			ctx,
			tx,
			request,
			payload,
		)
	if err != nil {
		return OutboxItem{},
			false,
			err
	}

	if err :=
		tx.Commit(
			ctx,
		); err != nil {

		return OutboxItem{},
			false,
			fmt.Errorf(
				"commit notification outbox transaction: %w",
				err,
			)
	}

	return item,
		inserted,
		nil
}

func (r *Repository) EnqueueTx(
	ctx context.Context,
	tx pgx.Tx,
	request EnqueueRequest,
	payload []byte,
) (
	OutboxItem,
	bool,
	error,
) {
	query :=
		`
			INSERT INTO notification_outbox (
				dedupe_key,
				category,
				event_type,
				channel,

				customer_id,
				order_id,
				case_id,

				recipient,

				template_key,
				payload,

				status,
				attempt_count,
				max_attempts,

				available_at,

				created_at,
				updated_at
			)

			VALUES (
				$1,
				$2,
				$3,
				$4,

				NULLIF($5, '')::uuid,
				NULLIF($6, '')::uuid,
				NULLIF($7, '')::uuid,

				NULLIF($8, ''),

				$9,
				$10::jsonb,

				'pending',
				0,
				$11,

				$12,

				now(),
				now()
			)

			ON CONFLICT (
				dedupe_key
			)

			DO NOTHING

			RETURNING
		` +
			outboxColumns

	item, err :=
		scanOutboxItem(
			tx.QueryRow(
				ctx,
				query,
				request.DedupeKey,
				request.Category,
				request.EventType,
				request.Channel,
				request.CustomerID,
				request.OrderID,
				request.CaseID,
				request.Recipient,
				request.TemplateKey,
				payload,
				request.MaxAttempts,
				request.AvailableAt,
			),
		)

	if err == nil {
		return item,
			true,
			nil
	}

	if !errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return OutboxItem{},
			false,
			fmt.Errorf(
				"insert notification outbox item: %w",
				err,
			)
	}

	item, err =
		r.GetByDedupeKeyTx(
			ctx,
			tx,
			request.DedupeKey,
		)
	if err != nil {
		return OutboxItem{},
			false,
			err
	}

	return item,
		false,
		nil
}

func (r *Repository) GetByDedupeKeyTx(
	ctx context.Context,
	tx pgx.Tx,
	dedupeKey string,
) (
	OutboxItem,
	error,
) {
	item, err :=
		scanOutboxItem(
			tx.QueryRow(
				ctx,
				`
					SELECT
				`+
					outboxColumns+
					`
					FROM notification_outbox

					WHERE
						dedupe_key = $1
				`,
				dedupeKey,
			),
		)

	if errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return OutboxItem{},
			ErrOutboxNotFound
	}

	if err != nil {
		return OutboxItem{},
			fmt.Errorf(
				"get notification outbox item by dedupe key: %w",
				err,
			)
	}

	return item,
		nil
}
