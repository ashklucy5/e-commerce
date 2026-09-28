package router

import (
	"github.com/gin-gonic/gin"

	"project.local/commerce-api/internal/auth"
	"project.local/commerce-api/internal/inventory"
	"project.local/commerce-api/internal/order"
	"project.local/commerce-api/internal/productrequest"
)

func registerProductRequestRoutes(
	engine *gin.Engine,
	deps Dependencies,
) {
	group :=
		engine.Group(
			"/api/v1/product-requests",
		)

	authRepository :=
		auth.NewRepository(
			deps.DB,
		)

	authService :=
		auth.NewService(
			authRepository,
		)

	repository :=
		productrequest.NewRepository(
			deps.DB,
		)

	paymentProviders :=
		newPaymentProviderRegistry(
			deps,
		)

	inventoryRepository :=
		inventory.NewRepository(
			deps.DB,
		)

	inventoryService :=
		inventory.NewService(
			inventoryRepository,
		)

	orderRepository :=
		order.NewRepository(
			deps.DB,
		)

	orderService :=
		order.NewService(
			orderRepository,
			inventoryService,
			order.PaymentMethodConfig{
				CODEnabled: deps.CODEnabled,

				BKashEnabled: deps.BKashEnabled,

				NagadEnabled: deps.NagadEnabled,

				RocketEnabled: deps.RocketEnabled,

				BankTransferEnabled: deps.BankTransferEnabled,

				ProviderAvailable: paymentProviders.Available,
			},
		)

	service :=
		productrequest.NewService(
			repository,
			orderService,
		)

	handler :=
		productrequest.NewHandler(
			service,
		)

	productrequest.RegisterRoutes(
		group,
		handler,
		auth.RequireAuth(
			authService,
		),
	)
}
