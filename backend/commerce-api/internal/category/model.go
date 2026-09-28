package category

import "time"

// Category represents one row from the categories table.
//
// ParentID is nil for a top-level category.
// For example:
//
//	Men
//	├── Shirts
//	└── Pants
//
// "Men" has ParentID = nil.
// "Shirts" can have ParentID = Men's ID.
type Category struct {
	ID          string    `json:"id"`
	ParentID    *string   `json:"parent_id,omitempty"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description *string   `json:"description,omitempty"`
	ImageURL    *string   `json:"image_url,omitempty"`
	IconURL     *string   `json:"icon_url,omitempty"`
	SortOrder   int       `json:"sort_order"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
