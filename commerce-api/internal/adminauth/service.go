package adminauth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	platformdatabase "project.local/commerce-api/internal/platform/database"
	platformsecurity "project.local/commerce-api/internal/platform/security"
)

const (
	AdminLoginChallengeTTL = 5 * time.Minute

	AdminChallengeMaxAttempts = 5
)

type Service struct {
	repository *Repository

	mfa *MFAService

	lockout *LockoutService

	dummyPasswordHash string

	now func() time.Time
}

func NewService(
	repository *Repository,
	mfa *MFAService,
	lockout *LockoutService,
) (
	*Service,
	error,
) {
	if repository == nil {
		return nil,
			fmt.Errorf(
				"Admin auth repository is required",
			)
	}

	if mfa == nil {
		return nil,
			fmt.Errorf(
				"Admin MFA service is required",
			)
	}

	if lockout == nil {
		return nil,
			fmt.Errorf(
				"Admin lockout service is required",
			)
	}

	dummyPasswordHash, err :=
		platformsecurity.HashPassword(
			"admin-auth-dummy-password-material",
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"create Admin authentication timing hash: %w",
				err,
			)
	}

	return &Service{
			repository: repository,

			mfa: mfa,

			lockout: lockout,

			dummyPasswordHash: dummyPasswordHash,

			now: func() time.Time {
				return time.Now().
					UTC()
			},
		},
		nil
}

func (s *Service) Login(
	ctx context.Context,
	request LoginRequest,
	metadata ClientMetadata,
) (
	LoginChallengeResponse,
	error,
) {
	identifier :=
		strings.TrimSpace(
			request.Identifier,
		)

	password :=
		request.Password

	if identifier == "" ||
		password == "" {

		return LoginChallengeResponse{},
			ErrInvalidCredentials
	}

	blocked, err :=
		s.lockout.IsBlocked(
			ctx,
			identifier,
			metadata.IPAddress,
		)
	if err != nil {
		return LoginChallengeResponse{},
			fmt.Errorf(
				"%w: %v",
				ErrAuthUnavailable,
				err,
			)
	}

	if blocked {
		_ =
			s.repository.InsertSecurityEvent(
				ctx,
				newSecurityEvent(
					SecurityEventLoginBlocked,
					SecurityOutcomeBlocked,
					nil,
					nil,
					identifier,
					metadata,
					nil,
					s.now(),
				),
			)

		return LoginChallengeResponse{},
			ErrLoginBlocked
	}

	account, err :=
		s.repository.FindLoginAccount(
			ctx,
			identifier,
		)
	if err != nil {
		if errors.Is(
			err,
			pgx.ErrNoRows,
		) {
			_, _, _ =
				verifyStoredPassword(
					password,
					s.dummyPasswordHash,
				)

			return LoginChallengeResponse{},
				s.failPasswordLogin(
					ctx,
					identifier,
					metadata,
					nil,
				)
		}

		return LoginChallengeResponse{},
			err
	}

	staffID :=
		account.ID

	valid,
		replacementHash,
		err :=
		verifyStoredPassword(
			password,
			account.PasswordHash,
		)
	if err != nil {
		return LoginChallengeResponse{},
			err
	}

	if account.Status != "active" ||
		!account.HasPanelAccess ||
		!valid {

		return LoginChallengeResponse{},
			s.failPasswordLogin(
				ctx,
				identifier,
				metadata,
				&staffID,
			)
	}

	if replacementHash != "" {
		if err :=
			s.repository.UpdatePasswordHash(
				ctx,
				account.ID,
				replacementHash,
			); err != nil {

			return LoginChallengeResponse{},
				err
		}
	}

	mfaBlocked, err :=
		s.lockout.IsMFABlocked(
			ctx,
			account.ID,
			metadata.IPAddress,
		)
	if err != nil {
		return LoginChallengeResponse{},
			fmt.Errorf(
				"%w: %v",
				ErrAuthUnavailable,
				err,
			)
	}

	if mfaBlocked {
		_ =
			s.repository.InsertSecurityEvent(
				ctx,
				newSecurityEvent(
					SecurityEventMFABlocked,
					SecurityOutcomeBlocked,
					&staffID,
					nil,
					identifier,
					metadata,
					nil,
					s.now(),
				),
			)

		return LoginChallengeResponse{},
			ErrMFABlocked
	}

	if err :=
		s.lockout.RecordSuccess(
			ctx,
			identifier,
		); err != nil {

		return LoginChallengeResponse{},
			fmt.Errorf(
				"%w: %v",
				ErrAuthUnavailable,
				err,
			)
	}

	now :=
		s.now().
			UTC()

	challengeToken, err :=
		platformsecurity.RandomURLSafe(
			32,
		)
	if err != nil {
		return LoginChallengeResponse{},
			fmt.Errorf(
				"generate Admin login challenge: %w",
				err,
			)
	}

	challengeHash :=
		platformsecurity.HashToken(
			challengeToken,
		)

	expiresAt :=
		now.Add(
			AdminLoginChallengeTTL,
		)

	var challenge LoginChallenge

	err =
		platformdatabase.WithinTx(
			ctx,
			s.repository.db,
			func(
				ctx context.Context,
				tx pgx.Tx,
			) error {

				if err :=
					s.repository.
						CancelPendingChallengesTx(
							ctx,
							tx,
							account.ID,
							ChallengePurposeLogin,
						); err != nil {

					return err
				}

				created, err :=
					s.repository.
						CreateLoginChallengeTx(
							ctx,
							tx,
							account.ID,
							challengeHash,
							ChallengePurposeLogin,
							expiresAt,
							AdminChallengeMaxAttempts,
							metadata,
						)
				if err != nil {
					return err
				}

				challenge =
					created

				return s.repository.
					InsertSecurityEventTx(
						ctx,
						tx,
						newSecurityEvent(
							SecurityEventPasswordLoginSucceeded,
							SecurityOutcomeSuccess,
							&staffID,
							nil,
							identifier,
							metadata,
							map[string]any{
								"challenge_id": challenge.ID,

								"mfa_enrollment_required": !account.HasActiveMFA,
							},
							now,
						),
					)
			},
		)
	if err != nil {
		return LoginChallengeResponse{},
			err
	}

	return LoginChallengeResponse{
			ChallengeToken: challengeToken,

			ChallengeExpiresAt: challenge.ExpiresAt,

			MFAEnrollmentRequired: !account.HasActiveMFA,
		},
		nil
}

