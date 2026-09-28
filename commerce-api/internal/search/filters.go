package search

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

const (
	SearchSortRelevance = "relevance"
	SearchSortFeatured  = "featured"
	SearchSortNewest    = "newest"
	SearchSortPriceAsc  = "price_asc"
	SearchSortPriceDesc = "price_desc"
	SearchSortNameAsc   = "name_asc"
	SearchSortNameDesc  = "name_desc"
)

var (
	ErrInvalidCategoryFilter = errors.New(
		"invalid search category filter",
	)

	ErrInvalidBrandFilter = errors.New(
		"invalid search brand filter",
	)

	ErrInvalidPriceFilter = errors.New(
		"invalid search price filter",
	)

	ErrInvalidPriceRange = errors.New(
		"invalid search price range",
	)

	ErrInvalidStockFilter = errors.New(
		"invalid search stock filter",
	)

	ErrInvalidSort = errors.New(
		"invalid search sort",
	)
)

type Filters struct {
	CategorySlug string

	Brand string

	MinPrice *int64
	MaxPrice *int64

	InStock *bool

	Sort string
}

func ParseFilters(
	category string,
	brand string,
	minPrice string,
	maxPrice string,
	inStock string,
	sortValue string,
) (Filters, error) {
	category =
		strings.TrimSpace(
			category,
		)

	brand =
		strings.TrimSpace(
			brand,
		)

	if len(category) > 140 {
		return Filters{},
			ErrInvalidCategoryFilter
	}

	if len(brand) > 160 {
		return Filters{},
			ErrInvalidBrandFilter
	}

	minValue, err :=
		parseOptionalNonNegativeAmount(
			minPrice,
		)
	if err != nil {
		return Filters{},
			ErrInvalidPriceFilter
	}

	maxValue, err :=
		parseOptionalNonNegativeAmount(
			maxPrice,
		)
	if err != nil {
		return Filters{},
			ErrInvalidPriceFilter
	}

	if minValue != nil &&
		maxValue != nil &&
		*maxValue < *minValue {

		return Filters{},
			ErrInvalidPriceRange
	}

	stockValue, err :=
		parseOptionalSearchBool(
			inStock,
		)
	if err != nil {
		return Filters{},
			ErrInvalidStockFilter
	}

	sortValue, err =
		normalizeSearchSort(
			sortValue,
		)
	if err != nil {
		return Filters{}, err
	}

	return Filters{
		CategorySlug: category,

		Brand: brand,

		MinPrice: minValue,

		MaxPrice: maxValue,

		InStock: stockValue,

		Sort: sortValue,
	}, nil
}

func normalizeSearchSort(
	value string,
) (string, error) {
	value =
		strings.ToLower(
			strings.TrimSpace(
				value,
			),
		)

	if value == "" {
		return SearchSortRelevance,
			nil
	}

	switch value {
	case SearchSortRelevance,
		SearchSortFeatured,
		SearchSortNewest,
		SearchSortPriceAsc,
		SearchSortPriceDesc,
		SearchSortNameAsc,
		SearchSortNameDesc:

		return value, nil

	default:

		return "",
			fmt.Errorf(
				"%w: %q",
				ErrInvalidSort,
				value,
			)
	}
}

func parseOptionalNonNegativeAmount(
	value string,
) (*int64, error) {
	value =
		strings.TrimSpace(
			value,
		)

	if value == "" {
		return nil, nil
	}

	parsed, err :=
		strconv.ParseInt(
			value,
			10,
			64,
		)
	if err != nil ||
		parsed < 0 {

		return nil,
			ErrInvalidPriceFilter
	}

	return &parsed, nil
}

func parseOptionalSearchBool(
	value string,
) (*bool, error) {
	value =
		strings.ToLower(
			strings.TrimSpace(
				value,
			),
		)

	if value == "" {
		return nil, nil
	}

	var result bool

	switch value {
	case "true",
		"1":

		result = true

	case "false",
		"0":

		result = false

	default:

		return nil,
			ErrInvalidStockFilter
	}

	return &result, nil
}
