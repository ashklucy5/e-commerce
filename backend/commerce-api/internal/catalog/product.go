package catalog

import "time"

// Product represents the core customer-visible product data.
type Product struct {
	ID               string     `json:"id"`
	CategoryID       string     `json:"category_id"`
	ProductCode      string     `json:"product_code"`
	Name             string     `json:"name"`
	Slug             string     `json:"slug"`
	Brand            *string    `json:"brand,omitempty"`
	ShortDescription *string    `json:"short_description,omitempty"`
	Description      *string    `json:"description,omitempty"`
	IsFeatured       bool       `json:"is_featured"`
	PublishedAt      *time.Time `json:"published_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}
