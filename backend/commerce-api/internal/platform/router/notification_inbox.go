package router

import (
	"github.com/gin-gonic/gin"

	admincore "project.local/commerce-api/internal/admin"
	"project.local/commerce-api/internal/notification"
	platformmiddleware "project.local/commerce-api/internal/platform/middleware"
)

func registerCustomerNotificationRoutes(
	root *gin.RouterGroup,
	deps Dependencies,
	requireAuth gin.HandlerFunc,
) {
	repository :=
		notification.NewRepository(
			deps.DB,
		)

	service :=
		notification.NewService(
			repository,
		)

	handler :=
		notification.NewInboxHandler(
			service,
		)

	notification.RegisterCustomerInboxRoutes(
		root,
		handler,
		requireAuth,
	)
}

func registerAdminNotificationRoutes(
	admin *gin.RouterGroup,
	deps Dependencies,
) {
	repository :=
		notification.NewRepository(
			deps.DB,
		)

	service :=
		notification.NewService(
			repository,
		)

	handler :=
		notification.NewInboxHandler(
			service,
		)

	routes :=
		admin.Group(
			"",
		)

	routes.Use(
		platformmiddleware.
			RequireAdminPermission(
				admincore.
					PermissionNotificationRead,
			),
	)

	notification.RegisterAdminInboxRoutes(
		routes,
		handler,
	)
}
