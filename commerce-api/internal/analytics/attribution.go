package analytics

import (
	"context"
	"fmt"
)

type PromotionAttribution struct {
	PromotionID string `json:"promotion_id"`
	Name        string `json:"name"`
	Code        string `json:"code,omitempty"`

	CollectedOrders int64 `json:"collected_orders"`
	UnitsSold       int64 `json:"units_sold"`

	GrossCollectedRevenueAmount  int64  `json:"gross_collected_revenue_amount"`
	DiscountAmount               int64  `json:"discount_amount"`
	AverageGrossOrderValueAmount int64  `json:"average_gross_order_value_amount"`
	Currency                     string `json:"currency"`
}

func (s *Service) PromotionAttribution(
	ctx context.Context,
	query Query,
	limit int,
) (
	[]PromotionAttribution,
	error,
) {
	if limit <= 0 ||
		limit > 50 {

		return nil,
			fmt.Errorf(
				"%w: limit must be between 1 and 50",
				ErrInvalidLimit,
			)
	}

	queryCtx,
		cancel,
		err :=
		s.queryContext(
			ctx,
		)
	if err != nil {
		return nil, err
	}
	defer cancel()

	rows, err :=
		s.db.Query(
			queryCtx,
			`
				WITH collected_orders AS (
					SELECT
						o.id,
						o.promotion_id,
						o.promotion_code,
						o.total_amount,
						o.discount_amount
					FROM orders o
					WHERE
						o.promotion_id IS NOT NULL
						AND o.paid_at >= $1
						AND o.paid_at < $2
						AND o.currency = $3
						AND o.payment_status IN (
							'paid',
							'cod_collected',
							'refunded'
						)
				),
				order_units AS (
					SELECT
						oi.order_id,
						SUM(oi.quantity)::bigint
							AS units_sold
					FROM order_items oi
					JOIN collected_orders co
						ON co.id = oi.order_id
					GROUP BY oi.order_id
				)
				SELECT
					co.promotion_id::text,
					COALESCE(
						p.name,
						'Unknown promotion'
					),
					COALESCE(
						NULLIF(
							co.promotion_code,
							''
						),
						p.code,
						''
					),
					COUNT(*)::bigint,
					COALESCE(
						SUM(
							ou.units_sold
						),
						0
					)::bigint,
					SUM(
						co.total_amount
					)::bigint,
					SUM(
						co.discount_amount
					)::bigint
				FROM collected_orders co
				LEFT JOIN promotions p
					ON p.id =
						co.promotion_id
				LEFT JOIN order_units ou
					ON ou.order_id =
						co.id
				GROUP BY
					co.promotion_id,
					p.name,
					COALESCE(
						NULLIF(
							co.promotion_code,
							''
						),
						p.code,
						''
					)
				ORDER BY
					SUM(
						co.total_amount
					) DESC,
					co.promotion_id
				LIMIT $4
			`,
			query.Start,
			query.End,
			query.Currency,
			limit,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"load promotion attribution analytics: %w",
				err,
			)
	}
	defer rows.Close()

	result :=
		make(
			[]PromotionAttribution,
			0,
			limit,
		)

	for rows.Next() {
		var item PromotionAttribution

		if err :=
			rows.Scan(
				&item.PromotionID,
				&item.Name,
				&item.Code,
				&item.CollectedOrders,
				&item.UnitsSold,
				&item.GrossCollectedRevenueAmount,
				&item.DiscountAmount,
			); err != nil {

			return nil,
				fmt.Errorf(
					"scan promotion attribution analytics: %w",
					err,
				)
		}

		item.AverageGrossOrderValueAmount =
			roundedAverageAmount(
				item.GrossCollectedRevenueAmount,
				item.CollectedOrders,
			)

		item.Currency =
			query.Currency

		result =
			append(
				result,
				item,
			)
	}

	if err :=
		rows.Err(); err != nil {

		return nil,
			fmt.Errorf(
				"iterate promotion attribution analytics: %w",
				err,
			)
	}

	return result, nil
}
