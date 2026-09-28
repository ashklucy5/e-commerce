package category

// CategoryNode is the public hierarchical representation
// of a category.
type CategoryNode struct {
	ID          string         `json:"id"`
	ParentID    *string        `json:"parent_id,omitempty"`
	Name        string         `json:"name"`
	Slug        string         `json:"slug"`
	Description *string        `json:"description,omitempty"`
	ImageURL    *string        `json:"image_url,omitempty"`
	IconURL     *string        `json:"icon_url,omitempty"`
	SortOrder   int            `json:"sort_order"`
	Children    []CategoryNode `json:"children"`
}
