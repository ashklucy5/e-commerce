package router

import (
	"github.com/gin-gonic/gin"

	"project.local/commerce-api/internal/auth"
	"project.local/commerce-api/internal/crm"
	"project.local/commerce-api/internal/supportattachment"
)

func registerCRMRoutes(
	engine *gin.Engine,
	deps Dependencies,
) {
	group := engine.Group(
		"/api/v1/crm",
	)

	authRepository := auth.NewRepository(
		deps.DB,
	)

	authService := auth.NewService(
		authRepository,
	)

	requireAuth :=
		auth.RequireAuth(
			authService,
		)

	crmRepository := crm.NewRepository(
		deps.DB,
	)

	crmService := crm.NewService(
		crmRepository,
	)

	crmHandler := crm.NewHandler(
		crmService,
	)

	attachmentRepository :=
		supportattachment.NewRepository(
			deps.DB,
		)

	attachmentService :=
		supportattachment.NewService(
			attachmentRepository,
			deps.Storage,
		)

	attachmentHandler :=
		crm.NewAttachmentHandler(
			crmRepository,
			attachmentService,
		)

	crm.RegisterRoutes(
		group,
		crmHandler,
		requireAuth,
	)

	crm.RegisterAttachmentRoutes(
		group,
		attachmentHandler,
		requireAuth,
	)
}
