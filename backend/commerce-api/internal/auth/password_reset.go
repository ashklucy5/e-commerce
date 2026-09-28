package auth

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	passwordResetOTPChallengeTTL = 5 * time.Minute

	passwordResetOTPExpiredGrace = 1 * time.Minute

	passwordResetOTPResendCooldown = 60 * time.Second

	passwordResetOTPMaxAttempts = 5

	passwordResetOTPMaxResends = 5

	passwordResetLockTTL = 20 * time.Second

	passwordResetGrantTTL = 10 * time.Minute
)

const (
	passwordResetChallengePrefix = "auth:customer:password-reset:challenge:"

	passwordResetChallengeLockPrefix = "auth:customer:password-reset:challenge-lock:"

	passwordResetGrantPrefix = "auth:customer:password-reset:grant:"

	passwordResetGrantLockPrefix = "auth:customer:password-reset:grant-lock:"

	passwordResetActiveGrantPrefix = "auth:customer:password-reset:active:"
)

type PasswordResetOTPService struct {
	client *redis.Client

	sender PasswordResetOTPSender

	now func() time.Time
}

func NewPasswordResetOTPService(
	client *redis.Client,
	sender PasswordResetOTPSender,
) (
	*PasswordResetOTPService,
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
				"%w: password reset OTP sender is required",
				ErrOTPDeliveryUnavailable,
			)
	}

	return &PasswordResetOTPService{
		client: client,

		sender: sender,

		now: time.Now,
	}, nil
}

func (s *PasswordResetOTPService) Start(
	ctx context.Context,
	pending pendingPasswordReset,
) (
	PasswordResetChallenge,
	error,
) {
	verificationID, err :=
		generateVerificationID()
	if err != nil {
		return PasswordResetChallenge{},
			err
	}

	code, err :=
		generateRegistrationOTP()
	if err != nil {
		return PasswordResetChallenge{},
			err
	}

	now :=
		s.now().UTC()

	pending.VerificationID =
		verificationID

	pending.OTPHash =
		hashPasswordResetOTP(
			verificationID,
			code,
		)

	pending.ExpiresAt =
		now.Add(
			passwordResetOTPChallengeTTL,
		)

	pending.ResendAvailableAt =
		now.Add(
			passwordResetOTPResendCooldown,
		)

	pending.Attempts = 0

	pending.ResendCount = 0

	if err :=
		s.savePending(
			ctx,
			pending,
		); err != nil {

		return PasswordResetChallenge{},
			err
	}

	/*
		CustomerID == "" represents an intentionally opaque
		non-existent/disabled account challenge.

		We return the same API shape but send no SMS.
	*/
	if pending.CustomerID != "" {
		if err :=
			s.sender.SendPasswordResetOTP(
				ctx,
				pending.Phone,
				code,
			); err != nil {

			_ =
				s.client.Del(
					context.Background(),
					passwordResetChallengeKey(
						verificationID,
					),
				).Err()

			return PasswordResetChallenge{},
				fmt.Errorf(
					"%w: %v",
					ErrOTPDeliveryUnavailable,
					err,
				)
		}
	}

	return passwordResetChallenge(
		pending,
	), nil
}

func (s *PasswordResetOTPService) Resend(
	ctx context.Context,
	verificationID string,
) (
	PasswordResetChallenge,
	error,
) {
	verificationID =
		strings.TrimSpace(
			verificationID,
		)

	if !validVerificationID(
		verificationID,
	) {
		return PasswordResetChallenge{},
			ErrInvalidVerification
	}

	var result PasswordResetChallenge

	err :=
		s.withChallengeLock(
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
					passwordResetOTPMaxResends {

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

				previous :=
					pending

				pending.OTPHash =
					hashPasswordResetOTP(
						verificationID,
						code,
					)

				pending.ExpiresAt =
					now.Add(
						passwordResetOTPChallengeTTL,
					)

				pending.ResendAvailableAt =
					now.Add(
						passwordResetOTPResendCooldown,
					)

				pending.ResendCount++

				if err :=
					s.savePending(
						ctx,
						pending,
					); err != nil {

					return err
				}

				if pending.CustomerID != "" {
					if err :=
						s.sender.SendPasswordResetOTP(
							ctx,
							pending.Phone,
							code,
						); err != nil {

						_ =
							s.savePending(
								context.Background(),
								previous,
							)

						return fmt.Errorf(
							"%w: %v",
							ErrOTPDeliveryUnavailable,
							err,
						)
					}
				}

				result =
					passwordResetChallenge(
						pending,
					)

				return nil
			},
		)
	if err != nil {
		return PasswordResetChallenge{},
			err
	}

	return result, nil
}

