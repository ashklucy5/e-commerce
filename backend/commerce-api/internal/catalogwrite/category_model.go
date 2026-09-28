package catalogwrite

type AdminCategory struct {
	ID                string          `json:"id"`
	ParentID          *string         `json:"parent_id,omitempty"`
	Name              string          `json:"name"`
	Slug              string          `json:"slug"`
	Description       *string         `json:"description,omitempty"`
	SortOrder         int             `json:"sort_order"`
	IsActive          bool            `json:"is_active"`
	ProductCodePrefix *string         `json:"product_code_prefix,omitempty"`
	ProductCodeReady  bool            `json:"product_code_ready"`
	Children          []AdminCategory `json:"children,omitempty"`
}

type CreateCategoryInput struct {
	Name              string
	ParentID          *string
	Slug              string
	Description       string
	SortOrder         int
	IsActive          bool
	ProductCodePrefix string
}
