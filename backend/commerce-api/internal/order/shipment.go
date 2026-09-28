package order

import "time"

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
