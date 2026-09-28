package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	registrationOTPChallengeTTL = 5 * time.Minute

	registrationOTPExpiredGrace = 1 * time.Minute

	registrationOTPResendCooldown = 45 * time.Second

	registrationOTPMaxAttempts = 5

	registrationOTPMaxResends = 5

	registrationOTPLockTTL = 20 * time.Second
)

const (
	registrationOTPKeyPrefix = "auth:customer:register:challenge:"

	registrationOTPLockPrefix = "auth:customer:register:lock:"
)

type RegistrationOTPSender interface {
	SendRegistrationOTP(
		ctx context.Context,
		phone string,
		code string,
	) error
}

/*
developmentRegistrationOTPSender is used while we do not
yet have a real Bangladesh SMS gateway connected.

The OTP is written to the backend/Air terminal so the
complete registration flow can be runtime-tested.
*/
type developmentRegistrationOTPSender struct{}

func (
	*developmentRegistrationOTPSender,
) SendRegistrationOTP(
	_ context.Context,
	phone string,
	code string,
) error {
	log.Printf(
		"[customer-auth][development] registration OTP phone=%s code=%s",
		phone,
		code,
	)

	return nil
}

/*
Production must never silently fall back to logging OTPs.
Until an SMS provider is configured, production delivery
therefore fails closed.
*/
type unavailableRegistrationOTPSender struct{}

func (
	*unavailableRegistrationOTPSender,
) SendRegistrationOTP(
	_ context.Context,
	_ string,
	_ string,
) error {
	return ErrOTPDeliveryUnavailable
}

func NewRegistrationOTPSender(
	appEnv string,
) RegistrationOTPSender {
	if strings.EqualFold(
		strings.TrimSpace(appEnv),
		"production",
	) {
		return &unavailableRegistrationOTPSender{}
	}

	return &developmentRegistrationOTPSender{}
}

type RegistrationOTPService struct {
	client *redis.Client

	sender RegistrationOTPSender

	now func() time.Time
}

func NewRegistrationOTPService(
	client *redis.Client,
	sender RegistrationOTPSender,
) (
	*RegistrationOTPService,
	error,
) {
	if client == nil {
		return nil,
			fmt.Errorf(
				"%w: redis client is required",
				ErrAuthUnavailable,
			)
	}

	if sender == nil {
		return nil,
			fmt.Errorf(
				"%w: OTP sender is required",
				ErrOTPDeliveryUnavailable,
			)
	}

	return &RegistrationOTPService{
		client: client,
		sender: sender,
		now:    time.Now,
	}, nil
}

/*
Start creates a pending registration challenge.

The customer is NOT created here.

Only:
- normalized registration information
- password hash
- OTP hash
- verification timing information

are temporarily stored in Redis.
*/
func (s *RegistrationOTPService) Start(
	ctx context.Context,
	pending pendingRegistration,
) (
	RegistrationChallenge,
	error,
) {
	verificationID, err :=
		generateVerificationID()
	if err != nil {
		return RegistrationChallenge{},
			err
	}

	code, err :=
		generateRegistrationOTP()
	if err != nil {
		return RegistrationChallenge{},
			err
	}

	now :=
		s.now().UTC()

	pending.VerificationID =
		verificationID

	pending.OTPHash =
		hashRegistrationOTP(
			verificationID,
			code,
		)

	pending.ExpiresAt =
		now.Add(
			registrationOTPChallengeTTL,
		)

	pending.ResendAvailableAt =
		now.Add(
			registrationOTPResendCooldown,
		)

	pending.Attempts = 0

	pending.ResendCount = 0

	if err :=
		s.savePending(
			ctx,
			pending,
		); err != nil {

		return RegistrationChallenge{},
			err
	}

	if err :=
		s.sender.SendRegistrationOTP(
			ctx,
			pending.Phone,
			code,
		); err != nil {

		/*
			If delivery fails, remove the challenge.
			The customer should not receive a challenge
			they cannot complete.
		*/
		_ =
			s.client.Del(
				context.Background(),
				registrationOTPKey(
					verificationID,
				),
			).Err()

		if err == ErrOTPDeliveryUnavailable {
			return RegistrationChallenge{},
				err
		}

		return RegistrationChallenge{},
			fmt.Errorf(
				"%w: %v",
				ErrOTPDeliveryUnavailable,
				err,
			)
	}

	return registrationChallenge(
		pending,
	), nil
}

