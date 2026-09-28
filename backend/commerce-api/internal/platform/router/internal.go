package router

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"project.local/commerce-api/internal/jobruntime"
	"project.local/commerce-api/internal/platform/config"
	platformsecurity "project.local/commerce-api/internal/platform/security"
)

func registerInternalRuntimeRoutes(
	engine *gin.Engine,
	cfg config.RuntimeTickConfig,
	runner *jobruntime.LockedTickRunner,
) {
	if !cfg.Enabled {
		return
	}

	if runner == nil {
		panic(
			"runtime tick is enabled but runtime tick runner is not configured",
		)
	}

	engine.POST(
		"/api/v1/internal/runtime/tick",
		func(
			c *gin.Context,
		) {
			c.Header(
				"Cache-Control",
				"no-store",
			)

			if !validRuntimeTickAuthorization(
				c.GetHeader(
					"Authorization",
				),
				cfg.BearerToken,
			) {

				c.Header(
					"WWW-Authenticate",
					"Bearer",
				)

				c.JSON(
					http.StatusUnauthorized,
					gin.H{
						"status": "unauthorized",
					},
				)

				return
			}

			result, err :=
				runner.Run(
					c.Request.Context(),
				)
			if err != nil {
				c.JSON(
					http.StatusInternalServerError,
					gin.H{
						"status": "error",
					},
				)

				return
			}

			if !result.Acquired {
				c.JSON(
					http.StatusOK,
					gin.H{
						"status": "busy",

						"jobs_processed": 0,
					},
				)

				return
			}

			c.JSON(
				http.StatusOK,
				gin.H{
					"status": "ok",

					"jobs_processed": result.
						Tick.
						JobsProcessed,

					"duration_ms": result.
						Tick.
						Duration.
						Milliseconds(),
				},
			)
		},
	)
}

func validRuntimeTickAuthorization(
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

	if strings.TrimSpace(
		expectedToken,
	) == "" {

		return false
	}

	return platformsecurity.ConstantTimeEqual(
		parts[1],
		expectedToken,
	)
}
