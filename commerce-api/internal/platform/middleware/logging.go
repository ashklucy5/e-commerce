package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"

	platformlogger "project.local/commerce-api/internal/platform/logger"
)

func Logging(
	baseLogger *slog.Logger,
) gin.HandlerFunc {
	if baseLogger == nil {
		baseLogger =
			slog.Default()
	}

	return func(
		c *gin.Context,
	) {
		startedAt :=
			time.Now()

		requestID, _ :=
			RequestIDFromContext(
				c,
			)

		requestLogger :=
			baseLogger.With(
				"request_id",
				requestID,

				"method",
				c.Request.Method,
			)

		c.Request =
			c.Request.WithContext(
				platformlogger.IntoContext(
					c.Request.Context(),
					requestLogger,
				),
			)

		c.Next()

		route :=
			c.FullPath()

		if route == "" {
			route = "unmatched"
		}

		duration :=
			time.Since(
				startedAt,
			)

		attributes :=
			[]any{
				"route",
				route,

				"status",
				c.Writer.Status(),

				"duration_ms",
				float64(
					duration.Microseconds(),
				) / 1000,

				"response_bytes",
				c.Writer.Size(),

				"client_ip",
				c.ClientIP(),

				"user_agent",
				c.Request.UserAgent(),
			}

		if len(
			c.Errors,
		) > 0 {
			attributes =
				append(
					attributes,
					"gin_errors",
					c.Errors.String(),
				)
		}

		switch status :=
			c.Writer.Status(); {

		case status >= 500:
			requestLogger.Error(
				"http request completed",
				attributes...,
			)

		case status >= 400:
			requestLogger.Warn(
				"http request completed",
				attributes...,
			)

		default:
			requestLogger.Info(
				"http request completed",
				attributes...,
			)
		}
	}
}
