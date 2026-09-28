package delivery

import "time"

const (
	DeliveryModeCourier        = "courier"
	DeliveryModeSelfPickup     = "self_pickup"
	DeliveryModeCommunityRider = "community_rider"
)

const (
	ShipmentStatusPending              = "pending"
	ShipmentStatusShipped              = "shipped"
	ShipmentStatusAwaitingConfirmation = "awaiting_confirmation"
	ShipmentStatusDelivered            = "delivered"
	ShipmentStatusCancelled            = "cancelled"
)

const (
	ConfirmationSourceCustomer = "customer"
	ConfirmationSourceSupport  = "support"
	ConfirmationSourceAdmin    = "admin"
)

type Shipment struct {
	ID      string `json:"id"`
	OrderID string `json:"order_id"`

	OriginWarehouseID string `json:"origin_warehouse_id,omitempty"`
	DeliveryMode      string `json:"delivery_mode"`

	ProviderCode       string `json:"provider_code,omitempty"`
	ProviderShipmentID string `json:"provider_shipment_id,omitempty"`
	ProviderStatus     string `json:"provider_status,omitempty"`

	CourierName      string `json:"courier_name,omitempty"`
	CourierReference string `json:"courier_reference,omitempty"`
	RiderReference   string `json:"rider_reference,omitempty"`
	TrackingNumber   string `json:"tracking_number,omitempty"`
	TrackingURL      string `json:"tracking_url,omitempty"`

	Status string `json:"status"`

	ShippedAt              *time.Time `json:"shipped_at,omitempty"`
	ProviderDeliveredAt    *time.Time `json:"provider_delivered_at,omitempty"`
	AwaitingConfirmationAt *time.Time `json:"awaiting_confirmation_at,omitempty"`
	ConfirmedReceivedAt    *time.Time `json:"confirmed_received_at,omitempty"`
	DeliveredAt            *time.Time `json:"delivered_at,omitempty"`
	LastProviderSyncAt     *time.Time `json:"last_provider_sync_at,omitempty"`

	ConfirmationSource string `json:"confirmation_source,omitempty"`
	ConfirmedByActorID string `json:"confirmed_by_actor_id,omitempty"`
	ConfirmationNote   string `json:"confirmation_note,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type TrackingEvent struct {
	ID         string `json:"id"`
	ShipmentID string `json:"shipment_id"`
	OrderID    string `json:"order_id"`

	Source    string `json:"source"`
	EventCode string `json:"event_code"`
	Status    string `json:"status,omitempty"`
	Message   string `json:"message,omitempty"`

	Latitude  *float64 `json:"latitude,omitempty"`
	Longitude *float64 `json:"longitude,omitempty"`

	ExternalEventID string         `json:"external_event_id,omitempty"`
	Metadata        map[string]any `json:"metadata,omitempty"`

	OccurredAt time.Time `json:"occurred_at"`
	CreatedAt  time.Time `json:"created_at"`
}

type TrackingDetail struct {
	Shipment Shipment        `json:"shipment"`
	Events   []TrackingEvent `json:"events"`
}

type orderState struct {
	ID            string
	CustomerID    string
	Status        string
	PaymentStatus string
	PaymentMethod string
}
