package middleware

import (
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const defaultHSTSMaxAge = 365 * 24 * time.Hour

type SecurityHeadersConfig struct {
	EnableHSTS bool

	HSTSMaxAge time.Duration

	HSTSIncludeSubDomains bool

	HSTSPreload bool

	ContentSecurityPolicy string

	FrameOptions string

	ReferrerPolicy string

	PermissionsPolicy string

	CrossOriginResourcePolicy string
}

func DefaultSecurityHeadersConfig() SecurityHeadersConfig {
	return SecurityHeadersConfig{
		HSTSMaxAge: defaultHSTSMaxAge,

		ContentSecurityPolicy: "default-src 'none'; frame-ancestors 'none'; base-uri 'none'",

		FrameOptions: "DENY",

		ReferrerPolicy: "strict-origin-when-cross-origin",

		PermissionsPolicy: "camera=(), microphone=(), geolocation=(), usb=()",

		CrossOriginResourcePolicy: "same-site",
	}
}

func SecurityHeaders(
	cfg SecurityHeadersConfig,
) gin.HandlerFunc {
	defaults :=
		DefaultSecurityHeadersConfig()

	if cfg.HSTSMaxAge <= 0 {
		cfg.HSTSMaxAge =
			defaults.HSTSMaxAge
	}

	if strings.TrimSpace(
		cfg.ContentSecurityPolicy,
	) == "" {
		cfg.ContentSecurityPolicy =
			defaults.ContentSecurityPolicy
	}

	if strings.TrimSpace(
		cfg.FrameOptions,
	) == "" {
		cfg.FrameOptions =
			defaults.FrameOptions
	}

	if strings.TrimSpace(
		cfg.ReferrerPolicy,
	) == "" {
		cfg.ReferrerPolicy =
			defaults.ReferrerPolicy
	}

	if strings.TrimSpace(
		cfg.PermissionsPolicy,
	) == "" {
		cfg.PermissionsPolicy =
			defaults.PermissionsPolicy
	}

	if strings.TrimSpace(
		cfg.CrossOriginResourcePolicy,
	) == "" {
		cfg.CrossOriginResourcePolicy =
			defaults.CrossOriginResourcePolicy
	}

	return func(
		c *gin.Context,
	) {
		c.Header(
			"X-Content-Type-Options",
			"nosniff",
		)

		c.Header(
			"X-Frame-Options",
			cfg.FrameOptions,
		)

		c.Header(
			"Referrer-Policy",
			cfg.ReferrerPolicy,
		)

		c.Header(
			"Permissions-Policy",
			cfg.PermissionsPolicy,
		)

		c.Header(
			"Cross-Origin-Resource-Policy",
			cfg.CrossOriginResourcePolicy,
		)

		c.Header(
			"Content-Security-Policy",
			cfg.ContentSecurityPolicy,
		)

		if cfg.EnableHSTS {
			value :=
				"max-age=" +
					strconv.FormatInt(
						int64(
							cfg.HSTSMaxAge/time.Second,
						),
						10,
					)

			if cfg.HSTSIncludeSubDomains {
				value +=
					"; includeSubDomains"
			}

			if cfg.HSTSPreload {
				value +=
					"; preload"
			}

			c.Header(
				"Strict-Transport-Security",
				value,
			)
		}

		c.Next()
	}
}
