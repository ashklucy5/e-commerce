package crm

import (
	"encoding/json"
	"time"
)

type Case struct {
	ID         string `json:"id"`
	CaseNumber string `json:"case_number"`
	CustomerID string `json:"customer_id"`

	CaseType string `json:"case_type"`
	Subject  string `json:"subject"`
	Status   string `json:"status"`
	Priority string `json:"priority"`

	ProductID string `json:"product_id,omitempty"`
	VariantID string `json:"variant_id,omitempty"`
	OrderID   string `json:"order_id,omitempty"`

	RequestedQuantity         *int `json:"requested_quantity,omitempty"`
	AvailableQuantitySnapshot *int `json:"available_quantity_snapshot,omitempty"`

	ContextSnapshot json.RawMessage `json:"context_snapshot"`

	LastMessageAt         time.Time  `json:"last_message_at"`
	LastCustomerMessageAt *time.Time `json:"last_customer_message_at,omitempty"`
	LastSupportMessageAt  *time.Time `json:"last_support_message_at,omitempty"`

	ResolvedAt *time.Time `json:"resolved_at,omitempty"`
	ClosedAt   *time.Time `json:"closed_at,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Message struct {
	ID     string `json:"id"`
	CaseID string `json:"case_id"`

	AuthorType string `json:"author_type"`
	Visibility string `json:"visibility"`
	Body       string `json:"body"`

	SupportActorCode string `json:"support_actor_code,omitempty"`
	SupportActorType string `json:"support_actor_type,omitempty"`
	SupportActorName string `json:"support_actor_name,omitempty"`

	CreatedAt   time.Time       `json:"created_at"`
	Attachments json.RawMessage `json:"attachments,omitempty"`
}

type ProductContext struct {
	ProductID   string `json:"product_id"`
	ProductCode string `json:"product_code"`
	ProductName string `json:"product_name"`
	Status      string `json:"status"`

	VariantID string `json:"variant_id,omitempty"`
	SKU       string `json:"sku,omitempty"`

	MinimumOrderQuantity int    `json:"minimum_order_quantity,omitempty"`
	PriceAmount          int64  `json:"price_amount,omitempty"`
	Currency             string `json:"currency,omitempty"`

	QuantityOnHand    int `json:"quantity_on_hand,omitempty"`
	QuantityReserved  int `json:"quantity_reserved,omitempty"`
	AvailableQuantity int `json:"available_quantity,omitempty"`
}

type OrderContext struct {
	OrderID       string `json:"order_id"`
	OrderNumber   string `json:"order_number"`
	Status        string `json:"status"`
	PaymentStatus string `json:"payment_status"`
	PaymentMethod string `json:"payment_method"`
	TotalAmount   int64  `json:"total_amount"`
	Currency      string `json:"currency"`
}

type ContextSnapshot struct {
	Product *ProductContext `json:"product,omitempty"`
	Order   *OrderContext   `json:"order,omitempty"`
}
