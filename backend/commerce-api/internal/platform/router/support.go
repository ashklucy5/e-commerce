package router

import (
	"github.com/gin-gonic/gin"

	"project.local/commerce-api/internal/staff"
	"project.local/commerce-api/internal/support"
	"project.local/commerce-api/internal/supportattachment"
)

func registerSupportRoutes(
	engine *gin.Engine,
	deps Dependencies,
) {
	group :=
		engine.Group(
			"/api/v1/support",
		)

	staffRepository :=
		staff.NewRepository(
			deps.DB,
		)

	staffService :=
		staff.NewService(
			staffRepository,
		)

	staffHandler :=
		staff.NewHandler(
			staffService,
		)

	staff.RegisterRoutes(
		group,
		staffHandler,
		staffService,
	)

	supportRepository :=
		support.NewRepository(
			deps.DB,
		)

	supportService :=
		support.NewService(
			supportRepository,
		)

	supportHandler :=
		support.NewHandler(
			supportService,
		)

	support.RegisterRoutes(
		group,
		supportHandler,
		staffService,
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
		support.NewAttachmentHandler(
			supportService,
			attachmentService,
		)

	support.RegisterAttachmentRoutes(
		group,
		attachmentHandler,
		staffService,
	)
}
