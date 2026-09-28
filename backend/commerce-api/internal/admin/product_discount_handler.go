package admin

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	platformerrors "project.local/commerce-api/internal/platform/errors"
	platformvalidation "project.local/commerce-api/internal/platform/validation"
)

type productDiscountConfigRequest struct {
	DiscountType string `json:"discount_type"`

	PercentageBPS *int   `json:"percentage_bps"`
	FixedAmount   *int64 `json:"fixed_amount"`
}

func (h *Handler) ProductDiscounts(c *gin.Context) {
	params, ok := adminOperationalPagination(c)
	if !ok {
		return
	}

	query, ok := adminOperationalQuery(c)
	if !ok {
		return
	}

	queryID := ""
	if platformvalidation.IsUUID(query) {
		queryID = query
	}

	result, err := h.service.ListProductDiscounts(
		c.Request.Context(),
		params,
		ProductDiscountReadFilter{
			Query:   query,
			QueryID: queryID,
		},
	)
	if err != nil {
		platformerrors.Write(c, platformerrors.Internal(err))
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

func (h *Handler) ApplyProductDiscount(c *gin.Context) {
	variantID, ok := adminPromotionVariantIDParam(c)
	if !ok {
		return
	}

	var request productDiscountConfigRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_PRODUCT_DISCOUNT_REQUEST",
				"Invalid product discount request",
			),
		)
		return
	}

	metadata, ok := adminMutationMetadata(c)
	if !ok {
		return
	}

	result, err := h.service.ApplyProductDiscount(
		c.Request.Context(),
		variantID,
		ProductDiscountConfigInput{
			DiscountType:  request.DiscountType,
			PercentageBPS: request.PercentageBPS,
			FixedAmount:   request.FixedAmount,
		},
		metadata,
	)
	if writeAdminProductDiscountError(c, err) {
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (h *Handler) ClearProductDiscount(c *gin.Context) {
	variantID, ok := adminPromotionVariantIDParam(c)
	if !ok {
		return
	}

	metadata, ok := adminMutationMetadata(c)
	if !ok {
		return
	}

	result, err := h.service.ClearProductDiscount(
		c.Request.Context(),
		variantID,
		metadata,
	)
	if writeAdminProductDiscountError(c, err) {
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

func adminPromotionVariantIDParam(c *gin.Context) (string, bool) {
	variantID := strings.TrimSpace(c.Param("variant_id"))
	if !platformvalidation.IsUUID(variantID) {
		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_VARIANT_ID",
				"variant_id must be a valid UUID",
			),
		)
		return "", false
	}

	return variantID, true
}

func writeAdminProductDiscountError(c *gin.Context, err error) bool {
	if err == nil {
		return false
	}

	switch {
	case errors.Is(err, ErrAdminProductDiscountVariantNotFound):
		platformerrors.Write(
			c,
			platformerrors.NotFound(
				"PRODUCT_VARIANT_NOT_FOUND",
				"Product variant not found",
			),
		)

	case errors.Is(err, ErrInvalidAdminProductDiscount):
		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_PRODUCT_DISCOUNT_CONFIGURATION",
				err.Error(),
			),
		)

	default:
		platformerrors.Write(c, platformerrors.Internal(err))
	}

	return true
}
