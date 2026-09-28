package refund

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
				"begin refund transaction: %w",
				err,
			)
	}

	return tx, nil
}

type returnSnapshot struct {
	ID      string
	OrderID string
	Status  string

	OrderStatus   string
	PaymentStatus string
	PaymentMethod string

	Currency string

	SubtotalAmount int64
	DiscountAmount int64
	TotalAmount    int64
}

func (r *Repository) LockReturnTx(
	ctx context.Context,
	tx pgx.Tx,
	returnID string,
) (returnSnapshot, error) {
	const query = `
		SELECT
			r.id::text,
			r.order_id::text,
			r.status,
			o.status,
			o.payment_status,
			o.payment_method,
			o.currency,
			o.subtotal_amount,
			o.discount_amount,
			o.total_amount
		FROM returns r
		JOIN orders o
			ON o.id = r.order_id
		WHERE r.id = $1::uuid
		FOR UPDATE OF r, o
	`

	var result returnSnapshot

	err :=
		tx.QueryRow(
			ctx,
			query,
			returnID,
		).Scan(
			&result.ID,
			&result.OrderID,
			&result.Status,
			&result.OrderStatus,
			&result.PaymentStatus,
			&result.PaymentMethod,
			&result.Currency,
			&result.SubtotalAmount,
			&result.DiscountAmount,
			&result.TotalAmount,
		)

	if errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return returnSnapshot{},
			ErrReturnNotFound
	}

	if err != nil {
		return returnSnapshot{},
			fmt.Errorf(
				"lock refund return: %w",
				err,
			)
	}

	return result, nil
}

func (r *Repository) ReturnGrossAmountTx(
	ctx context.Context,
	tx pgx.Tx,
	returnID string,
) (int64, error) {
	const query = `
		SELECT
			COALESCE(
				SUM(
					ri.received_quantity::bigint *
					oi.unit_price_amount
				),
				0
			)::bigint
		FROM return_items ri
		JOIN order_items oi
			ON oi.id = ri.order_item_id
		WHERE ri.return_id = $1::uuid
	`

	var amount int64

	if err :=
		tx.QueryRow(
			ctx,
			query,
			returnID,
		).Scan(
			&amount,
		); err != nil {
		return 0,
			fmt.Errorf(
				"calculate return refund amount: %w",
				err,
			)
	}

	return amount, nil
}

type paymentSnapshot struct {
	ID       string
	Amount   int64
	Currency string
	Status   string
	Provider string
}

func (r *Repository) LockPaymentForOrderTx(
	ctx context.Context,
	tx pgx.Tx,
	orderID string,
) (paymentSnapshot, error) {
	const query = `
		SELECT
			id::text,
			amount,
			currency,
			status,
			provider
		FROM payments
		WHERE
			order_id = $1::uuid
			AND status IN (
				'succeeded',
				'refunded'
			)
		ORDER BY
			created_at,
			id
		LIMIT 1
		FOR UPDATE
	`

	var result paymentSnapshot

	err :=
		tx.QueryRow(
			ctx,
			query,
			orderID,
		).Scan(
			&result.ID,
			&result.Amount,
			&result.Currency,
			&result.Status,
			&result.Provider,
		)

	if errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return paymentSnapshot{},
			ErrPaymentNotFound
	}

	if err != nil {
		return paymentSnapshot{},
			fmt.Errorf(
				"lock refundable payment: %w",
				err,
			)
	}

	return result, nil
}

func (r *Repository) ExistingByReturnIDTx(
	ctx context.Context,
	tx pgx.Tx,
	returnID string,
) (Refund, bool, error) {
	result, err :=
		scanRefund(
			tx.QueryRow(
				ctx,
				refundSelect+`
					WHERE
						return_id = $1::uuid
						AND status <> 'cancelled'
					ORDER BY
						created_at DESC,
						id DESC
					LIMIT 1
					FOR UPDATE
				`,
				returnID,
			),
		)

	if errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return Refund{},
			false,
			nil
	}

	if err != nil {
		return Refund{},
			false,
			fmt.Errorf(
				"load existing return refund: %w",
				err,
			)
	}

	return result,
		true,
		nil
}

