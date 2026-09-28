package refund

import "github.com/gin-gonic/gin"

func RegisterRoutes(
	public *gin.RouterGroup,
	handler *Handler,
) {
	public.GET(
		"/refunds/:refund_id",
		handler.Get,
	)

	public.GET(
		"/refunds/:refund_id/timeline",
		handler.Timeline,
	)
}

func RegisterAdminRoutes(
	admin *gin.RouterGroup,
	handler *AdminHandler,
) {
	admin.POST(
		"/returns/:return_id/refunds",
		handler.RequestForReturn,
	)

	admin.POST(
		"/refunds/:refund_id/approve",
		handler.Approve,
	)

	admin.POST(
		"/refunds/:refund_id/process",
		handler.StartProcessing,
	)

	admin.POST(
		"/refunds/:refund_id/succeed",
		handler.Succeed,
	)

	admin.POST(
		"/refunds/:refund_id/fail",
		handler.Fail,
	)
}
