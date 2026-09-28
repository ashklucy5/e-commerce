package adminauth

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	platformerrors "project.local/commerce-api/internal/platform/errors"
)

func (h *Handler) BeginSelfMFARotation(c *gin.Context) {
	accessToken, csrfToken, ok := h.selfSecuritySessionMaterial(c)
	if !ok {
		return
	}

	var request BeginMFARotationRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeSelfSecurityError(c, ErrInvalidRequest)
		return
	}

	result, err := h.service.BeginSelfMFARotation(
		c.Request.Context(),
		accessToken,
		csrfToken,
		request,
		metadataFromGin(c),
	)
	if err != nil {
		writeSelfSecurityError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (h *Handler) ConfirmSelfMFARotation(c *gin.Context) {
	accessToken, csrfToken, ok := h.selfSecuritySessionMaterial(c)
	if !ok {
		return
	}

	var request ConfirmMFARotationRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeSelfSecurityError(c, ErrInvalidRequest)
		return
	}

	result, err := h.service.ConfirmSelfMFARotation(
		c.Request.Context(),
		accessToken,
		csrfToken,
		request,
		metadataFromGin(c),
	)
	if err != nil {
		writeSelfSecurityError(c, err)
		return
	}

	h.setSessionCookies(c, result.Session.Material)

	c.JSON(
		http.StatusOK,
		gin.H{
			"data": SelfSecurityMutationResponse{
				Principal:     result.Session.Principal,
				CSRFToken:     result.Session.Material.CSRFToken,
				RecoveryCodes: result.RecoveryCodes,
			},
		},
	)
}

func (h *Handler) RegenerateSelfRecoveryCodes(c *gin.Context) {
	accessToken, csrfToken, ok := h.selfSecuritySessionMaterial(c)
	if !ok {
		return
	}

	var request RegenerateRecoveryCodesRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeSelfSecurityError(c, ErrInvalidRequest)
		return
	}

	result, err := h.service.RegenerateSelfRecoveryCodes(
		c.Request.Context(),
		accessToken,
		csrfToken,
		request,
		metadataFromGin(c),
	)
	if err != nil {
		writeSelfSecurityError(c, err)
		return
	}

	h.setSessionCookies(c, result.Session.Material)

	c.JSON(
		http.StatusOK,
		gin.H{
			"data": SelfSecurityMutationResponse{
				Principal:     result.Session.Principal,
				CSRFToken:     result.Session.Material.CSRFToken,
				RecoveryCodes: result.RecoveryCodes,
			},
		},
	)
}

func (h *Handler) selfSecuritySessionMaterial(
	c *gin.Context,
) (
	string,
	string,
	bool,
) {
	accessToken, err := c.Cookie(AdminAccessCookieName)
	if err != nil {
		writeSelfSecurityError(c, ErrInvalidAccessToken)
		return "", "", false
	}

	csrfToken, err := h.requestCSRF(c)
	if err != nil {
		writeSelfSecurityError(c, err)
		return "", "", false
	}

	return accessToken, csrfToken, true
}

func writeSelfSecurityError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrInvalidMFARotation):
		platformerrors.Write(
			c,
			platformerrors.Unauthorized(
				"INVALID_ADMIN_MFA_ROTATION",
				"Invalid or expired Admin MFA rotation",
			),
		)

	default:
		writeAdminAuthError(c, err)
	}
}
