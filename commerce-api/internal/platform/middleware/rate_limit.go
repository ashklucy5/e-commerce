package middleware

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	platformerrors "project.local/commerce-api/internal/platform/errors"
	platformsecurity "project.local/commerce-api/internal/platform/security"
)

type RateLimitConfig struct {
	Scope string

	Limit int64

	Window time.Duration
}

var rateLimitScript = redis.NewScript(
	`
local count = redis.call('INCR', KEYS[1])
if count == 1 then
  redis.call('EXPIRE', KEYS[1], ARGV[1])
end
return count
`,
)

func DefaultCustomerAuthRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		Scope: "customer-auth",

		Limit: 120,

		Window: time.Minute,
	}
}

func DefaultRecommendationEventRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		Scope: "recommendation-events",

		Limit: 300,

		Window: time.Minute,
	}
}

func NewRateLimiter(
	client *redis.Client,
	cfg RateLimitConfig,
) (
	gin.HandlerFunc,
	error,
) {
	if client == nil {
		return nil,
			fmt.Errorf(
				"rate limiter requires Redis",
			)
	}

	cfg.Scope =
		normalizeRateLimitScope(
			cfg.Scope,
		)

	if cfg.Scope == "" {
		return nil,
			fmt.Errorf(
				"rate limit scope is required",
			)
	}

	if cfg.Limit <= 0 {
		return nil,
			fmt.Errorf(
				"rate limit must be greater than zero",
			)
	}

	if cfg.Window < time.Second {
		return nil,
			fmt.Errorf(
				"rate limit window must be at least one second",
			)
	}

	windowSeconds :=
		int64(
			cfg.Window /
				time.Second,
		)

	return func(
		c *gin.Context,
	) {
		clientIP :=
			strings.TrimSpace(
				c.ClientIP(),
			)

		if clientIP == "" {
			clientIP =
				"unknown"
		}

		key :=
			"rate:" +
				cfg.Scope +
				":" +
				platformsecurity.SHA256String(
					clientIP,
				)

		count, err :=
			rateLimitScript.Run(
				c.Request.Context(),
				client,
				[]string{
					key,
				},
				strconv.FormatInt(
					windowSeconds,
					10,
				),
			).Int64()
		if err != nil {
			platformerrors.Abort(
				c,
				platformerrors.ServiceUnavailable(
					err,
				),
			)

			return
		}

		remaining :=
			cfg.Limit -
				count

		if remaining < 0 {
			remaining = 0
		}

		c.Header(
			"X-RateLimit-Limit",
			strconv.FormatInt(
				cfg.Limit,
				10,
			),
		)

		c.Header(
			"X-RateLimit-Remaining",
			strconv.FormatInt(
				remaining,
				10,
			),
		)

		if count > cfg.Limit {
			c.Header(
				"Retry-After",
				strconv.FormatInt(
					windowSeconds,
					10,
				),
			)

			platformerrors.Abort(
				c,
				platformerrors.TooManyRequests(
					"RATE_LIMITED",
					"Too many requests",
				),
			)

			return
		}

		c.Next()
	}, nil
}

func normalizeRateLimitScope(
	value string,
) string {
	value =
		strings.ToLower(
			strings.TrimSpace(
				value,
			),
		)

	if value == "" {
		return ""
	}

	builder :=
		strings.Builder{}

	builder.Grow(
		len(value),
	)

	for index :=
		0; index < len(value); index++ {

		character :=
			value[index]

		if (character >= 'a' &&
			character <= 'z') ||
			(character >= '0' &&
				character <= '9') ||
			character == '-' ||
			character == '_' {

			builder.WriteByte(
				character,
			)
		}
	}

	return builder.String()
}
