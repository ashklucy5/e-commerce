package delivery

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

const (
	JourneyStageOrderConfirmed       = "order_confirmed"
	JourneyStageInChina              = "in_china"
	JourneyStagePackedInChina        = "packed_in_china"
	JourneyStageInternationalTransit = "international_transit"
	JourneyStageArrivedBangladesh    = "arrived_bangladesh"
	JourneyStageWarehouseReceived    = "warehouse_received"
	JourneyStageDelivering           = "delivering"
	JourneyStageDelivered            = "delivered"
	JourneyStageCancelled            = "cancelled"
)

type JourneyLocation struct {
	CountryCode string `json:"country_code,omitempty"`
	City        string `json:"city,omitempty"`

	LocationName string `json:"location_name,omitempty"`

	Latitude  *float64 `json:"latitude,omitempty"`
	Longitude *float64 `json:"longitude,omitempty"`
}

type CustomerJourneyEvent struct {
	ID string `json:"id"`

	Kind string `json:"kind"`

	Source string `json:"source"`

	EventCode string `json:"event_code"`

	Status string `json:"status,omitempty"`

	Stage string `json:"stage"`

	Message string `json:"message"`

	Location *JourneyLocation `json:"location,omitempty"`

	OccurredAt time.Time `json:"occurred_at"`
}

