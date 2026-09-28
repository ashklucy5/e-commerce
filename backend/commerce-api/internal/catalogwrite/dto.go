package catalogwrite

type CreateProductRequest struct {
	ProductCode      string                 `json:"product_code"`
	Name             string                 `json:"name"`
	CategoryID       string                 `json:"category_id"`
	CategoryPath     string                 `json:"category_path"`
	Slug             string                 `json:"slug"`
	Brand            string                 `json:"brand"`
	ShortDescription string                 `json:"short_description"`
	Description      string                 `json:"description"`
	Status           string                 `json:"status"`
	IsFeatured       bool                   `json:"is_featured"`
	Variants         []CreateVariantRequest `json:"variants"`
}

type CreateVariantRequest struct {
	SKU string `json:"sku"`

	ColorName string `json:"color_name"`
	ColorHex  string `json:"color_hex"`
	Size      string `json:"size"`

	MinimumOrderQuantity int `json:"minimum_order_quantity"`
	OrderIncrement       int `json:"order_increment"`

	PriceAmount          int64  `json:"price_amount"`
	CompareAtPriceAmount *int64 `json:"compare_at_price_amount"`
	CostAmount           *int64 `json:"cost_amount"`
	Currency             string `json:"currency"`

	Barcode     string `json:"barcode"`
	WeightGrams *int   `json:"weight_grams"`
	IsActive    *bool  `json:"is_active"`

	Stock        int `json:"stock"`
	ReorderLevel int `json:"reorder_level"`

	PriceTiers []CreatePriceTierRequest `json:"price_tiers"`
}

type CreatePriceTierRequest struct {
	MinQuantity     int   `json:"min_quantity"`
	UnitPriceAmount int64 `json:"unit_price_amount"`
}

func (r CreateProductRequest) ToInput() ProductInput {
	input := ProductInput{
		ProductCode:      r.ProductCode,
		Name:             r.Name,
		CategoryID:       r.CategoryID,
		CategoryPath:     r.CategoryPath,
		Slug:             r.Slug,
		Brand:            r.Brand,
		ShortDescription: r.ShortDescription,
		Description:      r.Description,
		Status:           r.Status,
		IsFeatured:       r.IsFeatured,
		Variants: make(
			[]VariantInput,
			0,
			len(r.Variants),
		),
	}

	for _, variant := range r.Variants {
		isActive := true

		if variant.IsActive != nil {
			isActive = *variant.IsActive
		}

		variantInput := VariantInput{
			SKU:                  variant.SKU,
			ColorName:            variant.ColorName,
			ColorHex:             variant.ColorHex,
			Size:                 variant.Size,
			MinimumOrderQuantity: variant.MinimumOrderQuantity,
			OrderIncrement:       variant.OrderIncrement,
			PriceAmount:          variant.PriceAmount,
			CompareAtPriceAmount: variant.CompareAtPriceAmount,
			CostAmount:           variant.CostAmount,
			Currency:             variant.Currency,
			Barcode:              variant.Barcode,
			WeightGrams:          variant.WeightGrams,
			IsActive:             isActive,
			Stock:                variant.Stock,
			ReorderLevel:         variant.ReorderLevel,
			PriceTiers: make(
				[]PriceTierInput,
				0,
				len(variant.PriceTiers),
			),
		}

		for _, tier := range variant.PriceTiers {
			variantInput.PriceTiers = append(
				variantInput.PriceTiers,
				PriceTierInput{
					MinQuantity:     tier.MinQuantity,
					UnitPriceAmount: tier.UnitPriceAmount,
				},
			)
		}

		input.Variants = append(
			input.Variants,
			variantInput,
		)
	}

	return input
}
