package order

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"project.local/commerce-api/internal/notification"
)

type invoiceNotificationSnapshot struct {
	InvoiceID  string
	OrderID    string
	CustomerID string

	InvoiceNumber string
	OrderNumber   string

	CustomerEmail string

	Currency    string
	TotalAmount int64
}

type InvoiceNotificationEnqueueResult struct {
	OutboxID string `json:"outbox_id"`
	Inserted bool   `json:"inserted"`
}

type InvoiceResendResult struct {
	RequestKey string `json:"request_key"`

	Customer InvoiceNotificationEnqueueResult `json:"customer"`
	Business InvoiceNotificationEnqueueResult `json:"business"`
}

func (r *Repository) enqueueInvoiceIssuedNotificationsTx(
	ctx context.Context,
	tx pgx.Tx,
	orderID string,
	config invoiceRuntimeConfig,
) error {
	snapshot, err :=
		r.invoiceNotificationSnapshotTx(
			ctx,
			tx,
			orderID,
		)
	if err != nil {
		return err
	}

	_, _, err =
		enqueueInvoiceEmailPairTx(
			ctx,
			tx,
			snapshot,
			config,
			"invoice:"+
				snapshot.InvoiceID+
				":issued",
			notification.EventInvoiceIssued,
			notification.TemplateInvoiceIssuedCustomer,
			notification.TemplateInvoiceIssuedBusiness,
		)
	if err != nil {
		return fmt.Errorf(
			"enqueue invoice issued notifications: %w",
			err,
		)
	}

	return nil
}

func (r *Repository) ResendInvoiceNotifications(
	ctx context.Context,
	orderID string,
	idempotencyKey string,
) (
	InvoiceResendResult,
	error,
) {
	orderID =
		strings.TrimSpace(
			orderID,
		)

	if !uuidPattern.MatchString(
		orderID,
	) {
		return InvoiceResendResult{},
			ErrInvalidOrderID
	}

	requestKey, err :=
		normalizeInvoiceResendKey(
			idempotencyKey,
		)
	if err != nil {
		return InvoiceResendResult{},
			err
	}

	config, err :=
		loadInvoiceRuntimeConfig()
	if err != nil {
		return InvoiceResendResult{},
			fmt.Errorf(
				"load invoice runtime configuration: %w",
				err,
			)
	}

	tx, err :=
		r.db.Begin(
			ctx,
		)
	if err != nil {
		return InvoiceResendResult{},
			fmt.Errorf(
				"begin invoice resend transaction: %w",
				err,
			)
	}

	defer func() {
		_ = tx.Rollback(
			ctx,
		)
	}()

	snapshot, err :=
		r.invoiceNotificationSnapshotTx(
			ctx,
			tx,
			orderID,
		)
	if err != nil {
		return InvoiceResendResult{},
			err
	}

	customer,
		business,
		err :=
		enqueueInvoiceEmailPairTx(
			ctx,
			tx,
			snapshot,
			config,
			"invoice:"+
				snapshot.InvoiceID+
				":resend:"+
				requestKey,
			notification.EventInvoiceResent,
			notification.TemplateInvoiceResentCustomer,
			notification.TemplateInvoiceResentBusiness,
		)
	if err != nil {
		return InvoiceResendResult{},
			fmt.Errorf(
				"enqueue invoice resend notifications: %w",
				err,
			)
	}

	if err :=
		tx.Commit(
			ctx,
		); err != nil {

		return InvoiceResendResult{},
			fmt.Errorf(
				"commit invoice resend transaction: %w",
				err,
			)
	}

	return InvoiceResendResult{
		RequestKey: requestKey,

		Customer: customer,

		Business: business,
	}, nil
}

