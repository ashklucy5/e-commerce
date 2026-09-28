package checkout

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"project.local/commerce-api/internal/auth"
)

func (h *Handler) DeliveryOptions(
	c *gin.Context,
) {
	customerID, _ :=
		auth.CustomerIDFromContext(
			c,
		)

	result, err :=
		h.service.DeliveryOptionsForCustomer(
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
