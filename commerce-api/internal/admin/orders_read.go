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

func (s *Service) ListOrders(
	ctx context.Context,
	params platformpagination.Params,
	filter OrderReadFilter,
) (
	OrderListResult,
	error,
) {
	queryCtx, cancel := adminReadContext(ctx)
	defer cancel()

	rows, err := s.db.Query(
		queryCtx,
		`
			SELECT
				o.id::text,
				o.order_number,
				COALESCE(o.customer_id::text, ''),
				o.customer_name,
				o.customer_phone,
				o.status,
				o.payment_status,
				o.payment_method,
				o.currency,
				o.total_amount,
				o.delivery_method,
				o.created_at,
				o.updated_at,
				COUNT(*) OVER()::bigint
			FROM orders o
			WHERE
				($1 = '' OR o.status = $1)
				AND ($2 = '' OR o.payment_status = $2)
				AND (
					$3 = ''
					OR o.order_number = upper($3)
					OR o.customer_phone = $3
				)
			ORDER BY o.created_at DESC, o.id DESC
			LIMIT $4 OFFSET $5
		`,
		filter.Status,
		filter.PaymentStatus,
		filter.Query,
		params.Limit,
		params.Offset(),
	)
	if err != nil {
		return OrderListResult{},
			fmt.Errorf(
				"list admin orders: %w",
				err,
			)
	}
	defer rows.Close()

	items := make(
		[]OrderListItem,
		0,
		params.Limit,
	)

	var total int64

	for rows.Next() {
		var item OrderListItem

		if err := rows.Scan(
			&item.ID,
			&item.OrderNumber,
			&item.CustomerID,
			&item.CustomerName,
			&item.CustomerPhone,
			&item.Status,
			&item.PaymentStatus,
			&item.PaymentMethod,
			&item.Currency,
			&item.TotalAmount,
			&item.DeliveryMethod,
			&item.CreatedAt,
			&item.UpdatedAt,
			&total,
		); err != nil {
			return OrderListResult{},
				fmt.Errorf(
					"scan admin order: %w",
					err,
				)
		}

		items = append(
			items,
			item,
		)
	}

	if err := rows.Err(); err != nil {
		return OrderListResult{},
			fmt.Errorf(
				"iterate admin orders: %w",
				err,
			)
	}

	return OrderListResult{
		Items: items,
		Meta: platformpagination.NewMeta(
			params,
			total,
		),
	}, nil
}

func (s *Service) GetOrder(
	ctx context.Context,
	orderID string,
) (
	OrderDetail,
	error,
) {
	queryCtx, cancel := adminReadContext(ctx)
	defer cancel()

	var result OrderDetail

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
			if err := scanOrderDetail(
				ctx,
				tx,
				orderID,
				&result,
			); err != nil {
				return err
			}

			items, err := listOrderItemsRead(
				ctx,
				tx,
				orderID,
			)
			if err != nil {
				return err
			}

			payments, err := listOrderPaymentSummaries(
				ctx,
				tx,
				orderID,
			)
			if err != nil {
				return err
			}

			returns, err := listOrderReturnSummaries(
				ctx,
				tx,
				orderID,
			)
			if err != nil {
				return err
			}

			refunds, err := listRefundSummaries(
				ctx,
				tx,
				"order_id",
				orderID,
			)
			if err != nil {
				return err
			}

			result.Items = items
			result.Payments = payments
			result.Returns = returns
			result.Refunds = refunds

			return nil
		},
	)
	if err != nil {
		return OrderDetail{}, err
	}

	return result, nil
}

func scanOrderDetail(
	ctx context.Context,
	querier adminReadQuerier,
	orderID string,
	result *OrderDetail,
) error {
	var paymentDueAt pgtype.Timestamptz
	var paidAt pgtype.Timestamptz
	var confirmedAt pgtype.Timestamptz
	var processingAt pgtype.Timestamptz
	var shippedAt pgtype.Timestamptz
	var deliveredAt pgtype.Timestamptz
	var completedAt pgtype.Timestamptz
	var cancelledAt pgtype.Timestamptz

	err := querier.QueryRow(
		ctx,
		`
			SELECT
				o.id::text,
				o.order_number,
				COALESCE(o.customer_id::text, ''),
				o.customer_name,
				o.customer_phone,
				o.status,
				o.payment_status,
				o.payment_method,
				o.currency,
				o.total_amount,
				o.delivery_method,
				o.created_at,
				o.updated_at,
				o.order_type,
				COALESCE(o.checkout_id::text, ''),
				COALESCE(o.cart_id::text, ''),
				COALESCE(o.customer_email, ''),
				o.subtotal_amount,
				o.discount_amount,
				COALESCE(o.promotion_id::text, ''),
				COALESCE(o.promotion_code, ''),
				o.shipping_amount,
				o.shipping_address_line1,
				COALESCE(o.shipping_address_line2, ''),
				o.shipping_city,
				o.shipping_area,
				COALESCE(o.shipping_postal_code, ''),
				o.payment_due_at,
				o.paid_at,
				o.confirmed_at,
				o.processing_at,
				o.shipped_at,
				o.delivered_at,
				o.completed_at,
				o.cancelled_at,
				COALESCE(o.cancellation_reason, ''),
				COALESCE(o.cancelled_by, '')
			FROM orders o
			WHERE o.id = $1::uuid
		`,
		orderID,
	).Scan(
		&result.ID,
		&result.OrderNumber,
		&result.CustomerID,
		&result.CustomerName,
		&result.CustomerPhone,
		&result.Status,
		&result.PaymentStatus,
		&result.PaymentMethod,
		&result.Currency,
		&result.TotalAmount,
		&result.DeliveryMethod,
		&result.CreatedAt,
		&result.UpdatedAt,
		&result.OrderType,
		&result.CheckoutID,
		&result.CartID,
		&result.CustomerEmail,
		&result.SubtotalAmount,
		&result.DiscountAmount,
		&result.PromotionID,
		&result.PromotionCode,
		&result.ShippingAmount,
		&result.ShippingAddressLine1,
		&result.ShippingAddressLine2,
		&result.ShippingCity,
		&result.ShippingArea,
		&result.ShippingPostalCode,
		&paymentDueAt,
		&paidAt,
		&confirmedAt,
		&processingAt,
		&shippedAt,
		&deliveredAt,
		&completedAt,
		&cancelledAt,
		&result.CancellationReason,
		&result.CancelledBy,
	)

	if errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return ErrAdminOrderNotFound
	}

	if err != nil {
		return fmt.Errorf(
			"get admin order: %w",
			err,
		)
	}

	result.PaymentDueAt = adminTimePointer(paymentDueAt)
	result.PaidAt = adminTimePointer(paidAt)
	result.ConfirmedAt = adminTimePointer(confirmedAt)
	result.ProcessingAt = adminTimePointer(processingAt)
	result.ShippedAt = adminTimePointer(shippedAt)
	result.DeliveredAt = adminTimePointer(deliveredAt)
	result.CompletedAt = adminTimePointer(completedAt)
	result.CancelledAt = adminTimePointer(cancelledAt)

	return nil
}

