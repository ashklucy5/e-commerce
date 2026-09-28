package search

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"project.local/commerce-api/internal/auth"
)

func (h *Handler) Browse(
	c *gin.Context,
) {
	page, err :=
		optionalQueryInt(
			c.Query(
				"page",
			),
		)
	if err != nil {
		writeSearchError(
			c,
			ErrInvalidPage,
		)

		return
	}

	limit, err :=
		optionalQueryInt(
			c.Query(
				"limit",
			),
		)
	if err != nil {
		writeSearchError(
			c,
			ErrInvalidLimit,
		)

		return
	}

	filters, err :=
		ParseFilters(
			c.Query(
				"category",
			),
			c.Query(
				"brand",
			),
			c.Query(
				"min_price",
			),
			c.Query(
				"max_price",
			),
			c.Query(
				"in_stock",
			),
			c.Query(
				"sort",
			),
		)
	if err != nil {
		writeBrowseError(
			c,
			err,
		)

		return
	}

	customerID, _ :=
		auth.CustomerIDFromContext(
			c,
		)

	result, err :=
		h.service.Browse(
			c.Request.Context(),
			customerID,
			BrowseInput{
				Query: c.Query(
					"q",
				),

				Page: page,

				Limit: limit,

				Filters: filters,
			},
		)
	if err != nil {
		if errors.Is(
			err,
			ErrInvalidQuery,
		) ||
			errors.Is(
				err,
				ErrInvalidPage,
			) ||
			errors.Is(
				err,
				ErrInvalidLimit,
			) {

			writeSearchError(
				c,
				err,
			)

			return
		}

		if isBrowseValidationError(
			err,
		) {
			writeBrowseError(
				c,
				err,
			)

			return
		}

		writeSearchJSONError(
			c,
			http.StatusInternalServerError,
			"INTERNAL_ERROR",
			"Unable to search products",
		)

		return
	}

	c.JSON(
		http.StatusOK,
		gin.H{
			"data": result.Items,

			"meta": result.Meta,

			"facets": result.Facets,
		},
	)
}

func isBrowseValidationError(
	err error,
) bool {
	return errors.Is(
		err,
		ErrInvalidCategoryFilter,
	) ||
		errors.Is(
			err,
			ErrInvalidBrandFilter,
		) ||
		errors.Is(
			err,
			ErrInvalidPriceFilter,
		) ||
		errors.Is(
			err,
			ErrInvalidPriceRange,
		) ||
		errors.Is(
			err,
			ErrInvalidStockFilter,
		) ||
		errors.Is(
			err,
			ErrInvalidSort,
		)
}

func writeBrowseError(
	c *gin.Context,
	err error,
) {
	code :=
		"INVALID_SEARCH_FILTER"

	message :=
		"Invalid search filter"

	switch {
	case errors.Is(
		err,
		ErrInvalidCategoryFilter,
	):

		code =
			"INVALID_CATEGORY_FILTER"

		message =
			"Invalid category filter"

	case errors.Is(
		err,
		ErrInvalidBrandFilter,
	):

		code =
			"INVALID_BRAND_FILTER"

		message =
			"Invalid brand filter"

	case errors.Is(
		err,
		ErrInvalidPriceFilter,
	),
		errors.Is(
			err,
			ErrInvalidPriceRange,
		):

		code =
			"INVALID_PRICE_FILTER"

		message =
			"Invalid price filter"

	case errors.Is(
		err,
		ErrInvalidStockFilter,
	):

		code =
			"INVALID_STOCK_FILTER"

		message =
			"Invalid stock filter"

	case errors.Is(
		err,
		ErrInvalidSort,
	):

		code =
			"INVALID_SEARCH_SORT"

		message =
			"Invalid search sort"
	}

	writeSearchJSONError(
		c,
		http.StatusBadRequest,
		code,
		message,
	)
}
