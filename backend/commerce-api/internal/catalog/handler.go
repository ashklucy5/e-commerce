package catalog

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"project.local/commerce-api/internal/auth"
	"project.local/commerce-api/internal/platform/pagination"
)

// Handler handles customer-facing catalog HTTP requests.
type Handler struct {
	service *Service
}

// NewHandler creates a catalog handler.
func NewHandler(
	service *Service,
) *Handler {
	return &Handler{
		service: service,
	}
}

// ListActiveProducts handles:
//
//	GET /api/v1/products
func (h *Handler) ListActiveProducts(
	c *gin.Context,
) {
	input, err :=
		ParseListProductsInput(
			c.Query(
				"page",
			),
			c.Query(
				"limit",
			),
			c.Query(
				"category",
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
		writeProductListError(
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
		h.service.ListProductsForCustomer(
			c.Request.Context(),
			customerID,
			input,
		)
	if err != nil {
		if isProductListValidationError(
			err,
		) {
			writeProductListError(
				c,
				err,
			)

			return
		}

		c.JSON(
			http.StatusInternalServerError,
			gin.H{
				"error": gin.H{
					"code": "INTERNAL_ERROR",

					"message": "Unable to load products",
				},
			},
		)

		return
	}

	c.JSON(
		http.StatusOK,
		gin.H{
			"data": result.Items,

			"meta": result.Meta,
		},
	)
}

// GetActiveProductBySlug handles:
//
//	GET /api/v1/products/:slug
func (h *Handler) GetActiveProductBySlug(
	c *gin.Context,
) {
	slug :=
		c.Param(
			"slug",
		)

	product, err :=
		h.service.GetActiveProductBySlug(
			c.Request.Context(),
			slug,
		)
	if err != nil {
		switch {

		case errors.Is(
			err,
			ErrInvalidProductSlug,
		):

			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"error": gin.H{
						"code": "INVALID_PRODUCT_SLUG",

						"message": "Product slug is required",
					},
				},
			)

		case errors.Is(
			err,
			ErrProductNotFound,
		):

			c.JSON(
				http.StatusNotFound,
				gin.H{
					"error": gin.H{
						"code": "PRODUCT_NOT_FOUND",

						"message": "Product not found",
					},
				},
			)

		default:

			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"error": gin.H{
						"code": "INTERNAL_ERROR",

						"message": "Unable to load product",
					},
				},
			)
		}

		return
	}

	c.JSON(
		http.StatusOK,
		gin.H{
			"data": product,
		},
	)
}

func isProductListValidationError(
	err error,
) bool {
	return errors.Is(
		err,
		pagination.ErrInvalidPage,
	) ||
		errors.Is(
			err,
			pagination.ErrInvalidLimit,
		) ||
		errors.Is(
			err,
			ErrInvalidCategoryFilter,
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
			ErrInvalidProductSort,
		)
}

func writeProductListError(
	c *gin.Context,
	err error,
) {
	code :=
		"INVALID_PRODUCT_QUERY"

	message :=
		"Invalid product query"

	switch {

	case errors.Is(
		err,
		pagination.ErrInvalidPage,
	):

		code =
			"INVALID_PAGE"

		message =
			"Invalid product page"

	case errors.Is(
		err,
		pagination.ErrInvalidLimit,
	):

		code =
			"INVALID_LIMIT"

		message =
			"Invalid product page size"

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
		ErrInvalidProductSort,
	):

		code =
			"INVALID_SORT"

		message =
			"Invalid product sort"
	}

	c.JSON(
		http.StatusBadRequest,
		gin.H{
			"error": gin.H{
				"code": code,

				"message": message,
			},
		},
	)
}
