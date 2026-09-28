package catalogimport

import (
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(
	admin *gin.RouterGroup,
	handler *Handler,
) {
	admin.GET(
		"/catalog-imports",
		handler.List,
	)

	admin.POST(
		"/catalog-imports",
		handler.Create,
	)

	admin.GET(
		"/catalog-imports/:id",
		handler.Preview,
	)

	admin.POST(
		"/catalog-imports/:id/apply",
		handler.Apply,
	)
}
