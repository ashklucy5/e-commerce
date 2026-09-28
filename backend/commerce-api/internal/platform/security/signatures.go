package security

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"strings"
)

func HMACSHA256Hex(
	secret []byte,
	payload []byte,
) string {
	mac :=
		hmac.New(
			sha256.New,
			secret,
		)

	_, _ =
		mac.Write(
			payload,
		)

	return hex.EncodeToString(
		mac.Sum(
			nil,
		),
	)
}

func VerifyHMACSHA256Hex(
	secret []byte,
	payload []byte,
	expectedSignature string,
) bool {
	expectedSignature =
		strings.TrimSpace(
			expectedSignature,
		)

	if expectedSignature == "" {
		return false
	}

	expected, err :=
		hex.DecodeString(
			expectedSignature,
		)
	if err != nil ||
		len(expected) !=
			sha256.Size {
		return false
	}

	mac :=
		hmac.New(
			sha256.New,
			secret,
		)

	_, _ =
		mac.Write(
			payload,
		)

	actual :=
		mac.Sum(
			nil,
		)

	return subtle.ConstantTimeCompare(
		actual,
		expected,
	) == 1
}
