package checkout

import "github.com/gin-gonic/gin"

func RegisterRoutes(
	group *gin.RouterGroup,
	handler *Handler,
) {
	options :=
		group.Group(
			"/checkout-options",
		)

	options.GET(
		"/payment-methods",
		handler.PaymentOptions,
	)

	checkouts :=
		group.Group(
			"/checkouts",
		)

	checkouts.POST(
		"",
		handler.Start,
	)

	checkouts.GET(
		"/:checkout_key",
		handler.Get,
	)

	checkouts.GET(
		"/:checkout_key/delivery-methods",
		handler.DeliveryOptions,
	)

	checkouts.PATCH(
		"/:checkout_key",
		handler.Update,
	)

	checkouts.POST(
		"/:checkout_key/cancel",
		handler.Cancel,
	)
}
