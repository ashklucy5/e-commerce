package delivery

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(
	db *pgxpool.Pool,
) *Repository {
	return &Repository{
		db: db,
	}
}

func (r *Repository) Begin(
	ctx context.Context,
) (pgx.Tx, error) {
	tx, err :=
		r.db.Begin(
			ctx,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"begin delivery transaction: %w",
				err,
			)
	}

	return tx, nil
}

type rowScanner interface {
	Scan(...any) error
}

type queryRower interface {
	QueryRow(
		context.Context,
		string,
		...any,
	) pgx.Row
}

const shipmentSelect = `
	SELECT
		id::text,
		order_id::text,
		COALESCE(origin_warehouse_id::text, ''),
		delivery_mode,
		COALESCE(provider_code, ''),
		COALESCE(provider_shipment_id, ''),
		COALESCE(provider_status, ''),
		COALESCE(courier_name, ''),
		COALESCE(courier_reference, ''),
		COALESCE(rider_reference, ''),
		COALESCE(tracking_number, ''),
		COALESCE(tracking_url, ''),
		status,
		shipped_at,
		provider_delivered_at,
		awaiting_confirmation_at,
		confirmed_received_at,
		COALESCE(confirmation_source, ''),
		COALESCE(confirmed_by_actor_id, ''),
		COALESCE(confirmation_note, ''),
		last_provider_sync_at,
		delivered_at,
		created_at,
		updated_at
	FROM shipments
`

func scanShipment(
	row rowScanner,
) (Shipment, error) {
	var item Shipment

	var shippedAt pgtype.Timestamptz
	var providerDeliveredAt pgtype.Timestamptz
	var awaitingConfirmationAt pgtype.Timestamptz
	var confirmedReceivedAt pgtype.Timestamptz
	var lastProviderSyncAt pgtype.Timestamptz
	var deliveredAt pgtype.Timestamptz

	err :=
		row.Scan(
			&item.ID,
			&item.OrderID,
			&item.OriginWarehouseID,
			&item.DeliveryMode,
			&item.ProviderCode,
			&item.ProviderShipmentID,
			&item.ProviderStatus,
			&item.CourierName,
			&item.CourierReference,
			&item.RiderReference,
			&item.TrackingNumber,
			&item.TrackingURL,
			&item.Status,
			&shippedAt,
			&providerDeliveredAt,
			&awaitingConfirmationAt,
			&confirmedReceivedAt,
			&item.ConfirmationSource,
			&item.ConfirmedByActorID,
			&item.ConfirmationNote,
			&lastProviderSyncAt,
			&deliveredAt,
			&item.CreatedAt,
			&item.UpdatedAt,
		)
	if err != nil {
		return Shipment{}, err
	}

	if shippedAt.Valid {
		value := shippedAt.Time
		item.ShippedAt = &value
	}

	if providerDeliveredAt.Valid {
		value :=
			providerDeliveredAt.Time

		item.ProviderDeliveredAt =
			&value
	}

	if awaitingConfirmationAt.Valid {
		value :=
			awaitingConfirmationAt.Time

		item.AwaitingConfirmationAt =
			&value
	}

	if confirmedReceivedAt.Valid {
		value :=
			confirmedReceivedAt.Time

		item.ConfirmedReceivedAt =
			&value
	}

	if lastProviderSyncAt.Valid {
		value :=
			lastProviderSyncAt.Time

		item.LastProviderSyncAt =
			&value
	}

	if deliveredAt.Valid {
		value := deliveredAt.Time
		item.DeliveredAt = &value
	}

	return item, nil
}

func shipmentByID(
	ctx context.Context,
	queryer queryRower,
	shipmentID string,
	lock bool,
) (Shipment, error) {
	query :=
		shipmentSelect +
			` WHERE id = $1::uuid`

	if lock {
		query += ` FOR UPDATE`
	}

	item, err :=
		scanShipment(
			queryer.QueryRow(
				ctx,
				query,
				shipmentID,
			),
		)

	if errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return Shipment{},
			ErrShipmentNotFound
	}

	if err != nil {
		return Shipment{},
			fmt.Errorf(
				"load shipment: %w",
				err,
			)
	}

	return item, nil
}

func (r *Repository) GetShipment(
	ctx context.Context,
	shipmentID string,
) (Shipment, error) {
	return shipmentByID(
		ctx,
		r.db,
		shipmentID,
		false,
	)
}

