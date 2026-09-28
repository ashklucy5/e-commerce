package catalog

import (
	"fmt"
	"strconv"
	"strings"

	"project.local/commerce-api/internal/platform/pagination"
)

const maxCategorySlugLength = 140

func ParseListProductsInput(
	pageValue string,
	limitValue string,
	categoryValue string,
	minPriceValue string,
	maxPriceValue string,
	inStockValue string,
	sortValue string,
) (ListProductsInput, error) {
	params, err :=
		pagination.Parse(
			pageValue,
			limitValue,
		)
	if err != nil {
		return ListProductsInput{}, err
	}

	minPrice, err :=
		parseOptionalNonNegativeInt64(
			minPriceValue,
		)
	if err != nil {
		return ListProductsInput{},
			fmt.Errorf(
				"%w: min_price must be a non-negative integer",
				ErrInvalidPriceFilter,
			)
	}

	maxPrice, err :=
		parseOptionalNonNegativeInt64(
			maxPriceValue,
		)
	if err != nil {
		return ListProductsInput{},
			fmt.Errorf(
				"%w: max_price must be a non-negative integer",
				ErrInvalidPriceFilter,
			)
	}

	if minPrice != nil &&
		maxPrice != nil &&
		*maxPrice < *minPrice {

		return ListProductsInput{},
			fmt.Errorf(
				"%w: max_price must be greater than or equal to min_price",
				ErrInvalidPriceRange,
			)
	}

	inStock, err :=
		parseOptionalBool(
			inStockValue,
		)
	if err != nil {
		return ListProductsInput{},
			fmt.Errorf(
				"%w: in_stock must be true or false",
				ErrInvalidStockFilter,
			)
	}

	categorySlug :=
		strings.TrimSpace(
			categoryValue,
		)

	if len(categorySlug) >
		maxCategorySlugLength {

		return ListProductsInput{},
			fmt.Errorf(
				"%w: category slug is too long",
				ErrInvalidCategoryFilter,
			)
	}

	sort, err :=
		normalizeProductSort(
			sortValue,
		)
	if err != nil {
		return ListProductsInput{}, err
	}

	return ListProductsInput{
		Page:  params.Page,
		Limit: params.Limit,

		CategorySlug: categorySlug,
		MinPrice:     minPrice,
		MaxPrice:     maxPrice,
		InStock:      inStock,
		Sort:         sort,
	}, nil
}

func normalizeListProductsInput(
	input ListProductsInput,
) (
	ListProductsInput,
	pagination.Params,
	error,
) {
	page :=
		input.Page

	if page == 0 {
		page =
			pagination.DefaultPage
	}

	limit :=
		input.Limit

	if limit == 0 {
		limit =
			pagination.DefaultLimit
	}

	params, err :=
		pagination.New(
			page,
			limit,
		)
	if err != nil {
		return ListProductsInput{},
			pagination.Params{},
			err
	}

	if input.MinPrice != nil &&
		*input.MinPrice < 0 {

		return ListProductsInput{},
			pagination.Params{},
			ErrInvalidPriceFilter
	}

	if input.MaxPrice != nil &&
		*input.MaxPrice < 0 {

		return ListProductsInput{},
			pagination.Params{},
			ErrInvalidPriceFilter
	}

	if input.MinPrice != nil &&
		input.MaxPrice != nil &&
		*input.MaxPrice <
			*input.MinPrice {

		return ListProductsInput{},
			pagination.Params{},
			ErrInvalidPriceRange
	}

	categorySlug :=
		strings.TrimSpace(
			input.CategorySlug,
		)

	if len(categorySlug) >
		maxCategorySlugLength {

		return ListProductsInput{},
			pagination.Params{},
			ErrInvalidCategoryFilter
	}

	sort, err :=
		normalizeProductSort(
			input.Sort,
		)
	if err != nil {
		return ListProductsInput{},
			pagination.Params{},
			err
	}

	input.Page =
		params.Page

	input.Limit =
		params.Limit

	input.CategorySlug =
		categorySlug

	input.Sort =
		sort

	return input,
		params,
		nil
}

func normalizeProductSort(
	value string,
) (string, error) {
	value =
		strings.ToLower(
			strings.TrimSpace(
				value,
			),
		)

	if value == "" {
		return ProductSortRecommended,
			nil
	}

	switch value {

	case ProductSortRecommended,
		ProductSortFeatured,
		ProductSortNewest,
		ProductSortPriceAsc,
		ProductSortPriceDesc,
		ProductSortNameAsc,
		ProductSortNameDesc:

		return value, nil

	default:

		return "",
			fmt.Errorf(
				"%w: unsupported sort %q",
				ErrInvalidProductSort,
				value,
			)
	}
}

func parseOptionalNonNegativeInt64(
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

func parseOptionalBool(
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

	var parsed bool

	switch value {

	case "true",
		"1":

		parsed = true

	case "false",
		"0":

		parsed = false

	default:

		return nil,
			ErrInvalidStockFilter
	}

	return &parsed, nil
}
