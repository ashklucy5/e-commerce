package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	platformmetrics "project.local/commerce-api/internal/platform/metrics"
)

func Metrics(
	registry *platformmetrics.Registry,
	excludedPaths ...string,
) gin.HandlerFunc {
	excluded :=
		make(
			map[string]struct{},
			len(excludedPaths),
		)

	for _, path := range excludedPaths {

		path =
			strings.TrimSpace(
				path,
			)

		if path == "" {
			continue
		}

		excluded[path] =
			struct{}{}
	}

	return func(
		c *gin.Context,
	) {
		if registry == nil {
			c.Next()

			return
		}

		if _, skip := excluded[c.Request.URL.Path]; skip {
			c.Next()

			return
		}

		trace :=
			platformmetrics.StartTrace()

		registry.HTTPRequestStarted()

		defer func() {
			registry.HTTPRequestFinished()

			route :=
				c.FullPath()

			if route == "" {
				route =
					"unmatched"
			}

			registry.ObserveHTTP(
				c.Request.Method,
				route,
				c.Writer.Status(),
				trace.Duration(),
				c.Writer.Size(),
			)
		}()

		c.Next()
	}
}