func (r *Repository) LockShipmentTx(
	ctx context.Context,
	tx pgx.Tx,
	shipmentID string,
) (Shipment, error) {
	return shipmentByID(
		ctx,
		tx,
		shipmentID,
		true,
	)
}

func (r *Repository) GetShipmentByOrder(
	ctx context.Context,
	orderID string,
) (Shipment, error) {
	item, err :=
		scanShipment(
			r.db.QueryRow(
				ctx,
				shipmentSelect+
					` WHERE order_id = $1::uuid`,
				orderID,
			),
		)

	if errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return Shipment{},
			ErrShipmentNotFound
	}

	if err != nil {
		return Shipment{},
			fmt.Errorf(
				"load order shipment: %w",
				err,
			)
	}

	return item, nil
}

func (r *Repository) GetShipmentByOrderTx(
	ctx context.Context,
	tx pgx.Tx,
	orderID string,
) (Shipment, bool, error) {
	item, err :=
		scanShipment(
			tx.QueryRow(
				ctx,
				shipmentSelect+
					` WHERE order_id = $1::uuid FOR UPDATE`,
				orderID,
			),
		)

	if errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return Shipment{},
			false,
			nil
	}

	if err != nil {
		return Shipment{},
			false,
			fmt.Errorf(
				"load order shipment for update: %w",
				err,
			)
	}

	return item, true, nil
}

func (r *Repository) LockOrderTx(
	ctx context.Context,
	tx pgx.Tx,
	orderID string,
) (orderState, error) {
	var item orderState

	err :=
		tx.QueryRow(
			ctx,
			`
				SELECT
					id::text,
					COALESCE(customer_id::text, ''),
					status,
					payment_status,
					payment_method
				FROM orders
				WHERE id = $1::uuid
				FOR UPDATE
			`,
			orderID,
		).Scan(
			&item.ID,
			&item.CustomerID,
			&item.Status,
			&item.PaymentStatus,
			&item.PaymentMethod,
		)

	if errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return orderState{},
			ErrOrderNotFound
	}

	if err != nil {
		return orderState{},
			fmt.Errorf(
				"lock delivery order: %w",
				err,
			)
	}

	return item, nil
}

func (r *Repository) GetOrderState(
	ctx context.Context,
	orderID string,
) (orderState, error) {
	var item orderState

	err :=
		r.db.QueryRow(
			ctx,
			`
				SELECT
					id::text,
					COALESCE(customer_id::text, ''),
					status,
					payment_status,
					payment_method
				FROM orders
				WHERE id = $1::uuid
			`,
			orderID,
		).Scan(
			&item.ID,
			&item.CustomerID,
			&item.Status,
			&item.PaymentStatus,
			&item.PaymentMethod,
		)

	if errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return orderState{},
			ErrOrderNotFound
	}

	if err != nil {
		return orderState{},
			fmt.Errorf(
				"load delivery order: %w",
				err,
			)
	}

	return item, nil
}

func (r *Repository) OrderQuantityTx(
	ctx context.Context,
	tx pgx.Tx,
	orderID string,
) (int, error) {
	var quantity int

	err :=
		tx.QueryRow(
			ctx,
			`
				SELECT
					COALESCE(
						SUM(quantity),
						0
					)::integer
				FROM order_items
				WHERE order_id = $1::uuid
			`,
			orderID,
		).Scan(
			&quantity,
		)
	if err != nil {
		return 0,
			fmt.Errorf(
				"load order quantity for delivery: %w",
				err,
			)
	}

	return quantity, nil
}

func (r *Repository) WarehouseReadinessTx(
	ctx context.Context,
	tx pgx.Tx,
	orderID string,
) (
	allocatedQuantity int,
	readyQuantity int,
	warehouseID string,
	warehouseCount int,
	err error,
) {
	err =
		tx.QueryRow(
			ctx,
			`
				SELECT
					COALESCE(
						SUM(quantity),
						0
					)::integer,

					COALESCE(
						SUM(quantity) FILTER (
							WHERE status = 'ready_for_handoff'
						),
						0
					)::integer,

					COALESCE(
						MIN(warehouse_id::text),
						''
					),

					COUNT(
						DISTINCT warehouse_id
					)::integer

				FROM warehouse_fulfillments

				WHERE
					order_id = $1::uuid
					AND status <> 'cancelled'
			`,
			orderID,
		).Scan(
			&allocatedQuantity,
			&readyQuantity,
			&warehouseID,
			&warehouseCount,
		)

	if err != nil {
		return 0,
			0,
			"",
			0,
			fmt.Errorf(
				"load warehouse readiness: %w",
				err,
			)
	}

	return allocatedQuantity,
		readyQuantity,
		warehouseID,
		warehouseCount,
		nil
}

