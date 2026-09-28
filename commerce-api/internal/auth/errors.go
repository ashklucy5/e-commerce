package auth

import "errors"

var (
	ErrInvalidRequest = errors.New(
		"invalid request",
	)

	ErrInvalidPhone = errors.New(
		"invalid phone number",
	)

	ErrInvalidEmail = errors.New(
		"invalid email address",
	)

	ErrInvalidPassword = errors.New(
		"password must be between 8 and 72 bytes",
	)

	ErrInvalidFullName = errors.New(
		"invalid full name",
	)

	ErrPhoneInUse = errors.New(
		"phone number is already registered",
	)

	ErrEmailInUse = errors.New(
		"email address is already registered",
	)

	ErrInvalidCredentials = errors.New(
		"invalid phone number or password",
	)

	ErrLoginBlocked = errors.New(
		"customer login is temporarily blocked",
	)

	ErrAuthUnavailable = errors.New(
		"customer authentication is temporarily unavailable",
	)

	ErrCustomerDisabled = errors.New(
		"customer account is disabled",
	)

	ErrInvalidAccessToken = errors.New(
		"invalid or expired access token",
	)

	ErrInvalidRefreshToken = errors.New(
		"invalid or expired refresh token",
	)

	ErrInvalidVerification = errors.New(
		"invalid registration verification",
	)

	ErrVerificationExpired = errors.New(
		"registration verification has expired",
	)

	ErrInvalidVerificationCode = errors.New(
		"invalid verification code",
	)

	ErrVerificationAttemptsExceeded = errors.New(
		"too many verification attempts",
	)

	ErrVerificationResendTooSoon = errors.New(
		"verification code cannot be resent yet",
	)

	ErrVerificationResendLimit = errors.New(
		"verification resend limit reached",
	)

	ErrVerificationBusy = errors.New(
		"registration verification is already being processed",
	)

	ErrOTPDeliveryUnavailable = errors.New(
		"phone verification delivery is temporarily unavailable",
	)

	ErrPasswordConfirmation = errors.New(
		"password confirmation does not match",
	)

	ErrCurrentPasswordIncorrect = errors.New(
		"current password is incorrect",
	)

	ErrPasswordUnchanged = errors.New(
		"new password must be different from the current password",
	)

	ErrInvalidPasswordResetGrant = errors.New(
		"invalid password reset grant",
	)

	ErrPasswordResetGrantExpired = errors.New(
		"password reset grant has expired",
	)

	ErrOTPDisabled = errors.New(
		"phone verification is disabled",
	)

	ErrUnverifiedSignupDisabled = errors.New(
		"unverified signup is disabled",
	)
)
