package adminupload

import (
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(
	admin *gin.RouterGroup,
	handler *Handler,
) {
	admin.POST(
		"/uploads",
		handler.CreateUpload,
	)
}
