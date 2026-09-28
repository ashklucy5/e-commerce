package adminauth

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	platformerrors "project.local/commerce-api/internal/platform/errors"
)

func (h *Handler) ChangeSelfPassword(c *gin.Context) {
	accessToken, csrfToken, ok := h.selfSecuritySessionMaterial(c)
	if !ok {
		return
	}

	var request ChangeSelfPasswordRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeSelfPasswordChangeError(c, ErrInvalidRequest)
		return
	}

	result, err := h.service.ChangeSelfPassword(
		c.Request.Context(),
		accessToken,
		csrfToken,
		request,
		metadataFromGin(c),
	)
	if err != nil {
		writeSelfPasswordChangeError(c, err)
		return
	}

	h.setSessionCookies(c, result.Material)

	c.JSON(
		http.StatusOK,
		gin.H{
			"data": ChangeSelfPasswordResponse{
				Principal: result.Principal,
				CSRFToken: result.Material.CSRFToken,
			},
		},
	)
}

func writeSelfPasswordChangeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrPasswordUnchanged):
		platformerrors.Write(
			c,
			platformerrors.Conflict(
				"ADMIN_PASSWORD_UNCHANGED",
				"New Admin password must be different from the current password",
			),
		)

	case errors.Is(err, ErrPasswordRequired),
		errors.Is(err, ErrPasswordTooShort),
		errors.Is(err, ErrPasswordTooLong),
		errors.Is(err, ErrPasswordCommon):
		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_STAFF_PASSWORD",
				err.Error(),
			),
		)

	default:
		writeSelfSecurityError(c, err)
	}
}
