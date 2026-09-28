package catalog

// PriceTier represents B2B quantity-based pricing.
//
// Example:
// 12+ units  -> 1490 BDT each
// 50+ units  -> 1390 BDT each
// 100+ units -> 1290 BDT each
type PriceTier struct {
	MinQuantity     int   `json:"min_quantity"`
	UnitPriceAmount int64 `json:"unit_price_amount"`
}
