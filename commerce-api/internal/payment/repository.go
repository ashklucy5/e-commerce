package payment

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

func (r *Repository) Begin(
	ctx context.Context,
) (pgx.Tx, error) {
	tx, err :=
		r.db.Begin(
			ctx,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"begin payment transaction: %w",
				err,
			)
	}

	return tx, nil
}

type orderPaymentSnapshot struct {
	ID string

	Status        string
	PaymentStatus string
	PaymentMethod string

	Currency    string
	TotalAmount int64
}

func (r *Repository) LockOrderPaymentSnapshotTx(
	ctx context.Context,
	tx pgx.Tx,
	orderID string,
) (orderPaymentSnapshot, error) {
	const query = `
		SELECT
			id::text,
			status,
			payment_status,
			payment_method,
			currency,
			total_amount
		FROM orders
		WHERE id = $1::uuid
		FOR UPDATE
	`

	var result orderPaymentSnapshot

	err :=
		tx.QueryRow(
			ctx,
			query,
			orderID,
		).Scan(
			&result.ID,
			&result.Status,
			&result.PaymentStatus,
			&result.PaymentMethod,
			&result.Currency,
			&result.TotalAmount,
		)

	if errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return orderPaymentSnapshot{},
			ErrOrderNotFound
	}

	if err != nil {
		return orderPaymentSnapshot{},
			fmt.Errorf(
				"lock payment order: %w",
				err,
			)
	}

	return result, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

const paymentSelect = `
	SELECT
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
	FROM payments
`

func scanPayment(
	row rowScanner,
) (Payment, error) {
	var result Payment

	var paidAt pgtype.Timestamptz
	var failedAt pgtype.Timestamptz

	err :=
		row.Scan(
			&result.ID,
			&result.OrderID,
			&result.Provider,
			&result.Status,
			&result.Amount,
			&result.Currency,
			&result.ProviderPaymentID,
			&result.ProviderTransactionID,
			&paidAt,
			&failedAt,
			&result.FailureCode,
			&result.FailureMessage,
			&result.CreatedAt,
			&result.UpdatedAt,
		)
	if err != nil {
		return Payment{}, err
	}

	if paidAt.Valid {
		value :=
			paidAt.Time

		result.PaidAt =
			&value
	}

	if failedAt.Valid {
		value :=
			failedAt.Time

		result.FailedAt =
			&value
	}

	return result, nil
}

func (r *Repository) GetByProviderPaymentIDTx(
	ctx context.Context,
	tx pgx.Tx,
	provider string,
	providerPaymentID string,
) (Payment, bool, error) {
	result, err :=
		scanPayment(
			tx.QueryRow(
				ctx,
				paymentSelect+`
					WHERE
						provider = $1
						AND provider_payment_id = $2
					FOR UPDATE
				`,
				provider,
				providerPaymentID,
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
				"get payment by provider payment id: %w",
				err,
			)
	}

	return result,
		true,
		nil
}

func (r *Repository) GetSucceededByOrderIDTx(
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
						AND status = 'succeeded'
					ORDER BY
						created_at,
						id
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
				"get successful payment by order: %w",
				err,
			)
	}

	return result,
		true,
		nil
}

func (r *Repository) InsertSucceededPaymentTx(
	ctx context.Context,
	tx pgx.Tx,
	input ConfirmVerifiedPaymentInput,
) (Payment, error) {
	const query = `
		INSERT INTO payments (
			order_id,
			provider,
			status,
			amount,
			currency,
			provider_payment_id,
			provider_transaction_id,
			paid_at,
			created_at,
			updated_at
		)
		VALUES (
			$1::uuid,
			$2,
			'succeeded',
			$3,
			$4,
			$5,
			NULLIF($6, ''),
			$7,
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
			provider_payment_id,
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
				input.OrderID,
				input.Provider,
				input.Amount,
				input.Currency,
				input.ProviderPaymentID,
				input.ProviderTransactionID,
				input.PaidAt,
			),
		)
	if err != nil {
		return Payment{},
			fmt.Errorf(
				"insert successful payment: %w",
				err,
			)
	}

	return result, nil
}

func (r *Repository) UpdateSucceededPaymentTx(
	ctx context.Context,
	tx pgx.Tx,
	paymentID string,
	input ConfirmVerifiedPaymentInput,
) (Payment, error) {
	const query = `
		UPDATE payments
		SET
			status = 'succeeded',
			provider_transaction_id =
				COALESCE(
					NULLIF($2, ''),
					provider_transaction_id
				),
			paid_at = $3,
			failed_at = NULL,
			failure_code = NULL,
			failure_message = NULL,
			updated_at = now()
		WHERE
			id = $1::uuid
			AND status IN (
				'pending',
				'failed',
				'expired',
				'succeeded'
			)
		RETURNING
			id::text,
			order_id::text,
			provider,
			status,
			amount,
			currency,
			provider_payment_id,
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
				"update successful payment: %w",
				err,
			)
	}

	return result, nil
}

