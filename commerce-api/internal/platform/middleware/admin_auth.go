package middleware

import (
	"errors"

	"github.com/gin-gonic/gin"

	"project.local/commerce-api/internal/adminauth"
	platformerrors "project.local/commerce-api/internal/platform/errors"
)

func RequireAdminAuth(
	service *adminauth.Service,
) gin.HandlerFunc {
	if service == nil {
		panic("Admin authentication service is required")
	}

	return func(
		c *gin.Context,
	) {
		accessToken, err :=
			c.Cookie(
				adminauth.AdminAccessCookieName,
			)
		if err != nil ||
			accessToken == "" {
			abortAdminUnauthorized(
				c,
			)

			return
		}

		principal, err :=
			service.AuthenticateAccessToken(
				c.Request.Context(),
				accessToken,
			)
		if err != nil {
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

func writeAdminAuthenticationMiddlewareError(
	c *gin.Context,
	err error,
) {
	switch {
	case errors.Is(
		err,
		adminauth.ErrAdminAccountDisabled,
	),
		errors.Is(
			err,
			adminauth.ErrAdminPanelAccessRequired,
		):

		platformerrors.Abort(
			c,
			platformerrors.Forbidden(
				"ADMIN_ACCESS_DENIED",
				"Admin access denied",
			),
		)

	case errors.Is(
		err,
		adminauth.ErrInvalidAccessToken,
	),
		errors.Is(
			err,
			adminauth.ErrSessionRevoked,
		):

		abortAdminUnauthorized(
			c,
		)

	default:
		platformerrors.Abort(
			c,
			platformerrors.Internal(
				err,
			),
		)
	}
}

func abortAdminUnauthorized(
	c *gin.Context,
) {
	platformerrors.Abort(
		c,
		platformerrors.Unauthorized(
			"ADMIN_AUTHENTICATION_REQUIRED",
			"Admin authentication required",
		),
	)
}
