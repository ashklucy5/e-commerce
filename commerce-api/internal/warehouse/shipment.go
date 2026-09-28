package warehouse

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
)

const (
	HandoffTypeCourier        = "courier"
	HandoffTypeSelfPickup     = "self_pickup"
	HandoffTypeCommunityRider = "community_rider"
)

var (
	ErrShipmentNotFound = errors.New(
		"shipment not found",
	)

	ErrShipmentNotReadyForHandoff = errors.New(
		"shipment is not ready for warehouse handoff",
	)

	ErrShipmentOrderMismatch = errors.New(
		"shipment and fulfillment belong to different orders",
	)

	ErrShipmentWarehouseMismatch = errors.New(
		"shipment and fulfillment belong to different warehouses",
	)

	ErrHandoffTypeMismatch = errors.New(
		"handoff type does not match shipment delivery mode",
	)

	ErrSelfPickupNotAllowed = errors.New(
		"self pickup is not enabled for this warehouse",
	)

	ErrFulfillmentNotReadyForHandoff = errors.New(
		"warehouse fulfillment is not ready for handoff",
	)

	ErrFulfillmentQuantityExceeded = errors.New(
		"handoff quantity exceeds the remaining fulfillment quantity",
	)
)

type HandoffFulfillmentRequest struct {
	FulfillmentID string `json:"fulfillment_id"`
	Quantity      int    `json:"quantity"`
}

type CreateHandoffRequest struct {
	ShipmentID  string                      `json:"shipment_id"`
	HandoffType string                      `json:"handoff_type"`
	Reference   string                      `json:"reference"`
	Items       []HandoffFulfillmentRequest `json:"items"`
}

type Handoff struct {
	ID                   string    `json:"id"`
	ShipmentID           string    `json:"shipment_id"`
	WarehouseID          string    `json:"warehouse_id"`
	HandoffType          string    `json:"handoff_type"`
	Reference            string    `json:"reference,omitempty"`
	HandedOffByActorType string    `json:"handed_off_by_actor_type,omitempty"`
	HandedOffByActorID   string    `json:"handed_off_by_actor_id,omitempty"`
	HandedOffAt          time.Time `json:"handed_off_at"`
	CreatedAt            time.Time `json:"created_at"`
}

type HandoffResult struct {
	Handoff      Handoff       `json:"handoff"`
	Fulfillments []Fulfillment `json:"fulfillments"`
}

type shipmentRecord struct {
	ID                string
	OrderID           string
	OriginWarehouseID string
	DeliveryMode      string
	Status            string
}

