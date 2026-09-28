package security

import (
	"encoding/base64"
	"errors"
	"strings"
	"testing"
)

func TestAES256GCMEncryptDecrypt(
	t *testing.T,
) {
	key :=
		make(
			[]byte,
			AES256KeyBytes,
		)

	for index := range key {

		key[index] =
			byte(
				index + 1,
			)
	}

	keyring, err :=
		NewSingleKeyKeyring(
			"test-key-v1",
			key,
		)
	if err != nil {
		t.Fatalf(
			"create keyring: %v",
			err,
		)
	}

	const plaintext = "JBSWY3DPEHPK3PXP"

	aad :=
		[]byte(
			"admin-totp:staff-1:credential-1",
		)

	encrypted, err :=
		keyring.EncryptString(
			plaintext,
			aad,
		)
	if err != nil {
		t.Fatalf(
			"encrypt: %v",
			err,
		)
	}

	if encrypted.KeyID !=
		"test-key-v1" {

		t.Fatalf(
			"unexpected key ID %q",
			encrypted.KeyID,
		)
	}

	if strings.Contains(
		encrypted.Ciphertext,
		plaintext,
	) {
		t.Fatal(
			"ciphertext contains plaintext",
		)
	}

	decrypted, err :=
		keyring.DecryptString(
			encrypted,
			aad,
		)
	if err != nil {
		t.Fatalf(
			"decrypt: %v",
			err,
		)
	}

	if decrypted !=
		plaintext {

		t.Fatalf(
			"expected %q, got %q",
			plaintext,
			decrypted,
		)
	}
}

func TestAES256GCMRejectsWrongAssociatedData(
	t *testing.T,
) {
	key :=
		make(
			[]byte,
			AES256KeyBytes,
		)

	keyring, err :=
		NewSingleKeyKeyring(
			"test-key-v1",
			key,
		)
	if err != nil {
		t.Fatalf(
			"create keyring: %v",
			err,
		)
	}

	encrypted, err :=
		keyring.EncryptString(
			"secret",
			[]byte(
				"record-a",
			),
		)
	if err != nil {
		t.Fatalf(
			"encrypt: %v",
			err,
		)
	}

	_, err =
		keyring.DecryptString(
			encrypted,
			[]byte(
				"record-b",
			),
		)

	if !errors.Is(
		err,
		ErrDecryptFailed,
	) {
		t.Fatalf(
			"expected ErrDecryptFailed, got %v",
			err,
		)
	}
}

func TestAES256GCMKeyRotation(
	t *testing.T,
) {
	oldKey :=
		make(
			[]byte,
			AES256KeyBytes,
		)

	newKey :=
		make(
			[]byte,
			AES256KeyBytes,
		)

	for index := range newKey {

		newKey[index] =
			byte(
				index + 10,
			)
	}

	oldRing, err :=
		NewSingleKeyKeyring(
			"key-v1",
			oldKey,
		)
	if err != nil {
		t.Fatalf(
			"create old keyring: %v",
			err,
		)
	}

	encrypted, err :=
		oldRing.EncryptString(
			"rotatable-secret",
			nil,
		)
	if err != nil {
		t.Fatalf(
			"encrypt old value: %v",
			err,
		)
	}

	rotatedRing, err :=
		NewKeyring(
			"key-v2",
			map[string][]byte{
				"key-v1": oldKey,

				"key-v2": newKey,
			},
		)
	if err != nil {
		t.Fatalf(
			"create rotated keyring: %v",
			err,
		)
	}

	decrypted, err :=
		rotatedRing.DecryptString(
			encrypted,
			nil,
		)
	if err != nil {
		t.Fatalf(
			"decrypt old ciphertext with rotated keyring: %v",
			err,
		)
	}

	if decrypted !=
		"rotatable-secret" {

		t.Fatalf(
			"unexpected plaintext %q",
			decrypted,
		)
	}

	newEncrypted, err :=
		rotatedRing.EncryptString(
			"new-secret",
			nil,
		)
	if err != nil {
		t.Fatalf(
			"encrypt with active rotated key: %v",
			err,
		)
	}

	if newEncrypted.KeyID !=
		"key-v2" {

		t.Fatalf(
			"expected key-v2, got %q",
			newEncrypted.KeyID,
		)
	}
}

func TestDecodeAES256KeyBase64(
	t *testing.T,
) {
	key :=
		make(
			[]byte,
			AES256KeyBytes,
		)

	encoded :=
		base64.StdEncoding.
			EncodeToString(
				key,
			)

	decoded, err :=
		DecodeAES256KeyBase64(
			encoded,
		)
	if err != nil {
		t.Fatalf(
			"decode key: %v",
			err,
		)
	}

	if len(decoded) !=
		AES256KeyBytes {

		t.Fatalf(
			"expected %d bytes, got %d",
			AES256KeyBytes,
			len(decoded),
		)
	}
}
