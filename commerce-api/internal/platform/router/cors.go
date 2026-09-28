package router

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"

	platformmiddleware "project.local/commerce-api/internal/platform/middleware"
)

func newCommerceCORS(
	appAllowedOrigins []string,
	adminAllowedOrigins []string,
) (
	gin.HandlerFunc,
	error,
) {
	appConfig :=
		platformmiddleware.
			DefaultCORSConfig()

	appConfig.AllowedOrigins =
		append(
			[]string(nil),
			appAllowedOrigins...,
		)

	// Customer authentication is bearer-token based.
	// Browser cookies are not required for the storefront API.
	appConfig.AllowCredentials =
		false

	appConfig.AllowedHeaders =
		append(
			appConfig.AllowedHeaders,
			"X-Checkout-Key",
		)

	appCORS, err :=
		platformmiddleware.NewCORS(
			appConfig,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"customer CORS configuration: %w",
				err,
			)
	}

	adminConfig :=
		platformmiddleware.
			DefaultCORSConfig()

	adminConfig.AllowedOrigins =
		append(
			[]string(nil),
			adminAllowedOrigins...,
		)

	// Admin authentication uses secure cookies + CSRF.
	adminConfig.AllowCredentials =
		true

	adminCORS, err :=
		platformmiddleware.NewCORS(
			adminConfig,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"Admin CORS configuration: %w",
				err,
			)
	}

	return func(
			c *gin.Context,
		) {
			path :=
				c.Request.URL.Path

			switch {

			case pathWithin(
				path,
				"/api/v1/admin",
			),
				pathWithin(
					path,
					"/api/v1/support",
				):

				adminCORS(
					c,
				)

			case pathWithin(
				path,
				"/api/v1",
			):

				appCORS(
					c,
				)

			default:

				c.Next()
			}
		},
		nil
}

func pathWithin(
	path string,
	prefix string,
) bool {
	return path == prefix ||
		strings.HasPrefix(
			path,
			prefix+"/",
		)
}
