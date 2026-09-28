package adminauth

import "errors"

var (
	ErrInvalidRequest     = errors.New("invalid Admin authentication request")
	ErrInvalidCredentials = errors.New("invalid Admin credentials")
	ErrLoginBlocked       = errors.New("Admin login is temporarily blocked")
	ErrMFABlocked         = errors.New("Admin MFA verification is temporarily blocked")
	ErrAuthUnavailable    = errors.New("Admin authentication is temporarily unavailable")

	ErrInvalidChallenge  = errors.New("invalid Admin login challenge")
	ErrChallengeExpired  = errors.New("Admin login challenge expired")
	ErrChallengeLocked   = errors.New("Admin login challenge is locked")
	ErrChallengeConsumed = errors.New("Admin login challenge is already consumed")

	ErrInvalidAccessToken  = errors.New("invalid or expired Admin access token")
	ErrInvalidRefreshToken = errors.New("invalid or expired Admin refresh token")
	ErrInvalidCSRFToken    = errors.New("invalid Admin CSRF token")
	ErrSessionRevoked      = errors.New("Admin session is revoked")
)
