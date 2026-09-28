package payment

import (
	"crypto/sha256"
	"encoding/hex"
)

func PayloadSHA256(
	payload []byte,
) string {
	sum :=
		sha256.Sum256(
			payload,
		)

	return hex.EncodeToString(
		sum[:],
	)
}