type CustomerTrackingMilestone struct {
	Stage string `json:"stage"`
	Label string `json:"label"`

	Completed bool `json:"completed"`
	Current   bool `json:"current"`

	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

type CustomerShipmentSummary struct {
	ID string `json:"id"`

	DeliveryMode string `json:"delivery_mode"`

	ProviderCode string `json:"provider_code,omitempty"`

	CourierName string `json:"courier_name,omitempty"`

	TrackingNumber string `json:"tracking_number,omitempty"`
	TrackingURL    string `json:"tracking_url,omitempty"`

	Status string `json:"status"`

	ShippedAt *time.Time `json:"shipped_at,omitempty"`

	ProviderDeliveredAt *time.Time `json:"provider_delivered_at,omitempty"`

	AwaitingConfirmationAt *time.Time `json:"awaiting_confirmation_at,omitempty"`

	DeliveredAt *time.Time `json:"delivered_at,omitempty"`
}

type CustomerTrackingDetail struct {
	OrderID string `json:"order_id"`

	CurrentStage string `json:"current_stage"`

	CurrentLocation *JourneyLocation `json:"current_location,omitempty"`

	Shipment *CustomerShipmentSummary `json:"shipment,omitempty"`

	Milestones []CustomerTrackingMilestone `json:"milestones"`

	Events []CustomerJourneyEvent `json:"events"`
}

type journeyFulfillmentState struct {
	Source string
	Status string

	CountryCode string
	City        string

	WarehouseName string

	Latitude  *float64
	Longitude *float64

	ReceivedAt *time.Time
	PickingAt  *time.Time
	PackedAt   *time.Time

	ReadyForHandoffAt *time.Time
	HandedOffAt       *time.Time

	UpdatedAt time.Time
}

func (s *Service) GetJourneyTrackingForViewer(
	ctx context.Context,
	customerID string,
	checkoutKey string,
	orderID string,
) (CustomerTrackingDetail, error) {
	customerID =
		strings.TrimSpace(
			customerID,
		)

	checkoutKey =
		normalizeGuestCheckoutKey(
			checkoutKey,
		)

	orderID =
		strings.TrimSpace(
			orderID,
		)

	if !uuidPattern.MatchString(
		orderID,
	) {
		return CustomerTrackingDetail{},
			ErrOrderNotFound
	}

	access, err :=
		s.repository.GetTrackingAccess(
			ctx,
			orderID,
		)
	if err != nil {
		return CustomerTrackingDetail{},
			err
	}

	if access.CustomerID != "" {
		if customerID == "" ||
			!uuidPattern.MatchString(
				customerID,
			) ||
			access.CustomerID !=
				customerID {

			return CustomerTrackingDetail{},
				ErrCustomerOrderMismatch
		}
	} else {
		if checkoutKey == "" ||
			!constantTimeEqual(
				checkoutKey,
				access.CheckoutKey,
			) {

			return CustomerTrackingDetail{},
				ErrCustomerOrderMismatch
		}
	}

	return s.GetCustomerJourneyTracking(
		ctx,
		orderID,
	)
}

func (s *Service) GetCustomerJourneyTracking(
	ctx context.Context,
	orderID string,
) (CustomerTrackingDetail, error) {
	orderID =
		strings.TrimSpace(
			orderID,
		)

	if !uuidPattern.MatchString(
		orderID,
	) {
		return CustomerTrackingDetail{},
			ErrOrderNotFound
	}

	order, err :=
		s.repository.GetOrderState(
			ctx,
			orderID,
		)
	if err != nil {
		return CustomerTrackingDetail{},
			err
	}

	inboundEvents, err :=
		s.repository.ListCustomerInboundJourneyEvents(
			ctx,
			orderID,
		)
	if err != nil {
		return CustomerTrackingDetail{},
			err
	}

	deliveryEvents, err :=
		s.repository.ListCustomerDeliveryJourneyEvents(
			ctx,
			orderID,
		)
	if err != nil {
		return CustomerTrackingDetail{},
			err
	}

	fulfillments, err :=
		s.repository.ListJourneyFulfillments(
			ctx,
			orderID,
		)
	if err != nil {
		return CustomerTrackingDetail{},
			err
	}

	var shipment *Shipment

	shipmentValue, err :=
		s.repository.GetShipmentByOrder(
			ctx,
			orderID,
		)

	switch {
	case err == nil:
		shipment =
			&shipmentValue

	case errors.Is(
		err,
		ErrShipmentNotFound,
	):
		// Tracking must work while the item is still in China.

	default:
		return CustomerTrackingDetail{},
			err
	}

	events :=
		make(
			[]CustomerJourneyEvent,
			0,
			len(inboundEvents)+
				len(deliveryEvents),
		)

	events =
		append(
			events,
			inboundEvents...,
		)

	events =
		append(
			events,
			deliveryEvents...,
		)

	sort.SliceStable(
		events,
		func(
			left int,
			right int,
		) bool {
			if events[left].OccurredAt.Equal(
				events[right].OccurredAt,
			) {
				return events[left].ID <
					events[right].ID
			}

			return events[left].
				OccurredAt.
				Before(
					events[right].
						OccurredAt,
				)
		},
	)

	maxRank :=
		0

	for _, event := range events {

		rank :=
			journeyStageRank(
				event.Stage,
			)

		if rank > maxRank {
			maxRank =
				rank
		}
	}

	for _, fulfillment := range fulfillments {

		rank :=
			journeyFulfillmentRank(
				fulfillment,
			)

		if rank > maxRank {
			maxRank =
				rank
		}
	}

	if shipment != nil {
		rank :=
			journeyShipmentRank(
				*shipment,
			)

		if rank > maxRank {
			maxRank =
				rank
		}
	}

	if order.Status == "delivered" &&
		maxRank < 7 {

		maxRank = 7
	}

	cancelled :=
		order.Status ==
			"cancelled"

	if shipment != nil &&
		shipment.Status ==
			ShipmentStatusCancelled {

		cancelled = true
	}

	currentStage :=
		journeyStageForRank(
			maxRank,
		)

	if cancelled {
		currentStage =
			JourneyStageCancelled
	}

	currentLocation :=
		resolveCurrentJourneyLocation(
			maxRank,
			events,
			fulfillments,
		)

	milestones :=
		buildCustomerJourneyMilestones(
			maxRank,
			currentStage,
			events,
			fulfillments,
			shipment,
		)

	result :=
		CustomerTrackingDetail{
			OrderID: orderID,

			CurrentStage: currentStage,

			CurrentLocation: currentLocation,

			Milestones: milestones,

			Events: events,
		}

	if shipment != nil {
		result.Shipment =
			customerShipmentSummary(
				*shipment,
			)
	}

	return result,
		nil
}

func (r *Repository) ListCustomerInboundJourneyEvents(
	ctx context.Context,
	orderID string,
) ([]CustomerJourneyEvent, error) {
	rows, err :=
		r.db.Query(
			ctx,
			`
				SELECT
					e.id::text,

					e.status,

					COALESCE(
						e.public_message,
						''
					),

					COALESCE(
						e.country_code,
						''
					),

					COALESCE(
						e.city,
						''
					),

					COALESCE(
						e.location_name,
						''
					),

					e.latitude,
					e.longitude,

					e.occurred_at

				FROM inbound_shipment_events e

				WHERE
					e.customer_visible = true

					AND EXISTS (
						SELECT 1

						FROM warehouse_fulfillments wf

						WHERE
							wf.order_id = $1::uuid

							AND wf.inbound_shipment_id =
								e.inbound_shipment_id
					)

				ORDER BY
					e.occurred_at,
					e.created_at,
					e.id
			`,
			orderID,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"list customer inbound journey events: %w",
				err,
			)
	}

	defer rows.Close()

	items :=
		make(
			[]CustomerJourneyEvent,
			0,
		)

	for rows.Next() {
		var item CustomerJourneyEvent

		var publicMessage string

		var countryCode string
		var city string
		var locationName string

		var latitude pgtype.Float8
		var longitude pgtype.Float8

		if err :=
			rows.Scan(
				&item.ID,
				&item.Status,
				&publicMessage,
				&countryCode,
				&city,
				&locationName,
				&latitude,
				&longitude,
				&item.OccurredAt,
			); err != nil {

			return nil,
				fmt.Errorf(
					"scan customer inbound journey event: %w",
					err,
				)
		}

		item.Kind =
			"inbound"

		item.Source =
			"china_inbound"

		item.EventCode =
			item.Status

		item.Stage =
			journeyStageForInboundStatus(
				item.Status,
			)

		item.Message =
			publicMessage

		if item.Message == "" {
			item.Message =
				customerJourneyMessageForInboundStatus(
					item.Status,
				)
		}

		item.Location =
			newJourneyLocation(
				countryCode,
				city,
				locationName,
				latitude,
				longitude,
			)

		items =
			append(
				items,
				item,
			)
	}

	if err :=
		rows.Err(); err != nil {

		return nil,
			fmt.Errorf(
				"iterate customer inbound journey events: %w",
				err,
			)
	}

	return items,
		nil
}

