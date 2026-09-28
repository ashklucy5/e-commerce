package returns

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"project.local/commerce-api/internal/platform/pagination"
)

func (r *Repository) ListHistoryForCustomer(
	ctx context.Context,
	customerID string,
	params pagination.Params,
	status string,
) (ReturnHistoryResult, error) {
	const countQuery = `
		SELECT COUNT(*)::bigint
		FROM returns r
		JOIN orders o
			ON o.id = r.order_id
		WHERE
			o.customer_id = $1::uuid
			AND (
				$2 = ''
				OR r.status = $2
			)
	`

	const listQuery = `
		WITH page AS (
			SELECT
				r.id,
				r.return_number,
				r.order_id,
				o.order_number,
				r.status,
				o.currency,
				r.requested_at,
				r.updated_at

			FROM returns r

			JOIN orders o
				ON o.id = r.order_id

			WHERE
				o.customer_id = $1::uuid
				AND (
					$2 = ''
					OR r.status = $2
				)

			ORDER BY
				r.requested_at DESC,
				r.id DESC

			LIMIT $3
			OFFSET $4
		)

		SELECT
			page.id::text,
			page.return_number,
			page.order_id::text,
			page.order_number,
			page.status,
			page.currency,
			COALESCE(item_stats.requested_amount, 0)::bigint,
			COALESCE(item_stats.item_count, 0)::bigint,
			COALESCE(item_stats.quantity_total, 0)::bigint,
			COALESCE(item_stats.first_product_name, ''),
			page.requested_at,
			page.updated_at

		FROM page

		LEFT JOIN LATERAL (
			SELECT
				COALESCE(
					SUM(
						ri.quantity::bigint *
						oi.unit_price_amount
					),
					0
				)::bigint AS requested_amount,

				COUNT(*)::bigint AS item_count,

				COALESCE(
					SUM(ri.quantity),
					0
				)::bigint AS quantity_total,

				COALESCE(
					(
						array_agg(
							oi.product_name
							ORDER BY
								ri.created_at ASC,
								ri.id ASC
						)
					)[1],
					''
				) AS first_product_name

			FROM return_items ri

			JOIN order_items oi
				ON oi.id = ri.order_item_id

			WHERE
				ri.return_id = page.id
		) AS item_stats
			ON true

		ORDER BY
			page.requested_at DESC,
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

		return ReturnHistoryResult{},
			fmt.Errorf(
				"count customer return history: %w",
				err,
			)
	}

	rows, err :=
		results.Query()
	if err != nil {
		return ReturnHistoryResult{},
			fmt.Errorf(
				"query customer return history: %w",
				err,
			)
	}
	defer rows.Close()

	items :=
		make(
			[]ReturnHistoryItem,
			0,
			params.Limit,
		)

	for rows.Next() {
		var item ReturnHistoryItem

		if err :=
			rows.Scan(
				&item.ID,
				&item.ReturnNumber,
				&item.OrderID,
				&item.OrderNumber,
				&item.Status,
				&item.Currency,
				&item.RequestedAmount,
				&item.ItemCount,
				&item.QuantityTotal,
				&item.FirstProductName,
				&item.RequestedAt,
				&item.UpdatedAt,
			); err != nil {

			return ReturnHistoryResult{},
				fmt.Errorf(
					"scan customer return history: %w",
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

		return ReturnHistoryResult{},
			fmt.Errorf(
				"iterate customer return history: %w",
				err,
			)
	}

	return ReturnHistoryResult{
		Items: items,

		Meta: pagination.NewMeta(
			params,
			total,
		),
	}, nil
}
