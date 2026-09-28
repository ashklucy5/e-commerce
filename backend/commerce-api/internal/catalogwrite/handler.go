package catalogwrite

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgconn"

	platformerrors "project.local/commerce-api/internal/platform/errors"
)

type Handler struct {
	service *Service
}

func NewHandler(
	service *Service,
) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) CreateProduct(
	c *gin.Context,
) {
	var request CreateProductRequest

	if err := c.ShouldBindJSON(
		&request,
	); err != nil {
		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_PRODUCT_REQUEST",
				"Request body is not valid JSON",
			),
		)

		return
	}

	result, err := h.service.CreateProduct(
		c.Request.Context(),
		request.ToInput(),
	)
	if err != nil {
		writeCreateProductError(
			c,
			err,
		)

		return
	}

	c.JSON(
		http.StatusCreated,
		gin.H{
			"data": result,
		},
	)
}

func writeCreateProductError(
	c *gin.Context,
	err error,
) {
	var validationError *ProductValidationError

	switch {
	case errors.As(
		err,
		&validationError,
	):
		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_PRODUCT_REQUEST",
				validationError.Error(),
			),
		)

	case errors.Is(
		err,
		ErrProductCategoryNotFound,
	):
		platformerrors.Write(
			c,
			platformerrors.NotFound(
				"CATEGORY_NOT_FOUND",
				"Category not found",
			),
		)

	case errors.Is(
		err,
		ErrProductCategoryInactive,
	):
		platformerrors.Write(
			c,
			platformerrors.Conflict(
				"INACTIVE_CATEGORY",
				"Active products require an active category",
			),
		)

	case isCatalogWriteConflict(
		err,
	):
		platformerrors.Write(
			c,
			platformerrors.Conflict(
				"CATALOG_CONFLICT",
				"Product code, slug, SKU, barcode, or price tier conflicts with existing catalog data",
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

func isCatalogWriteConflict(
	err error,
) bool {
	var postgresError *pgconn.PgError

	if !errors.As(
		err,
		&postgresError,
	) {
		return false
	}

	return postgresError.Code ==
		"23505"
}
