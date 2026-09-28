package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"
)

const (
	AES256KeyBytes = 32

	EncryptionEnvelopeVersion = "v1"
)

var (
	ErrInvalidEncryptionKey = errors.New(
		"invalid encryption key",
	)

	ErrUnknownEncryptionKey = errors.New(
		"unknown encryption key",
	)

	ErrInvalidCiphertext = errors.New(
		"invalid ciphertext",
	)

	ErrDecryptFailed = errors.New(
		"decrypt failed",
	)
)

type EncryptedValue struct {
	KeyID      string
	Ciphertext string
}

// Keyring supports one active encryption key plus older keys retained
// only for decryption during key rotation.
//
// The actual keys must come from application configuration or a secret
// manager. They must never be stored in PostgreSQL.
type Keyring struct {
	activeKeyID string

	keys map[string][]byte
}

func NewKeyring(
	activeKeyID string,
	keys map[string][]byte,
) (*Keyring, error) {
	activeKeyID =
		strings.TrimSpace(
			activeKeyID,
		)

	if activeKeyID == "" {
		return nil,
			fmt.Errorf(
				"%w: active key ID is required",
				ErrInvalidEncryptionKey,
			)
	}

	if len(keys) == 0 {
		return nil,
			fmt.Errorf(
				"%w: at least one key is required",
				ErrInvalidEncryptionKey,
			)
	}

	normalized :=
		make(
			map[string][]byte,
			len(keys),
		)

	for rawID, rawKey := range keys {

		keyID :=
			strings.TrimSpace(
				rawID,
			)

		if keyID == "" {
			return nil,
				fmt.Errorf(
					"%w: key ID cannot be blank",
					ErrInvalidEncryptionKey,
				)
		}

		if len(rawKey) !=
			AES256KeyBytes {
			return nil,
				fmt.Errorf(
					"%w: key %q must be exactly %d bytes",
					ErrInvalidEncryptionKey,
					keyID,
					AES256KeyBytes,
				)
		}

		keyCopy :=
			make(
				[]byte,
				len(rawKey),
			)

		copy(
			keyCopy,
			rawKey,
		)

		normalized[keyID] = keyCopy
	}

	if _, exists :=
		normalized[activeKeyID]; !exists {

		return nil,
			fmt.Errorf(
				"%w: active key %q not present in keyring",
				ErrInvalidEncryptionKey,
				activeKeyID,
			)
	}

	return &Keyring{
			activeKeyID: activeKeyID,

			keys: normalized,
		},
		nil
}

func NewSingleKeyKeyring(
	keyID string,
	key []byte,
) (*Keyring, error) {
	return NewKeyring(
		keyID,
		map[string][]byte{
			keyID: key,
		},
	)
}

func (k *Keyring) ActiveKeyID() string {
	if k == nil {
		return ""
	}

	return k.activeKeyID
}

// EncryptString encrypts plaintext using AES-256-GCM.
//
// associatedData is authenticated but not encrypted. For Admin TOTP,
// we will bind ciphertext to stable record identity such as:
//
//	admin-totp:<staff_account_id>:<credential_id>
//
// This prevents valid ciphertext from one database record being moved
// to another record and successfully decrypted.
func (k *Keyring) EncryptString(
	plaintext string,
	associatedData []byte,
) (
	EncryptedValue,
	error,
) {
	if k == nil {
		return EncryptedValue{},
			ErrInvalidEncryptionKey
	}

	key, exists :=
		k.keys[k.activeKeyID]
	if !exists {
		return EncryptedValue{},
			ErrUnknownEncryptionKey
	}

	block, err :=
		aes.NewCipher(
			key,
		)
	if err != nil {
		return EncryptedValue{},
			fmt.Errorf(
				"create AES cipher: %w",
				err,
			)
	}

	gcm, err :=
		cipher.NewGCM(
			block,
		)
	if err != nil {
		return EncryptedValue{},
			fmt.Errorf(
				"create AES-GCM cipher: %w",
				err,
			)
	}

	nonce :=
		make(
			[]byte,
			gcm.NonceSize(),
		)

	if _, err :=
		io.ReadFull(
			rand.Reader,
			nonce,
		); err != nil {

		return EncryptedValue{},
			fmt.Errorf(
				"generate encryption nonce: %w",
				err,
			)
	}

	aad :=
		encryptionAAD(
			k.activeKeyID,
			associatedData,
		)

	ciphertext :=
		gcm.Seal(
			nil,
			nonce,
			[]byte(
				plaintext,
			),
			aad,
		)

	envelope :=
		strings.Join(
			[]string{
				EncryptionEnvelopeVersion,

				base64.RawURLEncoding.
					EncodeToString(
						nonce,
					),

				base64.RawURLEncoding.
					EncodeToString(
						ciphertext,
					),
			},
			".",
		)

	return EncryptedValue{
			KeyID: k.activeKeyID,

			Ciphertext: envelope,
		},
		nil
}

