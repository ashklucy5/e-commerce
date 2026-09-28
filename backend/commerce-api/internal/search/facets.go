package search

type CategoryFacet struct {
	ID string `json:"id"`

	Name string `json:"name"`

	Slug string `json:"slug"`

	Count int64 `json:"count"`
}

type BrandFacet struct {
	Name string `json:"name"`

	Count int64 `json:"count"`
}

type PriceFacet struct {
	MinAmount *int64 `json:"min_amount,omitempty"`

	MaxAmount *int64 `json:"max_amount,omitempty"`
}

type StockFacet struct {
	InStock int64 `json:"in_stock"`

	OutOfStock int64 `json:"out_of_stock"`
}

type Facets struct {
	Categories []CategoryFacet `json:"categories"`

	Brands []BrandFacet `json:"brands"`

	Price PriceFacet `json:"price"`

	Stock StockFacet `json:"stock"`
}
