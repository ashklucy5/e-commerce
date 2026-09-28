package adminauth

import (
	"errors"
	"strings"

	"golang.org/x/crypto/bcrypt"

	platformsecurity "project.local/commerce-api/internal/platform/security"
)

// verifyStoredPassword accepts current Argon2id hashes and legacy bcrypt
// hashes. A non-empty replacement hash means the caller should persist an
// Argon2id upgrade after successful authentication.
func verifyStoredPassword(
	password string,
	encodedHash string,
) (
	bool,
	string,
	error,
) {
	encodedHash =
		strings.TrimSpace(
			encodedHash,
		)

	if password == "" ||
		len(password) >
			platformsecurity.MaxPasswordBytes ||
		encodedHash == "" {

		return false,
			"",
			nil
	}

	if strings.HasPrefix(
		encodedHash,
		"$argon2id$",
	) {
		valid, err :=
			platformsecurity.VerifyPassword(
				password,
				encodedHash,
			)
		if err != nil {
			return false,
				"",
				err
		}

		if !valid {
			return false,
				"",
				nil
		}

		needsUpgrade, err :=
			platformsecurity.PasswordHashNeedsUpgrade(
				encodedHash,
			)
		if err != nil {
			return false,
				"",
				err
		}

		if !needsUpgrade {
			return true,
				"",
				nil
		}

		replacement, err :=
			platformsecurity.HashPassword(
				password,
			)
		if err != nil {
			return false,
				"",
				err
		}

		return true,
			replacement,
			nil
	}

	if err :=
		bcrypt.CompareHashAndPassword(
			[]byte(
				encodedHash,
			),
			[]byte(
				password,
			),
		); err != nil {

		if errors.Is(
			err,
			bcrypt.ErrMismatchedHashAndPassword,
		) {
			return false,
				"",
				nil
		}

		return false,
			"",
			err
	}

	replacement, err :=
		platformsecurity.HashPassword(
			password,
		)
	if err != nil {
		return false,
			"",
			err
	}

	return true,
		replacement,
		nil
}
