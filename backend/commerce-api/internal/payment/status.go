package payment

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

const maxPaymentCheckoutKeyLength = 160

type OrderPaymentStatus struct {
	OrderID string `json:"order_id"`

	OrderNumber string `json:"order_number"`

	PaymentMethod string `json:"payment_method"`
	PaymentStatus string `json:"payment_status"`

	Currency    string `json:"currency"`
	TotalAmount int64  `json:"total_amount"`

	PaymentDueAt *time.Time `json:"payment_due_at,omitempty"`
	PaidAt       *time.Time `json:"paid_at,omitempty"`

	LatestAttempt *PaymentAttemptSummary `json:"latest_attempt,omitempty"`
}

type PaymentAttemptSummary struct {
	ID string `json:"id"`

	Provider string `json:"provider"`
	Status   string `json:"status"`

	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`

	PaidAt   *time.Time `json:"paid_at,omitempty"`
	FailedAt *time.Time `json:"failed_at,omitempty"`

	FailureCode string `json:"failure_code,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (s *Service) GetOrderPaymentStatus(
	ctx context.Context,
	customerID string,
	orderID string,
	checkoutKey string,
) (OrderPaymentStatus, error) {
	orderID = strings.TrimSpace(orderID)

	if !paymentUUIDPattern.MatchString(orderID) {
		return OrderPaymentStatus{}, ErrInvalidInput
	}

	customerID = strings.TrimSpace(customerID)

	checkoutKey =
		normalizePaymentCheckoutKey(
			checkoutKey,
		)

	return s.repository.GetOrderPaymentStatus(
		ctx,
		customerID,
		orderID,
		checkoutKey,
	)
}

func normalizePaymentCheckoutKey(
	value string,
) string {
	value =
		strings.TrimSpace(
			value,
		)

	if value == "" ||
		len(value) > maxPaymentCheckoutKeyLength ||
		!strings.HasPrefix(
			value,
			"chk_",
		) {

		return ""
	}

	return value
}

func (r *Repository) GetOrderPaymentStatus(
	ctx context.Context,
	customerID string,
	orderID string,
	checkoutKey string,
) (OrderPaymentStatus, error) {
	const query = `
		SELECT
			o.id::text,
			o.order_number,
			o.payment_method,
			o.payment_status,
			o.currency,
			o.total_amount,
			o.payment_due_at,
			o.paid_at,

			p.id::text,
			p.provider,
			p.status,
			p.amount,
			p.currency,
			p.paid_at,
			p.failed_at,
			p.failure_code,
			p.created_at,
			p.updated_at

		FROM orders o

		LEFT JOIN checkout_sessions cs
			ON cs.id = o.checkout_id

		LEFT JOIN LATERAL (
			SELECT
				payment.id,
				payment.provider,
				payment.status,
				payment.amount,
				payment.currency,
				payment.paid_at,
				payment.failed_at,
				payment.failure_code,
				payment.created_at,
				payment.updated_at

			FROM payments payment

			WHERE payment.order_id = o.id

			ORDER BY
				payment.created_at DESC,
				payment.id DESC

			LIMIT 1
		) p ON TRUE

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
	`

	var result OrderPaymentStatus

	var orderPaymentDueAt pgtype.Timestamptz
	var orderPaidAt pgtype.Timestamptz

	var paymentID pgtype.Text
	var paymentProvider pgtype.Text
	var paymentStatus pgtype.Text
	var paymentAmount pgtype.Int8
	var paymentCurrency pgtype.Text

	var paymentPaidAt pgtype.Timestamptz
	var paymentFailedAt pgtype.Timestamptz

	var paymentFailureCode pgtype.Text

	var paymentCreatedAt pgtype.Timestamptz
	var paymentUpdatedAt pgtype.Timestamptz

	err :=
		r.db.QueryRow(
			ctx,
			query,
			orderID,
			customerID,
			checkoutKey,
		).Scan(
			&result.OrderID,
			&result.OrderNumber,
			&result.PaymentMethod,
			&result.PaymentStatus,
			&result.Currency,
			&result.TotalAmount,
			&orderPaymentDueAt,
			&orderPaidAt,
			&paymentID,
			&paymentProvider,
			&paymentStatus,
			&paymentAmount,
			&paymentCurrency,
			&paymentPaidAt,
			&paymentFailedAt,
			&paymentFailureCode,
			&paymentCreatedAt,
			&paymentUpdatedAt,
		)

	if errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return OrderPaymentStatus{},
			ErrOrderNotFound
	}

	if err != nil {
		return OrderPaymentStatus{},
			fmt.Errorf(
				"get order payment status: %w",
				err,
			)
	}

	if orderPaymentDueAt.Valid {
		value :=
			orderPaymentDueAt.Time

		result.PaymentDueAt =
			&value
	}

	if orderPaidAt.Valid {
		value :=
			orderPaidAt.Time

		result.PaidAt =
			&value
	}

	if paymentID.Valid {
		attempt :=
			PaymentAttemptSummary{
				ID: paymentID.String,

				Provider: paymentProvider.String,
				Status:   paymentStatus.String,

				Amount:   paymentAmount.Int64,
				Currency: paymentCurrency.String,

				FailureCode: paymentFailureCode.String,

				CreatedAt: paymentCreatedAt.Time,
				UpdatedAt: paymentUpdatedAt.Time,
			}

		if paymentPaidAt.Valid {
			value :=
				paymentPaidAt.Time

			attempt.PaidAt =
				&value
		}

		if paymentFailedAt.Valid {
			value :=
				paymentFailedAt.Time

			attempt.FailedAt =
				&value
		}

		result.LatestAttempt =
			&attempt
	}

	return result,
		nil
}
