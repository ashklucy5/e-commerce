package crm

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"project.local/commerce-api/internal/auth"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) CreateCase(c *gin.Context) {
	customerID, ok := auth.CustomerIDFromContext(c)
	if !ok {
		writeError(c, auth.ErrInvalidAccessToken)
		return
	}

	var request CreateCaseRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, ErrInvalidRequest)
		return
	}

	result, err := h.service.CreateCase(
		c.Request.Context(),
		customerID,
		request,
	)
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": result})
}

func (h *Handler) ListCases(c *gin.Context) {
	customerID, ok := auth.CustomerIDFromContext(c)
	if !ok {
		writeError(c, auth.ErrInvalidAccessToken)
		return
	}

	limit, err := queryInt(c.Query("limit"))
	if err != nil {
		writeError(c, ErrInvalidRequest)
		return
	}

	offset, err := queryInt(c.Query("offset"))
	if err != nil {
		writeError(c, ErrInvalidRequest)
		return
	}

	result, err := h.service.ListCases(
		c.Request.Context(),
		customerID,
		limit,
		offset,
	)
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": result.Items,
		"meta": gin.H{
			"limit":  result.Limit,
			"offset": result.Offset,
		},
	})
}

func (h *Handler) GetCase(c *gin.Context) {
	customerID, ok := auth.CustomerIDFromContext(c)
	if !ok {
		writeError(c, auth.ErrInvalidAccessToken)
		return
	}

	result, err := h.service.GetCase(
		c.Request.Context(),
		customerID,
		c.Param("id"),
	)
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (h *Handler) ListMessages(c *gin.Context) {
	customerID, ok := auth.CustomerIDFromContext(c)
	if !ok {
		writeError(c, auth.ErrInvalidAccessToken)
		return
	}

	limit, err := queryInt(c.Query("limit"))
	if err != nil {
		writeError(c, ErrInvalidRequest)
		return
	}

	offset, err := queryInt(c.Query("offset"))
	if err != nil {
		writeError(c, ErrInvalidRequest)
		return
	}

	result, err := h.service.ListMessages(
		c.Request.Context(),
		customerID,
		c.Param("id"),
		limit,
		offset,
	)
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": result.Items,
		"meta": gin.H{
			"limit":  result.Limit,
			"offset": result.Offset,
		},
	})
}

func (h *Handler) AddMessage(c *gin.Context) {
	customerID, ok := auth.CustomerIDFromContext(c)
	if !ok {
		writeError(c, auth.ErrInvalidAccessToken)
		return
	}

	var request AddMessageRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, ErrInvalidRequest)
		return
	}

	result, err := h.service.AddCustomerMessage(
		c.Request.Context(),
		customerID,
		c.Param("id"),
		request,
	)
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": result})
}

func queryInt(value string) (int, error) {
	if value == "" {
		return 0, nil
	}

	return strconv.Atoi(value)
}

func writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, auth.ErrInvalidAccessToken):
		writeJSONError(
			c,
			http.StatusUnauthorized,
			"INVALID_ACCESS_TOKEN",
			"Invalid or expired access token",
		)

	case errors.Is(err, ErrInvalidRequest),
		errors.Is(err, ErrInvalidCaseType),
		errors.Is(err, ErrInvalidRequestedQuantity):
		writeJSONError(
			c,
			http.StatusBadRequest,
			"INVALID_CRM_REQUEST",
			"Invalid customer service request",
		)

	case errors.Is(err, ErrProductNotFound):
		writeJSONError(
			c,
			http.StatusNotFound,
			"CRM_PRODUCT_NOT_FOUND",
			"Product was not found",
		)

	case errors.Is(err, ErrVariantNotFound):
		writeJSONError(
			c,
			http.StatusNotFound,
			"CRM_VARIANT_NOT_FOUND",
			"Product variant was not found",
		)

	case errors.Is(err, ErrOrderNotFound):
		writeJSONError(
			c,
			http.StatusNotFound,
			"CRM_ORDER_NOT_FOUND",
			"Order was not found",
		)

	case errors.Is(err, ErrCaseNotFound):
		writeJSONError(
			c,
			http.StatusNotFound,
			"CRM_CASE_NOT_FOUND",
			"Customer service case was not found",
		)

	case errors.Is(err, ErrContextMismatch):
		writeJSONError(
			c,
			http.StatusBadRequest,
			"CRM_CONTEXT_MISMATCH",
			"Product and variant do not match",
		)

	case errors.Is(err, ErrBulkStockAvailable):
		writeJSONError(
			c,
			http.StatusConflict,
			"BULK_STOCK_AVAILABLE",
			"Requested quantity does not exceed current available stock",
		)

	case errors.Is(err, ErrCaseClosed):
		writeJSONError(
			c,
			http.StatusConflict,
			"CRM_CASE_CLOSED",
			"Closed customer service cases cannot receive new messages",
		)

	default:
		writeJSONError(
			c,
			http.StatusInternalServerError,
			"CRM_FAILED",
			"Unable to process customer service request",
		)
	}
}

func writeJSONError(
	c *gin.Context,
	status int,
	code string,
	message string,
) {
	c.JSON(status, gin.H{
		"error": gin.H{
			"code":    code,
			"message": message,
		},
	})
}
