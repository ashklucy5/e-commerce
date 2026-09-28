package delivery

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"
)

const (
	OTPChannelSMS = "sms"
)

const (
	OTPStatusPending   = "pending"
	OTPStatusVerified  = "verified"
	OTPStatusConsumed  = "consumed"
	OTPStatusExpired   = "expired"
	OTPStatusLocked    = "locked"
	OTPStatusCancelled = "cancelled"
)

type OTPConfig struct {
	TTL         time.Duration
	MaxAttempts int
	CodeDigits  int
	BcryptCost  int
}

func DefaultOTPConfig() OTPConfig {
	return OTPConfig{
		TTL:         5 * time.Minute,
		MaxAttempts: 5,
		CodeDigits:  6,
		BcryptCost:  bcrypt.DefaultCost,
	}
}

func normalizeOTPConfig(
	config OTPConfig,
) OTPConfig {
	if config.TTL <
		30*time.Second ||
		config.TTL >
			30*time.Minute {
		config.TTL =
			5 * time.Minute
	}

	if config.MaxAttempts < 1 ||
		config.MaxAttempts > 10 {
		config.MaxAttempts =
			5
	}

	if config.CodeDigits < 4 ||
		config.CodeDigits > 8 {
		config.CodeDigits =
			6
	}

	if config.BcryptCost <
		bcrypt.MinCost ||
		config.BcryptCost >
			bcrypt.MaxCost {
		config.BcryptCost =
			bcrypt.DefaultCost
	}

	return config
}

