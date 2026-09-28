package auth

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(
	service *Service,
) *Handler {
	return &Handler{
		service: service,
	}
}

/*
Register begins phone-verified registration.

POST /api/v1/auth/register
*/
func (h *Handler) Register(
	c *gin.Context,
) {
	var request RegisterRequest

	if err :=
		c.ShouldBindJSON(
			&request,
		); err != nil {

		writeAuthJSONError(
			c,
			http.StatusBadRequest,
			"INVALID_REQUEST",
			"Invalid registration request",
		)

		return
	}

	result, err :=
		h.service.Register(
			c.Request.Context(),
			request,
		)
	if err != nil {
		writeAuthError(
			c,
			err,
		)

		return
	}

	status :=
		http.StatusCreated

	if result.VerificationRequired {
		status =
			http.StatusAccepted
	}

	c.JSON(
		status,
		gin.H{
			"data": result,
		},
	)
}

/*
VerifyRegistration completes registration.

POST /api/v1/auth/register/verify
*/
func (h *Handler) VerifyRegistration(
	c *gin.Context,
) {
	var request RegisterVerifyRequest

	if err :=
		c.ShouldBindJSON(
			&request,
		); err != nil {

		writeAuthJSONError(
			c,
			http.StatusBadRequest,
			"INVALID_REQUEST",
			"Invalid verification request",
		)

		return
	}

	result, err :=
		h.service.VerifyRegistration(
			c.Request.Context(),
			request,
		)
	if err != nil {
		writeAuthError(
			c,
			err,
		)

		return
	}

	c.JSON(
		http.StatusCreated,
		gin.H{
			"data": result,
		},
	)
}

/*
ResendRegistration sends a new registration OTP.

POST /api/v1/auth/register/resend
*/
func (h *Handler) ResendRegistration(
	c *gin.Context,
) {
	var request RegisterResendRequest

	if err :=
		c.ShouldBindJSON(
			&request,
		); err != nil {

		writeAuthJSONError(
			c,
			http.StatusBadRequest,
			"INVALID_REQUEST",
			"Invalid resend request",
		)

		return
	}

	challenge, err :=
		h.service.ResendRegistration(
			c.Request.Context(),
			request,
		)
	if err != nil {
		writeAuthError(
			c,
			err,
		)

		return
	}

	c.JSON(
		http.StatusOK,
		gin.H{
			"data": challenge,
		},
	)
}

/*
ForgotPassword begins password recovery.

The API deliberately does not reveal whether an account
exists for the submitted phone number.

POST /api/v1/auth/password/forgot
*/
func (h *Handler) ForgotPassword(
	c *gin.Context,
) {
	var request ForgotPasswordRequest

	if err :=
		c.ShouldBindJSON(
			&request,
		); err != nil {

		writeAuthJSONError(
			c,
			http.StatusBadRequest,
			"INVALID_REQUEST",
			"Invalid password recovery request",
		)

		return
	}

	challenge, err :=
		h.service.ForgotPassword(
			c.Request.Context(),
			request,
		)
	if err != nil {
		writeAuthError(
			c,
			err,
		)

		return
	}

	c.JSON(
		http.StatusAccepted,
		gin.H{
			"data": challenge,
		},
	)
}

/*
VerifyForgotPassword verifies the recovery OTP and returns
a short-lived one-time password-reset authorization.

POST /api/v1/auth/password/forgot/verify
*/
func (h *Handler) VerifyForgotPassword(
	c *gin.Context,
) {
	var request ForgotPasswordVerifyRequest

	if err :=
		c.ShouldBindJSON(
			&request,
		); err != nil {

		writeAuthJSONError(
			c,
			http.StatusBadRequest,
			"INVALID_REQUEST",
			"Invalid password recovery verification",
		)

		return
	}

	grant, err :=
		h.service.VerifyForgotPassword(
			c.Request.Context(),
			request,
		)
	if err != nil {
		writeAuthError(
			c,
			err,
		)

		return
	}

	c.JSON(
		http.StatusOK,
		gin.H{
			"data": grant,
		},
	)
}

