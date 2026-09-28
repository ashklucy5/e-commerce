package cart

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
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

// Create handles:
//
//	POST /api/v1/carts
func (h *Handler) Create(
	c *gin.Context,
) {
	result, err :=
		h.service.Create(
			c.Request.Context(),
		)
	if err != nil {
		writeCartError(
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

// Get handles:
//
//	GET /api/v1/carts/:cart_key
func (h *Handler) Get(
	c *gin.Context,
) {
	result, err :=
		h.service.Get(
			c.Request.Context(),
			c.Param(
				"cart_key",
			),
		)
	if err != nil {
		writeCartError(
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

// AddItem handles:
//
//	POST /api/v1/carts/:cart_key/items
func (h *Handler) AddItem(
	c *gin.Context,
) {
	var request AddItemRequest

	if err := c.ShouldBindJSON(
		&request,
	); err != nil {
		c.JSON(
			http.StatusBadRequest,
			gin.H{
				"error": gin.H{
					"code":    "INVALID_REQUEST",
					"message": "Invalid cart item request",
				},
			},
		)

		return
	}

	result, err :=
		h.service.AddItem(
			c.Request.Context(),
			c.Param(
				"cart_key",
			),
			request,
		)
	if err != nil {
		writeCartError(
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

// UpdateItem handles:
//
//	PATCH /api/v1/carts/:cart_key/items/:item_id
func (h *Handler) UpdateItem(
	c *gin.Context,
) {
	var request UpdateItemRequest

	if err := c.ShouldBindJSON(
		&request,
	); err != nil {
		c.JSON(
			http.StatusBadRequest,
			gin.H{
				"error": gin.H{
					"code":    "INVALID_REQUEST",
					"message": "Invalid cart item request",
				},
			},
		)

		return
	}

	result, err :=
		h.service.UpdateItem(
			c.Request.Context(),
			c.Param(
				"cart_key",
			),
			c.Param(
				"item_id",
			),
			request,
		)
	if err != nil {
		writeCartError(
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

// RemoveItem handles:
//
//	DELETE /api/v1/carts/:cart_key/items/:item_id
func (h *Handler) RemoveItem(
	c *gin.Context,
) {
	result, err :=
		h.service.RemoveItem(
			c.Request.Context(),
			c.Param(
				"cart_key",
			),
			c.Param(
				"item_id",
			),
		)
	if err != nil {
		writeCartError(
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

func writeCartError(
	c *gin.Context,
	err error,
) {
	switch {
	case errors.Is(
		err,
		ErrInvalidCartKey,
	):
		writeCartJSONError(
			c,
			http.StatusBadRequest,
			"INVALID_CART_KEY",
			err.Error(),
		)

	case errors.Is(
		err,
		ErrInvalidVariantID,
	):
		writeCartJSONError(
			c,
			http.StatusBadRequest,
			"INVALID_VARIANT_ID",
			err.Error(),
		)

	case errors.Is(
		err,
		ErrInvalidItemID,
	):
		writeCartJSONError(
			c,
			http.StatusBadRequest,
			"INVALID_CART_ITEM_ID",
			err.Error(),
		)

	case errors.Is(
		err,
		ErrInvalidQuantity,
	):
		writeCartJSONError(
			c,
			http.StatusUnprocessableEntity,
			"INVALID_QUANTITY",
			err.Error(),
		)

	case errors.Is(
		err,
		ErrBelowMinimumOrderQuantity,
	):
		writeCartJSONError(
			c,
			http.StatusUnprocessableEntity,
			"BELOW_MINIMUM_ORDER_QUANTITY",
			err.Error(),
		)

	case errors.Is(
		err,
		ErrInvalidOrderIncrement,
	):
		writeCartJSONError(
			c,
			http.StatusUnprocessableEntity,
			"INVALID_ORDER_INCREMENT",
			err.Error(),
		)

	case errors.Is(
		err,
		ErrCartNotFound,
	):
		writeCartJSONError(
			c,
			http.StatusNotFound,
			"CART_NOT_FOUND",
			"Cart not found",
		)

	case errors.Is(
		err,
		ErrItemNotFound,
	):
		writeCartJSONError(
			c,
			http.StatusNotFound,
			"CART_ITEM_NOT_FOUND",
			"Cart item not found",
		)

	case errors.Is(
		err,
		ErrCartExpired,
	):
		writeCartJSONError(
			c,
			http.StatusGone,
			"CART_EXPIRED",
			"Cart has expired",
		)

	case errors.Is(
		err,
		ErrCartInactive,
	):
		writeCartJSONError(
			c,
			http.StatusConflict,
			"CART_INACTIVE",
			"Cart is no longer active",
		)

	case errors.Is(
		err,
		ErrVariantUnavailable,
	):
		writeCartJSONError(
			c,
			http.StatusConflict,
			"VARIANT_UNAVAILABLE",
			"Variant is unavailable",
		)

	case errors.Is(
		err,
		ErrInsufficientStock,
	):
		writeCartJSONError(
			c,
			http.StatusConflict,
			"INSUFFICIENT_STOCK",
			err.Error(),
		)

	case errors.Is(
		err,
		ErrCurrencyMismatch,
	):
		writeCartJSONError(
			c,
			http.StatusConflict,
			"CURRENCY_MISMATCH",
			"Variant currency does not match cart currency",
		)

	default:
		writeCartJSONError(
			c,
			http.StatusInternalServerError,
			"INTERNAL_ERROR",
			"Unable to process cart",
		)
	}
}

func writeCartJSONError(
	c *gin.Context,
	status int,
	code string,
	message string,
) {
	c.JSON(
		status,
		gin.H{
			"error": gin.H{
				"code":    code,
				"message": message,
			},
		},
	)
}
