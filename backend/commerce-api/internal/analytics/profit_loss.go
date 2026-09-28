package analytics

import (
	"context"
	"fmt"
	"time"
)

type ExpenseCategoryAmount struct {
	Category string `json:"category"`
	Entries  int64  `json:"entries"`
	Amount   int64  `json:"amount"`
}

type ProfitLoss struct {
	GeneratedAt time.Time `json:"generated_at"`

	CollectedOrders int64 `json:"collected_orders"`
	UnitsSold       int64 `json:"units_sold"`

	GrossMerchandiseRevenueAmount int64 `json:"gross_merchandise_revenue_amount"`
	DiscountAmount                int64 `json:"discount_amount"`
	NetMerchandiseRevenueAmount   int64 `json:"net_merchandise_revenue_amount"`
	ShippingRevenueAmount         int64 `json:"shipping_revenue_amount"`
	GrossCollectedRevenueAmount   int64 `json:"gross_collected_revenue_amount"`
	SuccessfulRefundAmount        int64 `json:"successful_refund_amount"`
	NetCollectedRevenueAmount     int64 `json:"net_collected_revenue_amount"`

	CostedUnits               int64 `json:"costed_units"`
	MissingCostUnits          int64 `json:"missing_cost_units"`
	SalesCOGSCoverageBPS      int64 `json:"sales_cogs_coverage_bps"`
	GrossCOGSAmount           int64 `json:"gross_cogs_amount"`
	RestockedUnits            int64 `json:"restocked_units"`
	RestockCostedUnits        int64 `json:"restock_costed_units"`
	RestockMissingCostUnits   int64 `json:"restock_missing_cost_units"`
	RestockCOGSCoverageBPS    int64 `json:"restock_cogs_coverage_bps"`
	RestockCOGSRecoveryAmount int64 `json:"restock_cogs_recovery_amount"`
	NetCOGSAmount             int64 `json:"net_cogs_amount"`
	COGSCoverageBPS           int64 `json:"cogs_coverage_bps"`

	ReturnReceivedUnits            int64 `json:"return_received_units"`
	NonRestockedReturnUnits        int64 `json:"non_restocked_return_units"`
	KnownReturnInventoryLossAmount int64 `json:"known_return_inventory_loss_amount"`
	ReturnLossMissingCostUnits     int64 `json:"return_loss_missing_cost_units"`

	KnownGrossProfitAmount int64  `json:"known_gross_profit_amount"`
	GrossProfitAmount      *int64 `json:"gross_profit_amount"`
	GrossMarginBPS         *int64 `json:"gross_margin_bps"`
	ProfitComplete         bool   `json:"profit_complete"`

	RecordedExpenseEntries int64 `json:"recorded_expense_entries"`
	RecordedExpensesAmount int64 `json:"recorded_expenses_amount"`

	ExpenseCategories []ExpenseCategoryAmount `json:"expense_categories"`

	ForeignCurrencyExpenseEntriesExcluded int64 `json:"foreign_currency_expense_entries_excluded"`

	NetProfitAfterRecordedExpensesAmount *int64 `json:"net_profit_after_recorded_expenses_amount"`

	ExpenseBasis string `json:"expense_basis"`

	Currency string `json:"currency"`

	Warnings []string `json:"warnings"`
}

type ProfitLossPoint struct {
	BucketStart string `json:"bucket_start"`

	CollectedOrders int64 `json:"collected_orders"`
	UnitsSold       int64 `json:"units_sold"`

	GrossMerchandiseRevenueAmount int64 `json:"gross_merchandise_revenue_amount"`
	DiscountAmount                int64 `json:"discount_amount"`
	NetMerchandiseRevenueAmount   int64 `json:"net_merchandise_revenue_amount"`
	ShippingRevenueAmount         int64 `json:"shipping_revenue_amount"`
	GrossCollectedRevenueAmount   int64 `json:"gross_collected_revenue_amount"`
	SuccessfulRefundAmount        int64 `json:"successful_refund_amount"`
	NetCollectedRevenueAmount     int64 `json:"net_collected_revenue_amount"`

	CostedUnits             int64 `json:"costed_units"`
	MissingCostUnits        int64 `json:"missing_cost_units"`
	RestockedUnits          int64 `json:"restocked_units"`
	RestockCostedUnits      int64 `json:"restock_costed_units"`
	RestockMissingCostUnits int64 `json:"restock_missing_cost_units"`

	GrossCOGSAmount           int64 `json:"gross_cogs_amount"`
	RestockCOGSRecoveryAmount int64 `json:"restock_cogs_recovery_amount"`
	NetCOGSAmount             int64 `json:"net_cogs_amount"`
	COGSCoverageBPS           int64 `json:"cogs_coverage_bps"`

	KnownGrossProfitAmount int64  `json:"known_gross_profit_amount"`
	GrossProfitAmount      *int64 `json:"gross_profit_amount"`
	GrossMarginBPS         *int64 `json:"gross_margin_bps"`
	ProfitComplete         bool   `json:"profit_complete"`

	RecordedExpensesAmount int64 `json:"recorded_expenses_amount"`

	NetProfitAfterRecordedExpensesAmount *int64 `json:"net_profit_after_recorded_expenses_amount"`
}

