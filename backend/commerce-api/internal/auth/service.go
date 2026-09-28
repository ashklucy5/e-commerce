package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

type Service struct {
	repository *Repository

	loginLockout *LoginLockoutService

	registrationOTP *RegistrationOTPService

	passwordResetOTP *PasswordResetOTPService

	otpPolicy OTPPolicy

	now func() time.Time
}

func NewService(
	repository *Repository,
) *Service {
	return &Service{
		repository: repository,

		otpPolicy: DefaultOTPPolicy(),

		now: time.Now,
	}
}

func (s *Service) SetLoginLockout(
	lockout *LoginLockoutService,
) {
	s.loginLockout =
		lockout
}

func (s *Service) SetOTPPolicy(
	policy OTPPolicy,
) {
	s.otpPolicy =
		policy
}

func (s *Service) SetRegistrationOTP(
	otp *RegistrationOTPService,
) {
	s.registrationOTP =
		otp

	s.passwordResetOTP =
		nil

	if otp == nil {
		return
	}

	passwordSender, ok :=
		otp.sender.(PasswordResetOTPSender)
	if !ok {
		return
	}

	passwordResetOTP, err :=
		NewPasswordResetOTPService(
			otp.client,
			passwordSender,
		)
	if err != nil {
		return
	}

	s.passwordResetOTP =
		passwordResetOTP
}

func (s *Service) SetPasswordResetOTP(
	otp *PasswordResetOTPService,
) {
	s.passwordResetOTP =
		otp
}

/*
Register supports two modes.

OTP enabled:

	validate
	→ duplicate-phone check
	→ hash password
	→ create Redis OTP challenge
	→ return verification_required=true

OTP disabled + unverified signup allowed:

	validate
	→ duplicate-phone check
	→ hash password
	→ immediately create customer + session
	→ return verification_required=false

OTP disabled + unverified signup NOT allowed:

	registration is unavailable.
*/
func (s *Service) Register(
	ctx context.Context,
	request RegisterRequest,
) (
	RegisterResult,
	error,
) {
	phone, err :=
		normalizePhone(
			request.Phone,
		)
	if err != nil {
		return RegisterResult{},
			err
	}

	email, err :=
		normalizeEmail(
			request.Email,
		)
	if err != nil {
		return RegisterResult{},
			err
	}

	fullName, err :=
		normalizeFullName(
			request.FullName,
		)
	if err != nil {
		return RegisterResult{},
			err
	}

	if err :=
		validatePassword(
			request.Password,
		); err != nil {

		return RegisterResult{},
			err
	}

	registered, err :=
		s.repository.IsPhoneRegistered(
			ctx,
			phone,
		)
	if err != nil {
		return RegisterResult{},
			err
	}

	if registered {
		return RegisterResult{},
			ErrPhoneInUse
	}

	passwordHash, err :=
		hashPassword(
			request.Password,
		)
	if err != nil {
		return RegisterResult{},
			fmt.Errorf(
				"hash password: %w",
				err,
			)
	}

	/*
		Launch bypass.

		No OTP sender is touched when OTP is disabled.
	*/
	if !s.otpPolicy.Enabled {
		if !s.otpPolicy.
			AllowUnverifiedSignup {

			return RegisterResult{},
				ErrUnverifiedSignupDisabled
		}

		result, err :=
			s.createRegistrationSession(
				ctx,
				phone,
				email,
				passwordHash,
				fullName,
			)
		if err != nil {
			return RegisterResult{},
				err
		}

		customer :=
			result.Customer

		tokens :=
			result.Tokens

		return RegisterResult{
			VerificationRequired: false,

			Customer: &customer,

			Tokens: &tokens,
		}, nil
	}

	/*
		Normal production OTP flow.
	*/
	if s.registrationOTP == nil {
		return RegisterResult{},
			ErrOTPDeliveryUnavailable
	}

	challenge, err :=
		s.registrationOTP.Start(
			ctx,
			pendingRegistration{
				Phone: phone,

				Email: email,

				FullName: fullName,

				PasswordHash: passwordHash,
			},
		)
	if err != nil {
		return RegisterResult{},
			err
	}

	return RegisterResult{
		VerificationRequired: true,

		Verification: &challenge,
	}, nil
}

