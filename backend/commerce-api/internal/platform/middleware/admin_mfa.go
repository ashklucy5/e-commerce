package middleware

import (
	"github.com/gin-gonic/gin"

	"project.local/commerce-api/internal/adminauth"
	platformerrors "project.local/commerce-api/internal/platform/errors"
)

func RequireAdminMFA() gin.HandlerFunc {
	return func(
		c *gin.Context,
	) {
		principal, ok :=
			adminauth.PrincipalFromContext(
				c,
			)
		if !ok {
			abortAdminUnauthorized(
				c,
			)

			return
		}

		if principal.MFAVerifiedAt.IsZero() {
			platformerrors.Abort(
				c,
				platformerrors.Forbidden(
					"ADMIN_MFA_REQUIRED",
					"Admin MFA verification required",
				),
			)

			return
		}

		c.Next()
	}
}
