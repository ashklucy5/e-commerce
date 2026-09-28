package delivery

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"project.local/commerce-api/internal/notification"
)

type deliveryNotificationSnapshot struct {
	CustomerID string

	OrderNumber string

	CustomerPhone string
	CustomerEmail string
}

type deliveryOrderEventNotificationSpec struct {
	EventType string

	TemplateKey string

	DedupeSuffix string
}

func (r *Repository) enqueueDeliveryOrderEventNotificationTx(
	ctx context.Context,
	tx pgx.Tx,
	orderID string,
	orderEventType string,
) error {
	spec, ok :=
		deliveryNotificationSpec(
			orderEventType,
		)
	if !ok {
		return nil
	}

	snapshot, err :=
		r.deliveryNotificationSnapshotTx(
			ctx,
			tx,
			orderID,
		)
	if err != nil {
		return err
	}

	var message string

	switch orderEventType {
	case "order_shipped":
		message =
			fmt.Sprintf(
				"Order %s is out for delivery.",
				snapshot.OrderNumber,
			)

	case "order_delivered":
		message =
			fmt.Sprintf(
				"Order %s has been delivered.",
				snapshot.OrderNumber,
			)

	default:
		return nil
	}

	if err :=
		notification.EnqueueCustomerEventTx(
			ctx,
			tx,
			notification.CustomerEventRequest{
				BaseDedupeKey: "delivery:" +
					orderID +
					":" +
					spec.DedupeSuffix,

				Category: notification.CategoryDelivery,

				EventType: spec.EventType,

				CustomerID: snapshot.CustomerID,

				OrderID: orderID,

				Phone: snapshot.CustomerPhone,

				Email: snapshot.CustomerEmail,

				TemplateKey: spec.TemplateKey,

				Message: message,

				Payload: map[string]any{
					"order_number": snapshot.OrderNumber,

					"stage": spec.DedupeSuffix,
				},
			},
		); err != nil {

		return fmt.Errorf(
			"enqueue delivery notification: %w",
			err,
		)
	}

	return nil
}

func (r *Repository) deliveryNotificationSnapshotTx(
	ctx context.Context,
	tx pgx.Tx,
	orderID string,
) (
	deliveryNotificationSnapshot,
	error,
) {
	var result deliveryNotificationSnapshot

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
					)

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
		)
	if err != nil {
		return deliveryNotificationSnapshot{},
			fmt.Errorf(
				"load delivery notification snapshot: %w",
				err,
			)
	}

	return result,
		nil
}

func deliveryNotificationSpec(
	orderEventType string,
) (
	deliveryOrderEventNotificationSpec,
	bool,
) {
	switch orderEventType {
	case "order_shipped":
		return deliveryOrderEventNotificationSpec{
				EventType: notification.EventDeliverySent,

				TemplateKey: notification.TemplateDeliveryDispatched,

				DedupeSuffix: "dispatched",
			},
			true

	case "order_delivered":
		return deliveryOrderEventNotificationSpec{
				EventType: notification.EventDeliveryComplete,

				TemplateKey: notification.TemplateDeliveryDelivered,

				DedupeSuffix: "delivered",
			},
			true

	default:
		return deliveryOrderEventNotificationSpec{},
			false
	}
}
