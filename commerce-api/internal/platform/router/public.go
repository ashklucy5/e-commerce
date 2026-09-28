package router

import (
	"context"

	"github.com/gin-gonic/gin"

	"project.local/commerce-api/internal/auth"
	"project.local/commerce-api/internal/cart"
	"project.local/commerce-api/internal/catalog"
	"project.local/commerce-api/internal/category"
	"project.local/commerce-api/internal/checkout"
	"project.local/commerce-api/internal/customer"
	"project.local/commerce-api/internal/customeraccess"
	"project.local/commerce-api/internal/inventory"
	"project.local/commerce-api/internal/order"
	platformcache "project.local/commerce-api/internal/platform/cache"
	platformmiddleware "project.local/commerce-api/internal/platform/middleware"
	"project.local/commerce-api/internal/promotion"
	"project.local/commerce-api/internal/recommendation"
	"project.local/commerce-api/internal/refund"
	"project.local/commerce-api/internal/returns"
	"project.local/commerce-api/internal/search"
)

func registerPublicRoutes(
	engine *gin.Engine,
	deps Dependencies,
) {
	public :=
		engine.Group(
			"/api/v1",
		)

	// ---------------------------------------------------------
	// Customer authentication
	// ---------------------------------------------------------

	authRepository :=
		auth.NewRepository(
			deps.DB,
		)

	authService :=
		auth.NewService(
			authRepository,
		)

	// Load the customer OTP/signup policy from the runtime
	// environment.
	//
	// NewService intentionally starts from a conservative
	// fail-closed policy, so startup must explicitly wire the
	// environment policy before auth routes are exposed.
	otpPolicy, err :=
		auth.LoadOTPPolicyFromEnv()
	if err != nil {
		panic(
			"initialize customer OTP policy: " +
				err.Error(),
		)
	}

	authService.SetOTPPolicy(
		otpPolicy,
	)

	loginLockout, err :=
		auth.NewLoginLockoutService(
			deps.Redis,
			auth.DefaultLoginLockoutConfig(),
		)
	if err != nil {
		panic(
			"initialize customer login lockout: " +
				err.Error(),
		)
	}

	authService.SetLoginLockout(
		loginLockout,
	)

	authRateLimiter, err :=
		platformmiddleware.NewRateLimiter(
			deps.Redis,
			platformmiddleware.DefaultCustomerAuthRateLimitConfig(),
		)
	if err != nil {
		panic(
			"initialize customer auth rate limiter: " +
				err.Error(),
		)
	}

	authHandler :=
		auth.NewHandler(
			authService,
		)

	auth.RegisterRoutes(
		public,
		authHandler,
		authRateLimiter,
	)

	// ---------------------------------------------------------
	// Customer-aware public routes
	// ---------------------------------------------------------

	customerAware :=
		public.Group(
			"",
		)

	customerAware.Use(
		auth.OptionalAuth(
			authService,
		),
	)

	customerAccess :=
		customeraccess.NewGuard(
			deps.DB,
		)

	customerAware.Use(
		customerAccess.Middleware(),
	)

	// ---------------------------------------------------------
	// Customer profile
	// ---------------------------------------------------------

	customerRepository :=
		customer.NewRepository(
			deps.DB,
		)

	customerService :=
		customer.NewService(
			customerRepository,
			deps.Storage,
		)

	customerHandler :=
		customer.NewHandler(
			customerService,
		)

	customer.RegisterRoutes(
		public,
		customerHandler,
		auth.RequireAuth(
			authService,
		),
	)

	// ---------------------------------------------------------
	// Customer in-app notifications
	// ---------------------------------------------------------

	registerCustomerNotificationRoutes(
		public,
		deps,
		auth.RequireAuth(
			authService,
		),
	)

	// ---------------------------------------------------------
	// Shared Redis cache
	// ---------------------------------------------------------

	publicCache, err :=
		platformcache.NewStore(
			deps.Redis,
		)
	if err != nil {
		panic(
			"initialize public cache: " +
				err.Error(),
		)
	}

	// ---------------------------------------------------------
	// Categories
	// ---------------------------------------------------------

	categoryRepository :=
		category.NewRepository(
			deps.DB,
		)

	categoryService :=
		category.NewService(
			categoryRepository,
		)

	categoryService.SetCache(
		publicCache,
	)

	categoryHandler :=
		category.NewHandler(
			categoryService,
		)

	category.RegisterRoutes(
		public,
		categoryHandler,
	)

	// ---------------------------------------------------------
	// Recommendation ranker
	// ---------------------------------------------------------

	recommendationRepository :=
		recommendation.NewRepository(
			deps.DB,
		)

	recommendationService :=
		recommendation.NewService(
			recommendationRepository,
		)

	recommendationTelemetry :=
		recommendation.NewTelemetryService(
			recommendationRepository,
		)

	recommendationRateLimiter, err :=
		platformmiddleware.NewRateLimiter(
			deps.Redis,
			platformmiddleware.DefaultRecommendationEventRateLimitConfig(),
		)
	if err != nil {
		panic(
			"initialize recommendation event rate limiter: " +
				err.Error(),
		)
	}

	recommendation.RegisterRoutes(
		customerAware,
		recommendation.NewHandler(
			recommendationTelemetry,
		),
		recommendationRateLimiter,
	)

	// ---------------------------------------------------------
	// Catalog
	// ---------------------------------------------------------

	catalogRepository :=
		catalog.NewRepository(
			deps.DB,
		)

	catalogService :=
		catalog.NewService(
			catalogRepository,
		)

	catalogService.SetProductRanker(
		recommendationService,
	)

	catalogHandler :=
		catalog.NewHandler(
			catalogService,
		)

	catalog.RegisterRoutes(
		customerAware,
		catalogHandler,
	)

	// ---------------------------------------------------------
	// Search
	// ---------------------------------------------------------

	searchRepository :=
		search.NewRepository(
			deps.DB,
		)

	if err :=
		search.ValidateDatabaseCapabilities(
			context.Background(),
			deps.DB,
		); err != nil {

		panic(
			"validate search database capabilities: " +
				err.Error(),
		)
	}

	searchService :=
		search.NewService(
			searchRepository,
		)

	searchHandler :=
		search.NewHandler(
			searchService,
		)

	search.RegisterRoutes(
		customerAware,
		searchHandler,
	)

	// ---------------------------------------------------------
	// Cart
	// ---------------------------------------------------------

	cartRepository :=
		cart.NewRepository(
			deps.DB,
		)

	cartService :=
		cart.NewService(
			cartRepository,
		)

	cartHandler :=
		cart.NewHandler(
			cartService,
		)

	cart.RegisterRoutes(
		public,
		cartHandler,
	)

	// ---------------------------------------------------------
	// Promotions
	// ---------------------------------------------------------

	promotionRepository :=
		promotion.NewRepository(
			deps.DB,
		)

	promotionService :=
		promotion.NewService(
			promotionRepository,
		)

	promotionHandler :=
		promotion.NewHandler(
			promotionService,
		)

	promotion.RegisterPublicRoutes(
		public,
		promotionHandler,
	)

	// ---------------------------------------------------------
	// Payment provider registry
	//
	// The same registry is shared between checkout availability
	// and order placement so both layers make the same decision.
	//
	// No external adapters are currently registered.
	// ---------------------------------------------------------

	paymentProviders :=
		newPaymentProviderRegistry(
			deps,
		)

	// ---------------------------------------------------------
	// Checkout
	// ---------------------------------------------------------

	checkoutRepository :=
		checkout.NewRepository(
			deps.DB,
		)

	checkoutDeliveryMethods :=
		loadCheckoutDeliveryMethods()

	checkoutService :=
		checkout.NewService(
			checkoutRepository,
			checkout.PaymentMethodConfig{
				CODEnabled: deps.CODEnabled,

				BKashEnabled: deps.BKashEnabled,

				NagadEnabled: deps.NagadEnabled,

				RocketEnabled: deps.RocketEnabled,

				BankTransferEnabled: deps.BankTransferEnabled,

				ProviderAvailable: paymentProviders.Available,
			},
			promotionService,
		)

	if err :=
		checkoutService.SetDeliveryMethods(
			checkoutDeliveryMethods,
		); err != nil {

		panic(
			"initialize checkout delivery methods: " +
				err.Error(),
		)
	}

	checkoutHandler :=
		checkout.NewHandler(
			checkoutService,
		)

	checkout.RegisterRoutes(
		customerAware,
		checkoutHandler,
	)

	// ---------------------------------------------------------
	// Shared inventory service
	// ---------------------------------------------------------

	inventoryRepository :=
		inventory.NewRepository(
			deps.DB,
		)

	inventoryService :=
		inventory.NewService(
			inventoryRepository,
		)

	// ---------------------------------------------------------
	// Shared refund service
	// ---------------------------------------------------------

	refundRepository :=
		refund.NewRepository(
			deps.DB,
		)

	refundService :=
		refund.NewService(
			refundRepository,
		)

	// ---------------------------------------------------------
	// Orders
	// ---------------------------------------------------------

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

	orderService.SetPromotionEvaluator(
		promotionService,
	)

	if err :=
		orderService.SetDeliveryMethods(
			checkoutDeliveryMethods,
		); err != nil {

		panic(
			"initialize order delivery methods: " +
				err.Error(),
		)
	}

	orderService.SetCancellationRefundLifecycle(
		refundService,
	)

	orderHandler :=
		order.NewHandler(
			orderService,
		)

	order.RegisterRoutes(
		customerAware,
		orderHandler,
	)

	order.RegisterAccountRoutes(
		public,
		orderHandler,
		auth.RequireAuth(
			authService,
		),
	)

	// ---------------------------------------------------------
	// Returns
	// ---------------------------------------------------------

	returnRepository :=
		returns.NewRepository(
			deps.DB,
		)

	returnService :=
		returns.NewService(
			returnRepository,
			inventoryService,
		)

	returnHandler :=
		returns.NewHandler(
			returnService,
		)

	returns.RegisterRoutes(
		customerAware,
		returnHandler,
	)

	// ---------------------------------------------------------
	// Refunds
	// ---------------------------------------------------------

	refundHandler :=
		refund.NewHandler(
			refundService,
		)

	refund.RegisterRoutes(
		customerAware,
		refundHandler,
	)
}