func (s *Service) BeginMFAEnrollment(
	ctx context.Context,
	request MFAEnrollRequest,
	metadata ClientMetadata,
) (
	MFAEnrollResponse,
	error,
) {
	challenge, err :=
		s.validateChallenge(
			ctx,
			request.ChallengeToken,
		)
	if err != nil {
		return MFAEnrollResponse{},
			err
	}

	blocked, err :=
		s.lockout.IsMFABlocked(
			ctx,
			challenge.StaffAccountID,
			metadata.IPAddress,
		)
	if err != nil {
		return MFAEnrollResponse{},
			fmt.Errorf(
				"%w: %v",
				ErrAuthUnavailable,
				err,
			)
	}

	if blocked {
		staffID :=
			challenge.StaffAccountID

		_ =
			s.repository.InsertSecurityEvent(
				ctx,
				newSecurityEvent(
					SecurityEventMFABlocked,
					SecurityOutcomeBlocked,
					&staffID,
					nil,
					"",
					metadata,
					map[string]any{
						"challenge_id": challenge.ID,

						"stage": "enrollment",
					},
					s.now(),
				),
			)

		return MFAEnrollResponse{},
			ErrMFABlocked
	}

	account, err :=
		s.mfa.requireAdminAccount(
			ctx,
			challenge.StaffAccountID,
		)
	if err != nil {
		return MFAEnrollResponse{},
			err
	}

	enrollment, err :=
		s.mfa.BeginEnrollment(
			ctx,
			account.ID,
			request.Label,
		)
	if err != nil {
		return MFAEnrollResponse{},
			err
	}

	staffID :=
		account.ID

	_ =
		s.repository.InsertSecurityEvent(
			ctx,
			newSecurityEvent(
				SecurityEventMFAEnrollmentStarted,
				SecurityOutcomeSuccess,
				&staffID,
				nil,
				"",
				metadata,
				map[string]any{
					"challenge_id": challenge.ID,

					"credential_id": enrollment.CredentialID,
				},
				s.now(),
			),
		)

	return MFAEnrollResponse{
			Enrollment: enrollment,
		},
		nil
}

