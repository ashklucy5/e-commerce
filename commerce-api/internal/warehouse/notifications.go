package warehouse

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"project.local/commerce-api/internal/notification"
)

type inboundNotificationOrderSnapshot struct {
	OrderID string

	CustomerID string

	OrderNumber string

	CustomerPhone string
	CustomerEmail string
}

func (r *Repository) enqueueInboundJourneyNotificationTx(
	ctx context.Context,
	tx pgx.Tx,
	inboundID string,
	status string,
	publicMessage string,
) error {
	stage, ok :=
		inboundNotificationStage(
			status,
		)
	if !ok {
		return nil
	}

	/*
		One China inbound shipment may contain fulfillments for
		multiple orders.

		Read every affected order first, close the pgx Rows, and
		only then insert outbox rows. pgx cannot execute another
		statement while an active Rows result still owns the
		connection.
	*/
	rows, err :=
		tx.Query(
			ctx,
			`
				SELECT DISTINCT
					o.id::text,

					COALESCE(
						o.customer_id::text,
						''
					),

					o.order_number,

					o.customer_phone,

					COALESCE(
						o.customer_email,
						''
					)

				FROM warehouse_fulfillments wf

				JOIN orders o
					ON o.id = wf.order_id

				WHERE
					wf.inbound_shipment_id =
						$1::uuid

				ORDER BY
					o.id::text
			`,
			inboundID,
		)
	if err != nil {
		return fmt.Errorf(
			"load inbound notification orders: %w",
			err,
		)
	}

	snapshots :=
		make(
			[]inboundNotificationOrderSnapshot,
			0,
		)

	for rows.Next() {
		var item inboundNotificationOrderSnapshot

		if err :=
			rows.Scan(
				&item.OrderID,
				&item.CustomerID,
				&item.OrderNumber,
				&item.CustomerPhone,
				&item.CustomerEmail,
			); err != nil {

			rows.Close()

			return fmt.Errorf(
				"scan inbound notification order: %w",
				err,
			)
		}

		snapshots =
			append(
				snapshots,
				item,
			)
	}

	if err :=
		rows.Err(); err != nil {

		rows.Close()

		return fmt.Errorf(
			"iterate inbound notification orders: %w",
			err,
		)
	}

	rows.Close()

	message :=
		strings.TrimSpace(
			publicMessage,
		)

	if message == "" {
		message =
			defaultInboundJourneyPublicMessage(
				status,
			)
	}

	for _, snapshot := range snapshots {

		if err :=
			notification.EnqueueCustomerEventTx(
				ctx,
				tx,
				notification.CustomerEventRequest{
					BaseDedupeKey: "delivery:" +
						snapshot.OrderID +
						":" +
						stage,

					Category: notification.CategoryDelivery,

					EventType: notification.EventDeliveryUpdated,

					CustomerID: snapshot.CustomerID,

					OrderID: snapshot.OrderID,

					Phone: snapshot.CustomerPhone,

					Email: snapshot.CustomerEmail,

					TemplateKey: notification.TemplateDeliveryUpdated,

					Message: message,

					Payload: map[string]any{
						"order_number": snapshot.OrderNumber,

						"stage": stage,

						"status": status,

						"inbound_shipment_id": inboundID,
					},
				},
			); err != nil {

			return fmt.Errorf(
				"enqueue inbound journey notification: %w",
				err,
			)
		}
	}

	return nil
}

func inboundNotificationStage(
	status string,
) (
	string,
	bool,
) {
	switch status {
	case InboundStatusPackedInChina:
		return "packed_in_china",
			true

	case InboundStatusDepartedChina,
		InboundStatusInInternationalTransit:

		/*
			Both statuses represent the same customer notification
			milestone.

			The shared dedupe key means transitioning:

			    departed_china
			        ↓
			    in_international_transit

			does not send two SMS/email messages.
		*/
		return "international_transit",
			true

	case InboundStatusArrivedBangladesh:
		return "arrived_bangladesh",
			true

	case InboundStatusReceivedAtWarehouse:
		return "warehouse_received",
			true

	default:
		return "",
			false
	}
}
