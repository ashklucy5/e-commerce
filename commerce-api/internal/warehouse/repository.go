package warehouse

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil,
			fmt.Errorf(
				"begin warehouse transaction: %w",
				err,
			)
	}

	return tx, nil
}

type queryRower interface {
	QueryRow(
		context.Context,
		string,
		...any,
	) pgx.Row
}

type rowScanner interface {
	Scan(...any) error
}

const warehouseSelect = `
	SELECT
		id::text,
		code,
		name,
		country_code,
		COALESCE(city, ''),
		COALESCE(address_line1, ''),
		status,
		is_default,
		allows_self_pickup,
		created_at,
		updated_at
	FROM warehouses
`

func scanWarehouse(
	row rowScanner,
) (Warehouse, error) {
	var warehouse Warehouse

	err := row.Scan(
		&warehouse.ID,
		&warehouse.Code,
		&warehouse.Name,
		&warehouse.CountryCode,
		&warehouse.City,
		&warehouse.AddressLine1,
		&warehouse.Status,
		&warehouse.IsDefault,
		&warehouse.AllowsSelfPickup,
		&warehouse.CreatedAt,
		&warehouse.UpdatedAt,
	)

	return warehouse, err
}

func warehouseByID(
	ctx context.Context,
	queryer queryRower,
	id string,
	lock bool,
) (Warehouse, error) {
	query := warehouseSelect +
		` WHERE id = $1::uuid`

	if lock {
		query += ` FOR UPDATE`
	}

	warehouse, err :=
		scanWarehouse(
			queryer.QueryRow(
				ctx,
				query,
				id,
			),
		)

	if errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return Warehouse{},
			ErrWarehouseNotFound
	}

	if err != nil {
		return Warehouse{},
			fmt.Errorf(
				"load warehouse: %w",
				err,
			)
	}

	return warehouse, nil
}

func (r *Repository) GetWarehouse(
	ctx context.Context,
	id string,
) (Warehouse, error) {
	return warehouseByID(
		ctx,
		r.db,
		id,
		false,
	)
}

func (r *Repository) GetWarehouseTx(
	ctx context.Context,
	tx pgx.Tx,
	id string,
) (Warehouse, error) {
	return warehouseByID(
		ctx,
		tx,
		id,
		true,
	)
}

func (r *Repository) GetDefaultWarehouseTx(
	ctx context.Context,
	tx pgx.Tx,
) (Warehouse, error) {
	warehouse, err :=
		scanWarehouse(
			tx.QueryRow(
				ctx,
				warehouseSelect+
					` WHERE is_default = true FOR UPDATE`,
			),
		)

	if errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return Warehouse{},
			ErrWarehouseNotFound
	}

	if err != nil {
		return Warehouse{},
			fmt.Errorf(
				"load default warehouse: %w",
				err,
			)
	}

	return warehouse, nil
}

func (r *Repository) ListWarehouses(
	ctx context.Context,
) ([]Warehouse, error) {
	rows, err := r.db.Query(
		ctx,
		warehouseSelect+
			` ORDER BY is_default DESC, code ASC`,
	)
	if err != nil {
		return nil,
			fmt.Errorf(
				"list warehouses: %w",
				err,
			)
	}
	defer rows.Close()

	items := make(
		[]Warehouse,
		0,
	)

	for rows.Next() {
		item, err :=
			scanWarehouse(
				rows,
			)
		if err != nil {
			return nil,
				fmt.Errorf(
					"scan warehouse: %w",
					err,
				)
		}

		items = append(
			items,
			item,
		)
	}

	if err := rows.Err(); err != nil {
		return nil,
			fmt.Errorf(
				"iterate warehouses: %w",
				err,
			)
	}

	return items, nil
}

func (r *Repository) ClearDefaultWarehouseTx(
	ctx context.Context,
	tx pgx.Tx,
) error {
	_, err := tx.Exec(
		ctx,
		`
			UPDATE warehouses
			SET
				is_default = false,
				updated_at = now()
			WHERE is_default = true
		`,
	)
	if err != nil {
		return fmt.Errorf(
			"clear default warehouse: %w",
			err,
		)
	}

	return nil
}

