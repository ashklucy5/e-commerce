package order

import "time"

type Item struct {
	ID          string `json:"id"`
	OrderID     string `json:"order_id"`
	VariantID   string `json:"variant_id"`
	SKU         string `json:"sku"`
	ProductName string `json:"product_name"`

	Quantity             int `json:"quantity"`
	MinimumOrderQuantity int `json:"minimum_order_quantity"`

	UnitPriceAmount int64  `json:"unit_price_amount"`
	LineTotalAmount int64  `json:"line_total_amount"`
	Currency        string `json:"currency"`

	CreatedAt time.Time `json:"created_at"`
}
