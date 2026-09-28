package recommendation

import "github.com/gin-gonic/gin"

func RegisterRoutes(
	group *gin.RouterGroup,
	handler *Handler,
	rateLimit gin.HandlerFunc,
) {
	routes := group.Group("/recommendations")
	if rateLimit != nil {
		routes.Use(rateLimit)
	}
	routes.POST("/events", handler.RecordEvent)
}
