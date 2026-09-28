package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	"project.local/commerce-api/internal/adminauth"
	platformerrors "project.local/commerce-api/internal/platform/errors"
)

func RequireAdminPermission(
	permission string,
) gin.HandlerFunc {
	permission =
		strings.TrimSpace(
			permission,
		)

	if permission == "" {
		panic("Admin permission is required")
	}

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

		if !principal.Staff.HasPermission(
			permission,
		) {
			platformerrors.Abort(
				c,
				platformerrors.Forbidden(
					"ADMIN_PERMISSION_DENIED",
					"Admin permission denied",
				),
			)

			return
		}

		c.Next()
	}
}
