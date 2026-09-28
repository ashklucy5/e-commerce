package warehouse

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handler) UpdateInboundShipmentJourneyStatus(
	c *gin.Context,
) {
	var request UpdateInboundJourneyStatusRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		writeWarehouseError(
			c,
			ErrInvalidInput,
		)
		return
	}

	item, err :=
		h.service.UpdateInboundShipmentJourneyStatus(
			c.Request.Context(),
			c.Param("inbound_id"),
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

func (h *Handler) SetWarehouseMapLocation(
	c *gin.Context,
) {
	var request WarehouseMapLocationRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		writeWarehouseError(
			c,
			ErrInvalidInput,
		)
		return
	}

	item, err :=
		h.service.SetWarehouseMapLocation(
			c.Request.Context(),
			c.Param("warehouse_id"),
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
		http.StatusOK,
		gin.H{
			"data": item,
		},
	)
}
