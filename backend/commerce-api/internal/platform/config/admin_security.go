package config

import (
	"fmt"
	"os"
	"strings"
)

const DefaultAdminTOTPIssuer = "Commerce Admin"

type AdminSecurityConfig struct {
	EncryptionKeyID     string
	EncryptionKeyBase64 string
	TOTPIssuer          string

	AllowedOrigins  []string
	AllowedNetworks []string
	CookieDomain    string
}

func LoadAdminSecurityConfig(
	appEnv string,
) (
	AdminSecurityConfig,
	error,
) {
	keyID, err :=
		requireEnv(
			"ADMIN_ENCRYPTION_KEY_ID",
		)
	if err != nil {
		return AdminSecurityConfig{},
			err
	}

	keyBase64, err :=
		requireEnv(
			"ADMIN_ENCRYPTION_KEY_BASE64",
		)
	if err != nil {
		return AdminSecurityConfig{},
			err
	}

	issuer :=
		strings.TrimSpace(
			getEnv(
				"ADMIN_TOTP_ISSUER",
				DefaultAdminTOTPIssuer,
			),
		)

	if issuer == "" {
		return AdminSecurityConfig{},
			fmt.Errorf(
				"ADMIN_TOTP_ISSUER cannot be blank",
			)
	}

	allowedOrigins :=
		splitCSV(
			os.Getenv(
				"ADMIN_ALLOWED_ORIGINS",
			),
		)

	isProduction :=
		strings.EqualFold(
			strings.TrimSpace(
				appEnv,
			),
			"production",
		)

	if len(allowedOrigins) == 0 {
		if isProduction {
			return AdminSecurityConfig{},
				fmt.Errorf(
					"ADMIN_ALLOWED_ORIGINS is required in production",
				)
		}

		allowedOrigins =
			[]string{
				"http://localhost:3000",
				"http://127.0.0.1:3000",
			}
	}

	for _, origin := range allowedOrigins {

		if origin == "*" {
			return AdminSecurityConfig{},
				fmt.Errorf(
					"ADMIN_ALLOWED_ORIGINS cannot contain wildcard origin when Admin cookies are enabled",
				)
		}
	}

	return AdminSecurityConfig{
			EncryptionKeyID: strings.TrimSpace(
				keyID,
			),

			EncryptionKeyBase64: strings.TrimSpace(
				keyBase64,
			),

			TOTPIssuer: issuer,

			AllowedOrigins: allowedOrigins,

			AllowedNetworks: splitCSV(
				os.Getenv(
					"ADMIN_ALLOWED_NETWORKS",
				),
			),

			CookieDomain: strings.TrimSpace(
				os.Getenv(
					"ADMIN_COOKIE_DOMAIN",
				),
			),
		},
		nil
}

func splitCSV(
	value string,
) []string {
	parts :=
		strings.Split(
			value,
			",",
		)

	result :=
		make(
			[]string,
			0,
			len(parts),
		)

	seen :=
		make(
			map[string]struct{},
			len(parts),
		)

	for _, part := range parts {

		item :=
			strings.TrimSpace(
				part,
			)

		if item == "" {
			continue
		}

		if _, exists :=
			seen[item]; exists {

			continue
		}

		seen[item] =
			struct{}{}

		result =
			append(
				result,
				item,
			)
	}

	return result
}