func (s *Service) ConfirmMFAEnrollment(
	ctx context.Context,
	request MFAConfirmEnrollmentRequest,
	metadata ClientMetadata,
) (
	EnrollmentSessionResult,
	error,
) {
	challengeToken :=
		strings.TrimSpace(
			request.ChallengeToken,
		)

	code :=
		strings.TrimSpace(
			request.Code,
		)

	if challengeToken == "" ||
		code == "" {

		return EnrollmentSessionResult{},
			ErrInvalidRequest
	}

	preChallenge, err :=
		s.validateChallenge(
			ctx,
			challengeToken,
		)
	if err != nil {
		return EnrollmentSessionResult{},
			err
	}

	blocked, err :=
		s.lockout.IsMFABlocked(
			ctx,
			preChallenge.StaffAccountID,
			metadata.IPAddress,
		)
	if err != nil {
		return EnrollmentSessionResult{},
			fmt.Errorf(
				"%w: %v",
				ErrAuthUnavailable,
				err,
			)
	}

	if blocked {
		staffID :=
			preChallenge.StaffAccountID

		_ =
			s.repository.InsertSecurityEvent(
				ctx,
				newSecurityEvent(
					SecurityEventMFABlocked,
					SecurityOutcomeBlocked,
					&staffID,
					nil,
					"",
					metadata,
					map[string]any{
						"challenge_id": preChallenge.ID,

						"stage": "enrollment_confirmation",
					},
					s.now(),
				),
			)

		return EnrollmentSessionResult{},
			ErrMFABlocked
	}

	now :=
		s.now().
			UTC()

	var result EnrollmentSessionResult

	var publicErr error

	mfaFailed :=
		false

	err =
		platformdatabase.WithinTx(
			ctx,
			s.repository.db,
			func(
				ctx context.Context,
				tx pgx.Tx,
			) error {

				challenge, err :=
					s.repository.
						LockChallengeByTokenHashTx(
							ctx,
							tx,
							platformsecurity.HashToken(
								challengeToken,
							),
						)
				if err != nil {
					if errors.Is(
						err,
						pgx.ErrNoRows,
					) {
						publicErr =
							ErrInvalidChallenge

						return nil
					}

					return err
				}

				if err :=
					s.validateLockedChallengeTx(
						ctx,
						tx,
						challenge,
						now,
					); err != nil {

					publicErr =
						err

					return nil
				}

				account, err :=
					s.repository.GetMFAAccount(
						ctx,
						challenge.StaffAccountID,
					)
				if err != nil {
					return err
				}

				if account.Status != "active" ||
					!account.HasPanelAccess {

					publicErr =
						ErrInvalidCredentials

					return nil
				}

				credential, err :=
					s.repository.
						LockTOTPCredentialTx(
							ctx,
							tx,
							challenge.StaffAccountID,
						)
				if err != nil {
					if errors.Is(
						err,
						pgx.ErrNoRows,
					) {
						publicErr =
							ErrMFANotEnrolled

						return nil
					}

					return err
				}

				if credential.Status !=
					TOTPCredentialStatusPending {

					publicErr =
						ErrMFAEnrollmentNotPending

					return nil
				}

				secret, err :=
					s.mfa.decryptCredential(
						credential,
					)
				if err != nil {
					return err
				}

				verification, err :=
					VerifyTOTP(
						secret,
						code,
						now,
						credential.LastAcceptedStep,
						totpConfigFromCredential(
							credential,
						),
					)
				if err != nil &&
					!errors.Is(
						err,
						ErrInvalidTOTPCode,
					) {

					return err
				}

				if err != nil ||
					!verification.Valid {

					mfaFailed =
						true

					status, updateErr :=
						s.repository.
							IncrementChallengeFailureTx(
								ctx,
								tx,
								challenge,
							)
					if updateErr != nil {
						return updateErr
					}

					staffID :=
						challenge.StaffAccountID

					if eventErr :=
						s.repository.
							InsertSecurityEventTx(
								ctx,
								tx,
								newSecurityEvent(
									SecurityEventMFAFailed,
									SecurityOutcomeFailure,
									&staffID,
									nil,
									"",
									metadata,
									map[string]any{
										"challenge_id": challenge.ID,

										"stage": "enrollment",
									},
									now,
								),
							); eventErr != nil {

						return eventErr
					}

					if status ==
						ChallengeStatusLocked {

						publicErr =
							ErrChallengeLocked
					} else {
						publicErr =
							ErrInvalidMFACode
					}

					return nil
				}

				recoveryCodes,
					hashes,
					err :=
					generateRecoveryCodesAndHashes()
				if err != nil {
					return err
				}

				if err :=
					s.repository.
						ReplaceRecoveryCodesTx(
							ctx,
							tx,
							challenge.StaffAccountID,
							hashes,
						); err != nil {

					return err
				}

				if err :=
					s.repository.
						ActivateTOTPCredentialTx(
							ctx,
							tx,
							challenge.StaffAccountID,
							verification.Step,
							now,
						); err != nil {

					return err
				}

				material, err :=
					newAdminSessionMaterial(
						now,
						nil,
					)
				if err != nil {
					return err
				}

				if err :=
					s.repository.
						ConsumeChallengeTx(
							ctx,
							tx,
							challenge.ID,
							now,
						); err != nil {

					return err
				}

				sessionID, err :=
					s.repository.
						CreateAdminSessionTx(
							ctx,
							tx,
							challenge.StaffAccountID,
							challenge.ID,
							material,
							now,
							now,
							metadata,
						)
				if err != nil {
					return err
				}

				if err :=
					s.repository.
						UpdateLastLoginTx(
							ctx,
							tx,
							challenge.StaffAccountID,
							now,
						); err != nil {

					return err
				}

				staffID :=
					challenge.StaffAccountID

				if err :=
					s.repository.
						InsertSecurityEventTx(
							ctx,
							tx,
							newSecurityEvent(
								SecurityEventMFAEnrollmentCompleted,
								SecurityOutcomeSuccess,
								&staffID,
								&sessionID,
								"",
								metadata,
								map[string]any{
									"challenge_id": challenge.ID,

									"credential_id": credential.ID,
								},
								now,
							),
						); err != nil {

					return err
				}

				if err :=
					s.repository.
						InsertSecurityEventTx(
							ctx,
							tx,
							newSecurityEvent(
								SecurityEventSessionCreated,
								SecurityOutcomeSuccess,
								&staffID,
								&sessionID,
								"",
								metadata,
								nil,
								now,
							),
						); err != nil {

					return err
				}

				result =
					EnrollmentSessionResult{
						Session: SessionResult{
							Principal: AdminPrincipal{
								SessionID: sessionID,

								AuthenticatedAt: now,

								MFAVerifiedAt: now,

								AccessExpiresAt: material.AccessExpiresAt,
							},

							Material: material,
						},

						RecoveryCodes: recoveryCodes,
					}

				return nil
			},
		)
	if err != nil {
		return EnrollmentSessionResult{},
			err
	}

	if mfaFailed {
		_ =
			s.lockout.RecordMFAFailure(
				ctx,
				preChallenge.StaffAccountID,
				metadata.IPAddress,
			)
	}

	if publicErr != nil {
		return EnrollmentSessionResult{},
			publicErr
	}

	_ =
		s.lockout.RecordMFASuccess(
			ctx,
			preChallenge.StaffAccountID,
		)

	principal,
		_,
		_,
		err :=
		s.repository.GetPrincipalByAccessTokenHash(
			ctx,
			result.Session.Material.AccessTokenHash,
		)
	if err != nil {
		return EnrollmentSessionResult{},
			err
	}

	result.Session.Principal =
		principal

	return result,
		nil
}

