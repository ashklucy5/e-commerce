package staff

import (
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"

	platformsecurity "project.local/commerce-api/internal/platform/security"
)

func TestVerifyPasswordAcceptsCurrentArgon2id(
	t *testing.T,
) {
	const password = "correct horse battery staple"

	hash, err :=
		platformsecurity.HashPassword(
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
			"verify Argon2id password: %v",
			err,
		)
	}

	if !valid {
		t.Fatal(
			"expected Argon2id password to verify",
		)
	}

	if replacement != "" {
		t.Fatal(
			"current Argon2id password unexpectedly requested upgrade",
		)
	}
}

func TestVerifyPasswordRejectsWrongArgon2idPassword(
	t *testing.T,
) {
	hash, err :=
		platformsecurity.HashPassword(
			"correct-password",
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

	if replacement != "" {
		t.Fatal(
			"wrong password unexpectedly produced replacement hash",
		)
	}
}

func TestVerifyPasswordMigratesLegacyBcrypt(
	t *testing.T,
) {
	const password = "legacy support password"

	legacyHash, err :=
		bcrypt.GenerateFromPassword(
			[]byte(
				password,
			),
			bcrypt.DefaultCost,
		)
	if err != nil {
		t.Fatalf(
			"generate bcrypt hash: %v",
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
			"verify bcrypt password: %v",
			err,
		)
	}

	if !valid {
		t.Fatal(
			"expected bcrypt password to verify",
		)
	}

	if replacement == "" {
		t.Fatal(
			"expected bcrypt password to produce Argon2id replacement",
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
			"verify replacement Argon2id hash: %v",
			err,
		)
	}

	if !replacementValid {
		t.Fatal(
			"replacement Argon2id hash does not verify",
		)
	}
}

func TestVerifyPasswordRejectsWrongLegacyBcryptPassword(
	t *testing.T,
) {
	legacyHash, err :=
		bcrypt.GenerateFromPassword(
			[]byte(
				"correct-password",
			),
			bcrypt.DefaultCost,
		)
	if err != nil {
		t.Fatalf(
			"generate bcrypt hash: %v",
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
			"verify bcrypt password: %v",
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
			"wrong bcrypt password unexpectedly produced replacement hash",
		)
	}
}

func TestVerifyPasswordPreservesWhitespace(
	t *testing.T,
) {
	const password = "  password with spaces  "

	hash, err :=
		platformsecurity.HashPassword(
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
			"verify password: %v",
			err,
		)
	}

	if !valid {
		t.Fatal(
			"expected exact password including whitespace to verify",
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
			"trimmed password must not verify against whitespace-preserving hash",
		)
	}
}
