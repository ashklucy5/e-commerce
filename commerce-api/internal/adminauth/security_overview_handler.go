package adminauth

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	platformerrors "project.local/commerce-api/internal/platform/errors"
)

func (h *Handler) SelfSecurityOverview(
	c *gin.Context,
) {
	accessToken, err :=
		c.Cookie(
			AdminAccessCookieName,
		)
	if err != nil {
		writeAdminAuthError(
			c,
			ErrInvalidAccessToken,
		)

		return
	}

	result, err :=
		h.service.SelfSecurityOverview(
			c.Request.Context(),
			accessToken,
		)
	if err != nil {
		writeSecurityOverviewError(
			c,
			err,
		)

		return
	}

	c.JSON(
		http.StatusOK,
		gin.H{
			"data": result,
		},
	)
}

func (h *Handler) RevokeSelfSecuritySession(
	c *gin.Context,
) {
	accessToken, err :=
		c.Cookie(
			AdminAccessCookieName,
		)
	if err != nil {
		writeAdminAuthError(
			c,
			ErrInvalidAccessToken,
		)

		return
	}

	csrfToken, err :=
		h.requestCSRF(
			c,
		)
	if err != nil {
		writeAdminAuthError(
			c,
			err,
		)

		return
	}

	if err :=
		h.service.RevokeSelfSecuritySession(
			c.Request.Context(),
			accessToken,
			csrfToken,
			strings.TrimSpace(
				c.Param(
					"session_id",
				),
			),
			metadataFromGin(
				c,
			),
		); err != nil {

		writeSecurityOverviewError(
			c,
			err,
		)

		return
	}

	c.Status(
		http.StatusNoContent,
	)
}

func (h *Handler) RevokeOtherSelfSecuritySessions(
	c *gin.Context,
) {
	accessToken, err :=
		c.Cookie(
			AdminAccessCookieName,
		)
	if err != nil {
		writeAdminAuthError(
			c,
			ErrInvalidAccessToken,
		)

		return
	}

	csrfToken, err :=
		h.requestCSRF(
			c,
		)
	if err != nil {
		writeAdminAuthError(
			c,
			err,
		)

		return
	}

	result, err :=
		h.service.RevokeOtherSelfSecuritySessions(
			c.Request.Context(),
			accessToken,
			csrfToken,
			metadataFromGin(
				c,
			),
		)
	if err != nil {
		writeSecurityOverviewError(
			c,
			err,
		)

		return
	}

	c.JSON(
		http.StatusOK,
		gin.H{
			"data": result,
		},
	)
}

func writeSecurityOverviewError(
	c *gin.Context,
	err error,
) {
	switch {
	case errors.Is(
		err,
		ErrCannotRevokeCurrentAdminSession,
	):
		platformerrors.Write(
			c,
			platformerrors.Conflict(
				"ADMIN_CURRENT_SESSION_REQUIRES_LOGOUT",
				"Use logout to end the current Admin session",
			),
		)

	case errors.Is(
		err,
		ErrAdminSessionNotFound,
	):
		platformerrors.Write(
			c,
			platformerrors.NotFound(
				"ADMIN_SESSION_NOT_FOUND",
				"Admin session was not found",
			),
		)

	default:
		writeAdminAuthError(
			c,
			err,
		)
	}
}
