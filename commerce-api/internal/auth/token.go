package auth

import (
	"time"

	platformsecurity "project.local/commerce-api/internal/platform/security"
)

type tokenMaterial = platformsecurity.SessionTokenMaterial

func newTokenMaterial(
	now time.Time,
) (
	tokenMaterial,
	error,
) {
	return platformsecurity.NewSessionTokenMaterial(
		now,
		platformsecurity.DefaultSessionTokenConfig(),
	)
}

func hashToken(
	token string,
) string {
	return platformsecurity.HashToken(
		token,
	)
}

func publicTokens(
	material tokenMaterial,
) Tokens {
	return Tokens{
		AccessToken: material.AccessToken,

		RefreshToken: material.RefreshToken,

		TokenType: "Bearer",

		AccessExpiresAt: material.AccessExpiresAt,

		RefreshExpiresAt: material.RefreshExpiresAt,
	}
}
