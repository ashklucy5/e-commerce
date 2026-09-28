package order

import "github.com/gin-gonic/gin"

// RegisterAccountRoutes registers authenticated customer account order routes.
//
// These routes are intentionally separate from RegisterRoutes because
// individual order access supports both authenticated customers and secure
// guest checkout-key access, while order history is customer-account only.
func RegisterAccountRoutes(
	group *gin.RouterGroup,
	handler *Handler,
	middleware ...gin.HandlerFunc,
) {
	orders :=
		group.Group(
			"/orders",
		)

	if len(middleware) > 0 {
		orders.Use(
			middleware...,
		)
	}

	orders.GET(
		"",
		handler.List,
	)
}
