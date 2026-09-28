package adminauth

import (
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base32"
	"encoding/binary"
	"errors"
	"fmt"
	"hash"
	"net/url"
	"strconv"
	"strings"
	"time"

	platformsecurity "project.local/commerce-api/internal/platform/security"
)

const (
	DefaultTOTPAlgorithm = "SHA1"

	DefaultTOTPDigits = 6

	DefaultTOTPPeriodSeconds int64 = 30

	DefaultTOTPSkew int64 = 1

	DefaultTOTPSecretBytes = 32
)

var (
	ErrTOTPSecretRequired = errors.New(
		"TOTP secret is required",
	)

	ErrInvalidTOTPSecret = errors.New(
		"invalid TOTP secret",
	)

	ErrInvalidTOTPCode = errors.New(
		"invalid TOTP code",
	)

	ErrInvalidTOTPConfig = errors.New(
		"invalid TOTP configuration",
	)
)

type TOTPConfig struct {
	Algorithm string

	Digits int

	PeriodSeconds int64

	// Skew controls how many periods either side of the current time
	// are accepted during verification.
	//
	// A value of 1 accepts previous/current/next time steps.
	Skew int64
}

type TOTPVerification struct {
	Valid bool

	Step int64
}

func DefaultTOTPConfig() TOTPConfig {
	return TOTPConfig{
		Algorithm: DefaultTOTPAlgorithm,

		Digits: DefaultTOTPDigits,

		PeriodSeconds: DefaultTOTPPeriodSeconds,

		Skew: DefaultTOTPSkew,
	}
}

func GenerateTOTPSecret() (
	string,
	error,
) {
	random, err :=
		platformsecurity.RandomBytes(
			DefaultTOTPSecretBytes,
		)
	if err != nil {
		return "",
			fmt.Errorf(
				"generate TOTP secret: %w",
				err,
			)
	}

	return base32.StdEncoding.
			WithPadding(
				base32.NoPadding,
			).
			EncodeToString(
				random,
			),
		nil
}

func BuildTOTPEnrollmentURI(
	issuer string,
	accountName string,
	secret string,
	cfg TOTPConfig,
) (
	string,
	error,
) {
	issuer =
		strings.TrimSpace(
			issuer,
		)

	accountName =
		strings.TrimSpace(
			accountName,
		)

	if issuer == "" {
		return "",
			fmt.Errorf(
				"%w: issuer is required",
				ErrInvalidTOTPConfig,
			)
	}

	if accountName == "" {
		return "",
			fmt.Errorf(
				"%w: account name is required",
				ErrInvalidTOTPConfig,
			)
	}

	if err :=
		validateTOTPConfig(
			cfg,
		); err != nil {

		return "", err
	}

	normalizedSecret,
		_,
		err :=
		decodeTOTPSecret(
			secret,
		)
	if err != nil {
		return "", err
	}

	query :=
		url.Values{}

	query.Set(
		"secret",
		normalizedSecret,
	)

	query.Set(
		"issuer",
		issuer,
	)

	query.Set(
		"algorithm",
		normalizeTOTPAlgorithm(
			cfg.Algorithm,
		),
	)

	query.Set(
		"digits",
		strconv.Itoa(
			cfg.Digits,
		),
	)

	query.Set(
		"period",
		strconv.FormatInt(
			cfg.PeriodSeconds,
			10,
		),
	)

	uri :=
		&url.URL{
			Scheme: "otpauth",

			Host: "totp",

			Path: "/" +
				issuer +
				":" +
				accountName,

			RawQuery: query.Encode(),
		}

	return uri.String(),
		nil
}

func GenerateTOTP(
	secret string,
	at time.Time,
	cfg TOTPConfig,
) (
	string,
	int64,
	error,
) {
	if err :=
		validateTOTPConfig(
			cfg,
		); err != nil {

		return "", 0, err
	}

	_,
		secretBytes,
		err :=
		decodeTOTPSecret(
			secret,
		)
	if err != nil {
		return "", 0, err
	}

	unix :=
		at.UTC().
			Unix()

	if unix < 0 {
		return "",
			0,
			fmt.Errorf(
				"%w: time cannot be before Unix epoch",
				ErrInvalidTOTPConfig,
			)
	}

	step :=
		unix /
			cfg.PeriodSeconds

	code, err :=
		generateTOTPForStep(
			secretBytes,
			step,
			cfg,
		)
	if err != nil {
		return "",
			0,
			err
	}

	return code,
		step,
		nil
}

