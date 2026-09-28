package order

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

func (r *Repository) prepareNewOrderCommercialSnapshotTx(
	ctx context.Context,
	tx pgx.Tx,
	orderID string,
) error {
	var invoiceExists bool

	err :=
		tx.QueryRow(
			ctx,
			`
				SELECT EXISTS (
					SELECT 1
					FROM invoices i
					WHERE i.order_id = o.id
				)
				FROM orders o
				WHERE o.id = $1::uuid
				FOR UPDATE
			`,
			orderID,
		).Scan(
			&invoiceExists,
		)

	if errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return fmt.Errorf(
			"prepare commercial snapshot: order %s not found",
			orderID,
		)
	}

	if err != nil {
		return fmt.Errorf(
			"lock order for commercial snapshot: %w",
			err,
		)
	}

	// The persisted invoice is the idempotency boundary. Once an invoice
	// exists, replaying order_placed must never re-snapshot a later product
	// cost into the historical order.
	if invoiceExists {
		return nil
	}

	if err :=
		r.snapshotOrderItemCostsTx(
			ctx,
			tx,
			orderID,
		); err != nil {
		return err
	}

	if err :=
		r.issueInvoiceSnapshotTx(
			ctx,
			tx,
			orderID,
		); err != nil {
		return err
	}

	return nil
}

func (r *Repository) snapshotOrderItemCostsTx(
	ctx context.Context,
	tx pgx.Tx,
	orderID string,
) error {
	var itemCount int64

	if err :=
		tx.QueryRow(
			ctx,
			`
				SELECT COUNT(*)::bigint
				FROM order_items
				WHERE order_id = $1::uuid
			`,
			orderID,
		).Scan(
			&itemCount,
		); err != nil {
		return fmt.Errorf(
			"count order items for cost snapshot: %w",
			err,
		)
	}

	if itemCount < 1 {
		return fmt.Errorf(
			"snapshot order item costs: order %s has no items",
			orderID,
		)
	}

	tag, err :=
		tx.Exec(
			ctx,
			`
				UPDATE order_items oi
				SET
					unit_cost_amount = v.cost_amount,

					line_cost_amount = CASE
						WHEN v.cost_amount IS NULL THEN NULL
						ELSE (
							v.cost_amount::numeric *
							oi.quantity::numeric
						)::bigint
					END,

					cost_currency = CASE
						WHEN v.cost_amount IS NULL THEN NULL
						ELSE v.currency
					END

				FROM product_variants v

				WHERE
					oi.order_id = $1::uuid
					AND v.id = oi.variant_id
			`,
			orderID,
		)
	if err != nil {
		return fmt.Errorf(
			"snapshot order item costs: %w",
			err,
		)
	}

	if tag.RowsAffected() != itemCount {
		return fmt.Errorf(
			"snapshot order item costs: expected %d rows, updated %d",
			itemCount,
			tag.RowsAffected(),
		)
	}

	return nil
}
