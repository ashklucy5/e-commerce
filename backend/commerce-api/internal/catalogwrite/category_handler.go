package catalogwrite

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	platformerrors "project.local/commerce-api/internal/platform/errors"
)

func (h *Handler) CategoryTree(
	c *gin.Context,
) {
	items, err := h.service.CategoryTree(
		c.Request.Context(),
	)
	if err != nil {
		platformerrors.Write(
			c,
			platformerrors.Internal(
				err,
			),
		)

		return
	}

	c.JSON(
		http.StatusOK,
		gin.H{
			"data": items,
		},
	)
}

func (h *Handler) CreateCategory(
	c *gin.Context,
) {
	var request CreateCategoryRequest

	if err := c.ShouldBindJSON(
		&request,
	); err != nil {
		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_ADMIN_CATEGORY_REQUEST",
				"Request body is not valid JSON",
			),
		)

		return
	}

	created, err := h.service.CreateCategory(
		c.Request.Context(),
		request.ToInput(),
	)
	if err != nil {
		writeCreateCategoryError(
			c,
			err,
		)

		return
	}

	c.JSON(
		http.StatusCreated,
		gin.H{
			"data": created,
		},
	)
}

func writeCreateCategoryError(
	c *gin.Context,
	err error,
) {
	switch {
	case errors.Is(
		err,
		ErrCategoryNameRequired,
	):
		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_ADMIN_CATEGORY_REQUEST",
				"Category name is required",
			),
		)

	case errors.Is(
		err,
		ErrInvalidCategorySortOrder,
	):
		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_CATEGORY_SORT_ORDER",
				"sort_order cannot be negative",
			),
		)

	case errors.Is(
		err,
		ErrCategoryParentNotFound,
	):
		platformerrors.Write(
			c,
			platformerrors.NotFound(
				"CATEGORY_PARENT_NOT_FOUND",
				"Parent category not found",
			),
		)

	case errors.Is(
		err,
		ErrCategoryAlreadyExists,
	):
		platformerrors.Write(
			c,
			platformerrors.Conflict(
				"CATEGORY_ALREADY_EXISTS",
				"Category already exists under the selected parent",
			),
		)

	case errors.Is(
		err,
		ErrInvalidProductCodePrefix,
	):
		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_PRODUCT_CODE_PREFIX",
				"Product-code prefix must use the format AAA-BBB with uppercase letters or digits",
			),
		)

	case errors.Is(
		err,
		ErrProductCodePrefixInUse,
	):
		platformerrors.Write(
			c,
			platformerrors.Conflict(
				"PRODUCT_CODE_PREFIX_IN_USE",
				"Product-code prefix is already assigned to another category",
			),
		)

	default:
		platformerrors.Write(
			c,
			platformerrors.Internal(
				err,
			),
		)
	}
}