func (r *Repository) CreatePendingShipmentTx(
	ctx context.Context,
	tx pgx.Tx,
	orderID string,
	warehouseID string,
	request PrepareShipmentRequest,
) (string, error) {
	var id string

	err :=
		tx.QueryRow(
			ctx,
			`
				INSERT INTO shipments (
					order_id,
					origin_warehouse_id,
					delivery_mode,
					provider_code,
					provider_shipment_id,
					provider_status,
					courier_name,
					courier_reference,
					rider_reference,
					tracking_number,
					tracking_url,
					status,
					created_at,
					updated_at
				)
				VALUES (
					$1::uuid,
					$2::uuid,
					$3,
					NULLIF($4, ''),
					NULLIF($5, ''),
					NULLIF($6, ''),
					NULLIF($7, ''),
					NULLIF($8, ''),
					NULLIF($9, ''),
					NULLIF($10, ''),
					NULLIF($11, ''),
					'pending',
					now(),
					now()
				)
				RETURNING id::text
			`,
			orderID,
			warehouseID,
			request.DeliveryMode,
			request.ProviderCode,
			request.ProviderShipmentID,
			request.ProviderStatus,
			request.CourierName,
			request.CourierReference,
			request.RiderReference,
			request.TrackingNumber,
			request.TrackingURL,
		).Scan(
			&id,
		)
	if err != nil {
		return "",
			fmt.Errorf(
				"create pending shipment: %w",
				err,
			)
	}

	return id, nil
}

func (r *Repository) HandoffCoverageTx(
	ctx context.Context,
	tx pgx.Tx,
	shipmentID string,
) (
	assignedQuantity int,
	handedOffQuantity int,
	handoffCount int,
	err error,
) {
	err =
		tx.QueryRow(
			ctx,
			`
				SELECT
					COALESCE(
						SUM(sf.quantity),
						0
					)::integer,

					COALESCE(
						SUM(sf.quantity) FILTER (
							WHERE wf.status = 'handed_off'
						),
						0
					)::integer,

					(
						SELECT COUNT(*)::integer
						FROM warehouse_handoffs wh
						WHERE
							wh.shipment_id = $1::uuid
					)

				FROM shipment_fulfillments sf

				JOIN warehouse_fulfillments wf
					ON wf.id = sf.fulfillment_id

				WHERE
					sf.shipment_id = $1::uuid
			`,
			shipmentID,
		).Scan(
			&assignedQuantity,
			&handedOffQuantity,
			&handoffCount,
		)

	if err != nil {
		return 0,
			0,
			0,
			fmt.Errorf(
				"load warehouse handoff coverage: %w",
				err,
			)
	}

	return assignedQuantity,
		handedOffQuantity,
		handoffCount,
		nil
}

func (r *Repository) MarkShipmentShippedTx(
	ctx context.Context,
	tx pgx.Tx,
	shipmentID string,
	now time.Time,
) error {
	tag, err :=
		tx.Exec(
			ctx,
			`
				UPDATE shipments

				SET
					status = 'shipped',

					shipped_at =
						COALESCE(
							shipped_at,
							$2::timestamptz
						),

					updated_at = now()

				WHERE
					id = $1::uuid
					AND status = 'pending'
			`,
			shipmentID,
			now,
		)
	if err != nil {
		return fmt.Errorf(
			"mark shipment shipped: %w",
			err,
		)
	}

	if tag.RowsAffected() != 1 {
		return ErrShipmentNotReady
	}

	return nil
}

func (r *Repository) MarkOrderShippedTx(
	ctx context.Context,
	tx pgx.Tx,
	orderID string,
	now time.Time,
) error {
	tag, err :=
		tx.Exec(
			ctx,
			`
				UPDATE orders

				SET
					status = 'shipped',

					shipped_at =
						COALESCE(
							shipped_at,
							$2::timestamptz
						),

					updated_at = now()

				WHERE
					id = $1::uuid
					AND status = 'processing'
			`,
			orderID,
			now,
		)
	if err != nil {
		return fmt.Errorf(
			"mark order shipped: %w",
			err,
		)
	}

	if tag.RowsAffected() != 1 {
		return ErrOrderNotReady
	}

	return nil
}

