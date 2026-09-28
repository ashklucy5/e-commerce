package config

import "testing"

func TestLoadAdminSecurityConfig(
	t *testing.T,
) {
	t.Setenv(
		"ADMIN_ENCRYPTION_KEY_ID",
		"admin-test-v1",
	)

	t.Setenv(
		"ADMIN_ENCRYPTION_KEY_BASE64",
		"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
	)

	t.Setenv(
		"ADMIN_TOTP_ISSUER",
		"Test Commerce Admin",
	)

	t.Setenv(
		"ADMIN_ALLOWED_ORIGINS",
		"http://localhost:3000,http://127.0.0.1:3000",
	)

	t.Setenv(
		"ADMIN_ALLOWED_NETWORKS",
		"127.0.0.1/32,::1/128",
	)

	t.Setenv(
		"ADMIN_COOKIE_DOMAIN",
		"",
	)

	cfg, err :=
		LoadAdminSecurityConfig(
			"development",
		)
	if err != nil {
		t.Fatalf(
			"load Admin security config: %v",
			err,
		)
	}

	if cfg.EncryptionKeyID !=
		"admin-test-v1" {

		t.Fatalf(
			"unexpected key ID %q",
			cfg.EncryptionKeyID,
		)
	}

	if cfg.TOTPIssuer !=
		"Test Commerce Admin" {

		t.Fatalf(
			"unexpected issuer %q",
			cfg.TOTPIssuer,
		)
	}

	if len(
		cfg.AllowedOrigins,
	) != 2 {

		t.Fatalf(
			"expected 2 allowed origins, got %d",
			len(
				cfg.AllowedOrigins,
			),
		)
	}

	if len(
		cfg.AllowedNetworks,
	) != 2 {

		t.Fatalf(
			"expected 2 allowed networks, got %d",
			len(
				cfg.AllowedNetworks,
			),
		)
	}
}

func TestLoadAdminSecurityConfigRequiresProductionOrigins(
	t *testing.T,
) {
	t.Setenv(
		"ADMIN_ENCRYPTION_KEY_ID",
		"admin-test-v1",
	)

	t.Setenv(
		"ADMIN_ENCRYPTION_KEY_BASE64",
		"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
	)

	t.Setenv(
		"ADMIN_ALLOWED_ORIGINS",
		"",
	)

	if _, err :=
		LoadAdminSecurityConfig(
			"production",
		); err == nil {

		t.Fatal(
			"expected production Admin origins to be required",
		)
	}
}
