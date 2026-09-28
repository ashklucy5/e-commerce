package order

import (
	"errors"
	"net/http"

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

func (h *Handler) PlaceOrder(
	c *gin.Context,
) {
	customerID, _ :=
		auth.CustomerIDFromContext(
			c,
		)

	result, err :=
		h.service.PlaceOrder(
			c.Request.Context(),
			customerID,
			c.Param(
				"checkout_key",
			),
		)
	if err != nil {
		writeOrderError(
			c,
			err,
		)

		return
	}

	status :=
		http.StatusCreated

	if !result.Created {
		status =
			http.StatusOK
	}

	c.JSON(
		status,
		gin.H{
			"data": result.Order,
			"meta": gin.H{
				"created": result.Created,
			},
		},
	)
}

func (h *Handler) Get(
	c *gin.Context,
) {
	customerID, _ :=
		auth.CustomerIDFromContext(
			c,
		)

	result, err :=
		h.service.GetForCustomer(
			c.Request.Context(),
			customerID,
			c.Param(
				"order_id",
			),
		)
	if err != nil {
		writeOrderError(
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

func (h *Handler) Cancel(
	c *gin.Context,
) {
	var request CancelRequest

	if err :=
		c.ShouldBindJSON(
			&request,
		); err != nil {
		c.JSON(
			http.StatusBadRequest,
			gin.H{
				"error": gin.H{
					"code": "INVALID_CANCELLATION",

					"message": "Invalid request body",
				},
			},
		)

		return
	}

	customerID, _ :=
		auth.CustomerIDFromContext(
			c,
		)

	result, err :=
		h.service.CancelForCustomer(
			c.Request.Context(),
			customerID,
			c.Param(
				"order_id",
			),
			request,
		)
	if err != nil {
		writeOrderError(
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

func (h *Handler) Timeline(
	c *gin.Context,
) {
	customerID, _ :=
		auth.CustomerIDFromContext(
			c,
		)

	result, err :=
		h.service.TimelineForCustomer(
			c.Request.Context(),
			customerID,
			c.Param(
				"order_id",
			),
		)
	if err != nil {
		writeOrderError(
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

func (h *Handler) Invoice(
	c *gin.Context,
) {
	customerID, _ :=
		auth.CustomerIDFromContext(
			c,
		)

	result, err :=
		h.service.InvoiceForCustomer(
			c.Request.Context(),
			customerID,
			c.Param(
				"order_id",
			),
		)
	if err != nil {
		writeOrderError(
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

func writeOrderError(
	c *gin.Context,
	err error,
) {
	switch {
	case errors.Is(
		err,
		ErrInvalidCheckoutKey,
	),
		errors.Is(
			err,
			ErrInvalidOrderID,
		),
		errors.Is(
			err,
			ErrCustomerDetailsRequired,
		),
		errors.Is(
			err,
			ErrShippingDetailsRequired,
		),
		errors.Is(
			err,
			ErrInvalidPaymentMethod,
		):
		c.JSON(
			http.StatusBadRequest,
			gin.H{
				"error": gin.H{
					"code": "INVALID_ORDER_REQUEST",

					"message": err.Error(),
				},
			},
		)

	case errors.Is(
		err,
		ErrCancellationReasonRequired,
	):
		c.JSON(
			http.StatusBadRequest,
			gin.H{
				"error": gin.H{
					"code": "INVALID_CANCELLATION",

					"message": err.Error(),
				},
			},
		)

	case errors.Is(
		err,
		ErrCheckoutNotFound,
	):
		c.JSON(
			http.StatusNotFound,
			gin.H{
				"error": gin.H{
					"code": "CHECKOUT_NOT_FOUND",

					"message": "Checkout not found",
				},
			},
		)

	case errors.Is(
		err,
		ErrOrderNotFound,
	):
		c.JSON(
			http.StatusNotFound,
			gin.H{
				"error": gin.H{
					"code": "ORDER_NOT_FOUND",

					"message": "Order not found",
				},
			},
		)

	case errors.Is(
		err,
		ErrCheckoutExpired,
	),
		errors.Is(
			err,
			ErrPaymentWindowExpired,
		):
		c.JSON(
			http.StatusGone,
			gin.H{
				"error": gin.H{
					"code": "EXPIRED",

					"message": err.Error(),
				},
			},
		)

	case errors.Is(
		err,
		ErrCheckoutNotActive,
	),
		errors.Is(
			err,
			ErrCartNotActive,
		),
		errors.Is(
			err,
			ErrEmptyCheckout,
		),
		errors.Is(
			err,
			ErrItemUnavailable,
		),
		errors.Is(
			err,
			ErrBelowMinimumOrderQuantity,
		),
		errors.Is(
			err,
			ErrCheckoutChanged,
		),
		errors.Is(
			err,
			ErrInsufficientStock,
		),
		errors.Is(
			err,
			ErrInventoryReservationMismatch,
		),
		errors.Is(
			err,
			ErrOrderNotPendingPayment,
		),
		errors.Is(
			err,
			ErrCancellationNotAllowed,
		),
		errors.Is(
			err,
			ErrRefundRequired,
		),
		errors.Is(
			err,
			ErrMoneyOverflow,
		):
		c.JSON(
			http.StatusConflict,
			gin.H{
				"error": gin.H{
					"code": "ORDER_CONFLICT",

					"message": err.Error(),
				},
			},
		)

	default:
		c.JSON(
			http.StatusInternalServerError,
			gin.H{
				"error": gin.H{
					"code": "INTERNAL_ERROR",

					"message": "Unable to process order",
				},
			},
		)
	}
}
