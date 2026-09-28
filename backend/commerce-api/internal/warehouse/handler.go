package warehouse

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
	actorID string
}

func NewHandler(
	service *Service,
	actorID string,
) *Handler {
	return &Handler{
		service: service,
		actorID: normalizeActorID(
			actorID,
		),
	}
}

func (h *Handler) ListWarehouses(
	c *gin.Context,
) {
	items, err :=
		h.service.ListWarehouses(
			c.Request.Context(),
		)
	if err != nil {
		writeWarehouseError(
			c,
			err,
		)
		return
	}

	c.JSON(
		http.StatusOK,
		gin.H{
			"data": items,
		},
	)
}

func (h *Handler) CreateWarehouse(
	c *gin.Context,
) {
	var request CreateWarehouseRequest

	if err := c.ShouldBindJSON(
		&request,
	); err != nil {
		writeWarehouseError(
			c,
			ErrInvalidInput,
		)
		return
	}

	item, err :=
		h.service.CreateWarehouse(
			c.Request.Context(),
			request,
		)
	if err != nil {
		writeWarehouseError(
			c,
			err,
		)
		return
	}

	c.JSON(
		http.StatusCreated,
		gin.H{
			"data": item,
		},
	)
}

func (h *Handler) CreateFulfillment(
	c *gin.Context,
) {
	var request CreateFulfillmentRequest

	if err := c.ShouldBindJSON(
		&request,
	); err != nil {
		writeWarehouseError(
			c,
			ErrInvalidInput,
		)
		return
	}

	item, err :=
		h.service.CreateFulfillment(
			c.Request.Context(),
			request,
			h.actorID,
		)
	if err != nil {
		writeWarehouseError(
			c,
			err,
		)
		return
	}

	c.JSON(
		http.StatusCreated,
		gin.H{
			"data": item,
		},
	)
}

func (h *Handler) GetFulfillment(
	c *gin.Context,
) {
	item, err :=
		h.service.GetFulfillment(
			c.Request.Context(),
			c.Param(
				"fulfillment_id",
			),
		)
	if err != nil {
		writeWarehouseError(
			c,
			err,
		)
		return
	}

	c.JSON(
		http.StatusOK,
		gin.H{
			"data": item,
		},
	)
}

func (h *Handler) ListOrderFulfillments(
	c *gin.Context,
) {
	items, err :=
		h.service.ListOrderFulfillments(
			c.Request.Context(),
			c.Param(
				"order_id",
			),
		)
	if err != nil {
		writeWarehouseError(
			c,
			err,
		)
		return
	}

	c.JSON(
		http.StatusOK,
		gin.H{
			"data": items,
		},
	)
}

func (h *Handler) ListFulfillmentEvents(
	c *gin.Context,
) {
	items, err :=
		h.service.ListFulfillmentEvents(
			c.Request.Context(),
			c.Param(
				"fulfillment_id",
			),
		)
	if err != nil {
		writeWarehouseError(
			c,
			err,
		)
		return
	}

	c.JSON(
		http.StatusOK,
		gin.H{
			"data": items,
		},
	)
}

func (h *Handler) StartPicking(
	c *gin.Context,
) {
	request, ok :=
		bindOptionalWarehouseAction(
			c,
		)
	if !ok {
		return
	}

	item, err :=
		h.service.StartPicking(
			c.Request.Context(),
			c.Param(
				"fulfillment_id",
			),
			request,
			h.actorID,
		)
	if err != nil {
		writeWarehouseError(
			c,
			err,
		)
		return
	}

	c.JSON(
		http.StatusOK,
		gin.H{
			"data": item,
		},
	)
}

