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

type MFAService struct {
	repository *Repository

	keyring *platformsecurity.Keyring

	issuer string

	now func() time.Time
}

func NewMFAService(
	repository *Repository,
	keyring *platformsecurity.Keyring,
	issuer string,
) (
	*MFAService,
	error,
) {
	if repository == nil {
		return nil,
			fmt.Errorf(
				"%w: repository is required",
				ErrMFAConfiguration,
			)
	}

	if keyring == nil ||
		keyring.ActiveKeyID() == "" {

		return nil,
			fmt.Errorf(
				"%w: encryption keyring is required",
				ErrMFAConfiguration,
			)
	}

	issuer =
		strings.TrimSpace(
			issuer,
		)

	if issuer == "" {
		return nil,
			fmt.Errorf(
				"%w: TOTP issuer is required",
				ErrMFAConfiguration,
			)
	}

	return &MFAService{
			repository: repository,

			keyring: keyring,

			issuer: issuer,

			now: func() time.Time {
				return time.Now().
					UTC()
			},
		},
		nil
}

func (s *MFAService) BeginEnrollment(
	ctx context.Context,
	staffAccountID string,
	label string,
) (
	MFAEnrollment,
	error,
) {
	account, err :=
		s.requireAdminAccount(
			ctx,
			staffAccountID,
		)
	if err != nil {
		return MFAEnrollment{},
			err
	}

	label =
		strings.TrimSpace(
			label,
		)

	if label == "" {
		label =
			"Authenticator"
	}

	var result MFAEnrollment

	err =
		platformdatabase.WithinTx(
			ctx,
			s.repository.db,
			func(
				ctx context.Context,
				tx pgx.Tx,
			) error {

				existing, err :=
					s.repository.
						LockTOTPCredentialTx(
							ctx,
							tx,
							account.ID,
						)

				var credentialID string

				switch {
				case err == nil:
					if existing.Status ==
						TOTPCredentialStatusActive {

						return ErrMFAAlreadyEnrolled
					}

					credentialID =
						existing.ID

				case errors.Is(
					err,
					pgx.ErrNoRows,
				):
					credentialID, err =
						s.repository.NewUUIDTx(
							ctx,
							tx,
						)
					if err != nil {
						return err
					}

				default:
					return err
				}

				secret, err :=
					GenerateTOTPSecret()
				if err != nil {
					return err
				}

				cfg :=
					DefaultTOTPConfig()

				encrypted, err :=
					s.keyring.EncryptString(
						secret,
						totpAAD(
							account.ID,
							credentialID,
						),
					)
				if err != nil {
					return fmt.Errorf(
						"encrypt Admin TOTP secret: %w",
						err,
					)
				}

				credential :=
					TOTPCredential{
						ID: credentialID,

						StaffAccountID: account.ID,

						Label: label,

						SecretCiphertext: encrypted.Ciphertext,

						EncryptionKeyID: encrypted.KeyID,

						Algorithm: cfg.Algorithm,

						Digits: cfg.Digits,

						PeriodSeconds: int(
							cfg.PeriodSeconds,
						),

						Status: TOTPCredentialStatusPending,
					}

				if err :=
					s.repository.
						SavePendingTOTPCredentialTx(
							ctx,
							tx,
							credential,
						); err != nil {

					return err
				}

				enrollmentURI, err :=
					BuildTOTPEnrollmentURI(
						s.issuer,
						account.Email,
						secret,
						cfg,
					)
				if err != nil {
					return err
				}

				result =
					MFAEnrollment{
						CredentialID: credentialID,

						Label: label,

						Secret: secret,

						EnrollmentURI: enrollmentURI,

						Algorithm: cfg.Algorithm,

						Digits: cfg.Digits,

						PeriodSeconds: int(
							cfg.PeriodSeconds,
						),
					}

				return nil
			},
		)
	if err != nil {
		return MFAEnrollment{},
			err
	}

	return result,
		nil
}

