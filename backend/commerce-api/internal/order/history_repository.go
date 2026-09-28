package order

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"project.local/commerce-api/internal/platform/pagination"
)

func (r *Repository) ListForCustomer(
	ctx context.Context,
	customerID string,
	params pagination.Params,
	status string,
) (OrderHistoryResult, error) {
	const countQuery = `
		SELECT COUNT(*)::bigint
		FROM orders o
		WHERE
			o.customer_id = $1::uuid
			AND (
				$2 = ''
				OR o.status = $2
			)
	`

	const listQuery = `
		WITH page AS (
			SELECT
				o.id,
				o.order_number,
				o.order_type,
				o.status,
				o.payment_status,
				o.payment_method,
				o.currency,
				o.subtotal_amount,
				o.discount_amount,
				o.shipping_amount,
				o.total_amount,
				o.delivery_method,
				o.created_at,
				o.updated_at

			FROM orders o

			WHERE
				o.customer_id = $1::uuid
				AND (
					$2 = ''
					OR o.status = $2
				)

			ORDER BY
				o.created_at DESC,
				o.id DESC

			LIMIT $3
			OFFSET $4
		)

		SELECT
			page.id::text,
			page.order_number,
			page.order_type,
			page.status,
			page.payment_status,
			page.payment_method,
			page.currency,
			page.subtotal_amount,
			page.discount_amount,
			page.shipping_amount,
			page.total_amount,
			page.delivery_method,
			COALESCE(item_stats.item_count, 0)::bigint,
			COALESCE(item_stats.quantity_total, 0)::bigint,
			COALESCE(item_stats.first_product_name, ''),
			page.created_at,
			page.updated_at

		FROM page

		LEFT JOIN LATERAL (
			SELECT
				COUNT(*)::bigint AS item_count,

				COALESCE(
					SUM(oi.quantity),
					0
				)::bigint AS quantity_total,

				COALESCE(
					(
						array_agg(
							oi.product_name
							ORDER BY
								oi.created_at ASC,
								oi.id ASC
						)
					)[1],
					''
				) AS first_product_name

			FROM order_items oi

			WHERE
				oi.order_id = page.id
		) AS item_stats
			ON true

		ORDER BY
			page.created_at DESC,
			page.id DESC
	`

	batch :=
		&pgx.Batch{}

	batch.Queue(
		countQuery,
		customerID,
		status,
	)

	batch.Queue(
		listQuery,
		customerID,
		status,
		params.Limit,
		params.Offset(),
	)

	results :=
		r.db.SendBatch(
			ctx,
			batch,
		)

	defer results.Close()

	var total int64

	if err :=
		results.QueryRow().Scan(
			&total,
		); err != nil {

		return OrderHistoryResult{},
			fmt.Errorf(
				"count customer order history: %w",
				err,
			)
	}

	rows, err :=
		results.Query()
	if err != nil {
		return OrderHistoryResult{},
			fmt.Errorf(
				"query customer order history: %w",
				err,
			)
	}
	defer rows.Close()

	items :=
		make(
			[]OrderHistoryItem,
			0,
			params.Limit,
		)

	for rows.Next() {
		var item OrderHistoryItem

		if err :=
			rows.Scan(
				&item.ID,
				&item.OrderNumber,
				&item.OrderType,
				&item.Status,
				&item.PaymentStatus,
				&item.PaymentMethod,
				&item.Currency,
				&item.SubtotalAmount,
				&item.DiscountAmount,
				&item.ShippingAmount,
				&item.TotalAmount,
				&item.DeliveryMethod,
				&item.ItemCount,
				&item.QuantityTotal,
				&item.FirstProductName,
				&item.CreatedAt,
				&item.UpdatedAt,
			); err != nil {

			return OrderHistoryResult{},
				fmt.Errorf(
					"scan customer order history: %w",
					err,
				)
		}

		items =
			append(
				items,
				item,
			)
	}

	if err :=
		rows.Err(); err != nil {

		return OrderHistoryResult{},
			fmt.Errorf(
				"iterate customer order history: %w",
				err,
			)
	}

	return OrderHistoryResult{
		Items: items,

		Meta: pagination.NewMeta(
			params,
			total,
		),
	}, nil
}
