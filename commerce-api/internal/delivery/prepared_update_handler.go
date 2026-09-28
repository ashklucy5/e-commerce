package delivery

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *AdminHandler) UpdatePreparedShipment(
	c *gin.Context,
) {
	var request UpdatePreparedShipmentRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		writeDeliveryError(c, ErrInvalidInput)
		return
	}

	result, err := h.service.UpdatePreparedShipment(
		c.Request.Context(),
		c.Param("shipment_id"),
		request,
		h.actorID,
	)
	if err != nil {
		writeDeliveryError(c, err)
		return
	}

	c.JSON(
		http.StatusOK,
		gin.H{
			"data": result,
		},
	)
}
