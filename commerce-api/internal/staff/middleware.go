package staff

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const staffContextKey = "auth_staff_account"

func RequireAuth(
	service *Service,
) gin.HandlerFunc {
	return func(
		c *gin.Context,
	) {
		token, err :=
			extractBearerToken(
				c.GetHeader(
					"Authorization",
				),
			)
		if err != nil {
			writeStaffJSONError(
				c,
				http.StatusUnauthorized,
				"INVALID_STAFF_ACCESS_TOKEN",
				"Invalid or expired staff access token",
			)

			c.Abort()

			return
		}

		account, err :=
			service.AuthenticateAccessToken(
				c.Request.Context(),
				token,
			)
		if err != nil {
			writeStaffError(
				c,
				err,
			)

			c.Abort()

			return
		}

		c.Set(
			staffContextKey,
			account,
		)

		c.Next()
	}
}

func RequirePermission(
	permission string,
) gin.HandlerFunc {
	return func(
		c *gin.Context,
	) {
		account, ok :=
			AccountFromContext(
				c,
			)

		if !ok ||
			!account.HasPermission(
				permission,
			) {
			writeStaffError(
				c,
				ErrForbidden,
			)

			c.Abort()

			return
		}

		c.Next()
	}
}

func AccountFromContext(
	c *gin.Context,
) (Account, bool) {
	value, exists :=
		c.Get(
			staffContextKey,
		)
	if !exists {
		return Account{}, false
	}

	account, ok :=
		value.(Account)

	if !ok ||
		account.ID == "" {
		return Account{}, false
	}

	return account, true
}

func extractBearerToken(
	header string,
) (string, error) {
	parts :=
		strings.Fields(
			header,
		)

	if len(parts) != 2 ||
		!strings.EqualFold(
			parts[0],
			"Bearer",
		) ||
		strings.TrimSpace(
			parts[1],
		) == "" {
		return "",
			ErrInvalidAccessToken
	}

	return parts[1], nil
}
