package auth

import (
	"errors"
	"strings"

	"golang.org/x/crypto/bcrypt"

	platformsecurity "project.local/commerce-api/internal/platform/security"
)

func hashPassword(
	password string,
) (string, error) {
	return platformsecurity.HashPassword(
		password,
	)
}

func verifyPassword(
	password string,
	encodedHash string,
) (
	bool,
	string,
	error,
) {
	if password == "" ||
		len([]byte(password)) >
			platformsecurity.MaxPasswordBytes {

		return false, "", nil
	}

	encodedHash =
		strings.TrimSpace(
			encodedHash,
		)

	if encodedHash == "" {
		return false, "", nil
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
			return false, "", err
		}

		if !valid {
			return false, "", nil
		}

		needsUpgrade, err :=
			platformsecurity.PasswordHashNeedsUpgrade(
				encodedHash,
			)
		if err != nil {
			return false, "", err
		}

		if !needsUpgrade {
			return true, "", nil
		}

		replacement, err :=
			platformsecurity.HashPassword(
				password,
			)
		if err != nil {
			return false, "", err
		}

		return true,
			replacement,
			nil
	}

	err :=
		bcrypt.CompareHashAndPassword(
			[]byte(
				encodedHash,
			),
			[]byte(
				password,
			),
		)

	if err != nil {
		if errors.Is(
			err,
			bcrypt.ErrMismatchedHashAndPassword,
		) {
			return false, "", nil
		}

		return false, "", err
	}

	replacement, err :=
		platformsecurity.HashPassword(
			password,
		)
	if err != nil {
		return false, "", err
	}

	return true,
		replacement,
		nil
}