func (s *Service) VerifyMFA(
	ctx context.Context,
	request MFAVerifyRequest,
	metadata ClientMetadata,
) (
	SessionResult,
	error,
) {
	challengeToken :=
		strings.TrimSpace(
			request.ChallengeToken,
		)

	method :=
		strings.ToLower(
			strings.TrimSpace(
				request.Method,
			),
		)

	code :=
		strings.TrimSpace(
			request.Code,
		)

	if challengeToken == "" ||
		code == "" {

		return SessionResult{},
			ErrInvalidRequest
	}

	if method == "" {
		method =
			MFAMethodTOTP
	}

	if method != MFAMethodTOTP &&
		method != MFAMethodRecoveryCode {

		return SessionResult{},
			ErrInvalidRequest
	}

	preChallenge, err :=
		s.validateChallenge(
			ctx,
			challengeToken,
		)
	if err != nil {
		return SessionResult{},
			err
	}

	blocked, err :=
		s.lockout.IsMFABlocked(
			ctx,
			preChallenge.StaffAccountID,
			metadata.IPAddress,
		)
	if err != nil {
		return SessionResult{},
			fmt.Errorf(
				"%w: %v",
				ErrAuthUnavailable,
				err,
			)
	}

	if blocked {
		staffID :=
			preChallenge.StaffAccountID

		_ =
			s.repository.InsertSecurityEvent(
				ctx,
				newSecurityEvent(
					SecurityEventMFABlocked,
					SecurityOutcomeBlocked,
					&staffID,
					nil,
					"",
					metadata,
					map[string]any{
						"challenge_id": preChallenge.ID,

						"method": method,
					},
					s.now(),
				),
			)

		return SessionResult{},
			ErrMFABlocked
	}

	now :=
		s.now().
			UTC()

	var result SessionResult

	var publicErr error

	mfaFailed :=
		false

	err =
		platformdatabase.WithinTx(
			ctx,
			s.repository.db,
			func(
				ctx context.Context,
				tx pgx.Tx,
			) error {

				challenge, err :=
					s.repository.
						LockChallengeByTokenHashTx(
							ctx,
							tx,
							platformsecurity.HashToken(
								challengeToken,
							),
						)
				if err != nil {
					if errors.Is(
						err,
						pgx.ErrNoRows,
					) {
						publicErr =
							ErrInvalidChallenge

						return nil
					}

					return err
				}

				if err :=
					s.validateLockedChallengeTx(
						ctx,
						tx,
						challenge,
						now,
					); err != nil {

					publicErr =
						err

					return nil
				}

				account, err :=
					s.repository.GetMFAAccount(
						ctx,
						challenge.StaffAccountID,
					)
				if err != nil {
					return err
				}

				if account.Status != "active" ||
					!account.HasPanelAccess {

					publicErr =
						ErrInvalidCredentials

					return nil
				}

				credential, err :=
					s.repository.
						LockTOTPCredentialTx(
							ctx,
							tx,
							challenge.StaffAccountID,
						)
				if err != nil {
					if errors.Is(
						err,
						pgx.ErrNoRows,
					) {
						publicErr =
							ErrMFANotEnrolled

						return nil
					}

					return err
				}

				if credential.Status !=
					TOTPCredentialStatusActive {

					publicErr =
						ErrMFANotEnrolled

					return nil
				}

				valid :=
					false

				usedRecoveryCode :=
					false

				switch method {
				case MFAMethodTOTP:
					secret, err :=
						s.mfa.decryptCredential(
							credential,
						)
					if err != nil {
						return err
					}

					verification,
						verifyErr :=
						VerifyTOTP(
							secret,
							code,
							now,
							credential.LastAcceptedStep,
							totpConfigFromCredential(
								credential,
							),
						)

					if verifyErr != nil &&
						!errors.Is(
							verifyErr,
							ErrInvalidTOTPCode,
						) {

						return verifyErr
					}

					if verifyErr == nil &&
						verification.Valid {

						if err :=
							s.repository.
								UpdateAcceptedTOTPStepTx(
									ctx,
									tx,
									challenge.StaffAccountID,
									verification.Step,
									now,
								); err != nil {

							return err
						}

						valid =
							true
					}

				case MFAMethodRecoveryCode:
					hash, err :=
						HashRecoveryCode(
							code,
						)

					if err == nil {
						consumed,
							consumeErr :=
							s.repository.
								ConsumeRecoveryCodeTx(
									ctx,
									tx,
									challenge.StaffAccountID,
									hash,
									now,
								)
						if consumeErr != nil {
							return consumeErr
						}

						valid =
							consumed

						usedRecoveryCode =
							consumed
					}
				}

				if !valid {
					mfaFailed =
						true

					status, updateErr :=
						s.repository.
							IncrementChallengeFailureTx(
								ctx,
								tx,
								challenge,
							)
					if updateErr != nil {
						return updateErr
					}

					staffID :=
						challenge.StaffAccountID

					if eventErr :=
						s.repository.
							InsertSecurityEventTx(
								ctx,
								tx,
								newSecurityEvent(
									SecurityEventMFAFailed,
									SecurityOutcomeFailure,
									&staffID,
									nil,
									"",
									metadata,
									map[string]any{
										"challenge_id": challenge.ID,

										"method": method,
									},
									now,
								),
							); eventErr != nil {

						return eventErr
					}

					if status ==
						ChallengeStatusLocked {

						publicErr =
							ErrChallengeLocked
					} else {
						publicErr =
							ErrInvalidMFACode
					}

					return nil
				}

				material, err :=
					newAdminSessionMaterial(
						now,
						nil,
					)
				if err != nil {
					return err
				}

				if err :=
					s.repository.
						ConsumeChallengeTx(
							ctx,
							tx,
							challenge.ID,
							now,
						); err != nil {

					return err
				}

				sessionID, err :=
					s.repository.
						CreateAdminSessionTx(
							ctx,
							tx,
							challenge.StaffAccountID,
							challenge.ID,
							material,
							now,
							now,
							metadata,
						)
				if err != nil {
					return err
				}

				if err :=
					s.repository.
						UpdateLastLoginTx(
							ctx,
							tx,
							challenge.StaffAccountID,
							now,
						); err != nil {

					return err
				}

				staffID :=
					challenge.StaffAccountID

				if usedRecoveryCode {
					if err :=
						s.repository.
							InsertSecurityEventTx(
								ctx,
								tx,
								newSecurityEvent(
									SecurityEventRecoveryCodeUsed,
									SecurityOutcomeSuccess,
									&staffID,
									&sessionID,
									"",
									metadata,
									map[string]any{
										"challenge_id": challenge.ID,
									},
									now,
								),
							); err != nil {

						return err
					}
				}

				if err :=
					s.repository.
						InsertSecurityEventTx(
							ctx,
							tx,
							newSecurityEvent(
								SecurityEventMFAVerified,
								SecurityOutcomeSuccess,
								&staffID,
								&sessionID,
								"",
								metadata,
								map[string]any{
									"challenge_id": challenge.ID,

									"method": method,
								},
								now,
							),
						); err != nil {

					return err
				}

				if err :=
					s.repository.
						InsertSecurityEventTx(
							ctx,
							tx,
							newSecurityEvent(
								SecurityEventSessionCreated,
								SecurityOutcomeSuccess,
								&staffID,
								&sessionID,
								"",
								metadata,
								nil,
								now,
							),
						); err != nil {

					return err
				}

				result =
					SessionResult{
						Principal: AdminPrincipal{
							SessionID: sessionID,

							AuthenticatedAt: now,

							MFAVerifiedAt: now,

							AccessExpiresAt: material.AccessExpiresAt,
						},

						Material: material,
					}

				return nil
			},
		)
	if err != nil {
		return SessionResult{},
			err
	}

	if mfaFailed {
		_ =
			s.lockout.RecordMFAFailure(
				ctx,
				preChallenge.StaffAccountID,
				metadata.IPAddress,
			)
	}

	if publicErr != nil {
		return SessionResult{},
			publicErr
	}

	_ =
		s.lockout.RecordMFASuccess(
			ctx,
			preChallenge.StaffAccountID,
		)

	principal,
		_,
		_,
		err :=
		s.repository.GetPrincipalByAccessTokenHash(
			ctx,
			result.Material.AccessTokenHash,
		)
	if err != nil {
		return SessionResult{},
			err
	}

	result.Principal =
		principal

	return result,
		nil
}

