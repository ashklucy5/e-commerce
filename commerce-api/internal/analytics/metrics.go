package analytics

import (
	"context"
	"fmt"
	"time"
)

type Overview struct {
	GeneratedAt time.Time `json:"generated_at"`

	OrdersCreated   int64 `json:"orders_created"`
	CollectedOrders int64 `json:"collected_orders"`
	UnitsSold       int64 `json:"units_sold"`
	NewCustomers    int64 `json:"new_customers"`

	GrossCollectedRevenueAmount int64 `json:"gross_collected_revenue_amount"`
	SuccessfulRefundAmount      int64 `json:"successful_refund_amount"`
	NetCollectedRevenueAmount   int64 `json:"net_collected_revenue_amount"`
	DiscountAmount              int64 `json:"discount_amount"`
	AverageOrderValueAmount     int64 `json:"average_order_value_amount"`

	Currency string `json:"currency"`
}

type SalesPoint struct {
	BucketStart string `json:"bucket_start"`

	OrdersCreated   int64 `json:"orders_created"`
	CollectedOrders int64 `json:"collected_orders"`
	NewCustomers    int64 `json:"new_customers"`

	GrossCollectedRevenueAmount int64 `json:"gross_collected_revenue_amount"`
	SuccessfulRefundAmount      int64 `json:"successful_refund_amount"`
	NetCollectedRevenueAmount   int64 `json:"net_collected_revenue_amount"`
}

type TopProduct struct {
	VariantID string `json:"variant_id"`
	ProductID string `json:"product_id"`

	SKU         string `json:"sku"`
	ProductName string `json:"product_name"`

	CollectedOrders int64 `json:"collected_orders"`
	UnitsSold       int64 `json:"units_sold"`

	GrossCollectedMerchandiseAmount int64 `json:"gross_collected_merchandise_amount"`

	Currency string `json:"currency"`
}

func (s *Service) Overview(
	ctx context.Context,
	query Query,
) (
	Overview,
	error,
) {
	queryCtx,
		cancel,
		err :=
		s.queryContext(
			ctx,
		)
	if err != nil {
		return Overview{}, err
	}
	defer cancel()

	var result Overview

	err =
		s.db.QueryRow(
			queryCtx,
			`
				WITH window_orders AS (
					SELECT id
					FROM orders
					WHERE
						created_at >= $1
						AND created_at < $2
						AND currency = $3
				),
				collected_orders AS (
					SELECT
						id,
						total_amount,
						discount_amount
					FROM orders
					WHERE
						paid_at >= $1
						AND paid_at < $2
						AND currency = $3
						AND payment_status IN (
							'paid',
							'cod_collected',
							'refunded'
						)
				)
				SELECT
					(
						SELECT COUNT(*)
						FROM window_orders
					),
					(
						SELECT COUNT(*)
						FROM collected_orders
					),
					(
						SELECT COALESCE(
							SUM(total_amount),
							0
						)
						FROM collected_orders
					),
					(
						SELECT COALESCE(
							SUM(discount_amount),
							0
						)
						FROM collected_orders
					),
					(
						SELECT COALESCE(
							SUM(r.amount),
							0
						)
						FROM refunds r
						WHERE
							r.status = 'succeeded'
							AND r.succeeded_at >= $1
							AND r.succeeded_at < $2
							AND r.currency = $3
					),
					(
						SELECT COUNT(*)
						FROM customers c
						WHERE
							c.created_at >= $1
							AND c.created_at < $2
					),
					(
						SELECT COALESCE(
							SUM(oi.quantity),
							0
						)
						FROM collected_orders co
						JOIN order_items oi
							ON oi.order_id = co.id
					),
					now()
			`,
			query.Start,
			query.End,
			query.Currency,
		).Scan(
			&result.OrdersCreated,
			&result.CollectedOrders,
			&result.GrossCollectedRevenueAmount,
			&result.DiscountAmount,
			&result.SuccessfulRefundAmount,
			&result.NewCustomers,
			&result.UnitsSold,
			&result.GeneratedAt,
		)
	if err != nil {
		return Overview{},
			fmt.Errorf(
				"load analytics overview: %w",
				err,
			)
	}

	result.NetCollectedRevenueAmount =
		result.GrossCollectedRevenueAmount -
			result.SuccessfulRefundAmount

	result.AverageOrderValueAmount =
		roundedAverageAmount(
			result.GrossCollectedRevenueAmount,
			result.CollectedOrders,
		)

	result.Currency =
		query.Currency

	return result, nil
}

