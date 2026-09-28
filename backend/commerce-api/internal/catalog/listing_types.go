package catalog

import "project.local/commerce-api/internal/platform/pagination"

const (
	ProductSortRecommended = "recommended"
	ProductSortFeatured    = "featured"
	ProductSortNewest      = "newest"
	ProductSortPriceAsc    = "price_asc"
	ProductSortPriceDesc   = "price_desc"
	ProductSortNameAsc     = "name_asc"
	ProductSortNameDesc    = "name_desc"
)

type ListProductsInput struct {
	Page  int
	Limit int

	CategorySlug string
	MinPrice     *int64
	MaxPrice     *int64
	InStock      *bool
	Sort         string
}

type ListProductsQuery struct {
	Pagination pagination.Params

	CategorySlug string
	MinPrice     *int64
	MaxPrice     *int64
	InStock      *bool
	Sort         string

	RankedProductIDs []string
}

type ProductListPage struct {
	Items []ProductCard
	Total int64
}

type RecommendationMeta struct {
	Strategy     string `json:"strategy"`
	Personalized bool   `json:"personalized"`
}

type ProductListMeta struct {
	pagination.Meta

	Sort string `json:"sort"`

	Recommendation *RecommendationMeta `json:"recommendation,omitempty"`
}

type ProductListResult struct {
	Items []ProductCard
	Meta  ProductListMeta
}
