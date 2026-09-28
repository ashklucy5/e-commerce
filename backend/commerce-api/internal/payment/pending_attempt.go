package payment

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

func (r *Repository) GetPendingByOrderIDTx(
	ctx context.Context,
	tx pgx.Tx,
	orderID string,
) (Payment, bool, error) {
	result, err :=
		scanPayment(
			tx.QueryRow(
				ctx,
				paymentSelect+`
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
		return Payment{},
			false,
			nil
	}

	if err != nil {
		return Payment{},
			false,
			fmt.Errorf(
				"get pending payment by order: %w",
				err,
			)
	}

	return result,
		true,
		nil
}

func (r *Repository) PromotePendingToSucceededTx(
	ctx context.Context,
	tx pgx.Tx,
	paymentID string,
	input ConfirmVerifiedPaymentInput,
) (Payment, error) {
	const query = `
		UPDATE payments
		SET
			status = 'succeeded',

			provider_payment_id = $2,

			provider_transaction_id =
				COALESCE(
					NULLIF($3, ''),
					provider_transaction_id
				),

			paid_at = $4,

			failed_at = NULL,
			failure_code = NULL,
			failure_message = NULL,

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
			COALESCE(provider_payment_id, ''),
			COALESCE(provider_transaction_id, ''),
			paid_at,
			failed_at,
			COALESCE(failure_code, ''),
			COALESCE(failure_message, ''),
			created_at,
			updated_at
	`

	result, err :=
		scanPayment(
			tx.QueryRow(
				ctx,
				query,
				paymentID,
				input.ProviderPaymentID,
				input.ProviderTransactionID,
				input.PaidAt,
			),
		)

	if errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return Payment{},
			ErrPaymentReferenceConflict
	}

	if err != nil {
		return Payment{},
			fmt.Errorf(
				"promote pending payment to succeeded: %w",
				err,
			)
	}

	return result,
		nil
}