func (s *Service) VerifyRegistration(
	ctx context.Context,
	request RegisterVerifyRequest,
) (
	AuthResult,
	error,
) {
	if !s.otpPolicy.Enabled {
		return AuthResult{},
			ErrOTPDisabled
	}

	if s.registrationOTP == nil {
		return AuthResult{},
			ErrOTPDeliveryUnavailable
	}

	var result AuthResult

	err :=
		s.registrationOTP.
			VerifyAndConsume(
				ctx,
				request.VerificationID,
				request.Code,
				func(
					pending pendingRegistration,
				) error {
					created, err :=
						s.createRegistrationSession(
							ctx,
							pending.Phone,
							pending.Email,
							pending.PasswordHash,
							pending.FullName,
						)
					if err != nil {
						return err
					}

					result =
						created

					return nil
				},
			)
	if err != nil {
		return AuthResult{},
			err
	}

	return result,
		nil
}

func (s *Service) ResendRegistration(
	ctx context.Context,
	request RegisterResendRequest,
) (
	RegistrationChallenge,
	error,
) {
	if !s.otpPolicy.Enabled {
		return RegistrationChallenge{},
			ErrOTPDisabled
	}

	if s.registrationOTP == nil {
		return RegistrationChallenge{},
			ErrOTPDeliveryUnavailable
	}

	return s.registrationOTP.Resend(
		ctx,
		request.VerificationID,
	)
}

/*
createRegistrationSession performs the permanent part of
registration.

It is used by BOTH:

	verified OTP signup

and

	temporary unverified-signup bypass

so those paths cannot drift apart.
*/
func (s *Service) createRegistrationSession(
	ctx context.Context,
	phone string,
	email string,
	passwordHash string,
	fullName string,
) (
	AuthResult,
	error,
) {
	material, err :=
		newTokenMaterial(
			s.now(),
		)
	if err != nil {
		return AuthResult{},
			err
	}

	tx, err :=
		s.repository.Begin(
			ctx,
		)
	if err != nil {
		return AuthResult{},
			err
	}

	defer func() {
		_ =
			tx.Rollback(
				ctx,
			)
	}()

	customer, err :=
		s.repository.
			CreateCustomerTx(
				ctx,
				tx,
				phone,
				email,
				passwordHash,
				fullName,
			)
	if err != nil {
		return AuthResult{},
			err
	}

	if err :=
		s.repository.
			CreateSessionTx(
				ctx,
				tx,
				customer.ID,
				material,
			); err != nil {

		return AuthResult{},
			err
	}

	if err :=
		tx.Commit(
			ctx,
		); err != nil {

		return AuthResult{},
			fmt.Errorf(
				"commit registration: %w",
				err,
			)
	}

	customer.PasswordHash =
		""

	return AuthResult{
		Customer: customer,

		Tokens: publicTokens(
			material,
		),
	}, nil
}

/*
ForgotPassword is intentionally unavailable when the OTP
feature switch is OFF.

We never permit password recovery without proving control
of the customer's phone.
*/
func (s *Service) ForgotPassword(
	ctx context.Context,
	request ForgotPasswordRequest,
) (
	PasswordResetChallenge,
	error,
) {
	if !s.otpPolicy.Enabled {
		return PasswordResetChallenge{},
			ErrOTPDisabled
	}

	phone, err :=
		normalizePhone(
			request.Phone,
		)
	if err != nil {
		return PasswordResetChallenge{},
			err
	}

	if s.passwordResetOTP == nil {
		return PasswordResetChallenge{},
			ErrOTPDeliveryUnavailable
	}

	pending :=
		pendingPasswordReset{
			Phone: phone,
		}

	customer, err :=
		s.repository.GetCustomerByPhone(
			ctx,
			phone,
		)

	switch {
	case err == nil:
		if customer.Status ==
			"active" {

			pending.CustomerID =
				customer.ID
		}

	case errors.Is(
		err,
		ErrInvalidCredentials,
	):
		/*
			Continue with an opaque dummy challenge.

			This prevents account enumeration.
		*/

	default:
		return PasswordResetChallenge{},
			err
	}

	return s.passwordResetOTP.Start(
		ctx,
		pending,
	)
}