type ProductProfitability struct {
	VariantID string `json:"variant_id"`
	ProductID string `json:"product_id"`

	SKU         string `json:"sku"`
	ProductName string `json:"product_name"`

	CollectedOrders int64 `json:"collected_orders"`
	UnitsSold       int64 `json:"units_sold"`

	GrossMerchandiseRevenueAmount int64 `json:"gross_merchandise_revenue_amount"`
	NetMerchandiseRevenueAmount   int64 `json:"net_merchandise_revenue_amount"`

	CostedUnits      int64 `json:"costed_units"`
	MissingCostUnits int64 `json:"missing_cost_units"`
	COGSCoverageBPS  int64 `json:"cogs_coverage_bps"`
	KnownCOGSAmount  int64 `json:"known_cogs_amount"`

	GrossProfitBeforeRefundsAmount *int64 `json:"gross_profit_before_refunds_amount"`
	GrossMarginBeforeRefundsBPS    *int64 `json:"gross_margin_before_refunds_bps"`
	ProfitComplete                 bool   `json:"profit_complete"`

	Currency string `json:"currency"`
}

// Cost resolution for profitability is intentionally conservative:
//  1. use the immutable order-item snapshot when it exists in the reporting currency;
//  2. otherwise use the latest recorded SKU buying cost effective at order creation;
//  3. otherwise leave the cost unknown.
//
// A snapshot in another currency is never replaced or converted implicitly.
func (s *Service) ProfitLoss(
	ctx context.Context,
	query Query,
) (
	ProfitLoss,
	error,
) {
	queryCtx, cancel, err :=
		s.queryContext(
			ctx,
		)
	if err != nil {
		return ProfitLoss{}, err
	}

	defer cancel()

	var result ProfitLoss

	err =
		s.db.QueryRow(
			queryCtx,
			`
				WITH collected_orders AS (
					SELECT
						id,
						created_at,
						subtotal_amount,
						discount_amount,
						shipping_amount,
						total_amount
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
				),

				order_totals AS (
					SELECT
						COUNT(*)::bigint
							AS collected_orders,

						COALESCE(
							SUM(subtotal_amount),
							0
						)::bigint
							AS gross_merchandise,

						COALESCE(
							SUM(discount_amount),
							0
						)::bigint
							AS discounts,

						COALESCE(
							SUM(shipping_amount),
							0
						)::bigint
							AS shipping,

						COALESCE(
							SUM(total_amount),
							0
						)::bigint
							AS gross_collected

					FROM collected_orders
				),

				costed_items AS (
					SELECT
						oi.quantity,

						CASE
							WHEN
								oi.unit_cost_amount IS NOT NULL
								AND oi.cost_currency = $3
							THEN oi.unit_cost_amount

							WHEN
								oi.unit_cost_amount IS NULL
							THEN historical_cost.unit_cost_amount

							ELSE NULL
						END AS effective_unit_cost_amount

					FROM collected_orders co

					JOIN order_items oi
						ON oi.order_id = co.id

					LEFT JOIN LATERAL (
						SELECT
							h.unit_cost_amount
						FROM finance_variant_cost_history h
						WHERE
							oi.unit_cost_amount IS NULL
							AND h.variant_id = oi.variant_id
							AND h.currency = $3
							AND h.effective_at <= co.created_at
						ORDER BY
							h.effective_at DESC,
							h.created_at DESC,
							h.id DESC
						LIMIT 1
					) AS historical_cost
						ON true
				),

				item_totals AS (
					SELECT
						COALESCE(
							SUM(quantity),
							0
						)::bigint
							AS units_sold,

						COALESCE(
							SUM(
								CASE
									WHEN effective_unit_cost_amount IS NOT NULL
									THEN quantity
									ELSE 0
								END
							),
							0
						)::bigint
							AS costed_units,

						COALESCE(
							SUM(
								CASE
									WHEN effective_unit_cost_amount IS NULL
									THEN quantity
									ELSE 0
								END
							),
							0
						)::bigint
							AS missing_cost_units,

						COALESCE(
							SUM(
								CASE
									WHEN effective_unit_cost_amount IS NOT NULL
									THEN
										effective_unit_cost_amount::numeric *
										quantity::numeric
									ELSE 0::numeric
								END
							),
							0::numeric
						)::bigint
							AS gross_cogs

					FROM costed_items
				),

				refund_totals AS (
					SELECT
						COALESCE(
							SUM(amount),
							0
						)::bigint
							AS refunds

					FROM refunds

					WHERE
						status = 'succeeded'
						AND succeeded_at >= $1
						AND succeeded_at < $2
						AND currency = $3
				),

				costed_returns AS (
					SELECT
						ri.received_quantity,
						ri.restock_quantity,

						CASE
							WHEN
								oi.unit_cost_amount IS NOT NULL
								AND oi.cost_currency = $3
							THEN oi.unit_cost_amount

							WHEN
								oi.unit_cost_amount IS NULL
							THEN historical_cost.unit_cost_amount

							ELSE NULL
						END AS effective_unit_cost_amount

					FROM returns r

					JOIN return_items ri
						ON ri.return_id = r.id

					JOIN order_items oi
						ON oi.id = ri.order_item_id

					JOIN orders o
						ON o.id = r.order_id

					LEFT JOIN LATERAL (
						SELECT
							h.unit_cost_amount
						FROM finance_variant_cost_history h
						WHERE
							oi.unit_cost_amount IS NULL
							AND h.variant_id = oi.variant_id
							AND h.currency = $3
							AND h.effective_at <= o.created_at
						ORDER BY
							h.effective_at DESC,
							h.created_at DESC,
							h.id DESC
						LIMIT 1
					) AS historical_cost
						ON true

					WHERE
						r.inspected_at >= $1
						AND r.inspected_at < $2
						AND o.currency = $3
				),

				return_totals AS (
					SELECT
						COALESCE(
							SUM(received_quantity),
							0
						)::bigint
							AS received_units,

						COALESCE(
							SUM(restock_quantity),
							0
						)::bigint
							AS restocked_units,

						COALESCE(
							SUM(
								CASE
									WHEN effective_unit_cost_amount IS NOT NULL
									THEN restock_quantity
									ELSE 0
								END
							),
							0
						)::bigint
							AS restock_costed_units,

						COALESCE(
							SUM(
								CASE
									WHEN effective_unit_cost_amount IS NULL
									THEN restock_quantity
									ELSE 0
								END
							),
							0
						)::bigint
							AS restock_missing_cost_units,

						COALESCE(
							SUM(
								CASE
									WHEN effective_unit_cost_amount IS NOT NULL
									THEN
										effective_unit_cost_amount::numeric *
										restock_quantity::numeric
									ELSE 0::numeric
								END
							),
							0::numeric
						)::bigint
							AS restock_recovery,

						COALESCE(
							SUM(
								CASE
									WHEN effective_unit_cost_amount IS NOT NULL
									THEN
										effective_unit_cost_amount::numeric *
										(
											received_quantity -
											restock_quantity
										)::numeric
									ELSE 0::numeric
								END
							),
							0::numeric
						)::bigint
							AS known_return_loss,

						COALESCE(
							SUM(
								CASE
									WHEN effective_unit_cost_amount IS NULL
									THEN
										received_quantity -
										restock_quantity
									ELSE 0
								END
							),
							0
						)::bigint
							AS return_loss_missing_cost_units

					FROM costed_returns
				),

				expense_totals AS (
					SELECT
						COUNT(*)::bigint
							AS entries,

						COALESCE(
							SUM(amount),
							0
						)::bigint
							AS amount

					FROM finance_expenses

					WHERE
						status = 'active'
						AND occurred_at >= $1
						AND occurred_at < $2
						AND currency = $3
				),

				foreign_expenses AS (
					SELECT
						COUNT(*)::bigint
							AS entries

					FROM finance_expenses

					WHERE
						status = 'active'
						AND occurred_at >= $1
						AND occurred_at < $2
						AND currency <> $3
				)

				SELECT
					ot.collected_orders,
					it.units_sold,

					ot.gross_merchandise,
					ot.discounts,
					ot.shipping,
					ot.gross_collected,

					rf.refunds,

					it.costed_units,
					it.missing_cost_units,
					it.gross_cogs,

					rt.received_units,
					rt.restocked_units,
					rt.restock_costed_units,
					rt.restock_missing_cost_units,
					rt.restock_recovery,
					rt.known_return_loss,
					rt.return_loss_missing_cost_units,

					ex.entries,
					ex.amount,

					fx.entries,

					now()

				FROM order_totals ot

				CROSS JOIN item_totals it
				CROSS JOIN refund_totals rf
				CROSS JOIN return_totals rt
				CROSS JOIN expense_totals ex
				CROSS JOIN foreign_expenses fx
			`,
			query.Start,
			query.End,
			query.Currency,
		).Scan(
			&result.CollectedOrders,
			&result.UnitsSold,

			&result.GrossMerchandiseRevenueAmount,
			&result.DiscountAmount,
			&result.ShippingRevenueAmount,
			&result.GrossCollectedRevenueAmount,

			&result.SuccessfulRefundAmount,

			&result.CostedUnits,
			&result.MissingCostUnits,
			&result.GrossCOGSAmount,

			&result.ReturnReceivedUnits,
			&result.RestockedUnits,
			&result.RestockCostedUnits,
			&result.RestockMissingCostUnits,
			&result.RestockCOGSRecoveryAmount,
			&result.KnownReturnInventoryLossAmount,
			&result.ReturnLossMissingCostUnits,

			&result.RecordedExpenseEntries,
			&result.RecordedExpensesAmount,

			&result.ForeignCurrencyExpenseEntriesExcluded,

			&result.GeneratedAt,
		)
	if err != nil {
		return ProfitLoss{},
			fmt.Errorf(
				"load profit/loss analytics: %w",
				err,
			)
	}

	rows, err :=
		s.db.Query(
			queryCtx,
			`
				SELECT
					category,
					COUNT(*)::bigint,
					COALESCE(
						SUM(amount),
						0
					)::bigint

				FROM finance_expenses

				WHERE
					status = 'active'
					AND occurred_at >= $1
					AND occurred_at < $2
					AND currency = $3

				GROUP BY category

				ORDER BY category
			`,
			query.Start,
			query.End,
			query.Currency,
		)
	if err != nil {
		return ProfitLoss{},
			fmt.Errorf(
				"load profit/loss expense categories: %w",
				err,
			)
	}

	defer rows.Close()

	result.ExpenseCategories =
		make(
			[]ExpenseCategoryAmount,
			0,
		)

	for rows.Next() {
		var item ExpenseCategoryAmount

		if err :=
			rows.Scan(
				&item.Category,
				&item.Entries,
				&item.Amount,
			); err != nil {

			return ProfitLoss{},
				fmt.Errorf(
					"scan profit/loss expense category: %w",
					err,
				)
		}

		result.ExpenseCategories =
			append(
				result.ExpenseCategories,
				item,
			)
	}

	if err :=
		rows.Err(); err != nil {

		return ProfitLoss{},
			fmt.Errorf(
				"iterate profit/loss expense categories: %w",
				err,
			)
	}

	finalizeProfitLoss(
		&result,
		query.Currency,
	)

	return result, nil
}

