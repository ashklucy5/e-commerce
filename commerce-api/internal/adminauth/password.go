package adminauth

import (
	"errors"
	"strings"
	"unicode/utf8"

	platformsecurity "project.local/commerce-api/internal/platform/security"
)

const (
	AdminPasswordMinRunes = 16

	AdminPasswordMaxRunes = 256
)

var (
	ErrPasswordRequired = errors.New(
		"admin password is required",
	)

	ErrPasswordTooShort = errors.New(
		"admin password must be at least 16 characters",
	)

	ErrPasswordTooLong = errors.New(
		"admin password must be at most 256 characters",
	)

	ErrPasswordCommon = errors.New(
		"admin password is too common",
	)
)

var commonPasswords = map[string]struct{}{
	"password": {},

	"password123": {},

	"password123456": {},

	"admin": {},

	"admin123": {},

	"admin123456": {},

	"administrator": {},

	"qwerty": {},

	"qwerty123": {},

	"letmein": {},

	"welcome": {},

	"welcome123": {},

	"test123": {},

	"test123456": {},

	"test123456789": {},
}

func ValidateNewPassword(
	password string,
) error {
	if password == "" {
		return ErrPasswordRequired
	}

	length :=
		utf8.RuneCountInString(
			password,
		)

	if length <
		AdminPasswordMinRunes {
		return ErrPasswordTooShort
	}

	if length >
		AdminPasswordMaxRunes {
		return ErrPasswordTooLong
	}

	normalized :=
		strings.ToLower(
			strings.TrimSpace(
				password,
			),
		)

	if _, exists :=
		commonPasswords[normalized]; exists {

		return ErrPasswordCommon
	}

	return nil
}

func HashNewPassword(
	password string,
) (string, error) {
	if err :=
		ValidateNewPassword(
			password,
		); err != nil {

		return "", err
	}

	return platformsecurity.
		HashPassword(
			password,
		)
}
