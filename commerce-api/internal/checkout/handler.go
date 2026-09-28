package checkout

import (
	"errors"
	"log"
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

func (h *Handler) Start(
	c *gin.Context,
) {
	var request StartRequest

	if err := c.ShouldBindJSON(
		&request,
	); err != nil {
		writeCheckoutJSONError(
			c,
			http.StatusBadRequest,
			"INVALID_REQUEST",
			"Invalid checkout request",
		)

		return
	}

	customerID, _ :=
		auth.CustomerIDFromContext(
			c,
		)

	result, err :=
		h.service.Start(
			c.Request.Context(),
			customerID,
			request,
		)
	if err != nil {
		writeCheckoutError(
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
				"checkout_key",
			),
		)
	if err != nil {
		writeCheckoutError(
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

func (h *Handler) Update(
	c *gin.Context,
) {
	var request UpdateRequest

	if err := c.ShouldBindJSON(
		&request,
	); err != nil {
		writeCheckoutJSONError(
			c,
			http.StatusBadRequest,
			"INVALID_REQUEST",
			"Invalid checkout update",
		)

		return
	}

	customerID, _ :=
		auth.CustomerIDFromContext(
			c,
		)

	result, err :=
		h.service.UpdateForCustomer(
			c.Request.Context(),
			customerID,
			c.Param(
				"checkout_key",
			),
			request,
		)
	if err != nil {
		writeCheckoutError(
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
	customerID, _ :=
		auth.CustomerIDFromContext(
			c,
		)

	result, err :=
		h.service.CancelForCustomer(
			c.Request.Context(),
			customerID,
			c.Param(
				"checkout_key",
			),
		)
	if err != nil {
		writeCheckoutError(
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

func (h *Handler) PaymentOptions(
	c *gin.Context,
) {
	customerID, _ :=
		auth.CustomerIDFromContext(
			c,
		)

	result, err :=
		h.service.PaymentOptionsForCustomer(
			c.Request.Context(),
			customerID,
			c.Query(
				"checkout_key",
			),
		)
	if err != nil {
		writeCheckoutError(
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

func writeCheckoutError(
	c *gin.Context,
	err error,
) {
	switch {
	case errors.Is(
		err,
		ErrInvalidCartKey,
	):
		writeCheckoutJSONError(
			c,
			http.StatusBadRequest,
			"INVALID_CART_KEY",
			"Invalid cart key",
		)

	case errors.Is(
		err,
		ErrInvalidCheckoutKey,
	):
		writeCheckoutJSONError(
			c,
			http.StatusBadRequest,
			"INVALID_CHECKOUT_KEY",
			"Invalid checkout key",
		)

	case errors.Is(
		err,
		ErrCartNotFound,
	):
		writeCheckoutJSONError(
			c,
			http.StatusNotFound,
			"CART_NOT_FOUND",
			"Cart not found",
		)

	case errors.Is(
		err,
		ErrCheckoutNotFound,
	):
		writeCheckoutJSONError(
			c,
			http.StatusNotFound,
			"CHECKOUT_NOT_FOUND",
			"Checkout not found",
		)

	case errors.Is(
		err,
		ErrCartInactive,
	):
		writeCheckoutJSONError(
			c,
			http.StatusConflict,
			"CART_INACTIVE",
			"Cart is no longer active",
		)

	case errors.Is(
		err,
		ErrCartExpired,
	):
		writeCheckoutJSONError(
			c,
			http.StatusGone,
			"CART_EXPIRED",
			"Cart has expired",
		)

	case errors.Is(
		err,
		ErrEmptyCart,
	):
		writeCheckoutJSONError(
			c,
			http.StatusUnprocessableEntity,
			"EMPTY_CART",
			"Cart is empty",
		)

	case errors.Is(
		err,
		ErrBelowMinimumOrderQuantity,
	):
		writeCheckoutJSONError(
			c,
			http.StatusUnprocessableEntity,
			"BELOW_MINIMUM_ORDER_QUANTITY",
			err.Error(),
		)

	case errors.Is(
		err,
		ErrCartItemUnavailable,
	):
		writeCheckoutJSONError(
			c,
			http.StatusConflict,
			"CART_ITEM_UNAVAILABLE",
			err.Error(),
		)

	case errors.Is(
		err,
		ErrInsufficientStock,
	):
		writeCheckoutJSONError(
			c,
			http.StatusConflict,
			"INSUFFICIENT_STOCK",
			err.Error(),
		)

	case errors.Is(
		err,
		ErrCurrencyMismatch,
	):
		writeCheckoutJSONError(
			c,
			http.StatusConflict,
			"CURRENCY_MISMATCH",
			"Cart contains incompatible currency",
		)

	case errors.Is(
		err,
		ErrCheckoutNotActive,
	):
		writeCheckoutJSONError(
			c,
			http.StatusConflict,
			"CHECKOUT_NOT_ACTIVE",
			"Checkout is not active",
		)

	case errors.Is(
		err,
		ErrInvalidCustomerDetails,
	):
		writeCheckoutJSONError(
			c,
			http.StatusUnprocessableEntity,
			"INVALID_CUSTOMER_DETAILS",
			"Invalid customer details",
		)

	case errors.Is(
		err,
		ErrInvalidShippingDetails,
	):
		writeCheckoutJSONError(
			c,
			http.StatusUnprocessableEntity,
			"INVALID_SHIPPING_DETAILS",
			"Invalid shipping details",
		)

	case errors.Is(
		err,
		ErrInvalidPaymentMethod,
	):
		writeCheckoutJSONError(
			c,
			http.StatusUnprocessableEntity,
			"INVALID_PAYMENT_METHOD",
			"Unsupported payment method",
		)

	case errors.Is(
		err,
		ErrInvalidDeliveryMethod,
	):
		writeCheckoutJSONError(
			c,
			http.StatusUnprocessableEntity,
			"INVALID_DELIVERY_METHOD",
			"Invalid delivery method",
		)

	case errors.Is(
		err,
		ErrMoneyOverflow,
	):
		writeCheckoutJSONError(
			c,
			http.StatusUnprocessableEntity,
			"INVALID_CHECKOUT_TOTAL",
			"Checkout total exceeds supported amount",
		)

	case errors.Is(
		err,
		ErrInvalidPromotionCode,
	):
		writeCheckoutJSONError(
			c,
			http.StatusUnprocessableEntity,
			"INVALID_PROMOTION_CODE",
			"Invalid promotion code",
		)

	case errors.Is(
		err,
		ErrPromotionNotFound,
	):
		writeCheckoutJSONError(
			c,
			http.StatusUnprocessableEntity,
			"PROMOTION_NOT_FOUND",
			"Promotion code was not found",
		)

	case errors.Is(
		err,
		ErrPromotionNotApplicable,
	):
		writeCheckoutJSONError(
			c,
			http.StatusUnprocessableEntity,
			"PROMOTION_NOT_APPLICABLE",
			"Promotion is not applicable to this checkout",
		)

	case errors.Is(
		err,
		ErrPromotionUnavailable,
	):
		writeCheckoutJSONError(
			c,
			http.StatusServiceUnavailable,
			"PROMOTION_UNAVAILABLE",
			"Promotion service is unavailable",
		)

	default:
		log.Printf(
			"checkout error: %v",
			err,
		)
		writeCheckoutJSONError(
			c,
			http.StatusInternalServerError,
			"INTERNAL_ERROR",
			"Unable to process checkout",
		)
	}
}

func writeCheckoutJSONError(
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
