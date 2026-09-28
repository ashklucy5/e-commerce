package router

import (
	"github.com/gin-gonic/gin"

	"project.local/commerce-api/internal/adminauth"
	platformerrors "project.local/commerce-api/internal/platform/errors"
)

func adminActorHandler[T any](
	factory func(
		actorID string,
	) *T,
	invoke func(
		*T,
		*gin.Context,
	),
) gin.HandlerFunc {
	return func(
		c *gin.Context,
	) {
		actorID, ok :=
			adminauth.ActorIDFromContext(
				c,
			)
		if !ok {
			platformerrors.Abort(
				c,
				platformerrors.Unauthorized(
					"ADMIN_AUTHENTICATION_REQUIRED",
					"Admin authentication required",
				),
			)

			return
		}

		invoke(
			factory(
				actorID,
			),
			c,
		)
	}
}
