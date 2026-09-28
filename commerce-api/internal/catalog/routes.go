package catalog

import "github.com/gin-gonic/gin"

// RegisterRoutes registers public customer-facing catalog routes.
func RegisterRoutes(
	group *gin.RouterGroup,
	handler *Handler,
) {
	products :=
		group.Group(
			"/products",
		)

	products.GET(
		"",
		handler.ListActiveProducts,
	)

	// Must stay above /:slug so "home-feed"
	// cannot be interpreted as a product slug.
	products.GET(
		"/home-feed",
		handler.ListHomeProducts,
	)

	products.GET(
		"/:slug/immersive/360",
		handler.GetProduct360Frames,
	)

	products.GET(
		"/:slug",
		handler.GetActiveProductBySlug,
	)
}
