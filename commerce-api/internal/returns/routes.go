package returns

import "github.com/gin-gonic/gin"

func RegisterRoutes(
	public *gin.RouterGroup,
	handler *Handler,
) {
	// Authenticated customer account history.
	// This group already has OptionalAuth; the handler requires
	// a resolved customer identity before querying any returns.
	public.GET(
		"/returns",
		handler.ListForCustomer,
	)

	// Customer or secure guest return creation.
	public.POST(
		"/orders/:order_id/returns",
		handler.Create,
	)

	// Customer or secure guest return detail routes.
	public.GET(
		"/returns/:return_id",
		handler.Get,
	)

	public.GET(
		"/returns/:return_id/timeline",
		handler.Timeline,
	)

	public.POST(
		"/returns/:return_id/cancel",
		handler.Cancel,
	)
}

func RegisterAdminRoutes(
	admin *gin.RouterGroup,
	handler *AdminHandler,
) {
	admin.POST(
		"/returns/:return_id/approve",
		handler.Approve,
	)

	admin.POST(
		"/returns/:return_id/reject",
		handler.Reject,
	)

	admin.POST(
		"/returns/:return_id/receive",
		handler.Receive,
	)

	admin.POST(
		"/returns/:return_id/inspect",
		handler.Inspect,
	)
}