func (s *Service) VerifyForgotPassword(
	ctx context.Context,
	request ForgotPasswordVerifyRequest,
) (
	PasswordResetGrant,
	error,
) {
	if !s.otpPolicy.Enabled {
		return PasswordResetGrant{},
			ErrOTPDisabled
	}

	if s.passwordResetOTP == nil {
		return PasswordResetGrant{},
			ErrOTPDeliveryUnavailable
	}

	return s.passwordResetOTP.Verify(
		ctx,
		request.VerificationID,
		request.Code,
	)
}

func (s *Service) ResendForgotPassword(
	ctx context.Context,
	request ForgotPasswordResendRequest,
) (
	PasswordResetChallenge,
	error,
) {
	if !s.otpPolicy.Enabled {
		return PasswordResetChallenge{},
			ErrOTPDisabled
	}

	if s.passwordResetOTP == nil {
		return PasswordResetChallenge{},
			ErrOTPDeliveryUnavailable
	}

	return s.passwordResetOTP.Resend(
		ctx,
		request.VerificationID,
	)
}

func (s *Service) ResetPassword(
	ctx context.Context,
	request ResetPasswordRequest,
) (
	AuthResult,
	error,
) {
	if !s.otpPolicy.Enabled {
		return AuthResult{},
			ErrOTPDisabled
	}

	if s.passwordResetOTP == nil {
		return AuthResult{},
			ErrOTPDeliveryUnavailable
	}

	if err :=
		validatePassword(
			request.NewPassword,
		); err != nil {

		return AuthResult{},
			err
	}

	if request.NewPassword !=
		request.ConfirmPassword {

		return AuthResult{},
			ErrPasswordConfirmation
	}

	revokeOtherSessions :=
		true

	if request.RevokeOtherSessions !=
		nil {

		revokeOtherSessions =
			*request.RevokeOtherSessions
	}

	var result AuthResult

	err :=
		s.passwordResetOTP.
			ConsumeGrant(
				ctx,
				request.ResetToken,
				func(
					customerID string,
				) error {
					tx, err :=
						s.repository.Begin(
							ctx,
						)
					if err != nil {
						return err
					}

					defer func() {
						_ =
							tx.Rollback(
								ctx,
							)
					}()

					customer, err :=
						s.repository.
							GetCustomerByIDTx(
								ctx,
								tx,
								customerID,
							)
					if err != nil {
						return err
					}

					if customer.Status !=
						"active" {

						return ErrCustomerDisabled
					}

					samePassword,
						_,
						err :=
						verifyPassword(
							request.NewPassword,
							customer.PasswordHash,
						)
					if err != nil {
						return fmt.Errorf(
							"compare reset password with current password: %w",
							err,
						)
					}

					if samePassword {
						return ErrPasswordUnchanged
					}

					passwordHash, err :=
						hashPassword(
							request.NewPassword,
						)
					if err != nil {
						return fmt.Errorf(
							"hash reset password: %w",
							err,
						)
					}

					if err :=
						s.repository.
							UpdatePasswordHashTx(
								ctx,
								tx,
								customer.ID,
								passwordHash,
							); err != nil {

						return err
					}

					if revokeOtherSessions {
						if err :=
							s.repository.
								RevokeAllCustomerSessionsTx(
									ctx,
									tx,
									customer.ID,
								); err != nil {

							return err
						}
					}

					material, err :=
						newTokenMaterial(
							s.now(),
						)
					if err != nil {
						return err
					}

					if err :=
						s.repository.
							CreateSessionTx(
								ctx,
								tx,
								customer.ID,
								material,
							); err != nil {

						return err
					}

					if err :=
						tx.Commit(
							ctx,
						); err != nil {

						return fmt.Errorf(
							"commit password reset: %w",
							err,
						)
					}

					customer.PasswordHash =
						""

					result =
						AuthResult{
							Customer: customer,

							Tokens: publicTokens(
								material,
							),
						}

					return nil
				},
			)
	if err != nil {
		return AuthResult{},
			err
	}

	return result,
		nil
}

