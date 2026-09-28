package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"

	platformerrors "project.local/commerce-api/internal/platform/errors"
	platformlogger "project.local/commerce-api/internal/platform/logger"
)

func Recovery() gin.HandlerFunc {
	return func(
		c *gin.Context,
	) {
		defer func() {
			recovered :=
				recover()

			if recovered == nil {
				return
			}

			requestID, _ :=
				RequestIDFromContext(
					c,
				)

			platformlogger.
				FromContext(
					c.Request.Context(),
				).
				Error(
					"panic recovered from HTTP request",

					"request_id",
					requestID,

					"panic",
					fmt.Sprint(
						recovered,
					),

					"stack",
					string(
						debug.Stack(),
					),
				)

			if c.Writer.Written() {
				c.Abort()

				return
			}

			platformerrors.Abort(
				c,
				platformerrors.New(
					http.StatusInternalServerError,
					platformerrors.CodeInternal,
					"Internal server error",
				),
			)
		}()

		c.Next()
	}
}
