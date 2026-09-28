package admin

import (
	"context"
	"fmt"
	"time"
)

type DashboardSummary struct {
	GeneratedAt time.Time `json:"generated_at"`

	ActiveProducts int64 `json:"active_products"`

	ActiveVariants int64 `json:"active_variants"`

	AvailableUnits int64 `json:"available_units"`

	ActiveCustomers int64 `json:"active_customers"`

	OrdersToday int64 `json:"orders_today"`

	RevenueTodayAmount int64 `json:"revenue_today_amount"`

	OpenCRMCases int64 `json:"open_crm_cases"`

	AwaitingConfirmation int64 `json:"awaiting_confirmation_shipments"`

	PendingFulfillments int64 `json:"pending_fulfillments"`

	PublishedReviews int64 `json:"published_reviews"`
}

func (s *Service) Dashboard(
	ctx context.Context,
) (
	DashboardSummary,
	error,
) {
	var result DashboardSummary

	err :=
		s.db.QueryRow(
			ctx,
			`
				SELECT
					(
						SELECT COUNT(*)
						FROM products
						WHERE status = 'active'
					),
					(
						SELECT COUNT(*)
						FROM product_variants pv
						JOIN products p
							ON p.id = pv.product_id
						WHERE
							pv.is_active = true
							AND p.status = 'active'
					),
					(
						SELECT COALESCE(
							SUM(
								quantity_on_hand -
								quantity_reserved
							),
							0
						)
						FROM inventory
					),
					(
						SELECT COUNT(*)
						FROM customers
						WHERE status = 'active'
					),
					(
						SELECT COUNT(*)
						FROM orders
						WHERE created_at >=
							date_trunc(
								'day',
								now()
							)
					),
					(
						SELECT COALESCE(
							SUM(total_amount),
							0
						)
						FROM orders
						WHERE
							paid_at >=
								date_trunc(
									'day',
									now()
								)
							AND payment_status IN (
								'paid',
								'cod_collected'
							)
					),
					(
						SELECT COUNT(*)
						FROM crm_cases
						WHERE status IN (
							'waiting_support',
							'waiting_customer'
						)
					),
					(
						SELECT COUNT(*)
						FROM shipments
						WHERE status =
							'awaiting_confirmation'
					),
					(
						SELECT COUNT(*)
						FROM warehouse_fulfillments
						WHERE status NOT IN (
							'handed_off',
							'cancelled'
						)
					),
					(
						SELECT COUNT(*)
						FROM reviews
						WHERE
							status = 'published'
							AND deleted_at IS NULL
					),
					now()
			`,
		).Scan(
			&result.ActiveProducts,
			&result.ActiveVariants,
			&result.AvailableUnits,
			&result.ActiveCustomers,
			&result.OrdersToday,
			&result.RevenueTodayAmount,
			&result.OpenCRMCases,
			&result.AwaitingConfirmation,
			&result.PendingFulfillments,
			&result.PublishedReviews,
			&result.GeneratedAt,
		)
	if err != nil {
		return DashboardSummary{},
			fmt.Errorf(
				"load Admin dashboard: %w",
				err,
			)
	}

	return result,
		nil
}