func (r *Repository) CommittedRefundAmountTx(
	ctx context.Context,
	tx pgx.Tx,
	orderID string,
) (int64, error) {
	const query = `
		SELECT
			COALESCE(
				SUM(amount),
				0
			)::bigint
		FROM refunds
		WHERE
			order_id = $1::uuid
			AND status IN (
				'requested',
				'approved',
				'processing',
				'succeeded'
			)
	`

	var amount int64

	if err :=
		tx.QueryRow(
			ctx,
			query,
			orderID,
		).Scan(
			&amount,
		); err != nil {
		return 0,
			fmt.Errorf(
				"calculate committed refunds: %w",
				err,
			)
	}

	return amount, nil
}

func (r *Repository) SuccessfulRefundAmountTx(
	ctx context.Context,
	tx pgx.Tx,
	orderID string,
) (int64, error) {
	const query = `
		SELECT
			COALESCE(
				SUM(amount),
				0
			)::bigint
		FROM refunds
		WHERE
			order_id = $1::uuid
			AND status = 'succeeded'
	`

	var amount int64

	if err := tx.QueryRow(
		ctx,
		query,
		orderID,
	).Scan(
		&amount,
	); err != nil {
		return 0,
			fmt.Errorf(
				"calculate successful refunds: %w",
				err,
			)
	}

	return amount, nil
}

func (r *Repository) CreateReturnRefundTx(
	ctx context.Context,
	tx pgx.Tx,
	refundNumber string,
	snapshot returnSnapshot,
	paymentID string,
	provider string,
	amount int64,
	reason string,
	actorID string,
) (string, error) {
	const query = `
		INSERT INTO refunds (
			refund_number,
			order_id,
			return_id,
			payment_id,
			source_type,
			status,
			amount,
			currency,
			provider,
			reason,
			requested_by,
			requested_at,
			created_at,
			updated_at
		)
		VALUES (
			$1,
			$2::uuid,
			$3::uuid,
			NULLIF($4, '')::uuid,
			'return',
			'requested',
			$5,
			$6,
			$7,
			$8,
			NULLIF($9, ''),
			now(),
			now(),
			now()
		)
		RETURNING id::text
	`

	var refundID string

	if err :=
		tx.QueryRow(
			ctx,
			query,
			refundNumber,
			snapshot.OrderID,
			snapshot.ID,
			paymentID,
			amount,
			snapshot.Currency,
			provider,
			reason,
			actorID,
		).Scan(
			&refundID,
		); err != nil {
		return "",
			fmt.Errorf(
				"create return refund: %w",
				err,
			)
	}

	return refundID, nil
}

func (r *Repository) LockByIDTx(
	ctx context.Context,
	tx pgx.Tx,
	refundID string,
) (Refund, error) {
	result, err :=
		scanRefund(
			tx.QueryRow(
				ctx,
				refundSelect+`
					WHERE id = $1::uuid
					FOR UPDATE
				`,
				refundID,
			),
		)

	if errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return Refund{},
			ErrRefundNotFound
	}

	if err != nil {
		return Refund{},
			fmt.Errorf(
				"lock refund: %w",
				err,
			)
	}

	return result, nil
}

func (r *Repository) GetByID(
	ctx context.Context,
	refundID string,
) (Refund, error) {
	result, err :=
		scanRefund(
			r.db.QueryRow(
				ctx,
				refundSelect+`
					WHERE id = $1::uuid
				`,
				refundID,
			),
		)

	if errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return Refund{},
			ErrRefundNotFound
	}

	if err != nil {
		return Refund{},
			fmt.Errorf(
				"get refund: %w",
				err,
			)
	}

	return result, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

const refundSelect = `
	SELECT
		id::text,
		refund_number,
		order_id::text,
		COALESCE(return_id::text, ''),
		COALESCE(payment_id::text, ''),
		source_type,
		status,
		amount,
		currency,
		COALESCE(provider, ''),
		COALESCE(provider_refund_id, ''),
		reason,
		COALESCE(requested_by, ''),
		COALESCE(approved_by, ''),
		COALESCE(failure_code, ''),
		COALESCE(failure_message, ''),
		requested_at,
		approved_at,
		processing_at,
		succeeded_at,
		failed_at,
		cancelled_at,
		created_at,
		updated_at
	FROM refunds
`

