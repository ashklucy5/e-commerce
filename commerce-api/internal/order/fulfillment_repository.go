package order

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type fulfillmentQueryer interface {
	QueryRow(
		ctx context.Context,
		sql string,
		args ...any,
	) pgx.Row
}

func attachFulfillmentState(
	ctx context.Context,
	queryer fulfillmentQueryer,
	order *Order,
) error {
	const query = `
		SELECT
			o.processing_at,
			o.shipped_at,
			o.delivered_at,
			o.completed_at,

			COALESCE(s.id::text, ''),
			COALESCE(s.order_id::text, ''),
			COALESCE(s.origin_warehouse_id::text, ''),
			COALESCE(s.delivery_mode, ''),
			COALESCE(s.provider_code, ''),
			COALESCE(s.provider_shipment_id, ''),
			COALESCE(s.provider_status, ''),
			COALESCE(s.courier_name, ''),
			COALESCE(s.courier_reference, ''),
			COALESCE(s.rider_reference, ''),
			COALESCE(s.tracking_number, ''),
			COALESCE(s.tracking_url, ''),
			COALESCE(s.status, ''),
			s.shipped_at,
			s.provider_delivered_at,
			s.awaiting_confirmation_at,
			s.confirmed_received_at,
			COALESCE(s.confirmation_source, ''),
			COALESCE(s.confirmed_by_actor_id, ''),
			COALESCE(s.confirmation_note, ''),
			s.last_provider_sync_at,
			s.delivered_at,
			s.created_at,
			s.updated_at

		FROM orders o

		LEFT JOIN shipments s
			ON s.order_id = o.id

		WHERE o.id = $1::uuid
	`

	var processingAt pgtype.Timestamptz
	var shippedAt pgtype.Timestamptz
	var deliveredAt pgtype.Timestamptz
	var completedAt pgtype.Timestamptz

	var shipment Shipment

	var shipmentShippedAt pgtype.Timestamptz
	var providerDeliveredAt pgtype.Timestamptz
	var awaitingConfirmationAt pgtype.Timestamptz
	var confirmedReceivedAt pgtype.Timestamptz
	var lastProviderSyncAt pgtype.Timestamptz
	var shipmentDeliveredAt pgtype.Timestamptz
	var shipmentCreatedAt pgtype.Timestamptz
	var shipmentUpdatedAt pgtype.Timestamptz

	err :=
		queryer.QueryRow(
			ctx,
			query,
			order.ID,
		).Scan(
			&processingAt,
			&shippedAt,
			&deliveredAt,
			&completedAt,

			&shipment.ID,
			&shipment.OrderID,
			&shipment.OriginWarehouseID,
			&shipment.DeliveryMode,
			&shipment.ProviderCode,
			&shipment.ProviderShipmentID,
			&shipment.ProviderStatus,
			&shipment.CourierName,
			&shipment.CourierReference,
			&shipment.RiderReference,
			&shipment.TrackingNumber,
			&shipment.TrackingURL,
			&shipment.Status,
			&shipmentShippedAt,
			&providerDeliveredAt,
			&awaitingConfirmationAt,
			&confirmedReceivedAt,
			&shipment.ConfirmationSource,
			&shipment.ConfirmedByActorID,
			&shipment.ConfirmationNote,
			&lastProviderSyncAt,
			&shipmentDeliveredAt,
			&shipmentCreatedAt,
			&shipmentUpdatedAt,
		)

	if errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return ErrOrderNotFound
	}

	if err != nil {
		return fmt.Errorf(
			"load order fulfillment state: %w",
			err,
		)
	}

	if processingAt.Valid {
		value :=
			processingAt.Time

		order.ProcessingAt =
			&value
	}

	if shippedAt.Valid {
		value :=
			shippedAt.Time

		order.ShippedAt =
			&value
	}

	if deliveredAt.Valid {
		value :=
			deliveredAt.Time

		order.DeliveredAt =
			&value
	}

	if completedAt.Valid {
		value :=
			completedAt.Time

		order.CompletedAt =
			&value
	}

	if shipment.ID == "" {
		order.Shipment = nil

		return nil
	}

	if shipmentShippedAt.Valid {
		value :=
			shipmentShippedAt.Time

		shipment.ShippedAt =
			&value
	}

	if providerDeliveredAt.Valid {
		value :=
			providerDeliveredAt.Time

		shipment.ProviderDeliveredAt =
			&value
	}

	if awaitingConfirmationAt.Valid {
		value :=
			awaitingConfirmationAt.Time

		shipment.AwaitingConfirmationAt =
			&value
	}

	if confirmedReceivedAt.Valid {
		value :=
			confirmedReceivedAt.Time

		shipment.ConfirmedReceivedAt =
			&value
	}

	if lastProviderSyncAt.Valid {
		value :=
			lastProviderSyncAt.Time

		shipment.LastProviderSyncAt =
			&value
	}

	if shipmentDeliveredAt.Valid {
		value :=
			shipmentDeliveredAt.Time

		shipment.DeliveredAt =
			&value
	}

	if shipmentCreatedAt.Valid {
		shipment.CreatedAt =
			shipmentCreatedAt.Time
	}

	if shipmentUpdatedAt.Valid {
		shipment.UpdatedAt =
			shipmentUpdatedAt.Time
	}

	order.Shipment =
		&shipment

	return nil
}

func (r *Repository) TransitionFulfillmentTx(
	ctx context.Context,
	tx pgx.Tx,
	orderID string,
	fromStatus string,
	toStatus string,
	now time.Time,
) error {
	const query = `
		UPDATE orders

		SET
			status = $3::varchar(30),

			confirmed_at =
			CASE
			WHEN $3::varchar(30) = 'confirmed'
			THEN COALESCE(
			confirmed_at,
			$4::timestamptz
			)
			ELSE confirmed_at
			END,

			processing_at =
				CASE
					WHEN $3::varchar(30) = 'processing'
					THEN COALESCE(
						processing_at,
						$4::timestamptz
					)
					ELSE processing_at
				END,

			completed_at =
				CASE
					WHEN $3::varchar(30) = 'completed'
					THEN COALESCE(
						completed_at,
						$4::timestamptz
					)
					ELSE completed_at
				END,

			updated_at = now()

		WHERE
			id = $1::uuid
			AND status = $2::varchar(30)
	`

	tag, err :=
		tx.Exec(
			ctx,
			query,
			orderID,
			fromStatus,
			toStatus,
			now,
		)
	if err != nil {
		return fmt.Errorf(
			"transition order fulfillment: %w",
			err,
		)
	}

	if tag.RowsAffected() != 1 {
		return ErrFulfillmentTransitionNotAllowed
	}

	return nil
}
