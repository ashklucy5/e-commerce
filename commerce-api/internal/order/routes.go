package order

import "github.com/gin-gonic/gin"

func RegisterRoutes(
	public *gin.RouterGroup,
	handler *Handler,
) {
	public.POST(
		"/checkouts/:checkout_key/place-order",
		handler.PlaceOrder,
	)

	public.GET(
		"/orders/:order_id",
		handler.Get,
	)

	public.POST(
		"/orders/:order_id/cancel",
		handler.Cancel,
	)

	public.GET(
		"/orders/:order_id/timeline",
		handler.Timeline,
	)

	public.GET(
		"/orders/:order_id/invoice",
		handler.Invoice,
	)
}
