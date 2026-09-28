package warehouse

import "time"

const (
	WarehouseStatusActive   = "active"
	WarehouseStatusInactive = "inactive"

	FulfillmentSourceBangladeshStock = "bangladesh_stock"
	FulfillmentSourceChinaInbound    = "china_inbound"

	FulfillmentStatusAllocated       = "allocated"
	FulfillmentStatusWaitingInbound  = "waiting_inbound"
	FulfillmentStatusReceived        = "received"
	FulfillmentStatusPicking         = "picking"
	FulfillmentStatusPacked          = "packed"
	FulfillmentStatusReadyForHandoff = "ready_for_handoff"
	FulfillmentStatusHandedOff       = "handed_off"
	FulfillmentStatusCancelled       = "cancelled"

	InboundStatusCreated                = "created"
	InboundStatusSupplierReady          = "supplier_ready"
	InboundStatusPickedUpInChina        = "picked_up_in_china"
	InboundStatusDepartedChina          = "departed_china"
	InboundStatusInInternationalTransit = "in_international_transit"
	InboundStatusArrivedBangladesh      = "arrived_bangladesh"
	InboundStatusCustomsProcessing      = "customs_processing"
	InboundStatusCustomsReleased        = "customs_released"
	InboundStatusReceivedAtWarehouse    = "received_at_warehouse"
	InboundStatusCancelled              = "cancelled"
)

type Warehouse struct {
	ID               string    `json:"id"`
	Code             string    `json:"code"`
	Name             string    `json:"name"`
	CountryCode      string    `json:"country_code"`
	City             string    `json:"city,omitempty"`
	AddressLine1     string    `json:"address_line1,omitempty"`
	Status           string    `json:"status"`
	IsDefault        bool      `json:"is_default"`
	AllowsSelfPickup bool      `json:"allows_self_pickup"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type InboundShipment struct {
	ID                     string     `json:"id"`
	ReferenceCode          string     `json:"reference_code"`
	OriginCountry          string     `json:"origin_country"`
	DestinationWarehouseID string     `json:"destination_warehouse_id"`
	DestinationWarehouse   string     `json:"destination_warehouse,omitempty"`
	CarrierName            string     `json:"carrier_name,omitempty"`
	ExternalReference      string     `json:"external_reference,omitempty"`
	TrackingNumber         string     `json:"tracking_number,omitempty"`
	Status                 string     `json:"status"`
	ETA                    *time.Time `json:"eta,omitempty"`
	DepartedAt             *time.Time `json:"departed_at,omitempty"`
	ArrivedBangladeshAt    *time.Time `json:"arrived_bangladesh_at,omitempty"`
	CustomsReleasedAt      *time.Time `json:"customs_released_at,omitempty"`
	ReceivedAt             *time.Time `json:"received_at,omitempty"`
	Notes                  string     `json:"notes,omitempty"`
	CreatedAt              time.Time  `json:"created_at"`
	UpdatedAt              time.Time  `json:"updated_at"`
}

type InboundShipmentEvent struct {
	ID                string    `json:"id"`
	InboundShipmentID string    `json:"inbound_shipment_id"`
	Status            string    `json:"status"`
	Message           string    `json:"message,omitempty"`
	ActorType         string    `json:"actor_type,omitempty"`
	ActorID           string    `json:"actor_id,omitempty"`
	OccurredAt        time.Time `json:"occurred_at"`
	CreatedAt         time.Time `json:"created_at"`
}

type InboundShipmentDetail struct {
	Shipment InboundShipment        `json:"shipment"`
	Events   []InboundShipmentEvent `json:"events"`
}

type Fulfillment struct {
	ID                string     `json:"id"`
	OrderID           string     `json:"order_id"`
	OrderItemID       string     `json:"order_item_id"`
	OrderItemSKU      string     `json:"order_item_sku,omitempty"`
	ProductName       string     `json:"product_name,omitempty"`
	WarehouseID       string     `json:"warehouse_id"`
	WarehouseCode     string     `json:"warehouse_code,omitempty"`
	WarehouseName     string     `json:"warehouse_name,omitempty"`
	InboundShipmentID string     `json:"inbound_shipment_id,omitempty"`
	InboundReference  string     `json:"inbound_reference,omitempty"`
	Source            string     `json:"source"`
	Status            string     `json:"status"`
	Quantity          int        `json:"quantity"`
	AllocatedAt       time.Time  `json:"allocated_at"`
	ReceivedAt        *time.Time `json:"received_at,omitempty"`
	PickingAt         *time.Time `json:"picking_at,omitempty"`
	PackedAt          *time.Time `json:"packed_at,omitempty"`
	ReadyForHandoffAt *time.Time `json:"ready_for_handoff_at,omitempty"`
	HandedOffAt       *time.Time `json:"handed_off_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

type FulfillmentEvent struct {
	ID            string    `json:"id"`
	FulfillmentID string    `json:"fulfillment_id"`
	EventType     string    `json:"event_type"`
	FromStatus    string    `json:"from_status,omitempty"`
	ToStatus      string    `json:"to_status,omitempty"`
	Message       string    `json:"message,omitempty"`
	ActorType     string    `json:"actor_type,omitempty"`
	ActorID       string    `json:"actor_id,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}
