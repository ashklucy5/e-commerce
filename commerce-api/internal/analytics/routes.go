package analytics

import "github.com/gin-gonic/gin"

func RegisterRoutes(
	group *gin.RouterGroup,
	handler *Handler,
	requirePermission func(string) gin.HandlerFunc,
	permission string,
) {
	analyticsGroup :=
		group.Group(
			"/analytics",
		)

	analyticsGroup.Use(
		requirePermission(
			permission,
		),
	)

	analyticsGroup.GET(
		"/overview",
		handler.Overview,
	)

	analyticsGroup.GET(
		"/sales",
		handler.SalesTrend,
	)

	analyticsGroup.GET(
		"/funnel",
		handler.Funnel,
	)

	analyticsGroup.GET(
		"/products/top",
		handler.TopProducts,
	)

	analyticsGroup.GET(
		"/promotions",
		handler.PromotionAttribution,
	)

	analyticsGroup.GET(
		"/insights",
		handler.BusinessInsights,
	)

	analyticsGroup.GET(
		"/recommendations/overview",
		handler.RecommendationOverview,
	)

	analyticsGroup.GET(
		"/recommendations/trend",
		handler.RecommendationTrend,
	)

	analyticsGroup.GET(
		"/recommendations/placements",
		handler.RecommendationPlacements,
	)

	analyticsGroup.GET(
		"/recommendations/strategies",
		handler.RecommendationStrategies,
	)

	analyticsGroup.GET(
		"/recommendations/products",
		handler.RecommendationProducts,
	)

	analyticsGroup.GET(
		"/recommendations/engine",
		handler.RecommendationDiagnostics,
	)

	analyticsGroup.GET(
		"/recommendations/relations",
		handler.RecommendationRelations,
	)
}

func RegisterFinanceRoutes(
	group *gin.RouterGroup,
	handler *Handler,
	requirePermission func(string) gin.HandlerFunc,
	permission string,
) {
	financeGroup :=
		group.Group(
			"/finance",
		)

	financeGroup.Use(
		requirePermission(
			permission,
		),
	)

	financeGroup.GET(
		"/profit-loss",
		handler.ProfitLoss,
	)

	financeGroup.GET(
		"/profit-loss/trend",
		handler.ProfitLossTrend,
	)

	financeGroup.GET(
		"/products/profitability",
		handler.ProductProfitability,
	)
}