type OTPChallenge struct {
	ID         string `json:"id"`
	ShipmentID string `json:"shipment_id"`

	Purpose string `json:"purpose"`
	Channel string `json:"channel"`

	Recipient string `json:"recipient"`

	Status string `json:"status"`

	ExpiresAt time.Time `json:"expires_at"`

	AttemptCount int `json:"attempt_count"`
	MaxAttempts  int `json:"max_attempts"`

	VerifiedAt *time.Time `json:"verified_at,omitempty"`
	ConsumedAt *time.Time `json:"consumed_at,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type IssueOTPInput struct {
	ShipmentID string `json:"shipment_id"`

	Purpose string `json:"purpose"`

	Channel string `json:"channel,omitempty"`

	Recipient string `json:"recipient"`
}

type VerifyOTPInput struct {
	ChallengeID string `json:"challenge_id"`

	Code string `json:"code"`

	Source string `json:"source"`

	ActorID string `json:"actor_id,omitempty"`
}

type ConsumeOTPInput struct {
	ChallengeID string `json:"challenge_id"`

	ShipmentID string `json:"shipment_id"`

	Purpose string `json:"purpose"`
}

// OTPMessage is the only structure that contains the plaintext code.
//
// It exists only in memory and is passed directly to OTPSender.
// The database stores a bcrypt hash instead.
type OTPMessage struct {
	ChallengeID string
	ShipmentID  string

	Purpose string
	Channel string

	Recipient string

	Code string

	ExpiresAt time.Time
}

// OTPSender is the seam that the future SMS service implements.
//
// Delivery does not know anything about Twilio, local Bangladesh SMS
// gateways, templates, HTTP APIs, credentials, or retries.
type OTPSender interface {
	SendOTP(
		ctx context.Context,
		message OTPMessage,
	) error
}

// OTPSenderFunc makes adapters/testing simple without another struct.
type OTPSenderFunc func(
	ctx context.Context,
	message OTPMessage,
) error

func (f OTPSenderFunc) SendOTP(
	ctx context.Context,
	message OTPMessage,
) error {
	return f(
		ctx,
		message,
	)
}

func (s *Service) SetOTPSender(
	sender OTPSender,
) {
	s.otpSender =
		sender
}

func (s *Service) SetOTPConfig(
	config OTPConfig,
) {
	s.otpConfig =
		normalizeOTPConfig(
			config,
		)
}

func generateOTPCode(
	digits int,
) (string, error) {
	if digits < 4 ||
		digits > 8 {
		return "",
			ErrInvalidInput
	}

	var builder strings.Builder

	builder.Grow(
		digits,
	)

	limit :=
		big.NewInt(
			10,
		)

	for i := 0; i < digits; i++ {

		value, err :=
			rand.Int(
				rand.Reader,
				limit,
			)
		if err != nil {
			return "",
				fmt.Errorf(
					"generate delivery OTP: %w",
					err,
				)
		}

		builder.WriteByte(
			byte(
				'0' +
					value.Int64(),
			),
		)
	}

	return builder.String(),
		nil
}

func normalizeIssueOTPInput(
	input *IssueOTPInput,
) {
	input.ShipmentID =
		strings.TrimSpace(
			input.ShipmentID,
		)

	input.Purpose =
		strings.ToLower(
			strings.TrimSpace(
				input.Purpose,
			),
		)

	input.Channel =
		strings.ToLower(
			strings.TrimSpace(
				input.Channel,
			),
		)

	input.Recipient =
		strings.TrimSpace(
			input.Recipient,
		)

	if input.Channel == "" {
		input.Channel =
			OTPChannelSMS
	}
}

func validateIssueOTPInput(
	input IssueOTPInput,
) error {
	if !uuidPattern.MatchString(
		input.ShipmentID,
	) ||
		!validProofPurpose(
			input.Purpose,
		) ||
		input.Channel !=
			OTPChannelSMS ||
		input.Recipient == "" ||
		utf8.RuneCountInString(
			input.Recipient,
		) > 255 {
		return ErrInvalidInput
	}

	return nil
}

func (s *Service) IssueOTP(
	ctx context.Context,
	input IssueOTPInput,
) (OTPChallenge, error) {
	normalizeIssueOTPInput(
		&input,
	)

	if err :=
		validateIssueOTPInput(
			input,
		); err != nil {
		return OTPChallenge{}, err
	}

	if s.otpSender == nil {
		return OTPChallenge{},
			ErrOTPSenderUnavailable
	}

	config :=
		normalizeOTPConfig(
			s.otpConfig,
		)

	code, err :=
		generateOTPCode(
			config.CodeDigits,
		)
	if err != nil {
		return OTPChallenge{}, err
	}

	hash, err :=
		bcrypt.GenerateFromPassword(
			[]byte(
				code,
			),
			config.BcryptCost,
		)
	if err != nil {
		return OTPChallenge{},
			fmt.Errorf(
				"hash delivery OTP: %w",
				err,
			)
	}

	now :=
		time.Now().UTC()

	expiresAt :=
		now.Add(
			config.TTL,
		)

	tx, err :=
		s.repository.Begin(
			ctx,
		)
	if err != nil {
		return OTPChallenge{}, err
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	shipment, err :=
		s.repository.LockShipmentTx(
			ctx,
			tx,
			input.ShipmentID,
		)
	if err != nil {
		return OTPChallenge{}, err
	}

	if shipment.Status ==
		ShipmentStatusCancelled {
		return OTPChallenge{},
			ErrShipmentNotReady
	}

	// Reissuing an OTP invalidates the previous pending/verified
	// challenge for the same shipment/purpose/recipient.
	if err :=
		s.repository.CancelActiveOTPChallengesTx(
			ctx,
			tx,
			input.ShipmentID,
			input.Purpose,
			input.Channel,
			input.Recipient,
		); err != nil {
		return OTPChallenge{}, err
	}

	challengeID, err :=
		s.repository.CreateOTPChallengeTx(
			ctx,
			tx,
			input,
			string(
				hash,
			),
			expiresAt,
			config.MaxAttempts,
		)
	if err != nil {
		return OTPChallenge{}, err
	}

	if err :=
		tx.Commit(
			ctx,
		); err != nil {
		return OTPChallenge{},
			fmt.Errorf(
				"commit delivery OTP challenge: %w",
				err,
			)
	}

	message :=
		OTPMessage{
			ChallengeID: challengeID,

			ShipmentID: input.ShipmentID,

			Purpose: input.Purpose,

			Channel: input.Channel,

			Recipient: input.Recipient,

			Code: code,

			ExpiresAt: expiresAt,
		}

	// Sending happens after the challenge exists durably.
	//
	// If the SMS provider fails, invalidate the challenge so the
	// user cannot be expected to verify a code they never received.
	if err :=
		s.otpSender.SendOTP(
			ctx,
			message,
		); err != nil {

		_ =
			s.repository.CancelOTPChallenge(
				ctx,
				challengeID,
			)

		return OTPChallenge{},
			fmt.Errorf(
				"send delivery OTP: %w",
				err,
			)
	}

	return s.repository.GetOTPChallenge(
		ctx,
		challengeID,
	)
}

func (s *Service) VerifyOTP(
	ctx context.Context,
	input VerifyOTPInput,
) (OTPChallenge, error) {
	input.ChallengeID =
		strings.TrimSpace(
			input.ChallengeID,
		)

	input.Code =
		strings.TrimSpace(
			input.Code,
		)

	input.Source =
		strings.ToLower(
			strings.TrimSpace(
				input.Source,
			),
		)

	input.ActorID =
		strings.TrimSpace(
			input.ActorID,
		)

	if !uuidPattern.MatchString(
		input.ChallengeID,
	) ||
		len(
			input.Code,
		) < 4 ||
		len(
			input.Code,
		) > 8 ||
		!allDigits(
			input.Code,
		) ||
		!validTrackingSource(
			input.Source,
		) ||
		utf8.RuneCountInString(
			input.ActorID,
		) > 160 {
		return OTPChallenge{},
			ErrInvalidInput
	}

	tx, err :=
		s.repository.Begin(
			ctx,
		)
	if err != nil {
		return OTPChallenge{}, err
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	challenge,
		codeHash,
		err :=
		s.repository.LockOTPChallengeWithHashTx(
			ctx,
			tx,
			input.ChallengeID,
		)
	if err != nil {
		return OTPChallenge{}, err
	}

	now :=
		time.Now().UTC()

	if challenge.Status ==
		OTPStatusConsumed {
		return OTPChallenge{},
			ErrOTPChallengeConsumed
	}

	if challenge.Status ==
		OTPStatusCancelled {
		return OTPChallenge{},
			ErrOTPChallengeNotFound
	}

	if challenge.Status ==
		OTPStatusLocked {
		return OTPChallenge{},
			ErrOTPChallengeLocked
	}

	if challenge.Status ==
		OTPStatusExpired ||
		!challenge.ExpiresAt.After(
			now,
		) {

		if challenge.Status !=
			OTPStatusExpired {

			if err :=
				s.repository.MarkOTPExpiredTx(
					ctx,
					tx,
					challenge.ID,
				); err != nil {
				return OTPChallenge{},
					err
			}

			if err :=
				tx.Commit(
					ctx,
				); err != nil {
				return OTPChallenge{},
					fmt.Errorf(
						"commit expired delivery OTP: %w",
						err,
					)
			}
		}

		return OTPChallenge{},
			ErrOTPChallengeExpired
	}

	compareErr :=
		bcrypt.CompareHashAndPassword(
			[]byte(
				codeHash,
			),
			[]byte(
				input.Code,
			),
		)

	// Verification is idempotent, but still requires the correct
	// code when replayed.
	if challenge.Status ==
		OTPStatusVerified {

		if compareErr != nil {
			return OTPChallenge{},
				ErrOTPInvalidCode
		}

		if err :=
			tx.Rollback(
				ctx,
			); err != nil {
			return OTPChallenge{},
				fmt.Errorf(
					"rollback repeated delivery OTP verification: %w",
					err,
				)
		}

		return challenge, nil
	}

	if challenge.Status !=
		OTPStatusPending {
		return OTPChallenge{},
			ErrOTPChallengeNotFound
	}

	if compareErr != nil {
		locked, err :=
			s.repository.RegisterOTPFailureTx(
				ctx,
				tx,
				challenge.ID,
			)
		if err != nil {
			return OTPChallenge{},
				err
		}

		if err :=
			tx.Commit(
				ctx,
			); err != nil {
			return OTPChallenge{},
				fmt.Errorf(
					"commit delivery OTP failure: %w",
					err,
				)
		}

		if locked {
			return OTPChallenge{},
				ErrOTPChallengeLocked
		}

		return OTPChallenge{},
			ErrOTPInvalidCode
	}

	if err :=
		s.repository.MarkOTPVerifiedTx(
			ctx,
			tx,
			challenge.ID,
			now,
		); err != nil {
		return OTPChallenge{}, err
	}

	// OTP verification also becomes a durable delivery proof.
	if _, err :=
		s.repository.InsertProofTx(
			ctx,
			tx,
			CreateProofInput{
				ShipmentID: challenge.ShipmentID,

				OTPChallengeID: challenge.ID,

				Purpose: challenge.Purpose,

				ProofType: ProofTypeOTP,

				Source: input.Source,

				ActorID: input.ActorID,

				VerificationStatus: ProofVerificationVerified,

				OccurredAt: &now,
			},
		); err != nil {
		return OTPChallenge{},
			err
	}

	if err :=
		tx.Commit(
			ctx,
		); err != nil {
		return OTPChallenge{},
			fmt.Errorf(
				"commit delivery OTP verification: %w",
				err,
			)
	}

	return s.repository.GetOTPChallenge(
		ctx,
		challenge.ID,
	)
}

func (s *Service) ConsumeOTP(
	ctx context.Context,
	input ConsumeOTPInput,
) (OTPChallenge, error) {
	input.ChallengeID =
		strings.TrimSpace(
			input.ChallengeID,
		)

	input.ShipmentID =
		strings.TrimSpace(
			input.ShipmentID,
		)

	input.Purpose =
		strings.ToLower(
			strings.TrimSpace(
				input.Purpose,
			),
		)

	if !uuidPattern.MatchString(
		input.ChallengeID,
	) ||
		!uuidPattern.MatchString(
			input.ShipmentID,
		) ||
		!validProofPurpose(
			input.Purpose,
		) {
		return OTPChallenge{},
			ErrInvalidInput
	}

	tx, err :=
		s.repository.Begin(
			ctx,
		)
	if err != nil {
		return OTPChallenge{}, err
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	challenge, err :=
		s.repository.LockOTPChallengeTx(
			ctx,
			tx,
			input.ChallengeID,
		)
	if err != nil {
		return OTPChallenge{}, err
	}

	if challenge.ShipmentID !=
		input.ShipmentID ||
		challenge.Purpose !=
			input.Purpose {
		return OTPChallenge{},
			ErrOTPChallengeNotFound
	}

	if challenge.Status ==
		OTPStatusConsumed {

		if err :=
			tx.Rollback(
				ctx,
			); err != nil {
			return OTPChallenge{},
				fmt.Errorf(
					"rollback repeated delivery OTP consumption: %w",
					err,
				)
		}

		return challenge, nil
	}

	if challenge.Status !=
		OTPStatusVerified {
		return OTPChallenge{},
			ErrOTPChallengeNotVerified
	}

	if !challenge.ExpiresAt.After(
		time.Now().UTC(),
	) {
		if err :=
			s.repository.MarkOTPExpiredTx(
				ctx,
				tx,
				challenge.ID,
			); err != nil {
			return OTPChallenge{},
				err
		}

		if err :=
			tx.Commit(
				ctx,
			); err != nil {
			return OTPChallenge{},
				fmt.Errorf(
					"commit expired verified delivery OTP: %w",
					err,
				)
		}

		return OTPChallenge{},
			ErrOTPChallengeExpired
	}

	now :=
		time.Now().UTC()

	if err :=
		s.repository.ConsumeOTPChallengeTx(
			ctx,
			tx,
			challenge.ID,
			now,
		); err != nil {
		return OTPChallenge{},
			err
	}

	if err :=
		tx.Commit(
			ctx,
		); err != nil {
		return OTPChallenge{},
			fmt.Errorf(
				"commit delivery OTP consumption: %w",
				err,
			)
	}

	return s.repository.GetOTPChallenge(
		ctx,
		challenge.ID,
	)
}

func (s *Service) GetOTPChallenge(
	ctx context.Context,
	challengeID string,
) (OTPChallenge, error) {
	challengeID =
		strings.TrimSpace(
			challengeID,
		)

	if !uuidPattern.MatchString(
		challengeID,
	) {
		return OTPChallenge{},
			ErrInvalidInput
	}

	return s.repository.GetOTPChallenge(
		ctx,
		challengeID,
	)
}

func allDigits(
	value string,
) bool {
	for _, r := range value {

		if r < '0' ||
			r > '9' {
			return false
		}
	}

	return value != ""
}
