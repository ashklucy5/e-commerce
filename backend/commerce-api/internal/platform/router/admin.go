package router

import (
	"strings"

	"github.com/gin-gonic/gin"

	admincore "project.local/commerce-api/internal/admin"
	"project.local/commerce-api/internal/adminauth"
	"project.local/commerce-api/internal/adminupload"
	"project.local/commerce-api/internal/catalogimport"
	"project.local/commerce-api/internal/catalogmediawrite"
	"project.local/commerce-api/internal/catalogwrite"
	"project.local/commerce-api/internal/inventory"
	"project.local/commerce-api/internal/order"
	platformmiddleware "project.local/commerce-api/internal/platform/middleware"
	"project.local/commerce-api/internal/refund"
	"project.local/commerce-api/internal/returns"
)

func registerAdminRoutes(
	engine *gin.Engine,
	deps Dependencies,
) {
	isProduction :=
		strings.EqualFold(
			strings.TrimSpace(
				deps.AppEnv,
			),
			"production",
		)

	keyring, err :=
		adminauth.NewEncryptionKeyring(
			deps.AdminSecurity,
		)
	if err != nil {
		panic(
			"initialize Admin encryption: " +
				err.Error(),
		)
	}

	authRepository :=
		adminauth.NewRepository(
			deps.DB,
		)

	mfaService, err :=
		adminauth.NewMFAService(
			authRepository,
			keyring,
			deps.AdminSecurity.TOTPIssuer,
		)
	if err != nil {
		panic(
			"initialize Admin MFA service: " +
				err.Error(),
		)
	}

	lockoutService, err :=
		adminauth.NewLockoutService(
			deps.Redis,
			adminauth.DefaultLockoutConfig(),
		)
	if err != nil {
		panic(
			"initialize Admin lockout service: " +
				err.Error(),
		)
	}

	authService, err :=
		adminauth.NewService(
			authRepository,
			mfaService,
			lockoutService,
		)
	if err != nil {
		panic(
			"initialize Admin auth service: " +
				err.Error(),
		)
	}

	authHandler :=
		adminauth.NewHandler(
			authService,
			adminauth.CookieConfig{
				Secure: isProduction,

				Domain: deps.AdminSecurity.CookieDomain,
			},
		)

	networkGuard, err :=
		platformmiddleware.
			NewAdminNetworkGuard(
				deps.AdminSecurity.
					AllowedNetworks,
			)
	if err != nil {
		panic(
			"initialize Admin network guard: " +
				err.Error(),
		)
	}

	authRateLimiter, err :=
		platformmiddleware.
			NewAdminRateLimiter(
				deps.Redis,
				platformmiddleware.
					DefaultAdminAuthRateLimitConfig(),
			)
	if err != nil {
		panic(
			"initialize Admin auth rate limiter: " +
				err.Error(),
		)
	}

	protectedRateLimiter, err :=
		platformmiddleware.
			NewAdminRateLimiter(
				deps.Redis,
				platformmiddleware.
					DefaultAdminProtectedRateLimitConfig(),
			)
	if err != nil {
		panic(
			"initialize Admin protected rate limiter: " +
				err.Error(),
		)
	}

	root :=
		engine.Group(
			"/api/v1/admin",
		)

	root.Use(
		networkGuard,
	)

	// Password/MFA/session endpoints.
	//
	// Refresh and logout perform their own CSRF validation because
	// refresh must continue to work after the short-lived access token
	// has expired.
	authRoutes :=
		root.Group(
			"",
		)

	authRoutes.Use(
		authRateLimiter,
	)

	adminauth.RegisterRoutes(
		authRoutes,
		authHandler,
	)

	// Everything outside /admin/auth requires a fully authenticated,
	// MFA-backed Admin session.
	protected :=
		root.Group(
			"",
		)

	protected.Use(
		protectedRateLimiter,

		platformmiddleware.
			RequireAdminAuth(
				authService,
			),

		platformmiddleware.
			RequireAdminMFA(),

		platformmiddleware.
			RequireAdminCSRF(
				authService,
			),
	)

	adminService :=
		admincore.NewService(
			deps.DB,
		)

	adminHandler :=
		admincore.NewHandler(
			adminService,
		)

	admincore.RegisterRoutes(
		protected,
		adminHandler,
		platformmiddleware.
			RequireAdminPermission,
	)

	// ---------------------------------------------------------
	// Admin CRM private support attachments
	// ---------------------------------------------------------

	crmAttachmentHandler :=
		admincore.NewCRMAttachmentHandler(
			deps.DB,
			deps.Storage,
		)

	protected.POST(
		"/crm/cases/:id/attachments/upload-target",
		platformmiddleware.
			RequireAdminPermission(
				admincore.PermissionCRMManage,
			),
		crmAttachmentHandler.
			CreateUploadTarget,
	)

	protected.POST(
		"/crm/attachments/:id/complete",
		platformmiddleware.
			RequireAdminPermission(
				admincore.PermissionCRMManage,
			),
		crmAttachmentHandler.
			CompleteUpload,
	)

	protected.GET(
		"/crm/attachments/:id",
		platformmiddleware.
			RequireAdminPermission(
				admincore.PermissionCRMRead,
			),
		crmAttachmentHandler.
			GetAttachment,
	)

	// ---------------------------------------------------------
	// Admin in-app notifications
	// ---------------------------------------------------------

	registerAdminNotificationRoutes(
		protected,
		deps,
	)

	registerAdminProductRequestRoutes(
		protected,
		deps,
	)

	registerCatalogAdminRoutes(
		protected,
		deps,
	)
	registerInventoryAdminRoutes(
		protected,
		deps,
	)

	registerOrderAdminRoutes(
		protected,
		deps,
	)

	registerReturnAdminRoutes(
		protected,
		deps,
	)

	registerRefundAdminRoutes(
		protected,
		deps,
	)

	registerAdminDeliveryRoutes(
		protected,
		deps,
	)

	registerAdminWarehouseRoutes(
		protected,
		deps,
	)
}