func (s *Service) RecordHandoff(
	ctx context.Context,
	request CreateHandoffRequest,
	actorID string,
) (HandoffResult, error) {
	request.ShipmentID = strings.TrimSpace(
		request.ShipmentID,
	)

	request.HandoffType = strings.ToLower(
		strings.TrimSpace(
			request.HandoffType,
		),
	)

	request.Reference = strings.TrimSpace(
		request.Reference,
	)

	actorID = normalizeActorID(
		actorID,
	)

	items, ok := normalizeHandoffItems(
		request.Items,
	)

	if !ok ||
		!uuidPattern.MatchString(
			request.ShipmentID,
		) ||
		!validHandoffType(
			request.HandoffType,
		) ||
		utf8.RuneCountInString(
			request.Reference,
		) > 160 {
		return HandoffResult{},
			ErrInvalidInput
	}

	tx, err := s.repository.Begin(
		ctx,
	)
	if err != nil {
		return HandoffResult{}, err
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	shipment, err :=
		s.repository.LockShipmentTx(
			ctx,
			tx,
			request.ShipmentID,
		)
	if err != nil {
		return HandoffResult{}, err
	}

	// During Logistics v0 we accept both pending and shipped.
	//
	// The existing Order implementation currently creates/updates
	// its shipment while moving the order to shipped. Delivery v0
	// will later own the final handoff -> shipped transition.
	if shipment.Status != "pending" &&
		shipment.Status != "shipped" {
		return HandoffResult{},
			ErrShipmentNotReadyForHandoff
	}

	if shipment.DeliveryMode !=
		request.HandoffType {
		return HandoffResult{},
			ErrHandoffTypeMismatch
	}

	warehouseID := ""

	now := time.Now().UTC()

	updatedIDs := make(
		[]string,
		0,
		len(items),
	)

	for _, requestItem := range items {
		fulfillment, err :=
			s.repository.LockFulfillmentTx(
				ctx,
				tx,
				requestItem.FulfillmentID,
			)
		if err != nil {
			return HandoffResult{}, err
		}

		if fulfillment.Status !=
			FulfillmentStatusReadyForHandoff {
			return HandoffResult{},
				ErrFulfillmentNotReadyForHandoff
		}

		if fulfillment.OrderID !=
			shipment.OrderID {
			return HandoffResult{},
				ErrShipmentOrderMismatch
		}

		if warehouseID == "" {
			warehouseID =
				fulfillment.WarehouseID
		} else if warehouseID !=
			fulfillment.WarehouseID {
			return HandoffResult{},
				ErrShipmentWarehouseMismatch
		}

		if shipment.OriginWarehouseID != "" &&
			shipment.OriginWarehouseID !=
				fulfillment.WarehouseID {
			return HandoffResult{},
				ErrShipmentWarehouseMismatch
		}

		assigned, err :=
			s.repository.AssignedQuantityTx(
				ctx,
				tx,
				fulfillment.ID,
			)
		if err != nil {
			return HandoffResult{}, err
		}

		if assigned+
			requestItem.Quantity >
			fulfillment.Quantity {
			return HandoffResult{},
				ErrFulfillmentQuantityExceeded
		}

		if err :=
			s.repository.CreateShipmentFulfillmentTx(
				ctx,
				tx,
				shipment.ID,
				fulfillment.ID,
				requestItem.Quantity,
			); err != nil {
			return HandoffResult{}, err
		}

		if assigned+
			requestItem.Quantity ==
			fulfillment.Quantity {
			if err :=
				s.repository.UpdateFulfillmentStatusTx(
					ctx,
					tx,
					fulfillment.ID,
					FulfillmentStatusHandedOff,
					now,
				); err != nil {
				return HandoffResult{}, err
			}

			if err :=
				s.repository.InsertFulfillmentEventTx(
					ctx,
					tx,
					fulfillment.ID,
					"handed_off",
					FulfillmentStatusReadyForHandoff,
					FulfillmentStatusHandedOff,
					fmt.Sprintf(
						"Handed off %d unit(s) to shipment %s",
						requestItem.Quantity,
						shipment.ID,
					),
					"admin",
					actorID,
				); err != nil {
				return HandoffResult{}, err
			}
		} else {
			if err :=
				s.repository.InsertFulfillmentEventTx(
					ctx,
					tx,
					fulfillment.ID,
					"partially_handed_off",
					FulfillmentStatusReadyForHandoff,
					FulfillmentStatusReadyForHandoff,
					fmt.Sprintf(
						"Assigned %d unit(s) to shipment %s",
						requestItem.Quantity,
						shipment.ID,
					),
					"admin",
					actorID,
				); err != nil {
				return HandoffResult{}, err
			}
		}

		updatedIDs = append(
			updatedIDs,
			fulfillment.ID,
		)
	}

	warehouse, err :=
		s.repository.GetWarehouseTx(
			ctx,
			tx,
			warehouseID,
		)
	if err != nil {
		return HandoffResult{}, err
	}

	if warehouse.Status !=
		WarehouseStatusActive {
		return HandoffResult{},
			ErrWarehouseInactive
	}

	if request.HandoffType ==
		HandoffTypeSelfPickup &&
		!warehouse.AllowsSelfPickup {
		return HandoffResult{},
			ErrSelfPickupNotAllowed
	}

	if shipment.OriginWarehouseID == "" {
		if err :=
			s.repository.SetShipmentOriginWarehouseTx(
				ctx,
				tx,
				shipment.ID,
				warehouseID,
			); err != nil {
			return HandoffResult{}, err
		}
	}

	handoffID, err :=
		s.repository.CreateHandoffTx(
			ctx,
			tx,
			shipment.ID,
			warehouseID,
			request.HandoffType,
			request.Reference,
			actorID,
			now,
		)
	if err != nil {
		return HandoffResult{}, err
	}

	if err := tx.Commit(
		ctx,
	); err != nil {
		return HandoffResult{},
			fmt.Errorf(
				"commit warehouse handoff: %w",
				err,
			)
	}

	handoff, err :=
		s.repository.GetHandoff(
			ctx,
			handoffID,
		)
	if err != nil {
		return HandoffResult{}, err
	}

	fulfillments := make(
		[]Fulfillment,
		0,
		len(updatedIDs),
	)

	for _, id := range updatedIDs {
		item, err :=
			s.repository.GetFulfillment(
				ctx,
				id,
			)
		if err != nil {
			return HandoffResult{}, err
		}

		fulfillments = append(
			fulfillments,
			item,
		)
	}

	return HandoffResult{
		Handoff:      handoff,
		Fulfillments: fulfillments,
	}, nil
}

func validHandoffType(
	value string,
) bool {
	switch value {
	case HandoffTypeCourier,
		HandoffTypeSelfPickup,
		HandoffTypeCommunityRider:
		return true

	default:
		return false
	}
}

func normalizeHandoffItems(
	items []HandoffFulfillmentRequest,
) ([]HandoffFulfillmentRequest, bool) {
	if len(items) == 0 {
		return nil, false
	}

	normalized := make(
		[]HandoffFulfillmentRequest,
		len(items),
	)

	seen := make(
		map[string]struct{},
		len(items),
	)

	for i, item := range items {
		item.FulfillmentID =
			strings.TrimSpace(
				item.FulfillmentID,
			)

		if !uuidPattern.MatchString(
			item.FulfillmentID,
		) ||
			item.Quantity <= 0 {
			return nil, false
		}

		if _, exists :=
			seen[item.FulfillmentID]; exists {
			return nil, false
		}

		seen[item.FulfillmentID] =
			struct{}{}

		normalized[i] = item
	}

	// Deterministic lock ordering prevents two handoff transactions
	// that contain the same fulfillments in different request order
	// from unnecessarily deadlocking each other.
	sort.Slice(
		normalized,
		func(
			i int,
			j int,
		) bool {
			return normalized[i].FulfillmentID <
				normalized[j].FulfillmentID
		},
	)

	return normalized, true
}

func (r *Repository) LockShipmentTx(
	ctx context.Context,
	tx pgx.Tx,
	shipmentID string,
) (shipmentRecord, error) {
	var item shipmentRecord

	err := tx.QueryRow(
		ctx,
		`
			SELECT
				id::text,
				order_id::text,
				COALESCE(origin_warehouse_id::text, ''),
				delivery_mode,
				status
			FROM shipments
			WHERE id = $1::uuid
			FOR UPDATE
		`,
		shipmentID,
	).Scan(
		&item.ID,
		&item.OrderID,
		&item.OriginWarehouseID,
		&item.DeliveryMode,
		&item.Status,
	)

	if errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return shipmentRecord{},
			ErrShipmentNotFound
	}

	if err != nil {
		return shipmentRecord{},
			fmt.Errorf(
				"lock shipment: %w",
				err,
			)
	}

	return item, nil
}

