package catalogwrite

import "github.com/gin-gonic/gin"

func RegisterRoutes(
	admin *gin.RouterGroup,
	handler *Handler,
) {
	admin.GET(
		"/categories/tree",
		handler.CategoryTree,
	)

	admin.POST(
		"/categories",
		handler.CreateCategory,
	)

	admin.POST(
		"/products",
		handler.CreateProduct,
	)
}
