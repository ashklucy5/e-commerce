package delivery

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type JourneyTrackingEventRequest struct {
	Source string `json:"source"`

	EventCode string `json:"event_code"`

	Status string `json:"status"`

	// Internal/admin message. Never returned by the
	// customer journey endpoint.
	Message string `json:"message"`

	PublicMessage string `json:"public_message"`

	CountryCode string `json:"country_code"`

	City string `json:"city"`

	LocationName string `json:"location_name"`

	Latitude  *float64 `json:"latitude"`
	Longitude *float64 `json:"longitude"`

	CustomerVisible *bool `json:"customer_visible"`

	ExternalEventID string `json:"external_event_id"`

	Metadata map[string]any `json:"metadata"`

	OccurredAt *time.Time `json:"occurred_at"`
}

func (h *AdminHandler) AddJourneyTrackingEvent(
	c *gin.Context,
) {
	var request JourneyTrackingEventRequest

	if err :=
		c.ShouldBindJSON(
			&request,
		); err != nil {

		writeDeliveryError(
			c,
			ErrInvalidInput,
		)

		return
	}

	result, err :=
		h.service.AddJourneyTrackingEvent(
			c.Request.Context(),
			c.Param(
				"shipment_id",
			),
			request,
		)
	if err != nil {
		writeDeliveryError(
			c,
			err,
		)

		return
	}

	c.JSON(
		http.StatusCreated,
		gin.H{
			"data": result,
		},
	)
}

func (s *Service) AddJourneyTrackingEvent(
	ctx context.Context,
	shipmentID string,
	request JourneyTrackingEventRequest,
) (TrackingDetail, error) {
	shipmentID =
		strings.TrimSpace(
			shipmentID,
		)

	normalizeJourneyTrackingEventRequest(
		&request,
	)

	if !uuidPattern.MatchString(
		shipmentID,
	) ||
		!validJourneyTrackingEventRequest(
			request,
		) {

		return TrackingDetail{},
			ErrInvalidInput
	}

	tx, err :=
		s.repository.Begin(
			ctx,
		)
	if err != nil {
		return TrackingDetail{},
			err
	}

	defer func() {
		_ = tx.Rollback(
			ctx,
		)
	}()

	shipment, err :=
		s.repository.LockShipmentTx(
			ctx,
			tx,
			shipmentID,
		)
	if err != nil {
		return TrackingDetail{},
			err
	}

	if shipment.Status ==
		ShipmentStatusCancelled {

		return TrackingDetail{},
			ErrShipmentNotReady
	}

	if request.PublicMessage == "" {
		request.PublicMessage =
			customerJourneyMessageForDeliveryEvent(
				request.EventCode,
				request.Status,
			)
	}

	if err :=
		s.repository.InsertJourneyTrackingEventTx(
			ctx,
			tx,
			shipment,
			request,
			time.Now().
				UTC(),
		); err != nil {

		return TrackingDetail{},
			err
	}

	if err :=
		tx.Commit(
			ctx,
		); err != nil {

		return TrackingDetail{},
			fmt.Errorf(
				"commit journey tracking event: %w",
				err,
			)
	}

	return s.GetTracking(
		ctx,
		shipment.OrderID,
	)
}

func (r *Repository) InsertJourneyTrackingEventTx(
	ctx context.Context,
	tx pgx.Tx,
	shipment Shipment,
	request JourneyTrackingEventRequest,
	now time.Time,
) error {
	occurredAt :=
		now

	if request.OccurredAt != nil &&
		!request.OccurredAt.IsZero() {

		occurredAt =
			request.OccurredAt.
				UTC()
	}

	customerVisible :=
		true

	if request.CustomerVisible != nil {
		customerVisible =
			*request.CustomerVisible
	}

	metadataJSON :=
		""

	if len(
		request.Metadata,
	) > 0 {

		encoded, err :=
			json.Marshal(
				request.Metadata,
			)
		if err != nil {
			return fmt.Errorf(
				"encode journey tracking metadata: %w",
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
					public_message,

					country_code,
					city,
					location_name,

					latitude,
					longitude,

					customer_visible,

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
					NULLIF($7, ''),

					NULLIF($8, ''),
					NULLIF($9, ''),
					NULLIF($10, ''),

					$11,
					$12,

					$13,

					NULLIF($14, ''),
					NULLIF($15, '')::jsonb,

					$16::timestamptz,
					now()
				)

				ON CONFLICT (
					shipment_id,
					external_event_id
				)

				WHERE
					external_event_id IS NOT NULL

				DO NOTHING
			`,
			shipment.ID,
			shipment.OrderID,
			request.Source,
			request.EventCode,
			request.Status,
			request.Message,
			request.PublicMessage,
			request.CountryCode,
			request.City,
			request.LocationName,
			request.Latitude,
			request.Longitude,
			customerVisible,
			request.ExternalEventID,
			metadataJSON,
			occurredAt,
		)
	if err != nil {
		return fmt.Errorf(
			"insert delivery journey tracking event: %w",
			err,
		)
	}

	return nil
}

func normalizeJourneyTrackingEventRequest(
	request *JourneyTrackingEventRequest,
) {
	request.Source =
		strings.ToLower(
			strings.TrimSpace(
				request.Source,
			),
		)

	if request.Source == "" {
		request.Source =
			"admin"
	}

	request.EventCode =
		strings.ToLower(
			strings.TrimSpace(
				request.EventCode,
			),
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

	request.PublicMessage =
		strings.TrimSpace(
			request.PublicMessage,
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

	request.LocationName =
		strings.TrimSpace(
			request.LocationName,
		)

	request.ExternalEventID =
		strings.TrimSpace(
			request.ExternalEventID,
		)
}

func validJourneyTrackingEventRequest(
	request JourneyTrackingEventRequest,
) bool {
	if !validTrackingSource(
		request.Source,
	) ||
		request.EventCode == "" {

		return false
	}

	if utf8.RuneCountInString(
		request.EventCode,
	) > 80 ||
		utf8.RuneCountInString(
			request.Status,
		) > 80 ||
		utf8.RuneCountInString(
			request.Message,
		) > 1000 ||
		utf8.RuneCountInString(
			request.PublicMessage,
		) > 500 ||
		utf8.RuneCountInString(
			request.City,
		) > 120 ||
		utf8.RuneCountInString(
			request.LocationName,
		) > 160 ||
		utf8.RuneCountInString(
			request.ExternalEventID,
		) > 160 {

		return false
	}

	if request.CountryCode != "" &&
		(len(request.CountryCode) != 2 ||
			request.CountryCode !=
				strings.ToUpper(
					request.CountryCode,
				)) {

		return false
	}

	return validDeliveryJourneyCoordinatePair(
		request.Latitude,
		request.Longitude,
	)
}

func validDeliveryJourneyCoordinatePair(
	latitude *float64,
	longitude *float64,
) bool {
	if (latitude == nil) !=
		(longitude == nil) {

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
