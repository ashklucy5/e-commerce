package auth

import (
	"net/mail"
	"strings"
	"unicode"
)

func normalizePhone(
	value string,
) (string, error) {
	value = strings.TrimSpace(
		value,
	)

	if strings.HasPrefix(
		value,
		"00",
	) {
		value =
			"+" +
				value[2:]
	}

	var builder strings.Builder

	for index, char := range value {
		switch {
		case unicode.IsSpace(
			char,
		):
			continue

		case char == '-' ||
			char == '(' ||
			char == ')':
			continue

		case char == '+' &&
			index == 0:
			builder.WriteRune(
				char,
			)

		case char >= '0' &&
			char <= '9':
			builder.WriteRune(
				char,
			)

		default:
			return "",
				ErrInvalidPhone
		}
	}

	normalized :=
		builder.String()

	digits :=
		strings.TrimPrefix(
			normalized,
			"+",
		)

	if len(digits) < 7 ||
		len(digits) > 15 {
		return "",
			ErrInvalidPhone
	}

	return normalized, nil
}

func normalizeEmail(
	value string,
) (string, error) {
	value =
		strings.ToLower(
			strings.TrimSpace(
				value,
			),
		)

	if value == "" {
		return "", nil
	}

	if len(value) > 255 {
		return "",
			ErrInvalidEmail
	}

	address, err :=
		mail.ParseAddress(
			value,
		)
	if err != nil ||
		address.Address != value {
		return "",
			ErrInvalidEmail
	}

	return value, nil
}

func normalizeFullName(
	value string,
) (string, error) {
	value =
		strings.Join(
			strings.Fields(
				value,
			),
			" ",
		)

	if len(value) < 2 ||
		len(value) > 160 {
		return "",
			ErrInvalidFullName
	}

	return value, nil
}

func validatePassword(
	password string,
) error {
	length := len(
		[]byte(password),
	)

	if length < 8 ||
		length > 72 {
		return ErrInvalidPassword
	}

	return nil
}