func scanRefund(
	row rowScanner,
) (Refund, error) {
	var result Refund

	var approvedAt pgtype.Timestamptz
	var processingAt pgtype.Timestamptz
	var succeededAt pgtype.Timestamptz
	var failedAt pgtype.Timestamptz
	var cancelledAt pgtype.Timestamptz

	err :=
		row.Scan(
			&result.ID,
			&result.RefundNumber,
			&result.OrderID,
			&result.ReturnID,
			&result.PaymentID,
			&result.SourceType,
			&result.Status,
			&result.Amount,
			&result.Currency,
			&result.Provider,
			&result.ProviderRefundID,
			&result.Reason,
			&result.RequestedBy,
			&result.ApprovedBy,
			&result.FailureCode,
			&result.FailureMessage,
			&result.RequestedAt,
			&approvedAt,
			&processingAt,
			&succeededAt,
			&failedAt,
			&cancelledAt,
			&result.CreatedAt,
			&result.UpdatedAt,
		)

	if err != nil {
		return Refund{}, err
	}

	if approvedAt.Valid {
		value := approvedAt.Time
		result.ApprovedAt = &value
	}

	if processingAt.Valid {
		value := processingAt.Time
		result.ProcessingAt = &value
	}

	if succeededAt.Valid {
		value := succeededAt.Time
		result.SucceededAt = &value
	}

	if failedAt.Valid {
		value := failedAt.Time
		result.FailedAt = &value
	}

	if cancelledAt.Valid {
		value := cancelledAt.Time
		result.CancelledAt = &value
	}

	return result, nil
}

func (r *Repository) InsertEventTx(
	ctx context.Context,
	tx pgx.Tx,
	refundID string,
	input eventInsert,
) error {
	const query = `
		INSERT INTO refund_events (
			refund_id,
			event_type,
			from_status,
			to_status,
			message,
			actor_type,
			actor_id,
			created_at
		)
		VALUES (
			$1::uuid,
			$2,
			NULLIF($3, ''),
			NULLIF($4, ''),
			NULLIF($5, ''),
			NULLIF($6, ''),
			NULLIF($7, ''),
			now()
		)
	`

	if _, err :=
		tx.Exec(
			ctx,
			query,
			refundID,
			input.EventType,
			input.FromStatus,
			input.ToStatus,
			input.Message,
			input.ActorType,
			input.ActorID,
		); err != nil {
		return fmt.Errorf(
			"insert refund event: %w",
			err,
		)
	}

	return nil
}

func (r *Repository) ListEvents(
	ctx context.Context,
	refundID string,
) ([]Event, error) {
	const query = `
		SELECT
			id::text,
			refund_id::text,
			event_type,
			COALESCE(from_status, ''),
			COALESCE(to_status, ''),
			COALESCE(message, ''),
			COALESCE(actor_type, ''),
			COALESCE(actor_id, ''),
			created_at
		FROM refund_events
		WHERE refund_id = $1::uuid
		ORDER BY
			created_at,
			id
	`

	rows, err :=
		r.db.Query(
			ctx,
			query,
			refundID,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"list refund events: %w",
				err,
			)
	}
	defer rows.Close()

	result :=
		make(
			[]Event,
			0,
		)

	for rows.Next() {
		var event Event

		if err :=
			rows.Scan(
				&event.ID,
				&event.RefundID,
				&event.EventType,
				&event.FromStatus,
				&event.ToStatus,
				&event.Message,
				&event.ActorType,
				&event.ActorID,
				&event.CreatedAt,
			); err != nil {
			return nil,
				fmt.Errorf(
					"scan refund event: %w",
					err,
				)
		}

		result =
			append(
				result,
				event,
			)
	}

	if err :=
		rows.Err(); err != nil {
		return nil,
			fmt.Errorf(
				"iterate refund events: %w",
				err,
			)
	}

	return result, nil
}

