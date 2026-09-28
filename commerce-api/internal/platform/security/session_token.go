package security

import (
	"errors"
	"fmt"
	"time"
)

const (
	DefaultAccessTokenBytes = 32

	DefaultRefreshTokenBytes = 48

	DefaultAccessTokenTTL = 15 * time.Minute

	DefaultRefreshTokenTTL = 30 * 24 * time.Hour
)

var ErrInvalidSessionTokenConfig = errors.New(
	"invalid session token configuration",
)

type SessionTokenConfig struct {
	AccessTokenBytes int

	RefreshTokenBytes int

	AccessTokenTTL time.Duration

	RefreshTokenTTL time.Duration
}

type SessionTokenMaterial struct {
	AccessToken string

	RefreshToken string

	AccessTokenHash string

	RefreshTokenHash string

	AccessExpiresAt time.Time

	RefreshExpiresAt time.Time
}

func DefaultSessionTokenConfig() SessionTokenConfig {
	return SessionTokenConfig{
		AccessTokenBytes: DefaultAccessTokenBytes,

		RefreshTokenBytes: DefaultRefreshTokenBytes,

		AccessTokenTTL: DefaultAccessTokenTTL,

		RefreshTokenTTL: DefaultRefreshTokenTTL,
	}
}

func NewSessionTokenMaterial(
	now time.Time,
	cfg SessionTokenConfig,
) (SessionTokenMaterial, error) {
	if err :=
		validateSessionTokenConfig(
			cfg,
		); err != nil {
		return SessionTokenMaterial{},
			err
	}

	if now.IsZero() {
		now =
			time.Now().
				UTC()
	} else {
		now =
			now.UTC()
	}

	accessToken, err :=
		RandomURLSafe(
			cfg.AccessTokenBytes,
		)
	if err != nil {
		return SessionTokenMaterial{},
			fmt.Errorf(
				"generate access token: %w",
				err,
			)
	}

	refreshToken, err :=
		RandomURLSafe(
			cfg.RefreshTokenBytes,
		)
	if err != nil {
		return SessionTokenMaterial{},
			fmt.Errorf(
				"generate refresh token: %w",
				err,
			)
	}

	return SessionTokenMaterial{
			AccessToken: accessToken,

			RefreshToken: refreshToken,

			AccessTokenHash: HashToken(
				accessToken,
			),

			RefreshTokenHash: HashToken(
				refreshToken,
			),

			AccessExpiresAt: now.Add(
				cfg.AccessTokenTTL,
			),

			RefreshExpiresAt: now.Add(
				cfg.RefreshTokenTTL,
			),
		},
		nil
}

func validateSessionTokenConfig(
	cfg SessionTokenConfig,
) error {
	if cfg.AccessTokenBytes < 16 ||
		cfg.AccessTokenBytes >
			maxRandomBytes {
		return fmt.Errorf(
			"%w: access token bytes must be between 16 and %d",
			ErrInvalidSessionTokenConfig,
			maxRandomBytes,
		)
	}

	if cfg.RefreshTokenBytes < 16 ||
		cfg.RefreshTokenBytes >
			maxRandomBytes {
		return fmt.Errorf(
			"%w: refresh token bytes must be between 16 and %d",
			ErrInvalidSessionTokenConfig,
			maxRandomBytes,
		)
	}

	if cfg.AccessTokenTTL <= 0 {
		return fmt.Errorf(
			"%w: access token TTL must be greater than zero",
			ErrInvalidSessionTokenConfig,
		)
	}

	if cfg.RefreshTokenTTL <= 0 {
		return fmt.Errorf(
			"%w: refresh token TTL must be greater than zero",
			ErrInvalidSessionTokenConfig,
		)
	}

	if cfg.RefreshTokenTTL <=
		cfg.AccessTokenTTL {
		return fmt.Errorf(
			"%w: refresh token TTL must be greater than access token TTL",
			ErrInvalidSessionTokenConfig,
		)
	}

	return nil
}
