package support

import (
	"github.com/gin-gonic/gin"

	"project.local/commerce-api/internal/staff"
)

func RegisterRoutes(
	group *gin.RouterGroup,
	handler *Handler,
	staffService *staff.Service,
) {
	protected :=
		group.Group("")

	protected.Use(
		staff.RequireAuth(
			staffService,
		),
	)

	protected.Use(
		staff.RequirePermission(
			PermissionPanelAccess,
		),
	)

	protected.GET(
		"/me",
		handler.Me,
	)

	protected.GET(
		"/queues",
		staff.RequirePermission(
			PermissionQueueRead,
		),
		handler.Queues,
	)

	protected.PATCH(
		"/presence",
		staff.RequirePermission(
			PermissionPresenceUpdate,
		),
		handler.UpdatePresence,
	)

	protected.GET(
		"/cases",
		staff.RequirePermission(
			PermissionCaseRead,
		),
		handler.ListCases,
	)

	protected.GET(
		"/cases/:id",
		staff.RequirePermission(
			PermissionCaseRead,
		),
		handler.GetCase,
	)

	protected.GET(
		"/cases/:id/messages",
		staff.RequirePermission(
			PermissionCaseRead,
		),
		handler.ListMessages,
	)

	protected.POST(
		"/cases/:id/claim",
		staff.RequirePermission(
			PermissionCaseClaim,
		),
		handler.ClaimCase,
	)

	protected.POST(
		"/cases/:id/messages",
		staff.RequirePermission(
			PermissionCaseReply,
		),
		handler.Reply,
	)

	protected.POST(
		"/cases/:id/resolve",
		staff.RequirePermission(
			PermissionCaseResolve,
		),
		handler.Resolve,
	)

	protected.POST(
		"/cases/:id/close",
		staff.RequirePermission(
			PermissionCaseResolve,
		),
		handler.Close,
	)

	protected.POST(
		"/cases/:id/escalate",
		staff.RequirePermission(
			PermissionCaseEscalate,
		),
		handler.Escalate,
	)
}