func (s *Service) ProfitLossTrend(
	ctx context.Context,
	query Query,
) (
	[]ProfitLossPoint,
	error,
) {
	queryCtx, cancel, err :=
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
				WITH costed_sales AS (
					SELECT
						to_char(
							date_trunc(
								$4,
								o.paid_at
									AT TIME ZONE 'Asia/Dhaka'
							),
							'YYYY-MM-DD'
						) AS bucket_start,

						oi.quantity,

						CASE
							WHEN
								oi.unit_cost_amount IS NOT NULL
								AND oi.cost_currency = $3
							THEN oi.unit_cost_amount

							WHEN
								oi.unit_cost_amount IS NULL
							THEN historical_cost.unit_cost_amount

							ELSE NULL
						END AS effective_unit_cost_amount

					FROM orders o

					JOIN order_items oi
						ON oi.order_id = o.id

					LEFT JOIN LATERAL (
						SELECT
							h.unit_cost_amount
						FROM finance_variant_cost_history h
						WHERE
							oi.unit_cost_amount IS NULL
							AND h.variant_id = oi.variant_id
							AND h.currency = $3
							AND h.effective_at <= o.created_at
						ORDER BY
							h.effective_at DESC,
							h.created_at DESC,
							h.id DESC
						LIMIT 1
					) AS historical_cost
						ON true

					WHERE
						o.paid_at >= $1
						AND o.paid_at < $2
						AND o.currency = $3
						AND o.payment_status IN (
							'paid',
							'cod_collected',
							'refunded'
						)
				),

				costed_returns AS (
					SELECT
						to_char(
							date_trunc(
								$4,
								r.inspected_at
									AT TIME ZONE 'Asia/Dhaka'
							),
							'YYYY-MM-DD'
						) AS bucket_start,

						ri.restock_quantity,

						CASE
							WHEN
								oi.unit_cost_amount IS NOT NULL
								AND oi.cost_currency = $3
							THEN oi.unit_cost_amount

							WHEN
								oi.unit_cost_amount IS NULL
							THEN historical_cost.unit_cost_amount

							ELSE NULL
						END AS effective_unit_cost_amount

					FROM returns r

					JOIN return_items ri
						ON ri.return_id = r.id

					JOIN order_items oi
						ON oi.id = ri.order_item_id

					JOIN orders o
						ON o.id = r.order_id

					LEFT JOIN LATERAL (
						SELECT
							h.unit_cost_amount
						FROM finance_variant_cost_history h
						WHERE
							oi.unit_cost_amount IS NULL
							AND h.variant_id = oi.variant_id
							AND h.currency = $3
							AND h.effective_at <= o.created_at
						ORDER BY
							h.effective_at DESC,
							h.created_at DESC,
							h.id DESC
						LIMIT 1
					) AS historical_cost
						ON true

					WHERE
						r.inspected_at >= $1
						AND r.inspected_at < $2
						AND o.currency = $3
				),

				aggregated AS (
					SELECT
						to_char(
							date_trunc(
								$4,
								o.paid_at
									AT TIME ZONE 'Asia/Dhaka'
							),
							'YYYY-MM-DD'
						) AS bucket_start,

						COUNT(*)::bigint
							AS collected_orders,

						COALESCE(
							SUM(o.subtotal_amount),
							0
						)::bigint
							AS gross_merchandise,

						COALESCE(
							SUM(o.discount_amount),
							0
						)::bigint
							AS discounts,

						COALESCE(
							SUM(o.shipping_amount),
							0
						)::bigint
							AS shipping,

						COALESCE(
							SUM(o.total_amount),
							0
						)::bigint
							AS gross_collected,

						0::bigint AS refunds,
						0::bigint AS units_sold,
						0::bigint AS costed_units,
						0::bigint AS missing_cost_units,
						0::bigint AS gross_cogs,
						0::bigint AS restocked_units,
						0::bigint AS restock_costed_units,
						0::bigint AS restock_missing_cost_units,
						0::bigint AS restock_recovery,
						0::bigint AS expenses

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
						bucket_start,
						0::bigint,
						0::bigint,
						0::bigint,
						0::bigint,
						0::bigint,
						0::bigint,

						COALESCE(
							SUM(quantity),
							0
						)::bigint,

						COALESCE(
							SUM(
								CASE
									WHEN effective_unit_cost_amount IS NOT NULL
									THEN quantity
									ELSE 0
								END
							),
							0
						)::bigint,

						COALESCE(
							SUM(
								CASE
									WHEN effective_unit_cost_amount IS NULL
									THEN quantity
									ELSE 0
								END
							),
							0
						)::bigint,

						COALESCE(
							SUM(
								CASE
									WHEN effective_unit_cost_amount IS NOT NULL
									THEN
										effective_unit_cost_amount::numeric *
										quantity::numeric
									ELSE 0::numeric
								END
							),
							0::numeric
						)::bigint,

						0::bigint,
						0::bigint,
						0::bigint,
						0::bigint,
						0::bigint

					FROM costed_sales

					GROUP BY bucket_start

					UNION ALL

					SELECT
						to_char(
							date_trunc(
								$4,
								r.succeeded_at
									AT TIME ZONE 'Asia/Dhaka'
							),
							'YYYY-MM-DD'
						),

						0::bigint,
						0::bigint,
						0::bigint,
						0::bigint,
						0::bigint,

						COALESCE(
							SUM(r.amount),
							0
						)::bigint,

						0::bigint,
						0::bigint,
						0::bigint,
						0::bigint,
						0::bigint,
						0::bigint,
						0::bigint,
						0::bigint,
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
						bucket_start,
						0::bigint,
						0::bigint,
						0::bigint,
						0::bigint,
						0::bigint,
						0::bigint,
						0::bigint,
						0::bigint,
						0::bigint,
						0::bigint,

						COALESCE(
							SUM(restock_quantity),
							0
						)::bigint,

						COALESCE(
							SUM(
								CASE
									WHEN effective_unit_cost_amount IS NOT NULL
									THEN restock_quantity
									ELSE 0
								END
							),
							0
						)::bigint,

						COALESCE(
							SUM(
								CASE
									WHEN effective_unit_cost_amount IS NULL
									THEN restock_quantity
									ELSE 0
								END
							),
							0
						)::bigint,

						COALESCE(
							SUM(
								CASE
									WHEN effective_unit_cost_amount IS NOT NULL
									THEN
										effective_unit_cost_amount::numeric *
										restock_quantity::numeric
									ELSE 0::numeric
								END
							),
							0::numeric
						)::bigint,

						0::bigint

					FROM costed_returns

					GROUP BY bucket_start

					UNION ALL

					SELECT
						to_char(
							date_trunc(
								$4,
								fe.occurred_at
									AT TIME ZONE 'Asia/Dhaka'
							),
							'YYYY-MM-DD'
						),

						0::bigint,
						0::bigint,
						0::bigint,
						0::bigint,
						0::bigint,
						0::bigint,
						0::bigint,
						0::bigint,
						0::bigint,
						0::bigint,
						0::bigint,
						0::bigint,
						0::bigint,
						0::bigint,

						COALESCE(
							SUM(fe.amount),
							0
						)::bigint

					FROM finance_expenses fe

					WHERE
						fe.status = 'active'
						AND fe.occurred_at >= $1
						AND fe.occurred_at < $2
						AND fe.currency = $3

					GROUP BY 1
				)

				SELECT
					bucket_start,

					SUM(collected_orders)::bigint,
					SUM(gross_merchandise)::bigint,
					SUM(discounts)::bigint,
					SUM(shipping)::bigint,
					SUM(gross_collected)::bigint,
					SUM(refunds)::bigint,

					SUM(units_sold)::bigint,
					SUM(costed_units)::bigint,
					SUM(missing_cost_units)::bigint,
					SUM(gross_cogs)::bigint,

					SUM(restocked_units)::bigint,
					SUM(restock_costed_units)::bigint,
					SUM(restock_missing_cost_units)::bigint,
					SUM(restock_recovery)::bigint,

					SUM(expenses)::bigint

				FROM aggregated

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
				"load profit/loss trend: %w",
				err,
			)
	}

	defer rows.Close()

	observed :=
		make(
			map[string]ProfitLossPoint,
		)

	for rows.Next() {
		var point ProfitLossPoint

		if err :=
			rows.Scan(
				&point.BucketStart,

				&point.CollectedOrders,
				&point.GrossMerchandiseRevenueAmount,
				&point.DiscountAmount,
				&point.ShippingRevenueAmount,
				&point.GrossCollectedRevenueAmount,
				&point.SuccessfulRefundAmount,

				&point.UnitsSold,
				&point.CostedUnits,
				&point.MissingCostUnits,
				&point.GrossCOGSAmount,

				&point.RestockedUnits,
				&point.RestockCostedUnits,
				&point.RestockMissingCostUnits,
				&point.RestockCOGSRecoveryAmount,

				&point.RecordedExpensesAmount,
			); err != nil {

			return nil,
				fmt.Errorf(
					"scan profit/loss trend: %w",
					err,
				)
		}

		finalizeProfitLossPoint(
			&point,
		)

		observed[point.BucketStart] = point
	}

	if err :=
		rows.Err(); err != nil {

		return nil,
			fmt.Errorf(
				"iterate profit/loss trend: %w",
				err,
			)
	}

	return fillProfitLossTrend(
			query,
			observed,
		),
		nil
}

