package inventory

import "time"

type StockItem struct {
	VariantID         string    `json:"variant_id"`
	SKU               string    `json:"sku"`
	ProductID         string    `json:"product_id"`
	ProductCode       string    `json:"product_code"`
	ProductName       string    `json:"product_name"`
	QuantityOnHand    int       `json:"quantity_on_hand"`
	QuantityReserved  int       `json:"quantity_reserved"`
	AvailableQuantity int       `json:"available_quantity"`
	ReorderLevel      int       `json:"reorder_level"`
	LowStock          bool      `json:"low_stock"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type StockListResult struct {
	Items  []StockItem `json:"items"`
	Limit  int         `json:"limit"`
	Offset int         `json:"offset"`
}

type Movement struct {
	ID                    string    `json:"id"`
	VariantID             string    `json:"variant_id"`
	ReservationID         string    `json:"reservation_id,omitempty"`
	MovementType          string    `json:"movement_type"`
	QuantityOnHandDelta   int       `json:"quantity_on_hand_delta"`
	QuantityReservedDelta int       `json:"quantity_reserved_delta"`
	QuantityOnHandAfter   int       `json:"quantity_on_hand_after"`
	QuantityReservedAfter int       `json:"quantity_reserved_after"`
	ReferenceType         string    `json:"reference_type,omitempty"`
	ReferenceID           string    `json:"reference_id,omitempty"`
	Reason                string    `json:"reason,omitempty"`
	Note                  string    `json:"note,omitempty"`
	ActorType             string    `json:"actor_type,omitempty"`
	ActorID               string    `json:"actor_id,omitempty"`
	CreatedAt             time.Time `json:"created_at"`
}

type MovementListResult struct {
	Items  []Movement `json:"items"`
	Limit  int        `json:"limit"`
	Offset int        `json:"offset"`
}

type Reservation struct {
	ID            string     `json:"id"`
	VariantID     string     `json:"variant_id"`
	ReferenceType string     `json:"reference_type"`
	ReferenceID   string     `json:"reference_id"`
	Quantity      int        `json:"quantity"`
	Status        string     `json:"status"`
	ExpiresAt     *time.Time `json:"expires_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}
