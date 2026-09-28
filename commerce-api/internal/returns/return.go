package returns

import "time"

const (
	StatusRequested = "requested"
	StatusApproved  = "approved"
	StatusRejected  = "rejected"
	StatusReceived  = "received"
	StatusInspected = "inspected"
	StatusCompleted = "completed"
	StatusCancelled = "cancelled"
)

const (
	InspectionPending        = "pending"
	InspectionRestockable    = "restockable"
	InspectionDamaged        = "damaged"
	InspectionNonRestockable = "non_restockable"
)

type Return struct {
	ID           string `json:"id"`
	ReturnNumber string `json:"return_number"`
	OrderID      string `json:"order_id"`

	Status string `json:"status"`

	CustomerNote string `json:"customer_note,omitempty"`

	RequestedBy string `json:"requested_by,omitempty"`
	ApprovedBy  string `json:"approved_by,omitempty"`
	RejectedBy  string `json:"rejected_by,omitempty"`

	RejectionReason string `json:"rejection_reason,omitempty"`

	RequestedAt time.Time  `json:"requested_at"`
	ApprovedAt  *time.Time `json:"approved_at,omitempty"`
	RejectedAt  *time.Time `json:"rejected_at,omitempty"`
	ReceivedAt  *time.Time `json:"received_at,omitempty"`
	InspectedAt *time.Time `json:"inspected_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	CancelledAt *time.Time `json:"cancelled_at,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Items []Item `json:"items"`
}

type Item struct {
	ID          string `json:"id"`
	ReturnID    string `json:"return_id"`
	OrderItemID string `json:"order_item_id"`
	VariantID   string `json:"variant_id"`

	SKU         string `json:"sku"`
	ProductName string `json:"product_name"`

	Quantity int `json:"quantity"`

	ReasonCode string `json:"reason_code"`
	ReasonNote string `json:"reason_note,omitempty"`

	ReceivedQuantity int `json:"received_quantity"`
	RestockQuantity  int `json:"restock_quantity"`

	InspectionStatus string `json:"inspection_status"`
	InspectionNote   string `json:"inspection_note,omitempty"`

	UnitPriceAmount int64  `json:"unit_price_amount"`
	Currency        string `json:"currency"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
