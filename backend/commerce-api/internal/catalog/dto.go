package catalog

// CategorySummary is the small amount of category information
// needed on a product detail page.
type CategorySummary struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// ProductCard is the lightweight representation used by
// product grids, category pages, search results, and homepage sections.
type ProductCard struct {
	ID               string  `json:"id"`
	ProductCode      string  `json:"product_code"`
	Name             string  `json:"name"`
	Slug             string  `json:"slug"`
	Brand            *string `json:"brand,omitempty"`
	ShortDescription *string `json:"short_description,omitempty"`
	IsFeatured       bool    `json:"is_featured"`

	PrimaryImageURL *string `json:"primary_image_url,omitempty"`

	PriceAmount int64  `json:"price_amount"`
	Currency    string `json:"currency"`

	InStock bool `json:"in_stock"`
}

// ProductDetail is the complete customer-facing product response.
type ProductDetail struct {
	Product

	Category CategorySummary  `json:"category"`
	Images   []ProductImage   `json:"images"`
	Variants []ProductVariant `json:"variants"`

	ImmersiveMedia *ProductImmersiveMedia `json:"immersive_media,omitempty"`
}
