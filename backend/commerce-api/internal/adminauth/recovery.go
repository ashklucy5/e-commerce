package adminauth

import (
	"encoding/base32"
	"errors"
	"fmt"
	"strings"

	platformsecurity "project.local/commerce-api/internal/platform/security"
)

const (
	DefaultRecoveryCodeCount = 10

	recoveryCodeRandomBytes = 12

	recoveryCodeCharacters = 20
)

var ErrInvalidRecoveryCode = errors.New(
	"invalid recovery code",
)

func GenerateRecoveryCodes(
	count int,
) ([]string, error) {
	if count < 1 ||
		count > 50 {

		return nil,
			fmt.Errorf(
				"%w: count must be between 1 and 50",
				ErrInvalidRecoveryCode,
			)
	}

	result :=
		make(
			[]string,
			0,
			count,
		)

	seen :=
		make(
			map[string]struct{},
			count,
		)

	for len(result) < count {
		random, err :=
			platformsecurity.RandomBytes(
				recoveryCodeRandomBytes,
			)
		if err != nil {
			return nil,
				fmt.Errorf(
					"generate recovery code: %w",
					err,
				)
		}

		encoded :=
			base32.StdEncoding.
				WithPadding(
					base32.NoPadding,
				).
				EncodeToString(
					random,
				)

		if len(encoded) !=
			recoveryCodeCharacters {

			return nil,
				ErrInvalidRecoveryCode
		}

		if _, exists :=
			seen[encoded]; exists {

			continue
		}

		seen[encoded] = struct{}{}

		result =
			append(
				result,
				formatRecoveryCode(
					encoded,
				),
			)
	}

	return result,
		nil
}

func HashRecoveryCode(
	code string,
) (
	string,
	error,
) {
	normalized, err :=
		normalizeRecoveryCode(
			code,
		)
	if err != nil {
		return "",
			err
	}

	return platformsecurity.HashToken(
			normalized,
		),
		nil
}

func normalizeRecoveryCode(
	code string,
) (
	string,
	error,
) {
	value :=
		strings.ToUpper(
			strings.TrimSpace(
				code,
			),
		)

	value =
		strings.ReplaceAll(
			value,
			"-",
			"",
		)

	value =
		strings.ReplaceAll(
			value,
			" ",
			"",
		)

	if len(value) !=
		recoveryCodeCharacters {

		return "",
			ErrInvalidRecoveryCode
	}

	for index :=
		0; index < len(value); index++ {

		character :=
			value[index]

		valid :=
			(character >= 'A' &&
				character <= 'Z') ||
				(character >= '2' &&
					character <= '7')

		if !valid {
			return "",
				ErrInvalidRecoveryCode
		}
	}

	return value,
		nil
}

func formatRecoveryCode(
	value string,
) string {
	parts :=
		make(
			[]string,
			0,
			5,
		)

	for start :=
		0; start < len(value); start += 4 {

		parts =
			append(
				parts,
				value[start:start+4],
			)
	}

	return strings.Join(
		parts,
		"-",
	)
}
