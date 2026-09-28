package security

import (
	"encoding/base64"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestRandomURLSafe(
	t *testing.T,
) {
	first, err :=
		RandomURLSafe(
			32,
		)
	if err != nil {
		t.Fatalf(
			"generate first token: %v",
			err,
		)
	}

	second, err :=
		RandomURLSafe(
			32,
		)
	if err != nil {
		t.Fatalf(
			"generate second token: %v",
			err,
		)
	}

	if first == second {
		t.Fatal(
			"expected independently generated random tokens",
		)
	}

	decoded, err :=
		base64.RawURLEncoding.
			DecodeString(
				first,
			)
	if err != nil {
		t.Fatalf(
			"decode URL-safe token: %v",
			err,
		)
	}

	if len(decoded) != 32 {
		t.Fatalf(
			"expected 32 random bytes, got %d",
			len(decoded),
		)
	}
}

func TestRandomBytesRejectsInvalidLength(
	t *testing.T,
) {
	_, err :=
		RandomBytes(
			0,
		)

	if !errors.Is(
		err,
		ErrInvalidRandomLength,
	) {
		t.Fatalf(
			"expected ErrInvalidRandomLength, got %v",
			err,
		)
	}
}

func TestHashAndSecretComparison(
	t *testing.T,
) {
	const token = "token-value"

	if HashToken(
		token,
	) != SHA256String(
		token,
	) {
		t.Fatal(
			"expected token hash to use SHA-256",
		)
	}

	if !ConstantTimeEqual(
		"secret",
		"secret",
	) {
		t.Fatal(
			"expected matching secrets",
		)
	}

	if ConstantTimeEqual(
		"secret",
		"different",
	) {
		t.Fatal(
			"expected different secrets not to match",
		)
	}
}

func TestSessionTokenMaterial(
	t *testing.T,
) {
	now :=
		time.Date(
			2026,
			8,
			20,
			12,
			0,
			0,
			0,
			time.UTC,
		)

	cfg :=
		DefaultSessionTokenConfig()

	material, err :=
		NewSessionTokenMaterial(
			now,
			cfg,
		)
	if err != nil {
		t.Fatalf(
			"create session token material: %v",
			err,
		)
	}

	if material.AccessToken == "" ||
		material.RefreshToken == "" {
		t.Fatal(
			"expected public token material",
		)
	}

	if material.AccessToken ==
		material.RefreshToken {
		t.Fatal(
			"access and refresh tokens must differ",
		)
	}

	if material.AccessTokenHash !=
		HashToken(
			material.AccessToken,
		) {
		t.Fatal(
			"unexpected access token hash",
		)
	}

	if material.RefreshTokenHash !=
		HashToken(
			material.RefreshToken,
		) {
		t.Fatal(
			"unexpected refresh token hash",
		)
	}

	if !material.AccessExpiresAt.
		Equal(
			now.Add(
				cfg.AccessTokenTTL,
			),
		) {
		t.Fatal(
			"unexpected access token expiry",
		)
	}

	if !material.RefreshExpiresAt.
		Equal(
			now.Add(
				cfg.RefreshTokenTTL,
			),
		) {
		t.Fatal(
			"unexpected refresh token expiry",
		)
	}
}

func TestHMACSHA256Hex(
	t *testing.T,
) {
	secret :=
		[]byte(
			"test-secret",
		)

	payload :=
		[]byte(
			"payload",
		)

	signature :=
		HMACSHA256Hex(
			secret,
			payload,
		)

	if !VerifyHMACSHA256Hex(
		secret,
		payload,
		signature,
	) {
		t.Fatal(
			"expected valid signature",
		)
	}

	if VerifyHMACSHA256Hex(
		secret,
		[]byte(
			"changed",
		),
		signature,
	) {
		t.Fatal(
			"expected changed payload to fail verification",
		)
	}
}

func TestArgon2PasswordHash(
	t *testing.T,
) {
	const password = "correct horse battery staple"

	hash, err :=
		HashPassword(
			password,
		)
	if err != nil {
		t.Fatalf(
			"hash password: %v",
			err,
		)
	}

	if hash == password {
		t.Fatal(
			"password hash must not equal plaintext",
		)
	}

	if !strings.HasPrefix(
		hash,
		"$argon2id$",
	) {
		t.Fatalf(
			"expected argon2id PHC hash, got %q",
			hash,
		)
	}

	valid, err :=
		VerifyPassword(
			password,
			hash,
		)
	if err != nil {
		t.Fatalf(
			"verify password: %v",
			err,
		)
	}

	if !valid {
		t.Fatal(
			"expected correct password to verify",
		)
	}

	valid, err =
		VerifyPassword(
			"wrong password",
			hash,
		)
	if err != nil {
		t.Fatalf(
			"verify wrong password: %v",
			err,
		)
	}

	if valid {
		t.Fatal(
			"wrong password unexpectedly verified",
		)
	}
}

func TestArgon2UsesUniqueSalt(
	t *testing.T,
) {
	const password = "same-password"

	first, err :=
		HashPassword(
			password,
		)
	if err != nil {
		t.Fatalf(
			"first hash: %v",
			err,
		)
	}

	second, err :=
		HashPassword(
			password,
		)
	if err != nil {
		t.Fatalf(
			"second hash: %v",
			err,
		)
	}

	if first == second {
		t.Fatal(
			"expected different hashes from unique salts",
		)
	}
}

func TestArgon2NeedsUpgrade(
	t *testing.T,
) {
	params :=
		DefaultArgon2Params()

	params.Memory =
		32 * 1024

	hash, err :=
		HashPasswordWithParams(
			"upgrade-test-password",
			params,
		)
	if err != nil {
		t.Fatalf(
			"hash password: %v",
			err,
		)
	}

	needsUpgrade, err :=
		PasswordHashNeedsUpgrade(
			hash,
		)
	if err != nil {
		t.Fatalf(
			"check upgrade: %v",
			err,
		)
	}

	if !needsUpgrade {
		t.Fatal(
			"expected weaker password hash to need upgrade",
		)
	}
}

func TestArgon2CurrentParametersDoNotNeedUpgrade(
	t *testing.T,
) {
	hash, err :=
		HashPassword(
			"current-parameter-password",
		)
	if err != nil {
		t.Fatalf(
			"hash password: %v",
			err,
		)
	}

	needsUpgrade, err :=
		PasswordHashNeedsUpgrade(
			hash,
		)
	if err != nil {
		t.Fatalf(
			"check upgrade: %v",
			err,
		)
	}

	if needsUpgrade {
		t.Fatal(
			"current hash unexpectedly needs upgrade",
		)
	}
}
