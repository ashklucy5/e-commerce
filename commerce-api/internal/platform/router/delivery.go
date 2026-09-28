package router

import (
	"github.com/gin-gonic/gin"

	admincore "project.local/commerce-api/internal/admin"
	"project.local/commerce-api/internal/auth"
	"project.local/commerce-api/internal/delivery"
	platformmiddleware "project.local/commerce-api/internal/platform/middleware"
)

func registerDeliveryRoutes(
	engine *gin.Engine,
	deps Dependencies,
) {
	repository :=
		delivery.NewRepository(
			deps.DB,
		)

	service :=
		delivery.NewService(
			repository,
		)

	authRepository :=
		auth.NewRepository(
			deps.DB,
		)

	authService :=
		auth.NewService(
			authRepository,
		)

	public :=
		engine.Group(
			"/api/v1",
		)

	customerAware :=
		public.Group(
			"",
		)

	customerAware.Use(
		auth.OptionalAuth(
			authService,
		),
	)

	customerHandler :=
		delivery.NewCustomerHandler(
			service,
		)

	delivery.RegisterCustomerRoutes(
		customerAware,
		customerHandler,
		auth.RequireAuth(
			authService,
		),
	)
}

func registerAdminDeliveryRoutes(
	admin *gin.RouterGroup,
	deps Dependencies,
) {
	repository :=
		delivery.NewRepository(
			deps.DB,
		)

	service :=
		delivery.NewService(
			repository,
		)

	factory :=
		func(
			actorID string,
		) *delivery.AdminHandler {
			return delivery.NewAdminHandler(
				service,
				actorID,
			)
		}

	group :=
		admin.Group(
			"/delivery",
		)

	requireRead :=
		platformmiddleware.
			RequireAdminPermission(
				admincore.
					PermissionDeliveryRead,
			)

	requireManage :=
		platformmiddleware.
			RequireAdminPermission(
				admincore.
					PermissionDeliveryManage,
			)

	requireConfirm :=
		platformmiddleware.
			RequireAdminPermission(
				admincore.
					PermissionDeliveryConfirmReceipt,
			)

	group.GET(
		"/shipments",
		requireRead,
		adminActorHandler(
			factory,
			(*delivery.AdminHandler).
				ListShipments,
		),
	)

	group.GET(
		"/shipments/summary",
		requireRead,
		adminActorHandler(
			factory,
			(*delivery.AdminHandler).
				ShipmentSummary,
		),
	)

	group.POST(
		"/orders/:order_id/shipments",
		requireManage,
		adminActorHandler(
			factory,
			(*delivery.AdminHandler).
				PrepareShipment,
		),
	)

	group.PATCH(
		"/shipments/:shipment_id",
		requireManage,
		adminActorHandler(
			factory,
			(*delivery.AdminHandler).
				UpdatePreparedShipment,
		),
	)

	group.GET(
		"/orders/:order_id/tracking",
		requireRead,
		adminActorHandler(
			factory,
			(*delivery.AdminHandler).
				Tracking,
		),
	)

	group.POST(
		"/shipments/:shipment_id/dispatch",
		requireManage,
		adminActorHandler(
			factory,
			(*delivery.AdminHandler).
				DispatchShipment,
		),
	)

	group.POST(
		"/shipments/:shipment_id/tracking-events",
		requireManage,
		adminActorHandler(
			factory,
			(*delivery.AdminHandler).
				AddJourneyTrackingEvent,
		),
	)

	group.POST(
		"/shipments/:shipment_id/provider-delivered",
		requireManage,
		adminActorHandler(
			factory,
			(*delivery.AdminHandler).
				MarkProviderDelivered,
		),
	)

	group.POST(
		"/orders/:order_id/confirm-receipt",
		requireConfirm,
		adminActorHandler(
			factory,
			(*delivery.AdminHandler).
				ConfirmReceipt,
		),
	)
}
