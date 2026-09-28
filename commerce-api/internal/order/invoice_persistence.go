package order

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
)

func (r *Repository) issueInvoiceSnapshotTx(
	ctx context.Context,
	tx pgx.Tx,
	orderID string,
) error {
	config, err :=
		loadInvoiceRuntimeConfig()
	if err != nil {
		return fmt.Errorf(
			"load invoice runtime configuration: %w",
			err,
		)
	}

	merchantPayload, err :=
		json.Marshal(
			config.Merchant,
		)
	if err != nil {
		return fmt.Errorf(
			"encode invoice merchant snapshot: %w",
			err,
		)
	}

	var invoiceID string

	if err :=
		tx.QueryRow(
			ctx,
			`
				INSERT INTO invoices (
					order_id,
					customer_id,
					invoice_number,
					issued_at,
					currency,
					subtotal_amount,
					discount_amount,
					shipping_amount,
					total_amount,
					customer_name,
					customer_phone,
					customer_email,
					shipping_address_line1,
					shipping_address_line2,
					shipping_city,
					shipping_area,
					shipping_postal_code,
					delivery_method,
					payment_method,
					payment_status_at_issue,
					merchant_snapshot,
					created_at
				)

				SELECT
					o.id,
					o.customer_id,
					'INV-' || o.order_number,
					o.created_at,
					o.currency,
					o.subtotal_amount,
					o.discount_amount,
					o.shipping_amount,
					o.total_amount,
					o.customer_name,
					o.customer_phone,
					o.customer_email,
					o.shipping_address_line1,
					o.shipping_address_line2,
					o.shipping_city,
					o.shipping_area,
					o.shipping_postal_code,
					o.delivery_method,
					o.payment_method,
					o.payment_status,
					$2::jsonb,
					now()

				FROM orders o

				WHERE o.id = $1::uuid

				RETURNING id::text
			`,
			orderID,
			string(
				merchantPayload,
			),
		).Scan(
			&invoiceID,
		); err != nil {

		return fmt.Errorf(
			"issue invoice snapshot: %w",
			err,
		)
	}

	var expectedItems int64

	if err :=
		tx.QueryRow(
			ctx,
			`
				SELECT COUNT(*)::bigint

				FROM order_items

				WHERE
					order_id = $1::uuid
			`,
			orderID,
		).Scan(
			&expectedItems,
		); err != nil {

		return fmt.Errorf(
			"count invoice source items: %w",
			err,
		)
	}

	if expectedItems < 1 {
		return fmt.Errorf(
			"issue invoice snapshot: order %s has no items",
			orderID,
		)
	}

	tag, err :=
		tx.Exec(
			ctx,
			`
				INSERT INTO invoice_items (
					invoice_id,
					order_item_id,
					variant_id,
					sku,
					product_name,
					quantity,
					unit_price_amount,
					line_total_amount,
					currency,
					created_at
				)

				SELECT
					$1::uuid,
					oi.id,
					oi.variant_id,
					oi.sku,
					oi.product_name,
					oi.quantity,
					oi.unit_price_amount,
					oi.line_total_amount,
					oi.currency,
					now()

				FROM order_items oi

				WHERE
					oi.order_id = $2::uuid

				ORDER BY
					oi.created_at,
					oi.id
			`,
			invoiceID,
			orderID,
		)
	if err != nil {
		return fmt.Errorf(
			"issue invoice item snapshots: %w",
			err,
		)
	}

	if tag.RowsAffected() != expectedItems {
		return fmt.Errorf(
			"issue invoice item snapshots: expected %d rows, inserted %d",
			expectedItems,
			tag.RowsAffected(),
		)
	}

	/*
		Invoice creation and invoice-issued notifications share the
		same PostgreSQL transaction.

		If either customer/business outbox insertion fails, the
		invoice and the surrounding order placement roll back.
	*/
	if err :=
		r.enqueueInvoiceIssuedNotificationsTx(
			ctx,
			tx,
			orderID,
			config,
		); err != nil {

		return err
	}

	return nil
}
