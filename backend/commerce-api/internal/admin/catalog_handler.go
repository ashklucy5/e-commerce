package admin

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	platformerrors "project.local/commerce-api/internal/platform/errors"
	platformvalidation "project.local/commerce-api/internal/platform/validation"
)

type updateCatalogProductRequest struct {
	CategoryID *string `json:"category_id"`

	Name *string `json:"name"`
	Slug *string `json:"slug"`

	Brand *string `json:"brand"`

	ShortDescription *string `json:"short_description"`
	Description      *string `json:"description"`

	Status     *string `json:"status"`
	IsFeatured *bool   `json:"is_featured"`
}

func (h *Handler) CatalogProducts(
	c *gin.Context,
) {
	params, ok :=
		adminOperationalPagination(
			c,
		)
	if !ok {
		return
	}

	status :=
		strings.ToLower(
			strings.TrimSpace(
				c.Query(
					"status",
				),
			),
		)

	switch status {
	case "",
		"draft",
		"active",
		"archived":

	default:
		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_PRODUCT_STATUS",
				"status must be draft, active, or archived",
			),
		)

		return
	}

	query, ok :=
		adminOperationalQuery(
			c,
		)
	if !ok {
		return
	}

	queryID := ""

	if platformvalidation.IsUUID(
		query,
	) {
		queryID =
			query
	}

	result, err :=
		h.service.ListCatalogProducts(
			c.Request.Context(),
			params,
			CatalogProductReadFilter{
				Status: status,

				Query: query,

				QueryID: queryID,
			},
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
			"data": result.Items,

			"meta": result.Meta,
		},
	)
}

func (h *Handler) CatalogProduct(
	c *gin.Context,
) {
	productID, ok :=
		adminCatalogProductIDParam(
			c,
		)
	if !ok {
		return
	}

	result, err :=
		h.service.GetCatalogProduct(
			c.Request.Context(),
			productID,
		)

	if writeAdminCatalogError(
		c,
		err,
	) {
		return
	}

	c.JSON(
		http.StatusOK,
		gin.H{
			"data": result,
		},
	)
}

func (h *Handler) UpdateCatalogProduct(
	c *gin.Context,
) {
	productID, ok :=
		adminCatalogProductIDParam(
			c,
		)
	if !ok {
		return
	}

	var request updateCatalogProductRequest

	if err :=
		c.ShouldBindJSON(
			&request,
		); err != nil {

		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_PRODUCT_REQUEST",
				"Invalid product request",
			),
		)

		return
	}

	if request.CategoryID == nil &&
		request.Name == nil &&
		request.Slug == nil &&
		request.Brand == nil &&
		request.ShortDescription == nil &&
		request.Description == nil &&
		request.Status == nil &&
		request.IsFeatured == nil {

		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"EMPTY_PRODUCT_UPDATE",
				"At least one product field must be supplied",
			),
		)

		return
	}

	if request.CategoryID != nil {
		value :=
			strings.TrimSpace(
				*request.CategoryID,
			)

		if !platformvalidation.IsUUID(
			value,
		) {
			platformerrors.Write(
				c,
				platformerrors.BadRequest(
					"INVALID_CATEGORY_ID",
					"category_id must be a valid UUID",
				),
			)

			return
		}

		request.CategoryID =
			&value
	}

	metadata, ok :=
		adminMutationMetadata(
			c,
		)
	if !ok {
		return
	}

	result, err :=
		h.service.UpdateCatalogProduct(
			c.Request.Context(),
			productID,
			UpdateCatalogProductInput{
				CategoryID: request.CategoryID,

				Name: request.Name,

				Slug: request.Slug,

				Brand: request.Brand,

				ShortDescription: request.ShortDescription,

				Description: request.Description,

				Status: request.Status,

				IsFeatured: request.IsFeatured,
			},
			metadata,
		)

	if writeAdminCatalogError(
		c,
		err,
	) {
		return
	}

	c.JSON(
		http.StatusOK,
		gin.H{
			"data": result,
		},
	)
}

func adminCatalogProductIDParam(
	c *gin.Context,
) (
	string,
	bool,
) {
	productID :=
		strings.TrimSpace(
			c.Param(
				"product_id",
			),
		)

	if !platformvalidation.IsUUID(
		productID,
	) {
		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_PRODUCT_ID",
				"product_id must be a valid UUID",
			),
		)

		return "",
			false
	}

	return productID,
		true
}

func writeAdminCatalogError(
	c *gin.Context,
	err error,
) bool {
	if err == nil {
		return false
	}

	switch {
	case errors.Is(
		err,
		ErrAdminCatalogProductNotFound,
	):
		platformerrors.Write(
			c,
			platformerrors.NotFound(
				"PRODUCT_NOT_FOUND",
				"Product not found",
			),
		)

	case errors.Is(
		err,
		ErrInvalidAdminCatalogProduct,
	):
		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_CATALOG_DATA",
				err.Error(),
			),
		)

	case errors.Is(
		err,
		ErrAdminCatalogConflict,
	):
		platformerrors.Write(
			c,
			platformerrors.Conflict(
				"CATALOG_CONFLICT",
				"The product update conflicts with an existing product code or slug",
			),
		)

	case errors.Is(
		err,
		ErrAdminCatalogInactiveCategory,
	):
		platformerrors.Write(
			c,
			platformerrors.Conflict(
				"INACTIVE_CATEGORY",
				err.Error(),
			),
		)

	case errors.Is(
		err,
		ErrAdminCatalogNoActiveVariant,
	):
		platformerrors.Write(
			c,
			platformerrors.Conflict(
				"NO_ACTIVE_VARIANT",
				err.Error(),
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

	return true
}