/*
Resend rotates the OTP while keeping the same
verification ID.

The old OTP stops working immediately.
*/
func (s *RegistrationOTPService) Resend(
	ctx context.Context,
	verificationID string,
) (
	RegistrationChallenge,
	error,
) {
	verificationID =
		strings.TrimSpace(
			verificationID,
		)

	if !validVerificationID(
		verificationID,
	) {
		return RegistrationChallenge{},
			ErrInvalidVerification
	}

	var result RegistrationChallenge

	err :=
		s.withLock(
			ctx,
			verificationID,
			func() error {
				pending, err :=
					s.loadPending(
						ctx,
						verificationID,
					)
				if err != nil {
					return err
				}

				now :=
					s.now().UTC()

				if pending.ResendCount >=
					registrationOTPMaxResends {

					return ErrVerificationResendLimit
				}

				if now.Before(
					pending.ResendAvailableAt,
				) {
					return ErrVerificationResendTooSoon
				}

				code, err :=
					generateRegistrationOTP()
				if err != nil {
					return err
				}

				/*
					Keep a copy so we can restore the previous
					state if SMS delivery fails.
				*/
				previous :=
					pending

				pending.OTPHash =
					hashRegistrationOTP(
						verificationID,
						code,
					)

				pending.ExpiresAt =
					now.Add(
						registrationOTPChallengeTTL,
					)

				pending.ResendAvailableAt =
					now.Add(
						registrationOTPResendCooldown,
					)

				pending.ResendCount++

				if err :=
					s.savePending(
						ctx,
						pending,
					); err != nil {

					return err
				}

				if err :=
					s.sender.SendRegistrationOTP(
						ctx,
						pending.Phone,
						code,
					); err != nil {

					/*
						Best effort rollback of the Redis state.
					*/
					_ =
						s.savePending(
							context.Background(),
							previous,
						)

					if err ==
						ErrOTPDeliveryUnavailable {

						return err
					}

					return fmt.Errorf(
						"%w: %v",
						ErrOTPDeliveryUnavailable,
						err,
					)
				}

				result =
					registrationChallenge(
						pending,
					)

				return nil
			},
		)
	if err != nil {
		return RegistrationChallenge{},
			err
	}

	return result, nil
}

/*
VerifyAndConsume checks the OTP and, only after it is valid,
runs the supplied completion function.

The completion function is where the PostgreSQL customer
and auth session are created transactionally.
*/
func (
	s *RegistrationOTPService,
) VerifyAndConsume(
	ctx context.Context,
	verificationID string,
	code string,
	complete func(
		pending pendingRegistration,
	) error,
) error {
	verificationID =
		strings.TrimSpace(
			verificationID,
		)

	if !validVerificationID(
		verificationID,
	) {
		return ErrInvalidVerification
	}

	if complete == nil {
		return ErrInvalidRequest
	}

	return s.withLock(
		ctx,
		verificationID,
		func() error {
			pending, err :=
				s.loadPending(
					ctx,
					verificationID,
				)
			if err != nil {
				return err
			}

			normalizedCode :=
				strings.TrimSpace(
					code,
				)

			validCode :=
				isSixDigitCode(
					normalizedCode,
				)

			matched :=
				false

			if validCode {
				providedHash :=
					hashRegistrationOTP(
						verificationID,
						normalizedCode,
					)

				matched =
					subtle.ConstantTimeCompare(
						[]byte(
							providedHash,
						),
						[]byte(
							pending.OTPHash,
						),
					) == 1
			}

			if !matched {
				pending.Attempts++

				if pending.Attempts >=
					registrationOTPMaxAttempts {

					_ =
						s.client.Del(
							ctx,
							registrationOTPKey(
								verificationID,
							),
						).Err()

					return ErrVerificationAttemptsExceeded
				}

				if err :=
					s.savePending(
						ctx,
						pending,
					); err != nil {

					return err
				}

				return ErrInvalidVerificationCode
			}

			/*
				OTP is correct.

				Now allow Service.VerifyRegistration to
				create the real customer + session.
			*/
			if err :=
				complete(
					pending,
				); err != nil {

				return err
			}

			/*
				The account transaction has already committed.

				Redis cleanup is therefore best-effort.
				We must not report registration failure simply
				because deleting the challenge failed.
			*/
			_ =
				s.client.Del(
					context.Background(),
					registrationOTPKey(
						verificationID,
					),
				).Err()

			return nil
		},
	)
}

func (s *RegistrationOTPService) loadPending(
	ctx context.Context,
	verificationID string,
) (
	pendingRegistration,
	error,
) {
	payload, err :=
		s.client.Get(
			ctx,
			registrationOTPKey(
				verificationID,
			),
		).Bytes()
	if err != nil {
		if err == redis.Nil {
			return pendingRegistration{},
				ErrInvalidVerification
		}

		return pendingRegistration{},
			fmt.Errorf(
				"%w: load registration verification: %v",
				ErrAuthUnavailable,
				err,
			)
	}

	var pending pendingRegistration

	if err :=
		json.Unmarshal(
			payload,
			&pending,
		); err != nil {

		return pendingRegistration{},
			fmt.Errorf(
				"%w: decode registration verification: %v",
				ErrAuthUnavailable,
				err,
			)
	}

	if !pending.ExpiresAt.After(
		s.now().UTC(),
	) {
		return pendingRegistration{},
			ErrVerificationExpired
	}

	return pending, nil
}

