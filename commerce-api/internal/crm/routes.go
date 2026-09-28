package crm

import "github.com/gin-gonic/gin"

func RegisterRoutes(
	group *gin.RouterGroup,
	handler *Handler,
	requireAuth gin.HandlerFunc,
) {
	cases := group.Group("/cases")
	cases.Use(requireAuth)

	cases.POST("", handler.CreateCase)
	cases.GET("", handler.ListCases)
	cases.GET("/:id", handler.GetCase)

	cases.GET(
		"/:id/messages",
		handler.ListMessages,
	)

	cases.POST(
		"/:id/messages",
		handler.AddMessage,
	)
}