/*
ChangePassword remains available regardless of the OTP
feature switch.

The customer is already authenticated and must prove the
current password.
*/
func (s *Service) ChangePassword(
	ctx context.Context,
	accessToken string,
	request ChangePasswordRequest,
) error {
	accessToken =
		strings.TrimSpace(
			accessToken,
		)

	if accessToken == "" {
		return ErrInvalidAccessToken
	}

	if request.CurrentPassword ==
		"" {

		return ErrCurrentPasswordIncorrect
	}

	if err :=
		validatePassword(
			request.NewPassword,
		); err != nil {

		return err
	}

	if request.NewPassword !=
		request.ConfirmPassword {

		return ErrPasswordConfirmation
	}

	accessTokenHash :=
		hashToken(
			accessToken,
		)

	customer, err :=
		s.repository.
			GetCustomerByAccessTokenHash(
				ctx,
				accessTokenHash,
			)
	if err != nil {
		return err
	}

	valid,
		_,
		err :=
		verifyPassword(
			request.CurrentPassword,
			customer.PasswordHash,
		)
	if err != nil {
		return fmt.Errorf(
			"verify current password: %w",
			err,
		)
	}

	if !valid {
		return ErrCurrentPasswordIncorrect
	}

	samePassword,
		_,
		err :=
		verifyPassword(
			request.NewPassword,
			customer.PasswordHash,
		)
	if err != nil {
		return fmt.Errorf(
			"compare new password with current password: %w",
			err,
		)
	}

	if samePassword {
		return ErrPasswordUnchanged
	}

	passwordHash, err :=
		hashPassword(
			request.NewPassword,
		)
	if err != nil {
		return fmt.Errorf(
			"hash new password: %w",
			err,
		)
	}

	revokeOtherSessions :=
		true

	if request.RevokeOtherSessions !=
		nil {

		revokeOtherSessions =
			*request.RevokeOtherSessions
	}

	tx, err :=
		s.repository.Begin(
			ctx,
		)
	if err != nil {
		return err
	}

	defer func() {
		_ =
			tx.Rollback(
				ctx,
			)
	}()

	if err :=
		s.repository.
			UpdatePasswordHashTx(
				ctx,
				tx,
				customer.ID,
				passwordHash,
			); err != nil {

		return err
	}

	if revokeOtherSessions {
		if err :=
			s.repository.
				RevokeOtherCustomerSessionsTx(
					ctx,
					tx,
					customer.ID,
					accessTokenHash,
				); err != nil {

			return err
		}
	}

	if err :=
		tx.Commit(
			ctx,
		); err != nil {

		return fmt.Errorf(
			"commit password change: %w",
			err,
		)
	}

	return nil
}

func (s *Service) Login(
	ctx context.Context,
	request LoginRequest,
	ipAddress string,
) (
	AuthResult,
	error,
) {
	rawIdentifier :=
		strings.TrimSpace(
			request.Phone,
		)

	if s.loginLockout != nil {
		blocked, err :=
			s.loginLockout.IsBlocked(
				ctx,
				rawIdentifier,
				ipAddress,
			)
		if err != nil {
			return AuthResult{},
				fmt.Errorf(
					"%w: %v",
					ErrAuthUnavailable,
					err,
				)
		}

		if blocked {
			return AuthResult{},
				ErrLoginBlocked
		}
	}

	phone, err :=
		normalizePhone(
			request.Phone,
		)
	if err != nil {
		return AuthResult{},
			s.failLogin(
				ctx,
				rawIdentifier,
				ipAddress,
			)
	}

	if request.Password == "" {
		return AuthResult{},
			s.failLogin(
				ctx,
				phone,
				ipAddress,
			)
	}

	customer, err :=
		s.repository.GetCustomerByPhone(
			ctx,
			phone,
		)
	if err != nil {
		if errors.Is(
			err,
			ErrInvalidCredentials,
		) {
			return AuthResult{},
				s.failLogin(
					ctx,
					phone,
					ipAddress,
				)
		}

		return AuthResult{},
			err
	}

	valid,
		replacementPasswordHash,
		err :=
		verifyPassword(
			request.Password,
			customer.PasswordHash,
		)
	if err != nil {
		return AuthResult{},
			fmt.Errorf(
				"verify password: %w",
				err,
			)
	}

	if !valid {
		return AuthResult{},
			s.failLogin(
				ctx,
				phone,
				ipAddress,
			)
	}

	if customer.Status !=
		"active" {

		return AuthResult{},
			ErrCustomerDisabled
	}

	if s.loginLockout != nil {
		if err :=
			s.loginLockout.RecordSuccess(
				ctx,
				phone,
			); err != nil {

			return AuthResult{},
				fmt.Errorf(
					"%w: %v",
					ErrAuthUnavailable,
					err,
				)
		}
	}

	material, err :=
		newTokenMaterial(
			s.now(),
		)
	if err != nil {
		return AuthResult{},
			err
	}

	tx, err :=
		s.repository.Begin(
			ctx,
		)
	if err != nil {
		return AuthResult{},
			err
	}

	defer func() {
		_ =
			tx.Rollback(
				ctx,
			)
	}()

	if replacementPasswordHash != "" {
		if err :=
			s.repository.
				UpdatePasswordHashTx(
					ctx,
					tx,
					customer.ID,
					replacementPasswordHash,
				); err != nil {

			return AuthResult{},
				err
		}
	}

	if err :=
		s.repository.
			CreateSessionTx(
				ctx,
				tx,
				customer.ID,
				material,
			); err != nil {

		return AuthResult{},
			err
	}

	if err :=
		tx.Commit(
			ctx,
		); err != nil {

		return AuthResult{},
			fmt.Errorf(
				"commit login: %w",
				err,
			)
	}

	customer.PasswordHash =
		""

	return AuthResult{
		Customer: customer,

		Tokens: publicTokens(
			material,
		),
	}, nil
}

