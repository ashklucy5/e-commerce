package delivery

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"project.local/commerce-api/internal/auth"
)

// TrackingForViewer handles registered-customer or secure guest
// access to the unified China -> Bangladesh delivery journey.
//
// Registered orders require the owning customer's access token.
// Guest orders require the original X-Checkout-Key capability.
func (h *CustomerHandler) TrackingForViewer(
	c *gin.Context,
) {
	customerID, _ :=
		auth.CustomerIDFromContext(
			c,
		)

	result, err :=
		h.service.GetJourneyTrackingForViewer(
			c.Request.Context(),
			customerID,
			c.GetHeader(
				GuestCheckoutKeyHeader,
			),
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

	c.Header(
		"Cache-Control",
		"private, no-store",
	)

	c.JSON(
		http.StatusOK,
		gin.H{
			"data": result,
		},
	)
}