func (r *Repository) CreateWarehouseTx(
	ctx context.Context,
	tx pgx.Tx,
	request CreateWarehouseRequest,
) (string, error) {
	var id string

	err := tx.QueryRow(
		ctx,
		`
			INSERT INTO warehouses (
				code,
				name,
				country_code,
				city,
				address_line1,
				status,
				is_default,
				allows_self_pickup,
				created_at,
				updated_at
			)
			VALUES (
				$1,
				$2,
				$3,
				NULLIF($4, ''),
				NULLIF($5, ''),
				$6,
				$7,
				$8,
				now(),
				now()
			)
			RETURNING id::text
		`,
		request.Code,
		request.Name,
		request.CountryCode,
		request.City,
		request.AddressLine1,
		request.Status,
		request.IsDefault,
		request.AllowsSelfPickup,
	).Scan(
		&id,
	)

	if err != nil {
		var pgErr *pgconn.PgError

		if errors.As(
			err,
			&pgErr,
		) &&
			pgErr.ConstraintName ==
				"warehouses_code_key" {
			return "",
				ErrWarehouseCodeExists
		}

		return "",
			fmt.Errorf(
				"create warehouse: %w",
				err,
			)
	}

	return id, nil
}

type orderItemContext struct {
	ID          string
	OrderID     string
	OrderStatus string
	Quantity    int
}

func (r *Repository) LockOrderItemTx(
	ctx context.Context,
	tx pgx.Tx,
	orderItemID string,
) (orderItemContext, error) {
	var item orderItemContext

	err := tx.QueryRow(
		ctx,
		`
			SELECT
				oi.id::text,
				oi.order_id::text,
				o.status,
				oi.quantity
			FROM order_items oi
			JOIN orders o
				ON o.id = oi.order_id
			WHERE oi.id = $1::uuid
			FOR UPDATE OF oi, o
		`,
		orderItemID,
	).Scan(
		&item.ID,
		&item.OrderID,
		&item.OrderStatus,
		&item.Quantity,
	)

	if errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return orderItemContext{},
			ErrOrderItemNotFound
	}

	if err != nil {
		return orderItemContext{},
			fmt.Errorf(
				"lock order item: %w",
				err,
			)
	}

	return item, nil
}

func (r *Repository) AllocatedQuantityTx(
	ctx context.Context,
	tx pgx.Tx,
	orderItemID string,
) (int, error) {
	var quantity int

	err := tx.QueryRow(
		ctx,
		`
			SELECT
				COALESCE(
					SUM(quantity),
					0
				)::integer
			FROM warehouse_fulfillments
			WHERE
				order_item_id = $1::uuid
				AND status <> 'cancelled'
		`,
		orderItemID,
	).Scan(
		&quantity,
	)
	if err != nil {
		return 0,
			fmt.Errorf(
				"calculate allocated quantity: %w",
				err,
			)
	}

	return quantity, nil
}

const inboundSelect = `
	SELECT
		i.id::text,
		i.reference_code,
		i.origin_country,
		i.destination_warehouse_id::text,
		w.name,
		COALESCE(i.carrier_name, ''),
		COALESCE(i.external_reference, ''),
		COALESCE(i.tracking_number, ''),
		i.status,
		i.eta,
		i.departed_at,
		i.arrived_bangladesh_at,
		i.customs_released_at,
		i.received_at,
		COALESCE(i.notes, ''),
		i.created_at,
		i.updated_at
	FROM inbound_shipments i
	JOIN warehouses w
		ON w.id = i.destination_warehouse_id
`

func scanInbound(
	row rowScanner,
) (InboundShipment, error) {
	var shipment InboundShipment

	var eta pgtype.Timestamptz
	var departed pgtype.Timestamptz
	var arrived pgtype.Timestamptz
	var customs pgtype.Timestamptz
	var received pgtype.Timestamptz

	err := row.Scan(
		&shipment.ID,
		&shipment.ReferenceCode,
		&shipment.OriginCountry,
		&shipment.DestinationWarehouseID,
		&shipment.DestinationWarehouse,
		&shipment.CarrierName,
		&shipment.ExternalReference,
		&shipment.TrackingNumber,
		&shipment.Status,
		&eta,
		&departed,
		&arrived,
		&customs,
		&received,
		&shipment.Notes,
		&shipment.CreatedAt,
		&shipment.UpdatedAt,
	)
	if err != nil {
		return InboundShipment{},
			err
	}

	if eta.Valid {
		value := eta.Time
		shipment.ETA = &value
	}

	if departed.Valid {
		value := departed.Time
		shipment.DepartedAt = &value
	}

	if arrived.Valid {
		value := arrived.Time
		shipment.ArrivedBangladeshAt =
			&value
	}

	if customs.Valid {
		value := customs.Time
		shipment.CustomsReleasedAt =
			&value
	}

	if received.Valid {
		value := received.Time
		shipment.ReceivedAt = &value
	}

	return shipment, nil
}

