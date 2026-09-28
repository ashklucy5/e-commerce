package delivery

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

func (s *Service) PrepareShipment(
	ctx context.Context,
	orderID string,
	request PrepareShipmentRequest,
	actorID string,
) (TrackingDetail, error) {
	orderID = strings.TrimSpace(
		orderID,
	)

	actorID = normalizeActorID(
		actorID,
	)

	normalizePrepareShipmentRequest(
		&request,
	)

	if !uuidPattern.MatchString(
		orderID,
	) {
		return TrackingDetail{},
			ErrInvalidInput
	}

	if err :=
		validatePrepareShipmentRequest(
			request,
		); err != nil {
		return TrackingDetail{}, err
	}

	tx, err :=
		s.repository.Begin(
			ctx,
		)
	if err != nil {
		return TrackingDetail{}, err
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	order, err :=
		s.repository.LockOrderTx(
			ctx,
			tx,
			orderID,
		)
	if err != nil {
		return TrackingDetail{}, err
	}

	existing, exists, err :=
		s.repository.GetShipmentByOrderTx(
			ctx,
			tx,
			orderID,
		)
	if err != nil {
		return TrackingDetail{}, err
	}

	if exists {
		if existing.Status ==
			ShipmentStatusPending &&
			existing.DeliveryMode ==
				request.DeliveryMode {

			if err :=
				tx.Rollback(
					ctx,
				); err != nil {
				return TrackingDetail{},
					fmt.Errorf(
						"rollback existing pending shipment: %w",
						err,
					)
			}

			return s.GetTracking(
				ctx,
				orderID,
			)
		}

		return TrackingDetail{},
			ErrShipmentAlreadyExists
	}

	if order.Status !=
		"processing" {
		return TrackingDetail{},
			ErrOrderNotReady
	}

	if !paymentReadyForDelivery(
		order,
	) {
		return TrackingDetail{},
			ErrPaymentNotReady
	}

	orderQuantity, err :=
		s.repository.OrderQuantityTx(
			ctx,
			tx,
			orderID,
		)
	if err != nil {
		return TrackingDetail{}, err
	}

	allocatedQuantity,
		readyQuantity,
		warehouseID,
		warehouseCount,
		err :=
		s.repository.WarehouseReadinessTx(
			ctx,
			tx,
			orderID,
		)
	if err != nil {
		return TrackingDetail{}, err
	}

	if orderQuantity <= 0 ||
		allocatedQuantity !=
			orderQuantity ||
		readyQuantity !=
			orderQuantity ||
		warehouseID == "" {
		return TrackingDetail{},
			ErrWarehouseNotReady
	}

	if warehouseCount != 1 {
		return TrackingDetail{},
			ErrMultipleWarehousesNotSupported
	}

	if request.DeliveryMode ==
		DeliveryModeSelfPickup {
		allowed, err :=
			s.repository.WarehouseAllowsSelfPickupTx(
				ctx,
				tx,
				warehouseID,
			)
		if err != nil {
			return TrackingDetail{}, err
		}
		if !allowed {
			return TrackingDetail{},
				selfPickupNotAllowedError()
		}
	}

	shipmentID, err :=
		s.repository.CreatePendingShipmentTx(
			ctx,
			tx,
			orderID,
			warehouseID,
			request,
		)
	if err != nil {
		return TrackingDetail{}, err
	}

	shipment, err :=
		s.repository.LockShipmentTx(
			ctx,
			tx,
			shipmentID,
		)
	if err != nil {
		return TrackingDetail{}, err
	}

	if err :=
		s.repository.InsertTrackingEventTx(
			ctx,
			tx,
			shipment,
			TrackingEventRequest{
				Source: "admin",

				EventCode: EventShipmentPrepared,

				Status: ShipmentStatusPending,

				Message: "Shipment prepared for warehouse handoff",

				Metadata: map[string]any{
					"actor_id": actorID,
				},
			},
			time.Now().UTC(),
		); err != nil {
		return TrackingDetail{}, err
	}

	if err :=
		tx.Commit(
			ctx,
		); err != nil {
		return TrackingDetail{},
			fmt.Errorf(
				"commit shipment preparation: %w",
				err,
			)
	}

	return s.GetTracking(
		ctx,
		orderID,
	)
}

func (s *Service) DispatchShipment(
	ctx context.Context,
	shipmentID string,
	request DispatchShipmentRequest,
	actorID string,
) (TrackingDetail, error) {
	shipmentID =
		strings.TrimSpace(
			shipmentID,
		)

	request.Message =
		strings.TrimSpace(
			request.Message,
		)

	actorID =
		normalizeActorID(
			actorID,
		)

	if !uuidPattern.MatchString(
		shipmentID,
	) ||
		utf8.RuneCountInString(
			request.Message,
		) > 1000 {
		return TrackingDetail{},
			ErrInvalidInput
	}

	// Read only enough information to establish the Order ID.
	//
	// The transactional lock order below is deliberately:
	//
	//	order -> shipment
	//
	// This is shared with PrepareShipment, receipt confirmation and
	// Order cancellation.
	snapshot, err :=
		s.repository.GetShipment(
			ctx,
			shipmentID,
		)
	if err != nil {
		return TrackingDetail{}, err
	}

	tx, err :=
		s.repository.Begin(
			ctx,
		)
	if err != nil {
		return TrackingDetail{}, err
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	order, err :=
		s.repository.LockOrderTx(
			ctx,
			tx,
			snapshot.OrderID,
		)
	if err != nil {
		return TrackingDetail{}, err
	}

	shipment, err :=
		s.repository.LockShipmentTx(
			ctx,
			tx,
			shipmentID,
		)
	if err != nil {
		return TrackingDetail{}, err
	}

	if shipment.OrderID !=
		order.ID {
		return TrackingDetail{},
			ErrInvalidInput
	}

	if shipment.Status ==
		ShipmentStatusShipped {

		if order.Status !=
			"shipped" {
			return TrackingDetail{},
				ErrOrderNotReady
		}

		if err :=
			tx.Rollback(
				ctx,
			); err != nil {
			return TrackingDetail{},
				fmt.Errorf(
					"rollback already dispatched shipment: %w",
					err,
				)
		}

		return s.GetTracking(
			ctx,
			shipment.OrderID,
		)
	}

	if shipment.Status !=
		ShipmentStatusPending {
		return TrackingDetail{},
			ErrShipmentNotReady
	}

	if order.Status !=
		"processing" {
		return TrackingDetail{},
			ErrOrderNotReady
	}

	if !paymentReadyForDelivery(
		order,
	) {
		return TrackingDetail{},
			ErrPaymentNotReady
	}

	orderQuantity, err :=
		s.repository.OrderQuantityTx(
			ctx,
			tx,
			shipment.OrderID,
		)
	if err != nil {
		return TrackingDetail{}, err
	}

	assignedQuantity,
		handedOffQuantity,
		handoffCount,
		err :=
		s.repository.HandoffCoverageTx(
			ctx,
			tx,
			shipment.ID,
		)
	if err != nil {
		return TrackingDetail{}, err
	}

	if orderQuantity <= 0 ||
		assignedQuantity !=
			orderQuantity ||
		handedOffQuantity !=
			orderQuantity ||
		handoffCount < 1 {
		return TrackingDetail{},
			ErrHandoffIncomplete
	}

	now :=
		time.Now().UTC()

	if err :=
		s.repository.MarkShipmentShippedTx(
			ctx,
			tx,
			shipment.ID,
			now,
		); err != nil {
		return TrackingDetail{}, err
	}

	if err :=
		s.repository.MarkOrderShippedTx(
			ctx,
			tx,
			shipment.OrderID,
			now,
		); err != nil {
		return TrackingDetail{}, err
	}

	message :=
		request.Message

	if message == "" {
		message =
			"Shipment dispatched after warehouse handoff"
	}

	shipment.Status =
		ShipmentStatusShipped

	if err :=
		s.repository.InsertTrackingEventTx(
			ctx,
			tx,
			shipment,
			TrackingEventRequest{
				Source: "admin",

				EventCode: EventShipmentDispatched,

				Status: ShipmentStatusShipped,

				Message: message,

				Metadata: map[string]any{
					"actor_id": actorID,
				},
			},
			now,
		); err != nil {
		return TrackingDetail{}, err
	}

	if err :=
		s.repository.InsertOrderEventTx(
			ctx,
			tx,
			shipment.OrderID,
			"order_shipped",
			"processing",
			"shipped",
			message,
			"admin",
			actorID,
		); err != nil {
		return TrackingDetail{}, err
	}

	if err :=
		tx.Commit(
			ctx,
		); err != nil {
		return TrackingDetail{},
			fmt.Errorf(
				"commit shipment dispatch: %w",
				err,
			)
	}

	return s.GetTracking(
		ctx,
		shipment.OrderID,
	)
}