func (s *Service) ProductProfitability(
	ctx context.Context,
	query Query,
	limit int,
) (
	[]ProductProfitability,
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

	queryCtx, cancel, err :=
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
						id,
						created_at,
						subtotal_amount,
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
				),

				costed_items AS (
					SELECT
						co.id AS order_id,
						co.subtotal_amount,
						co.discount_amount,

						oi.variant_id,
						oi.sku,
						oi.product_name,
						oi.quantity,
						oi.line_total_amount,

						CASE
							WHEN
								oi.unit_cost_amount IS NOT NULL
								AND oi.cost_currency = $3
							THEN oi.unit_cost_amount

							WHEN
								oi.unit_cost_amount IS NULL
							THEN historical_cost.unit_cost_amount

							ELSE NULL
						END AS effective_unit_cost_amount

					FROM collected_orders co

					JOIN order_items oi
						ON oi.order_id = co.id

					LEFT JOIN LATERAL (
						SELECT
							h.unit_cost_amount
						FROM finance_variant_cost_history h
						WHERE
							oi.unit_cost_amount IS NULL
							AND h.variant_id = oi.variant_id
							AND h.currency = $3
							AND h.effective_at <= co.created_at
						ORDER BY
							h.effective_at DESC,
							h.created_at DESC,
							h.id DESC
						LIMIT 1
					) AS historical_cost
						ON true
				),

				aggregated AS (
					SELECT
						ci.variant_id,
						pv.product_id,

						MAX(ci.sku)
							AS sku,

						MAX(ci.product_name)
							AS product_name,

						COUNT(
							DISTINCT ci.order_id
						)::bigint
							AS collected_orders,

						COALESCE(
							SUM(ci.quantity),
							0
						)::bigint
							AS units_sold,

						COALESCE(
							SUM(
								ci.line_total_amount
							),
							0
						)::bigint
							AS gross_merchandise,

						ROUND(
							COALESCE(
								SUM(
									CASE
										WHEN
											ci.subtotal_amount > 0
										THEN
											ci.line_total_amount::numeric *
											GREATEST(
												ci.subtotal_amount -
												ci.discount_amount,
												0
											)::numeric /
											ci.subtotal_amount::numeric
										ELSE
											0::numeric
									END
								),
								0::numeric
							)
						)::bigint
							AS net_merchandise,

						COALESCE(
							SUM(
								CASE
									WHEN ci.effective_unit_cost_amount IS NOT NULL
									THEN ci.quantity
									ELSE 0
								END
							),
							0
						)::bigint
							AS costed_units,

						COALESCE(
							SUM(
								CASE
									WHEN ci.effective_unit_cost_amount IS NULL
									THEN ci.quantity
									ELSE 0
								END
							),
							0
						)::bigint
							AS missing_cost_units,

						COALESCE(
							SUM(
								CASE
									WHEN ci.effective_unit_cost_amount IS NOT NULL
									THEN
										ci.effective_unit_cost_amount::numeric *
										ci.quantity::numeric
									ELSE 0::numeric
								END
							),
							0::numeric
						)::bigint
							AS known_cogs

					FROM costed_items ci

					JOIN product_variants pv
						ON pv.id = ci.variant_id

					GROUP BY
						ci.variant_id,
						pv.product_id
				)

				SELECT
					variant_id::text,
					product_id::text,
					sku,
					product_name,
					collected_orders,
					units_sold,
					gross_merchandise,
					net_merchandise,
					costed_units,
					missing_cost_units,
					known_cogs

				FROM aggregated

				ORDER BY
					CASE
						WHEN missing_cost_units = 0
						THEN
							net_merchandise -
							known_cogs
						ELSE NULL
					END DESC NULLS LAST,

					net_merchandise DESC,
					variant_id

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
				"load product profitability: %w",
				err,
			)
	}

	defer rows.Close()

	result :=
		make(
			[]ProductProfitability,
			0,
			limit,
		)

	for rows.Next() {
		var item ProductProfitability

		if err :=
			rows.Scan(
				&item.VariantID,
				&item.ProductID,
				&item.SKU,
				&item.ProductName,
				&item.CollectedOrders,
				&item.UnitsSold,
				&item.GrossMerchandiseRevenueAmount,
				&item.NetMerchandiseRevenueAmount,
				&item.CostedUnits,
				&item.MissingCostUnits,
				&item.KnownCOGSAmount,
			); err != nil {

			return nil,
				fmt.Errorf(
					"scan product profitability: %w",
					err,
				)
		}

		item.COGSCoverageBPS =
			basisPoints(
				item.CostedUnits,
				item.UnitsSold,
			)

		item.ProfitComplete =
			item.MissingCostUnits == 0

		if item.ProfitComplete {
			grossProfit :=
				item.NetMerchandiseRevenueAmount -
					item.KnownCOGSAmount

			item.GrossProfitBeforeRefundsAmount =
				int64Ptr(
					grossProfit,
				)

			if margin, ok :=
				signedBasisPoints(
					grossProfit,
					item.NetMerchandiseRevenueAmount,
				); ok {

				item.GrossMarginBeforeRefundsBPS =
					int64Ptr(
						margin,
					)
			}
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
				"iterate product profitability: %w",
				err,
			)
	}

	return result, nil
}