func inboundByID(
	ctx context.Context,
	queryer queryRower,
	id string,
	lock bool,
) (InboundShipment, error) {
	query := inboundSelect +
		` WHERE i.id = $1::uuid`

	if lock {
		query += ` FOR UPDATE OF i`
	}

	item, err :=
		scanInbound(
			queryer.QueryRow(
				ctx,
				query,
				id,
			),
		)

	if errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return InboundShipment{},
			ErrInboundShipmentNotFound
	}

	if err != nil {
		return InboundShipment{},
			fmt.Errorf(
				"load inbound shipment: %w",
				err,
			)
	}

	return item, nil
}

func (r *Repository) GetInboundShipment(
	ctx context.Context,
	id string,
) (InboundShipment, error) {
	return inboundByID(
		ctx,
		r.db,
		id,
		false,
	)
}

func (r *Repository) LockInboundShipmentTx(
	ctx context.Context,
	tx pgx.Tx,
	id string,
) (InboundShipment, error) {
	return inboundByID(
		ctx,
		tx,
		id,
		true,
	)
}

func (r *Repository) CreateInboundShipmentTx(
	ctx context.Context,
	tx pgx.Tx,
	request CreateInboundShipmentRequest,
	warehouseID string,
) (string, error) {
	var id string

	err := tx.QueryRow(
		ctx,
		`
			INSERT INTO inbound_shipments (
				reference_code,
				origin_country,
				destination_warehouse_id,
				carrier_name,
				external_reference,
				tracking_number,
				status,
				eta,
				notes,
				created_at,
				updated_at
			)
			VALUES (
				$1,
				$2,
				$3::uuid,
				NULLIF($4, ''),
				NULLIF($5, ''),
				NULLIF($6, ''),
				'created',
				$7,
				NULLIF($8, ''),
				now(),
				now()
			)
			RETURNING id::text
		`,
		request.ReferenceCode,
		request.OriginCountry,
		warehouseID,
		request.CarrierName,
		request.ExternalReference,
		request.TrackingNumber,
		request.ETA,
		request.Notes,
	).Scan(
		&id,
	)

	if err != nil {
		var pgErr *pgconn.PgError

		if errors.As(
			err,
			&pgErr,
		) &&
			pgErr.ConstraintName ==
				"inbound_shipments_reference_code_key" {
			return "",
				ErrInboundReferenceExists
		}

		return "",
			fmt.Errorf(
				"create inbound shipment: %w",
				err,
			)
	}

	return id, nil
}

func (r *Repository) UpdateInboundStatusTx(
	ctx context.Context,
	tx pgx.Tx,
	id string,
	status string,
	now time.Time,
) error {
	tag, err := tx.Exec(
		ctx,
		`
			UPDATE inbound_shipments
			SET
				status = $2::varchar(40),
				departed_at = CASE
					WHEN $2::varchar(40) = 'departed_china'
						THEN COALESCE(departed_at, $3::timestamptz)
					ELSE departed_at
				END,
				arrived_bangladesh_at = CASE
					WHEN $2::varchar(40) = 'arrived_bangladesh'
						THEN COALESCE(arrived_bangladesh_at, $3::timestamptz)
					ELSE arrived_bangladesh_at
				END,
				customs_released_at = CASE
					WHEN $2::varchar(40) = 'customs_released'
						THEN COALESCE(customs_released_at, $3::timestamptz)
					ELSE customs_released_at
				END,
				received_at = CASE
					WHEN $2::varchar(40) = 'received_at_warehouse'
						THEN COALESCE(received_at, $3::timestamptz)
					ELSE received_at
				END,
				updated_at = now()
			WHERE id = $1::uuid
		`,
		id,
		status,
		now,
	)
	if err != nil {
		return fmt.Errorf(
			"update inbound shipment status: %w",
			err,
		)
	}

	if tag.RowsAffected() != 1 {
		return ErrInboundShipmentNotFound
	}

	return nil
}