func (k *Keyring) DecryptString(
	value EncryptedValue,
	associatedData []byte,
) (
	string,
	error,
) {
	if k == nil {
		return "",
			ErrInvalidEncryptionKey
	}

	keyID :=
		strings.TrimSpace(
			value.KeyID,
		)

	if keyID == "" {
		return "",
			ErrUnknownEncryptionKey
	}

	key, exists :=
		k.keys[keyID]
	if !exists {
		return "",
			fmt.Errorf(
				"%w: %s",
				ErrUnknownEncryptionKey,
				keyID,
			)
	}

	parts :=
		strings.Split(
			strings.TrimSpace(
				value.Ciphertext,
			),
			".",
		)

	if len(parts) != 3 ||
		parts[0] !=
			EncryptionEnvelopeVersion {

		return "",
			ErrInvalidCiphertext
	}

	nonce, err :=
		base64.RawURLEncoding.
			DecodeString(
				parts[1],
			)
	if err != nil {
		return "",
			ErrInvalidCiphertext
	}

	ciphertext, err :=
		base64.RawURLEncoding.
			DecodeString(
				parts[2],
			)
	if err != nil {
		return "",
			ErrInvalidCiphertext
	}

	block, err :=
		aes.NewCipher(
			key,
		)
	if err != nil {
		return "",
			fmt.Errorf(
				"create AES cipher: %w",
				err,
			)
	}

	gcm, err :=
		cipher.NewGCM(
			block,
		)
	if err != nil {
		return "",
			fmt.Errorf(
				"create AES-GCM cipher: %w",
				err,
			)
	}

	if len(nonce) !=
		gcm.NonceSize() {
		return "",
			ErrInvalidCiphertext
	}

	aad :=
		encryptionAAD(
			keyID,
			associatedData,
		)

	plaintext, err :=
		gcm.Open(
			nil,
			nonce,
			ciphertext,
			aad,
		)
	if err != nil {
		return "",
			ErrDecryptFailed
	}

	return string(
			plaintext,
		),
		nil
}

// DecodeAES256KeyBase64 parses a base64-encoded 256-bit application
// encryption key.
func DecodeAES256KeyBase64(
	value string,
) ([]byte, error) {
	value =
		strings.TrimSpace(
			value,
		)

	if value == "" {
		return nil,
			ErrInvalidEncryptionKey
	}

	encodings :=
		[]*base64.Encoding{
			base64.StdEncoding,
			base64.RawStdEncoding,
			base64.URLEncoding,
			base64.RawURLEncoding,
		}

	for _, encoding := range encodings {

		decoded, err :=
			encoding.DecodeString(
				value,
			)
		if err != nil {
			continue
		}

		if len(decoded) !=
			AES256KeyBytes {
			continue
		}

		return decoded,
			nil
	}

	return nil,
		fmt.Errorf(
			"%w: expected base64 encoding of exactly %d bytes",
			ErrInvalidEncryptionKey,
			AES256KeyBytes,
		)
}

func encryptionAAD(
	keyID string,
	associatedData []byte,
) []byte {
	prefix :=
		[]byte(
			EncryptionEnvelopeVersion +
				"\x00" +
				keyID +
				"\x00",
		)

	result :=
		make(
			[]byte,
			0,
			len(prefix)+
				len(associatedData),
		)

	result =
		append(
			result,
			prefix...,
		)

	result =
		append(
			result,
			associatedData...,
		)

	return result
}
