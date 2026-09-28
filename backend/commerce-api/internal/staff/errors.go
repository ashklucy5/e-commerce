package staff

import "errors"

var (
	ErrInvalidRequest = errors.New(
		"invalid staff authentication request",
	)

	ErrInvalidCredentials = errors.New(
		"invalid staff credentials",
	)

	ErrStaffDisabled = errors.New(
		"staff account is not active",
	)

	ErrInvalidAccessToken = errors.New(
		"invalid or expired staff access token",
	)

	ErrInvalidRefreshToken = errors.New(
		"invalid or expired staff refresh token",
	)

	ErrForbidden = errors.New(
		"staff permission denied",
	)
)