func (r *Repository) InsertInboundEventTx(
	ctx context.Context,
	tx pgx.Tx,
	inboundID string,
	status string,
	message string,
	actorType string,
	actorID string,
	now time.Time,
) error {
	_, err := tx.Exec(
		ctx,
		`
			INSERT INTO inbound_shipment_events (
				inbound_shipment_id,
				status,
				message,
				actor_type,
				actor_id,
				occurred_at,
				created_at
			)
			VALUES (
				$1::uuid,
				$2,
				NULLIF($3, ''),
				NULLIF($4, ''),
				NULLIF($5, ''),
				$6,
				now()
			)
		`,
		inboundID,
		status,
		message,
		actorType,
		actorID,
		now,
	)
	if err != nil {
		return fmt.Errorf(
			"insert inbound event: %w",
			err,
		)
	}

	return nil
}

func (r *Repository) ListInboundEvents(
	ctx context.Context,
	inboundID string,
) ([]InboundShipmentEvent, error) {
	rows, err := r.db.Query(
		ctx,
		`
			SELECT
				id::text,
				inbound_shipment_id::text,
				status,
				COALESCE(message, ''),
				COALESCE(actor_type, ''),
				COALESCE(actor_id, ''),
				occurred_at,
				created_at
			FROM inbound_shipment_events
			WHERE inbound_shipment_id = $1::uuid
			ORDER BY
				occurred_at ASC,
				created_at ASC
		`,
		inboundID,
	)
	if err != nil {
		return nil,
			fmt.Errorf(
				"list inbound events: %w",
				err,
			)
	}
	defer rows.Close()

	items := make(
		[]InboundShipmentEvent,
		0,
	)

	for rows.Next() {
		var event InboundShipmentEvent

		if err := rows.Scan(
			&event.ID,
			&event.InboundShipmentID,
			&event.Status,
			&event.Message,
			&event.ActorType,
			&event.ActorID,
			&event.OccurredAt,
			&event.CreatedAt,
		); err != nil {
			return nil,
				fmt.Errorf(
					"scan inbound event: %w",
					err,
				)
		}

		items = append(
			items,
			event,
		)
	}

	if err := rows.Err(); err != nil {
		return nil,
			fmt.Errorf(
				"iterate inbound events: %w",
				err,
			)
	}

	return items, nil
}

const fulfillmentSelect = `
	SELECT
		wf.id::text,
		wf.order_id::text,
		wf.order_item_id::text,
		oi.sku,
		oi.product_name,
		wf.warehouse_id::text,
		w.code,
		w.name,
		COALESCE(wf.inbound_shipment_id::text, ''),
		COALESCE(i.reference_code, ''),
		wf.source,
		wf.status,
		wf.quantity,
		wf.allocated_at,
		wf.received_at,
		wf.picking_at,
		wf.packed_at,
		wf.ready_for_handoff_at,
		wf.handed_off_at,
		wf.created_at,
		wf.updated_at
	FROM warehouse_fulfillments wf
	JOIN order_items oi
		ON oi.id = wf.order_item_id
	JOIN warehouses w
		ON w.id = wf.warehouse_id
	LEFT JOIN inbound_shipments i
		ON i.id = wf.inbound_shipment_id
`