func registerCatalogAdminRoutes(
	admin *gin.RouterGroup,
	deps Dependencies,
) {
	catalogWriteRepository :=
		catalogwrite.NewRepository(
			deps.DB,
		)

	catalogWriteService :=
		catalogwrite.NewService(
			catalogWriteRepository,
		)

	catalogWriteHandler :=
		catalogwrite.NewHandler(
			catalogWriteService,
		)

	catalogWriteRoutes :=
		admin.Group(
			"",
		)

	catalogWriteRoutes.Use(
		platformmiddleware.
			RequireAdminPermission(
				admincore.
					PermissionCatalogWrite,
			),
	)

	catalogwrite.RegisterRoutes(
		catalogWriteRoutes,
		catalogWriteHandler,
	)

	adminUploadService :=
		adminupload.NewService(
			deps.Storage,
		)

	adminUploadHandler :=
		adminupload.NewHandler(
			adminUploadService,
		)

	adminUploadRoutes :=
		admin.Group(
			"",
		)

	adminUploadRoutes.Use(
		platformmiddleware.
			RequireAdminPermission(
				admincore.
					PermissionCatalogWrite,
			),
	)

	adminupload.RegisterRoutes(
		adminUploadRoutes,
		adminUploadHandler,
	)

	catalogMediaRepository :=
		catalogmediawrite.
			NewRepository(
				deps.DB,
			)

	catalogMediaService :=
		catalogmediawrite.
			NewService(
				catalogMediaRepository,
				deps.Storage,
			)

	catalogMediaHandler :=
		catalogmediawrite.
			NewHandler(
				catalogMediaService,
			)

	catalogMediaRoutes :=
		admin.Group(
			"",
		)

	catalogMediaRoutes.Use(
		platformmiddleware.
			RequireAdminPermission(
				admincore.
					PermissionCatalogWrite,
			),
	)

	catalogmediawrite.RegisterRoutes(
		catalogMediaRoutes,
		catalogMediaHandler,
	)

	catalogImportRepository :=
		catalogimport.NewRepository(
			deps.DB,
		)

	catalogImportService :=
		catalogimport.NewService(
			catalogImportRepository,
			deps.Storage,
		)

	catalogImportHandler :=
		catalogimport.NewHandler(
			catalogImportService,
		)

	catalogImportRoutes :=
		admin.Group(
			"",
		)

	catalogImportRoutes.Use(
		platformmiddleware.
			RequireAdminPermission(
				admincore.
					PermissionCatalogImport,
			),
	)

	catalogimport.RegisterRoutes(
		catalogImportRoutes,
		catalogImportHandler,
	)
}

func registerInventoryAdminRoutes(
	admin *gin.RouterGroup,
	deps Dependencies,
) {
	repository :=
		inventory.NewRepository(
			deps.DB,
		)

	service :=
		inventory.NewService(
			repository,
		)

	handler :=
		inventory.NewHandler(
			service,
		)

	requireRead :=
		platformmiddleware.
			RequireAdminPermission(
				admincore.
					PermissionInventoryRead,
			)

	requireAdjust :=
		platformmiddleware.
			RequireAdminPermission(
				admincore.
					PermissionInventoryAdjust,
			)

	admin.GET(
		"/inventory",
		requireRead,
		handler.List,
	)

	admin.GET(
		"/inventory/:variant_id",
		requireRead,
		handler.Get,
	)

	admin.GET(
		"/inventory/:variant_id/movements",
		requireRead,
		handler.Movements,
	)

	admin.POST(
		"/inventory/:variant_id/adjust",
		requireAdjust,
		handler.Adjust,
	)

	admin.PATCH(
		"/inventory/:variant_id",
		requireAdjust,
		handler.Update,
	)
}

