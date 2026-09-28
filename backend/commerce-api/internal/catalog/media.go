package catalog

// ProductImage represents an image shown on the storefront.
type ProductImage struct {
	ID        string  `json:"id"`
	VariantID *string `json:"variant_id,omitempty"`
	URL       string  `json:"url"`
	AltText   *string `json:"alt_text,omitempty"`
	SortOrder int     `json:"sort_order"`
	IsPrimary bool    `json:"is_primary"`
}
