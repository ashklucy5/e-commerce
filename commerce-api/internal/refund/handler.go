package refund

import (
	"errors"
	"log"
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

func (h *Handler) Get(
	c *gin.Context,
) {
	result, err :=
		h.service.Get(
			c.Request.Context(),
			c.Param(
				"refund_id",
			),
		)
	if err != nil {
		writeError(
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
	result, err :=
		h.service.Timeline(
			c.Request.Context(),
			c.Param(
				"refund_id",
			),
		)
	if err != nil {
		writeError(
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

func writeError(
	c *gin.Context,
	err error,
) {
	switch {
	case errors.Is(
		err,
		ErrInvalidInput,
	),
		errors.Is(
			err,
			ErrProviderRefundIDRequired,
		),
		errors.Is(
			err,
			ErrFailureDetailsRequired,
		):
		c.JSON(
			http.StatusBadRequest,
			gin.H{
				"error": gin.H{
					"code": "INVALID_REFUND",

					"message": err.Error(),
				},
			},
		)

	case errors.Is(
		err,
		ErrRefundNotFound,
	),
		errors.Is(
			err,
			ErrReturnNotFound,
		),
		errors.Is(
			err,
			ErrOrderNotFound,
		),
		errors.Is(
			err,
			ErrPaymentNotFound,
		):
		c.JSON(
			http.StatusNotFound,
			gin.H{
				"error": gin.H{
					"code": "REFUND_NOT_FOUND",

					"message": err.Error(),
				},
			},
		)

	case errors.Is(
		err,
		ErrReturnNotReady,
	),
		errors.Is(
			err,
			ErrRefundNotAllowed,
		),
		errors.Is(
			err,
			ErrNothingToRefund,
		),
		errors.Is(
			err,
			ErrRefundAmountExceeded,
		),
		errors.Is(
			err,
			ErrInvalidRefundTransition,
		):
		c.JSON(
			http.StatusConflict,
			gin.H{
				"error": gin.H{
					"code": "REFUND_CONFLICT",

					"message": err.Error(),
				},
			},
		)

	default:
		log.Printf(
			"refund internal error: %v",
			err,
		)

		c.JSON(
			http.StatusInternalServerError,
			gin.H{
				"error": gin.H{
					"code": "INTERNAL_ERROR",

					"message": "Unable to process refund",
				},
			},
		)
	}
}