func (r *Repository) MarkApprovedTx(
	ctx context.Context,
	tx pgx.Tx,
	refundID string,
	actorID string,
) error {
	const query = `
		UPDATE refunds
		SET
			status = 'approved',
			approved_by = NULLIF($2, ''),
			approved_at = now(),
			updated_at = now()
		WHERE
			id = $1::uuid
			AND status = 'requested'
	`

	tag, err :=
		tx.Exec(
			ctx,
			query,
			refundID,
			actorID,
		)

	if err != nil {
		return fmt.Errorf(
			"approve refund: %w",
			err,
		)
	}

	if tag.RowsAffected() != 1 {
		return ErrInvalidRefundTransition
	}

	return nil
}

func (r *Repository) MarkProcessingTx(
	ctx context.Context,
	tx pgx.Tx,
	refundID string,
) error {
	const query = `
		UPDATE refunds
		SET
			status = 'processing',
			processing_at = now(),
			failed_at = NULL,
			failure_code = NULL,
			failure_message = NULL,
			updated_at = now()
		WHERE
			id = $1::uuid
			AND status IN (
				'approved',
				'failed'
			)
	`

	tag, err :=
		tx.Exec(
			ctx,
			query,
			refundID,
		)

	if err != nil {
		return fmt.Errorf(
			"start refund processing: %w",
			err,
		)
	}

	if tag.RowsAffected() != 1 {
		return ErrInvalidRefundTransition
	}

	return nil
}

func (r *Repository) MarkFailedTx(
	ctx context.Context,
	tx pgx.Tx,
	refundID string,
	request FailureRequest,
) error {
	const query = `
		UPDATE refunds
		SET
			status = 'failed',
			failure_code = $2,
			failure_message = $3,
			failed_at = now(),
			updated_at = now()
		WHERE
			id = $1::uuid
			AND status = 'processing'
	`

	tag, err :=
		tx.Exec(
			ctx,
			query,
			refundID,
			request.FailureCode,
			request.FailureMessage,
		)

	if err != nil {
		return fmt.Errorf(
			"mark refund failed: %w",
			err,
		)
	}

	if tag.RowsAffected() != 1 {
		return ErrInvalidRefundTransition
	}

	return nil
}

func (r *Repository) MarkSucceededTx(
	ctx context.Context,
	tx pgx.Tx,
	refundID string,
	providerRefundID string,
) error {
	const query = `
		UPDATE refunds
		SET
			status = 'succeeded',
			provider_refund_id = $2,
			succeeded_at = now(),
			failed_at = NULL,
			failure_code = NULL,
			failure_message = NULL,
			updated_at = now()
		WHERE
			id = $1::uuid
			AND status = 'processing'
	`

	tag, err :=
		tx.Exec(
			ctx,
			query,
			refundID,
			providerRefundID,
		)

	if err != nil {
		return fmt.Errorf(
			"mark refund succeeded: %w",
			err,
		)
	}

	if tag.RowsAffected() != 1 {
		return ErrInvalidRefundTransition
	}

	return nil
}

func (r *Repository) CompleteReturnTx(
	ctx context.Context,
	tx pgx.Tx,
	returnID string,
	refundID string,
) error {
	const updateQuery = `
		UPDATE returns
		SET
			status = 'completed',
			completed_at = now(),
			updated_at = now()
		WHERE
			id = $1::uuid
			AND status = 'inspected'
	`

	tag, err :=
		tx.Exec(
			ctx,
			updateQuery,
			returnID,
		)

	if err != nil {
		return fmt.Errorf(
			"complete refunded return: %w",
			err,
		)
	}

	if tag.RowsAffected() != 1 {
		return ErrReturnNotReady
	}

	const eventQuery = `
		INSERT INTO return_events (
			return_id,
			event_type,
			from_status,
			to_status,
			message,
			actor_type,
			actor_id,
			created_at
		)
		VALUES (
			$1::uuid,
			'return_completed',
			'inspected',
			'completed',
			'Return completed after refund',
			'refund',
			$2,
			now()
		)
	`

	if _, err :=
		tx.Exec(
			ctx,
			eventQuery,
			returnID,
			refundID,
		); err != nil {
		return fmt.Errorf(
			"insert completed return event: %w",
			err,
		)
	}

	return nil
}

