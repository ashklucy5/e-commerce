package payment

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type reconciliationCandidate struct {
	ID string

	OrderID string

	Provider string

	Status string

	Amount int64

	Currency string

	AttemptKey string

	ProviderPaymentID string

	ProviderTransactionID string

	CreatedAt time.Time

	UpdatedAt time.Time
}

type reconciliationTerminalUpdate struct {
	PaymentID string

	Status string

	ProviderPaymentID string

	ProviderTransactionID string

	FailureCode string

	FailureMessage string

	EventID string

	EventType string

	PayloadSHA256 string
}

func (r *Repository) ListPendingReconciliationCandidates(
	ctx context.Context,
	before time.Time,
	limit int,
) (
	[]reconciliationCandidate,
	error,
) {
	if limit <= 0 {
		limit = 100
	}

	if limit > 1000 {
		limit = 1000
	}

	rows, err :=
		r.db.Query(
			ctx,
			`
				SELECT
					id::text,
					order_id::text,
					provider,
					status,
					amount,
					currency,

					COALESCE(
						attempt_key,
						''
					),

					COALESCE(
						provider_payment_id,
						''
					),

					COALESCE(
						provider_transaction_id,
						''
					),

					created_at,
					updated_at

				FROM payments

				WHERE
					status = 'pending'

					AND (
						provider_payment_id IS NOT NULL
						OR attempt_key IS NOT NULL
					)

					AND updated_at <= $1

				ORDER BY
					updated_at,
					id

				LIMIT $2
			`,
			before,
			limit,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"list payment reconciliation candidates: %w",
				err,
			)
	}

	defer rows.Close()

	result :=
		make(
			[]reconciliationCandidate,
			0,
			limit,
		)

	for rows.Next() {
		var candidate reconciliationCandidate

		if err :=
			rows.Scan(
				&candidate.ID,
				&candidate.OrderID,
				&candidate.Provider,
				&candidate.Status,
				&candidate.Amount,
				&candidate.Currency,
				&candidate.AttemptKey,
				&candidate.ProviderPaymentID,
				&candidate.ProviderTransactionID,
				&candidate.CreatedAt,
				&candidate.UpdatedAt,
			); err != nil {

			return nil,
				fmt.Errorf(
					"scan payment reconciliation candidate: %w",
					err,
				)
		}

		result =
			append(
				result,
				candidate,
			)
	}

	if err :=
		rows.Err(); err != nil {

		return nil,
			fmt.Errorf(
				"iterate payment reconciliation candidates: %w",
				err,
			)
	}

	return result,
		nil
}

func (r *Repository) DeferReconciliation(
	ctx context.Context,
	paymentID string,
) error {
	_, err :=
		r.db.Exec(
			ctx,
			`
				UPDATE payments

				SET
					updated_at = now()

				WHERE
					id = $1::uuid
					AND status = 'pending'
			`,
			paymentID,
		)
	if err != nil {
		return fmt.Errorf(
			"defer payment reconciliation: %w",
			err,
		)
	}

	return nil
}

func (r *Repository) BindReconciliationProviderIdentity(
	ctx context.Context,
	paymentID string,
	providerPaymentID string,
	providerTransactionID string,
) error {
	tag, err :=
		r.db.Exec(
			ctx,
			`
				UPDATE payments

				SET
					provider_payment_id =
						CASE
							WHEN NULLIF($2, '') IS NULL
								THEN provider_payment_id

							ELSE COALESCE(
								provider_payment_id,
								NULLIF($2, '')
							)
						END,

					provider_transaction_id =
						COALESCE(
							NULLIF($3, ''),
							provider_transaction_id
						),

					updated_at = now()

				WHERE
					id = $1::uuid
					AND status = 'pending'

					AND (
						NULLIF($2, '') IS NULL
						OR provider_payment_id IS NULL
						OR provider_payment_id = $2
					)
			`,
			paymentID,
			providerPaymentID,
			providerTransactionID,
		)
	if err != nil {
		if isPaymentUniqueViolation(
			err,
		) {
			return ErrPaymentReferenceConflict
		}

		return fmt.Errorf(
			"bind reconciliation provider identity: %w",
			err,
		)
	}

	if tag.RowsAffected() == 1 {
		return nil
	}

	var status string
	var existingProviderPaymentID string

	err =
		r.db.QueryRow(
			ctx,
			`
				SELECT
					status,
					COALESCE(
						provider_payment_id,
						''
					)

				FROM payments

				WHERE id = $1::uuid
			`,
			paymentID,
		).Scan(
			&status,
			&existingProviderPaymentID,
		)

	if errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return ErrPaymentNotFound
	}

	if err != nil {
		return fmt.Errorf(
			"load reconciliation payment state: %w",
			err,
		)
	}

	if status != StatusPending {
		return nil
	}

	if providerPaymentID != "" &&
		existingProviderPaymentID != "" &&
		existingProviderPaymentID !=
			providerPaymentID {

		return ErrPaymentReferenceConflict
	}

	return nil
}