func registerOrderAdminRoutes(
	admin *gin.RouterGroup,
	deps Dependencies,
) {
	inventoryRepository :=
		inventory.NewRepository(
			deps.DB,
		)

	inventoryService :=
		inventory.NewService(
			inventoryRepository,
		)

	repository :=
		order.NewRepository(
			deps.DB,
		)

	service :=
		order.NewService(
			repository,
			inventoryService,
			order.PaymentMethodConfig{
				CODEnabled: deps.CODEnabled,

				BKashEnabled: deps.BKashEnabled,

				NagadEnabled: deps.NagadEnabled,

				RocketEnabled: deps.RocketEnabled,

				BankTransferEnabled: deps.BankTransferEnabled,
			},
		)

	requireManage :=
		platformmiddleware.
			RequireAdminPermission(
				admincore.
					PermissionOrderManage,
			)

	admin.PATCH(
		"/orders/:order_id/fulfillment",
		requireManage,
		adminActorHandler(
			func(
				actorID string,
			) *order.AdminHandler {
				return order.NewAdminHandler(
					service,
					actorID,
				)
			},
			(*order.AdminHandler).
				TransitionFulfillment,
		),
	)
}

func registerReturnAdminRoutes(
	admin *gin.RouterGroup,
	deps Dependencies,
) {
	inventoryRepository :=
		inventory.NewRepository(
			deps.DB,
		)

	inventoryService :=
		inventory.NewService(
			inventoryRepository,
		)

	repository :=
		returns.NewRepository(
			deps.DB,
		)

	service :=
		returns.NewService(
			repository,
			inventoryService,
		)

	requireManage :=
		platformmiddleware.
			RequireAdminPermission(
				admincore.
					PermissionReturnManage,
			)

	factory :=
		func(
			actorID string,
		) *returns.AdminHandler {
			return returns.NewAdminHandler(
				service,
				actorID,
			)
		}

	admin.POST(
		"/returns/:return_id/approve",
		requireManage,
		adminActorHandler(
			factory,
			(*returns.AdminHandler).Approve,
		),
	)

	admin.POST(
		"/returns/:return_id/reject",
		requireManage,
		adminActorHandler(
			factory,
			(*returns.AdminHandler).Reject,
		),
	)

	admin.POST(
		"/returns/:return_id/receive",
		requireManage,
		adminActorHandler(
			factory,
			(*returns.AdminHandler).Receive,
		),
	)

	admin.POST(
		"/returns/:return_id/inspect",
		requireManage,
		adminActorHandler(
			factory,
			(*returns.AdminHandler).Inspect,
		),
	)
}

func registerRefundAdminRoutes(
	admin *gin.RouterGroup,
	deps Dependencies,
) {
	repository :=
		refund.NewRepository(
			deps.DB,
		)

	service :=
		refund.NewService(
			repository,
		)

	requireRefund :=
		platformmiddleware.
			RequireAdminPermission(
				admincore.
					PermissionPaymentRefund,
			)

	factory :=
		func(
			actorID string,
		) *refund.AdminHandler {
			return refund.NewAdminHandler(
				service,
				actorID,
			)
		}

	admin.POST(
		"/returns/:return_id/refunds",
		requireRefund,
		adminActorHandler(
			factory,
			(*refund.AdminHandler).
				RequestForReturn,
		),
	)

	admin.POST(
		"/refunds/:refund_id/approve",
		requireRefund,
		adminActorHandler(
			factory,
			(*refund.AdminHandler).Approve,
		),
	)

	admin.POST(
		"/refunds/:refund_id/process",
		requireRefund,
		adminActorHandler(
			factory,
			(*refund.AdminHandler).
				StartProcessing,
		),
	)

	admin.POST(
		"/refunds/:refund_id/succeed",
		requireRefund,
		adminActorHandler(
			factory,
			(*refund.AdminHandler).Succeed,
		),
	)

	admin.POST(
		"/refunds/:refund_id/fail",
		requireRefund,
		adminActorHandler(
			factory,
			(*refund.AdminHandler).Fail,
		),
	)
}
