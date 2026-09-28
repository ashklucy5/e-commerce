package category

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Handler converts HTTP requests into service calls
// and service results into HTTP responses.
type Handler struct {
	service *Service
}

// NewHandler creates a category HTTP handler.
func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
}

// ListActive handles:
//
//	GET /api/v1/categories
func (h *Handler) ListActive(c *gin.Context) {
	categories, err := h.service.ListActive(
		c.Request.Context(),
	)
	if err != nil {
		c.JSON(
			http.StatusInternalServerError,
			gin.H{
				"error": gin.H{
					"code":    "INTERNAL_ERROR",
					"message": "Unable to load categories",
				},
			},
		)

		return
	}

	c.JSON(
		http.StatusOK,
		gin.H{
			"data": categories,
		},
	)
}

// GetActiveBySlug handles:
//
//	GET /api/v1/categories/:slug
func (h *Handler) GetActiveBySlug(c *gin.Context) {
	slug := c.Param("slug")

	item, err := h.service.GetActiveBySlug(
		c.Request.Context(),
		slug,
	)
	if err != nil {
		switch {
		case errors.Is(
			err,
			ErrInvalidCategorySlug,
		):
			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"error": gin.H{
						"code":    "INVALID_CATEGORY_SLUG",
						"message": "Category slug is required",
					},
				},
			)

		case errors.Is(
			err,
			ErrCategoryNotFound,
		):
			c.JSON(
				http.StatusNotFound,
				gin.H{
					"error": gin.H{
						"code":    "CATEGORY_NOT_FOUND",
						"message": "Category not found",
					},
				},
			)

		default:
			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"error": gin.H{
						"code":    "INTERNAL_ERROR",
						"message": "Unable to load category",
					},
				},
			)
		}

		return
	}

	c.JSON(
		http.StatusOK,
		gin.H{
			"data": item,
		},
	)
}

// Tree handles:
//
//	GET /api/v1/categories/tree
func (h *Handler) Tree(c *gin.Context) {
	tree, err := h.service.Tree(
		c.Request.Context(),
	)
	if err != nil {
		c.JSON(
			http.StatusInternalServerError,
			gin.H{
				"error": gin.H{
					"code":    "INTERNAL_ERROR",
					"message": "Unable to load category tree",
				},
			},
		)

		return
	}

	c.JSON(
		http.StatusOK,
		gin.H{
			"data": tree,
		},
	)
}
