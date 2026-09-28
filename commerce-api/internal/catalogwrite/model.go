package catalogwrite

type ProductInput struct {
	ProductCode      string
	Name             string
	CategoryID       string
	CategoryPath     string
	Slug             string
	Brand            string
	ShortDescription string
	Description      string
	Status           string
	IsFeatured       bool
	Variants         []VariantInput
}

type VariantInput struct {
	SKU string

	ColorName string
	ColorHex  string
	Size      string

	MinimumOrderQuantity int
	OrderIncrement       int

	PriceAmount          int64
	CompareAtPriceAmount *int64
	CostAmount           *int64
	Currency             string

	Barcode     string
	WeightGrams *int
	IsActive    bool

	Stock        int
	ReorderLevel int

	PriceTiers []PriceTierInput
}

type PriceTierInput struct {
	MinQuantity     int
	UnitPriceAmount int64
}

type ProductResult struct {
	ID          string          `json:"id"`
	ProductCode string          `json:"product_code"`
	Slug        string          `json:"slug"`
	Variants    []VariantResult `json:"variants"`
}

type VariantResult struct {
	ID  string `json:"id"`
	SKU string `json:"sku"`
}