func (r *Repository) ListCustomerDeliveryJourneyEvents(
	ctx context.Context,
	orderID string,
) ([]CustomerJourneyEvent, error) {
	rows, err :=
		r.db.Query(
			ctx,
			`
				SELECT
					id::text,

					source,
					event_code,

					COALESCE(
						status,
						''
					),

					COALESCE(
						public_message,
						''
					),

					COALESCE(
						country_code,
						''
					),

					COALESCE(
						city,
						''
					),

					COALESCE(
						location_name,
						''
					),

					latitude,
					longitude,

					occurred_at

				FROM delivery_tracking_events

				WHERE
					order_id = $1::uuid

					AND customer_visible = true

				ORDER BY
					occurred_at,
					created_at,
					id
			`,
			orderID,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"list customer delivery journey events: %w",
				err,
			)
	}

	defer rows.Close()

	items :=
		make(
			[]CustomerJourneyEvent,
			0,
		)

	for rows.Next() {
		var item CustomerJourneyEvent

		var publicMessage string

		var countryCode string
		var city string
		var locationName string

		var latitude pgtype.Float8
		var longitude pgtype.Float8

		if err :=
			rows.Scan(
				&item.ID,
				&item.Source,
				&item.EventCode,
				&item.Status,
				&publicMessage,
				&countryCode,
				&city,
				&locationName,
				&latitude,
				&longitude,
				&item.OccurredAt,
			); err != nil {

			return nil,
				fmt.Errorf(
					"scan customer delivery journey event: %w",
					err,
				)
		}

		item.Kind =
			"delivery"

		item.Stage =
			journeyStageForDeliveryEvent(
				item.EventCode,
				item.Status,
			)

		item.Message =
			publicMessage

		if item.Message == "" {
			item.Message =
				customerJourneyMessageForDeliveryEvent(
					item.EventCode,
					item.Status,
				)
		}

		item.Location =
			newJourneyLocation(
				countryCode,
				city,
				locationName,
				latitude,
				longitude,
			)

		items =
			append(
				items,
				item,
			)
	}

	if err :=
		rows.Err(); err != nil {

		return nil,
			fmt.Errorf(
				"iterate customer delivery journey events: %w",
				err,
			)
	}

	return items,
		nil
}