func finalizeProfitLoss(
	result *ProfitLoss,
	currency string,
) {
	result.NetMerchandiseRevenueAmount =
		result.GrossMerchandiseRevenueAmount -
			result.DiscountAmount

	result.NetCollectedRevenueAmount =
		result.GrossCollectedRevenueAmount -
			result.SuccessfulRefundAmount

	result.NonRestockedReturnUnits =
		result.ReturnReceivedUnits -
			result.RestockedUnits

	if result.NonRestockedReturnUnits < 0 {
		result.NonRestockedReturnUnits = 0
	}

	result.NetCOGSAmount =
		result.GrossCOGSAmount -
			result.RestockCOGSRecoveryAmount

	result.SalesCOGSCoverageBPS =
		basisPoints(
			result.CostedUnits,
			result.UnitsSold,
		)

	result.RestockCOGSCoverageBPS =
		basisPoints(
			result.RestockCostedUnits,
			result.RestockedUnits,
		)

	result.COGSCoverageBPS =
		basisPoints(
			result.CostedUnits+
				result.RestockCostedUnits,
			result.UnitsSold+
				result.RestockedUnits,
		)

	result.KnownGrossProfitAmount =
		result.NetCollectedRevenueAmount -
			result.NetCOGSAmount

	result.ProfitComplete =
		result.MissingCostUnits == 0 &&
			result.RestockMissingCostUnits == 0

	if result.ProfitComplete {
		result.GrossProfitAmount =
			int64Ptr(
				result.KnownGrossProfitAmount,
			)

		if margin, ok :=
			signedBasisPoints(
				result.KnownGrossProfitAmount,
				result.NetCollectedRevenueAmount,
			); ok {

			result.GrossMarginBPS =
				int64Ptr(
					margin,
				)
		}

		netProfit :=
			result.KnownGrossProfitAmount -
				result.RecordedExpensesAmount

		result.NetProfitAfterRecordedExpensesAmount =
			int64Ptr(
				netProfit,
			)
	}

	result.ExpenseBasis =
		"recorded_only"

	result.Currency =
		currency

	result.Warnings =
		make(
			[]string,
			0,
			3,
		)

	if !result.ProfitComplete {
		result.Warnings =
			append(
				result.Warnings,
				"Gross and net profit are incomplete because one or more cost-sensitive units lack both a usable order-item cost snapshot and an effective historical SKU buying cost in the reporting currency.",
			)
	}

	if result.ForeignCurrencyExpenseEntriesExcluded > 0 {
		result.Warnings =
			append(
				result.Warnings,
				"Active expense entries in other currencies are excluded; no automatic FX conversion is performed.",
			)
	}

	result.Warnings =
		append(
			result.Warnings,
			"Net profit after recorded expenses includes only active expenses recorded in the finance expense ledger; unrecorded real-world expenses cannot be inferred.",
		)
}