func listOrderItemsRead(
	ctx context.Context,
	querier adminReadQuerier,
	orderID string,
) (
	[]OrderItemRead,
	error,
) {
	rows, err := querier.Query(
		ctx,
		`
			SELECT
				id::text,
				variant_id::text,
				sku,
				product_name,
				quantity,
				minimum_order_quantity,
				unit_price_amount,
				line_total_amount,
				currency
			FROM order_items
			WHERE order_id = $1::uuid
			ORDER BY created_at, id
		`,
		orderID,
	)
	if err != nil {
		return nil,
			fmt.Errorf(
				"list admin order items: %w",
				err,
			)
	}
	defer rows.Close()

	result := make(
		[]OrderItemRead,
		0,
	)

	for rows.Next() {
		var item OrderItemRead

		if err := rows.Scan(
			&item.ID,
			&item.VariantID,
			&item.SKU,
			&item.ProductName,
			&item.Quantity,
			&item.MinimumOrderQuantity,
			&item.UnitPriceAmount,
			&item.LineTotalAmount,
			&item.Currency,
		); err != nil {
			return nil,
				fmt.Errorf(
					"scan admin order item: %w",
					err,
				)
		}

		result = append(
			result,
			item,
		)
	}

	if err := rows.Err(); err != nil {
		return nil,
			fmt.Errorf(
				"iterate admin order items: %w",
				err,
			)
	}

	return result, nil
}

func listOrderPaymentSummaries(
	ctx context.Context,
	querier adminReadQuerier,
	orderID string,
) (
	[]PaymentSummary,
	error,
) {
	rows, err := querier.Query(
		ctx,
		`
			SELECT
				id::text,
				provider,
				status,
				amount,
				currency,
				provider_payment_id,
				COALESCE(provider_transaction_id, ''),
				paid_at,
				failed_at,
				created_at
			FROM payments
			WHERE order_id = $1::uuid
			ORDER BY created_at DESC, id DESC
		`,
		orderID,
	)
	if err != nil {
		return nil,
			fmt.Errorf(
				"list admin order payments: %w",
				err,
			)
	}
	defer rows.Close()

	result := make(
		[]PaymentSummary,
		0,
	)

	for rows.Next() {
		var item PaymentSummary
		var paidAt pgtype.Timestamptz
		var failedAt pgtype.Timestamptz

		if err := rows.Scan(
			&item.ID,
			&item.Provider,
			&item.Status,
			&item.Amount,
			&item.Currency,
			&item.ProviderPaymentID,
			&item.ProviderTransactionID,
			&paidAt,
			&failedAt,
			&item.CreatedAt,
		); err != nil {
			return nil,
				fmt.Errorf(
					"scan admin order payment: %w",
					err,
				)
		}

		item.PaidAt = adminTimePointer(paidAt)
		item.FailedAt = adminTimePointer(failedAt)

		result = append(
			result,
			item,
		)
	}

	if err := rows.Err(); err != nil {
		return nil,
			fmt.Errorf(
				"iterate admin order payments: %w",
				err,
			)
	}

	return result, nil
}

func listOrderReturnSummaries(
	ctx context.Context,
	querier adminReadQuerier,
	orderID string,
) (
	[]ReturnSummary,
	error,
) {
	rows, err := querier.Query(
		ctx,
		`
			SELECT
				id::text,
				return_number,
				status,
				requested_at
			FROM returns
			WHERE order_id = $1::uuid
			ORDER BY created_at DESC, id DESC
		`,
		orderID,
	)
	if err != nil {
		return nil,
			fmt.Errorf(
				"list admin order returns: %w",
				err,
			)
	}
	defer rows.Close()

	result := make(
		[]ReturnSummary,
		0,
	)

	for rows.Next() {
		var item ReturnSummary

		if err := rows.Scan(
			&item.ID,
			&item.ReturnNumber,
			&item.Status,
			&item.RequestedAt,
		); err != nil {
			return nil,
				fmt.Errorf(
					"scan admin order return: %w",
					err,
				)
		}

		result = append(
			result,
			item,
		)
	}

	if err := rows.Err(); err != nil {
		return nil,
			fmt.Errorf(
				"iterate admin order returns: %w",
				err,
			)
	}

	return result, nil
}
