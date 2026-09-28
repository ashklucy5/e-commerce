package delivery

import "github.com/gin-gonic/gin"

func RegisterCustomerRoutes(
	public *gin.RouterGroup,
	handler *CustomerHandler,
	requireAuth gin.HandlerFunc,
) {
	/*
		Registered-customer or secure guest tracking.

		This route works throughout the full journey, including
		before a Bangladesh delivery shipment exists.
	*/
	public.GET(
		"/orders/:order_id/tracking",
		handler.TrackingForViewer,
	)

	/*
		Receipt confirmation remains authenticated only.

		Provider/courier "delivered" is deliberately not the final
		customer-delivered state until receipt confirmation.
	*/
	customer :=
		public.Group(
			"",
		)

	customer.Use(
		requireAuth,
	)

	customer.POST(
		"/orders/:order_id/confirm-receipt",
		handler.ConfirmReceipt,
	)
}

func RegisterAdminRoutes(
	admin *gin.RouterGroup,
	handler *AdminHandler,
) {
	group :=
		admin.Group(
			"/delivery",
		)

	group.POST(
		"/orders/:order_id/shipments",
		handler.PrepareShipment,
	)

	group.GET(
		"/orders/:order_id/tracking",
		handler.Tracking,
	)

	group.POST(
		"/shipments/:shipment_id/dispatch",
		handler.DispatchShipment,
	)

	/*
		Manual/provider-safe delivery updates now support:

		- public customer message
		- country/city/location
		- latitude/longitude
		- visibility control
		- private metadata kept away from customers
	*/
	group.POST(
		"/shipments/:shipment_id/tracking-events",
		handler.AddJourneyTrackingEvent,
	)

	group.POST(
		"/shipments/:shipment_id/provider-delivered",
		handler.MarkProviderDelivered,
	)

	group.POST(
		"/orders/:order_id/confirm-receipt",
		handler.ConfirmReceipt,
	)
}
