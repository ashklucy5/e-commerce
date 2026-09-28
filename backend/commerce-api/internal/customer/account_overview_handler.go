package customer

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handler) GetAccountOverview(
	c *gin.Context,
) {
	customerID, ok :=
		accountCustomerID(
			c,
		)
	if !ok {
		return
	}

	result, err :=
		h.service.GetAccountOverview(
			c.Request.Context(),
			customerID,
		)
	if err != nil {
		writeCustomerError(
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