func (s *Service) SalesTrend(
	ctx context.Context,
	query Query,
) (
	[]SalesPoint,
	error,
) {
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
				SELECT
					bucket_start,
					SUM(orders_created)::bigint,
					SUM(collected_orders)::bigint,
					SUM(gross_revenue)::bigint,
					SUM(refund_amount)::bigint,
					SUM(new_customers)::bigint
				FROM (
					SELECT
						to_char(
							date_trunc(
								$4,
								o.created_at
									AT TIME ZONE 'Asia/Dhaka'
							),
							'YYYY-MM-DD'
						) AS bucket_start,
						COUNT(*)::bigint
							AS orders_created,
						0::bigint
							AS collected_orders,
						0::bigint
							AS gross_revenue,
						0::bigint
							AS refund_amount,
						0::bigint
							AS new_customers
					FROM orders o
					WHERE
						o.created_at >= $1
						AND o.created_at < $2
						AND o.currency = $3
					GROUP BY 1

					UNION ALL

					SELECT
						to_char(
							date_trunc(
								$4,
								o.paid_at
									AT TIME ZONE 'Asia/Dhaka'
							),
							'YYYY-MM-DD'
						) AS bucket_start,
						0::bigint,
						COUNT(*)::bigint,
						COALESCE(
							SUM(o.total_amount),
							0
						)::bigint,
						0::bigint,
						0::bigint
					FROM orders o
					WHERE
						o.paid_at >= $1
						AND o.paid_at < $2
						AND o.currency = $3
						AND o.payment_status IN (
							'paid',
							'cod_collected',
							'refunded'
						)
					GROUP BY 1

					UNION ALL

					SELECT
						to_char(
							date_trunc(
								$4,
								r.succeeded_at
									AT TIME ZONE 'Asia/Dhaka'
							),
							'YYYY-MM-DD'
						) AS bucket_start,
						0::bigint,
						0::bigint,
						0::bigint,
						COALESCE(
							SUM(r.amount),
							0
						)::bigint,
						0::bigint
					FROM refunds r
					WHERE
						r.status = 'succeeded'
						AND r.succeeded_at >= $1
						AND r.succeeded_at < $2
						AND r.currency = $3
					GROUP BY 1

					UNION ALL

					SELECT
						to_char(
							date_trunc(
								$4,
								c.created_at
									AT TIME ZONE 'Asia/Dhaka'
							),
							'YYYY-MM-DD'
						) AS bucket_start,
						0::bigint,
						0::bigint,
						0::bigint,
						0::bigint,
						COUNT(*)::bigint
					FROM customers c
					WHERE
						c.created_at >= $1
						AND c.created_at < $2
					GROUP BY 1
				) AS aggregated
				GROUP BY bucket_start
				ORDER BY bucket_start
			`,
			query.Start,
			query.End,
			query.Currency,
			query.Granularity,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"load analytics sales trend: %w",
				err,
			)
	}
	defer rows.Close()

	observed :=
		make(
			map[string]SalesPoint,
		)

	for rows.Next() {
		var point SalesPoint

		if err :=
			rows.Scan(
				&point.BucketStart,
				&point.OrdersCreated,
				&point.CollectedOrders,
				&point.GrossCollectedRevenueAmount,
				&point.SuccessfulRefundAmount,
				&point.NewCustomers,
			); err != nil {

			return nil,
				fmt.Errorf(
					"scan analytics sales trend: %w",
					err,
				)
		}

		point.NetCollectedRevenueAmount =
			point.GrossCollectedRevenueAmount -
				point.SuccessfulRefundAmount

		observed[point.BucketStart] =
			point
	}

	if err :=
		rows.Err(); err != nil {

		return nil,
			fmt.Errorf(
				"iterate analytics sales trend: %w",
				err,
			)
	}

	return fillSalesTrend(
			query,
			observed,
		),
		nil
}

func (s *Service) TopProducts(
	ctx context.Context,
	query Query,
	limit int,
) (
	[]TopProduct,
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
				SELECT
					oi.variant_id::text,
					pv.product_id::text,
					MAX(oi.sku),
					MAX(oi.product_name),
					COUNT(*)::bigint,
					SUM(oi.quantity)::bigint,
					ROUND(
						COALESCE(
							SUM(
								CASE
									WHEN
										o.subtotal_amount > 0
									THEN
										oi.line_total_amount::numeric *
										GREATEST(
											o.subtotal_amount -
												o.discount_amount,
											0
										)::numeric /
										o.subtotal_amount::numeric
									ELSE
										0::numeric
								END
							),
							0::numeric
						)
					)::bigint
						AS gross_collected_merchandise_amount
				FROM orders o
				JOIN order_items oi
					ON oi.order_id = o.id
				JOIN product_variants pv
					ON pv.id = oi.variant_id
				WHERE
					o.paid_at >= $1
					AND o.paid_at < $2
					AND o.currency = $3
					AND o.payment_status IN (
						'paid',
						'cod_collected',
						'refunded'
					)
				GROUP BY
					oi.variant_id,
					pv.product_id
				ORDER BY
					SUM(oi.quantity) DESC,
					gross_collected_merchandise_amount DESC,
					oi.variant_id
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
				"load top products analytics: %w",
				err,
			)
	}
	defer rows.Close()

	result :=
		make(
			[]TopProduct,
			0,
			limit,
		)

	for rows.Next() {
		var item TopProduct

		if err :=
			rows.Scan(
				&item.VariantID,
				&item.ProductID,
				&item.SKU,
				&item.ProductName,
				&item.CollectedOrders,
				&item.UnitsSold,
				&item.GrossCollectedMerchandiseAmount,
			); err != nil {

			return nil,
				fmt.Errorf(
					"scan top products analytics: %w",
					err,
				)
		}

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
				"iterate top products analytics: %w",
				err,
			)
	}

	return result, nil
}

func fillSalesTrend(
	query Query,
	observed map[string]SalesPoint,
) []SalesPoint {
	fromLocal, _ :=
		time.ParseInLocation(
			time.DateOnly,
			query.FromDate,
			businessLocation,
		)

	toLocal, _ :=
		time.ParseInLocation(
			time.DateOnly,
			query.ToDate,
			businessLocation,
		)

	cursor :=
		analyticsBucketStart(
			fromLocal,
			query.Granularity,
		)

	endBucket :=
		analyticsBucketStart(
			toLocal,
			query.Granularity,
		)

	stepDays :=
		1

	if query.Granularity ==
		GranularityWeek {

		stepDays = 7
	}

	capacity :=
		int(
			endBucket.Sub(
				cursor,
			).Hours()/24,
		)/
			stepDays +
			1

	if capacity < 0 {
		capacity = 0
	}

	result :=
		make(
			[]SalesPoint,
			0,
			capacity,
		)

	for !cursor.After(
		endBucket,
	) {
		key :=
			cursor.Format(
				time.DateOnly,
			)

		point, exists :=
			observed[key]

		if !exists {
			point =
				SalesPoint{
					BucketStart: key,
				}
		}

		result =
			append(
				result,
				point,
			)

		cursor =
			cursor.AddDate(
				0,
				0,
				stepDays,
			)
	}

	return result
}
