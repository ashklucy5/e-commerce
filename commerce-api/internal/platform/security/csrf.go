package security

import (
	"errors"
	"strings"
)

const DefaultCSRFTokenBytes = 32

var ErrInvalidCSRFToken = errors.New(
	"invalid CSRF token",
)

// NewCSRFToken creates a high-entropy browser CSRF secret.
//
// The plaintext value may be sent to the authorized client while only
// its SHA-256 lookup hash needs to be stored server-side.
func NewCSRFToken() (
	string,
	error,
) {
	return RandomURLSafe(
		DefaultCSRFTokenBytes,
	)
}

func HashCSRFToken(
	token string,
) (
	string,
	error,
) {
	token =
		strings.TrimSpace(
			token,
		)

	if token == "" {
		return "",
			ErrInvalidCSRFToken
	}

	return HashToken(
		token,
	), nil
}

func VerifyCSRFToken(
	token string,
	expectedHash string,
) bool {
	token =
		strings.TrimSpace(
			token,
		)

	expectedHash =
		strings.TrimSpace(
			expectedHash,
		)

	if token == "" ||
		expectedHash == "" {
		return false
	}

	actualHash :=
		HashToken(
			token,
		)

	return ConstantTimeEqual(
		actualHash,
		expectedHash,
	)
}