func (h *Handler) MarkPacked(
	c *gin.Context,
) {
	request, ok :=
		bindOptionalWarehouseAction(
			c,
		)
	if !ok {
		return
	}

	item, err :=
		h.service.MarkPacked(
			c.Request.Context(),
			c.Param(
				"fulfillment_id",
			),
			request,
			h.actorID,
		)
	if err != nil {
		writeWarehouseError(
			c,
			err,
		)
		return
	}

	c.JSON(
		http.StatusOK,
		gin.H{
			"data": item,
		},
	)
}

func (h *Handler) MarkReadyForHandoff(
	c *gin.Context,
) {
	request, ok :=
		bindOptionalWarehouseAction(
			c,
		)
	if !ok {
		return
	}

	item, err :=
		h.service.MarkReadyForHandoff(
			c.Request.Context(),
			c.Param(
				"fulfillment_id",
			),
			request,
			h.actorID,
		)
	if err != nil {
		writeWarehouseError(
			c,
			err,
		)
		return
	}

	c.JSON(
		http.StatusOK,
		gin.H{
			"data": item,
		},
	)
}

func (h *Handler) CancelFulfillment(
	c *gin.Context,
) {
	request, ok :=
		bindOptionalWarehouseAction(
			c,
		)
	if !ok {
		return
	}

	item, err :=
		h.service.CancelFulfillment(
			c.Request.Context(),
			c.Param(
				"fulfillment_id",
			),
			request,
			h.actorID,
		)
	if err != nil {
		writeWarehouseError(
			c,
			err,
		)
		return
	}

	c.JSON(
		http.StatusOK,
		gin.H{
			"data": item,
		},
	)
}

func (h *Handler) CreateInboundShipment(
	c *gin.Context,
) {
	var request CreateInboundShipmentRequest

	if err := c.ShouldBindJSON(
		&request,
	); err != nil {
		writeWarehouseError(
			c,
			ErrInvalidInput,
		)
		return
	}

	item, err :=
		h.service.CreateInboundShipment(
			c.Request.Context(),
			request,
			h.actorID,
		)
	if err != nil {
		writeWarehouseError(
			c,
			err,
		)
		return
	}

	c.JSON(
		http.StatusCreated,
		gin.H{
			"data": item,
		},
	)
}

func (h *Handler) GetInboundShipment(
	c *gin.Context,
) {
	item, err :=
		h.service.GetInboundShipment(
			c.Request.Context(),
			c.Param(
				"inbound_id",
			),
		)
	if err != nil {
		writeWarehouseError(
			c,
			err,
		)
		return
	}

	c.JSON(
		http.StatusOK,
		gin.H{
			"data": item,
		},
	)
}

func (h *Handler) UpdateInboundShipmentStatus(
	c *gin.Context,
) {
	var request UpdateInboundShipmentStatusRequest

	if err := c.ShouldBindJSON(
		&request,
	); err != nil {
		writeWarehouseError(
			c,
			ErrInvalidInput,
		)
		return
	}

	item, err :=
		h.service.UpdateInboundShipmentStatus(
			c.Request.Context(),
			c.Param(
				"inbound_id",
			),
			request,
			h.actorID,
		)
	if err != nil {
		writeWarehouseError(
			c,
			err,
		)
		return
	}

	c.JSON(
		http.StatusOK,
		gin.H{
			"data": item,
		},
	)
}