func (s *Service) Refresh(
	ctx context.Context,
	refreshToken string,
	csrfToken string,
	metadata ClientMetadata,
) (
	SessionResult,
	error,
) {
	refreshToken =
		strings.TrimSpace(
			refreshToken,
		)

	csrfToken =
		strings.TrimSpace(
			csrfToken,
		)

	if refreshToken == "" {
		return SessionResult{},
			ErrInvalidRefreshToken
	}

	if csrfToken == "" {
		return SessionResult{},
			ErrInvalidCSRFToken
	}

	now :=
		s.now().
			UTC()

	var result SessionResult

	err :=
		platformdatabase.WithinTx(
			ctx,
			s.repository.db,
			func(
				ctx context.Context,
				tx pgx.Tx,
			) error {

				session, err :=
					s.repository.
						LockSessionByRefreshTokenHashTx(
							ctx,
							tx,
							platformsecurity.HashToken(
								refreshToken,
							),
						)
				if err != nil {
					if errors.Is(
						err,
						pgx.ErrNoRows,
					) {
						return ErrInvalidRefreshToken
					}

					return err
				}

				if session.RevokedAt != nil {
					return ErrSessionRevoked
				}

				if session.StaffStatus !=
					"active" {

					return ErrAdminAccountDisabled
				}

				if !session.RefreshExpiresAt.After(
					now,
				) {
					return ErrInvalidRefreshToken
				}

				if !platformsecurity.VerifyCSRFToken(
					csrfToken,
					session.CSRFTokenHash,
				) {
					return ErrInvalidCSRFToken
				}

				account, err :=
					s.repository.GetStaffAccountByID(
						ctx,
						session.StaffAccountID,
					)
				if err != nil {
					return err
				}

				if !account.HasPermission(
					"admin.panel.access",
				) {
					return ErrAdminPanelAccessRequired
				}

				material, err :=
					newAdminSessionMaterial(
						now,
						&session.RefreshExpiresAt,
					)
				if err != nil {
					return err
				}

				if err :=
					s.repository.
						RotateAdminSessionTx(
							ctx,
							tx,
							session.ID,
							material,
							metadata.IPAddress,
							metadata.UserAgent,
							now,
						); err != nil {

					return err
				}

				staffID :=
					session.StaffAccountID

				sessionID :=
					session.ID

				if err :=
					s.repository.
						InsertSecurityEventTx(
							ctx,
							tx,
							newSecurityEvent(
								SecurityEventSessionRefreshed,
								SecurityOutcomeSuccess,
								&staffID,
								&sessionID,
								"",
								metadata,
								map[string]any{
									"refresh_generation": session.RefreshGeneration +
										1,
								},
								now,
							),
						); err != nil {

					return err
				}

				result =
					SessionResult{
						Principal: AdminPrincipal{
							SessionID: session.ID,

							Staff: account,

							AuthenticatedAt: session.AuthenticatedAt,

							MFAVerifiedAt: session.MFAVerifiedAt,

							AccessExpiresAt: material.AccessExpiresAt,
						},

						Material: material,
					}

				return nil
			},
		)
	if err != nil {
		return SessionResult{},
			err
	}

	return result,
		nil
}

