package admin

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	platformerrors "project.local/commerce-api/internal/platform/errors"
	platformvalidation "project.local/commerce-api/internal/platform/validation"
	promotiondomain "project.local/commerce-api/internal/promotion"
)

type promotionTargetRequest struct {
	ProductID string `json:"product_id"`
	VariantID string `json:"variant_id"`
}

type promotionConfigRequest struct {
	Name string `json:"name"`

	Code *string `json:"code"`

	Scope        string `json:"scope"`
	CampaignType string `json:"campaign_type"`

	DiscountType string `json:"discount_type"`

	PercentageBPS *int   `json:"percentage_bps"`
	FixedAmount   *int64 `json:"fixed_amount"`

	MinimumSubtotalAmount int64  `json:"minimum_subtotal_amount"`
	MaximumDiscountAmount *int64 `json:"maximum_discount_amount"`

	Currency string `json:"currency"`
	Status   string `json:"status"`

	StartsAt *time.Time `json:"starts_at"`
	EndsAt   *time.Time `json:"ends_at"`

	Targets []promotionTargetRequest `json:"targets"`
}

func (h *Handler) Promotions(c *gin.Context) {
	params, ok := adminOperationalPagination(c)
	if !ok {
		return
	}

	status := strings.ToLower(strings.TrimSpace(c.Query("status")))
	switch status {
	case "", promotiondomain.StatusDraft, promotiondomain.StatusActive, promotiondomain.StatusDisabled:
	default:
		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_PROMOTION_STATUS",
				"status must be draft, active, or disabled",
			),
		)
		return
	}

	mode := strings.ToLower(strings.TrimSpace(c.Query("mode")))
	switch mode {
	case "", adminPromotionModeAutomatic, adminPromotionModeCode:
	default:
		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_PROMOTION_MODE",
				"mode must be automatic or code",
			),
		)
		return
	}

	scope := strings.ToLower(strings.TrimSpace(c.Query("scope")))
	switch scope {
	case "", promotiondomain.ScopeOrder, promotiondomain.ScopeProduct:
	default:
		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_PROMOTION_SCOPE",
				"scope must be order or product",
			),
		)
		return
	}

	campaignType := strings.ToLower(strings.TrimSpace(c.Query("campaign_type")))
	switch campaignType {
	case "", promotiondomain.CampaignTypeStandard, promotiondomain.CampaignTypeFlashSale:
	default:
		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_PROMOTION_CAMPAIGN_TYPE",
				"campaign_type must be standard or flash_sale",
			),
		)
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

	result, err := h.service.ListPromotions(
		c.Request.Context(),
		params,
		PromotionReadFilter{
			Status:       status,
			Mode:         mode,
			Scope:        scope,
			CampaignType: campaignType,
			Query:        query,
			QueryID:      queryID,
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

func (h *Handler) Promotion(c *gin.Context) {
	promotionID, ok := adminPromotionIDParam(c)
	if !ok {
		return
	}

	result, err := h.service.GetPromotion(
		c.Request.Context(),
		promotionID,
	)
	if writeAdminPromotionError(c, err) {
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (h *Handler) CreatePromotion(c *gin.Context) {
	var request promotionConfigRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_PROMOTION_REQUEST",
				"Invalid promotion request",
			),
		)
		return
	}

	metadata, ok := adminMutationMetadata(c)
	if !ok {
		return
	}

	result, err := h.service.CreatePromotion(
		c.Request.Context(),
		promotionInputFromRequest(request),
		metadata,
	)
	if writeAdminPromotionError(c, err) {
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": result})
}

func (h *Handler) ReplacePromotion(c *gin.Context) {
	promotionID, ok := adminPromotionIDParam(c)
	if !ok {
		return
	}

	var request promotionConfigRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_PROMOTION_REQUEST",
				"Invalid promotion request",
			),
		)
		return
	}

	metadata, ok := adminMutationMetadata(c)
	if !ok {
		return
	}

	result, err := h.service.ReplacePromotion(
		c.Request.Context(),
		promotionID,
		promotionInputFromRequest(request),
		metadata,
	)
	if writeAdminPromotionError(c, err) {
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

func promotionInputFromRequest(
	request promotionConfigRequest,
) PromotionConfigInput {
	targets := make([]PromotionTargetInput, 0, len(request.Targets))
	for _, target := range request.Targets {
		targets = append(
			targets,
			PromotionTargetInput{
				ProductID: target.ProductID,
				VariantID: target.VariantID,
			},
		)
	}

	return PromotionConfigInput{
		Name:                  request.Name,
		Code:                  request.Code,
		Scope:                 request.Scope,
		CampaignType:          request.CampaignType,
		DiscountType:          request.DiscountType,
		PercentageBPS:         request.PercentageBPS,
		FixedAmount:           request.FixedAmount,
		MinimumSubtotalAmount: request.MinimumSubtotalAmount,
		MaximumDiscountAmount: request.MaximumDiscountAmount,
		Currency:              request.Currency,
		Status:                request.Status,
		StartsAt:              request.StartsAt,
		EndsAt:                request.EndsAt,
		Targets:               targets,
	}
}

func adminPromotionIDParam(c *gin.Context) (string, bool) {
	promotionID := strings.TrimSpace(c.Param("promotion_id"))
	if !platformvalidation.IsUUID(promotionID) {
		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_PROMOTION_ID",
				"promotion_id must be a valid UUID",
			),
		)
		return "", false
	}

	return promotionID, true
}

func writeAdminPromotionError(c *gin.Context, err error) bool {
	if err == nil {
		return false
	}

	switch {
	case errors.Is(err, ErrAdminPromotionNotFound):
		platformerrors.Write(
			c,
			platformerrors.NotFound(
				"PROMOTION_NOT_FOUND",
				"Promotion not found",
			),
		)

	case errors.Is(err, ErrAdminPromotionConflict):
		platformerrors.Write(
			c,
			platformerrors.Conflict(
				"PROMOTION_CODE_CONFLICT",
				"Another promotion already uses this code",
			),
		)

	case errors.Is(err, ErrAdminPromotionTargetConflict):
		platformerrors.Write(
			c,
			platformerrors.Conflict(
				"PROMOTION_TARGET_CONFLICT",
				"Flash sale overlaps another active flash sale for one or more targeted variants",
			),
		)

	case errors.Is(err, ErrInvalidAdminPromotion):
		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_PROMOTION_CONFIGURATION",
				err.Error(),
			),
		)

	default:
		platformerrors.Write(c, platformerrors.Internal(err))
	}

	return true
}