func (r *Repository) AssignedQuantityTx(
	ctx context.Context,
	tx pgx.Tx,
	fulfillmentID string,
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
			FROM shipment_fulfillments
			WHERE fulfillment_id = $1::uuid
		`,
		fulfillmentID,
	).Scan(
		&quantity,
	)
	if err != nil {
		return 0,
			fmt.Errorf(
				"calculate shipment-assigned quantity: %w",
				err,
			)
	}

	return quantity, nil
}

func (r *Repository) CreateShipmentFulfillmentTx(
	ctx context.Context,
	tx pgx.Tx,
	shipmentID string,
	fulfillmentID string,
	quantity int,
) error {
	_, err := tx.Exec(
		ctx,
		`
			INSERT INTO shipment_fulfillments (
				shipment_id,
				fulfillment_id,
				quantity,
				created_at
			)
			VALUES (
				$1::uuid,
				$2::uuid,
				$3,
				now()
			)
			ON CONFLICT (
				shipment_id,
				fulfillment_id
			)
			DO UPDATE SET
				quantity =
					shipment_fulfillments.quantity +
					EXCLUDED.quantity
		`,
		shipmentID,
		fulfillmentID,
		quantity,
	)
	if err != nil {
		return fmt.Errorf(
			"link fulfillment to shipment: %w",
			err,
		)
	}

	return nil
}

func (r *Repository) SetShipmentOriginWarehouseTx(
	ctx context.Context,
	tx pgx.Tx,
	shipmentID string,
	warehouseID string,
) error {
	_, err := tx.Exec(
		ctx,
		`
			UPDATE shipments
			SET
				origin_warehouse_id = $2::uuid,
				updated_at = now()
			WHERE
				id = $1::uuid
				AND origin_warehouse_id IS NULL
		`,
		shipmentID,
		warehouseID,
	)
	if err != nil {
		return fmt.Errorf(
			"set shipment origin warehouse: %w",
			err,
		)
	}

	return nil
}

func (r *Repository) CreateHandoffTx(
	ctx context.Context,
	tx pgx.Tx,
	shipmentID string,
	warehouseID string,
	handoffType string,
	reference string,
	actorID string,
	now time.Time,
) (string, error) {
	var id string

	err := tx.QueryRow(
		ctx,
		`
			INSERT INTO warehouse_handoffs (
				shipment_id,
				warehouse_id,
				handoff_type,
				reference,
				handed_off_by_actor_type,
				handed_off_by_actor_id,
				handed_off_at,
				created_at
			)
			VALUES (
				$1::uuid,
				$2::uuid,
				$3,
				NULLIF($4, ''),
				'admin',
				$5,
				$6,
				now()
			)
			RETURNING id::text
		`,
		shipmentID,
		warehouseID,
		handoffType,
		reference,
		actorID,
		now,
	).Scan(
		&id,
	)
	if err != nil {
		return "",
			fmt.Errorf(
				"create warehouse handoff: %w",
				err,
			)
	}

	return id, nil
}

func (r *Repository) GetHandoff(
	ctx context.Context,
	id string,
) (Handoff, error) {
	var item Handoff

	err := r.db.QueryRow(
		ctx,
		`
			SELECT
				id::text,
				shipment_id::text,
				warehouse_id::text,
				handoff_type,
				COALESCE(reference, ''),
				COALESCE(handed_off_by_actor_type, ''),
				COALESCE(handed_off_by_actor_id, ''),
				handed_off_at,
				created_at
			FROM warehouse_handoffs
			WHERE id = $1::uuid
		`,
		id,
	).Scan(
		&item.ID,
		&item.ShipmentID,
		&item.WarehouseID,
		&item.HandoffType,
		&item.Reference,
		&item.HandedOffByActorType,
		&item.HandedOffByActorID,
		&item.HandedOffAt,
		&item.CreatedAt,
	)

	if errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return Handoff{},
			ErrShipmentNotFound
	}

	if err != nil {
		return Handoff{},
			fmt.Errorf(
				"load warehouse handoff: %w",
				err,
			)
	}

	return item, nil
}
