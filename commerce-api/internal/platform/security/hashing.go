package security

import (
	"crypto/sha256"
	"encoding/hex"
)

func SHA256Hex(
	value []byte,
) string {
	digest :=
		sha256.Sum256(
			value,
		)

	return hex.EncodeToString(
		digest[:],
	)
}

func SHA256String(
	value string,
) string {
	return SHA256Hex(
		[]byte(
			value,
		),
	)
}

func HashToken(
	token string,
) string {
	return SHA256String(
		token,
	)
}