func (s *Service) AuthenticateAccessToken(
	ctx context.Context,
	accessToken string,
) (
	AdminPrincipal,
	error,
) {
	principal,
		_,
		_,
		err :=
		s.authenticateAccessToken(
			ctx,
			accessToken,
		)

	return principal,
		err
}

func (s *Service) AuthenticateAccessTokenWithCSRF(
	ctx context.Context,
	accessToken string,
	csrfToken string,
) (
	AdminPrincipal,
	error,
) {
	principal,
		csrfHash,
		_,
		err :=
		s.authenticateAccessToken(
			ctx,
			accessToken,
		)
	if err != nil {
		return AdminPrincipal{},
			err
	}

	if !platformsecurity.VerifyCSRFToken(
		csrfToken,
		csrfHash,
	) {
		return AdminPrincipal{},
			ErrInvalidCSRFToken
	}

	return principal,
		nil
}

func (s *Service) authenticateAccessToken(
	ctx context.Context,
	accessToken string,
) (
	AdminPrincipal,
	string,
	*time.Time,
	error,
) {
	accessToken =
		strings.TrimSpace(
			accessToken,
		)

	if accessToken == "" {
		return AdminPrincipal{},
			"",
			nil,
			ErrInvalidAccessToken
	}

	principal,
		csrfHash,
		revokedAt,
		err :=
		s.repository.GetPrincipalByAccessTokenHash(
			ctx,
			platformsecurity.HashToken(
				accessToken,
			),
		)
	if err != nil {
		if errors.Is(
			err,
			pgx.ErrNoRows,
		) {
			return AdminPrincipal{},
				"",
				nil,
				ErrInvalidAccessToken
		}

		return AdminPrincipal{},
			"",
			nil,
			err
	}

	if revokedAt != nil {
		return AdminPrincipal{},
			"",
			revokedAt,
			ErrSessionRevoked
	}

	if principal.Staff.Status !=
		"active" {

		return AdminPrincipal{},
			"",
			nil,
			ErrAdminAccountDisabled
	}

	if !principal.Staff.HasPermission(
		"admin.panel.access",
	) {
		return AdminPrincipal{},
			"",
			nil,
			ErrAdminPanelAccessRequired
	}

	if !principal.AccessExpiresAt.After(
		s.now().
			UTC(),
	) {
		return AdminPrincipal{},
			"",
			nil,
			ErrInvalidAccessToken
	}

	return principal,
		csrfHash,
		nil,
		nil
}

