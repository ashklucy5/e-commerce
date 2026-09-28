package order

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"project.local/commerce-api/internal/notification"
)

type orderNotificationSnapshot struct {
	CustomerID    string
	OrderNumber   string
	CustomerPhone string
	CustomerEmail string
	PaymentMethod string
}

func (r *Repository) enqueueOrderEventNotificationTx(
	ctx context.Context,
	tx pgx.Tx,
	orderID string,
	input eventInsert,
) error {
	var (
		category    string
		eventType   string
		templateKey string
		baseKey     string
	)

	switch input.EventType {
	case EventOrderPlaced:
		category =
			notification.CategoryOrder

		eventType =
			notification.EventOrderPlaced

		templateKey =
			notification.TemplateOrderPlaced

		baseKey =
			"order:" +
				orderID +
				":placed"

	case EventPaymentConfirmed:
		category =
			notification.CategoryPayment

		eventType =
			notification.EventPaymentSucceeded

		templateKey =
			notification.TemplatePaymentSucceeded

		baseKey =
			"order:" +
				orderID +
				":payment_succeeded"

	case EventOrderProcessing:
		category =
			notification.CategoryOrder

		eventType =
			notification.EventOrderProcessing

		templateKey =
			notification.TemplateOrderProcessing

		baseKey =
			"order:" +
				orderID +
				":processing"

	default:
		return nil
	}

	snapshot, err :=
		r.orderNotificationSnapshotTx(
			ctx,
			tx,
			orderID,
		)

	if err != nil {
		return err
	}

	var message string

	switch input.EventType {
	case EventOrderPlaced:
		if snapshot.PaymentMethod ==
			PaymentMethodCOD {

			message =
				fmt.Sprintf(
					"We received order %s. Your order is confirmed.",
					snapshot.OrderNumber,
				)
		} else {
			message =
				fmt.Sprintf(
					"We received order %s. Payment is pending.",
					snapshot.OrderNumber,
				)
		}

	case EventPaymentConfirmed:
		message =
			fmt.Sprintf(
				"Payment received for order %s.",
				snapshot.OrderNumber,
			)

	case EventOrderProcessing:
		message =
			fmt.Sprintf(
				"Order %s is being prepared.",
				snapshot.OrderNumber,
			)
	}

	if err :=
		notification.EnqueueCustomerEventTx(
			ctx,
			tx,
			notification.CustomerEventRequest{
				BaseDedupeKey: baseKey,
				Category:      category,
				EventType:     eventType,
				CustomerID:    snapshot.CustomerID,
				OrderID:       orderID,
				Phone:         snapshot.CustomerPhone,
				Email:         snapshot.CustomerEmail,
				TemplateKey:   templateKey,
				Message:       message,

				Payload: map[string]any{
					"order_number": snapshot.OrderNumber,
				},
			},
		); err != nil {

		return fmt.Errorf(
			"enqueue order notification: %w",
			err,
		)
	}

	if input.EventType ==
		EventOrderPlaced {

		if err :=
			notification.EnqueueStaffEventTx(
				ctx,
				tx,
				notification.StaffEventRequest{
					DedupeKey: "staff:order:" +
						orderID +
						":placed",

					Category: notification.StaffCategoryOrder,

					EventType: notification.StaffEventOrderPlaced,

					Priority: notification.StaffPriorityInfo,

					Title: "New order",

					Message: fmt.Sprintf(
						"Order %s has been placed.",
						snapshot.OrderNumber,
					),

					ActionURL: "/admin/orders",

					EntityType: "order",

					EntityID: orderID,

					RequiredPermission: "admin.order.read",

					Metadata: map[string]any{
						"order_number": snapshot.OrderNumber,

						"payment_method": snapshot.PaymentMethod,
					},
				},
			); err != nil {

			return fmt.Errorf(
				"enqueue staff order notification: %w",
				err,
			)
		}
	}

	return nil
}

func (r *Repository) orderNotificationSnapshotTx(
	ctx context.Context,
	tx pgx.Tx,
	orderID string,
) (
	orderNotificationSnapshot,
	error,
) {
	var result orderNotificationSnapshot

	err :=
		tx.QueryRow(
			ctx,
			`
				SELECT
					COALESCE(
						customer_id::text,
						''
					),
					order_number,
					customer_phone,
					COALESCE(
						customer_email,
						''
					),
					payment_method
				FROM orders
				WHERE
					id = $1::uuid
			`,
			orderID,
		).Scan(
			&result.CustomerID,
			&result.OrderNumber,
			&result.CustomerPhone,
			&result.CustomerEmail,
			&result.PaymentMethod,
		)

	if err != nil {
		return orderNotificationSnapshot{},
			fmt.Errorf(
				"load order notification snapshot: %w",
				err,
			)
	}

	return result,
		nil
}