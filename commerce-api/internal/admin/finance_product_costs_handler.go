package admin

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	platformerrors "project.local/commerce-api/internal/platform/errors"
	platformpagination "project.local/commerce-api/internal/platform/pagination"
	platformvalidation "project.local/commerce-api/internal/platform/validation"
)

type financeVariantCostCreateRequest struct {
	VariantID      string    `json:"variant_id"`
	SKU            string    `json:"sku"`
	UnitCostAmount *int64    `json:"unit_cost_amount"`
	Currency       string    `json:"currency"`
	EffectiveAt    time.Time `json:"effective_at"`
	Description    string    `json:"description"`
	Reference      string    `json:"reference"`
	SetCurrent     bool      `json:"set_current"`
}

func (h *Handler) FinanceVariantCosts(
	c *gin.Context,
) {
	params, err :=
		platformpagination.Parse(
			c.Query(
				"page",
			),
			c.Query(
				"limit",
			),
		)
	if err != nil {
		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_PAGINATION",
				"Invalid pagination",
			),
		)

		return
	}

	filter, err :=
		financeVariantCostFilterFromRequest(
			c,
		)
	if err != nil {
		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_FINANCE_VARIANT_COST_FILTER",
				err.Error(),
			),
		)

		return
	}

	result, err :=
		h.service.ListFinanceVariantCosts(
			c.Request.Context(),
			params,
			filter,
		)

	if writeFinanceVariantCostError(
		c,
		err,
	) {
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

func (h *Handler) FinanceVariantCost(
	c *gin.Context,
) {
	costID, ok :=
		financeVariantCostIDParam(
			c,
		)
	if !ok {
		return
	}

	result, err :=
		h.service.GetFinanceVariantCost(
			c.Request.Context(),
			costID,
		)

	if writeFinanceVariantCostError(
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

func (h *Handler) CreateFinanceVariantCost(
	c *gin.Context,
) {
	var request financeVariantCostCreateRequest

	if err :=
		c.ShouldBindJSON(
			&request,
		); err != nil {

		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_FINANCE_VARIANT_COST_REQUEST",
				"Invalid finance variant cost request",
			),
		)

		return
	}

	if request.UnitCostAmount == nil {
		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_FINANCE_VARIANT_COST_REQUEST",
				"unit_cost_amount is required",
			),
		)

		return
	}

	metadata, ok :=
		adminMutationMetadata(
			c,
		)
	if !ok {
		return
	}

	result, err :=
		h.service.CreateFinanceVariantCost(
			c.Request.Context(),
			FinanceVariantCostInput{
				VariantID: request.VariantID,

				SKU: request.SKU,

				UnitCostAmount: *request.UnitCostAmount,

				Currency: request.Currency,

				EffectiveAt: request.EffectiveAt,

				Source: FinanceVariantCostSourceManual,

				Description: request.Description,

				Reference: request.Reference,

				SetCurrent: request.SetCurrent,
			},
			metadata,
		)

	if writeFinanceVariantCostError(
		c,
		err,
	) {
		return
	}

	c.JSON(
		http.StatusCreated,
		gin.H{
			"data": result,
		},
	)
}

func financeVariantCostIDParam(
	c *gin.Context,
) (
	string,
	bool,
) {
	costID :=
		strings.TrimSpace(
			c.Param(
				"cost_id",
			),
		)

	if !platformvalidation.IsUUID(
		costID,
	) {
		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_FINANCE_VARIANT_COST_ID",
				"cost_id must be a valid UUID",
			),
		)

		return "",
			false
	}

	return costID,
		true
}

func financeVariantCostFilterFromRequest(
	c *gin.Context,
) (
	FinanceVariantCostFilter,
	error,
) {
	filter :=
		FinanceVariantCostFilter{
			ProductID: strings.TrimSpace(
				c.Query(
					"product_id",
				),
			),

			VariantID: strings.TrimSpace(
				c.Query(
					"variant_id",
				),
			),

			SKU: strings.TrimSpace(
				c.Query(
					"sku",
				),
			),

			Source: strings.ToLower(
				strings.TrimSpace(
					c.Query(
						"source",
					),
				),
			),

			Currency: strings.ToUpper(
				strings.TrimSpace(
					c.Query(
						"currency",
					),
				),
			),
		}

	fromValue :=
		strings.TrimSpace(
			c.Query(
				"from",
			),
		)

	toValue :=
		strings.TrimSpace(
			c.Query(
				"to",
			),
		)

	if fromValue != "" {
		from, err :=
			time.ParseInLocation(
				time.DateOnly,
				fromValue,
				financeExpenseBusinessLocation,
			)
		if err != nil {
			return FinanceVariantCostFilter{},
				errors.Join(
					ErrInvalidFinanceVariantCost,
					errors.New(
						"from must use YYYY-MM-DD",
					),
				)
		}

		filter.FromSet =
			true

		filter.From =
			from.UTC()
	}

	if toValue != "" {
		to, err :=
			time.ParseInLocation(
				time.DateOnly,
				toValue,
				financeExpenseBusinessLocation,
			)
		if err != nil {
			return FinanceVariantCostFilter{},
				errors.Join(
					ErrInvalidFinanceVariantCost,
					errors.New(
						"to must use YYYY-MM-DD",
					),
				)
		}

		filter.ToSet =
			true

		filter.To =
			to.AddDate(
				0,
				0,
				1,
			).UTC()
	}

	if err :=
		validateFinanceVariantCostFilter(
			filter,
		); err != nil {

		return FinanceVariantCostFilter{},
			err
	}

	if filter.FromSet &&
		filter.ToSet {

		days :=
			int(
				filter.To.Sub(
					filter.From,
				).Hours() / 24,
			)

		if days >
			financeExpenseMaxFilterDays {

			return FinanceVariantCostFilter{},
				errors.Join(
					ErrInvalidFinanceVariantCost,
					errors.New(
						"finance product-cost date range may not exceed 366 days",
					),
				)
		}
	}

	return filter,
		nil
}

func writeFinanceVariantCostError(
	c *gin.Context,
	err error,
) bool {
	if err == nil {
		return false
	}

	switch {
	case errors.Is(
		err,
		ErrFinanceVariantCostNotFound,
	):
		platformerrors.Write(
			c,
			platformerrors.NotFound(
				"FINANCE_VARIANT_COST_NOT_FOUND",
				"Finance variant cost history record not found",
			),
		)

	case errors.Is(
		err,
		ErrFinanceVariantNotFound,
	):
		platformerrors.Write(
			c,
			platformerrors.NotFound(
				"VARIANT_NOT_FOUND",
				"Product variant not found",
			),
		)

	case errors.Is(
		err,
		ErrFinanceVariantCostCurrencyMismatch,
	):
		platformerrors.Write(
			c,
			platformerrors.Conflict(
				"FINANCE_VARIANT_COST_CURRENCY_MISMATCH",
				err.Error(),
			),
		)

	case errors.Is(
		err,
		ErrInvalidFinanceVariantCost,
	):
		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_FINANCE_VARIANT_COST",
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
