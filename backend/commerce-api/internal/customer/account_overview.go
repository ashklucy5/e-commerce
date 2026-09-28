package customer

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

const accountOverviewRecentOrderLimit = 3

type AccountOverview struct {
	Profile Customer `json:"profile"`

	Summary AccountOverviewSummary `json:"summary"`

	RecentOrders []AccountOverviewOrder `json:"recent_orders"`
}

type AccountOverviewSummary struct {
	TotalOrders     int64 `json:"total_orders"`
	ActiveOrders    int64 `json:"active_orders"`
	FulfilledOrders int64 `json:"fulfilled_orders"`

	WishlistItems int64 `json:"wishlist_items"`

	Addresses int64 `json:"addresses"`

	SavedSizes int64 `json:"saved_sizes"`

	OpenSupportCases int64 `json:"open_support_cases"`

	ActiveReturns int64 `json:"active_returns"`

	ActiveSessions int64 `json:"active_sessions"`
}

type AccountOverviewOrder struct {
	ID string `json:"id"`

	OrderNumber string `json:"order_number"`

	Status string `json:"status"`

	PaymentStatus string `json:"payment_status"`

	PaymentMethod string `json:"payment_method"`

	Currency string `json:"currency"`

	TotalAmount int64 `json:"total_amount"`

	DeliveryMethod string `json:"delivery_method"`

	ItemCount int64 `json:"item_count"`

	QuantityTotal int64 `json:"quantity_total"`

	FirstProductName string `json:"first_product_name,omitempty"`

	CreatedAt time.Time `json:"created_at"`

	UpdatedAt time.Time `json:"updated_at"`
}

func (s *Service) GetAccountOverview(
	ctx context.Context,
	customerID string,
) (AccountOverview, error) {
	profile, err :=
		s.GetProfile(
			ctx,
			customerID,
		)
	if err != nil {
		return AccountOverview{},
			err
	}

	summary, recentOrders, err :=
		s.repository.GetAccountOverviewData(
			ctx,
			customerID,
		)
	if err != nil {
		return AccountOverview{},
			err
	}

	if recentOrders == nil {
		recentOrders =
			make(
				[]AccountOverviewOrder,
				0,
			)
	}

	return AccountOverview{
			Profile: profile,

			Summary: summary,

			RecentOrders: recentOrders,
		},
		nil
}

func (r *Repository) GetAccountOverviewData(
	ctx context.Context,
	customerID string,
) (
	AccountOverviewSummary,
	[]AccountOverviewOrder,
	error,
) {
	batch :=
		&pgx.Batch{}

	batch.Queue(
		`
			SELECT
				(
					SELECT count(*)
					FROM orders
					WHERE customer_id = $1::uuid
				)::bigint,

				(
					SELECT count(*)
					FROM orders
					WHERE
						customer_id = $1::uuid
						AND status IN (
'pending_payment',
'awaiting_procurement',
'confirmed',
'processing',
'shipped'
)
				)::bigint,

				(
					SELECT count(*)
					FROM orders
					WHERE
						customer_id = $1::uuid
						AND status IN (
							'delivered',
							'completed'
						)
				)::bigint,

				(
					SELECT count(*)
					FROM customer_wishlist_items
					WHERE customer_id = $1::uuid
				)::bigint,

				(
					SELECT count(*)
					FROM customer_addresses
					WHERE customer_id = $1::uuid
				)::bigint,

				(
					SELECT count(*)
					FROM customer_saved_sizes
					WHERE customer_id = $1::uuid
				)::bigint,

				(
					SELECT count(*)
					FROM crm_cases
					WHERE
						customer_id = $1::uuid
						AND status IN (
							'waiting_support',
							'waiting_customer'
						)
				)::bigint,

				(
					SELECT count(*)
					FROM returns r
					INNER JOIN orders o
						ON o.id = r.order_id
					WHERE
						o.customer_id = $1::uuid
						AND r.status IN (
							'requested',
							'approved',
							'received',
							'inspected'
						)
				)::bigint,

				(
					SELECT count(*)
					FROM auth_sessions
					WHERE
						customer_id = $1::uuid
						AND revoked_at IS NULL
						AND refresh_expires_at > now()
				)::bigint
		`,
		customerID,
	)

	batch.Queue(
		`
			SELECT
				o.id::text,
				o.order_number,
				o.status,
				o.payment_status,
				o.payment_method,
				o.currency,
				o.total_amount,
				o.delivery_method,

				(
					SELECT count(*)
					FROM order_items oi
					WHERE oi.order_id = o.id
				)::bigint AS item_count,

				COALESCE(
					(
						SELECT sum(oi.quantity)
						FROM order_items oi
						WHERE oi.order_id = o.id
					),
					0
				)::bigint AS quantity_total,

				COALESCE(
					(
						SELECT oi.product_name
						FROM order_items oi
						WHERE oi.order_id = o.id
						ORDER BY
							oi.created_at ASC,
							oi.id ASC
						LIMIT 1
					),
					''
				) AS first_product_name,

				o.created_at,
				o.updated_at

			FROM orders o

			WHERE o.customer_id = $1::uuid

			ORDER BY
				o.created_at DESC,
				o.id DESC

			LIMIT 3
		`,
		customerID,
	)

	results :=
		r.db.SendBatch(
			ctx,
			batch,
		)

	defer func() {
		_ = results.Close()
	}()

	var summary AccountOverviewSummary

	if err :=
		results.QueryRow().Scan(
			&summary.TotalOrders,
			&summary.ActiveOrders,
			&summary.FulfilledOrders,
			&summary.WishlistItems,
			&summary.Addresses,
			&summary.SavedSizes,
			&summary.OpenSupportCases,
			&summary.ActiveReturns,
			&summary.ActiveSessions,
		); err != nil {

		return AccountOverviewSummary{},
			nil,
			fmt.Errorf(
				"get account overview summary: %w",
				err,
			)
	}

	rows, err :=
		results.Query()
	if err != nil {
		return AccountOverviewSummary{},
			nil,
			fmt.Errorf(
				"query recent account orders: %w",
				err,
			)
	}

	recentOrders :=
		make(
			[]AccountOverviewOrder,
			0,
			accountOverviewRecentOrderLimit,
		)

	for rows.Next() {
		var order AccountOverviewOrder

		if err :=
			rows.Scan(
				&order.ID,
				&order.OrderNumber,
				&order.Status,
				&order.PaymentStatus,
				&order.PaymentMethod,
				&order.Currency,
				&order.TotalAmount,
				&order.DeliveryMethod,
				&order.ItemCount,
				&order.QuantityTotal,
				&order.FirstProductName,
				&order.CreatedAt,
				&order.UpdatedAt,
			); err != nil {

			rows.Close()

			return AccountOverviewSummary{},
				nil,
				fmt.Errorf(
					"scan recent account order: %w",
					err,
				)
		}

		recentOrders =
			append(
				recentOrders,
				order,
			)
	}

	if err :=
		rows.Err(); err != nil {

		rows.Close()

		return AccountOverviewSummary{},
			nil,
			fmt.Errorf(
				"iterate recent account orders: %w",
				err,
			)
	}

	rows.Close()

	return summary,
		recentOrders,
		nil
}