func (s *Service) Logout(
	ctx context.Context,
	accessToken string,
	csrfToken string,
	metadata ClientMetadata,
) error {
	accessToken =
		strings.TrimSpace(
			accessToken,
		)

	csrfToken =
		strings.TrimSpace(
			csrfToken,
		)

	if accessToken == "" {
		return ErrInvalidAccessToken
	}

	principal,
		csrfHash,
		revokedAt,
		err :=
		s.repository.GetPrincipalByAccessTokenHash(
			ctx,
			platformsecurity.HashToken(
				accessToken,
			),
		)
	if err != nil {
		if errors.Is(
			err,
			pgx.ErrNoRows,
		) {
			return ErrInvalidAccessToken
		}

		return err
	}

	if revokedAt != nil {
		return nil
	}

	if !platformsecurity.VerifyCSRFToken(
		csrfToken,
		csrfHash,
	) {
		return ErrInvalidCSRFToken
	}

	now :=
		s.now().
			UTC()

	sessionID,
		staffID,
		err :=
		s.repository.
			RevokeSessionByAccessTokenHash(
				ctx,
				platformsecurity.HashToken(
					accessToken,
				),
				"logout",
				now,
			)
	if err != nil {
		return err
	}

	_ =
		s.repository.InsertSecurityEvent(
			ctx,
			newSecurityEvent(
				SecurityEventLogout,
				SecurityOutcomeSuccess,
				staffID,
				&sessionID,
				"",
				metadata,
				map[string]any{
					"staff_code": principal.Staff.StaffCode,
				},
				now,
			),
		)

	return nil
}

