package cart

import "github.com/gin-gonic/gin"

func RegisterRoutes(
	group *gin.RouterGroup,
	handler *Handler,
) {
	carts := group.Group(
		"/carts",
	)

	carts.POST(
		"",
		handler.Create,
	)

	carts.GET(
		"/:cart_key",
		handler.Get,
	)

	carts.POST(
		"/:cart_key/items",
		handler.AddItem,
	)

	carts.PATCH(
		"/:cart_key/items/:item_id",
		handler.UpdateItem,
	)

	carts.DELETE(
		"/:cart_key/items/:item_id",
		handler.RemoveItem,
	)
}