func (s *Service) Refresh(
	ctx context.Context,
	request RefreshRequest,
) (
	AuthResult,
	error,
) {
	refreshToken :=
		strings.TrimSpace(
			request.RefreshToken,
		)

	if refreshToken == "" {
		return AuthResult{},
			ErrInvalidRefreshToken
	}

	refreshTokenHash :=
		hashToken(
			refreshToken,
		)

	tx, err :=
		s.repository.Begin(
			ctx,
		)
	if err != nil {
		return AuthResult{},
			err
	}

	defer func() {
		_ =
			tx.Rollback(
				ctx,
			)
	}()

	session, err :=
		s.repository.
			LockSessionByRefreshTokenTx(
				ctx,
				tx,
				refreshTokenHash,
			)
	if err != nil {
		return AuthResult{},
			err
	}

	now :=
		s.now()

	if !session.RefreshExpiresAt.After(
		now,
	) {
		return AuthResult{},
			ErrInvalidRefreshToken
	}

	if session.Customer.Status !=
		"active" {

		return AuthResult{},
			ErrCustomerDisabled
	}

	material, err :=
		newTokenMaterial(
			now,
		)
	if err != nil {
		return AuthResult{},
			err
	}

	if err :=
		s.repository.
			RotateSessionTx(
				ctx,
				tx,
				session.ID,
				material,
			); err != nil {

		return AuthResult{},
			err
	}

	if err :=
		tx.Commit(
			ctx,
		); err != nil {

		return AuthResult{},
			fmt.Errorf(
				"commit token refresh: %w",
				err,
			)
	}

	customer :=
		session.Customer

	customer.PasswordHash =
		""

	return AuthResult{
		Customer: customer,

		Tokens: publicTokens(
			material,
		),
	}, nil
}

func (s *Service) Logout(
	ctx context.Context,
	accessToken string,
) error {
	accessToken =
		strings.TrimSpace(
			accessToken,
		)

	if accessToken == "" {
		return ErrInvalidAccessToken
	}

	return s.repository.
		RevokeSessionByAccessTokenHash(
			ctx,
			hashToken(
				accessToken,
			),
		)
}

func (s *Service) AuthenticateAccessToken(
	ctx context.Context,
	accessToken string,
) (
	Customer,
	error,
) {
	accessToken =
		strings.TrimSpace(
			accessToken,
		)

	if accessToken == "" {
		return Customer{},
			ErrInvalidAccessToken
	}

	customer, err :=
		s.repository.
			GetCustomerByAccessTokenHash(
				ctx,
				hashToken(
					accessToken,
				),
			)
	if err != nil {
		return Customer{},
			err
	}

	if customer.Status !=
		"active" {

		return Customer{},
			ErrCustomerDisabled
	}

	customer.PasswordHash =
		""

	return customer,
		nil
}

func (s *Service) failLogin(
	ctx context.Context,
	identifier string,
	ipAddress string,
) error {
	if s.loginLockout == nil {
		return ErrInvalidCredentials
	}

	if err :=
		s.loginLockout.RecordFailure(
			ctx,
			identifier,
			ipAddress,
		); err != nil {

		return fmt.Errorf(
			"%w: %v",
			ErrAuthUnavailable,
			err,
		)
	}

	return ErrInvalidCredentials
}
