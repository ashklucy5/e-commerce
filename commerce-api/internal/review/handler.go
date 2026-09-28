package review

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"project.local/commerce-api/internal/auth"
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

func (h *Handler) Create(
	c *gin.Context,
) {
	customerID, ok :=
		auth.CustomerIDFromContext(
			c,
		)
	if !ok {
		writeReviewJSONError(
			c,
			http.StatusUnauthorized,
			"INVALID_ACCESS_TOKEN",
			"Invalid or expired access token",
		)
		return
	}

	var request CreateRequest

	if err :=
		c.ShouldBindJSON(
			&request,
		); err != nil {
		writeReviewError(
			c,
			ErrInvalidRequest,
		)
		return
	}

	result, err :=
		h.service.Create(
			c.Request.Context(),
			customerID,
			request,
		)
	if err != nil {
		writeReviewError(
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

func (h *Handler) ListMine(
	c *gin.Context,
) {
	customerID, ok :=
		auth.CustomerIDFromContext(
			c,
		)
	if !ok {
		writeReviewJSONError(
			c,
			http.StatusUnauthorized,
			"INVALID_ACCESS_TOKEN",
			"Invalid or expired access token",
		)
		return
	}

	limit, err :=
		reviewQueryInt(
			c.Query("limit"),
		)
	if err != nil {
		writeReviewError(
			c,
			ErrInvalidRequest,
		)
		return
	}

	offset, err :=
		reviewQueryInt(
			c.Query("offset"),
		)
	if err != nil {
		writeReviewError(
			c,
			ErrInvalidRequest,
		)
		return
	}

	items,
		resolvedLimit,
		resolvedOffset,
		err :=
		h.service.ListMine(
			c.Request.Context(),
			customerID,
			limit,
			offset,
		)
	if err != nil {
		writeReviewError(
			c,
			err,
		)
		return
	}

	c.JSON(
		http.StatusOK,
		gin.H{
			"data": items,
			"meta": gin.H{
				"limit": resolvedLimit,

				"offset": resolvedOffset,
			},
		},
	)
}

func (h *Handler) Update(
	c *gin.Context,
) {
	customerID, ok :=
		auth.CustomerIDFromContext(
			c,
		)
	if !ok {
		writeReviewJSONError(
			c,
			http.StatusUnauthorized,
			"INVALID_ACCESS_TOKEN",
			"Invalid or expired access token",
		)
		return
	}

	var request UpdateRequest

	if err :=
		c.ShouldBindJSON(
			&request,
		); err != nil {
		writeReviewError(
			c,
			ErrInvalidRequest,
		)
		return
	}

	result, err :=
		h.service.Update(
			c.Request.Context(),
			customerID,
			c.Param("id"),
			request,
		)
	if err != nil {
		writeReviewError(
			c,
			err,
		)
		return
	}

	c.JSON(
		http.StatusOK,
		gin.H{
			"data": result,
		},
	)
}

func (h *Handler) Delete(
	c *gin.Context,
) {
	customerID, ok :=
		auth.CustomerIDFromContext(
			c,
		)
	if !ok {
		writeReviewJSONError(
			c,
			http.StatusUnauthorized,
			"INVALID_ACCESS_TOKEN",
			"Invalid or expired access token",
		)
		return
	}

	err :=
		h.service.Delete(
			c.Request.Context(),
			customerID,
			c.Param("id"),
		)
	if err != nil {
		writeReviewError(
			c,
			err,
		)
		return
	}

	c.Status(
		http.StatusNoContent,
	)
}

func (h *Handler) ListProduct(
	c *gin.Context,
) {
	limit, err :=
		reviewQueryInt(
			c.Query("limit"),
		)
	if err != nil {
		writeReviewError(
			c,
			ErrInvalidRequest,
		)
		return
	}

	offset, err :=
		reviewQueryInt(
			c.Query("offset"),
		)
	if err != nil {
		writeReviewError(
			c,
			ErrInvalidRequest,
		)
		return
	}

	items,
		resolvedLimit,
		resolvedOffset,
		err :=
		h.service.ListProduct(
			c.Request.Context(),
			c.Param("productID"),
			limit,
			offset,
		)
	if err != nil {
		writeReviewError(
			c,
			err,
		)
		return
	}

	c.JSON(
		http.StatusOK,
		gin.H{
			"data": items,
			"meta": gin.H{
				"limit": resolvedLimit,

				"offset": resolvedOffset,
			},
		},
	)
}

func (h *Handler) Summary(
	c *gin.Context,
) {
	result, err :=
		h.service.Summary(
			c.Request.Context(),
			c.Param("productID"),
		)
	if err != nil {
		writeReviewError(
			c,
			err,
		)
		return
	}

	c.JSON(
		http.StatusOK,
		gin.H{
			"data": result,
		},
	)
}

func reviewQueryInt(
	value string,
) (int, error) {
	value =
		strings.TrimSpace(
			value,
		)

	if value == "" {
		return 0, nil
	}

	return strconv.Atoi(
		value,
	)
}

func writeReviewError(
	c *gin.Context,
	err error,
) {
	switch {
	case errors.Is(
		err,
		ErrInvalidRequest,
	):
		writeReviewJSONError(
			c,
			http.StatusBadRequest,
			"INVALID_REVIEW_REQUEST",
			"Review request is invalid",
		)

	case errors.Is(
		err,
		ErrOrderItemNotFound,
	):
		writeReviewJSONError(
			c,
			http.StatusNotFound,
			"REVIEW_ORDER_ITEM_NOT_FOUND",
			"Order item was not found for this customer",
		)

	case errors.Is(
		err,
		ErrOrderNotEligible,
	):
		writeReviewJSONError(
			c,
			http.StatusConflict,
			"REVIEW_ORDER_NOT_ELIGIBLE",
			"Order item can be reviewed only after the order is delivered or completed",
		)

	case errors.Is(
		err,
		ErrAlreadyExists,
	):
		writeReviewJSONError(
			c,
			http.StatusConflict,
			"REVIEW_ALREADY_EXISTS",
			"An active review already exists for this order item",
		)

	case errors.Is(
		err,
		ErrReviewNotFound,
	):
		writeReviewJSONError(
			c,
			http.StatusNotFound,
			"REVIEW_NOT_FOUND",
			"Review was not found",
		)

	case errors.Is(
		err,
		ErrProductNotFound,
	):
		writeReviewJSONError(
			c,
			http.StatusNotFound,
			"REVIEW_PRODUCT_NOT_FOUND",
			"Product was not found",
		)

	default:
		writeReviewJSONError(
			c,
			http.StatusInternalServerError,
			"REVIEW_FAILED",
			"Unable to process review request",
		)
	}
}

func writeReviewJSONError(
	c *gin.Context,
	status int,
	code string,
	message string,
) {
	c.JSON(
		status,
		gin.H{
			"error": gin.H{
				"code": code,

				"message": message,
			},
		},
	)
}
