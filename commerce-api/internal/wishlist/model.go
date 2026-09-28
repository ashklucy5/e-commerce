package wishlist

import "time"

type Item struct {
	ID        string    `json:"id"`
	ProductID string    `json:"product_id"`
	AddedAt   time.Time `json:"added_at"`

	ProductCode string  `json:"product_code"`
	Name        string  `json:"name"`
	Slug        string  `json:"slug"`
	Brand       *string `json:"brand,omitempty"`

	PrimaryImageURL *string `json:"primary_image_url,omitempty"`

	PriceAmount          *int64 `json:"price_amount,omitempty"`
	CompareAtPriceAmount *int64 `json:"compare_at_price_amount,omitempty"`
	Currency             string `json:"currency,omitempty"`

	InStock     bool `json:"in_stock"`
	IsAvailable bool `json:"is_available"`
}

type State struct {
	ProductID  string `json:"product_id"`
	Wishlisted bool   `json:"wishlisted"`
}

type Count struct {
	Total int64 `json:"total"`
}
