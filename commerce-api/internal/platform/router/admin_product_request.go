package router

import (
	"github.com/gin-gonic/gin"

	platformmiddleware "project.local/commerce-api/internal/platform/middleware"
	"project.local/commerce-api/internal/productrequest"
)

func registerAdminProductRequestRoutes(
	admin *gin.RouterGroup,
	deps Dependencies,
) {
	repository :=
		productrequest.NewRepository(
			deps.DB,
		)

	service :=
		productrequest.NewAdminService(
			repository,
		)

	handler :=
		productrequest.NewAdminHandler(
			service,
		)

	productrequest.RegisterAdminRoutes(
		admin,
		handler,
		platformmiddleware.
			RequireAdminPermission,
	)
}