func (r *Repository) invoiceNotificationSnapshotTx(
	ctx context.Context,
	tx pgx.Tx,
	orderID string,
) (
	invoiceNotificationSnapshot,
	error,
) {
	var result invoiceNotificationSnapshot

	err :=
		tx.QueryRow(
			ctx,
			`
				SELECT
					i.id::text,
					i.order_id::text,
					COALESCE(
						i.customer_id::text,
						''
					),
					i.invoice_number,
					o.order_number,
					COALESCE(
						i.customer_email,
						''
					),
					i.currency,
					i.total_amount

				FROM invoices i

				JOIN orders o
					ON o.id = i.order_id

				WHERE
					i.order_id = $1::uuid
			`,
			orderID,
		).Scan(
			&result.InvoiceID,
			&result.OrderID,
			&result.CustomerID,
			&result.InvoiceNumber,
			&result.OrderNumber,
			&result.CustomerEmail,
			&result.Currency,
			&result.TotalAmount,
		)

	if errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return invoiceNotificationSnapshot{},
			ErrInvoiceNotFound
	}

	if err != nil {
		return invoiceNotificationSnapshot{},
			fmt.Errorf(
				"load invoice notification snapshot: %w",
				err,
			)
	}

	return result,
		nil
}

func enqueueInvoiceEmailPairTx(
	ctx context.Context,
	tx pgx.Tx,
	snapshot invoiceNotificationSnapshot,
	config invoiceRuntimeConfig,
	baseDedupeKey string,
	eventType string,
	customerTemplate string,
	businessTemplate string,
) (
	InvoiceNotificationEnqueueResult,
	InvoiceNotificationEnqueueResult,
	error,
) {
	payload :=
		map[string]any{
			"invoice_id": snapshot.InvoiceID,

			"invoice_number": snapshot.InvoiceNumber,

			"order_number": snapshot.OrderNumber,

			"currency": snapshot.Currency,

			"total_amount": snapshot.TotalAmount,
		}

	customerItem,
		customerInserted,
		err :=
		notification.EnqueueTx(
			ctx,
			tx,
			notification.EnqueueRequest{
				DedupeKey: notification.ChannelDedupeKey(
					baseDedupeKey+
						":customer",
					notification.ChannelEmail,
				),

				Category: notification.CategoryOrder,

				EventType: eventType,

				Channel: notification.ChannelEmail,

				CustomerID: snapshot.CustomerID,

				OrderID: snapshot.OrderID,

				Recipient: snapshot.CustomerEmail,

				TemplateKey: customerTemplate,

				Payload: payload,
			},
		)
	if err != nil {
		return InvoiceNotificationEnqueueResult{},
			InvoiceNotificationEnqueueResult{},
			err
	}

	businessItem,
		businessInserted,
		err :=
		notification.EnqueueTx(
			ctx,
			tx,
			notification.EnqueueRequest{
				DedupeKey: notification.ChannelDedupeKey(
					baseDedupeKey+
						":business",
					notification.ChannelEmail,
				),

				Category: notification.CategoryOrder,

				EventType: eventType,

				Channel: notification.ChannelEmail,

				OrderID: snapshot.OrderID,

				Recipient: config.BusinessEmail,

				TemplateKey: businessTemplate,

				Payload: payload,
			},
		)
	if err != nil {
		return InvoiceNotificationEnqueueResult{},
			InvoiceNotificationEnqueueResult{},
			err
	}

	return InvoiceNotificationEnqueueResult{
			OutboxID: customerItem.ID,

			Inserted: customerInserted,
		},
		InvoiceNotificationEnqueueResult{
			OutboxID: businessItem.ID,

			Inserted: businessInserted,
		},
		nil
}

func normalizeInvoiceResendKey(
	value string,
) (
	string,
	error,
) {
	value =
		strings.TrimSpace(
			value,
		)

	if value == "" ||
		len(
			value,
		) > 200 {

		return "",
			ErrInvalidInvoiceResendKey
	}

	/*
		Do not persist the caller's raw idempotency key in the
		notification dedupe key.

		The deterministic SHA-256 digest still gives retry
		idempotency without exposing the original value.
	*/
	digest :=
		sha256.Sum256(
			[]byte(
				value,
			),
		)

	return hex.EncodeToString(
		digest[:],
	), nil
}
