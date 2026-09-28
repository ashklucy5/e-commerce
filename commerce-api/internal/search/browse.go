package search

import (
	"context"
	"fmt"
	"strings"
)

type BrowseInput struct {
	Query string

	Page int

	Limit int

	Filters Filters
}

type BrowseMeta struct {
	Query string `json:"query"`

	Page int `json:"page"`

	Limit int `json:"limit"`

	Total int64 `json:"total"`

	TotalPages int `json:"total_pages"`

	Sort string `json:"sort"`

	NoMatch bool `json:"no_match"`
}

type BrowseResult struct {
	Items []ProductResult

	Meta BrowseMeta

	Facets Facets
}

type BrowsePage struct {
	Items []ProductResult

	Total int64

	Facets Facets
}

type browseStore interface {
	SearchBrowse(
		ctx context.Context,
		normalizedQuery string,
		page int,
		limit int,
		filters Filters,
	) (BrowsePage, error)
}

func (s *Service) Browse(
	ctx context.Context,
	customerID string,
	input BrowseInput,
) (BrowseResult, error) {
	normalized, err :=
		normalizeInput(
			Input{
				Query: input.Query,

				Page: input.Page,

				Limit: input.Limit,
			},
		)
	if err != nil {
		return BrowseResult{}, err
	}

	filters :=
		input.Filters

	filters.CategorySlug =
		strings.TrimSpace(
			filters.CategorySlug,
		)

	filters.Brand =
		strings.TrimSpace(
			filters.Brand,
		)

	filters.Sort, err =
		normalizeSearchSort(
			filters.Sort,
		)
	if err != nil {
		return BrowseResult{}, err
	}

	if filters.MinPrice != nil &&
		*filters.MinPrice < 0 {

		return BrowseResult{},
			ErrInvalidPriceFilter
	}

	if filters.MaxPrice != nil &&
		*filters.MaxPrice < 0 {

		return BrowseResult{},
			ErrInvalidPriceFilter
	}

	if filters.MinPrice != nil &&
		filters.MaxPrice != nil &&
		*filters.MaxPrice <
			*filters.MinPrice {

		return BrowseResult{},
			ErrInvalidPriceRange
	}

	store, ok :=
		s.store.(browseStore)
	if !ok {
		return BrowseResult{},
			fmt.Errorf(
				"filterable search repository is unavailable",
			)
	}

	pageResult, err :=
		store.SearchBrowse(
			ctx,
			normalized.NormalizedQuery,
			normalized.Page,
			normalized.Limit,
			filters,
		)
	if err != nil {
		return BrowseResult{}, err
	}

	if pageResult.Items == nil {
		pageResult.Items =
			make(
				[]ProductResult,
				0,
			)
	}

	if pageResult.Facets.Categories == nil {
		pageResult.Facets.Categories =
			make(
				[]CategoryFacet,
				0,
			)
	}

	if pageResult.Facets.Brands == nil {
		pageResult.Facets.Brands =
			make(
				[]BrandFacet,
				0,
			)
	}

	s.recordHistory(
		ctx,
		customerID,
		normalized.Page,
		normalized.DisplayQuery,
		normalized.NormalizedQuery,
		SearchPage{
			Items: pageResult.Items,

			Total: pageResult.Total,

			PrimaryTotal: pageResult.Total,
		},
	)

	return BrowseResult{
		Items: pageResult.Items,

		Meta: BrowseMeta{
			Query: normalized.DisplayQuery,

			Page: normalized.Page,

			Limit: normalized.Limit,

			Total: pageResult.Total,

			TotalPages: totalPages(
				pageResult.Total,
				normalized.Limit,
			),

			Sort: filters.Sort,

			NoMatch: pageResult.Total == 0,
		},

		Facets: pageResult.Facets,
	}, nil
}
