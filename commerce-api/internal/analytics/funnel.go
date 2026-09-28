package analytics

import (
	"context"
	"fmt"
)

type Funnel struct {
	CartsCreated int64 `json:"carts_created"`

	CheckoutStartedCarts int64 `json:"checkout_started_carts"`
	OrderedCarts         int64 `json:"ordered_carts"`
	CollectedCarts       int64 `json:"collected_carts"`

	CartToCheckoutBPS   int64 `json:"cart_to_checkout_bps"`
	CheckoutToOrderBPS  int64 `json:"checkout_to_order_bps"`
	OrderToCollectedBPS int64 `json:"order_to_collected_bps"`
	CartToCollectedBPS  int64 `json:"cart_to_collected_bps"`
}

func (s *Service) Funnel(
	ctx context.Context,
	query Query,
) (
	Funnel,
	error,
) {
	queryCtx,
		cancel,
		err :=
		s.queryContext(
			ctx,
		)
	if err != nil {
		return Funnel{}, err
	}
	defer cancel()

	var result Funnel

	err =
		s.db.QueryRow(
			queryCtx,
			`
				WITH cohort AS (
					SELECT c.id
					FROM carts c
					WHERE
						c.created_at >= $1
						AND c.created_at < $2
						AND c.currency = $3
				),
				stages AS (
					SELECT
						c.id,
						EXISTS (
							SELECT 1
							FROM checkout_sessions cs
							WHERE cs.cart_id = c.id
						) AS started_checkout,
						EXISTS (
							SELECT 1
							FROM checkout_sessions cs
							JOIN orders o
								ON o.checkout_id = cs.id
							WHERE cs.cart_id = c.id
						) AS placed_order,
						EXISTS (
							SELECT 1
							FROM checkout_sessions cs
							JOIN orders o
								ON o.checkout_id = cs.id
							WHERE
								cs.cart_id = c.id
								AND o.payment_status IN (
									'paid',
									'cod_collected',
									'refunded'
								)
						) AS collected
					FROM cohort c
				)
				SELECT
					COUNT(*)::bigint,
					COUNT(*) FILTER (
						WHERE started_checkout
					)::bigint,
					COUNT(*) FILTER (
						WHERE placed_order
					)::bigint,
					COUNT(*) FILTER (
						WHERE collected
					)::bigint
				FROM stages
			`,
			query.Start,
			query.End,
			query.Currency,
		).Scan(
			&result.CartsCreated,
			&result.CheckoutStartedCarts,
			&result.OrderedCarts,
			&result.CollectedCarts,
		)
	if err != nil {
		return Funnel{},
			fmt.Errorf(
				"load checkout funnel analytics: %w",
				err,
			)
	}

	result.CartToCheckoutBPS =
		basisPoints(
			result.CheckoutStartedCarts,
			result.CartsCreated,
		)

	result.CheckoutToOrderBPS =
		basisPoints(
			result.OrderedCarts,
			result.CheckoutStartedCarts,
		)

	result.OrderToCollectedBPS =
		basisPoints(
			result.CollectedCarts,
			result.OrderedCarts,
		)

	result.CartToCollectedBPS =
		basisPoints(
			result.CollectedCarts,
			result.CartsCreated,
		)

	return result, nil
}
