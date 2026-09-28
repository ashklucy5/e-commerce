package order

import (
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type AdminHandler struct {
	service *Service
	actorID string
}

func NewAdminHandler(
	service *Service,
	actorID string,
) *AdminHandler {
	return &AdminHandler{
		service: service,
		actorID: strings.TrimSpace(
			actorID,
		),
	}
}

func (h *AdminHandler) TransitionFulfillment(
	c *gin.Context,
) {
	var request FulfillmentRequest

	if err :=
		c.ShouldBindJSON(
			&request,
		); err != nil {
		c.JSON(
			http.StatusBadRequest,
			gin.H{
				"error": gin.H{
					"code": "INVALID_FULFILLMENT",

					"message": "Invalid request body",
				},
			},
		)

		return
	}

	result, err :=
		h.service.TransitionFulfillment(
			c.Request.Context(),
			c.Param(
				"order_id",
			),
			request,
			h.actorID,
		)
	if err != nil {
		writeAdminOrderError(
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

func writeAdminOrderError(
	c *gin.Context,
	err error,
) {
	switch {
	case errors.Is(
		err,
		ErrInvalidOrderID,
	),
		errors.Is(
			err,
			ErrInvalidFulfillmentStatus,
		),
		errors.Is(
			err,
			ErrCourierNameRequired,
		),
		errors.Is(
			err,
			ErrInvalidFulfillmentDetails,
		):
		c.JSON(
			http.StatusBadRequest,
			gin.H{
				"error": gin.H{
					"code": "INVALID_FULFILLMENT",

					"message": err.Error(),
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
		ErrInsufficientStock,
	),
		errors.Is(
			err,
			ErrFulfillmentTransitionNotAllowed,
		),
		errors.Is(
			err,
			ErrPaymentNotReadyForFulfillment,
		),
		errors.Is(
			err,
			ErrShipmentNotFound,
		):
		c.JSON(
			http.StatusConflict,
			gin.H{
				"error": gin.H{
					"code": "FULFILLMENT_CONFLICT",

					"message": err.Error(),
				},
			},
		)

	default:
		log.Printf(
			"order fulfillment error: %v",
			err,
		)

		c.JSON(
			http.StatusInternalServerError,
			gin.H{
				"error": gin.H{
					"code": "INTERNAL_ERROR",

					"message": "Unable to update order fulfillment",
				},
			},
		)
	}
}
