package search

type CategorySummary struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type ProductResult struct {
	ID               string          `json:"id"`
	ProductCode      string          `json:"product_code"`
	Name             string          `json:"name"`
	Slug             string          `json:"slug"`
	Brand            *string         `json:"brand,omitempty"`
	ShortDescription *string         `json:"short_description,omitempty"`
	IsFeatured       bool            `json:"is_featured"`
	PrimaryImageURL  *string         `json:"primary_image_url,omitempty"`
	PriceAmount      int64           `json:"price_amount"`
	Currency         string          `json:"currency"`
	InStock          bool            `json:"in_stock"`
	MatchedSKU       *string         `json:"matched_sku,omitempty"`
	Category         CategorySummary `json:"category"`
	MatchType        string          `json:"match_type"`

	SearchTier int `json:"-"`
}

type Input struct {
	Query string
	Page  int
	Limit int
}

type ImageInput struct {
	MediaType string
	Data      []byte
	Page      int
	Limit     int
}

type Meta struct {
	Mode           string `json:"mode"`
	Query          string `json:"query"`
	Page           int    `json:"page"`
	Limit          int    `json:"limit"`
	Total          int64  `json:"total"`
	TotalPages     int    `json:"total_pages"`
	PrimaryTotal   int64  `json:"primary_total"`
	SimilarTotal   int64  `json:"similar_total"`
	NoMatch        bool   `json:"no_match"`
	ShowingSimilar bool   `json:"showing_similar"`
}

type Result struct {
	Items []ProductResult `json:"data"`
	Meta  Meta            `json:"meta"`
}

type SearchPage struct {
	Items        []ProductResult
	Total        int64
	PrimaryTotal int64
	SimilarTotal int64
}

type HistoryCategorySignal struct {
	CategoryID      string
	RelevanceWeight int
}
