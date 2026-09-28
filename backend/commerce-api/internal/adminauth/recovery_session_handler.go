package adminauth

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	platformerrors "project.local/commerce-api/internal/platform/errors"
)

func (h *Handler) ResetPasswordFromRecoverySession(
	c *gin.Context,
) {
	accessToken, err :=
		c.Cookie(
			AdminAccessCookieName,
		)
	if err != nil {
		writeRecoverySessionError(
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
		writeRecoverySessionError(
			c,
			err,
		)

		return
	}

	var request RecoveryPasswordResetRequest

	if err :=
		c.ShouldBindJSON(
			&request,
		); err != nil {

		writeRecoverySessionError(
			c,
			ErrInvalidRequest,
		)

		return
	}

	result, err :=
		h.service.ResetPasswordFromRecoverySession(
			c.Request.Context(),
			accessToken,
			csrfToken,
			request,
			metadataFromGin(
				c,
			),
		)
	if err != nil {
		writeRecoverySessionError(
			c,
			err,
		)

		return
	}

	// A successful reset revokes the recovery-authenticated Admin session.
	// Clear the browser cookies immediately; the returned challenge is used
	// only to enroll and confirm a fresh authenticator.
	h.clearSessionCookies(
		c,
	)

	c.JSON(
		http.StatusOK,
		gin.H{
			"data": result,
		},
	)
}

func writeRecoverySessionError(
	c *gin.Context,
	err error,
) {
	switch {
	case errors.Is(
		err,
		ErrRecoverySessionRequired,
	):
		platformerrors.Write(
			c,
			platformerrors.Forbidden(
				"ADMIN_RECOVERY_SESSION_REQUIRED",
				"A recent recovery-code authenticated Admin session is required",
			),
		)

	case errors.Is(
		err,
		ErrRecoverySessionExpired,
	):
		platformerrors.Write(
			c,
			platformerrors.Forbidden(
				"ADMIN_RECOVERY_SESSION_EXPIRED",
				"The recovery authorization window has expired",
			),
		)

	case errors.Is(
		err,
		ErrPasswordRequired,
	),
		errors.Is(
			err,
			ErrPasswordTooShort,
		),
		errors.Is(
			err,
			ErrPasswordTooLong,
		),
		errors.Is(
			err,
			ErrPasswordCommon,
		):

		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_STAFF_PASSWORD",
				err.Error(),
			),
		)

	default:
		writeAdminAuthError(
			c,
			err,
		)
	}
}
