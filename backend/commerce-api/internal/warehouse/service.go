package warehouse

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
)

var uuidPattern = regexp.MustCompile(
	`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`,
)

type Service struct {
	repository *Repository
}

func NewService(
	repository *Repository,
) *Service {
	return &Service{
		repository: repository,
	}
}

func (s *Service) ListWarehouses(
	ctx context.Context,
) ([]Warehouse, error) {
	return s.repository.ListWarehouses(
		ctx,
	)
}

func (s *Service) CreateWarehouse(
	ctx context.Context,
	request CreateWarehouseRequest,
) (Warehouse, error) {
	normalizeWarehouseRequest(
		&request,
	)

	if !validWarehouseRequest(
		request,
	) {
		return Warehouse{},
			ErrInvalidInput
	}

	tx, err := s.repository.Begin(
		ctx,
	)
	if err != nil {
		return Warehouse{}, err
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if request.IsDefault {
		if err :=
			s.repository.ClearDefaultWarehouseTx(
				ctx,
				tx,
			); err != nil {
			return Warehouse{}, err
		}
	}

	id, err :=
		s.repository.CreateWarehouseTx(
			ctx,
			tx,
			request,
		)
	if err != nil {
		return Warehouse{}, err
	}

	if err := tx.Commit(
		ctx,
	); err != nil {
		return Warehouse{},
			fmt.Errorf(
				"commit warehouse creation: %w",
				err,
			)
	}

	return s.repository.GetWarehouse(
		ctx,
		id,
	)
}

func (s *Service) CreateFulfillment(
	ctx context.Context,
	request CreateFulfillmentRequest,
	actorID string,
) (Fulfillment, error) {
	normalizeFulfillmentRequest(
		&request,
	)

	actorID =
		normalizeActorID(
			actorID,
		)

	if !uuidPattern.MatchString(
		request.OrderItemID,
	) ||
		request.Quantity <= 0 ||
		(request.WarehouseID != "" &&
			!uuidPattern.MatchString(
				request.WarehouseID,
			)) ||
		(request.InboundShipmentID != "" &&
			!uuidPattern.MatchString(
				request.InboundShipmentID,
			)) ||
		!validFulfillmentSource(
			request.Source,
		) {
		return Fulfillment{},
			ErrInvalidInput
	}

	if request.Source ==
		FulfillmentSourceBangladeshStock &&
		request.InboundShipmentID != "" {
		return Fulfillment{},
			ErrInvalidInput
	}

	if request.Source ==
		FulfillmentSourceChinaInbound &&
		request.InboundShipmentID == "" {
		return Fulfillment{},
			ErrInboundShipmentRequired
	}

	tx, err := s.repository.Begin(
		ctx,
	)
	if err != nil {
		return Fulfillment{}, err
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	item, err :=
		s.repository.LockOrderItemTx(
			ctx,
			tx,
			request.OrderItemID,
		)
	if err != nil {
		return Fulfillment{}, err
	}

	if item.OrderStatus != "confirmed" &&
		item.OrderStatus != "processing" {
		return Fulfillment{},
			ErrOrderNotFulfillable
	}

	warehouse, err :=
		s.resolveWarehouseTx(
			ctx,
			tx,
			request.WarehouseID,
		)
	if err != nil {
		return Fulfillment{}, err
	}

	if warehouse.Status !=
		WarehouseStatusActive {
		return Fulfillment{},
			ErrWarehouseInactive
	}

	allocated, err :=
		s.repository.AllocatedQuantityTx(
			ctx,
			tx,
			request.OrderItemID,
		)
	if err != nil {
		return Fulfillment{}, err
	}

	if request.Quantity >
		item.Quantity-allocated {
		return Fulfillment{},
			ErrAllocationExceeded
	}

	status :=
		FulfillmentStatusAllocated

	inboundID := ""

	message :=
		"Allocated from Bangladesh stock"

	if request.Source ==
		FulfillmentSourceChinaInbound {
		inbound, err :=
			s.repository.LockInboundShipmentTx(
				ctx,
				tx,
				request.InboundShipmentID,
			)
		if err != nil {
			return Fulfillment{}, err
		}

		if inbound.OriginCountry != "CN" {
			return Fulfillment{},
				ErrInboundSourceMismatch
		}

		if inbound.DestinationWarehouseID !=
			warehouse.ID {
			return Fulfillment{},
				ErrInboundWarehouseMismatch
		}

		if inbound.Status ==
			InboundStatusCancelled {
			return Fulfillment{},
				ErrInboundTransitionNotAllowed
		}

		inboundID = inbound.ID

		status =
			FulfillmentStatusWaitingInbound

		message =
			"Allocated to China inbound shipment " +
				inbound.ReferenceCode

		if inbound.Status ==
			InboundStatusReceivedAtWarehouse {
			status =
				FulfillmentStatusReceived

			message =
				"Allocated from received China inbound shipment " +
					inbound.ReferenceCode
		}
	}

	now := time.Now().UTC()

	id, err :=
		s.repository.CreateFulfillmentTx(
			ctx,
			tx,
			item.OrderID,
			item.ID,
			warehouse.ID,
			inboundID,
			request.Source,
			status,
			request.Quantity,
			now,
		)
	if err != nil {
		return Fulfillment{}, err
	}

	if err :=
		s.repository.InsertFulfillmentEventTx(
			ctx,
			tx,
			id,
			"allocated",
			"",
			status,
			message,
			"admin",
			actorID,
		); err != nil {
		return Fulfillment{}, err
	}

	if err := tx.Commit(
		ctx,
	); err != nil {
		return Fulfillment{},
			fmt.Errorf(
				"commit warehouse allocation: %w",
				err,
			)
	}

	return s.repository.GetFulfillment(
		ctx,
		id,
	)
}

func (s *Service) ListOrderFulfillments(
	ctx context.Context,
	orderID string,
) ([]Fulfillment, error) {
	orderID =
		strings.TrimSpace(
			orderID,
		)

	if !uuidPattern.MatchString(
		orderID,
	) {
		return nil,
			ErrInvalidInput
	}

	exists, err :=
		s.repository.OrderExists(
			ctx,
			orderID,
		)
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil,
			ErrOrderNotFound
	}

	return s.repository.ListOrderFulfillments(
		ctx,
		orderID,
	)
}

func (s *Service) UpdateFulfillmentStatus(
	ctx context.Context,
	id string,
	request UpdateFulfillmentStatusRequest,
	actorID string,
) (Fulfillment, error) {
	id =
		strings.TrimSpace(
			id,
		)

	request.Status =
		strings.ToLower(
			strings.TrimSpace(
				request.Status,
			),
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
		id,
	) ||
		!validFulfillmentStatus(
			request.Status,
		) ||
		utf8.RuneCountInString(
			request.Message,
		) > 1000 {
		return Fulfillment{},
			ErrInvalidInput
	}

	tx, err := s.repository.Begin(
		ctx,
	)
	if err != nil {
		return Fulfillment{}, err
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	current, err :=
		s.repository.LockFulfillmentTx(
			ctx,
			tx,
			id,
		)
	if err != nil {
		return Fulfillment{}, err
	}

	if current.Status ==
		request.Status {
		return current, nil
	}

	if !allowedManualFulfillmentTransition(
		current.Status,
		request.Status,
	) {
		return Fulfillment{},
			ErrFulfillmentTransitionNotAllowed
	}

	now := time.Now().UTC()

	if err :=
		s.repository.UpdateFulfillmentStatusTx(
			ctx,
			tx,
			id,
			request.Status,
			now,
		); err != nil {
		return Fulfillment{}, err
	}

	message := request.Message

	if message == "" {
		message =
			"Warehouse fulfillment moved to " +
				request.Status
	}

	if err :=
		s.repository.InsertFulfillmentEventTx(
			ctx,
			tx,
			id,
			"status_changed",
			current.Status,
			request.Status,
			message,
			"admin",
			actorID,
		); err != nil {
		return Fulfillment{}, err
	}

	if err := tx.Commit(
		ctx,
	); err != nil {
		return Fulfillment{},
			fmt.Errorf(
				"commit warehouse fulfillment transition: %w",
				err,
			)
	}

	return s.repository.GetFulfillment(
		ctx,
		id,
	)
}

func (s *Service) CreateInboundShipment(
	ctx context.Context,
	request CreateInboundShipmentRequest,
	actorID string,
) (InboundShipmentDetail, error) {
	normalizeInboundRequest(
		&request,
	)

	actorID =
		normalizeActorID(
			actorID,
		)

	if !validInboundRequest(
		request,
	) {
		return InboundShipmentDetail{},
			ErrInvalidInput
	}

	tx, err := s.repository.Begin(
		ctx,
	)
	if err != nil {
		return InboundShipmentDetail{},
			err
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	warehouse, err :=
		s.resolveWarehouseTx(
			ctx,
			tx,
			request.DestinationWarehouseID,
		)
	if err != nil {
		return InboundShipmentDetail{},
			err
	}

	if warehouse.Status !=
		WarehouseStatusActive {
		return InboundShipmentDetail{},
			ErrWarehouseInactive
	}

	id, err :=
		s.repository.CreateInboundShipmentTx(
			ctx,
			tx,
			request,
			warehouse.ID,
		)
	if err != nil {
		return InboundShipmentDetail{},
			err
	}

	now := time.Now().UTC()

	if err :=
		s.repository.InsertInboundEventTx(
			ctx,
			tx,
			id,
			InboundStatusCreated,
			"Inbound shipment created",
			"admin",
			actorID,
			now,
		); err != nil {
		return InboundShipmentDetail{},
			err
	}

	if err := tx.Commit(
		ctx,
	); err != nil {
		return InboundShipmentDetail{},
			fmt.Errorf(
				"commit inbound shipment creation: %w",
				err,
			)
	}

	return s.GetInboundShipment(
		ctx,
		id,
	)
}

func (s *Service) GetInboundShipment(
	ctx context.Context,
	id string,
) (InboundShipmentDetail, error) {
	id =
		strings.TrimSpace(
			id,
		)

	if !uuidPattern.MatchString(
		id,
	) {
		return InboundShipmentDetail{},
			ErrInboundShipmentNotFound
	}

	shipment, err :=
		s.repository.GetInboundShipment(
			ctx,
			id,
		)
	if err != nil {
		return InboundShipmentDetail{},
			err
	}

	events, err :=
		s.repository.ListInboundEvents(
			ctx,
			id,
		)
	if err != nil {
		return InboundShipmentDetail{},
			err
	}

	return InboundShipmentDetail{
		Shipment: shipment,
		Events:   events,
	}, nil
}

func (s *Service) UpdateInboundShipmentStatus(
	ctx context.Context,
	id string,
	request UpdateInboundShipmentStatusRequest,
	actorID string,
) (InboundShipmentDetail, error) {
	id =
		strings.TrimSpace(
			id,
		)

	request.Status =
		strings.ToLower(
			strings.TrimSpace(
				request.Status,
			),
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
		id,
	) ||
		!validInboundStatus(
			request.Status,
		) ||
		utf8.RuneCountInString(
			request.Message,
		) > 1000 {
		return InboundShipmentDetail{},
			ErrInvalidInput
	}

	tx, err := s.repository.Begin(
		ctx,
	)
	if err != nil {
		return InboundShipmentDetail{},
			err
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	current, err :=
		s.repository.LockInboundShipmentTx(
			ctx,
			tx,
			id,
		)
	if err != nil {
		return InboundShipmentDetail{},
			err
	}

	if current.Status ==
		request.Status {
		if err := tx.Rollback(
			ctx,
		); err != nil {
			return InboundShipmentDetail{},
				fmt.Errorf(
					"rollback unchanged inbound transition: %w",
					err,
				)
		}

		return s.GetInboundShipment(
			ctx,
			id,
		)
	}

	if !allowedInboundTransition(
		current.Status,
		request.Status,
	) {
		return InboundShipmentDetail{},
			ErrInboundTransitionNotAllowed
	}

	now := time.Now().UTC()

	if err :=
		s.repository.UpdateInboundStatusTx(
			ctx,
			tx,
			id,
			request.Status,
			now,
		); err != nil {
		return InboundShipmentDetail{},
			err
	}

	message := request.Message

	if message == "" {
		message =
			"Inbound shipment moved to " +
				request.Status
	}

	if err :=
		s.repository.InsertInboundEventTx(
			ctx,
			tx,
			id,
			request.Status,
			message,
			"admin",
			actorID,
			now,
		); err != nil {
		return InboundShipmentDetail{},
			err
	}

	if request.Status ==
		InboundStatusReceivedAtWarehouse {
		if err :=
			s.transitionInboundFulfillments(
				ctx,
				tx,
				id,
				FulfillmentStatusWaitingInbound,
				FulfillmentStatusReceived,
				"inbound_received",
				"China inbound shipment received at warehouse",
				actorID,
				now,
			); err != nil {
			return InboundShipmentDetail{},
				err
		}
	}

	if request.Status ==
		InboundStatusCancelled {
		if err :=
			s.transitionInboundFulfillments(
				ctx,
				tx,
				id,
				FulfillmentStatusWaitingInbound,
				FulfillmentStatusCancelled,
				"inbound_cancelled",
				"China inbound shipment was cancelled",
				actorID,
				now,
			); err != nil {
			return InboundShipmentDetail{},
				err
		}
	}

	if err := tx.Commit(
		ctx,
	); err != nil {
		return InboundShipmentDetail{},
			fmt.Errorf(
				"commit inbound shipment transition: %w",
				err,
			)
	}

	return s.GetInboundShipment(
		ctx,
		id,
	)
}

func (s *Service) transitionInboundFulfillments(
	ctx context.Context,
	tx pgx.Tx,
	inboundID string,
	fromStatus string,
	toStatus string,
	eventType string,
	message string,
	actorID string,
	now time.Time,
) error {
	ids, err :=
		s.repository.TransitionInboundFulfillmentsTx(
			ctx,
			tx,
			inboundID,
			fromStatus,
			toStatus,
			now,
		)
	if err != nil {
		return err
	}

	for _, id := range ids {
		if err :=
			s.repository.InsertFulfillmentEventTx(
				ctx,
				tx,
				id,
				eventType,
				fromStatus,
				toStatus,
				message,
				"admin",
				actorID,
			); err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) resolveWarehouseTx(
	ctx context.Context,
	tx pgx.Tx,
	id string,
) (Warehouse, error) {
	if id == "" {
		return s.repository.GetDefaultWarehouseTx(
			ctx,
			tx,
		)
	}

	return s.repository.GetWarehouseTx(
		ctx,
		tx,
		id,
	)
}

func normalizeWarehouseRequest(
	request *CreateWarehouseRequest,
) {
	request.Code =
		strings.ToUpper(
			strings.TrimSpace(
				request.Code,
			),
		)

	request.Name =
		strings.TrimSpace(
			request.Name,
		)

	request.CountryCode =
		strings.ToUpper(
			strings.TrimSpace(
				request.CountryCode,
			),
		)

	request.City =
		strings.TrimSpace(
			request.City,
		)

	request.AddressLine1 =
		strings.TrimSpace(
			request.AddressLine1,
		)

	request.Status =
		strings.ToLower(
			strings.TrimSpace(
				request.Status,
			),
		)

	if request.CountryCode == "" {
		request.CountryCode = "BD"
	}

	if request.Status == "" {
		request.Status =
			WarehouseStatusActive
	}
}

func validWarehouseRequest(
	request CreateWarehouseRequest,
) bool {
	return request.Code != "" &&
		request.Name != "" &&
		utf8.RuneCountInString(
			request.Code,
		) <= 40 &&
		utf8.RuneCountInString(
			request.Name,
		) <= 160 &&
		len(request.CountryCode) == 2 &&
		utf8.RuneCountInString(
			request.City,
		) <= 120 &&
		utf8.RuneCountInString(
			request.AddressLine1,
		) <= 255 &&
		(request.Status ==
			WarehouseStatusActive ||
			request.Status ==
				WarehouseStatusInactive)
}

func normalizeFulfillmentRequest(
	request *CreateFulfillmentRequest,
) {
	request.OrderItemID =
		strings.TrimSpace(
			request.OrderItemID,
		)

	request.WarehouseID =
		strings.TrimSpace(
			request.WarehouseID,
		)

	request.InboundShipmentID =
		strings.TrimSpace(
			request.InboundShipmentID,
		)

	request.Source =
		strings.ToLower(
			strings.TrimSpace(
				request.Source,
			),
		)
}

func normalizeInboundRequest(
	request *CreateInboundShipmentRequest,
) {
	request.ReferenceCode =
		strings.ToUpper(
			strings.TrimSpace(
				request.ReferenceCode,
			),
		)

	request.OriginCountry =
		strings.ToUpper(
			strings.TrimSpace(
				request.OriginCountry,
			),
		)

	request.DestinationWarehouseID =
		strings.TrimSpace(
			request.DestinationWarehouseID,
		)

	request.CarrierName =
		strings.TrimSpace(
			request.CarrierName,
		)

	request.ExternalReference =
		strings.TrimSpace(
			request.ExternalReference,
		)

	request.TrackingNumber =
		strings.TrimSpace(
			request.TrackingNumber,
		)

	request.Notes =
		strings.TrimSpace(
			request.Notes,
		)

	if request.OriginCountry == "" {
		request.OriginCountry = "CN"
	}
}

func validInboundRequest(
	request CreateInboundShipmentRequest,
) bool {
	if request.ReferenceCode == "" ||
		utf8.RuneCountInString(
			request.ReferenceCode,
		) > 80 ||
		request.OriginCountry != "CN" ||
		utf8.RuneCountInString(
			request.CarrierName,
		) > 160 ||
		utf8.RuneCountInString(
			request.ExternalReference,
		) > 160 ||
		utf8.RuneCountInString(
			request.TrackingNumber,
		) > 160 ||
		utf8.RuneCountInString(
			request.Notes,
		) > 5000 {
		return false
	}

	return request.DestinationWarehouseID == "" ||
		uuidPattern.MatchString(
			request.DestinationWarehouseID,
		)
}

func normalizeActorID(
	actorID string,
) string {
	actorID =
		strings.TrimSpace(
			actorID,
		)

	if actorID == "" {
		return "development-admin"
	}

	return actorID
}

func validFulfillmentSource(
	source string,
) bool {
	return source ==
		FulfillmentSourceBangladeshStock ||
		source ==
			FulfillmentSourceChinaInbound
}

func validFulfillmentStatus(
	status string,
) bool {
	switch status {
	case FulfillmentStatusAllocated,
		FulfillmentStatusWaitingInbound,
		FulfillmentStatusReceived,
		FulfillmentStatusPicking,
		FulfillmentStatusPacked,
		FulfillmentStatusReadyForHandoff,
		FulfillmentStatusHandedOff,
		FulfillmentStatusCancelled:
		return true

	default:
		return false
	}
}

func allowedManualFulfillmentTransition(
	from string,
	to string,
) bool {
	if to ==
		FulfillmentStatusCancelled {
		switch from {
		case FulfillmentStatusAllocated,
			FulfillmentStatusWaitingInbound,
			FulfillmentStatusReceived,
			FulfillmentStatusPicking,
			FulfillmentStatusPacked,
			FulfillmentStatusReadyForHandoff:
			return true

		default:
			return false
		}
	}

	switch from {
	case FulfillmentStatusAllocated,
		FulfillmentStatusReceived:
		return to ==
			FulfillmentStatusPicking

	case FulfillmentStatusPicking:
		return to ==
			FulfillmentStatusPacked

	case FulfillmentStatusPacked:
		return to ==
			FulfillmentStatusReadyForHandoff

	default:
		return false
	}
}

func validInboundStatus(
	status string,
) bool {
	switch status {
	case InboundStatusCreated,
		InboundStatusSupplierReady,
		InboundStatusPickedUpInChina,
		InboundStatusDepartedChina,
		InboundStatusInInternationalTransit,
		InboundStatusArrivedBangladesh,
		InboundStatusCustomsProcessing,
		InboundStatusCustomsReleased,
		InboundStatusReceivedAtWarehouse,
		InboundStatusCancelled:
		return true

	default:
		return false
	}
}

func allowedInboundTransition(
	from string,
	to string,
) bool {
	if to ==
		InboundStatusCancelled {
		return from !=
			InboundStatusReceivedAtWarehouse &&
			from !=
				InboundStatusCancelled
	}

	switch from {
	case InboundStatusCreated:
		return to ==
			InboundStatusSupplierReady

	case InboundStatusSupplierReady:
		return to ==
			InboundStatusPickedUpInChina

	case InboundStatusPickedUpInChina:
		return to ==
			InboundStatusDepartedChina

	case InboundStatusDepartedChina:
		return to ==
			InboundStatusInInternationalTransit

	case InboundStatusInInternationalTransit:
		return to ==
			InboundStatusArrivedBangladesh

	case InboundStatusArrivedBangladesh:
		return to ==
			InboundStatusCustomsProcessing

	case InboundStatusCustomsProcessing:
		return to ==
			InboundStatusCustomsReleased

	case InboundStatusCustomsReleased:
		return to ==
			InboundStatusReceivedAtWarehouse

	default:
		return false
	}
}