func (s *PasswordResetOTPService) Verify(
	ctx context.Context,
	verificationID string,
	code string,
) (
	PasswordResetGrant,
	error,
) {
	verificationID =
		strings.TrimSpace(
			verificationID,
		)

	if !validVerificationID(
		verificationID,
	) {
		return PasswordResetGrant{},
			ErrInvalidVerification
	}

	var result PasswordResetGrant

	err :=
		s.withChallengeLock(
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

				matched :=
					false

				if isSixDigitCode(
					normalizedCode,
				) {
					providedHash :=
						hashPasswordResetOTP(
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

				/*
					A dummy challenge must never produce a grant,
					even if somebody guesses its random OTP.
				*/
				if pending.CustomerID == "" {
					matched =
						false
				}

				if !matched {
					pending.Attempts++

					if pending.Attempts >=
						passwordResetOTPMaxAttempts {

						_ =
							s.client.Del(
								ctx,
								passwordResetChallengeKey(
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

				resetToken, err :=
					generateVerificationID()
				if err != nil {
					return err
				}

				now :=
					s.now().UTC()

				expiresAt :=
					now.Add(
						passwordResetGrantTTL,
					)

				tokenHash :=
					hashToken(
						resetToken,
					)

				record :=
					passwordResetGrantRecord{
						CustomerID: pending.CustomerID,

						ExpiresAt: expiresAt,
					}

				payload, err :=
					json.Marshal(
						record,
					)
				if err != nil {
					return fmt.Errorf(
						"encode password reset grant: %w",
						err,
					)
				}

				pipe :=
					s.client.TxPipeline()

				pipe.Set(
					ctx,
					passwordResetGrantKey(
						tokenHash,
					),
					payload,
					passwordResetGrantTTL,
				)

				/*
					Only the most recently verified reset grant
					for this customer is accepted.
				*/
				pipe.Set(
					ctx,
					passwordResetActiveGrantKey(
						pending.CustomerID,
					),
					tokenHash,
					passwordResetGrantTTL,
				)

				pipe.Del(
					ctx,
					passwordResetChallengeKey(
						verificationID,
					),
				)

				if _, err :=
					pipe.Exec(
						ctx,
					); err != nil {

					return fmt.Errorf(
						"%w: create password reset grant: %v",
						ErrAuthUnavailable,
						err,
					)
				}

				result =
					PasswordResetGrant{
						ResetToken: resetToken,

						ExpiresAt: expiresAt,
					}

				return nil
			},
		)
	if err != nil {
		return PasswordResetGrant{},
			err
	}

	return result, nil
}

func (s *PasswordResetOTPService) ConsumeGrant(
	ctx context.Context,
	resetToken string,
	complete func(
		customerID string,
	) error,
) error {
	resetToken =
		strings.TrimSpace(
			resetToken,
		)

	if !validVerificationID(
		resetToken,
	) {
		return ErrInvalidPasswordResetGrant
	}

	if complete == nil {
		return ErrInvalidRequest
	}

	tokenHash :=
		hashToken(
			resetToken,
		)

	return s.withGrantLock(
		ctx,
		tokenHash,
		func() error {
			payload, err :=
				s.client.Get(
					ctx,
					passwordResetGrantKey(
						tokenHash,
					),
				).Bytes()
			if err != nil {
				if err == redis.Nil {
					return ErrInvalidPasswordResetGrant
				}

				return fmt.Errorf(
					"%w: load password reset grant: %v",
					ErrAuthUnavailable,
					err,
				)
			}

			var record passwordResetGrantRecord

			if err :=
				json.Unmarshal(
					payload,
					&record,
				); err != nil {

				return fmt.Errorf(
					"%w: decode password reset grant: %v",
					ErrAuthUnavailable,
					err,
				)
			}

			if !record.ExpiresAt.After(
				s.now().UTC(),
			) {
				return ErrPasswordResetGrantExpired
			}

			activeHash, err :=
				s.client.Get(
					ctx,
					passwordResetActiveGrantKey(
						record.CustomerID,
					),
				).Result()
			if err != nil {
				if err == redis.Nil {
					return ErrInvalidPasswordResetGrant
				}

				return fmt.Errorf(
					"%w: load active password reset grant: %v",
					ErrAuthUnavailable,
					err,
				)
			}

			if subtle.ConstantTimeCompare(
				[]byte(
					activeHash,
				),
				[]byte(
					tokenHash,
				),
			) != 1 {
				return ErrInvalidPasswordResetGrant
			}

			if err :=
				complete(
					record.CustomerID,
				); err != nil {

				return err
			}

			pipe :=
				s.client.TxPipeline()

			pipe.Del(
				context.Background(),
				passwordResetGrantKey(
					tokenHash,
				),
			)

			pipe.Del(
				context.Background(),
				passwordResetActiveGrantKey(
					record.CustomerID,
				),
			)

			_, _ =
				pipe.Exec(
					context.Background(),
				)

			return nil
		},
	)
}

func (s *PasswordResetOTPService) loadPending(
	ctx context.Context,
	verificationID string,
) (
	pendingPasswordReset,
	error,
) {
	payload, err :=
		s.client.Get(
			ctx,
			passwordResetChallengeKey(
				verificationID,
			),
		).Bytes()
	if err != nil {
		if err == redis.Nil {
			return pendingPasswordReset{},
				ErrInvalidVerification
		}

		return pendingPasswordReset{},
			fmt.Errorf(
				"%w: load password reset challenge: %v",
				ErrAuthUnavailable,
				err,
			)
	}

	var pending pendingPasswordReset

	if err :=
		json.Unmarshal(
			payload,
			&pending,
		); err != nil {

		return pendingPasswordReset{},
			fmt.Errorf(
				"%w: decode password reset challenge: %v",
				ErrAuthUnavailable,
				err,
			)
	}

	if !pending.ExpiresAt.After(
		s.now().UTC(),
	) {
		return pendingPasswordReset{},
			ErrVerificationExpired
	}

	return pending, nil
}

func (s *PasswordResetOTPService) savePending(
	ctx context.Context,
	pending pendingPasswordReset,
) error {
	payload, err :=
		json.Marshal(
			pending,
		)
	if err != nil {
		return fmt.Errorf(
			"encode password reset challenge: %w",
			err,
		)
	}

	retainUntil :=
		pending.ExpiresAt.Add(
			passwordResetOTPExpiredGrace,
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
			passwordResetChallengeKey(
				pending.VerificationID,
			),
			payload,
			ttl,
		).Err(); err != nil {

		return fmt.Errorf(
			"%w: store password reset challenge: %v",
			ErrAuthUnavailable,
			err,
		)
	}

	return nil
}

func (s *PasswordResetOTPService) withChallengeLock(
	ctx context.Context,
	verificationID string,
	fn func() error,
) error {
	return s.withRedisLock(
		ctx,
		passwordResetChallengeLockPrefix+
			verificationID,
		fn,
	)
}

func (s *PasswordResetOTPService) withGrantLock(
	ctx context.Context,
	tokenHash string,
	fn func() error,
) error {
	return s.withRedisLock(
		ctx,
		passwordResetGrantLockPrefix+
			tokenHash,
		fn,
	)
}

func (s *PasswordResetOTPService) withRedisLock(
	ctx context.Context,
	key string,
	fn func() error,
) error {
	token, err :=
		generateVerificationID()
	if err != nil {
		return err
	}

	acquired, err :=
		s.client.SetNX(
			ctx,
			key,
			token,
			passwordResetLockTTL,
		).Result()
	if err != nil {
		return fmt.Errorf(
			"%w: acquire password reset lock: %v",
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

func passwordResetChallenge(
	pending pendingPasswordReset,
) PasswordResetChallenge {
	return PasswordResetChallenge{
		VerificationID: pending.VerificationID,

		ExpiresAt: pending.ExpiresAt,

		ResendAvailableAt: pending.ResendAvailableAt,
	}
}

func passwordResetChallengeKey(
	verificationID string,
) string {
	return passwordResetChallengePrefix +
		verificationID
}

func passwordResetGrantKey(
	tokenHash string,
) string {
	return passwordResetGrantPrefix +
		tokenHash
}

func passwordResetActiveGrantKey(
	customerID string,
) string {
	return passwordResetActiveGrantPrefix +
		customerID
}

func hashPasswordResetOTP(
	verificationID string,
	code string,
) string {
	sum :=
		sha256.Sum256(
			[]byte(
				verificationID +
					":password-reset:" +
					code,
			),
		)

	return hex.EncodeToString(
		sum[:],
	)
}
