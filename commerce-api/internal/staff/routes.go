package staff

import "github.com/gin-gonic/gin"

func RegisterRoutes(
	group *gin.RouterGroup,
	handler *Handler,
	service *Service,
) {
	authGroup :=
		group.Group(
			"/auth",
		)

	authGroup.POST(
		"/login",
		handler.Login,
	)

	authGroup.POST(
		"/refresh",
		handler.Refresh,
	)

	authGroup.POST(
		"/logout",
		RequireAuth(
			service,
		),
		handler.Logout,
	)
}
