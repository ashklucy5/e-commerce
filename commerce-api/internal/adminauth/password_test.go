package adminauth

import (
	"errors"
	"strings"
	"testing"

	platformsecurity "project.local/commerce-api/internal/platform/security"
)

func TestValidateNewPassword(
	t *testing.T,
) {
	if err :=
		ValidateNewPassword(
			"correct horse battery staple",
		); err != nil {

		t.Fatalf(
			"expected strong password to pass: %v",
			err,
		)
	}
}

func TestValidateNewPasswordRejectsShortPassword(
	t *testing.T,
) {
	err :=
		ValidateNewPassword(
			"Test123456789",
		)

	if !errors.Is(
		err,
		ErrPasswordTooShort,
	) {
		t.Fatalf(
			"expected ErrPasswordTooShort, got %v",
			err,
		)
	}
}

func TestHashNewPasswordUsesArgon2id(
	t *testing.T,
) {
	const password = "admin development password 2026"

	hash, err :=
		HashNewPassword(
			password,
		)
	if err != nil {
		t.Fatalf(
			"hash password: %v",
			err,
		)
	}

	if !strings.HasPrefix(
		hash,
		"$argon2id$",
	) {
		t.Fatalf(
			"expected Argon2id hash, got %q",
			hash,
		)
	}

	valid, err :=
		platformsecurity.VerifyPassword(
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
			"expected generated hash to verify",
		)
	}
}

func TestPasswordWhitespaceIsPreserved(
	t *testing.T,
) {
	const password = "  admin password with spaces  "

	hash, err :=
		HashNewPassword(
			password,
		)
	if err != nil {
		t.Fatalf(
			"hash password: %v",
			err,
		)
	}

	valid, err :=
		platformsecurity.VerifyPassword(
			password,
			hash,
		)
	if err != nil {
		t.Fatalf(
			"verify exact password: %v",
			err,
		)
	}

	if !valid {
		t.Fatal(
			"expected exact password to verify",
		)
	}

	valid, err =
		platformsecurity.VerifyPassword(
			strings.TrimSpace(
				password,
			),
			hash,
		)
	if err != nil {
		t.Fatalf(
			"verify trimmed password: %v",
			err,
		)
	}

	if valid {
		t.Fatal(
			"trimmed password must not verify",
		)
	}
}
