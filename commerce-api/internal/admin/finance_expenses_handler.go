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

const financeExpenseMaxFilterDays = 366

var financeExpenseBusinessLocation = time.FixedZone(
	"Asia/Dhaka",
	6*60*60,
)

type financeExpenseCreateRequest struct {
	Category string `json:"category"`
	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`

	OccurredAt time.Time `json:"occurred_at"`

	Description string `json:"description"`

	OrderID        string `json:"order_id"`
	ProductID      string `json:"product_id"`
	VariantID      string `json:"variant_id"`
	StaffAccountID string `json:"staff_account_id"`
	WarehouseID    string `json:"warehouse_id"`

	Reference string `json:"reference"`
}

type financeExpenseVoidRequest struct {
	Reason string `json:"reason"`
}

func (h *Handler) FinanceExpenses(
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
		financeExpenseFilterFromRequest(
			c,
		)
	if err != nil {
		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_FINANCE_EXPENSE_FILTER",
				err.Error(),
			),
		)

		return
	}

	result, err :=
		h.service.ListFinanceExpenses(
			c.Request.Context(),
			params,
			filter,
		)
	if writeFinanceExpenseError(
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

func (h *Handler) FinanceExpense(
	c *gin.Context,
) {
	expenseID, ok :=
		financeExpenseIDParam(
			c,
		)
	if !ok {
		return
	}

	result, err :=
		h.service.GetFinanceExpense(
			c.Request.Context(),
			expenseID,
		)

	if writeFinanceExpenseError(
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

func (h *Handler) CreateFinanceExpense(
	c *gin.Context,
) {
	var request financeExpenseCreateRequest

	if err :=
		c.ShouldBindJSON(
			&request,
		); err != nil {

		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_FINANCE_EXPENSE_REQUEST",
				"Invalid finance expense request",
			),
		)

		return
	}

	idempotencyKey :=
		strings.TrimSpace(
			c.GetHeader(
				"Idempotency-Key",
			),
		)

	if idempotencyKey == "" {
		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"IDEMPOTENCY_KEY_REQUIRED",
				"Idempotency-Key header is required",
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
		h.service.CreateFinanceExpense(
			c.Request.Context(),
			FinanceExpenseInput{
				Category: request.Category,
				Amount:   request.Amount,
				Currency: request.Currency,

				OccurredAt: request.OccurredAt,

				Description: request.Description,

				OrderID: request.OrderID,

				ProductID: request.ProductID,

				VariantID: request.VariantID,

				StaffAccountID: request.StaffAccountID,

				WarehouseID: request.WarehouseID,

				Reference: request.Reference,
			},
			idempotencyKey,
			metadata,
		)

	if writeFinanceExpenseError(
		c,
		err,
	) {
		return
	}

	status :=
		http.StatusCreated

	if !result.Inserted {
		status =
			http.StatusOK
	}

	c.JSON(
		status,
		gin.H{
			"data": result,
		},
	)
}

func (h *Handler) VoidFinanceExpense(
	c *gin.Context,
) {
	expenseID, ok :=
		financeExpenseIDParam(
			c,
		)
	if !ok {
		return
	}

	var request financeExpenseVoidRequest

	if err :=
		c.ShouldBindJSON(
			&request,
		); err != nil {

		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_FINANCE_EXPENSE_VOID_REQUEST",
				"Invalid finance expense void request",
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
		h.service.VoidFinanceExpense(
			c.Request.Context(),
			expenseID,
			request.Reason,
			metadata,
		)

	if writeFinanceExpenseError(
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

func financeExpenseIDParam(
	c *gin.Context,
) (
	string,
	bool,
) {
	expenseID :=
		strings.TrimSpace(
			c.Param(
				"expense_id",
			),
		)

	if !platformvalidation.IsUUID(
		expenseID,
	) {
		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_FINANCE_EXPENSE_ID",
				"expense_id must be a valid UUID",
			),
		)

		return "",
			false
	}

	return expenseID,
		true
}

func financeExpenseFilterFromRequest(
	c *gin.Context,
) (
	FinanceExpenseFilter,
	error,
) {
	filter :=
		FinanceExpenseFilter{
			Category: strings.ToLower(
				strings.TrimSpace(
					c.Query(
						"category",
					),
				),
			),

			Status: strings.ToLower(
				strings.TrimSpace(
					c.Query(
						"status",
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

			Scope: strings.ToLower(
				strings.TrimSpace(
					c.Query(
						"scope",
					),
				),
			),

			OrderID: strings.TrimSpace(
				c.Query(
					"order_id",
				),
			),

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

			StaffAccountID: strings.TrimSpace(
				c.Query(
					"staff_account_id",
				),
			),

			WarehouseID: strings.TrimSpace(
				c.Query(
					"warehouse_id",
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
			return FinanceExpenseFilter{},
				errors.Join(
					ErrInvalidFinanceExpense,
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
			return FinanceExpenseFilter{},
				errors.Join(
					ErrInvalidFinanceExpense,
					errors.New(
						"to must use YYYY-MM-DD",
					),
				)
		}

		filter.ToSet =
			true

		/*
			The HTTP `to` parameter is inclusive at business-date level.

			The repository uses an exclusive upper bound, so advance the
			local business date by one day before converting it to UTC.
		*/
		filter.To =
			to.AddDate(
				0,
				0,
				1,
			).UTC()
	}

	if err :=
		validateFinanceExpenseFilter(
			filter,
		); err != nil {

		return FinanceExpenseFilter{},
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

			return FinanceExpenseFilter{},
				errors.Join(
					ErrInvalidFinanceExpense,
					errors.New(
						"finance expense date range may not exceed 366 days",
					),
				)
		}
	}

	return filter,
		nil
}

func writeFinanceExpenseError(
	c *gin.Context,
	err error,
) bool {
	if err == nil {
		return false
	}

	switch {
	case errors.Is(
		err,
		ErrFinanceExpenseNotFound,
	):
		platformerrors.Write(
			c,
			platformerrors.NotFound(
				"FINANCE_EXPENSE_NOT_FOUND",
				"Finance expense not found",
			),
		)

	case errors.Is(
		err,
		ErrFinanceExpenseOrderNotFound,
	):
		platformerrors.Write(
			c,
			platformerrors.NotFound(
				"ORDER_NOT_FOUND",
				"Order not found",
			),
		)

	case errors.Is(
		err,
		ErrFinanceExpenseProductNotFound,
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
		ErrFinanceExpenseVariantNotFound,
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
		ErrFinanceExpenseStaffNotFound,
	):
		platformerrors.Write(
			c,
			platformerrors.NotFound(
				"STAFF_ACCOUNT_NOT_FOUND",
				"Staff account not found",
			),
		)

	case errors.Is(
		err,
		ErrFinanceExpenseWarehouseNotFound,
	):
		platformerrors.Write(
			c,
			platformerrors.NotFound(
				"WAREHOUSE_NOT_FOUND",
				"Warehouse not found",
			),
		)

	case errors.Is(
		err,
		ErrFinanceExpenseIdempotencyConflict,
	):
		platformerrors.Write(
			c,
			platformerrors.Conflict(
				"IDEMPOTENCY_KEY_CONFLICT",
				"Idempotency-Key was already used for a different finance expense request",
			),
		)

	case errors.Is(
		err,
		ErrInvalidFinanceExpense,
	):
		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_FINANCE_EXPENSE",
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
