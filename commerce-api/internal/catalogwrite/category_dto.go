package catalogwrite

type CreateCategoryRequest struct {
	Name              string  `json:"name"`
	ParentID          *string `json:"parent_id"`
	Slug              string  `json:"slug"`
	Description       string  `json:"description"`
	SortOrder         int     `json:"sort_order"`
	IsActive          *bool   `json:"is_active"`
	ProductCodePrefix string  `json:"product_code_prefix"`
}

func (r CreateCategoryRequest) ToInput() CreateCategoryInput {
	isActive := true

	if r.IsActive != nil {
		isActive = *r.IsActive
	}

	return CreateCategoryInput{
		Name:              r.Name,
		ParentID:          r.ParentID,
		Slug:              r.Slug,
		Description:       r.Description,
		SortOrder:         r.SortOrder,
		IsActive:          isActive,
		ProductCodePrefix: r.ProductCodePrefix,
	}
}