func scanFulfillment(
	row rowScanner,
) (Fulfillment, error) {
	var fulfillment Fulfillment

	var received pgtype.Timestamptz
	var picking pgtype.Timestamptz
	var packed pgtype.Timestamptz
	var ready pgtype.Timestamptz
	var handed pgtype.Timestamptz

	err := row.Scan(
		&fulfillment.ID,
		&fulfillment.OrderID,
		&fulfillment.OrderItemID,
		&fulfillment.OrderItemSKU,
		&fulfillment.ProductName,
		&fulfillment.WarehouseID,
		&fulfillment.WarehouseCode,
		&fulfillment.WarehouseName,
		&fulfillment.InboundShipmentID,
		&fulfillment.InboundReference,
		&fulfillment.Source,
		&fulfillment.Status,
		&fulfillment.Quantity,
		&fulfillment.AllocatedAt,
		&received,
		&picking,
		&packed,
		&ready,
		&handed,
		&fulfillment.CreatedAt,
		&fulfillment.UpdatedAt,
	)
	if err != nil {
		return Fulfillment{},
			err
	}

	if received.Valid {
		value := received.Time
		fulfillment.ReceivedAt =
			&value
	}

	if picking.Valid {
		value := picking.Time
		fulfillment.PickingAt =
			&value
	}

	if packed.Valid {
		value := packed.Time
		fulfillment.PackedAt =
			&value
	}

	if ready.Valid {
		value := ready.Time
		fulfillment.ReadyForHandoffAt =
			&value
	}

	if handed.Valid {
		value := handed.Time
		fulfillment.HandedOffAt =
			&value
	}

	return fulfillment, nil
}

func fulfillmentByID(
	ctx context.Context,
	queryer queryRower,
	id string,
	lock bool,
) (Fulfillment, error) {
	query := fulfillmentSelect +
		` WHERE wf.id = $1::uuid`

	if lock {
		query += ` FOR UPDATE OF wf`
	}

	item, err :=
		scanFulfillment(
			queryer.QueryRow(
				ctx,
				query,
				id,
			),
		)

	if errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return Fulfillment{},
			ErrFulfillmentNotFound
	}

	if err != nil {
		return Fulfillment{},
			fmt.Errorf(
				"load warehouse fulfillment: %w",
				err,
			)
	}

	return item, nil
}

func (r *Repository) GetFulfillment(
	ctx context.Context,
	id string,
) (Fulfillment, error) {
	return fulfillmentByID(
		ctx,
		r.db,
		id,
		false,
	)
}

func (r *Repository) LockFulfillmentTx(
	ctx context.Context,
	tx pgx.Tx,
	id string,
) (Fulfillment, error) {
	return fulfillmentByID(
		ctx,
		tx,
		id,
		true,
	)
}

func (r *Repository) CreateFulfillmentTx(
	ctx context.Context,
	tx pgx.Tx,
	orderID string,
	orderItemID string,
	warehouseID string,
	inboundID string,
	source string,
	status string,
	quantity int,
	now time.Time,
) (string, error) {
	var id string

	err := tx.QueryRow(
		ctx,
		`
			INSERT INTO warehouse_fulfillments (
				order_id,
				order_item_id,
				warehouse_id,
				inbound_shipment_id,
				source,
				status,
				quantity,
				allocated_at,
				received_at,
				created_at,
				updated_at
			)
			VALUES (
				$1::uuid,
				$2::uuid,
				$3::uuid,
				NULLIF($4::text, '')::uuid,
				$5::varchar(30),
				$6::varchar(40),
				$7::integer,
				$8::timestamptz,
				CASE
					WHEN $6::varchar(40) = 'received'
						THEN $8::timestamptz
					ELSE NULL
				END,
				now(),
				now()
			)
			RETURNING id::text
		`,
		orderID,
		orderItemID,
		warehouseID,
		inboundID,
		source,
		status,
		quantity,
		now,
	).Scan(
		&id,
	)
	if err != nil {
		return "",
			fmt.Errorf(
				"create warehouse fulfillment: %w",
				err,
			)
	}

	return id, nil
}

func (r *Repository) ListOrderFulfillments(
	ctx context.Context,
	orderID string,
) ([]Fulfillment, error) {
	rows, err := r.db.Query(
		ctx,
		fulfillmentSelect+
			`
				WHERE wf.order_id = $1::uuid
				ORDER BY
					wf.created_at ASC,
					wf.id ASC
			`,
		orderID,
	)
	if err != nil {
		return nil,
			fmt.Errorf(
				"list order fulfillments: %w",
				err,
			)
	}
	defer rows.Close()

	items := make(
		[]Fulfillment,
		0,
	)

	for rows.Next() {
		item, err :=
			scanFulfillment(
				rows,
			)
		if err != nil {
			return nil,
				fmt.Errorf(
					"scan warehouse fulfillment: %w",
					err,
				)
		}

		items = append(
			items,
			item,
		)
	}

	if err := rows.Err(); err != nil {
		return nil,
			fmt.Errorf(
				"iterate warehouse fulfillments: %w",
				err,
			)
	}

	return items, nil
}

