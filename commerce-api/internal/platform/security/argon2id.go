package security

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	argon2Version = argon2.Version

	DefaultArgon2Memory      uint32 = 64 * 1024 // 64 MiB, expressed in KiB
	DefaultArgon2Iterations  uint32 = 3
	DefaultArgon2Parallelism uint8  = 1
	DefaultArgon2SaltLength         = 16
	DefaultArgon2KeyLength   uint32 = 32

	// This is a defensive application limit, not an Argon2 requirement.
	// It prevents pathological inputs from consuming unnecessary memory
	// before password hashing starts.
	MaxPasswordBytes = 1024
)

var (
	ErrPasswordEmpty = errors.New(
		"password is required",
	)

	ErrPasswordTooLong = errors.New(
		"password exceeds maximum supported length",
	)

	ErrInvalidArgon2Hash = errors.New(
		"invalid argon2id password hash",
	)

	ErrInvalidArgon2Parameters = errors.New(
		"invalid argon2id parameters",
	)
)

type Argon2Params struct {
	Memory      uint32
	Iterations  uint32
	Parallelism uint8
	SaltLength  int
	KeyLength   uint32
}

func DefaultArgon2Params() Argon2Params {
	return Argon2Params{
		Memory:      DefaultArgon2Memory,
		Iterations:  DefaultArgon2Iterations,
		Parallelism: DefaultArgon2Parallelism,
		SaltLength:  DefaultArgon2SaltLength,
		KeyLength:   DefaultArgon2KeyLength,
	}
}

func HashPassword(
	password string,
) (string, error) {
	return HashPasswordWithParams(
		password,
		DefaultArgon2Params(),
	)
}

func HashPasswordWithParams(
	password string,
	params Argon2Params,
) (string, error) {
	if err := validatePassword(
		password,
	); err != nil {
		return "", err
	}

	if err := validateArgon2Params(
		params,
	); err != nil {
		return "", err
	}

	salt := make(
		[]byte,
		params.SaltLength,
	)

	if _, err := rand.Read(
		salt,
	); err != nil {
		return "",
			fmt.Errorf(
				"generate argon2id salt: %w",
				err,
			)
	}

	hash := argon2.IDKey(
		[]byte(password),
		salt,
		params.Iterations,
		params.Memory,
		params.Parallelism,
		params.KeyLength,
	)

	saltEncoded := base64.RawStdEncoding.
		EncodeToString(
			salt,
		)

	hashEncoded := base64.RawStdEncoding.
		EncodeToString(
			hash,
		)

	return fmt.Sprintf(
			"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
			argon2Version,
			params.Memory,
			params.Iterations,
			params.Parallelism,
			saltEncoded,
			hashEncoded,
		),
		nil
}

func VerifyPassword(
	password string,
	encodedHash string,
) (
	bool,
	error,
) {
	if err := validatePassword(
		password,
	); err != nil {
		return false, err
	}

	params, salt, expectedHash, err :=
		parseArgon2Hash(
			encodedHash,
		)
	if err != nil {
		return false, err
	}

	actualHash := argon2.IDKey(
		[]byte(password),
		salt,
		params.Iterations,
		params.Memory,
		params.Parallelism,
		params.KeyLength,
	)

	if len(actualHash) !=
		len(expectedHash) {
		return false, nil
	}

	return subtle.ConstantTimeCompare(
			actualHash,
			expectedHash,
		) == 1,
		nil
}

// PasswordHashNeedsUpgrade reports whether an existing Argon2id hash
// uses weaker parameters than the application's current defaults.
//
// After a successful login, callers may transparently re-hash the
// supplied password when this returns true.
func PasswordHashNeedsUpgrade(
	encodedHash string,
) (
	bool,
	error,
) {
	params, _, _, err :=
		parseArgon2Hash(
			encodedHash,
		)
	if err != nil {
		return false, err
	}

	current :=
		DefaultArgon2Params()

	return params.Memory <
			current.Memory ||
			params.Iterations <
				current.Iterations ||
			params.Parallelism <
				current.Parallelism ||
			params.SaltLength <
				current.SaltLength ||
			params.KeyLength <
				current.KeyLength,
		nil
}

func parseArgon2Hash(
	encodedHash string,
) (
	Argon2Params,
	[]byte,
	[]byte,
	error,
) {
	encodedHash =
		strings.TrimSpace(
			encodedHash,
		)

	parts :=
		strings.Split(
			encodedHash,
			"$",
		)

	if len(parts) != 6 ||
		parts[0] != "" ||
		parts[1] != "argon2id" {
		return Argon2Params{},
			nil,
			nil,
			ErrInvalidArgon2Hash
	}

	var version int

	if _, err := fmt.Sscanf(
		parts[2],
		"v=%d",
		&version,
	); err != nil {
		return Argon2Params{},
			nil,
			nil,
			ErrInvalidArgon2Hash
	}

	if version !=
		argon2Version {
		return Argon2Params{},
			nil,
			nil,
			fmt.Errorf(
				"%w: unsupported version %d",
				ErrInvalidArgon2Hash,
				version,
			)
	}

	var params Argon2Params

	if _, err := fmt.Sscanf(
		parts[3],
		"m=%d,t=%d,p=%d",
		&params.Memory,
		&params.Iterations,
		&params.Parallelism,
	); err != nil {
		return Argon2Params{},
			nil,
			nil,
			ErrInvalidArgon2Hash
	}

	salt, err :=
		base64.RawStdEncoding.
			DecodeString(
				parts[4],
			)
	if err != nil ||
		len(salt) == 0 {
		return Argon2Params{},
			nil,
			nil,
			ErrInvalidArgon2Hash
	}

	expectedHash, err :=
		base64.RawStdEncoding.
			DecodeString(
				parts[5],
			)
	if err != nil ||
		len(expectedHash) == 0 {
		return Argon2Params{},
			nil,
			nil,
			ErrInvalidArgon2Hash
	}

	params.SaltLength =
		len(salt)

	params.KeyLength =
		uint32(
			len(expectedHash),
		)

	if err :=
		validateArgon2Params(
			params,
		); err != nil {
		return Argon2Params{},
			nil,
			nil,
			fmt.Errorf(
				"%w: %v",
				ErrInvalidArgon2Hash,
				err,
			)
	}

	return params,
		salt,
		expectedHash,
		nil
}

func validatePassword(
	password string,
) error {
	if password == "" {
		return ErrPasswordEmpty
	}

	if len(
		[]byte(password),
	) > MaxPasswordBytes {
		return ErrPasswordTooLong
	}

	return nil
}

func validateArgon2Params(
	params Argon2Params,
) error {
	if params.Memory <
		8*1024 {
		return fmt.Errorf(
			"%w: memory must be at least 8 MiB",
			ErrInvalidArgon2Parameters,
		)
	}

	if params.Iterations == 0 {
		return fmt.Errorf(
			"%w: iterations must be greater than zero",
			ErrInvalidArgon2Parameters,
		)
	}

	if params.Parallelism == 0 {
		return fmt.Errorf(
			"%w: parallelism must be greater than zero",
			ErrInvalidArgon2Parameters,
		)
	}

	if params.SaltLength < 16 {
		return fmt.Errorf(
			"%w: salt must be at least 16 bytes",
			ErrInvalidArgon2Parameters,
		)
	}

	if params.KeyLength < 16 {
		return fmt.Errorf(
			"%w: key length must be at least 16 bytes",
			ErrInvalidArgon2Parameters,
		)
	}

	return nil
}
