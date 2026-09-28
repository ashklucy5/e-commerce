package adminauth

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	platformerrors "project.local/commerce-api/internal/platform/errors"
	platformsecurity "project.local/commerce-api/internal/platform/security"
)

const (
	AdminAccessCookieName = "commerce_admin_access"

	AdminRefreshCookieName = "commerce_admin_refresh"

	AdminCSRFCookieName = "commerce_admin_csrf"

	AdminCSRFHeaderName = "X-CSRF-Token"

	AdminCookiePath = "/api/v1/admin"
)

type CookieConfig struct {
	Secure bool

	Domain string
}

type Handler struct {
	service *Service

	cookies CookieConfig
}

func NewHandler(
	service *Service,
	cookies CookieConfig,
) *Handler {
	return &Handler{
		service: service,

		cookies: cookies,
	}
}

func (h *Handler) Login(
	c *gin.Context,
) {
	var request LoginRequest

	if err :=
		c.ShouldBindJSON(
			&request,
		); err != nil {

		writeAdminAuthError(
			c,
			ErrInvalidRequest,
		)

		return
	}

	result, err :=
		h.service.Login(
			c.Request.Context(),
			request,
			metadataFromGin(
				c,
			),
		)
	if err != nil {
		writeAdminAuthError(
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

func (h *Handler) BeginMFAEnrollment(
	c *gin.Context,
) {
	var request MFAEnrollRequest

	if err :=
		c.ShouldBindJSON(
			&request,
		); err != nil {

		writeAdminAuthError(
			c,
			ErrInvalidRequest,
		)

		return
	}

	result, err :=
		h.service.BeginMFAEnrollment(
			c.Request.Context(),
			request,
			metadataFromGin(
				c,
			),
		)
	if err != nil {
		writeAdminAuthError(
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

func (h *Handler) ConfirmMFAEnrollment(
	c *gin.Context,
) {
	var request MFAConfirmEnrollmentRequest

	if err :=
		c.ShouldBindJSON(
			&request,
		); err != nil {

		writeAdminAuthError(
			c,
			ErrInvalidRequest,
		)

		return
	}

	result, err :=
		h.service.ConfirmMFAEnrollment(
			c.Request.Context(),
			request,
			metadataFromGin(
				c,
			),
		)
	if err != nil {
		writeAdminAuthError(
			c,
			err,
		)

		return
	}

	h.setSessionCookies(
		c,
		result.Session.Material,
	)

	c.JSON(
		http.StatusOK,
		gin.H{
			"data": MFAEnrollmentCompleteResponse{
				Principal: result.Session.Principal,

				CSRFToken: result.Session.Material.CSRFToken,

				RecoveryCodes: result.RecoveryCodes,
			},
		},
	)
}

func (h *Handler) VerifyMFA(
	c *gin.Context,
) {
	var request MFAVerifyRequest

	if err :=
		c.ShouldBindJSON(
			&request,
		); err != nil {

		writeAdminAuthError(
			c,
			ErrInvalidRequest,
		)

		return
	}

	result, err :=
		h.service.VerifyMFA(
			c.Request.Context(),
			request,
			metadataFromGin(
				c,
			),
		)
	if err != nil {
		writeAdminAuthError(
			c,
			err,
		)

		return
	}

	h.setSessionCookies(
		c,
		result.Material,
	)

	c.JSON(
		http.StatusOK,
		gin.H{
			"data": SessionResponse{
				Principal: result.Principal,

				CSRFToken: result.Material.CSRFToken,
			},
		},
	)
}

func (h *Handler) Refresh(
	c *gin.Context,
) {
	refreshToken, err :=
		c.Cookie(
			AdminRefreshCookieName,
		)
	if err != nil {
		writeAdminAuthError(
			c,
			ErrInvalidRefreshToken,
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
		h.service.Refresh(
			c.Request.Context(),
			refreshToken,
			csrfToken,
			metadataFromGin(
				c,
			),
		)
	if err != nil {
		writeAdminAuthError(
			c,
			err,
		)

		return
	}

	h.setSessionCookies(
		c,
		result.Material,
	)

	c.JSON(
		http.StatusOK,
		gin.H{
			"data": RefreshResponse{
				Principal: result.Principal,

				CSRFToken: result.Material.CSRFToken,
			},
		},
	)
}

func (h *Handler) Logout(
	c *gin.Context,
) {
	accessToken, err :=
		c.Cookie(
			AdminAccessCookieName,
		)
	if err != nil {
		h.clearSessionCookies(
			c,
		)

		c.Status(
			http.StatusNoContent,
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
		h.service.Logout(
			c.Request.Context(),
			accessToken,
			csrfToken,
			metadataFromGin(
				c,
			),
		); err != nil {

		writeAdminAuthError(
			c,
			err,
		)

		return
	}

	h.clearSessionCookies(
		c,
	)

	c.Status(
		http.StatusNoContent,
	)
}

func (h *Handler) Me(
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
		h.service.Me(
			c.Request.Context(),
			accessToken,
		)
	if err != nil {
		writeAdminAuthError(
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

func (h *Handler) requestCSRF(
	c *gin.Context,
) (
	string,
	error,
) {
	header :=
		strings.TrimSpace(
			c.GetHeader(
				AdminCSRFHeaderName,
			),
		)

	if header == "" {
		return "",
			ErrInvalidCSRFToken
	}

	cookie, err :=
		c.Cookie(
			AdminCSRFCookieName,
		)

	if err != nil ||
		!platformsecurity.ConstantTimeEqual(
			header,
			cookie,
		) {

		return "",
			ErrInvalidCSRFToken
	}

	return header,
		nil
}

func (h *Handler) setSessionCookies(
	c *gin.Context,
	material AdminSessionMaterial,
) {
	c.SetSameSite(
		http.SameSiteStrictMode,
	)

	c.SetCookie(
		AdminAccessCookieName,
		material.AccessToken,
		secondsUntil(
			material.AccessExpiresAt,
		),
		AdminCookiePath,
		h.cookies.Domain,
		h.cookies.Secure,
		true,
	)

	c.SetCookie(
		AdminRefreshCookieName,
		material.RefreshToken,
		secondsUntil(
			material.RefreshExpiresAt,
		),
		AdminCookiePath,
		h.cookies.Domain,
		h.cookies.Secure,
		true,
	)

	c.SetCookie(
		AdminCSRFCookieName,
		material.CSRFToken,
		secondsUntil(
			material.RefreshExpiresAt,
		),
		AdminCookiePath,
		h.cookies.Domain,
		h.cookies.Secure,
		false,
	)
}

func (h *Handler) clearSessionCookies(
	c *gin.Context,
) {
	c.SetSameSite(
		http.SameSiteStrictMode,
	)

	c.SetCookie(
		AdminAccessCookieName,
		"",
		-1,
		AdminCookiePath,
		h.cookies.Domain,
		h.cookies.Secure,
		true,
	)

	c.SetCookie(
		AdminRefreshCookieName,
		"",
		-1,
		AdminCookiePath,
		h.cookies.Domain,
		h.cookies.Secure,
		true,
	)

	c.SetCookie(
		AdminCSRFCookieName,
		"",
		-1,
		AdminCookiePath,
		h.cookies.Domain,
		h.cookies.Secure,
		false,
	)
}

func secondsUntil(
	expiresAt time.Time,
) int {
	seconds :=
		int(
			time.Until(
				expiresAt,
			).Seconds(),
		)

	if seconds < 1 {
		return 1
	}

	return seconds
}

func metadataFromGin(
	c *gin.Context,
) ClientMetadata {
	return ClientMetadata{
		IPAddress: strings.TrimSpace(
			c.ClientIP(),
		),

		UserAgent: strings.TrimSpace(
			c.Request.UserAgent(),
		),

		RequestID: strings.TrimSpace(
			c.GetHeader(
				"X-Request-ID",
			),
		),
	}
}

func writeAdminAuthError(
	c *gin.Context,
	err error,
) {
	var apiErr error

	switch {
	case errors.Is(
		err,
		ErrInvalidRequest,
	):
		apiErr =
			platformerrors.BadRequest(
				"INVALID_ADMIN_AUTH_REQUEST",
				"Invalid Admin authentication request",
			)

	case errors.Is(
		err,
		ErrInvalidCredentials,
	):
		apiErr =
			platformerrors.Unauthorized(
				"INVALID_ADMIN_CREDENTIALS",
				"Invalid Admin credentials",
			)

	case errors.Is(
		err,
		ErrLoginBlocked,
	):
		apiErr =
			platformerrors.TooManyRequests(
				"ADMIN_LOGIN_BLOCKED",
				"Too many failed Admin login attempts",
			)

	case errors.Is(
		err,
		ErrMFABlocked,
	):
		apiErr =
			platformerrors.TooManyRequests(
				"ADMIN_MFA_BLOCKED",
				"Too many failed Admin MFA attempts",
			)

	case errors.Is(
		err,
		ErrAuthUnavailable,
	):
		apiErr =
			platformerrors.ServiceUnavailable(
				err,
			)

	case errors.Is(
		err,
		ErrInvalidChallenge,
	),
		errors.Is(
			err,
			ErrChallengeExpired,
		),
		errors.Is(
			err,
			ErrChallengeConsumed,
		):

		apiErr =
			platformerrors.Unauthorized(
				"INVALID_ADMIN_CHALLENGE",
				"Invalid or expired Admin login challenge",
			)

	case errors.Is(
		err,
		ErrChallengeLocked,
	):
		apiErr =
			platformerrors.TooManyRequests(
				"ADMIN_CHALLENGE_LOCKED",
				"Admin login challenge is locked",
			)

	case errors.Is(
		err,
		ErrMFAAlreadyEnrolled,
	):
		apiErr =
			platformerrors.Conflict(
				"ADMIN_MFA_ALREADY_ENROLLED",
				"Admin MFA is already enrolled",
			)

	case errors.Is(
		err,
		ErrMFANotEnrolled,
	),
		errors.Is(
			err,
			ErrMFAEnrollmentNotPending,
		):

		apiErr =
			platformerrors.Conflict(
				"ADMIN_MFA_NOT_READY",
				"Admin MFA enrollment is not ready",
			)

	case errors.Is(
		err,
		ErrInvalidMFACode,
	),
		errors.Is(
			err,
			ErrInvalidTOTPCode,
		),
		errors.Is(
			err,
			ErrInvalidRecoveryCode,
		):

		apiErr =
			platformerrors.Unauthorized(
				"INVALID_ADMIN_MFA_CODE",
				"Invalid Admin MFA code",
			)

	case errors.Is(
		err,
		ErrInvalidAccessToken,
	),
		errors.Is(
			err,
			ErrSessionRevoked,
		):

		apiErr =
			platformerrors.Unauthorized(
				"INVALID_ADMIN_ACCESS_TOKEN",
				"Invalid or expired Admin session",
			)

	case errors.Is(
		err,
		ErrInvalidRefreshToken,
	):
		apiErr =
			platformerrors.Unauthorized(
				"INVALID_ADMIN_REFRESH_TOKEN",
				"Invalid or expired Admin refresh token",
			)

	case errors.Is(
		err,
		ErrInvalidCSRFToken,
	):
		apiErr =
			platformerrors.Forbidden(
				"INVALID_ADMIN_CSRF_TOKEN",
				"Invalid Admin CSRF token",
			)

	case errors.Is(
		err,
		ErrAdminAccountDisabled,
	),
		errors.Is(
			err,
			ErrAdminPanelAccessRequired,
		):

		apiErr =
			platformerrors.Forbidden(
				"ADMIN_ACCESS_DENIED",
				"Admin access denied",
			)

	default:
		apiErr =
			platformerrors.Internal(
				err,
			)
	}

	platformerrors.Write(
		c,
		apiErr,
	)
}
