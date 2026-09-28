package router

import (
	"github.com/gin-gonic/gin"

	"project.local/commerce-api/internal/auth"
	"project.local/commerce-api/internal/wishlist"
)

func registerWishlistRoutes(
	engine *gin.Engine,
	deps Dependencies,
) {
	group :=
		engine.Group(
			"/api/v1",
		)

	authRepository :=
		auth.NewRepository(
			deps.DB,
		)

	authService :=
		auth.NewService(
			authRepository,
		)

	repository :=
		wishlist.NewRepository(
			deps.DB,
		)

	service :=
		wishlist.NewService(
			repository,
		)

	handler :=
		wishlist.NewHandler(
			service,
		)

	wishlist.RegisterRoutes(
		group,
		handler,
		auth.RequireAuth(
			authService,
		),
	)
}
