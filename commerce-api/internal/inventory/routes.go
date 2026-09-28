package inventory

import "github.com/gin-gonic/gin"

func RegisterRoutes(
	admin *gin.RouterGroup,
	handler *Handler,
) {
	admin.GET(
		"/inventory",
		handler.List,
	)

	admin.GET(
		"/inventory/:variant_id",
		handler.Get,
	)

	admin.POST(
		"/inventory/:variant_id/adjust",
		handler.Adjust,
	)

	admin.PATCH(
		"/inventory/:variant_id",
		handler.Update,
	)

	admin.GET(
		"/inventory/:variant_id/movements",
		handler.Movements,
	)
}
