package cart

import "time"

type Item struct {
	ID        string `json:"id"`
	VariantID string `json:"variant_id"`
	ProductID string `json:"product_id"`

	SKU         string `json:"sku"`
	ProductName string `json:"product_name"`
	ProductSlug string `json:"product_slug"`

	ColorName string `json:"color_name,omitempty"`
	Size      string `json:"size,omitempty"`
	ImageURL  string `json:"image_url,omitempty"`

	Quantity             int `json:"quantity"`
	MinimumOrderQuantity int `json:"minimum_order_quantity"`
	OrderIncrement       int `json:"order_increment"`
	AvailableQuantity    int `json:"available_quantity"`

	UnitPriceAmount int64  `json:"unit_price_amount"`
	LineTotalAmount int64  `json:"line_total_amount"`
	Currency        string `json:"currency"`

	IsAvailable bool `json:"is_available"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	variantActive bool
	productActive bool
}

type purchasableVariant struct {
	ID                   string
	MinimumOrderQuantity int
	OrderIncrement       int
	AvailableQuantity    int
	Currency             string
}
