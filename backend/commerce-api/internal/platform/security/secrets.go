package security

import (
	"crypto/sha256"
	"crypto/subtle"
)

// ConstantTimeEqual compares secret values through fixed-size SHA-256
// digests so the comparison primitive does not operate on the original
// secret lengths.
func ConstantTimeEqual(
	left string,
	right string,
) bool {
	leftDigest :=
		sha256.Sum256(
			[]byte(
				left,
			),
		)

	rightDigest :=
		sha256.Sum256(
			[]byte(
				right,
			),
		)

	return subtle.ConstantTimeCompare(
		leftDigest[:],
		rightDigest[:],
	) == 1
}
