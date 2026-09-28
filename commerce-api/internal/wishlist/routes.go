package wishlist

import "github.com/gin-gonic/gin"

func RegisterRoutes(
	group *gin.RouterGroup,
	handler *Handler,
	requireAuth gin.HandlerFunc,
) {
	wishlist :=
		group.Group(
			"/wishlist",
		)

	wishlist.Use(
		requireAuth,
	)

	wishlist.GET(
		"",
		handler.List,
	)

	wishlist.GET(
		"/count",
		handler.Count,
	)

	wishlist.GET(
		"/:product_id",
		handler.State,
	)

	wishlist.PUT(
		"/:product_id",
		handler.Add,
	)

	wishlist.DELETE(
		"/:product_id",
		handler.Remove,
	)
}
