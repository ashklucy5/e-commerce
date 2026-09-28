package warehouse

import "github.com/gin-gonic/gin"

func RegisterRoutes(
	admin *gin.RouterGroup,
	handler *Handler,
) {
	group := admin.Group(
		"/warehouse",
	)

	group.GET(
		"/locations",
		handler.ListWarehouses,
	)

	group.POST(
		"/locations",
		handler.CreateWarehouse,
	)

	group.POST(
		"/fulfillments",
		handler.CreateFulfillment,
	)

	group.GET(
		"/fulfillments/:fulfillment_id",
		handler.GetFulfillment,
	)

	group.GET(
		"/fulfillments/:fulfillment_id/events",
		handler.ListFulfillmentEvents,
	)

	group.GET(
		"/orders/:order_id/fulfillments",
		handler.ListOrderFulfillments,
	)

	group.POST(
		"/fulfillments/:fulfillment_id/start-picking",
		handler.StartPicking,
	)

	group.POST(
		"/fulfillments/:fulfillment_id/mark-packed",
		handler.MarkPacked,
	)

	group.POST(
		"/fulfillments/:fulfillment_id/ready-for-handoff",
		handler.MarkReadyForHandoff,
	)

	group.POST(
		"/fulfillments/:fulfillment_id/cancel",
		handler.CancelFulfillment,
	)

	group.POST(
		"/inbound-shipments",
		handler.CreateInboundShipment,
	)

	group.GET(
		"/inbound-shipments/:inbound_id",
		handler.GetInboundShipment,
	)

	group.PATCH(
		"/inbound-shipments/:inbound_id/status",
		handler.UpdateInboundShipmentStatus,
	)

	group.POST(
		"/handoffs",
		handler.RecordHandoff,
	)
}
