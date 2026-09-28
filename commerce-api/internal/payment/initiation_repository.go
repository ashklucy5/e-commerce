package payment

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type paymentInitiationOrder struct {
	ID string

	OrderNumber string

	Status string

	PaymentStatus string

	PaymentMethod string

	Currency string

	TotalAmount int64

	PaymentDueAt *time.Time
}

type paymentInitiationAttempt struct {
	ID string

	OrderID string

	Provider string

	Status string

	Amount int64

	Currency string

	AttemptKey string

	ProviderPaymentID string

	HandoffURL string

	ProviderExpiresAt *time.Time

	CreatedAt time.Time

	UpdatedAt time.Time
}

func (r *Repository) LockOwnedPaymentInitiationOrderTx(
	ctx context.Context,
	tx pgx.Tx,
	orderID string,
	customerID string,
	checkoutKey string,
) (paymentInitiationOrder, error) {
	const query = `
		SELECT
			o.id::text,
			o.order_number,
			o.status,
			o.payment_status,
			o.payment_method,
			o.currency,
			o.total_amount,
			o.payment_due_at

		FROM orders o

		LEFT JOIN checkout_sessions cs
			ON cs.id = o.checkout_id

		WHERE
			o.id = $1::uuid

			AND (
				(
					o.customer_id IS NOT NULL
					AND o.customer_id =
						NULLIF($2, '')::uuid
				)

				OR (
					o.customer_id IS NULL
					AND $3 <> ''
					AND cs.checkout_key = $3
				)
			)

		FOR UPDATE
	`

	var result paymentInitiationOrder

	var paymentDueAt pgtype.Timestamptz

	err :=
		tx.QueryRow(
			ctx,
			query,
			orderID,
			customerID,
			checkoutKey,
		).Scan(
			&result.ID,
			&result.OrderNumber,
			&result.Status,
			&result.PaymentStatus,
			&result.PaymentMethod,
			&result.Currency,
			&result.TotalAmount,
			&paymentDueAt,
		)

	if errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return paymentInitiationOrder{},
			ErrOrderNotFound
	}

	if err != nil {
		return paymentInitiationOrder{},
			fmt.Errorf(
				"lock owned payment initiation order: %w",
				err,
			)
	}

	if paymentDueAt.Valid {
		value :=
			paymentDueAt.Time

		result.PaymentDueAt =
			&value
	}

	return result,
		nil
}

func scanPaymentInitiationAttempt(
	row rowScanner,
) (
	paymentInitiationAttempt,
	error,
) {
	var result paymentInitiationAttempt

	var providerExpiresAt pgtype.Timestamptz

	err :=
		row.Scan(
			&result.ID,
			&result.OrderID,
			&result.Provider,
			&result.Status,
			&result.Amount,
			&result.Currency,
			&result.AttemptKey,
			&result.ProviderPaymentID,
			&result.HandoffURL,
			&providerExpiresAt,
			&result.CreatedAt,
			&result.UpdatedAt,
		)
	if err != nil {
		return paymentInitiationAttempt{},
			err
	}

	if providerExpiresAt.Valid {
		value :=
			providerExpiresAt.Time

		result.ProviderExpiresAt =
			&value
	}

	return result,
		nil
}

func (r *Repository) GetPendingInitiationByOrderIDTx(
	ctx context.Context,
	tx pgx.Tx,
	orderID string,
) (
	paymentInitiationAttempt,
	bool,
	error,
) {
	result, err :=
		scanPaymentInitiationAttempt(
			tx.QueryRow(
				ctx,
				`
					SELECT
						id::text,
						order_id::text,
						provider,
						status,
						amount,
						currency,
						COALESCE(attempt_key, ''),
						COALESCE(provider_payment_id, ''),
						COALESCE(handoff_url, ''),
						provider_expires_at,
						created_at,
						updated_at

					FROM payments

					WHERE
						order_id = $1::uuid
						AND status = 'pending'

					ORDER BY
						created_at DESC,
						id DESC

					LIMIT 1

					FOR UPDATE
				`,
				orderID,
			),
		)

	if errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return paymentInitiationAttempt{},
			false,
			nil
	}

	if err != nil {
		return paymentInitiationAttempt{},
			false,
			fmt.Errorf(
				"get pending payment initiation: %w",
				err,
			)
	}

	return result,
		true,
		nil
}

func (r *Repository) InsertPendingInitiationTx(
	ctx context.Context,
	tx pgx.Tx,
	order paymentInitiationOrder,
	attemptKey string,
) (
	paymentInitiationAttempt,
	error,
) {
	result, err :=
		scanPaymentInitiationAttempt(
			tx.QueryRow(
				ctx,
				`
					INSERT INTO payments (
						order_id,
						provider,
						status,
						amount,
						currency,
						attempt_key,
						created_at,
						updated_at
					)
					VALUES (
						$1::uuid,
						$2,
						'pending',
						$3,
						$4,
						$5,
						now(),
						now()
					)
					RETURNING
						id::text,
						order_id::text,
						provider,
						status,
						amount,
						currency,
						COALESCE(attempt_key, ''),
						COALESCE(provider_payment_id, ''),
						COALESCE(handoff_url, ''),
						provider_expires_at,
						created_at,
						updated_at
				`,
				order.ID,
				order.PaymentMethod,
				order.TotalAmount,
				order.Currency,
				attemptKey,
			),
		)
	if err != nil {
		return paymentInitiationAttempt{},
			fmt.Errorf(
				"insert pending payment initiation: %w",
				err,
			)
	}

	return result,
		nil
}

func (r *Repository) UpdateInitiationProviderHandoff(
	ctx context.Context,
	paymentID string,
	providerPaymentID string,
	handoffURL string,
	providerExpiresAt *time.Time,
) (
	paymentInitiationAttempt,
	error,
) {
	result, err :=
		scanPaymentInitiationAttempt(
			r.db.QueryRow(
				ctx,
				`
					UPDATE payments

					SET
						provider_payment_id = $2,

						handoff_url =
							NULLIF(
								$3,
								''
							),

						provider_expires_at = $4,

						updated_at = now()

					WHERE
						id = $1::uuid
						AND status = 'pending'

						AND (
							provider_payment_id IS NULL
							OR provider_payment_id = $2
						)

					RETURNING
						id::text,
						order_id::text,
						provider,
						status,
						amount,
						currency,
						COALESCE(attempt_key, ''),
						COALESCE(provider_payment_id, ''),
						COALESCE(handoff_url, ''),
						provider_expires_at,
						created_at,
						updated_at
				`,
				paymentID,
				providerPaymentID,
				handoffURL,
				providerExpiresAt,
			),
		)

	if errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return paymentInitiationAttempt{},
			ErrReconciliationRequired
	}

	if err != nil {
		return paymentInitiationAttempt{},
			fmt.Errorf(
				"update provider payment handoff: %w",
				err,
			)
	}

	return result,
		nil
}

func (r *Repository) MarkInitiationRejected(
	ctx context.Context,
	paymentID string,
) error {
	tag, err :=
		r.db.Exec(
			ctx,
			`
				UPDATE payments

				SET
					status = 'failed',

					failed_at = now(),

					failure_code =
						'provider_rejected',

					failure_message =
						'payment provider rejected initiation',

					updated_at = now()

				WHERE
					id = $1::uuid
					AND status = 'pending'
			`,
			paymentID,
		)
	if err != nil {
		return fmt.Errorf(
			"mark payment initiation rejected: %w",
			err,
		)
	}

	if tag.RowsAffected() != 1 {
		return ErrReconciliationRequired
	}

	return nil
}