func (r *Repository) MarkProviderDeliveredTx(
	ctx context.Context,
	tx pgx.Tx,
	shipmentID string,
	providerStatus string,
	occurredAt time.Time,
	syncedAt time.Time,
) error {
	tag, err :=
		tx.Exec(
			ctx,
			`
				UPDATE shipments

				SET
					status =
						'awaiting_confirmation',

					provider_status =
						COALESCE(
							NULLIF($2, ''),
							provider_status
						),

					provider_delivered_at =
						COALESCE(
							provider_delivered_at,
							$3::timestamptz
						),

					awaiting_confirmation_at =
						COALESCE(
							awaiting_confirmation_at,
							$3::timestamptz
						),

					last_provider_sync_at =
						$4::timestamptz,

					updated_at =
						now()

				WHERE
					id = $1::uuid
					AND status = 'shipped'
			`,
			shipmentID,
			providerStatus,
			occurredAt,
			syncedAt,
		)
	if err != nil {
		return fmt.Errorf(
			"mark provider delivered: %w",
			err,
		)
	}

	if tag.RowsAffected() != 1 {
		return ErrShipmentNotReady
	}

	return nil
}

func (r *Repository) MarkShipmentDeliveredTx(
	ctx context.Context,
	tx pgx.Tx,
	shipmentID string,
	source string,
	actorID string,
	note string,
	now time.Time,
) error {
	tag, err :=
		tx.Exec(
			ctx,
			`
				UPDATE shipments

				SET
					status = 'delivered',

					confirmed_received_at =
						COALESCE(
							confirmed_received_at,
							$5::timestamptz
						),

					confirmation_source =
						$2,

					confirmed_by_actor_id =
						$3,

					confirmation_note =
						NULLIF($4, ''),

					delivered_at =
						COALESCE(
							delivered_at,
							$5::timestamptz
						),

					updated_at =
						now()

				WHERE
					id = $1::uuid
					AND status IN (
						'shipped',
						'awaiting_confirmation'
					)
			`,
			shipmentID,
			source,
			actorID,
			note,
			now,
		)
	if err != nil {
		return fmt.Errorf(
			"confirm shipment receipt: %w",
			err,
		)
	}

	if tag.RowsAffected() != 1 {
		return ErrShipmentNotReady
	}

	return nil
}

func (r *Repository) MarkOrderDeliveredTx(
	ctx context.Context,
	tx pgx.Tx,
	orderID string,
	now time.Time,
) error {
	tag, err :=
		tx.Exec(
			ctx,
			`
				UPDATE orders

				SET
					status = 'delivered',

					delivered_at =
						COALESCE(
							delivered_at,
							$2::timestamptz
						),

					payment_status =
						CASE
							WHEN
								payment_method = 'cod'
								AND payment_status = 'cod_pending'
							THEN
								'cod_collected'
							ELSE
								payment_status
						END,

					paid_at =
						CASE
							WHEN
								payment_method = 'cod'
								AND payment_status = 'cod_pending'
							THEN
								COALESCE(
									paid_at,
									$2::timestamptz
								)
							ELSE
								paid_at
						END,

					updated_at = now()

				WHERE
					id = $1::uuid
					AND status = 'shipped'
			`,
			orderID,
			now,
		)
	if err != nil {
		return fmt.Errorf(
			"mark order delivered after confirmation: %w",
			err,
		)
	}

	if tag.RowsAffected() != 1 {
		return ErrOrderNotReady
	}

	return nil
}

func (r *Repository) InsertOrderEventTx(
	ctx context.Context,
	tx pgx.Tx,
	orderID string,
	eventType string,
	fromStatus string,
	toStatus string,
	message string,
	actorType string,
	actorID string,
) error {
	_, err :=
		tx.Exec(
			ctx,
			`
				INSERT INTO order_events (
					order_id,
					event_type,
					from_status,
					to_status,
					message,
					actor_type,
					actor_id,
					created_at
				)

				VALUES (
					$1::uuid,
					$2,
					NULLIF($3, ''),
					NULLIF($4, ''),
					NULLIF($5, ''),
					NULLIF($6, ''),
					NULLIF($7, ''),
					now()
				)
			`,
			orderID,
			eventType,
			fromStatus,
			toStatus,
			message,
			actorType,
			actorID,
		)
	if err != nil {
		return fmt.Errorf(
			"insert delivery order event: %w",
			err,
		)
	}

	/*
		The only delivery order events currently mapped to external
		notifications are:

		    order_shipped
		    order_delivered

		Provider-reported delivery never reaches this final-delivered
		notification path. Receipt confirmation does.
	*/
	if err :=
		r.enqueueDeliveryOrderEventNotificationTx(
			ctx,
			tx,
			orderID,
			eventType,
		); err != nil {

		return err
	}

	return nil
}

