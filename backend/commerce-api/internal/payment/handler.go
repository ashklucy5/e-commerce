package payment

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"project.local/commerce-api/internal/auth"
)

const guestCheckoutKeyHeader = "X-Checkout-Key"

type Handler struct {
	service *Service

	initiation *InitiationService
}

func NewHandler(
	service *Service,
) *Handler {
	return &Handler{
		service: service,
	}
}

func NewStatusHandler(
	repository *Repository,
) *Handler {
	return &Handler{
		service: &Service{
			repository: repository,
		},
	}
}

func (h *Handler) SetInitiationService(
	service *InitiationService,
) {
	h.initiation =
		service
}

func (h *Handler) GetOrderPaymentStatus(
	c *gin.Context,
) {
	customerID, _ :=
		auth.CustomerIDFromContext(
			c,
		)

	result, err :=
		h.service.GetOrderPaymentStatus(
			c.Request.Context(),
			customerID,
			c.Param(
				"order_id",
			),
			c.GetHeader(
				guestCheckoutKeyHeader,
			),
		)
	if err != nil {
		writePaymentHTTPError(
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

func (h *Handler) InitiateOrderPayment(
	c *gin.Context,
) {
	if h.initiation == nil {
		writePaymentHTTPError(
			c,
			errors.New(
				"payment initiation service unavailable",
			),
		)

		return
	}

	customerID, _ :=
		auth.CustomerIDFromContext(
			c,
		)

	result, err :=
		h.initiation.Initiate(
			c.Request.Context(),
			customerID,
			c.Param(
				"order_id",
			),
			c.GetHeader(
				guestCheckoutKeyHeader,
			),
		)
	if err != nil {
		writePaymentHTTPError(
			c,
			err,
		)

		return
	}

	status :=
		http.StatusCreated

	if result.Reused {
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

func writePaymentHTTPError(
	c *gin.Context,
	err error,
) {
	switch {
	case errors.Is(
		err,
		ErrInvalidInput,
	):
		c.JSON(
			http.StatusBadRequest,
			gin.H{
				"error": gin.H{
					"code": "INVALID_PAYMENT_REQUEST",

					"message": "Invalid payment request",
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
		ErrPaymentInitiationNotRequired,
	):
		c.JSON(
			http.StatusConflict,
			gin.H{
				"error": gin.H{
					"code": "PAYMENT_INITIATION_NOT_REQUIRED",

					"message": "This order does not require external payment initiation",
				},
			},
		)

	case errors.Is(
		err,
		ErrPaymentOrderNotPending,
	):
		c.JSON(
			http.StatusConflict,
			gin.H{
				"error": gin.H{
					"code": "ORDER_NOT_AWAITING_PAYMENT",

					"message": "Order is not awaiting payment",
				},
			},
		)

	case errors.Is(
		err,
		ErrPaymentWindowExpired,
	):
		c.JSON(
			http.StatusConflict,
			gin.H{
				"error": gin.H{
					"code": "PAYMENT_WINDOW_EXPIRED",

					"message": "Payment window has expired",
				},
			},
		)

	case errors.Is(
		err,
		ErrPaymentMethodUnavailable,
	):
		c.JSON(
			http.StatusConflict,
			gin.H{
				"error": gin.H{
					"code": "PAYMENT_METHOD_UNAVAILABLE",

					"message": "Selected payment method is currently unavailable",
				},
			},
		)

	case errors.Is(
		err,
		ErrPaymentProviderRejected,
	):
		c.JSON(
			http.StatusBadGateway,
			gin.H{
				"error": gin.H{
					"code": "PAYMENT_PROVIDER_REJECTED",

					"message": "Payment provider rejected the payment initiation",
				},
			},
		)

	case errors.Is(
		err,
		ErrPaymentProviderOutcomeUnknown,
	):
		c.JSON(
			http.StatusServiceUnavailable,
			gin.H{
				"error": gin.H{
					"code": "PAYMENT_PROVIDER_OUTCOME_UNKNOWN",

					"message": "Payment provider result is temporarily uncertain; retry using the same order",
				},
			},
		)

	case errors.Is(
		err,
		ErrReconciliationRequired,
	):
		c.JSON(
			http.StatusServiceUnavailable,
			gin.H{
				"error": gin.H{
					"code": "PAYMENT_RECONCILIATION_REQUIRED",

					"message": "Payment state requires reconciliation before another payment can be created",
				},
			},
		)

	default:
		log.Printf(
			"payment HTTP error: %v",
			err,
		)

		c.JSON(
			http.StatusInternalServerError,
			gin.H{
				"error": gin.H{
					"code": "INTERNAL_ERROR",

					"message": "Unable to process payment request",
				},
			},
		)
	}
}
