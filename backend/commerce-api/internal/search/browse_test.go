package search

import (
	"context"
	"errors"
	"testing"
)

type browseFakeStore struct {
	fakeStore

	browsePage BrowsePage

	browseErr error

	browseQuery string

	browsePageNum int

	browseLimit int

	browseFilters Filters
}

func (f *browseFakeStore) SearchBrowse(
	_ context.Context,
	normalizedQuery string,
	page int,
	limit int,
	filters Filters,
) (BrowsePage, error) {
	f.browseQuery =
		normalizedQuery

	f.browsePageNum =
		page

	f.browseLimit =
		limit

	f.browseFilters =
		filters

	if f.browseErr != nil {
		return BrowsePage{},
			f.browseErr
	}

	return f.browsePage,
		nil
}

func TestParseFilters(
	t *testing.T,
) {
	filters, err :=
		ParseFilters(
			"shirts",
			"Acme",
			"1000",
			"5000",
			"true",
			"price_asc",
		)
	if err != nil {
		t.Fatalf(
			"parse filters: %v",
			err,
		)
	}

	if filters.CategorySlug !=
		"shirts" ||
		filters.Brand !=
			"Acme" ||
		filters.MinPrice == nil ||
		*filters.MinPrice !=
			1000 ||
		filters.MaxPrice == nil ||
		*filters.MaxPrice !=
			5000 ||
		filters.InStock == nil ||
		!*filters.InStock ||
		filters.Sort !=
			SearchSortPriceAsc {

		t.Fatalf(
			"unexpected filters: %#v",
			filters,
		)
	}
}

func TestParseFiltersRejectsRange(
	t *testing.T,
) {
	_, err :=
		ParseFilters(
			"",
			"",
			"5000",
			"1000",
			"",
			"",
		)

	if !errors.Is(
		err,
		ErrInvalidPriceRange,
	) {
		t.Fatalf(
			"expected ErrInvalidPriceRange, got %v",
			err,
		)
	}
}

func TestBrowsePassesNormalizedFilters(
	t *testing.T,
) {
	store :=
		&browseFakeStore{
			browsePage: BrowsePage{
				Items: []ProductResult{
					{
						ID: "product-1",

						SearchTier: 1,

						MatchType: "primary",
					},
				},

				Total: 1,

				Facets: Facets{
					Categories: []CategoryFacet{},

					Brands: []BrandFacet{},
				},
			},
		}

	service :=
		lexicalTestService(
			store,
		)

	minPrice :=
		int64(
			1000,
		)

	stock :=
		true

	result, err :=
		service.Browse(
			context.Background(),
			"",
			BrowseInput{
				Query: "  Formal  Shirt ",

				Page: 2,

				Limit: 24,

				Filters: Filters{
					CategorySlug: "shirts",

					MinPrice: &minPrice,

					InStock: &stock,

					Sort: SearchSortNewest,
				},
			},
		)
	if err != nil {
		t.Fatalf(
			"browse: %v",
			err,
		)
	}

	if store.browseQuery !=
		"formal shirt" ||
		store.browsePageNum !=
			2 ||
		store.browseLimit !=
			24 ||
		store.browseFilters.Sort !=
			SearchSortNewest {

		t.Fatalf(
			"unexpected browse call: query=%q page=%d limit=%d filters=%#v",
			store.browseQuery,
			store.browsePageNum,
			store.browseLimit,
			store.browseFilters,
		)
	}

	if result.Meta.Total !=
		1 ||
		result.Meta.TotalPages !=
			1 {

		t.Fatalf(
			"unexpected browse meta: %#v",
			result.Meta,
		)
	}
}
