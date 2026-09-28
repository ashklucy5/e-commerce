package productrequest

import "github.com/gin-gonic/gin"

func RegisterRoutes(
	group *gin.RouterGroup,
	handler *Handler,
	requireAuth gin.HandlerFunc,
) {
	group.Use(
		requireAuth,
	)

	group.POST(
		"",
		handler.Create,
	)

	group.GET(
		"",
		handler.List,
	)

	group.GET(
		"/:id",
		handler.Get,
	)

	group.GET(
		"/:id/messages",
		handler.ListMessages,
	)

	group.POST(
		"/:id/messages",
		handler.AddMessage,
	)

	group.GET(
		"/:id/offers",
		handler.ListOffers,
	)

	group.GET(
		"/:id/offers/:offerId",
		handler.GetOffer,
	)

	group.POST(
		"/:id/offers/:offerId/accept",
		handler.AcceptOffer,
	)

	group.POST(
		"/:id/offers/:offerId/reject",
		handler.RejectOffer,
	)

	group.GET(
		"/:id/confirmation",
		handler.GetConfirmation,
	)

	group.POST(
		"/:id/order",
		handler.PlaceSourcingOrder,
	)
}
