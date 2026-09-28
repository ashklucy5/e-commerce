package router

import (
	"github.com/gin-gonic/gin"

	"project.local/commerce-api/internal/auth"
	"project.local/commerce-api/internal/review"
)

func registerReviewRoutes(
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

	reviewRepository :=
		review.NewRepository(
			deps.DB,
		)

	reviewService :=
		review.NewService(
			reviewRepository,
		)

	reviewHandler :=
		review.NewHandler(
			reviewService,
		)

	review.RegisterRoutes(
		group,
		reviewHandler,
		auth.RequireAuth(
			authService,
		),
	)
}
