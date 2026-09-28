package catalogmediawrite

import "github.com/gin-gonic/gin"

func RegisterRoutes(
	admin *gin.RouterGroup,
	handler *Handler,
) {
	admin.POST(
		"/products/:product_id/images",
		handler.CreateImage,
	)

	admin.POST(
		"/products/:product_id/360-frames",
		handler.Create360Frame,
	)

	admin.PUT(
		"/products/:product_id/3d-model",
		handler.Upsert3DModel,
	)
}
