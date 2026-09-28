package security

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
)

const maxRandomBytes = 4096

var ErrInvalidRandomLength = errors.New(
	"invalid random byte length",
)

func RandomBytes(
	byteLength int,
) ([]byte, error) {
	if byteLength <= 0 ||
		byteLength > maxRandomBytes {
		return nil,
			fmt.Errorf(
				"%w: must be between 1 and %d",
				ErrInvalidRandomLength,
				maxRandomBytes,
			)
	}

	value :=
		make(
			[]byte,
			byteLength,
		)

	if _, err :=
		rand.Read(
			value,
		); err != nil {
		return nil,
			fmt.Errorf(
				"read cryptographic random bytes: %w",
				err,
			)
	}

	return value, nil
}

func RandomURLSafe(
	byteLength int,
) (string, error) {
	value, err :=
		RandomBytes(
			byteLength,
		)
	if err != nil {
		return "", err
	}

	return base64.RawURLEncoding.
			EncodeToString(
				value,
			),
		nil
}

func RandomHex(
	byteLength int,
) (string, error) {
	value, err :=
		RandomBytes(
			byteLength,
		)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(
		value,
	), nil
}
