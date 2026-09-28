package search

import "github.com/gin-gonic/gin"

func RegisterRoutes(
	group *gin.RouterGroup,
	handler *Handler,
) {
	group.GET(
		"/search",
		handler.Search,
	)

	group.GET(
		"/search/products",
		handler.Browse,
	)

	group.POST(
		"/search/image",
		handler.SearchImage,
	)

	group.GET(
		"/search/suggestions",
		handler.Suggestions,
	)
}
