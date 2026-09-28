package checkout

import "time"

type Session struct {
	ID          string `json:"id"`
	CheckoutKey string `json:"checkout_key"`
	CartID      string `json:"cart_id"`
	Status      string `json:"status"`
	CustomerID  string `json:"-"`
	Currency    string `json:"currency"`

	SubtotalAmount int64 `json:"subtotal_amount"`
	DiscountAmount int64 `json:"discount_amount"`

	PromotionID   string `json:"promotion_id,omitempty"`
	PromotionCode string `json:"promotion_code,omitempty"`

	ShippingAmount int64 `json:"shipping_amount"`
	TotalAmount    int64 `json:"total_amount"`

	CustomerName  string `json:"customer_name,omitempty"`
	CustomerPhone string `json:"customer_phone,omitempty"`
	CustomerEmail string `json:"customer_email,omitempty"`

	ShippingAddressLine1 string `json:"shipping_address_line1,omitempty"`
	ShippingAddressLine2 string `json:"shipping_address_line2,omitempty"`
	ShippingCity         string `json:"shipping_city,omitempty"`
	ShippingArea         string `json:"shipping_area,omitempty"`
	ShippingPostalCode   string `json:"shipping_postal_code,omitempty"`

	DeliveryMethod string `json:"delivery_method,omitempty"`
	PaymentMethod  string `json:"payment_method,omitempty"`

	ExpiresAt   time.Time  `json:"expires_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	CancelledAt *time.Time `json:"cancelled_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`

	ItemCount     int `json:"item_count"`
	QuantityTotal int `json:"quantity_total"`

	Items []Item `json:"items"`
}

type Item struct {
	ID         string `json:"id"`
	CheckoutID string `json:"checkout_id"`
	VariantID  string `json:"variant_id"`

	SKU         string `json:"sku"`
	ProductName string `json:"product_name"`

	Quantity             int `json:"quantity"`
	MinimumOrderQuantity int `json:"minimum_order_quantity"`

	UnitPriceAmount int64 `json:"unit_price_amount"`
	LineTotalAmount int64 `json:"line_total_amount"`

	Currency string `json:"currency"`

	CreatedAt time.Time `json:"created_at"`
}

type cartSnapshot struct {
	ID        string
	CartKey   string
	Status    string
	Currency  string
	ExpiresAt *time.Time

	Items []cartSnapshotItem
}

type cartSnapshotItem struct {
	VariantID   string
	SKU         string
	ProductName string

	Quantity             int
	MinimumOrderQuantity int
	AvailableQuantity    int

	UnitPriceAmount int64
	Currency        string

	VariantActive bool
	ProductActive bool
}
