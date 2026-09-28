package auth

import (
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"

	platformsecurity "project.local/commerce-api/internal/platform/security"
)

func TestCurrentArgon2PasswordVerifiesWithoutUpgrade(
	t *testing.T,
) {
	const password = "CustomerPassword123"

	hash, err :=
		hashPassword(
			password,
		)
	if err != nil {
		t.Fatalf(
			"hash password: %v",
			err,
		)
	}

	valid,
		replacement,
		err :=
		verifyPassword(
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
			"expected password to verify",
		)
	}

	if replacement != "" {
		t.Fatal(
			"current Argon2id hash should not require upgrade",
		)
	}
}

func TestWrongArgon2PasswordRejected(
	t *testing.T,
) {
	hash, err :=
		hashPassword(
			"CustomerPassword123",
		)
	if err != nil {
		t.Fatalf(
			"hash password: %v",
			err,
		)
	}

	valid,
		_,
		err :=
		verifyPassword(
			"wrong-password",
			hash,
		)
	if err != nil {
		t.Fatalf(
			"verify password: %v",
			err,
		)
	}

	if valid {
		t.Fatal(
			"wrong password unexpectedly verified",
		)
	}
}

func TestLegacyBcryptPasswordUpgradesToArgon2id(
	t *testing.T,
) {
	const password = "CustomerPassword123"

	legacyHash, err :=
		bcrypt.GenerateFromPassword(
			[]byte(
				password,
			),
			bcrypt.DefaultCost,
		)
	if err != nil {
		t.Fatalf(
			"generate bcrypt password: %v",
			err,
		)
	}

	valid,
		replacement,
		err :=
		verifyPassword(
			password,
			string(
				legacyHash,
			),
		)
	if err != nil {
		t.Fatalf(
			"verify legacy password: %v",
			err,
		)
	}

	if !valid {
		t.Fatal(
			"expected legacy bcrypt password to verify",
		)
	}

	if !strings.HasPrefix(
		replacement,
		"$argon2id$",
	) {
		t.Fatalf(
			"expected Argon2id replacement, got %q",
			replacement,
		)
	}

	replacementValid, err :=
		platformsecurity.VerifyPassword(
			password,
			replacement,
		)
	if err != nil {
		t.Fatalf(
			"verify replacement: %v",
			err,
		)
	}

	if !replacementValid {
		t.Fatal(
			"replacement Argon2id hash did not verify",
		)
	}
}

func TestWrongLegacyBcryptPasswordRejected(
	t *testing.T,
) {
	legacyHash, err :=
		bcrypt.GenerateFromPassword(
			[]byte(
				"CustomerPassword123",
			),
			bcrypt.DefaultCost,
		)
	if err != nil {
		t.Fatalf(
			"generate bcrypt password: %v",
			err,
		)
	}

	valid,
		replacement,
		err :=
		verifyPassword(
			"wrong-password",
			string(
				legacyHash,
			),
		)
	if err != nil {
		t.Fatalf(
			"verify legacy password: %v",
			err,
		)
	}

	if valid {
		t.Fatal(
			"wrong bcrypt password unexpectedly verified",
		)
	}

	if replacement != "" {
		t.Fatal(
			"wrong password must not generate replacement hash",
		)
	}
}

func TestPasswordWhitespaceIsPreserved(
	t *testing.T,
) {
	const password = "  Customer Password 123  "

	hash, err :=
		hashPassword(
			password,
		)
	if err != nil {
		t.Fatalf(
			"hash password: %v",
			err,
		)
	}

	valid,
		_,
		err :=
		verifyPassword(
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
			"exact password should verify",
		)
	}

	valid,
		_,
		err =
		verifyPassword(
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