func (r *Repository) OrderExists(
	ctx context.Context,
	orderID string,
) (bool, error) {
	var exists bool

	err := r.db.QueryRow(
		ctx,
		`
			SELECT EXISTS (
				SELECT 1
				FROM orders
				WHERE id = $1::uuid
			)
		`,
		orderID,
	).Scan(
		&exists,
	)
	if err != nil {
		return false,
			fmt.Errorf(
				"check order: %w",
				err,
			)
	}

	return exists, nil
}

func (r *Repository) UpdateFulfillmentStatusTx(
	ctx context.Context,
	tx pgx.Tx,
	id string,
	status string,
	now time.Time,
) error {
	tag, err := tx.Exec(
		ctx,
		`
			UPDATE warehouse_fulfillments
			SET
				status = $2::varchar(40),
				received_at = CASE
					WHEN $2::varchar(40) = 'received'
						THEN COALESCE(received_at, $3::timestamptz)
					ELSE received_at
				END,
				picking_at = CASE
					WHEN $2::varchar(40) = 'picking'
						THEN COALESCE(picking_at, $3::timestamptz)
					ELSE picking_at
				END,
				packed_at = CASE
					WHEN $2::varchar(40) = 'packed'
						THEN COALESCE(packed_at, $3::timestamptz)
					ELSE packed_at
				END,
				ready_for_handoff_at = CASE
					WHEN $2::varchar(40) = 'ready_for_handoff'
						THEN COALESCE(ready_for_handoff_at, $3::timestamptz)
					ELSE ready_for_handoff_at
				END,
				handed_off_at = CASE
					WHEN $2::varchar(40) = 'handed_off'
						THEN COALESCE(handed_off_at, $3::timestamptz)
					ELSE handed_off_at
				END,
				updated_at = now()
			WHERE id = $1::uuid
		`,
		id,
		status,
		now,
	)
	if err != nil {
		return fmt.Errorf(
			"update fulfillment status: %w",
			err,
		)
	}

	if tag.RowsAffected() != 1 {
		return ErrFulfillmentNotFound
	}

	return nil
}

func (r *Repository) InsertFulfillmentEventTx(
	ctx context.Context,
	tx pgx.Tx,
	fulfillmentID string,
	eventType string,
	fromStatus string,
	toStatus string,
	message string,
	actorType string,
	actorID string,
) error {
	_, err := tx.Exec(
		ctx,
		`
			INSERT INTO warehouse_fulfillment_events (
				fulfillment_id,
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
		fulfillmentID,
		eventType,
		fromStatus,
		toStatus,
		message,
		actorType,
		actorID,
	)
	if err != nil {
		return fmt.Errorf(
			"insert fulfillment event: %w",
			err,
		)
	}

	return nil
}

func (r *Repository) TransitionInboundFulfillmentsTx(
	ctx context.Context,
	tx pgx.Tx,
	inboundID string,
	fromStatus string,
	toStatus string,
	now time.Time,
) ([]string, error) {
	rows, err := tx.Query(
		ctx,
		`
			UPDATE warehouse_fulfillments
			SET
				status = $3::varchar(40),
				received_at = CASE
					WHEN $3::varchar(40) = 'received'
						THEN COALESCE(received_at, $4::timestamptz)
					ELSE received_at
				END,
				updated_at = now()
			WHERE
				inbound_shipment_id = $1::uuid
				AND status = $2::varchar(40)
			RETURNING id::text
		`,
		inboundID,
		fromStatus,
		toStatus,
		now,
	)
	if err != nil {
		return nil,
			fmt.Errorf(
				"transition inbound fulfillments: %w",
				err,
			)
	}
	defer rows.Close()

	ids := make(
		[]string,
		0,
	)

	for rows.Next() {
		var id string

		if err := rows.Scan(
			&id,
		); err != nil {
			return nil,
				fmt.Errorf(
					"scan transitioned fulfillment: %w",
					err,
				)
		}

		ids = append(
			ids,
			id,
		)
	}

	if err := rows.Err(); err != nil {
		return nil,
			fmt.Errorf(
				"iterate transitioned fulfillments: %w",
				err,
			)
	}

	return ids, nil
}
