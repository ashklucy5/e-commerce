package warehouse

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

const InboundStatusPackedInChina = "packed_in_china"

type UpdateInboundJourneyStatusRequest struct {
	Status string `json:"status"`

	// Internal/admin note. Never expose this directly to customers.
	Message string `json:"message"`

	// Safe customer-facing message.
	PublicMessage string `json:"public_message"`

	CountryCode  string `json:"country_code"`
	City         string `json:"city"`
	LocationName string `json:"location_name"`

	Latitude  *float64 `json:"latitude"`
	Longitude *float64 `json:"longitude"`

	CustomerVisible *bool      `json:"customer_visible"`
	OccurredAt      *time.Time `json:"occurred_at"`
}

type WarehouseMapLocationRequest struct {
	Latitude  *float64 `json:"latitude"`
	Longitude *float64 `json:"longitude"`
}

type WarehouseMapLocation struct {
	WarehouseID string `json:"warehouse_id"`

	Latitude  *float64 `json:"latitude,omitempty"`
	Longitude *float64 `json:"longitude,omitempty"`
}

func (s *Service) UpdateInboundShipmentJourneyStatus(
	ctx context.Context,
	id string,
	request UpdateInboundJourneyStatusRequest,
	actorID string,
) (InboundShipmentDetail, error) {
	id = strings.TrimSpace(id)
	actorID = normalizeActorID(actorID)

	normalizeInboundJourneyStatusRequest(&request)

	if !uuidPattern.MatchString(id) ||
		!validJourneyInboundStatus(request.Status) ||
		!validInboundJourneyRequest(request) {
		return InboundShipmentDetail{}, ErrInvalidInput
	}

	tx, err := s.repository.Begin(ctx)
	if err != nil {
		return InboundShipmentDetail{}, err
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	current, err := s.repository.LockInboundShipmentTx(
		ctx,
		tx,
		id,
	)
	if err != nil {
		return InboundShipmentDetail{}, err
	}

	statusChanged := current.Status != request.Status

	if statusChanged &&
		!allowedJourneyInboundTransition(
			current.Status,
			request.Status,
		) {
		return InboundShipmentDetail{},
			ErrInboundTransitionNotAllowed
	}

	now := time.Now().UTC()
	occurredAt := now

	if request.OccurredAt != nil &&
		!request.OccurredAt.IsZero() {
		occurredAt = request.OccurredAt.UTC()
	}

	if statusChanged {
		if err := s.repository.UpdateInboundStatusTx(
			ctx,
			tx,
			id,
			request.Status,
			occurredAt,
		); err != nil {
			return InboundShipmentDetail{}, err
		}
	}

	message := request.Message

	if message == "" {
		message = "Inbound shipment moved to " + request.Status
	}

	publicMessage := request.PublicMessage

	if publicMessage == "" {
		publicMessage =
			defaultInboundJourneyPublicMessage(
				request.Status,
			)
	}

	customerVisible := true

	if request.CustomerVisible != nil {
		customerVisible = *request.CustomerVisible
	}

	if err := s.repository.InsertInboundJourneyEventTx(
		ctx,
		tx,
		id,
		request.Status,
		message,
		publicMessage,
		request.CountryCode,
		request.City,
		request.LocationName,
		request.Latitude,
		request.Longitude,
		customerVisible,
		"admin",
		actorID,
		occurredAt,
	); err != nil {
		return InboundShipmentDetail{}, err
	}

	if statusChanged &&
		request.Status == InboundStatusReceivedAtWarehouse {
		if err := s.transitionInboundFulfillments(
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
			return InboundShipmentDetail{}, err
		}
	}

	if statusChanged &&
		request.Status == InboundStatusCancelled {
		if err := s.transitionInboundFulfillments(
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
			return InboundShipmentDetail{}, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return InboundShipmentDetail{},
			fmt.Errorf(
				"commit inbound shipment journey transition: %w",
				err,
			)
	}

	return s.GetInboundShipment(ctx, id)
}

func (s *Service) SetWarehouseMapLocation(
	ctx context.Context,
	warehouseID string,
	request WarehouseMapLocationRequest,
) (WarehouseMapLocation, error) {
	warehouseID = strings.TrimSpace(warehouseID)

	if !uuidPattern.MatchString(warehouseID) ||
		!validJourneyCoordinatePair(
			request.Latitude,
			request.Longitude,
		) {
		return WarehouseMapLocation{}, ErrInvalidInput
	}

	return s.repository.SetWarehouseMapLocation(
		ctx,
		warehouseID,
		request.Latitude,
		request.Longitude,
	)
}

func (r *Repository) InsertInboundJourneyEventTx(
	ctx context.Context,
	tx pgx.Tx,
	inboundID string,
	status string,
	message string,
	publicMessage string,
	countryCode string,
	city string,
	locationName string,
	latitude *float64,
	longitude *float64,
	customerVisible bool,
	actorType string,
	actorID string,
	occurredAt time.Time,
) error {
	_, err :=
		tx.Exec(
			ctx,
			`
				INSERT INTO inbound_shipment_events (
					inbound_shipment_id,
					status,
					message,
					public_message,
					country_code,
					city,
					location_name,
					latitude,
					longitude,
					customer_visible,
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
					NULLIF($6, ''),
					NULLIF($7, ''),
					$8,
					$9,
					$10,
					NULLIF($11, ''),
					NULLIF($12, ''),
					$13::timestamptz,
					now()
				)
			`,
			inboundID,
			status,
			message,
			publicMessage,
			countryCode,
			city,
			locationName,
			latitude,
			longitude,
			customerVisible,
			actorType,
			actorID,
			occurredAt,
		)
	if err != nil {
		return fmt.Errorf(
			"insert inbound journey event: %w",
			err,
		)
	}

	/*
		Only explicitly customer-visible milestone updates can
		produce external notifications.

		Because this runs inside the same pgx transaction as the
		inbound status/event update, either both commit or both
		roll back.
	*/
	if customerVisible {
		if err :=
			r.enqueueInboundJourneyNotificationTx(
				ctx,
				tx,
				inboundID,
				status,
				publicMessage,
			); err != nil {

			return err
		}
	}

	return nil
}

func (r *Repository) SetWarehouseMapLocation(
	ctx context.Context,
	warehouseID string,
	latitude *float64,
	longitude *float64,
) (WarehouseMapLocation, error) {
	var result WarehouseMapLocation
	var storedLatitude pgtype.Float8
	var storedLongitude pgtype.Float8

	err := r.db.QueryRow(
		ctx,
		`
			UPDATE warehouses
			SET
				latitude = $2,
				longitude = $3,
				updated_at = now()
			WHERE id = $1::uuid
			RETURNING
				id::text,
				latitude,
				longitude
		`,
		warehouseID,
		latitude,
		longitude,
	).Scan(
		&result.WarehouseID,
		&storedLatitude,
		&storedLongitude,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return WarehouseMapLocation{},
				ErrWarehouseNotFound
		}

		return WarehouseMapLocation{},
			fmt.Errorf(
				"set warehouse map location: %w",
				err,
			)
	}

	if storedLatitude.Valid {
		value := storedLatitude.Float64
		result.Latitude = &value
	}

	if storedLongitude.Valid {
		value := storedLongitude.Float64
		result.Longitude = &value
	}

	return result, nil
}

func normalizeInboundJourneyStatusRequest(
	request *UpdateInboundJourneyStatusRequest,
) {
	request.Status = strings.ToLower(
		strings.TrimSpace(request.Status),
	)

	request.Message =
		strings.TrimSpace(request.Message)

	request.PublicMessage =
		strings.TrimSpace(request.PublicMessage)

	request.CountryCode = strings.ToUpper(
		strings.TrimSpace(request.CountryCode),
	)

	request.City =
		strings.TrimSpace(request.City)

	request.LocationName =
		strings.TrimSpace(request.LocationName)
}

func validInboundJourneyRequest(
	request UpdateInboundJourneyStatusRequest,
) bool {
	if utf8.RuneCountInString(request.Message) > 1000 ||
		utf8.RuneCountInString(request.PublicMessage) > 500 ||
		utf8.RuneCountInString(request.City) > 120 ||
		utf8.RuneCountInString(request.LocationName) > 160 {
		return false
	}

	if request.CountryCode != "" &&
		(len(request.CountryCode) != 2 ||
			request.CountryCode !=
				strings.ToUpper(request.CountryCode)) {
		return false
	}

	return validJourneyCoordinatePair(
		request.Latitude,
		request.Longitude,
	)
}

func validJourneyCoordinatePair(
	latitude *float64,
	longitude *float64,
) bool {
	if (latitude == nil) != (longitude == nil) {
		return false
	}

	if latitude == nil {
		return true
	}

	return *latitude >= -90 &&
		*latitude <= 90 &&
		*longitude >= -180 &&
		*longitude <= 180
}

func validJourneyInboundStatus(
	status string,
) bool {
	switch status {
	case InboundStatusCreated,
		InboundStatusSupplierReady,
		InboundStatusPackedInChina,
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

func allowedJourneyInboundTransition(
	from string,
	to string,
) bool {
	if to == InboundStatusCancelled {
		return from !=
			InboundStatusReceivedAtWarehouse &&
			from != InboundStatusCancelled
	}

	switch from {
	case InboundStatusCreated:
		return to ==
			InboundStatusSupplierReady

	case InboundStatusSupplierReady:
		return to ==
			InboundStatusPackedInChina

	case InboundStatusPackedInChina:
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

func defaultInboundJourneyPublicMessage(
	status string,
) string {
	switch status {
	case InboundStatusCreated:
		return "Your item is being prepared in China"

	case InboundStatusSupplierReady:
		return "Your item is ready at the supplier in China"

	case InboundStatusPackedInChina:
		return "Your item has been packed in China"

	case InboundStatusPickedUpInChina:
		return "Your item has been picked up in China"

	case InboundStatusDepartedChina:
		return "Your shipment has left China"

	case InboundStatusInInternationalTransit:
		return "Your shipment is on the way to Bangladesh"

	case InboundStatusArrivedBangladesh:
		return "Your shipment has arrived in Bangladesh"

	case InboundStatusCustomsProcessing:
		return "Your shipment is being processed by customs"

	case InboundStatusCustomsReleased:
		return "Your shipment has cleared customs"

	case InboundStatusReceivedAtWarehouse:
		return "Your shipment has reached our Bangladesh warehouse"

	case InboundStatusCancelled:
		return "The inbound shipment was cancelled"

	default:
		return "Shipment status updated"
	}
}