func (r *Repository) InsertTrackingEventTx(
	ctx context.Context,
	tx pgx.Tx,
	shipment Shipment,
	request TrackingEventRequest,
	now time.Time,
) error {
	occurredAt := now

	if request.OccurredAt != nil &&
		!request.OccurredAt.IsZero() {
		occurredAt =
			request.OccurredAt.UTC()
	}

	metadataJSON := ""

	if len(
		request.Metadata,
	) > 0 {
		encoded, err :=
			json.Marshal(
				request.Metadata,
			)
		if err != nil {
			return fmt.Errorf(
				"encode delivery tracking metadata: %w",
				err,
			)
		}

		metadataJSON =
			string(
				encoded,
			)
	}

	_, err :=
		tx.Exec(
			ctx,
			`
				INSERT INTO delivery_tracking_events (
					shipment_id,
					order_id,
					source,
					event_code,
					status,
					message,
					latitude,
					longitude,
					external_event_id,
					metadata,
					occurred_at,
					created_at
				)
				VALUES (
					$1::uuid,
					$2::uuid,
					$3,
					$4,
					NULLIF($5, ''),
					NULLIF($6, ''),
					$7,
					$8,
					NULLIF($9, ''),
					NULLIF($10, '')::jsonb,
					$11::timestamptz,
					now()
				)
				ON CONFLICT (
					shipment_id,
					external_event_id
				)
				WHERE external_event_id IS NOT NULL
				DO NOTHING
			`,
			shipment.ID,
			shipment.OrderID,
			request.Source,
			request.EventCode,
			request.Status,
			request.Message,
			request.Latitude,
			request.Longitude,
			request.ExternalEventID,
			metadataJSON,
			occurredAt,
		)
	if err != nil {
		return fmt.Errorf(
			"insert delivery tracking event: %w",
			err,
		)
	}

	return nil
}

func (r *Repository) ListTrackingEvents(
	ctx context.Context,
	shipmentID string,
) ([]TrackingEvent, error) {
	rows, err :=
		r.db.Query(
			ctx,
			`
				SELECT
					id::text,
					shipment_id::text,
					order_id::text,
					source,
					event_code,
					COALESCE(status, ''),
					COALESCE(message, ''),
					latitude,
					longitude,
					COALESCE(external_event_id, ''),
					metadata,
					occurred_at,
					created_at

				FROM delivery_tracking_events

				WHERE shipment_id = $1::uuid

				ORDER BY
					occurred_at ASC,
					created_at ASC,
					id ASC
			`,
			shipmentID,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"list delivery tracking events: %w",
				err,
			)
	}

	defer rows.Close()

	items :=
		make(
			[]TrackingEvent,
			0,
		)

	for rows.Next() {
		var item TrackingEvent
		var latitude pgtype.Float8
		var longitude pgtype.Float8
		var metadataJSON []byte

		if err :=
			rows.Scan(
				&item.ID,
				&item.ShipmentID,
				&item.OrderID,
				&item.Source,
				&item.EventCode,
				&item.Status,
				&item.Message,
				&latitude,
				&longitude,
				&item.ExternalEventID,
				&metadataJSON,
				&item.OccurredAt,
				&item.CreatedAt,
			); err != nil {
			return nil,
				fmt.Errorf(
					"scan delivery tracking event: %w",
					err,
				)
		}

		if latitude.Valid {
			value :=
				latitude.Float64

			item.Latitude =
				&value
		}

		if longitude.Valid {
			value :=
				longitude.Float64

			item.Longitude =
				&value
		}

		if len(
			metadataJSON,
		) > 0 {
			if err :=
				json.Unmarshal(
					metadataJSON,
					&item.Metadata,
				); err != nil {
				return nil,
					fmt.Errorf(
						"decode delivery tracking metadata: %w",
						err,
					)
			}
		}

		items =
			append(
				items,
				item,
			)
	}

	if err := rows.Err(); err != nil {
		return nil,
			fmt.Errorf(
				"iterate delivery tracking events: %w",
				err,
			)
	}

	return items, nil
}