func (s *RegistrationOTPService) savePending(
	ctx context.Context,
	pending pendingRegistration,
) error {
	payload, err :=
		json.Marshal(
			pending,
		)
	if err != nil {
		return fmt.Errorf(
			"encode registration verification: %w",
			err,
		)
	}

	/*
		Keep the Redis key slightly longer than the OTP's
		logical expiration.

		That lets us distinguish "expired" from
		"completely unknown verification ID".
	*/
	retainUntil :=
		pending.ExpiresAt.Add(
			registrationOTPExpiredGrace,
		)

	ttl :=
		retainUntil.Sub(
			s.now().UTC(),
		)

	if ttl <= 0 {
		return ErrVerificationExpired
	}

	if err :=
		s.client.Set(
			ctx,
			registrationOTPKey(
				pending.VerificationID,
			),
			payload,
			ttl,
		).Err(); err != nil {

		return fmt.Errorf(
			"%w: store registration verification: %v",
			ErrAuthUnavailable,
			err,
		)
	}

	return nil
}

/*
withLock prevents simultaneous Verify / Resend requests
from mutating the same registration challenge at once.
*/
func (s *RegistrationOTPService) withLock(
	ctx context.Context,
	verificationID string,
	fn func() error,
) error {
	token, err :=
		generateVerificationID()
	if err != nil {
		return err
	}

	key :=
		registrationOTPLockKey(
			verificationID,
		)

	acquired, err :=
		s.client.SetNX(
			ctx,
			key,
			token,
			registrationOTPLockTTL,
		).Result()
	if err != nil {
		return fmt.Errorf(
			"%w: acquire registration verification lock: %v",
			ErrAuthUnavailable,
			err,
		)
	}

	if !acquired {
		return ErrVerificationBusy
	}

	defer func() {
		releaseCtx, cancel :=
			context.WithTimeout(
				context.Background(),
				2*time.Second,
			)

		defer cancel()

		/*
			Delete the lock only if it is still owned by
			this request.
		*/
		const releaseScript = `
			if redis.call("GET", KEYS[1]) == ARGV[1] then
				return redis.call("DEL", KEYS[1])
			end
			return 0
		`

		_ =
			s.client.Eval(
				releaseCtx,
				releaseScript,
				[]string{
					key,
				},
				token,
			).Err()
	}()

	return fn()
}

func registrationChallenge(
	pending pendingRegistration,
) RegistrationChallenge {
	return RegistrationChallenge{
		VerificationID: pending.VerificationID,

		Phone: pending.Phone,

		ExpiresAt: pending.ExpiresAt,

		ResendAvailableAt: pending.ResendAvailableAt,
	}
}

func registrationOTPKey(
	verificationID string,
) string {
	return registrationOTPKeyPrefix +
		verificationID
}

func registrationOTPLockKey(
	verificationID string,
) string {
	return registrationOTPLockPrefix +
		verificationID
}

func validVerificationID(
	value string,
) bool {
	value =
		strings.TrimSpace(
			value,
		)

	/*
		24 random bytes encoded as hexadecimal:
		24 * 2 = 48 characters.
	*/
	if len(value) != 48 {
		return false
	}

	for _, r := range value {

		switch {
		case r >= '0' &&
			r <= '9':

		case r >= 'a' &&
			r <= 'f':

		default:
			return false
		}
	}

	return true
}

func isSixDigitCode(
	value string,
) bool {
	if len(value) != 6 {
		return false
	}

	for _, r := range value {

		if r < '0' ||
			r > '9' {

			return false
		}
	}

	return true
}

/*
The OTP itself is never stored in Redis.

Only this SHA-256 digest is stored.
*/
func hashRegistrationOTP(
	verificationID string,
	code string,
) string {
	sum :=
		sha256.Sum256(
			[]byte(
				verificationID +
					":" +
					code,
			),
		)

	return hex.EncodeToString(
		sum[:],
	)
}

func generateRegistrationOTP() (
	string,
	error,
) {
	max :=
		big.NewInt(
			1_000_000,
		)

	number, err :=
		rand.Int(
			rand.Reader,
			max,
		)
	if err != nil {
		return "",
			fmt.Errorf(
				"generate registration OTP: %w",
				err,
			)
	}

	return fmt.Sprintf(
		"%06d",
		number.Int64(),
	), nil
}

func generateVerificationID() (
	string,
	error,
) {
	buffer :=
		make(
			[]byte,
			24,
		)

	if _, err :=
		rand.Read(
			buffer,
		); err != nil {

		return "",
			fmt.Errorf(
				"generate registration verification ID: %w",
				err,
			)
	}

	return hex.EncodeToString(
		buffer,
	), nil
}
