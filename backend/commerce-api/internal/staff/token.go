package staff

import (
	"fmt"
	"time"

	platformsecurity "project.local/commerce-api/internal/platform/security"
)

type tokenMaterial = platformsecurity.SessionTokenMaterial

func newTokenMaterial(
	now time.Time,
) (tokenMaterial, error) {
	material, err :=
		platformsecurity.NewSessionTokenMaterial(
			now,
			platformsecurity.
				DefaultSessionTokenConfig(),
		)
	if err != nil {
		return tokenMaterial{},
			fmt.Errorf(
				"generate staff session tokens: %w",
				err,
			)
	}

	return material, nil
}

func hashToken(
	token string,
) string {
	return platformsecurity.HashToken(
		token,
	)
}

func tokensFromMaterial(
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
