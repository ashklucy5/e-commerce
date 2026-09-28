package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"project.local/commerce-api/internal/adminauth"
	platformerrors "project.local/commerce-api/internal/platform/errors"
	platformsecurity "project.local/commerce-api/internal/platform/security"
)

func RequireAdminCSRF(
	service *adminauth.Service,
) gin.HandlerFunc {
	if service == nil {
		panic("Admin authentication service is required for CSRF protection")
	}

	return func(
		c *gin.Context,
	) {
		if adminCSRFSafeMethod(
			c.Request.Method,
		) {
			c.Next()

			return
		}

		headerToken :=
			strings.TrimSpace(
				c.GetHeader(
					adminauth.AdminCSRFHeaderName,
				),
			)

		cookieToken, err :=
			c.Cookie(
				adminauth.AdminCSRFCookieName,
			)
		if err != nil ||
			headerToken == "" ||
			!platformsecurity.ConstantTimeEqual(
				headerToken,
				cookieToken,
			) {

			abortAdminCSRF(
				c,
			)

			return
		}

		accessToken, err :=
			c.Cookie(
				adminauth.AdminAccessCookieName,
			)
		if err != nil ||
			strings.TrimSpace(
				accessToken,
			) == "" {

			abortAdminUnauthorized(
				c,
			)

			return
		}

		principal, err :=
			service.AuthenticateAccessTokenWithCSRF(
				c.Request.Context(),
				accessToken,
				headerToken,
			)
		if err != nil {
			if errors.Is(
				err,
				adminauth.ErrInvalidCSRFToken,
			) {
				abortAdminCSRF(
					c,
				)

				return
			}

			writeAdminAuthenticationMiddlewareError(
				c,
				err,
			)

			return
		}

		adminauth.SetPrincipal(
			c,
			principal,
		)

		c.Next()
	}
}

func adminCSRFSafeMethod(
	method string,
) bool {
	switch method {
	case http.MethodGet,
		http.MethodHead,
		http.MethodOptions:

		return true

	default:
		return false
	}
}

func abortAdminCSRF(
	c *gin.Context,
) {
	platformerrors.Abort(
		c,
		platformerrors.Forbidden(
			"INVALID_ADMIN_CSRF_TOKEN",
			"Invalid Admin CSRF token",
		),
	)
}
