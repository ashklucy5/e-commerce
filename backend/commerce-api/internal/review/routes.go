package review

import "github.com/gin-gonic/gin"

func RegisterRoutes(
	group *gin.RouterGroup,
	handler *Handler,
	requireAuth gin.HandlerFunc,
) {
	// Public product review endpoints.
	group.GET(
		"/reviews/products/:productID/summary",
		handler.Summary,
	)

	group.GET(
		"/reviews/products/:productID",
		handler.ListProduct,
	)

	// Authenticated customer review endpoints.
	protected :=
		group.Group(
			"/reviews",
		)

	protected.Use(
		requireAuth,
	)

	protected.POST(
		"",
		handler.Create,
	)

	protected.GET(
		"/mine",
		handler.ListMine,
	)

	protected.PATCH(
		"/:id",
		handler.Update,
	)

	protected.DELETE(
		"/:id",
		handler.Delete,
	)
}