func (s *Service) Me(
	ctx context.Context,
	accessToken string,
) (
	MeResponse,
	error,
) {
	principal, err :=
		s.AuthenticateAccessToken(
			ctx,
			accessToken,
		)
	if err != nil {
		return MeResponse{},
			err
	}

	return MeResponse{
			Principal: principal,
		},
		nil
}

func (s *Service) failPasswordLogin(
	ctx context.Context,
	identifier string,
	metadata ClientMetadata,
	staffID *string,
) error {
	if err :=
		s.lockout.RecordFailure(
			ctx,
			identifier,
			metadata.IPAddress,
		); err != nil {

		return fmt.Errorf(
			"%w: %v",
			ErrAuthUnavailable,
			err,
		)
	}

	_ =
		s.repository.InsertSecurityEvent(
			ctx,
			newSecurityEvent(
				SecurityEventPasswordLoginFailed,
				SecurityOutcomeFailure,
				staffID,
				nil,
				identifier,
				metadata,
				nil,
				s.now(),
			),
		)

	return ErrInvalidCredentials
}

func (s *Service) validateChallenge(
	ctx context.Context,
	challengeToken string,
) (
	LoginChallenge,
	error,
) {
	challengeToken =
		strings.TrimSpace(
			challengeToken,
		)

	if challengeToken == "" {
		return LoginChallenge{},
			ErrInvalidChallenge
	}

	var result LoginChallenge

	var publicErr error

	now :=
		s.now().
			UTC()

	err :=
		platformdatabase.WithinTx(
			ctx,
			s.repository.db,
			func(
				ctx context.Context,
				tx pgx.Tx,
			) error {

				challenge, err :=
					s.repository.
						LockChallengeByTokenHashTx(
							ctx,
							tx,
							platformsecurity.HashToken(
								challengeToken,
							),
						)
				if err != nil {
					if errors.Is(
						err,
						pgx.ErrNoRows,
					) {
						publicErr =
							ErrInvalidChallenge

						return nil
					}

					return err
				}

				if err :=
					s.validateLockedChallengeTx(
						ctx,
						tx,
						challenge,
						now,
					); err != nil {

					publicErr =
						err

					return nil
				}

				result =
					challenge

				return nil
			},
		)
	if err != nil {
		return LoginChallenge{},
			err
	}

	if publicErr != nil {
		return LoginChallenge{},
			publicErr
	}

	return result,
		nil
}

func (s *Service) validateLockedChallengeTx(
	ctx context.Context,
	tx pgx.Tx,
	challenge LoginChallenge,
	now time.Time,
) error {
	switch challenge.Status {
	case ChallengeStatusPending:

	case ChallengeStatusLocked:
		return ErrChallengeLocked

	case ChallengeStatusConsumed:
		return ErrChallengeConsumed

	case ChallengeStatusExpired:
		return ErrChallengeExpired

	default:
		return ErrInvalidChallenge
	}

	if !challenge.ExpiresAt.After(
		now,
	) {
		if err :=
			s.repository.
				MarkChallengeExpiredTx(
					ctx,
					tx,
					challenge.ID,
				); err != nil {

			return err
		}

		return ErrChallengeExpired
	}

	if challenge.FailedAttempts >=
		challenge.MaxAttempts {

		return ErrChallengeLocked
	}

	return nil
}

func generateRecoveryCodesAndHashes() (
	[]string,
	[]string,
	error,
) {
	codes, err :=
		GenerateRecoveryCodes(
			DefaultRecoveryCodeCount,
		)
	if err != nil {
		return nil,
			nil,
			err
	}

	hashes :=
		make(
			[]string,
			0,
			len(
				codes,
			),
		)

	for _, code := range codes {

		hash, err :=
			HashRecoveryCode(
				code,
			)
		if err != nil {
			return nil,
				nil,
				err
		}

		hashes =
			append(
				hashes,
				hash,
			)
	}

	return codes,
		hashes,
		nil
}
