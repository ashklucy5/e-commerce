package payment

import "github.com/gin-gonic/gin"

func RegisterRoutes(
	group *gin.RouterGroup,
	handler *Handler,
) {
	group.GET(
		"/orders/:order_id/payment",
		handler.GetOrderPaymentStatus,
	)

	group.POST(
		"/orders/:order_id/payment/initiate",
		handler.InitiateOrderPayment,
	)
}
