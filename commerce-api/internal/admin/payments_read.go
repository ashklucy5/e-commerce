package admin

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	platformdatabase "project.local/commerce-api/internal/platform/database"
	platformpagination "project.local/commerce-api/internal/platform/pagination"
)

func (s *Service) ListPayments(
	ctx context.Context,
	params platformpagination.Params,
	filter PaymentReadFilter,
) (
	PaymentListResult,
	error,
) {
	queryCtx, cancel := adminReadContext(ctx)
	defer cancel()

	rows, err := s.db.Query(
		queryCtx,
		`
			SELECT
				p.id::text,
				p.order_id::text,
				o.order_number,
				o.customer_name,
				o.customer_phone,
				p.provider,
				p.status,
				p.amount,
				p.currency,
				p.provider_payment_id,
				COALESCE(p.provider_transaction_id, ''),
				p.paid_at,
				p.failed_at,
				COALESCE(p.failure_code, ''),
				COALESCE(p.failure_message, ''),
				p.created_at,
				p.updated_at,
				COUNT(*) OVER()::bigint
			FROM payments p
			JOIN orders o
				ON o.id = p.order_id
			WHERE
				($1 = '' OR p.status = $1)
				AND ($2 = '' OR p.provider = $2)
				AND (
					$3 = ''
					OR o.order_number = upper($3)
					OR o.customer_phone = $3
					OR p.provider_payment_id = $3
					OR COALESCE(p.provider_transaction_id, '') = $3
				)
			ORDER BY p.created_at DESC, p.id DESC
			LIMIT $4 OFFSET $5
		`,
		filter.Status,
		filter.Provider,
		filter.Query,
		params.Limit,
		params.Offset(),
	)
	if err != nil {
		return PaymentListResult{},
			fmt.Errorf(
				"list admin payments: %w",
				err,
			)
	}
	defer rows.Close()

	items := make(
		[]PaymentListItem,
		0,
		params.Limit,
	)

	var total int64

	for rows.Next() {
		var item PaymentListItem
		var paidAt pgtype.Timestamptz
		var failedAt pgtype.Timestamptz

		if err := rows.Scan(
			&item.ID,
			&item.OrderID,
			&item.OrderNumber,
			&item.CustomerName,
			&item.CustomerPhone,
			&item.Provider,
			&item.Status,
			&item.Amount,
			&item.Currency,
			&item.ProviderPaymentID,
			&item.ProviderTransactionID,
			&paidAt,
			&failedAt,
			&item.FailureCode,
			&item.FailureMessage,
			&item.CreatedAt,
			&item.UpdatedAt,
			&total,
		); err != nil {
			return PaymentListResult{},
				fmt.Errorf(
					"scan admin payment: %w",
					err,
				)
		}

		item.PaidAt = adminTimePointer(paidAt)
		item.FailedAt = adminTimePointer(failedAt)

		items = append(
			items,
			item,
		)
	}

	if err := rows.Err(); err != nil {
		return PaymentListResult{},
			fmt.Errorf(
				"iterate admin payments: %w",
				err,
			)
	}

	return PaymentListResult{
		Items: items,
		Meta: platformpagination.NewMeta(
			params,
			total,
		),
	}, nil
}

func (s *Service) GetPayment(
	ctx context.Context,
	paymentID string,
) (
	PaymentDetail,
	error,
) {
	queryCtx, cancel := adminReadContext(ctx)
	defer cancel()

	var result PaymentDetail

	err := platformdatabase.WithinTxOptions(
		queryCtx,
		s.db,
		pgx.TxOptions{
			IsoLevel:   pgx.RepeatableRead,
			AccessMode: pgx.ReadOnly,
		},
		func(
			ctx context.Context,
			tx pgx.Tx,
		) error {
			if err := scanPaymentDetail(
				ctx,
				tx,
				paymentID,
				&result,
			); err != nil {
				return err
			}

			events, err := listPaymentEventsRead(
				ctx,
				tx,
				paymentID,
			)
			if err != nil {
				return err
			}

			refunds, err := listRefundSummaries(
				ctx,
				tx,
				"payment_id",
				paymentID,
			)
			if err != nil {
				return err
			}

			result.Events = events
			result.Refunds = refunds

			return nil
		},
	)
	if err != nil {
		return PaymentDetail{}, err
	}

	return result, nil
}

func scanPaymentDetail(
	ctx context.Context,
	querier adminReadQuerier,
	paymentID string,
	result *PaymentDetail,
) error {
	var paidAt pgtype.Timestamptz
	var failedAt pgtype.Timestamptz

	err := querier.QueryRow(
		ctx,
		`
			SELECT
				p.id::text,
				p.order_id::text,
				o.order_number,
				o.customer_name,
				o.customer_phone,
				p.provider,
				p.status,
				p.amount,
				p.currency,
				p.provider_payment_id,
				COALESCE(p.provider_transaction_id, ''),
				p.paid_at,
				p.failed_at,
				COALESCE(p.failure_code, ''),
				COALESCE(p.failure_message, ''),
				p.created_at,
				p.updated_at
			FROM payments p
			JOIN orders o
				ON o.id = p.order_id
			WHERE p.id = $1::uuid
		`,
		paymentID,
	).Scan(
		&result.ID,
		&result.OrderID,
		&result.OrderNumber,
		&result.CustomerName,
		&result.CustomerPhone,
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

	if errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return ErrAdminPaymentNotFound
	}

	if err != nil {
		return fmt.Errorf(
			"get admin payment: %w",
			err,
		)
	}

	result.PaidAt = adminTimePointer(paidAt)
	result.FailedAt = adminTimePointer(failedAt)

	return nil
}

func listPaymentEventsRead(
	ctx context.Context,
	querier adminReadQuerier,
	paymentID string,
) (
	[]PaymentEventRead,
	error,
) {
	rows, err := querier.Query(
		ctx,
		`
			SELECT
				id::text,
				provider_event_id,
				event_type,
				provider_payment_id,
				COALESCE(provider_transaction_id, ''),
				payload_sha256,
				status,
				COALESCE(error_message, ''),
				received_at,
				processed_at
			FROM payment_events
			WHERE payment_id = $1::uuid
			ORDER BY received_at DESC, id DESC
			LIMIT 200
		`,
		paymentID,
	)
	if err != nil {
		return nil,
			fmt.Errorf(
				"list admin payment events: %w",
				err,
			)
	}
	defer rows.Close()

	result := make(
		[]PaymentEventRead,
		0,
	)

	for rows.Next() {
		var item PaymentEventRead
		var processedAt pgtype.Timestamptz

		if err := rows.Scan(
			&item.ID,
			&item.ProviderEventID,
			&item.EventType,
			&item.ProviderPaymentID,
			&item.ProviderTransactionID,
			&item.PayloadSHA256,
			&item.Status,
			&item.ErrorMessage,
			&item.ReceivedAt,
			&processedAt,
		); err != nil {
			return nil,
				fmt.Errorf(
					"scan admin payment event: %w",
					err,
				)
		}

		item.ProcessedAt = adminTimePointer(processedAt)

		result = append(
			result,
			item,
		)
	}

	if err := rows.Err(); err != nil {
		return nil,
			fmt.Errorf(
				"iterate admin payment events: %w",
				err,
			)
	}

	return result, nil
}