func (s *MFAService) ConfirmEnrollment(
	ctx context.Context,
	staffAccountID string,
	code string,
) (
	MFAEnrollmentConfirmation,
	error,
) {
	account, err :=
		s.requireAdminAccount(
			ctx,
			staffAccountID,
		)
	if err != nil {
		return MFAEnrollmentConfirmation{},
			err
	}

	var result MFAEnrollmentConfirmation

	err =
		platformdatabase.WithinTx(
			ctx,
			s.repository.db,
			func(
				ctx context.Context,
				tx pgx.Tx,
			) error {

				credential, err :=
					s.repository.
						LockTOTPCredentialTx(
							ctx,
							tx,
							account.ID,
						)
				if err != nil {
					if errors.Is(
						err,
						pgx.ErrNoRows,
					) {
						return ErrMFANotEnrolled
					}

					return err
				}

				if credential.Status !=
					TOTPCredentialStatusPending {

					return ErrMFAEnrollmentNotPending
				}

				secret, err :=
					s.decryptCredential(
						credential,
					)
				if err != nil {
					return err
				}

				cfg :=
					totpConfigFromCredential(
						credential,
					)

				now :=
					s.now().
						UTC()

				verification, err :=
					VerifyTOTP(
						secret,
						code,
						now,
						credential.LastAcceptedStep,
						cfg,
					)
				if err != nil {
					if errors.Is(
						err,
						ErrInvalidTOTPCode,
					) {
						return ErrInvalidMFACode
					}

					return err
				}

				if !verification.Valid {
					return ErrInvalidMFACode
				}

				recoveryCodes, err :=
					GenerateRecoveryCodes(
						DefaultRecoveryCodeCount,
					)
				if err != nil {
					return err
				}

				hashes :=
					make(
						[]string,
						0,
						len(recoveryCodes),
					)

				for _, recoveryCode := range recoveryCodes {

					hash, err :=
						HashRecoveryCode(
							recoveryCode,
						)
					if err != nil {
						return err
					}

					hashes =
						append(
							hashes,
							hash,
						)
				}

				if err :=
					s.repository.
						ReplaceRecoveryCodesTx(
							ctx,
							tx,
							account.ID,
							hashes,
						); err != nil {

					return err
				}

				if err :=
					s.repository.
						ActivateTOTPCredentialTx(
							ctx,
							tx,
							account.ID,
							verification.Step,
							now,
						); err != nil {

					return err
				}

				result =
					MFAEnrollmentConfirmation{
						CredentialID: credential.ID,

						VerifiedAt: now,

						RecoveryCodes: recoveryCodes,
					}

				return nil
			},
		)
	if err != nil {
		return MFAEnrollmentConfirmation{},
			err
	}

	return result,
		nil
}

// VerifyActiveTOTP will become the normal Admin login MFA verification
// primitive.
//
// The credential row is locked before verification and the accepted
// time-step is persisted inside the same transaction, which makes
// replay prevention safe under concurrent requests.
func (s *MFAService) VerifyActiveTOTP(
	ctx context.Context,
	staffAccountID string,
	code string,
) error {
	account, err :=
		s.requireAdminAccount(
			ctx,
			staffAccountID,
		)
	if err != nil {
		return err
	}

	return platformdatabase.WithinTx(
		ctx,
		s.repository.db,
		func(
			ctx context.Context,
			tx pgx.Tx,
		) error {

			credential, err :=
				s.repository.
					LockTOTPCredentialTx(
						ctx,
						tx,
						account.ID,
					)
			if err != nil {
				if errors.Is(
					err,
					pgx.ErrNoRows,
				) {
					return ErrMFANotEnrolled
				}

				return err
			}

			if credential.Status !=
				TOTPCredentialStatusActive {

				return ErrMFANotEnrolled
			}

			secret, err :=
				s.decryptCredential(
					credential,
				)
			if err != nil {
				return err
			}

			now :=
				s.now().
					UTC()

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
			if err != nil {
				if errors.Is(
					err,
					ErrInvalidTOTPCode,
				) {
					return ErrInvalidMFACode
				}

				return err
			}

			if !verification.Valid {
				return ErrInvalidMFACode
			}

			return s.repository.
				UpdateAcceptedTOTPStepTx(
					ctx,
					tx,
					account.ID,
					verification.Step,
					now,
				)
		},
	)
}

func (s *MFAService) requireAdminAccount(
	ctx context.Context,
	staffAccountID string,
) (
	MFAAccount,
	error,
) {
	staffAccountID =
		strings.TrimSpace(
			staffAccountID,
		)

	if staffAccountID == "" {
		return MFAAccount{},
			ErrAdminAccountNotFound
	}

	account, err :=
		s.repository.GetMFAAccount(
			ctx,
			staffAccountID,
		)
	if err != nil {
		if errors.Is(
			err,
			pgx.ErrNoRows,
		) {
			return MFAAccount{},
				ErrAdminAccountNotFound
		}

		return MFAAccount{},
			err
	}

	if account.Status !=
		"active" {

		return MFAAccount{},
			ErrAdminAccountDisabled
	}

	if !account.HasPanelAccess {
		return MFAAccount{},
			ErrAdminPanelAccessRequired
	}

	return account,
		nil
}

func (s *MFAService) decryptCredential(
	credential TOTPCredential,
) (
	string,
	error,
) {
	secret, err :=
		s.keyring.DecryptString(
			platformsecurity.EncryptedValue{
				KeyID: credential.EncryptionKeyID,

				Ciphertext: credential.SecretCiphertext,
			},
			totpAAD(
				credential.StaffAccountID,
				credential.ID,
			),
		)
	if err != nil {
		return "",
			fmt.Errorf(
				"decrypt Admin TOTP secret: %w",
				err,
			)
	}

	return secret,
		nil
}

func totpConfigFromCredential(
	credential TOTPCredential,
) TOTPConfig {
	return TOTPConfig{
		Algorithm: credential.Algorithm,

		Digits: credential.Digits,

		PeriodSeconds: int64(
			credential.PeriodSeconds,
		),

		Skew: DefaultTOTPSkew,
	}
}

func totpAAD(
	staffAccountID string,
	credentialID string,
) []byte {
	return []byte(
		"admin-totp:" +
			staffAccountID +
			":" +
			credentialID,
	)
}
