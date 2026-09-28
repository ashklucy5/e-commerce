package catalog

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"project.local/commerce-api/internal/platform/pagination"
)

// ListHomeProducts handles:
//
//	GET /api/v1/products/home-feed?page=1&limit=50
func (h *Handler) ListHomeProducts(
	c *gin.Context,
) {
	params, err :=
		pagination.Parse(
			c.Query(
				"page",
			),
			c.Query(
				"limit",
			),
		)

	if err != nil {
		code :=
			"INVALID_PAGINATION"

		message :=
			"Invalid product pagination"

		switch {

		case errors.Is(
			err,
			pagination.ErrInvalidPage,
		):

			code =
				"INVALID_PAGE"

			message =
				"Product page must be a valid positive page number"

		case errors.Is(
			err,
			pagination.ErrInvalidLimit,
		):

			code =
				"INVALID_LIMIT"

			message =
				"Product limit must be between 1 and 100"
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

		return
	}

	result, err :=
		h.service.ListHomeProducts(
			c.Request.Context(),
			params.Page,
			params.Limit,
		)

	if err != nil {
		c.JSON(
			http.StatusInternalServerError,
			gin.H{
				"error": gin.H{
					"code": "INTERNAL_ERROR",

					"message": "Unable to load homepage products",
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