/*
ResendForgotPassword rotates the recovery OTP.

POST /api/v1/auth/password/forgot/resend
*/
func (h *Handler) ResendForgotPassword(
	c *gin.Context,
) {
	var request ForgotPasswordResendRequest

	if err :=
		c.ShouldBindJSON(
			&request,
		); err != nil {

		writeAuthJSONError(
			c,
			http.StatusBadRequest,
			"INVALID_REQUEST",
			"Invalid password recovery resend request",
		)

		return
	}

	challenge, err :=
		h.service.ResendForgotPassword(
			c.Request.Context(),
			request,
		)
	if err != nil {
		writeAuthError(
			c,
			err,
		)

		return
	}

	c.JSON(
		http.StatusOK,
		gin.H{
			"data": challenge,
		},
	)
}

/*
ResetPassword changes the password after a successful
phone verification/reset grant.

It also returns a fresh authenticated session.

POST /api/v1/auth/password/reset
*/
func (h *Handler) ResetPassword(
	c *gin.Context,
) {
	var request ResetPasswordRequest

	if err :=
		c.ShouldBindJSON(
			&request,
		); err != nil {

		writeAuthJSONError(
			c,
			http.StatusBadRequest,
			"INVALID_REQUEST",
			"Invalid password reset request",
		)

		return
	}

	result, err :=
		h.service.ResetPassword(
			c.Request.Context(),
			request,
		)
	if err != nil {
		writeAuthError(
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

/*
ChangePassword changes a password for an authenticated
customer without requiring an OTP.

POST /api/v1/auth/password/change
*/
func (h *Handler) ChangePassword(
	c *gin.Context,
) {
	token, err :=
		extractBearerToken(
			c.GetHeader(
				"Authorization",
			),
		)
	if err != nil {
		writeAuthError(
			c,
			err,
		)

		return
	}

	var request ChangePasswordRequest

	if err :=
		c.ShouldBindJSON(
			&request,
		); err != nil {

		writeAuthJSONError(
			c,
			http.StatusBadRequest,
			"INVALID_REQUEST",
			"Invalid password change request",
		)

		return
	}

	if err :=
		h.service.ChangePassword(
			c.Request.Context(),
			token,
			request,
		); err != nil {

		writeAuthError(
			c,
			err,
		)

		return
	}

	c.Status(
		http.StatusNoContent,
	)
}

/*
Login handles:

POST /api/v1/auth/login
*/
func (h *Handler) Login(
	c *gin.Context,
) {
	var request LoginRequest

	if err :=
		c.ShouldBindJSON(
			&request,
		); err != nil {

		writeAuthJSONError(
			c,
			http.StatusBadRequest,
			"INVALID_REQUEST",
			"Invalid login request",
		)

		return
	}

	result, err :=
		h.service.Login(
			c.Request.Context(),
			request,
			c.ClientIP(),
		)
	if err != nil {
		writeAuthError(
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

/*
Refresh handles:

POST /api/v1/auth/refresh
*/
func (h *Handler) Refresh(
	c *gin.Context,
) {
	var request RefreshRequest

	if err :=
		c.ShouldBindJSON(
			&request,
		); err != nil {

		writeAuthJSONError(
			c,
			http.StatusBadRequest,
			"INVALID_REQUEST",
			"Invalid refresh request",
		)

		return
	}

	result, err :=
		h.service.Refresh(
			c.Request.Context(),
			request,
		)
	if err != nil {
		writeAuthError(
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

/*
Logout handles:

POST /api/v1/auth/logout
*/
func (h *Handler) Logout(
	c *gin.Context,
) {
	token, err :=
		extractBearerToken(
			c.GetHeader(
				"Authorization",
			),
		)
	if err != nil {
		writeAuthError(
			c,
			err,
		)

		return
	}

	if err :=
		h.service.Logout(
			c.Request.Context(),
			token,
		); err != nil {

		writeAuthError(
			c,
			err,
		)

		return
	}

	c.Status(
		http.StatusNoContent,
	)
}

func extractBearerToken(
	header string,
) (
	string,
	error,
) {
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

	return parts[1],
		nil
}

func writeAuthError(
	c *gin.Context,
	err error,
) {
	switch {
	case errors.Is(
		err,
		ErrInvalidRequest,
	):
		writeAuthJSONError(
			c,
			http.StatusBadRequest,
			"INVALID_REQUEST",
			"Invalid request",
		)

	case errors.Is(
		err,
		ErrInvalidPhone,
	):
		writeAuthJSONError(
			c,
			http.StatusBadRequest,
			"INVALID_PHONE",
			err.Error(),
		)

	case errors.Is(
		err,
		ErrInvalidEmail,
	):
		writeAuthJSONError(
			c,
			http.StatusBadRequest,
			"INVALID_EMAIL",
			err.Error(),
		)

	case errors.Is(
		err,
		ErrInvalidPassword,
	):
		writeAuthJSONError(
			c,
			http.StatusBadRequest,
			"INVALID_PASSWORD",
			err.Error(),
		)

	case errors.Is(
		err,
		ErrInvalidFullName,
	):
		writeAuthJSONError(
			c,
			http.StatusBadRequest,
			"INVALID_FULL_NAME",
			err.Error(),
		)

	case errors.Is(
		err,
		ErrPasswordConfirmation,
	):
		writeAuthJSONError(
			c,
			http.StatusBadRequest,
			"PASSWORD_CONFIRMATION_MISMATCH",
			"Password confirmation does not match",
		)

	case errors.Is(
		err,
		ErrCurrentPasswordIncorrect,
	):
		writeAuthJSONError(
			c,
			http.StatusBadRequest,
			"CURRENT_PASSWORD_INCORRECT",
			"Current password is incorrect",
		)

	case errors.Is(
		err,
		ErrPasswordUnchanged,
	):
		writeAuthJSONError(
			c,
			http.StatusBadRequest,
			"PASSWORD_UNCHANGED",
			"New password must be different from the current password",
		)

	case errors.Is(
		err,
		ErrPhoneInUse,
	):
		writeAuthJSONError(
			c,
			http.StatusConflict,
			"PHONE_ALREADY_REGISTERED",
			"An account already exists for this phone number",
		)

	case errors.Is(
		err,
		ErrEmailInUse,
	):
		writeAuthJSONError(
			c,
			http.StatusConflict,
			"EMAIL_ALREADY_REGISTERED",
			err.Error(),
		)

	case errors.Is(
		err,
		ErrInvalidVerification,
	):
		writeAuthJSONError(
			c,
			http.StatusBadRequest,
			"INVALID_VERIFICATION",
			"Invalid or expired verification",
		)

	case errors.Is(
		err,
		ErrVerificationExpired,
	):
		writeAuthJSONError(
			c,
			http.StatusGone,
			"VERIFICATION_EXPIRED",
			"Verification has expired",
		)

	case errors.Is(
		err,
		ErrInvalidVerificationCode,
	):
		writeAuthJSONError(
			c,
			http.StatusBadRequest,
			"INVALID_VERIFICATION_CODE",
			"The verification code is incorrect",
		)

	case errors.Is(
		err,
		ErrVerificationAttemptsExceeded,
	):
		writeAuthJSONError(
			c,
			http.StatusTooManyRequests,
			"VERIFICATION_ATTEMPTS_EXCEEDED",
			"Too many incorrect verification attempts. Start again.",
		)

	case errors.Is(
		err,
		ErrVerificationResendTooSoon,
	):
		writeAuthJSONError(
			c,
			http.StatusTooManyRequests,
			"VERIFICATION_RESEND_TOO_SOON",
			"Please wait before requesting another verification code",
		)

	case errors.Is(
		err,
		ErrVerificationResendLimit,
	):
		writeAuthJSONError(
			c,
			http.StatusTooManyRequests,
			"VERIFICATION_RESEND_LIMIT",
			"Too many verification codes requested. Start again.",
		)

	case errors.Is(
		err,
		ErrVerificationBusy,
	):
		writeAuthJSONError(
			c,
			http.StatusConflict,
			"VERIFICATION_BUSY",
			"Verification is already being processed",
		)

	case errors.Is(
		err,
		ErrInvalidPasswordResetGrant,
	):
		writeAuthJSONError(
			c,
			http.StatusBadRequest,
			"INVALID_PASSWORD_RESET",
			"Password reset authorization is invalid",
		)

	case errors.Is(
		err,
		ErrPasswordResetGrantExpired,
	):
		writeAuthJSONError(
			c,
			http.StatusGone,
			"PASSWORD_RESET_EXPIRED",
			"Password reset authorization has expired",
		)
	case errors.Is(
		err,
		ErrOTPDisabled,
	):
		writeAuthJSONError(
			c,
			http.StatusServiceUnavailable,
			"PHONE_VERIFICATION_DISABLED",
			"Phone verification is currently unavailable",
		)

	case errors.Is(
		err,
		ErrUnverifiedSignupDisabled,
	):
		writeAuthJSONError(
			c,
			http.StatusServiceUnavailable,
			"REGISTRATION_TEMPORARILY_UNAVAILABLE",
			"Account registration is temporarily unavailable",
		)
	case errors.Is(
		err,
		ErrOTPDeliveryUnavailable,
	):
		writeAuthJSONError(
			c,
			http.StatusServiceUnavailable,
			"OTP_DELIVERY_UNAVAILABLE",
			"SMS verification is temporarily unavailable",
		)

	case errors.Is(
		err,
		ErrInvalidCredentials,
	):
		writeAuthJSONError(
			c,
			http.StatusUnauthorized,
			"INVALID_CREDENTIALS",
			"Invalid phone number or password",
		)

	case errors.Is(
		err,
		ErrLoginBlocked,
	):
		writeAuthJSONError(
			c,
			http.StatusTooManyRequests,
			"LOGIN_TEMPORARILY_BLOCKED",
			"Too many failed sign-in attempts. Please try again later.",
		)

	case errors.Is(
		err,
		ErrAuthUnavailable,
	):
		writeAuthJSONError(
			c,
			http.StatusServiceUnavailable,
			"AUTH_UNAVAILABLE",
			"Customer authentication is temporarily unavailable",
		)

	case errors.Is(
		err,
		ErrCustomerDisabled,
	):
		writeAuthJSONError(
			c,
			http.StatusForbidden,
			"CUSTOMER_DISABLED",
			"Customer account is disabled",
		)

	case errors.Is(
		err,
		ErrInvalidAccessToken,
	):
		writeAuthJSONError(
			c,
			http.StatusUnauthorized,
			"INVALID_ACCESS_TOKEN",
			"Invalid or expired access token",
		)

	case errors.Is(
		err,
		ErrInvalidRefreshToken,
	):
		writeAuthJSONError(
			c,
			http.StatusUnauthorized,
			"INVALID_REFRESH_TOKEN",
			"Invalid or expired refresh token",
		)

	default:
		writeAuthJSONError(
			c,
			http.StatusInternalServerError,
			"INTERNAL_ERROR",
			"Unable to process authentication request",
		)
	}
}

func writeAuthJSONError(
	c *gin.Context,
	status int,
	code string,
	message string,
) {
	c.JSON(
		status,
		gin.H{
			"error": gin.H{
				"code": code,

				"message": message,
			},
		},
	)
}
