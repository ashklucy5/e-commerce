package catalogimport

type RowSource struct {
	Sheet string `json:"sheet"`
	Row   int    `json:"row"`
}

type ValidationError struct {
	Sheet   string `json:"sheet"`
	Row     int    `json:"row"`
	Field   string `json:"field,omitempty"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type Workbook struct {
	Products   []ProductRow   `json:"products"`
	Variants   []VariantRow   `json:"variants"`
	Images     []ImageRow     `json:"images"`
	PriceTiers []PriceTierRow `json:"price_tiers"`
	Categories []CategoryRow  `json:"categories"`
}

type ParseResult struct {
	Workbook Workbook          `json:"workbook"`
	Errors   []ValidationError `json:"errors"`
}

func (r ParseResult) Valid() bool {
	return len(r.Errors) == 0
}

type ProductRow struct {
	Source           RowSource `json:"source"`
	ProductCode      string    `json:"product_code"`
	ProductName      string    `json:"product_name"`
	CategoryPath     string    `json:"category_path"`
	Slug             string    `json:"slug,omitempty"`
	Brand            string    `json:"brand,omitempty"`
	ShortDescription string    `json:"short_description,omitempty"`
	Description      string    `json:"description,omitempty"`
	Status           string    `json:"status"`
	IsFeatured       bool      `json:"is_featured"`
}

type VariantRow struct {
	Source               RowSource `json:"source"`
	ProductCode          string    `json:"product_code"`
	SKU                  string    `json:"sku"`
	ColorName            string    `json:"color_name,omitempty"`
	ColorHex             string    `json:"color_hex,omitempty"`
	Size                 string    `json:"size,omitempty"`
	PriceAmount          int64     `json:"price_amount"`
	CompareAtPriceAmount *int64    `json:"compare_at_price_amount,omitempty"`
	CostAmount           *int64    `json:"cost_amount,omitempty"`
	Currency             string    `json:"currency"`
	MinimumOrderQuantity int       `json:"minimum_order_quantity"`
	OrderIncrement       int       `json:"order_increment"`
	Stock                int       `json:"stock"`
	ReorderLevel         int       `json:"reorder_level"`
	Barcode              string    `json:"barcode,omitempty"`
	WeightGrams          *int      `json:"weight_grams,omitempty"`
	IsActive             bool      `json:"is_active"`
}

type ImageRow struct {
	Source      RowSource `json:"source"`
	ProductCode string    `json:"product_code"`
	SKU         string    `json:"sku,omitempty"`
	ImageURL    string    `json:"image_url,omitempty"`
	ImageFile   string    `json:"image_file,omitempty"`
	AltText     string    `json:"alt_text,omitempty"`
	SortOrder   int       `json:"sort_order"`
	IsPrimary   bool      `json:"is_primary"`
}

type PriceTierRow struct {
	Source          RowSource `json:"source"`
	SKU             string    `json:"sku"`
	MinQuantity     int       `json:"min_quantity"`
	UnitPriceAmount int64     `json:"unit_price_amount"`
}

type CategoryRow struct {
	Source       RowSource `json:"source"`
	CategoryPath string    `json:"category_path"`
	Description  string    `json:"description,omitempty"`
	ImageURL     string    `json:"image_url,omitempty"`
	IconURL      string    `json:"icon_url,omitempty"`
	SortOrder    int       `json:"sort_order"`
	IsActive     bool      `json:"is_active"`
}
