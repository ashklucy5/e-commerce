package staff

import (
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"

	platformsecurity "project.local/commerce-api/internal/platform/security"
)

// verifyPassword verifies either the current Argon2id format or the
// legacy bcrypt format.
//
// replacementHash is returned only after successful verification when
// the stored password should be upgraded.
//
// That allows legacy password migration without ever needing the
// plaintext password outside the successful login request.
func verifyPassword(
	password string,
	encodedHash string,
) (
	valid bool,
	replacementHash string,
	err error,
) {
	encodedHash =
		strings.TrimSpace(
			encodedHash,
		)

	if encodedHash == "" {
		return false,
			"",
			fmt.Errorf(
				"staff password hash is empty",
			)
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
				fmt.Errorf(
					"verify staff Argon2id password: %w",
					err,
				)
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
				fmt.Errorf(
					"inspect staff Argon2id password parameters: %w",
					err,
				)
		}

		if !needsUpgrade {
			return true,
				"",
				nil
		}

		replacementHash, err :=
			platformsecurity.HashPassword(
				password,
			)
		if err != nil {
			return false,
				"",
				fmt.Errorf(
					"upgrade staff Argon2id password: %w",
					err,
				)
		}

		return true,
			replacementHash,
			nil
	}

	err =
		bcrypt.CompareHashAndPassword(
			[]byte(
				encodedHash,
			),
			[]byte(
				password,
			),
		)

	if errors.Is(
		err,
		bcrypt.ErrMismatchedHashAndPassword,
	) {
		return false,
			"",
			nil
	}

	if err != nil {
		return false,
			"",
			fmt.Errorf(
				"verify legacy staff bcrypt password: %w",
				err,
			)
	}

	replacementHash, err =
		platformsecurity.HashPassword(
			password,
		)
	if err != nil {
		return false,
			"",
			fmt.Errorf(
				"migrate legacy staff password to Argon2id: %w",
				err,
			)
	}

	return true,
		replacementHash,
		nil
}
