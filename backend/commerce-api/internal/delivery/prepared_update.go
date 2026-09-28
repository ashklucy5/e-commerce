package delivery

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// UpdatePreparedShipmentRequest edits delivery details while a shipment is still
// pending/prepared and before any warehouse handoff has taken place.
//
// It intentionally shares the same field contract and validation rules as
// PrepareShipmentRequest so a prepared shipment cannot be changed into a state
// that could not have been created normally.
type UpdatePreparedShipmentRequest struct {
	DeliveryMode string `json:"delivery_mode"`

	ProviderCode       string `json:"provider_code"`
	ProviderShipmentID string `json:"provider_shipment_id"`
	ProviderStatus     string `json:"provider_status"`

	CourierName      string `json:"courier_name"`
	CourierReference string `json:"courier_reference"`
	RiderReference   string `json:"rider_reference"`
	TrackingNumber   string `json:"tracking_number"`
	TrackingURL      string `json:"tracking_url"`
}

func (r UpdatePreparedShipmentRequest) prepareRequest() PrepareShipmentRequest {
	return PrepareShipmentRequest{
		DeliveryMode:       r.DeliveryMode,
		ProviderCode:       r.ProviderCode,
		ProviderShipmentID: r.ProviderShipmentID,
		ProviderStatus:     r.ProviderStatus,
		CourierName:        r.CourierName,
		CourierReference:   r.CourierReference,
		RiderReference:     r.RiderReference,
		TrackingNumber:     r.TrackingNumber,
		TrackingURL:        r.TrackingURL,
	}
}

func selfPickupNotAllowedError() error {
	return fmt.Errorf(
		"self pickup is not enabled for this warehouse: %w",
		ErrDeliveryModeMismatch,
	)
}

// UpdatePreparedShipment lets an operator correct delivery mode/provider data
// before custody has left the warehouse. Once a handoff assignment exists, the
// shipment details are locked and normal journey transitions must be used.
func (s *Service) UpdatePreparedShipment(
	ctx context.Context,
	shipmentID string,
	request UpdatePreparedShipmentRequest,
	actorID string,
) (TrackingDetail, error) {
	shipmentID = strings.TrimSpace(shipmentID)
	actorID = normalizeActorID(actorID)

	prepared := request.prepareRequest()
	normalizePrepareShipmentRequest(&prepared)

	if !uuidPattern.MatchString(shipmentID) {
		return TrackingDetail{}, ErrInvalidInput
	}

	if err := validatePrepareShipmentRequest(prepared); err != nil {
		return TrackingDetail{}, err
	}

	// Establish the order before opening the write transaction so this mutation
	// can use the same lock order as dispatch, receipt confirmation and order
	// cancellation: order -> shipment.
	snapshot, err := s.repository.GetShipment(ctx, shipmentID)
	if err != nil {
		return TrackingDetail{}, err
	}

	tx, err := s.repository.Begin(ctx)
	if err != nil {
		return TrackingDetail{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := s.repository.LockOrderTx(ctx, tx, snapshot.OrderID); err != nil {
		return TrackingDetail{}, err
	}

	shipment, err := s.repository.LockShipmentTx(ctx, tx, shipmentID)
	if err != nil {
		return TrackingDetail{}, err
	}

	if shipment.Status != ShipmentStatusPending {
		return TrackingDetail{}, ErrShipmentNotReady
	}

	assignedQuantity, handedOffQuantity, handoffCount, err :=
		s.repository.HandoffCoverageTx(ctx, tx, shipment.ID)
	if err != nil {
		return TrackingDetail{}, err
	}

	if assignedQuantity > 0 || handedOffQuantity > 0 || handoffCount > 0 {
		return TrackingDetail{}, fmt.Errorf(
			"prepared shipment can no longer be edited after warehouse handoff has started: %w",
			ErrShipmentNotReady,
		)
	}

	if prepared.DeliveryMode == DeliveryModeSelfPickup {
		allowed, err := s.repository.WarehouseAllowsSelfPickupTx(
			ctx,
			tx,
			shipment.OriginWarehouseID,
		)
		if err != nil {
			return TrackingDetail{}, err
		}
		if !allowed {
			return TrackingDetail{}, selfPickupNotAllowedError()
		}
	}

	if err := s.repository.UpdatePendingShipmentDetailsTx(
		ctx,
		tx,
		shipment.ID,
		prepared,
	); err != nil {
		return TrackingDetail{}, err
	}

	updated, err := s.repository.LockShipmentTx(ctx, tx, shipment.ID)
	if err != nil {
		return TrackingDetail{}, err
	}

	if err := s.repository.InsertTrackingEventTx(
		ctx,
		tx,
		updated,
		TrackingEventRequest{
			Source:    "admin",
			EventCode: "shipment_details_updated",
			Status:    ShipmentStatusPending,
			Message:   "Prepared shipment details updated",
			Metadata: map[string]any{
				"actor_id":      actorID,
				"delivery_mode": prepared.DeliveryMode,
			},
		},
		time.Now().UTC(),
	); err != nil {
		return TrackingDetail{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return TrackingDetail{}, fmt.Errorf(
			"commit prepared shipment update: %w",
			err,
		)
	}

	return s.GetTracking(ctx, shipment.OrderID)
}

func (r *Repository) WarehouseAllowsSelfPickupTx(
	ctx context.Context,
	tx pgx.Tx,
	warehouseID string,
) (bool, error) {
	var allowed bool

	err := tx.QueryRow(
		ctx,
		`
			SELECT allows_self_pickup
			FROM warehouses
			WHERE id = $1::uuid
		`,
		warehouseID,
	).Scan(&allowed)
	if err != nil {
		if err == pgx.ErrNoRows {
			return false, ErrWarehouseNotReady
		}
		return false, fmt.Errorf(
			"load warehouse self-pickup capability: %w",
			err,
		)
	}

	return allowed, nil
}

func (r *Repository) UpdatePendingShipmentDetailsTx(
	ctx context.Context,
	tx pgx.Tx,
	shipmentID string,
	request PrepareShipmentRequest,
) error {
	tag, err := tx.Exec(
		ctx,
		`
			UPDATE shipments
			SET
				delivery_mode = $2,
				provider_code = NULLIF($3, ''),
				provider_shipment_id = NULLIF($4, ''),
				provider_status = NULLIF($5, ''),
				courier_name = NULLIF($6, ''),
				courier_reference = NULLIF($7, ''),
				rider_reference = NULLIF($8, ''),
				tracking_number = NULLIF($9, ''),
				tracking_url = NULLIF($10, ''),
				updated_at = now()
			WHERE
				id = $1::uuid
				AND status = 'pending'
		`,
		shipmentID,
		request.DeliveryMode,
		request.ProviderCode,
		request.ProviderShipmentID,
		request.ProviderStatus,
		request.CourierName,
		request.CourierReference,
		request.RiderReference,
		request.TrackingNumber,
		request.TrackingURL,
	)
	if err != nil {
		return fmt.Errorf(
			"update prepared shipment details: %w",
			err,
		)
	}

	if tag.RowsAffected() != 1 {
		return ErrShipmentNotReady
	}

	return nil
}
