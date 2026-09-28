package delivery

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

func (s *Service) AddTrackingEvent(
	ctx context.Context,
	shipmentID string,
	request TrackingEventRequest,
) (TrackingDetail, error) {
	shipmentID = strings.TrimSpace(
		shipmentID,
	)

	normalizeTrackingEventRequest(
		&request,
	)

	if !uuidPattern.MatchString(
		shipmentID,
	) {
		return TrackingDetail{},
			ErrInvalidInput
	}

	if err :=
		validateTrackingEventRequest(
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

	shipment, err :=
		s.repository.LockShipmentTx(
			ctx,
			tx,
			shipmentID,
		)
	if err != nil {
		return TrackingDetail{}, err
	}

	if shipment.Status ==
		ShipmentStatusCancelled {
		return TrackingDetail{},
			ErrShipmentNotReady
	}

	if err :=
		s.repository.InsertTrackingEventTx(
			ctx,
			tx,
			shipment,
			request,
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
				"commit delivery tracking event: %w",
				err,
			)
	}

	return s.GetTracking(
		ctx,
		shipment.OrderID,
	)
}

func (s *Service) MarkProviderDelivered(
	ctx context.Context,
	shipmentID string,
	request ProviderDeliveredRequest,
) (TrackingDetail, error) {
	shipmentID = strings.TrimSpace(
		shipmentID,
	)

	request.ProviderStatus = strings.TrimSpace(
		request.ProviderStatus,
	)

	request.Message = strings.TrimSpace(
		request.Message,
	)

	request.ExternalEventID = strings.TrimSpace(
		request.ExternalEventID,
	)

	if !uuidPattern.MatchString(
		shipmentID,
	) ||
		utf8.RuneCountInString(
			request.ProviderStatus,
		) > 80 ||
		utf8.RuneCountInString(
			request.Message,
		) > 1000 ||
		utf8.RuneCountInString(
			request.ExternalEventID,
		) > 160 {
		return TrackingDetail{},
			ErrInvalidInput
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

	shipment, err :=
		s.repository.LockShipmentTx(
			ctx,
			tx,
			shipmentID,
		)
	if err != nil {
		return TrackingDetail{}, err
	}

	if shipment.DeliveryMode ==
		DeliveryModeSelfPickup {
		return TrackingDetail{},
			ErrDeliveryModeMismatch
	}

	if shipment.Status ==
		ShipmentStatusAwaitingConfirmation ||
		shipment.Status ==
			ShipmentStatusDelivered {

		if err :=
			tx.Rollback(
				ctx,
			); err != nil {
			return TrackingDetail{},
				fmt.Errorf(
					"rollback repeated provider delivery: %w",
					err,
				)
		}

		return s.GetTracking(
			ctx,
			shipment.OrderID,
		)
	}

	if shipment.Status !=
		ShipmentStatusShipped {
		return TrackingDetail{},
			ErrShipmentNotReady
	}

	syncedAt := time.Now().UTC()

	occurredAt :=
		syncedAt

	if request.OccurredAt != nil &&
		!request.OccurredAt.IsZero() {
		occurredAt =
			request.OccurredAt.UTC()
	}

	if err :=
		s.repository.MarkProviderDeliveredTx(
			ctx,
			tx,
			shipment.ID,
			request.ProviderStatus,
			occurredAt,
			syncedAt,
		); err != nil {
		return TrackingDetail{}, err
	}

	message := request.Message

	if message == "" {
		message =
			"Delivery provider reported the shipment delivered; receipt confirmation is pending"
	}

	source := "provider"

	if shipment.DeliveryMode ==
		DeliveryModeCommunityRider {
		source = "rider"
	}

	shipment.Status =
		ShipmentStatusAwaitingConfirmation

	if err :=
		s.repository.InsertTrackingEventTx(
			ctx,
			tx,
			shipment,
			TrackingEventRequest{
				Source: source,

				EventCode: EventProviderDelivered,

				Status: ShipmentStatusAwaitingConfirmation,

				Message: message,

				ExternalEventID: request.ExternalEventID,

				OccurredAt: &occurredAt,
			},
			syncedAt,
		); err != nil {
		return TrackingDetail{}, err
	}

	if err :=
		tx.Commit(
			ctx,
		); err != nil {
		return TrackingDetail{},
			fmt.Errorf(
				"commit provider delivered transition: %w",
				err,
			)
	}

	return s.GetTracking(
		ctx,
		shipment.OrderID,
	)
}

func (s *Service) ConfirmReceiptForCustomer(
	ctx context.Context,
	customerID string,
	orderID string,
	request ConfirmReceiptRequest,
) (TrackingDetail, error) {
	customerID = strings.TrimSpace(
		customerID,
	)

	orderID = strings.TrimSpace(
		orderID,
	)

	if customerID == "" {
		return TrackingDetail{},
			ErrCustomerAuthenticationRequired
	}

	if !uuidPattern.MatchString(
		customerID,
	) ||
		!uuidPattern.MatchString(
			orderID,
		) {
		return TrackingDetail{},
			ErrInvalidInput
	}

	return s.confirmReceipt(
		ctx,
		orderID,
		ConfirmationSourceCustomer,
		customerID,
		request,
		customerID,
	)
}

func (s *Service) ConfirmReceiptForSupport(
	ctx context.Context,
	orderID string,
	request ConfirmReceiptRequest,
	actorID string,
) (TrackingDetail, error) {
	return s.confirmReceipt(
		ctx,
		orderID,
		ConfirmationSourceSupport,
		normalizeActorID(
			actorID,
		),
		request,
		"",
	)
}

func (s *Service) ConfirmReceiptForAdmin(
	ctx context.Context,
	orderID string,
	request ConfirmReceiptRequest,
	actorID string,
) (TrackingDetail, error) {
	return s.confirmReceipt(
		ctx,
		orderID,
		ConfirmationSourceAdmin,
		normalizeActorID(
			actorID,
		),
		request,
		"",
	)
}

func (s *Service) confirmReceipt(
	ctx context.Context,
	orderID string,
	source string,
	actorID string,
	request ConfirmReceiptRequest,
	customerID string,
) (TrackingDetail, error) {
	orderID = strings.TrimSpace(
		orderID,
	)

	request.Note = strings.TrimSpace(
		request.Note,
	)

	if !uuidPattern.MatchString(
		orderID,
	) ||
		utf8.RuneCountInString(
			request.Note,
		) > 1000 {
		return TrackingDetail{},
			ErrInvalidInput
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

	if source ==
		ConfirmationSourceCustomer {
		if order.CustomerID == "" ||
			order.CustomerID !=
				customerID {
			return TrackingDetail{},
				ErrCustomerOrderMismatch
		}
	}

	shipment, exists, err :=
		s.repository.GetShipmentByOrderTx(
			ctx,
			tx,
			orderID,
		)
	if err != nil {
		return TrackingDetail{}, err
	}

	if !exists {
		return TrackingDetail{},
			ErrShipmentNotFound
	}

	if order.Status ==
		"delivered" &&
		shipment.Status ==
			ShipmentStatusDelivered &&
		shipment.ConfirmedReceivedAt != nil {

		if err :=
			tx.Rollback(
				ctx,
			); err != nil {
			return TrackingDetail{},
				fmt.Errorf(
					"rollback repeated receipt confirmation: %w",
					err,
				)
		}

		return s.GetTracking(
			ctx,
			orderID,
		)
	}

	if order.Status !=
		"shipped" ||
		(shipment.Status !=
			ShipmentStatusShipped &&
			shipment.Status !=
				ShipmentStatusAwaitingConfirmation) {
		return TrackingDetail{},
			ErrShipmentNotReady
	}

	now := time.Now().UTC()

	if err :=
		s.repository.MarkShipmentDeliveredTx(
			ctx,
			tx,
			shipment.ID,
			source,
			actorID,
			request.Note,
			now,
		); err != nil {
		return TrackingDetail{}, err
	}

	if err :=
		s.repository.MarkOrderDeliveredTx(
			ctx,
			tx,
			orderID,
			now,
		); err != nil {
		return TrackingDetail{}, err
	}

	message :=
		"Receipt confirmed"

	if order.PaymentMethod ==
		"cod" {
		message =
			"Receipt confirmed and COD collected"
	}

	shipment.Status =
		ShipmentStatusDelivered

	if err :=
		s.repository.InsertTrackingEventTx(
			ctx,
			tx,
			shipment,
			TrackingEventRequest{
				Source: source,

				EventCode: EventReceiptConfirmed,

				Status: ShipmentStatusDelivered,

				Message: message,
			},
			now,
		); err != nil {
		return TrackingDetail{}, err
	}

	if err :=
		s.repository.InsertOrderEventTx(
			ctx,
			tx,
			orderID,
			"order_delivered",
			"shipped",
			"delivered",
			message,
			source,
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
				"commit receipt confirmation: %w",
				err,
			)
	}

	return s.GetTracking(
		ctx,
		orderID,
	)
}
