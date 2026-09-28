package adminauth

import (
	"fmt"
	"time"

	platformsecurity "project.local/commerce-api/internal/platform/security"
)

const (
	AdminAccessTokenBytes = 32

	AdminRefreshTokenBytes = 48

	AdminAccessTokenTTL = 10 * time.Minute

	AdminRefreshTokenTTL = 8 * time.Hour
)

func newAdminSessionMaterial(
	now time.Time,
	refreshExpiresAt *time.Time,
) (
	AdminSessionMaterial,
	error,
) {
	now =
		now.UTC()

	accessToken, err :=
		platformsecurity.RandomURLSafe(
			AdminAccessTokenBytes,
		)
	if err != nil {
		return AdminSessionMaterial{},
			fmt.Errorf(
				"generate Admin access token: %w",
				err,
			)
	}

	refreshToken, err :=
		platformsecurity.RandomURLSafe(
			AdminRefreshTokenBytes,
		)
	if err != nil {
		return AdminSessionMaterial{},
			fmt.Errorf(
				"generate Admin refresh token: %w",
				err,
			)
	}

	csrfToken, err :=
		platformsecurity.NewCSRFToken()
	if err != nil {
		return AdminSessionMaterial{},
			fmt.Errorf(
				"generate Admin CSRF token: %w",
				err,
			)
	}

	csrfHash, err :=
		platformsecurity.HashCSRFToken(
			csrfToken,
		)
	if err != nil {
		return AdminSessionMaterial{},
			fmt.Errorf(
				"hash Admin CSRF token: %w",
				err,
			)
	}

	refreshExpiry :=
		now.Add(
			AdminRefreshTokenTTL,
		)

	if refreshExpiresAt != nil {
		refreshExpiry =
			refreshExpiresAt.UTC()
	}

	if !refreshExpiry.After(
		now,
	) {
		return AdminSessionMaterial{},
			ErrInvalidRefreshToken
	}

	accessExpiry :=
		now.Add(
			AdminAccessTokenTTL,
		)

	if !refreshExpiry.After(
		accessExpiry,
	) {
		accessExpiry =
			refreshExpiry.Add(
				-time.Second,
			)
	}

	if !accessExpiry.After(
		now,
	) {
		return AdminSessionMaterial{},
			ErrInvalidRefreshToken
	}

	return AdminSessionMaterial{
			AccessToken: accessToken,

			RefreshToken: refreshToken,

			CSRFToken: csrfToken,

			AccessTokenHash: platformsecurity.HashToken(
				accessToken,
			),

			RefreshTokenHash: platformsecurity.HashToken(
				refreshToken,
			),

			CSRFTokenHash: csrfHash,

			AccessExpiresAt: accessExpiry,

			RefreshExpiresAt: refreshExpiry,
		},
		nil
}
