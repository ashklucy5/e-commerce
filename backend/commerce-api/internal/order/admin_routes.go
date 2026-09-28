package order

import "github.com/gin-gonic/gin"

func RegisterAdminRoutes(
	admin *gin.RouterGroup,
	handler *AdminHandler,
) {
	admin.PATCH(
		"/orders/:order_id/fulfillment",
		handler.TransitionFulfillment,
	)
}
