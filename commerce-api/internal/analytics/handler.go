package analytics

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	platformerrors "project.local/commerce-api/internal/platform/errors"
)

const (
	defaultAnalyticsLimit = 10

	maxAnalyticsLimit = 50
)

type Handler struct {
	service *Service

	now func() time.Time
}

func NewHandler(
	service *Service,
) *Handler {
	return &Handler{
		service: service,

		now: time.Now,
	}
}

func (h *Handler) Overview(
	c *gin.Context,
) {
	query, ok :=
		h.query(
			c,
		)
	if !ok {
		return
	}

	result, err :=
		h.service.Overview(
			c.Request.Context(),
			query,
		)
	if err != nil {
		platformerrors.Write(
			c,
			platformerrors.Internal(
				err,
			),
		)

		return
	}

	c.JSON(
		http.StatusOK,
		gin.H{
			"data": result,

			"meta": query.Meta(),
		},
	)
}

func (h *Handler) SalesTrend(
	c *gin.Context,
) {
	query, ok :=
		h.query(
			c,
		)
	if !ok {
		return
	}

	result, err :=
		h.service.SalesTrend(
			c.Request.Context(),
			query,
		)
	if err != nil {
		platformerrors.Write(
			c,
			platformerrors.Internal(
				err,
			),
		)

		return
	}

	c.JSON(
		http.StatusOK,
		gin.H{
			"data": result,

			"meta": query.Meta(),
		},
	)
}

func (h *Handler) Funnel(
	c *gin.Context,
) {
	query, ok :=
		h.query(
			c,
		)
	if !ok {
		return
	}

	result, err :=
		h.service.Funnel(
			c.Request.Context(),
			query,
		)
	if err != nil {
		platformerrors.Write(
			c,
			platformerrors.Internal(
				err,
			),
		)

		return
	}

	c.JSON(
		http.StatusOK,
		gin.H{
			"data": result,

			"meta": query.Meta(),
		},
	)
}

func (h *Handler) TopProducts(
	c *gin.Context,
) {
	query, ok :=
		h.query(
			c,
		)
	if !ok {
		return
	}

	limit, ok :=
		analyticsLimit(
			c,
		)
	if !ok {
		return
	}

	result, err :=
		h.service.TopProducts(
			c.Request.Context(),
			query,
			limit,
		)
	if err != nil {
		platformerrors.Write(
			c,
			platformerrors.Internal(
				err,
			),
		)

		return
	}

	c.JSON(
		http.StatusOK,
		gin.H{
			"data": result,

			"meta": query.Meta(),
		},
	)
}

func (h *Handler) PromotionAttribution(
	c *gin.Context,
) {
	query, ok :=
		h.query(
			c,
		)
	if !ok {
		return
	}

	limit, ok :=
		analyticsLimit(
			c,
		)
	if !ok {
		return
	}

	result, err :=
		h.service.
			PromotionAttribution(
				c.Request.Context(),
				query,
				limit,
			)
	if err != nil {
		platformerrors.Write(
			c,
			platformerrors.Internal(
				err,
			),
		)

		return
	}

	c.JSON(
		http.StatusOK,
		gin.H{
			"data": result,

			"meta": query.Meta(),
		},
	)
}

func (h *Handler) query(
	c *gin.Context,
) (
	Query,
	bool,
) {
	query, err :=
		ParseQuery(
			c.Query(
				"from",
			),
			c.Query(
				"to",
			),
			c.Query(
				"granularity",
			),
			c.Query(
				"currency",
			),
			h.now(),
		)

	if err == nil {
		return query,
			true
	}

	code :=
		"INVALID_ANALYTICS_QUERY"

	switch {
	case errors.Is(
		err,
		ErrInvalidDateRange,
	):
		code =
			"INVALID_ANALYTICS_DATE_RANGE"

	case errors.Is(
		err,
		ErrInvalidGranularity,
	):
		code =
			"INVALID_ANALYTICS_GRANULARITY"

	case errors.Is(
		err,
		ErrInvalidCurrency,
	):
		code =
			"INVALID_ANALYTICS_CURRENCY"
	}

	platformerrors.Write(
		c,
		platformerrors.BadRequest(
			code,
			err.Error(),
		),
	)

	return Query{},
		false
}

func analyticsLimit(
	c *gin.Context,
) (
	int,
	bool,
) {
	value :=
		strings.TrimSpace(
			c.Query(
				"limit",
			),
		)

	if value == "" {
		return defaultAnalyticsLimit,
			true
	}

	limit, err :=
		strconv.Atoi(
			value,
		)

	if err != nil ||
		limit < 1 ||
		limit >
			maxAnalyticsLimit {

		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_ANALYTICS_LIMIT",
				"limit must be between 1 and 50",
			),
		)

		return 0,
			false
	}

	return limit,
		true
}

func (h *Handler) RecommendationOverview(
	c *gin.Context,
) {
	query, ok := h.query(c)
	if !ok {
		return
	}

	result, err := h.service.RecommendationOverview(c.Request.Context(), query)
	if err != nil {
		platformerrors.Write(c, platformerrors.Internal(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result, "meta": query.Meta()})
}

func (h *Handler) RecommendationTrend(
	c *gin.Context,
) {
	query, ok := h.query(c)
	if !ok {
		return
	}

	result, err := h.service.RecommendationTrend(c.Request.Context(), query)
	if err != nil {
		platformerrors.Write(c, platformerrors.Internal(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result, "meta": query.Meta()})
}

func (h *Handler) RecommendationPlacements(
	c *gin.Context,
) {
	query, ok := h.query(c)
	if !ok {
		return
	}

	result, err := h.service.RecommendationBreakdownByPlacement(c.Request.Context(), query)
	if err != nil {
		platformerrors.Write(c, platformerrors.Internal(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result, "meta": query.Meta()})
}

func (h *Handler) RecommendationStrategies(
	c *gin.Context,
) {
	query, ok := h.query(c)
	if !ok {
		return
	}

	result, err := h.service.RecommendationBreakdownByStrategy(c.Request.Context(), query)
	if err != nil {
		platformerrors.Write(c, platformerrors.Internal(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result, "meta": query.Meta()})
}

func (h *Handler) RecommendationProducts(
	c *gin.Context,
) {
	query, ok := h.query(c)
	if !ok {
		return
	}
	limit, ok := analyticsLimit(c)
	if !ok {
		return
	}

	result, err := h.service.RecommendationTopProducts(c.Request.Context(), query, limit)
	if err != nil {
		platformerrors.Write(c, platformerrors.Internal(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result, "meta": query.Meta()})
}

func (h *Handler) RecommendationDiagnostics(
	c *gin.Context,
) {
	result, err := h.service.RecommendationEngineDiagnostics(c.Request.Context())
	if err != nil {
		platformerrors.Write(c, platformerrors.Internal(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (h *Handler) RecommendationRelations(
	c *gin.Context,
) {
	limit, ok := analyticsLimit(c)
	if !ok {
		return
	}

	result, err := h.service.RecommendationCategoryRelations(c.Request.Context(), limit)
	if err != nil {
		platformerrors.Write(c, platformerrors.Internal(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (h *Handler) BusinessInsights(
	c *gin.Context,
) {
	query, ok := h.query(c)
	if !ok {
		return
	}

	result, err := h.service.BusinessInsights(c.Request.Context(), query)
	if err != nil {
		platformerrors.Write(c, platformerrors.Internal(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result, "meta": query.Meta()})
}