func (r *Repository) MarkReconciliationTerminal(
	ctx context.Context,
	update reconciliationTerminalUpdate,
) error {
	if update.Status !=
		StatusFailed &&
		update.Status !=
			StatusExpired {

		return ErrInvalidInput
	}

	tx, err :=
		r.Begin(
			ctx,
		)
	if err != nil {
		return err
	}

	defer func() {
		_ = tx.Rollback(
			ctx,
		)
	}()

	var orderID string
	var provider string
	var providerPaymentID string
	var providerTransactionID string

	err =
		tx.QueryRow(
			ctx,
			`
				UPDATE payments

				SET
					status = $2,

					provider_payment_id =
						CASE
							WHEN NULLIF($3, '') IS NULL
								THEN provider_payment_id

							ELSE COALESCE(
								provider_payment_id,
								NULLIF($3, '')
							)
						END,

					provider_transaction_id =
						COALESCE(
							NULLIF($4, ''),
							provider_transaction_id
						),

					failed_at =
						CASE
							WHEN $2 = 'failed'
								THEN COALESCE(
									failed_at,
									now()
								)

							ELSE failed_at
						END,

					failure_code =
						NULLIF(
							$5,
							''
						),

					failure_message =
						NULLIF(
							$6,
							''
						),

					updated_at = now()

				WHERE
					id = $1::uuid
					AND status = 'pending'

					AND (
						NULLIF($3, '') IS NULL
						OR provider_payment_id IS NULL
						OR provider_payment_id = $3
					)

				RETURNING
					order_id::text,
					provider,

					COALESCE(
						provider_payment_id,
						''
					),

					COALESCE(
						provider_transaction_id,
						''
					)
			`,
			update.PaymentID,
			update.Status,
			update.ProviderPaymentID,
			update.ProviderTransactionID,
			update.FailureCode,
			update.FailureMessage,
		).Scan(
			&orderID,
			&provider,
			&providerPaymentID,
			&providerTransactionID,
		)

	if errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		var currentStatus string

		var currentProviderPaymentID string

		loadErr :=
			tx.QueryRow(
				ctx,
				`
					SELECT
						status,

						COALESCE(
							provider_payment_id,
							''
						)

					FROM payments

					WHERE id = $1::uuid

					FOR UPDATE
				`,
				update.PaymentID,
			).Scan(
				&currentStatus,
				&currentProviderPaymentID,
			)

		if errors.Is(
			loadErr,
			pgx.ErrNoRows,
		) {
			return ErrPaymentNotFound
		}

		if loadErr != nil {
			return fmt.Errorf(
				"load terminal reconciliation payment: %w",
				loadErr,
			)
		}

		/*
			A webhook or another reconciliation worker may already
			have moved this row to a terminal/succeeded state.

			That is safe and requires no second mutation.
		*/
		if currentStatus !=
			StatusPending {

			return nil
		}

		if update.ProviderPaymentID != "" &&
			currentProviderPaymentID != "" &&
			currentProviderPaymentID !=
				update.ProviderPaymentID {

			return ErrPaymentReferenceConflict
		}

		return ErrReconciliationRequired
	}

	if err != nil {
		if isPaymentUniqueViolation(
			err,
		) {
			return ErrPaymentReferenceConflict
		}

		return fmt.Errorf(
			"mark payment reconciliation terminal: %w",
			err,
		)
	}

	/*
		payment_events requires a non-empty provider_payment_id.

		If a provider was queried only through merchant reference
		and returned a terminal result without its provider ID, the
		local terminal state is still safe to record, but there is
		not enough provider identity to create a valid payment_event.
	*/
	if providerPaymentID != "" {
		_, err =
			tx.Exec(
				ctx,
				`
					INSERT INTO payment_events (
						order_id,
						payment_id,
						provider,
						provider_event_id,
						event_type,
						provider_payment_id,
						provider_transaction_id,
						payload_sha256,
						status,
						received_at,
						processed_at
					)

					VALUES (
						$1::uuid,
						$2::uuid,
						$3,
						$4,
						$5,
						$6,
						NULLIF($7, ''),
						$8,
						'processed',
						now(),
						now()
					)

					ON CONFLICT (
						provider,
						provider_event_id
					)

					DO NOTHING
				`,
				orderID,
				update.PaymentID,
				provider,
				update.EventID,
				update.EventType,
				providerPaymentID,
				providerTransactionID,
				update.PayloadSHA256,
			)
		if err != nil {
			return fmt.Errorf(
				"record payment reconciliation event: %w",
				err,
			)
		}
	}

	if err :=
		tx.Commit(
			ctx,
		); err != nil {

		return fmt.Errorf(
			"commit payment reconciliation terminal state: %w",
			err,
		)
	}

	return nil
}

func isPaymentUniqueViolation(
	err error,
) bool {
	var pgErr *pgconn.PgError

	if !errors.As(
		err,
		&pgErr,
	) {
		return false
	}

	return pgErr.Code ==
		"23505"
}
