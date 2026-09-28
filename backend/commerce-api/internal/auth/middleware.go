package auth

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const customerIDContextKey = "auth_customer_id"

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
			writeAuthJSONError(
				c,
				http.StatusUnauthorized,
				"INVALID_ACCESS_TOKEN",
				"Invalid or expired access token",
			)

			c.Abort()

			return
		}

		customer, err :=
			service.AuthenticateAccessToken(
				c.Request.Context(),
				token,
			)
		if err != nil {
			writeAuthError(
				c,
				err,
			)

			c.Abort()

			return
		}

		c.Set(
			customerIDContextKey,
			customer.ID,
		)

		c.Next()
	}
}

func CustomerIDFromContext(
	c *gin.Context,
) (string, bool) {
	value, exists :=
		c.Get(
			customerIDContextKey,
		)
	if !exists {
		return "", false
	}

	customerID, ok :=
		value.(string)

	if !ok ||
		customerID == "" {
		return "", false
	}

	return customerID, true
}
func OptionalAuth(
	service *Service,
) gin.HandlerFunc {
	return func(
		c *gin.Context,
	) {
		header :=
			strings.TrimSpace(
				c.GetHeader(
					"Authorization",
				),
			)

		if header == "" {
			c.Next()

			return
		}

		token, err :=
			extractBearerToken(
				header,
			)
		if err != nil {
			writeAuthError(
				c,
				err,
			)

			c.Abort()

			return
		}

		customer, err :=
			service.AuthenticateAccessToken(
				c.Request.Context(),
				token,
			)
		if err != nil {
			writeAuthError(
				c,
				err,
			)

			c.Abort()

			return
		}

		c.Set(
			customerIDContextKey,
			customer.ID,
		)

		c.Next()
	}
}
