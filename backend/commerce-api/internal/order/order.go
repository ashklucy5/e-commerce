package order

import "time"

type Order struct {
	ID          string `json:"id"`
	OrderNumber string `json:"order_number"`
	OrderType   string `json:"order_type"`

	CheckoutID string `json:"checkout_id,omitempty"`
	CartID     string `json:"cart_id,omitempty"`
	CustomerID string `json:"-"`

	Status        string `json:"status"`
	PaymentStatus string `json:"payment_status"`
	PaymentMethod string `json:"payment_method"`

	Currency       string `json:"currency"`
	SubtotalAmount int64  `json:"subtotal_amount"`
	DiscountAmount int64  `json:"discount_amount"`

	PromotionID   string `json:"promotion_id,omitempty"`
	PromotionCode string `json:"promotion_code,omitempty"`

	ShippingAmount int64 `json:"shipping_amount"`
	TotalAmount    int64 `json:"total_amount"`

	CustomerName  string `json:"customer_name"`
	CustomerPhone string `json:"customer_phone"`
	CustomerEmail string `json:"customer_email,omitempty"`

	ShippingAddressLine1 string `json:"shipping_address_line1"`
	ShippingAddressLine2 string `json:"shipping_address_line2,omitempty"`
	ShippingCity         string `json:"shipping_city"`
	ShippingArea         string `json:"shipping_area"`
	ShippingPostalCode   string `json:"shipping_postal_code,omitempty"`

	DeliveryMethod string `json:"delivery_method"`

	PaymentDueAt *time.Time `json:"payment_due_at,omitempty"`
	PaidAt       *time.Time `json:"paid_at,omitempty"`
	ConfirmedAt  *time.Time `json:"confirmed_at,omitempty"`

	ProcessingAt *time.Time `json:"processing_at,omitempty"`
	ShippedAt    *time.Time `json:"shipped_at,omitempty"`
	DeliveredAt  *time.Time `json:"delivered_at,omitempty"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`

	CancelledAt *time.Time `json:"cancelled_at,omitempty"`

	CancellationReason string `json:"cancellation_reason,omitempty"`
	CancelledBy        string `json:"cancelled_by,omitempty"`

	Shipment *Shipment `json:"shipment,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	ItemCount     int `json:"item_count"`
	QuantityTotal int `json:"quantity_total"`

	Items []Item `json:"items"`
}
