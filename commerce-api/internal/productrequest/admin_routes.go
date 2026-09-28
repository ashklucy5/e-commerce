package productrequest

import (
	"github.com/gin-gonic/gin"

	admincore "project.local/commerce-api/internal/admin"
)

const (
	permissionSourcingOfferManage = "admin.sourcing.offer.manage"

	permissionSourcingFinalize = "admin.sourcing.finalize"
)

type AdminPermissionMiddleware func(
	permission string,
) gin.HandlerFunc

func RegisterAdminRoutes(
	group *gin.RouterGroup,
	handler *AdminHandler,
	requirePermission AdminPermissionMiddleware,
) {
	group.GET(
		"/product-requests",
		requirePermission(
			admincore.PermissionSourcingRead,
		),
		handler.List,
	)

	group.GET(
		"/product-requests/:id",
		requirePermission(
			admincore.PermissionSourcingRead,
		),
		handler.Get,
	)

	group.GET(
		"/product-requests/:id/messages",
		requirePermission(
			admincore.PermissionSourcingRead,
		),
		handler.ListMessages,
	)

	group.POST(
		"/product-requests/:id/messages",
		requirePermission(
			admincore.PermissionSourcingReview,
		),
		handler.AddMessage,
	)

	group.PATCH(
		"/product-requests/:id/status",
		requirePermission(
			admincore.PermissionSourcingReview,
		),
		handler.UpdateStatus,
	)

	group.GET(
		"/product-requests/:id/offers",
		requirePermission(
			admincore.PermissionSourcingRead,
		),
		handler.ListOffers,
	)

	group.POST(
		"/product-requests/:id/offers",
		requirePermission(
			permissionSourcingOfferManage,
		),
		handler.CreateOffer,
	)

	group.PATCH(
		"/product-requests/:id/offers/:offerId",
		requirePermission(
			permissionSourcingOfferManage,
		),
		handler.UpdateOffer,
	)

	group.POST(
		"/product-requests/:id/offers/:offerId/send",
		requirePermission(
			permissionSourcingOfferManage,
		),
		handler.SendOffer,
	)

	group.POST(
		"/product-requests/:id/offers/:offerId/finalize",
		requirePermission(
			permissionSourcingFinalize,
		),
		handler.FinalizeOffer,
	)

	group.GET(
		"/product-requests/:id/confirmation",
		requirePermission(
			admincore.PermissionSourcingRead,
		),
		handler.GetConfirmation,
	)
}
