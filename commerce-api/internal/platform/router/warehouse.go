package router

import (
	"github.com/gin-gonic/gin"

	admincore "project.local/commerce-api/internal/admin"
	platformmiddleware "project.local/commerce-api/internal/platform/middleware"
	warehouseops "project.local/commerce-api/internal/warehouse"
)

func registerAdminWarehouseRoutes(
	admin *gin.RouterGroup,
	deps Dependencies,
) {
	repository :=
		warehouseops.NewRepository(
			deps.DB,
		)

	service :=
		warehouseops.NewService(
			repository,
		)

	factory :=
		func(
			actorID string,
		) *warehouseops.Handler {
			return warehouseops.NewHandler(
				service,
				actorID,
			)
		}

	group :=
		admin.Group(
			"/warehouse",
		)

	requireRead :=
		platformmiddleware.
			RequireAdminPermission(
				admincore.
					PermissionWarehouseRead,
			)

	requireManage :=
		platformmiddleware.
			RequireAdminPermission(
				admincore.
					PermissionWarehouseManage,
			)

	group.GET(
		"/locations",
		requireRead,
		adminActorHandler(
			factory,
			(*warehouseops.Handler).
				ListWarehouses,
		),
	)

	group.POST(
		"/locations",
		requireManage,
		adminActorHandler(
			factory,
			(*warehouseops.Handler).
				CreateWarehouse,
		),
	)

	group.PATCH(
		"/locations/:warehouse_id/map-location",
		requireManage,
		adminActorHandler(
			factory,
			(*warehouseops.Handler).
				SetWarehouseMapLocation,
		),
	)

	group.POST(
		"/fulfillments",
		requireManage,
		adminActorHandler(
			factory,
			(*warehouseops.Handler).
				CreateFulfillment,
		),
	)

	group.GET(
		"/fulfillments/:fulfillment_id",
		requireRead,
		adminActorHandler(
			factory,
			(*warehouseops.Handler).
				GetFulfillment,
		),
	)

	group.GET(
		"/fulfillments/:fulfillment_id/events",
		requireRead,
		adminActorHandler(
			factory,
			(*warehouseops.Handler).
				ListFulfillmentEvents,
		),
	)

	group.GET(
		"/orders/:order_id/fulfillments",
		requireRead,
		adminActorHandler(
			factory,
			(*warehouseops.Handler).
				ListOrderFulfillments,
		),
	)

	group.POST(
		"/fulfillments/:fulfillment_id/start-picking",
		requireManage,
		adminActorHandler(
			factory,
			(*warehouseops.Handler).
				StartPicking,
		),
	)

	group.POST(
		"/fulfillments/:fulfillment_id/mark-packed",
		requireManage,
		adminActorHandler(
			factory,
			(*warehouseops.Handler).
				MarkPacked,
		),
	)

	group.POST(
		"/fulfillments/:fulfillment_id/ready-for-handoff",
		requireManage,
		adminActorHandler(
			factory,
			(*warehouseops.Handler).
				MarkReadyForHandoff,
		),
	)

	group.POST(
		"/fulfillments/:fulfillment_id/cancel",
		requireManage,
		adminActorHandler(
			factory,
			(*warehouseops.Handler).
				CancelFulfillment,
		),
	)

	group.POST(
		"/inbound-shipments",
		requireManage,
		adminActorHandler(
			factory,
			(*warehouseops.Handler).
				CreateInboundShipment,
		),
	)

	group.GET(
		"/inbound-shipments/:inbound_id",
		requireRead,
		adminActorHandler(
			factory,
			(*warehouseops.Handler).
				GetInboundShipment,
		),
	)

	group.PATCH(
		"/inbound-shipments/:inbound_id/status",
		requireManage,
		adminActorHandler(
			factory,
			(*warehouseops.Handler).
				UpdateInboundShipmentJourneyStatus,
		),
	)

	group.POST(
		"/handoffs",
		requireManage,
		adminActorHandler(
			factory,
			(*warehouseops.Handler).
				RecordHandoff,
		),
	)
}
