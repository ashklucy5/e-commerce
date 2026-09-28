package warehouse

import "time"

type CreateWarehouseRequest struct {
	Code             string `json:"code"`
	Name             string `json:"name"`
	CountryCode      string `json:"country_code"`
	City             string `json:"city"`
	AddressLine1     string `json:"address_line1"`
	Status           string `json:"status"`
	IsDefault        bool   `json:"is_default"`
	AllowsSelfPickup bool   `json:"allows_self_pickup"`
}

type CreateFulfillmentRequest struct {
	OrderItemID       string `json:"order_item_id"`
	WarehouseID       string `json:"warehouse_id"`
	InboundShipmentID string `json:"inbound_shipment_id"`
	Source            string `json:"source"`
	Quantity          int    `json:"quantity"`
}

type UpdateFulfillmentStatusRequest struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

type CreateInboundShipmentRequest struct {
	ReferenceCode          string     `json:"reference_code"`
	OriginCountry          string     `json:"origin_country"`
	DestinationWarehouseID string     `json:"destination_warehouse_id"`
	CarrierName            string     `json:"carrier_name"`
	ExternalReference      string     `json:"external_reference"`
	TrackingNumber         string     `json:"tracking_number"`
	ETA                    *time.Time `json:"eta"`
	Notes                  string     `json:"notes"`
}

type UpdateInboundShipmentStatusRequest struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}
