package router

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"project.local/commerce-api/internal/jobruntime"
	"project.local/commerce-api/internal/platform/config"
	platformlogger "project.local/commerce-api/internal/platform/logger"
	platformmetrics "project.local/commerce-api/internal/platform/metrics"
	platformmiddleware "project.local/commerce-api/internal/platform/middleware"
	"project.local/commerce-api/internal/platform/storage"
)

type Dependencies struct {
	DB      *pgxpool.Pool
	Redis   *redis.Client
	Storage *storage.Gateway
	AppEnv  string

	AdminSecurity config.AdminSecurityConfig

	AppAllowedOrigins []string
	TrustedProxies    []string

	Metrics config.MetricsConfig

	RuntimeTickConfig config.RuntimeTickConfig
	RuntimeTickRunner *jobruntime.LockedTickRunner

	CODEnabled          bool
	BKashEnabled        bool
	NagadEnabled        bool
	RocketEnabled       bool
	BankTransferEnabled bool
}

func New(
	deps Dependencies,
) *gin.Engine {
	appEnv :=
		strings.ToLower(
			strings.TrimSpace(
				deps.AppEnv,
			),
		)

	isProduction :=
		appEnv ==
			"production"

	logLevel :=
		"debug"

	if isProduction {
		logLevel =
			"info"
	}

	baseLogger :=
		platformlogger.New(
			platformlogger.Config{
				Level: logLevel,

				Format: "json",

				AddSource: false,
			},
		)

	metricsRegistry :=
		platformmetrics.NewRegistry(
			deps.DB,
			deps.Redis,
		)

	securityHeadersConfig :=
		platformmiddleware.
			DefaultSecurityHeadersConfig()

	securityHeadersConfig.EnableHSTS =
		isProduction

	securityHeadersConfig.HSTSIncludeSubDomains =
		isProduction

	securityHeadersConfig.HSTSPreload =
		false

	corsMiddleware, err :=
		newCommerceCORS(
			deps.AppAllowedOrigins,
			deps.AdminSecurity.AllowedOrigins,
		)
	if err != nil {
		panic(
			"invalid platform CORS configuration: " +
				err.Error(),
		)
	}

	engine :=
		gin.New()

	trustedProxies :=
		deps.TrustedProxies

	if len(
		trustedProxies,
	) == 0 {
		trustedProxies =
			nil
	}

	if err :=
		engine.SetTrustedProxies(
			trustedProxies,
		); err != nil {

		panic(
			"invalid trusted proxy configuration: " +
				err.Error(),
		)
	}

	// Metrics runs outside Recovery so a panic converted to HTTP 500
	// by Recovery is recorded as a completed 5xx response.
	engine.Use(
		platformmiddleware.RequestID(),

		platformmiddleware.Logging(
			baseLogger,
		),

		platformmiddleware.Metrics(
			metricsRegistry,
			deps.Metrics.Path,
		),

		platformmiddleware.Recovery(),

		platformmiddleware.SecurityHeaders(
			securityHeadersConfig,
		),

		corsMiddleware,
	)

	registerHealthRoutes(
		engine,
		HealthDependencies{
			DB: deps.DB,

			Redis: deps.Redis,
		},
	)

	registerMetricsRoutes(
		engine,
		metricsRegistry,
		deps.Metrics,
	)

	registerInternalRuntimeRoutes(
		engine,
		deps.RuntimeTickConfig,
		deps.RuntimeTickRunner,
	)

	registerPublicRoutes(
		engine,
		deps,
	)

	// Customer wishlist is kept as a separate router slice.
	registerWishlistRoutes(
		engine,
		deps,
	)

	registerCRMRoutes(
		engine,
		deps,
	)

	registerProductRequestRoutes(
		engine,
		deps,
	)

	registerReviewRoutes(
		engine,
		deps,
	)

	registerPaymentRoutes(
		engine,
		deps,
	)

	// Customer-facing delivery routes only.
	// Administrative delivery operations live under real Admin RBAC.
	registerDeliveryRoutes(
		engine,
		deps,
	)

	registerSupportRoutes(
		engine,
		deps,
	)

	registerAdminRoutes(
		engine,
		deps,
	)

	return engine
}
