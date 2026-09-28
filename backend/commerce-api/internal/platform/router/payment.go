package router

import (
	"github.com/gin-gonic/gin"

	"project.local/commerce-api/internal/auth"
	"project.local/commerce-api/internal/inventory"
	"project.local/commerce-api/internal/order"
	"project.local/commerce-api/internal/payment"
)

func registerPaymentRoutes(
	engine *gin.Engine,
	deps Dependencies,
) {
	paymentRepository :=
		payment.NewRepository(
			deps.DB,
		)

	paymentProviders :=
		newPaymentProviderRegistry(
			deps,
		)

	// ---------------------------------------------------------
	// Customer / guest payment routes
	// ---------------------------------------------------------

	customerGroup :=
		engine.Group(
			"/api/v1",
		)

	authRepository :=
		auth.NewRepository(
			deps.DB,
		)

	authService :=
		auth.NewService(
			authRepository,
		)

	/*
		Payment status/initiation supports:

		- authenticated customer ownership
		- guest checkout ownership through X-Checkout-Key
	*/
	customerGroup.Use(
		auth.OptionalAuth(
			authService,
		),
	)

	initiationService :=
		payment.NewInitiationService(
			paymentRepository,
			paymentProviders,
		)

	paymentHandler :=
		payment.NewStatusHandler(
			paymentRepository,
		)

	paymentHandler.SetInitiationService(
		initiationService,
	)

	payment.RegisterRoutes(
		customerGroup,
		paymentHandler,
	)

	// ---------------------------------------------------------
	// Verified provider webhook ingress
	// ---------------------------------------------------------

	/*
		Do NOT put provider webhooks behind customer authentication.

		The provider adapter is responsible for authenticating and
		verifying the provider request before payment state is
		trusted.
	*/

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

	/*
		This service instance is only used by the payment-confirmation
		transaction.

		MarkPaidTx uses the shared PostgreSQL transaction and inventory
		service, so payment success, inventory commit and order state
		remain atomic.
	*/
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

	verifiedPaymentService :=
		payment.NewService(
			paymentRepository,
			orderService,
		)

	webhookService :=
		payment.NewWebhookService(
			paymentProviders,
			verifiedPaymentService,
		)

	webhookHandler :=
		payment.NewWebhookHandler(
			webhookService,
		)

	/*
		Separate Gin group intentionally has no OptionalAuth/
		RequireAuth middleware.
	*/
	webhookGroup :=
		engine.Group(
			"/api/v1",
		)

	payment.RegisterWebhookRoutes(
		webhookGroup,
		webhookHandler,
	)
}