// VerifyTOTP verifies a code and returns the accepted TOTP time step.
//
// lastAcceptedStep enables replay protection. When supplied, a valid
// code from the same or an older step will not be accepted.
//
// The repository will persist the returned Step atomically when a code
// succeeds.
func VerifyTOTP(
	secret string,
	code string,
	at time.Time,
	lastAcceptedStep *int64,
	cfg TOTPConfig,
) (
	TOTPVerification,
	error,
) {
	if err :=
		validateTOTPConfig(
			cfg,
		); err != nil {

		return TOTPVerification{},
			err
	}

	code =
		strings.TrimSpace(
			code,
		)

	if !validTOTPCode(
		code,
		cfg.Digits,
	) {
		return TOTPVerification{},
			ErrInvalidTOTPCode
	}

	_,
		secretBytes,
		err :=
		decodeTOTPSecret(
			secret,
		)
	if err != nil {
		return TOTPVerification{},
			err
	}

	unix :=
		at.UTC().
			Unix()

	if unix < 0 {
		return TOTPVerification{},
			fmt.Errorf(
				"%w: time cannot be before Unix epoch",
				ErrInvalidTOTPConfig,
			)
	}

	currentStep :=
		unix /
			cfg.PeriodSeconds

	for _, offset := range verificationOffsets(
		cfg.Skew,
	) {

		candidateStep :=
			currentStep +
				offset

		if candidateStep < 0 {
			continue
		}

		if lastAcceptedStep != nil &&
			candidateStep <=
				*lastAcceptedStep {

			continue
		}

		expected, err :=
			generateTOTPForStep(
				secretBytes,
				candidateStep,
				cfg,
			)
		if err != nil {
			return TOTPVerification{},
				err
		}

		if platformsecurity.
			ConstantTimeEqual(
				code,
				expected,
			) {

			return TOTPVerification{
					Valid: true,

					Step: candidateStep,
				},
				nil
		}
	}

	return TOTPVerification{
			Valid: false,
		},
		nil
}

func generateTOTPForStep(
	secret []byte,
	step int64,
	cfg TOTPConfig,
) (
	string,
	error,
) {
	hashFunc, err :=
		totpHash(
			cfg.Algorithm,
		)
	if err != nil {
		return "", err
	}

	counter :=
		make(
			[]byte,
			8,
		)

	binary.BigEndian.
		PutUint64(
			counter,
			uint64(
				step,
			),
		)

	mac :=
		hmac.New(
			hashFunc,
			secret,
		)

	_, _ =
		mac.Write(
			counter,
		)

	sum :=
		mac.Sum(
			nil,
		)

	offset :=
		sum[len(sum)-1] & 0x0f

	binaryCode :=
		binary.BigEndian.
			Uint32(
				sum[offset:offset+4],
			) &
			0x7fffffff

	var modulo uint32

	switch cfg.Digits {
	case 6:
		modulo =
			1_000_000

	case 8:
		modulo =
			100_000_000

	default:
		return "",
			ErrInvalidTOTPConfig
	}

	value :=
		binaryCode %
			modulo

	return fmt.Sprintf(
			"%0*d",
			cfg.Digits,
			value,
		),
		nil
}

func validateTOTPConfig(
	cfg TOTPConfig,
) error {
	switch normalizeTOTPAlgorithm(
		cfg.Algorithm,
	) {
	case "SHA1",
		"SHA256",
		"SHA512":

	default:
		return fmt.Errorf(
			"%w: unsupported algorithm",
			ErrInvalidTOTPConfig,
		)
	}

	if cfg.Digits != 6 &&
		cfg.Digits != 8 {

		return fmt.Errorf(
			"%w: digits must be 6 or 8",
			ErrInvalidTOTPConfig,
		)
	}

	if cfg.PeriodSeconds < 15 ||
		cfg.PeriodSeconds > 120 {

		return fmt.Errorf(
			"%w: period must be between 15 and 120 seconds",
			ErrInvalidTOTPConfig,
		)
	}

	if cfg.Skew < 0 ||
		cfg.Skew > 2 {

		return fmt.Errorf(
			"%w: skew must be between 0 and 2",
			ErrInvalidTOTPConfig,
		)
	}

	return nil
}

func decodeTOTPSecret(
	secret string,
) (
	string,
	[]byte,
	error,
) {
	secret =
		strings.ToUpper(
			strings.ReplaceAll(
				strings.TrimSpace(
					secret,
				),
				" ",
				"",
			),
		)

	if secret == "" {
		return "",
			nil,
			ErrTOTPSecretRequired
	}

	decoded, err :=
		base32.StdEncoding.
			WithPadding(
				base32.NoPadding,
			).
			DecodeString(
				secret,
			)
	if err != nil {
		return "",
			nil,
			ErrInvalidTOTPSecret
	}

	// We control enrollment and generate 32-byte secrets. Twenty bytes
	// is retained as the minimum for standards compatibility.
	if len(decoded) < 20 {
		return "",
			nil,
			ErrInvalidTOTPSecret
	}

	return secret,
		decoded,
		nil
}

func totpHash(
	algorithm string,
) (
	func() hash.Hash,
	error,
) {
	switch normalizeTOTPAlgorithm(
		algorithm,
	) {
	case "SHA1":
		return sha1.New,
			nil

	case "SHA256":
		return sha256.New,
			nil

	case "SHA512":
		return sha512.New,
			nil

	default:
		return nil,
			ErrInvalidTOTPConfig
	}
}

func normalizeTOTPAlgorithm(
	value string,
) string {
	return strings.ToUpper(
		strings.TrimSpace(
			value,
		),
	)
}

func validTOTPCode(
	code string,
	digits int,
) bool {
	if len(code) !=
		digits {

		return false
	}

	for index :=
		0; index < len(code); index++ {

		if code[index] < '0' ||
			code[index] > '9' {

			return false
		}
	}

	return true
}

func verificationOffsets(
	skew int64,
) []int64 {
	result :=
		[]int64{
			0,
		}

	for distance :=
		int64(1); distance <= skew; distance++ {

		result =
			append(
				result,
				-distance,
				distance,
			)
	}

	return result
}