func (r *Repository) InsertReceivedEventTx(
	ctx context.Context,
	tx pgx.Tx,
	input ConfirmVerifiedPaymentInput,
) (string, bool, error) {
	const query = `
		INSERT INTO payment_events (
			order_id,
			provider,
			provider_event_id,
			event_type,
			provider_payment_id,
			provider_transaction_id,
			payload_sha256,
			status,
			received_at
		)
		VALUES (
			$1::uuid,
			$2,
			$3,
			$4,
			$5,
			NULLIF($6, ''),
			$7,
			'received',
			now()
		)
		ON CONFLICT (
			provider,
			provider_event_id
		)
		DO NOTHING
		RETURNING id::text
	`

	var eventID string

	err :=
		tx.QueryRow(
			ctx,
			query,
			input.OrderID,
			input.Provider,
			input.ProviderEventID,
			input.EventType,
			input.ProviderPaymentID,
			input.ProviderTransactionID,
			input.PayloadSHA256,
		).Scan(
			&eventID,
		)

	if errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return "",
			false,
			nil
	}

	if err != nil {
		return "",
			false,
			fmt.Errorf(
				"insert payment event: %w",
				err,
			)
	}

	return eventID,
		true,
		nil
}

func (r *Repository) MarkEventProcessedTx(
	ctx context.Context,
	tx pgx.Tx,
	eventID string,
	paymentID string,
) error {
	const query = `
		UPDATE payment_events
		SET
			payment_id = $2::uuid,
			status = 'processed',
			error_message = NULL,
			processed_at = now()
		WHERE id = $1::uuid
	`

	tag, err :=
		tx.Exec(
			ctx,
			query,
			eventID,
			paymentID,
		)
	if err != nil {
		return fmt.Errorf(
			"mark payment event processed: %w",
			err,
		)
	}

	if tag.RowsAffected() != 1 {
		return fmt.Errorf(
			"payment event not found",
		)
	}

	return nil
}

func (r *Repository) MarkEventFailedTx(
	ctx context.Context,
	tx pgx.Tx,
	eventID string,
	message string,
) error {
	const query = `
		UPDATE payment_events
		SET
			status = 'failed',
			error_message = $2,
			processed_at = now()
		WHERE id = $1::uuid
	`

	tag, err :=
		tx.Exec(
			ctx,
			query,
			eventID,
			message,
		)
	if err != nil {
		return fmt.Errorf(
			"mark payment event failed: %w",
			err,
		)
	}

	if tag.RowsAffected() != 1 {
		return fmt.Errorf(
			"payment event not found",
		)
	}

	return nil
}

func (r *Repository) GetLatestByOrderID(
	ctx context.Context,
	orderID string,
) (Payment, error) {
	result, err :=
		scanPayment(
			r.db.QueryRow(
				ctx,
				paymentSelect+`
					WHERE order_id = $1::uuid
					ORDER BY
						created_at DESC,
						id DESC
					LIMIT 1
				`,
				orderID,
			),
		)

	if errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return Payment{},
			ErrPaymentNotFound
	}

	if err != nil {
		return Payment{},
			fmt.Errorf(
				"get payment by order: %w",
				err,
			)
	}

	return result, nil
}
