package delivery

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"project.local/commerce-api/internal/auth"
)

type CustomerHandler struct {
	service *Service
}

func NewCustomerHandler(
	service *Service,
) *CustomerHandler {
	return &CustomerHandler{
		service: service,
	}
}

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

		actorID: normalizeActorID(
			actorID,
		),
	}
}

func (h *CustomerHandler) Tracking(
	c *gin.Context,
) {
	customerID, _ :=
		auth.CustomerIDFromContext(
			c,
		)

	result, err :=
		h.service.GetTrackingForCustomer(
			c.Request.Context(),
			customerID,
			c.Param(
				"order_id",
			),
		)
	if err != nil {
		writeDeliveryError(
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

func (h *CustomerHandler) ConfirmReceipt(
	c *gin.Context,
) {
	var request ConfirmReceiptRequest

	if c.Request.ContentLength != 0 {
		if err :=
			c.ShouldBindJSON(
				&request,
			); err != nil {
			writeDeliveryError(
				c,
				ErrInvalidInput,
			)

			return
		}
	}

	customerID, _ :=
		auth.CustomerIDFromContext(
			c,
		)

	result, err :=
		h.service.ConfirmReceiptForCustomer(
			c.Request.Context(),
			customerID,
			c.Param(
				"order_id",
			),
			request,
		)
	if err != nil {
		writeDeliveryError(
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

func (h *AdminHandler) PrepareShipment(
	c *gin.Context,
) {
	var request PrepareShipmentRequest

	if err :=
		c.ShouldBindJSON(
			&request,
		); err != nil {
		writeDeliveryError(
			c,
			ErrInvalidInput,
		)

		return
	}

	result, err :=
		h.service.PrepareShipment(
			c.Request.Context(),
			c.Param(
				"order_id",
			),
			request,
			h.actorID,
		)
	if err != nil {
		writeDeliveryError(
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

func (h *AdminHandler) DispatchShipment(
	c *gin.Context,
) {
	var request DispatchShipmentRequest

	if c.Request.ContentLength != 0 {
		if err :=
			c.ShouldBindJSON(
				&request,
			); err != nil {
			writeDeliveryError(
				c,
				ErrInvalidInput,
			)

			return
		}
	}

	result, err :=
		h.service.DispatchShipment(
			c.Request.Context(),
			c.Param(
				"shipment_id",
			),
			request,
			h.actorID,
		)
	if err != nil {
		writeDeliveryError(
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

func (h *AdminHandler) Tracking(
	c *gin.Context,
) {
	result, err :=
		h.service.GetTracking(
			c.Request.Context(),
			c.Param(
				"order_id",
			),
		)
	if err != nil {
		writeDeliveryError(
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

func (h *AdminHandler) AddTrackingEvent(
	c *gin.Context,
) {
	var request TrackingEventRequest

	if err :=
		c.ShouldBindJSON(
			&request,
		); err != nil {
		writeDeliveryError(
			c,
			ErrInvalidInput,
		)

		return
	}

	result, err :=
		h.service.AddTrackingEvent(
			c.Request.Context(),
			c.Param(
				"shipment_id",
			),
			request,
		)
	if err != nil {
		writeDeliveryError(
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

func (h *AdminHandler) MarkProviderDelivered(
	c *gin.Context,
) {
	var request ProviderDeliveredRequest

	if c.Request.ContentLength != 0 {
		if err :=
			c.ShouldBindJSON(
				&request,
			); err != nil {
			writeDeliveryError(
				c,
				ErrInvalidInput,
			)

			return
		}
	}

	result, err :=
		h.service.MarkProviderDelivered(
			c.Request.Context(),
			c.Param(
				"shipment_id",
			),
			request,
		)
	if err != nil {
		writeDeliveryError(
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

func (h *AdminHandler) ConfirmReceipt(
	c *gin.Context,
) {
	var request ConfirmReceiptRequest

	if c.Request.ContentLength != 0 {
		if err :=
			c.ShouldBindJSON(
				&request,
			); err != nil {
			writeDeliveryError(
				c,
				ErrInvalidInput,
			)

			return
		}
	}

	result, err :=
		h.service.ConfirmReceiptForAdmin(
			c.Request.Context(),
			c.Param(
				"order_id",
			),
			request,
			h.actorID,
		)
	if err != nil {
		writeDeliveryError(
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

func writeDeliveryError(
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
					"code": "INVALID_DELIVERY_REQUEST",

					"message": err.Error(),
				},
			},
		)

	case errors.Is(
		err,
		ErrCustomerAuthenticationRequired,
	):
		c.JSON(
			http.StatusUnauthorized,
			gin.H{
				"error": gin.H{
					"code": "AUTH_REQUIRED",

					"message": err.Error(),
				},
			},
		)

	case errors.Is(
		err,
		ErrOrderNotFound,
	),
		errors.Is(
			err,
			ErrCustomerOrderMismatch,
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
		ErrShipmentNotFound,
	):
		c.JSON(
			http.StatusNotFound,
			gin.H{
				"error": gin.H{
					"code": "SHIPMENT_NOT_FOUND",

					"message": "Shipment not found",
				},
			},
		)

	case errors.Is(
		err,
		ErrShipmentAlreadyExists,
	),
		errors.Is(
			err,
			ErrOrderNotReady,
		),
		errors.Is(
			err,
			ErrPaymentNotReady,
		),
		errors.Is(
			err,
			ErrWarehouseNotReady,
		),
		errors.Is(
			err,
			ErrMultipleWarehousesNotSupported,
		),
		errors.Is(
			err,
			ErrShipmentNotReady,
		),
		errors.Is(
			err,
			ErrHandoffIncomplete,
		),
		errors.Is(
			err,
			ErrDeliveryModeMismatch,
		):
		c.JSON(
			http.StatusConflict,
			gin.H{
				"error": gin.H{
					"code": "DELIVERY_CONFLICT",

					"message": err.Error(),
				},
			},
		)

	default:
		log.Printf(
			"delivery error: %v",
			err,
		)

		c.JSON(
			http.StatusInternalServerError,
			gin.H{
				"error": gin.H{
					"code": "INTERNAL_ERROR",

					"message": "Unable to process delivery request",
				},
			},
		)
	}
}