func (h *Handler) RecordHandoff(
	c *gin.Context,
) {
	var request CreateHandoffRequest

	if err := c.ShouldBindJSON(
		&request,
	); err != nil {
		writeWarehouseError(
			c,
			ErrInvalidInput,
		)
		return
	}

	result, err :=
		h.service.RecordHandoff(
			c.Request.Context(),
			request,
			h.actorID,
		)
	if err != nil {
		writeWarehouseError(
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

func bindOptionalWarehouseAction(
	c *gin.Context,
) (WarehouseActionRequest, bool) {
	var request WarehouseActionRequest

	if c.Request.ContentLength == 0 {
		return request, true
	}

	if err := c.ShouldBindJSON(
		&request,
	); err != nil {
		writeWarehouseError(
			c,
			ErrInvalidInput,
		)

		return WarehouseActionRequest{},
			false
	}

	return request, true
}

func writeWarehouseError(
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
			ErrInboundShipmentRequired,
		):
		writeWarehouseJSONError(
			c,
			http.StatusBadRequest,
			"INVALID_WAREHOUSE_REQUEST",
			err.Error(),
		)

	case errors.Is(
		err,
		ErrWarehouseNotFound,
	):
		writeWarehouseJSONError(
			c,
			http.StatusNotFound,
			"WAREHOUSE_NOT_FOUND",
			"Warehouse was not found",
		)

	case errors.Is(
		err,
		ErrOrderNotFound,
	):
		writeWarehouseJSONError(
			c,
			http.StatusNotFound,
			"ORDER_NOT_FOUND",
			"Order was not found",
		)

	case errors.Is(
		err,
		ErrOrderItemNotFound,
	):
		writeWarehouseJSONError(
			c,
			http.StatusNotFound,
			"ORDER_ITEM_NOT_FOUND",
			"Order item was not found",
		)

	case errors.Is(
		err,
		ErrInboundShipmentNotFound,
	):
		writeWarehouseJSONError(
			c,
			http.StatusNotFound,
			"INBOUND_SHIPMENT_NOT_FOUND",
			"Inbound shipment was not found",
		)

	case errors.Is(
		err,
		ErrFulfillmentNotFound,
	):
		writeWarehouseJSONError(
			c,
			http.StatusNotFound,
			"FULFILLMENT_NOT_FOUND",
			"Warehouse fulfillment was not found",
		)

	case errors.Is(
		err,
		ErrShipmentNotFound,
	):
		writeWarehouseJSONError(
			c,
			http.StatusNotFound,
			"SHIPMENT_NOT_FOUND",
			"Shipment was not found",
		)

	case errors.Is(
		err,
		ErrWarehouseCodeExists,
	):
		writeWarehouseJSONError(
			c,
			http.StatusConflict,
			"WAREHOUSE_CODE_EXISTS",
			err.Error(),
		)

	case errors.Is(
		err,
		ErrInboundReferenceExists,
	):
		writeWarehouseJSONError(
			c,
			http.StatusConflict,
			"INBOUND_REFERENCE_EXISTS",
			err.Error(),
		)

	case errors.Is(
		err,
		ErrWarehouseInactive,
	),
		errors.Is(
			err,
			ErrOrderNotFulfillable,
		),
		errors.Is(
			err,
			ErrAllocationExceeded,
		),
		errors.Is(
			err,
			ErrInboundWarehouseMismatch,
		),
		errors.Is(
			err,
			ErrInboundSourceMismatch,
		),
		errors.Is(
			err,
			ErrInboundTransitionNotAllowed,
		),
		errors.Is(
			err,
			ErrFulfillmentTransitionNotAllowed,
		),
		errors.Is(
			err,
			ErrFulfillmentNotReadyForHandoff,
		),
		errors.Is(
			err,
			ErrFulfillmentQuantityExceeded,
		),
		errors.Is(
			err,
			ErrShipmentNotReadyForHandoff,
		),
		errors.Is(
			err,
			ErrShipmentOrderMismatch,
		),
		errors.Is(
			err,
			ErrShipmentWarehouseMismatch,
		),
		errors.Is(
			err,
			ErrHandoffTypeMismatch,
		),
		errors.Is(
			err,
			ErrSelfPickupNotAllowed,
		):
		writeWarehouseJSONError(
			c,
			http.StatusConflict,
			"WAREHOUSE_CONFLICT",
			err.Error(),
		)

	default:
		log.Printf(
			"warehouse error: %v",
			err,
		)

		writeWarehouseJSONError(
			c,
			http.StatusInternalServerError,
			"INTERNAL_ERROR",
			"Unable to process warehouse request",
		)
	}
}

func writeWarehouseJSONError(
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