func (r *Repository) ListJourneyFulfillments(
	ctx context.Context,
	orderID string,
) ([]journeyFulfillmentState, error) {
	rows, err :=
		r.db.Query(
			ctx,
			`
				SELECT
					wf.source,
					wf.status,

					w.country_code,

					COALESCE(
						w.city,
						''
					),

					w.name,

					w.latitude,
					w.longitude,

					wf.received_at,
					wf.picking_at,
					wf.packed_at,
					wf.ready_for_handoff_at,
					wf.handed_off_at,

					wf.updated_at

				FROM warehouse_fulfillments wf

				JOIN warehouses w
					ON w.id =
						wf.warehouse_id

				WHERE
					wf.order_id =
						$1::uuid

				ORDER BY
					wf.updated_at DESC,
					wf.id
			`,
			orderID,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"list delivery journey fulfillments: %w",
				err,
			)
	}

	defer rows.Close()

	items :=
		make(
			[]journeyFulfillmentState,
			0,
		)

	for rows.Next() {
		var item journeyFulfillmentState

		var latitude pgtype.Float8
		var longitude pgtype.Float8

		var receivedAt pgtype.Timestamptz
		var pickingAt pgtype.Timestamptz
		var packedAt pgtype.Timestamptz

		var readyAt pgtype.Timestamptz
		var handedOffAt pgtype.Timestamptz

		if err :=
			rows.Scan(
				&item.Source,
				&item.Status,
				&item.CountryCode,
				&item.City,
				&item.WarehouseName,
				&latitude,
				&longitude,
				&receivedAt,
				&pickingAt,
				&packedAt,
				&readyAt,
				&handedOffAt,
				&item.UpdatedAt,
			); err != nil {

			return nil,
				fmt.Errorf(
					"scan delivery journey fulfillment: %w",
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

		if receivedAt.Valid {
			value :=
				receivedAt.Time.UTC()

			item.ReceivedAt =
				&value
		}

		if pickingAt.Valid {
			value :=
				pickingAt.Time.UTC()

			item.PickingAt =
				&value
		}

		if packedAt.Valid {
			value :=
				packedAt.Time.UTC()

			item.PackedAt =
				&value
		}

		if readyAt.Valid {
			value :=
				readyAt.Time.UTC()

			item.ReadyForHandoffAt =
				&value
		}

		if handedOffAt.Valid {
			value :=
				handedOffAt.Time.UTC()

			item.HandedOffAt =
				&value
		}

		items =
			append(
				items,
				item,
			)
	}

	if err :=
		rows.Err(); err != nil {

		return nil,
			fmt.Errorf(
				"iterate delivery journey fulfillments: %w",
				err,
			)
	}

	return items,
		nil
}

func newJourneyLocation(
	countryCode string,
	city string,
	locationName string,
	latitude pgtype.Float8,
	longitude pgtype.Float8,
) *JourneyLocation {
	location :=
		JourneyLocation{
			CountryCode: strings.TrimSpace(
				countryCode,
			),

			City: strings.TrimSpace(
				city,
			),

			LocationName: strings.TrimSpace(
				locationName,
			),
		}

	if latitude.Valid {
		value :=
			latitude.Float64

		location.Latitude =
			&value
	}

	if longitude.Valid {
		value :=
			longitude.Float64

		location.Longitude =
			&value
	}

	if location.CountryCode == "" &&
		location.City == "" &&
		location.LocationName == "" &&
		location.Latitude == nil &&
		location.Longitude == nil {

		return nil
	}

	return &location
}

func journeyStageForInboundStatus(
	status string,
) string {
	switch status {
	case "created",
		"supplier_ready",
		"picked_up_in_china":

		return JourneyStageInChina

	case "packed_in_china":
		return JourneyStagePackedInChina

	case "departed_china",
		"in_international_transit":

		return JourneyStageInternationalTransit

	case "arrived_bangladesh",
		"customs_processing",
		"customs_released":

		return JourneyStageArrivedBangladesh

	case "received_at_warehouse":
		return JourneyStageWarehouseReceived

	case "cancelled":
		return JourneyStageCancelled

	default:
		return JourneyStageOrderConfirmed
	}
}

func journeyStageForDeliveryEvent(
	eventCode string,
	status string,
) string {
	switch eventCode {
	case EventShipmentPrepared:
		return JourneyStageWarehouseReceived

	case EventShipmentDispatched:
		return JourneyStageDelivering

	case EventProviderDelivered:
		/*
			Provider-delivered is deliberately NOT final delivered.

			Your existing delivery model requires receipt
			confirmation before final delivery.
		*/
		return JourneyStageDelivering

	case EventReceiptConfirmed:
		return JourneyStageDelivered
	}

	switch status {
	case ShipmentStatusPending:
		return JourneyStageWarehouseReceived

	case ShipmentStatusShipped,
		ShipmentStatusAwaitingConfirmation:

		return JourneyStageDelivering

	case ShipmentStatusDelivered:
		return JourneyStageDelivered

	case ShipmentStatusCancelled:
		return JourneyStageCancelled

	default:
		return JourneyStageWarehouseReceived
	}
}

func journeyStageRank(
	stage string,
) int {
	switch stage {
	case JourneyStageInChina:
		return 1

	case JourneyStagePackedInChina:
		return 2

	case JourneyStageInternationalTransit:
		return 3

	case JourneyStageArrivedBangladesh:
		return 4

	case JourneyStageWarehouseReceived:
		return 5

	case JourneyStageDelivering:
		return 6

	case JourneyStageDelivered:
		return 7

	default:
		return 0
	}
}

func journeyStageForRank(
	rank int,
) string {
	switch {
	case rank >= 7:
		return JourneyStageDelivered

	case rank == 6:
		return JourneyStageDelivering

	case rank == 5:
		return JourneyStageWarehouseReceived

	case rank == 4:
		return JourneyStageArrivedBangladesh

	case rank == 3:
		return JourneyStageInternationalTransit

	case rank == 2:
		return JourneyStagePackedInChina

	case rank == 1:
		return JourneyStageInChina

	default:
		return JourneyStageOrderConfirmed
	}
}

func journeyFulfillmentRank(
	item journeyFulfillmentState,
) int {
	if item.Status == "cancelled" {
		return 0
	}

	if item.Source ==
		"china_inbound" &&
		item.Status ==
			"waiting_inbound" {

		return 1
	}

	switch item.Status {
	case "allocated",
		"received",
		"picking",
		"packed",
		"ready_for_handoff",
		"handed_off":

		return 5

	default:
		return 0
	}
}

func journeyShipmentRank(
	shipment Shipment,
) int {
	switch shipment.Status {
	case ShipmentStatusPending:
		return 5

	case ShipmentStatusShipped,
		ShipmentStatusAwaitingConfirmation:

		return 6

	case ShipmentStatusDelivered:
		return 7

	default:
		return 0
	}
}

func resolveCurrentJourneyLocation(
	maxRank int,
	events []CustomerJourneyEvent,
	fulfillments []journeyFulfillmentState,
) *JourneyLocation {
	if maxRank >= 6 {
		if location :=
			latestJourneyLocation(
				events,
				"delivery",
			); location != nil {

			return location
		}

		if location :=
			warehouseJourneyLocation(
				fulfillments,
			); location != nil {

			return location
		}
	}

	if maxRank >= 5 {
		if location :=
			warehouseJourneyLocation(
				fulfillments,
			); location != nil {

			return location
		}
	}

	if location :=
		latestJourneyLocation(
			events,
			"inbound",
		); location != nil {

		return location
	}

	return latestJourneyLocation(
		events,
		"",
	)
}

func latestJourneyLocation(
	events []CustomerJourneyEvent,
	kind string,
) *JourneyLocation {
	for index :=
		len(events) - 1; index >= 0; index-- {

		event :=
			events[index]

		if kind != "" &&
			event.Kind != kind {

			continue
		}

		if event.Location != nil {
			location :=
				*event.Location

			return &location
		}
	}

	return nil
}

func warehouseJourneyLocation(
	fulfillments []journeyFulfillmentState,
) *JourneyLocation {
	for _, item := range fulfillments {

		if item.CountryCode == "" &&
			item.City == "" &&
			item.WarehouseName == "" &&
			item.Latitude == nil &&
			item.Longitude == nil {

			continue
		}

		return &JourneyLocation{
			CountryCode: item.CountryCode,

			City: item.City,

			LocationName: item.WarehouseName,

			Latitude: item.Latitude,

			Longitude: item.Longitude,
		}
	}

	return nil
}

func buildCustomerJourneyMilestones(
	maxRank int,
	currentStage string,
	events []CustomerJourneyEvent,
	fulfillments []journeyFulfillmentState,
	shipment *Shipment,
) []CustomerTrackingMilestone {
	definitions :=
		[]struct {
			Stage string
			Label string
			Rank  int
		}{
			{
				Stage: JourneyStageInChina,
				Label: "In China",
				Rank:  1,
			},
			{
				Stage: JourneyStagePackedInChina,
				Label: "Packed in China",
				Rank:  2,
			},
			{
				Stage: JourneyStageInternationalTransit,
				Label: "Shipped to Bangladesh",
				Rank:  3,
			},
			{
				Stage: JourneyStageArrivedBangladesh,
				Label: "Arrived in Bangladesh",
				Rank:  4,
			},
			{
				Stage: JourneyStageWarehouseReceived,
				Label: "At Bangladesh warehouse",
				Rank:  5,
			},
			{
				Stage: JourneyStageDelivering,
				Label: "Out for delivery",
				Rank:  6,
			},
			{
				Stage: JourneyStageDelivered,
				Label: "Delivered",
				Rank:  7,
			},
		}

	result :=
		make(
			[]CustomerTrackingMilestone,
			0,
			len(definitions),
		)

	for _, definition := range definitions {

		milestone :=
			CustomerTrackingMilestone{
				Stage: definition.Stage,

				Label: definition.Label,

				Completed: maxRank >=
					definition.Rank,

				Current: currentStage ==
					definition.Stage,
			}

		milestone.CompletedAt =
			journeyMilestoneCompletedAt(
				definition.Rank,
				events,
				fulfillments,
				shipment,
			)

		result =
			append(
				result,
				milestone,
			)
	}

	return result
}

func journeyMilestoneCompletedAt(
	rank int,
	events []CustomerJourneyEvent,
	fulfillments []journeyFulfillmentState,
	shipment *Shipment,
) *time.Time {
	var earliest *time.Time

	for _, event := range events {

		if journeyStageRank(
			event.Stage,
		) < rank {

			continue
		}

		value :=
			event.OccurredAt.UTC()

		if earliest == nil ||
			value.Before(
				*earliest,
			) {

			copy :=
				value

			earliest =
				&copy
		}
	}

	if rank == 5 {
		for _, fulfillment := range fulfillments {

			values :=
				[]*time.Time{
					fulfillment.ReceivedAt,
					fulfillment.PickingAt,
					fulfillment.PackedAt,
					fulfillment.ReadyForHandoffAt,
					fulfillment.HandedOffAt,
				}

			for _, candidate := range values {

				earliest =
					earliestJourneyTime(
						earliest,
						candidate,
					)
			}
		}
	}

	if shipment != nil {
		switch rank {
		case 6:
			earliest =
				earliestJourneyTime(
					earliest,
					shipment.ShippedAt,
				)

		case 7:
			earliest =
				earliestJourneyTime(
					earliest,
					shipment.DeliveredAt,
				)
		}
	}

	return earliest
}

func earliestJourneyTime(
	current *time.Time,
	candidate *time.Time,
) *time.Time {
	if candidate == nil {
		return current
	}

	value :=
		candidate.UTC()

	if current == nil ||
		value.Before(
			*current,
		) {

		copy :=
			value

		return &copy
	}

	return current
}

func customerShipmentSummary(
	shipment Shipment,
) *CustomerShipmentSummary {
	return &CustomerShipmentSummary{
		ID: shipment.ID,

		DeliveryMode: shipment.DeliveryMode,

		ProviderCode: shipment.ProviderCode,

		CourierName: shipment.CourierName,

		TrackingNumber: shipment.TrackingNumber,

		TrackingURL: shipment.TrackingURL,

		Status: shipment.Status,

		ShippedAt: shipment.ShippedAt,

		ProviderDeliveredAt: shipment.ProviderDeliveredAt,

		AwaitingConfirmationAt: shipment.AwaitingConfirmationAt,

		DeliveredAt: shipment.DeliveredAt,
	}
}

func customerJourneyMessageForInboundStatus(
	status string,
) string {
	switch status {
	case "created":
		return "Your item is being prepared in China"

	case "supplier_ready":
		return "Your item is ready at the supplier in China"

	case "packed_in_china":
		return "Your item has been packed in China"

	case "picked_up_in_china":
		return "Your item has been picked up in China"

	case "departed_china":
		return "Your shipment has left China"

	case "in_international_transit":
		return "Your shipment is on the way to Bangladesh"

	case "arrived_bangladesh":
		return "Your shipment has arrived in Bangladesh"

	case "customs_processing":
		return "Your shipment is being processed by customs"

	case "customs_released":
		return "Your shipment has cleared customs"

	case "received_at_warehouse":
		return "Your shipment has reached our Bangladesh warehouse"

	case "cancelled":
		return "The inbound shipment was cancelled"

	default:
		return "Shipment status updated"
	}
}

func customerJourneyMessageForDeliveryEvent(
	eventCode string,
	status string,
) string {
	switch eventCode {
	case EventShipmentPrepared:
		return "Your order is being prepared for delivery"

	case EventShipmentDispatched:
		return "Your order is out for delivery"

	case EventProviderDelivered:
		return "The courier has reported delivery; receipt confirmation is pending"

	case EventReceiptConfirmed:
		return "Your order has been delivered"
	}

	switch status {
	case ShipmentStatusPending:
		return "Your order is being prepared for delivery"

	case ShipmentStatusShipped:
		return "Your order is out for delivery"

	case ShipmentStatusAwaitingConfirmation:
		return "Delivery has been reported and is awaiting confirmation"

	case ShipmentStatusDelivered:
		return "Your order has been delivered"

	case ShipmentStatusCancelled:
		return "Delivery was cancelled"

	default:
		return "Delivery status updated"
	}
}