func (r *Repository) MarkOrderFullyRefundedTx(
	ctx context.Context,
	tx pgx.Tx,
	orderID string,
) error {
	const query = `
		UPDATE orders
		SET
			payment_status = 'refunded',
			updated_at = now()
		WHERE
			id = $1::uuid
			AND payment_status IN (
				'paid',
				'cod_collected',
				'refunded'
			)
	`

	tag, err :=
		tx.Exec(
			ctx,
			query,
			orderID,
		)

	if err != nil {
		return fmt.Errorf(
			"mark order fully refunded: %w",
			err,
		)
	}

	if tag.RowsAffected() != 1 {
		return ErrRefundNotAllowed
	}

	return nil
}

func (r *Repository) MarkPaymentFullyRefundedTx(
	ctx context.Context,
	tx pgx.Tx,
	paymentID string,
) error {
	if paymentID == "" {
		return nil
	}

	const query = `
		UPDATE payments
		SET
			status = 'refunded',
			updated_at = now()
		WHERE
			id = $1::uuid
			AND status IN (
				'succeeded',
				'refunded'
			)
	`

	tag, err :=
		tx.Exec(
			ctx,
			query,
			paymentID,
		)

	if err != nil {
		return fmt.Errorf(
			"mark payment fully refunded: %w",
			err,
		)
	}

	if tag.RowsAffected() != 1 {
		return ErrRefundNotAllowed
	}

	return nil
}

type cancellationOrderSnapshot struct {
	ID string

	Status        string
	PaymentStatus string
	PaymentMethod string

	Currency    string
	TotalAmount int64
}

func (r *Repository) LockCancellationOrderTx(
	ctx context.Context,
	tx pgx.Tx,
	orderID string,
) (cancellationOrderSnapshot, error) {
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

	var result cancellationOrderSnapshot

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
		return cancellationOrderSnapshot{},
			ErrOrderNotFound
	}

	if err != nil {
		return cancellationOrderSnapshot{},
			fmt.Errorf(
				"lock cancellation refund order: %w",
				err,
			)
	}

	return result, nil
}

func (r *Repository) ExistingCancellationByOrderIDTx(
	ctx context.Context,
	tx pgx.Tx,
	orderID string,
) (Refund, bool, error) {
	result, err :=
		scanRefund(
			tx.QueryRow(
				ctx,
				refundSelect+`
					WHERE
						order_id = $1::uuid
						AND source_type = 'cancellation'
						AND status <> 'cancelled'
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
		return Refund{},
			false,
			nil
	}

	if err != nil {
		return Refund{},
			false,
			fmt.Errorf(
				"load existing cancellation refund: %w",
				err,
			)
	}

	return result,
		true,
		nil
}

func (r *Repository) CreateCancellationRefundTx(
	ctx context.Context,
	tx pgx.Tx,
	refundNumber string,
	orderID string,
	paymentID string,
	amount int64,
	currency string,
	provider string,
	reason string,
	actorID string,
) (string, error) {
	const query = `
		INSERT INTO refunds (
			refund_number,
			order_id,
			return_id,
			payment_id,
			source_type,
			status,
			amount,
			currency,
			provider,
			reason,
			requested_by,
			requested_at,
			created_at,
			updated_at
		)
		VALUES (
			$1,
			$2::uuid,
			NULL,
			$3::uuid,
			'cancellation',
			'requested',
			$4,
			$5,
			$6,
			$7,
			NULLIF($8, ''),
			now(),
			now(),
			now()
		)
		RETURNING id::text
	`

	var refundID string

	if err :=
		tx.QueryRow(
			ctx,
			query,
			refundNumber,
			orderID,
			paymentID,
			amount,
			currency,
			provider,
			reason,
			actorID,
		).Scan(
			&refundID,
		); err != nil {
		return "",
			fmt.Errorf(
				"create cancellation refund: %w",
				err,
			)
	}

	return refundID, nil
}
