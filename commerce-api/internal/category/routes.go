package category

import "github.com/gin-gonic/gin"

// RegisterRoutes registers the public category endpoints.
func RegisterRoutes(
	group *gin.RouterGroup,
	handler *Handler,
) {
	categories := group.Group(
		"/categories",
	)

	categories.GET(
		"",
		handler.ListActive,
	)

	categories.GET(
		"/tree",
		handler.Tree,
	)

	categories.GET(
		"/:slug",
		handler.GetActiveBySlug,
	)
}
