package delivery

import "time"

type PrepareShipmentRequest struct {
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

type DispatchShipmentRequest struct {
	Message string `json:"message"`
}

type TrackingEventRequest struct {
	Source    string `json:"source"`
	EventCode string `json:"event_code"`
	Status    string `json:"status"`
	Message   string `json:"message"`

	Latitude  *float64 `json:"latitude"`
	Longitude *float64 `json:"longitude"`

	ExternalEventID string         `json:"external_event_id"`
	Metadata        map[string]any `json:"metadata"`
	OccurredAt      *time.Time     `json:"occurred_at"`
}

type ProviderDeliveredRequest struct {
	ProviderStatus  string     `json:"provider_status"`
	Message         string     `json:"message"`
	ExternalEventID string     `json:"external_event_id"`
	OccurredAt      *time.Time `json:"occurred_at"`
}

type ConfirmReceiptRequest struct {
	Note string `json:"note"`
}
