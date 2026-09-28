package checkout

type StartRequest struct {
	CartKey       string  `json:"cart_key"`
	PromotionCode *string `json:"promotion_code"`
}

type UpdateRequest struct {
	CustomerName  *string `json:"customer_name"`
	CustomerPhone *string `json:"customer_phone"`
	CustomerEmail *string `json:"customer_email"`

	ShippingAddressLine1 *string `json:"shipping_address_line1"`
	ShippingAddressLine2 *string `json:"shipping_address_line2"`
	ShippingCity         *string `json:"shipping_city"`
	ShippingArea         *string `json:"shipping_area"`
	ShippingPostalCode   *string `json:"shipping_postal_code"`

	DeliveryMethod *string `json:"delivery_method"`
	PaymentMethod  *string `json:"payment_method"`

	PromotionCode *string `json:"promotion_code"`
}
