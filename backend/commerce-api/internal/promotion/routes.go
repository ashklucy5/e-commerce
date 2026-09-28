package promotion

import "github.com/gin-gonic/gin"

func RegisterPublicRoutes(
	group *gin.RouterGroup,
	handler *Handler,
) {
	group.GET(
		"/promotions/active",
		handler.Active,
	)

	group.GET(
		"/promotions/flash-sales/active",
		handler.ActiveFlashSales,
	)
}
