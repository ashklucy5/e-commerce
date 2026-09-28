package router

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"project.local/commerce-api/internal/platform/config"
	platformmetrics "project.local/commerce-api/internal/platform/metrics"
	platformsecurity "project.local/commerce-api/internal/platform/security"
)

const prometheusContentType = "text/plain; version=0.0.4; charset=utf-8"

func registerMetricsRoutes(
	engine *gin.Engine,
	registry *platformmetrics.Registry,
	cfg config.MetricsConfig,
) {
	if !cfg.Enabled ||
		registry == nil {

		return
	}

	engine.GET(
		cfg.Path,
		func(
			c *gin.Context,
		) {
			if cfg.BearerToken != "" &&
				!validMetricsAuthorization(
					c.GetHeader(
						"Authorization",
					),
					cfg.BearerToken,
				) {

				c.Header(
					"WWW-Authenticate",
					"Bearer",
				)

				c.Status(
					http.StatusUnauthorized,
				)

				return
			}

			c.Header(
				"Cache-Control",
				"no-store",
			)

			body :=
				registry.RenderPrometheus(
					c.Request.Context(),
				)

			c.Data(
				http.StatusOK,
				prometheusContentType,
				body,
			)
		},
	)
}

func validMetricsAuthorization(
	header string,
	expectedToken string,
) bool {
	parts :=
		strings.Fields(
			header,
		)

	if len(parts) != 2 ||
		!strings.EqualFold(
			parts[0],
			"Bearer",
		) {

		return false
	}

	return platformsecurity.ConstantTimeEqual(
		parts[1],
		expectedToken,
	)
}
