package catalog

// ProductVariant represents one sellable SKU.
//
// CostAmount is intentionally not exposed because it is
// internal business information.
type ProductVariant struct {
	ID                   string      `json:"id"`
	SKU                  string      `json:"sku"`
	ColorName            *string     `json:"color_name,omitempty"`
	ColorHex             *string     `json:"color_hex,omitempty"`
	Size                 *string     `json:"size,omitempty"`
	MinimumOrderQuantity int         `json:"minimum_order_quantity"`
	OrderIncrement       int         `json:"order_increment"`
	PriceAmount          int64       `json:"price_amount"`
	CompareAtPriceAmount *int64      `json:"compare_at_price_amount,omitempty"`
	Currency             string      `json:"currency"`
	WeightGrams          *int        `json:"weight_grams,omitempty"`
	AvailableQuantity    int         `json:"available_quantity"`
	InStock              bool        `json:"in_stock"`
	PriceTiers           []PriceTier `json:"price_tiers"`
}