func finalizeProfitLossPoint(
	point *ProfitLossPoint,
) {
	point.NetMerchandiseRevenueAmount =
		point.GrossMerchandiseRevenueAmount -
			point.DiscountAmount

	point.NetCollectedRevenueAmount =
		point.GrossCollectedRevenueAmount -
			point.SuccessfulRefundAmount

	point.NetCOGSAmount =
		point.GrossCOGSAmount -
			point.RestockCOGSRecoveryAmount

	point.COGSCoverageBPS =
		basisPoints(
			point.CostedUnits+
				point.RestockCostedUnits,
			point.UnitsSold+
				point.RestockedUnits,
		)

	point.KnownGrossProfitAmount =
		point.NetCollectedRevenueAmount -
			point.NetCOGSAmount

	point.ProfitComplete =
		point.MissingCostUnits == 0 &&
			point.RestockMissingCostUnits == 0

	if !point.ProfitComplete {
		return
	}

	point.GrossProfitAmount =
		int64Ptr(
			point.KnownGrossProfitAmount,
		)

	if margin, ok :=
		signedBasisPoints(
			point.KnownGrossProfitAmount,
			point.NetCollectedRevenueAmount,
		); ok {

		point.GrossMarginBPS =
			int64Ptr(
				margin,
			)
	}

	netProfit :=
		point.KnownGrossProfitAmount -
			point.RecordedExpensesAmount

	point.NetProfitAfterRecordedExpensesAmount =
		int64Ptr(
			netProfit,
		)
}

func fillProfitLossTrend(
	query Query,
	observed map[string]ProfitLossPoint,
) []ProfitLossPoint {
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
			[]ProfitLossPoint,
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
				ProfitLossPoint{
					BucketStart: key,
				}

			finalizeProfitLossPoint(
				&point,
			)
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

func signedBasisPoints(
	numerator int64,
	denominator int64,
) (
	int64,
	bool,
) {
	if denominator <= 0 {
		return 0,
			false
	}

	scaled :=
		numerator *
			10000

	if scaled >= 0 {
		return (scaled +
				denominator/2) /
				denominator,
			true
	}

	return (scaled -
			denominator/2) /
			denominator,
		true
}

func int64Ptr(
	value int64,
) *int64 {
	return &value
}
