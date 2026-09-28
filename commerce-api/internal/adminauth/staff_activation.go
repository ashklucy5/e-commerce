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
	SecurityEventStaffActivationPasswordSet = "staff_activation_password_set"

	SecurityEventStaffActivationMFAEnrollmentStarted = "staff_activation_mfa_enrollment_started"

	SecurityEventStaffActivationCompleted = "staff_activation_completed"
)

var (
	ErrInvalidStaffActivation = errors.New(
		"invalid staff activation",
	)

	ErrStaffActivationExpired = errors.New(
		"staff activation has expired",
	)

	ErrStaffActivationCompleted = errors.New(
		"staff activation has already been completed",
	)

	ErrStaffActivationPasswordAlreadySet = errors.New(
		"staff activation password has already been set",
	)

	ErrStaffActivationPasswordNotSet = errors.New(
		"staff activation password has not been set",
	)

	ErrStaffActivationStateConflict = errors.New(
		"staff activation state conflict",
	)
)

type StaffActivationPasswordRequest struct {
	Email string `json:"email"`

	ActivationToken string `json:"activation_token"`

	Password string `json:"password"`
}

type StaffActivationMFAEnrollRequest struct {
	Email string `json:"email"`

	ActivationToken string `json:"activation_token"`

	Label string `json:"label"`
}

type StaffActivationMFAConfirmRequest struct {
	Email string `json:"email"`

	ActivationToken string `json:"activation_token"`

	Code string `json:"code"`
}

type StaffActivationPasswordResult struct {
	StaffAccountID string `json:"staff_account_id"`

	Email string `json:"email"`

	PasswordSetAt time.Time `json:"password_set_at"`

	Next string `json:"next"`
}

type StaffActivationMFAEnrollResult struct {
	StaffAccountID string `json:"staff_account_id"`

	Email string `json:"email"`

	Enrollment MFAEnrollment `json:"enrollment"`
}

type StaffActivationCompleteResult struct {
	Activated bool `json:"activated"`

	StaffAccountID string `json:"staff_account_id"`

	Email string `json:"email"`

	ActivatedAt time.Time `json:"activated_at"`

	RecoveryCodes []string `json:"recovery_codes"`

	Next string `json:"next"`
}

type staffActivationRecord struct {
	InvitationID string

	StaffAccountID string

	Email string

	InvitationStatus string

	ExpiresAt time.Time

	PasswordSet bool

	StaffStatus string

	AccountEmailMatches bool

	HasPanelAccess bool
}

func (s *Service) SetActivationPassword(
	ctx context.Context,
	request StaffActivationPasswordRequest,
	metadata ClientMetadata,
) (
	StaffActivationPasswordResult,
	error,
) {
	email,
		activationToken,
		err :=
		normalizeStaffActivationCredentials(
			request.Email,
			request.ActivationToken,
		)
	if err != nil {
		return StaffActivationPasswordResult{},
			err
	}

	if err :=
		ValidateNewPassword(
			request.Password,
		); err != nil {

		return StaffActivationPasswordResult{},
			err
	}

	passwordHash, err :=
		HashNewPassword(
			request.Password,
		)
	if err != nil {
		return StaffActivationPasswordResult{},
			err
	}

	now :=
		s.now().
			UTC()

	var result StaffActivationPasswordResult

	var publicErr error

	err =
		platformdatabase.WithinTx(
			ctx,
			s.repository.db,
			func(
				ctx context.Context,
				tx pgx.Tx,
			) error {
				record, err :=
					s.lockStaffActivationTx(
						ctx,
						tx,
						email,
						activationToken,
					)
				if err != nil {
					if errors.Is(
						err,
						pgx.ErrNoRows,
					) {
						publicErr =
							ErrInvalidStaffActivation

						return nil
					}

					return err
				}

				validationErr :=
					validateStaffActivationRecord(
						record,
						now,
						false,
					)

				if validationErr != nil {
					if errors.Is(
						validationErr,
						ErrStaffActivationExpired,
					) {
						if err :=
							markStaffInvitationExpiredTx(
								ctx,
								tx,
								record.InvitationID,
								now,
							); err != nil {

							return err
						}
					}

					publicErr =
						validationErr

					return nil
				}

				if record.PasswordSet {
					publicErr =
						ErrStaffActivationPasswordAlreadySet

					return nil
				}

				tag, err :=
					tx.Exec(
						ctx,
						`
							UPDATE staff_accounts
							SET
								password_hash = $2,
								updated_at = $3
							WHERE
								id = $1::uuid
								AND status = 'pending_activation'
						`,
						record.StaffAccountID,
						passwordHash,
						now,
					)
				if err != nil {
					return fmt.Errorf(
						"set staff activation password: %w",
						err,
					)
				}

				if tag.RowsAffected() != 1 {
					publicErr =
						ErrStaffActivationStateConflict

					return nil
				}

				tag, err =
					tx.Exec(
						ctx,
						`
							UPDATE staff_invitations
							SET
								password_set_at = $2,
								updated_at = $2
							WHERE
								id = $1::uuid
								AND status = 'pending'
								AND password_set_at IS NULL
						`,
						record.InvitationID,
						now,
					)
				if err != nil {
					return fmt.Errorf(
						"mark staff activation password set: %w",
						err,
					)
				}

				if tag.RowsAffected() != 1 {
					publicErr =
						ErrStaffActivationStateConflict

					return nil
				}

				staffID :=
					record.StaffAccountID

				if err :=
					s.repository.
						InsertSecurityEventTx(
							ctx,
							tx,
							newSecurityEvent(
								SecurityEventStaffActivationPasswordSet,
								SecurityOutcomeSuccess,
								&staffID,
								nil,
								"",
								metadata,
								map[string]any{
									"invitation_id": record.InvitationID,
								},
								now,
							),
						); err != nil {

					return err
				}

				result =
					StaffActivationPasswordResult{
						StaffAccountID: record.StaffAccountID,

						Email: record.Email,

						PasswordSetAt: now,

						Next: "mfa_enroll",
					}

				return nil
			},
		)
	if err != nil {
		return StaffActivationPasswordResult{},
			err
	}

	if publicErr != nil {
		return StaffActivationPasswordResult{},
			publicErr
	}

	return result,
		nil
}

func (s *Service) BeginActivationMFA(
	ctx context.Context,
	request StaffActivationMFAEnrollRequest,
	metadata ClientMetadata,
) (
	StaffActivationMFAEnrollResult,
	error,
) {
	email,
		activationToken,
		err :=
		normalizeStaffActivationCredentials(
			request.Email,
			request.ActivationToken,
		)
	if err != nil {
		return StaffActivationMFAEnrollResult{},
			err
	}

	preRecord, err :=
		s.getStaffActivation(
			ctx,
			email,
			activationToken,
		)
	if err != nil {
		return StaffActivationMFAEnrollResult{},
			err
	}

	now :=
		s.now().
			UTC()

	if err :=
		validateStaffActivationRecord(
			preRecord,
			now,
			true,
		); err != nil {

		return StaffActivationMFAEnrollResult{},
			err
	}

	blocked, err :=
		s.lockout.IsMFABlocked(
			ctx,
			preRecord.StaffAccountID,
			metadata.IPAddress,
		)
	if err != nil {
		return StaffActivationMFAEnrollResult{},
			fmt.Errorf(
				"%w: %v",
				ErrAuthUnavailable,
				err,
			)
	}

	if blocked {
		staffID :=
			preRecord.StaffAccountID

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
						"stage": "staff_activation_enrollment",
					},
					now,
				),
			)

		return StaffActivationMFAEnrollResult{},
			ErrMFABlocked
	}

	label :=
		strings.TrimSpace(
			request.Label,
		)

	if label == "" {
		label =
			"Authenticator"
	}

	var result StaffActivationMFAEnrollResult

	var publicErr error

	err =
		platformdatabase.WithinTx(
			ctx,
			s.repository.db,
			func(
				ctx context.Context,
				tx pgx.Tx,
			) error {
				record, err :=
					s.lockStaffActivationTx(
						ctx,
						tx,
						email,
						activationToken,
					)
				if err != nil {
					if errors.Is(
						err,
						pgx.ErrNoRows,
					) {
						publicErr =
							ErrInvalidStaffActivation

						return nil
					}

					return err
				}

				validationErr :=
					validateStaffActivationRecord(
						record,
						now,
						true,
					)

				if validationErr != nil {
					if errors.Is(
						validationErr,
						ErrStaffActivationExpired,
					) {
						if err :=
							markStaffInvitationExpiredTx(
								ctx,
								tx,
								record.InvitationID,
								now,
							); err != nil {

							return err
						}
					}

					publicErr =
						validationErr

					return nil
				}

				existing, err :=
					s.repository.
						LockTOTPCredentialTx(
							ctx,
							tx,
							record.StaffAccountID,
						)

				var credentialID string

				switch {
				case err == nil:
					if existing.Status ==
						TOTPCredentialStatusActive {

						publicErr =
							ErrMFAAlreadyEnrolled

						return nil
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
					s.mfa.keyring.EncryptString(
						secret,
						totpAAD(
							record.StaffAccountID,
							credentialID,
						),
					)
				if err != nil {
					return fmt.Errorf(
						"encrypt staff activation TOTP secret: %w",
						err,
					)
				}

				credential :=
					TOTPCredential{
						ID: credentialID,

						StaffAccountID: record.StaffAccountID,

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
						s.mfa.issuer,
						record.Email,
						secret,
						cfg,
					)
				if err != nil {
					return err
				}

				staffID :=
					record.StaffAccountID

				if err :=
					s.repository.
						InsertSecurityEventTx(
							ctx,
							tx,
							newSecurityEvent(
								SecurityEventStaffActivationMFAEnrollmentStarted,
								SecurityOutcomeSuccess,
								&staffID,
								nil,
								"",
								metadata,
								map[string]any{
									"invitation_id": record.InvitationID,

									"credential_id": credentialID,
								},
								now,
							),
						); err != nil {

					return err
				}

				result =
					StaffActivationMFAEnrollResult{
						StaffAccountID: record.StaffAccountID,

						Email: record.Email,

						Enrollment: MFAEnrollment{
							CredentialID: credentialID,

							Label: label,

							Secret: secret,

							EnrollmentURI: enrollmentURI,

							Algorithm: cfg.Algorithm,

							Digits: cfg.Digits,

							PeriodSeconds: int(
								cfg.PeriodSeconds,
							),
						},
					}

				return nil
			},
		)
	if err != nil {
		return StaffActivationMFAEnrollResult{},
			err
	}

	if publicErr != nil {
		return StaffActivationMFAEnrollResult{},
			publicErr
	}

	return result,
		nil
}

func (s *Service) ConfirmActivationMFA(
	ctx context.Context,
	request StaffActivationMFAConfirmRequest,
	metadata ClientMetadata,
) (
	StaffActivationCompleteResult,
	error,
) {
	email,
		activationToken,
		err :=
		normalizeStaffActivationCredentials(
			request.Email,
			request.ActivationToken,
		)
	if err != nil {
		return StaffActivationCompleteResult{},
			err
	}

	code :=
		strings.TrimSpace(
			request.Code,
		)

	if code == "" {
		return StaffActivationCompleteResult{},
			ErrInvalidRequest
	}

	preRecord, err :=
		s.getStaffActivation(
			ctx,
			email,
			activationToken,
		)
	if err != nil {
		return StaffActivationCompleteResult{},
			err
	}

	now :=
		s.now().
			UTC()

	if err :=
		validateStaffActivationRecord(
			preRecord,
			now,
			true,
		); err != nil {

		return StaffActivationCompleteResult{},
			err
	}

	blocked, err :=
		s.lockout.IsMFABlocked(
			ctx,
			preRecord.StaffAccountID,
			metadata.IPAddress,
		)
	if err != nil {
		return StaffActivationCompleteResult{},
			fmt.Errorf(
				"%w: %v",
				ErrAuthUnavailable,
				err,
			)
	}

	if blocked {
		staffID :=
			preRecord.StaffAccountID

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
						"stage": "staff_activation_confirmation",
					},
					now,
				),
			)

		return StaffActivationCompleteResult{},
			ErrMFABlocked
	}

	var result StaffActivationCompleteResult

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
				record, err :=
					s.lockStaffActivationTx(
						ctx,
						tx,
						email,
						activationToken,
					)
				if err != nil {
					if errors.Is(
						err,
						pgx.ErrNoRows,
					) {
						publicErr =
							ErrInvalidStaffActivation

						return nil
					}

					return err
				}

				validationErr :=
					validateStaffActivationRecord(
						record,
						now,
						true,
					)

				if validationErr != nil {
					if errors.Is(
						validationErr,
						ErrStaffActivationExpired,
					) {
						if err :=
							markStaffInvitationExpiredTx(
								ctx,
								tx,
								record.InvitationID,
								now,
							); err != nil {

							return err
						}
					}

					publicErr =
						validationErr

					return nil
				}

				credential, err :=
					s.repository.
						LockTOTPCredentialTx(
							ctx,
							tx,
							record.StaffAccountID,
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

				verification, verifyErr :=
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

				if verifyErr != nil ||
					!verification.Valid {

					mfaFailed =
						true

					publicErr =
						ErrInvalidMFACode

					staffID :=
						record.StaffAccountID

					if err :=
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
										"invitation_id": record.InvitationID,

										"stage": "staff_activation_confirmation",
									},
									now,
								),
							); err != nil {

						return err
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
							record.StaffAccountID,
							hashes,
						); err != nil {

					return err
				}

				if err :=
					s.repository.
						ActivateTOTPCredentialTx(
							ctx,
							tx,
							record.StaffAccountID,
							verification.Step,
							now,
						); err != nil {

					return err
				}

				tag, err :=
					tx.Exec(
						ctx,
						`
							UPDATE staff_accounts
							SET
								status = 'active',
								updated_at = $2
							WHERE
								id = $1::uuid
								AND status = 'pending_activation'
						`,
						record.StaffAccountID,
						now,
					)
				if err != nil {
					return fmt.Errorf(
						"activate staff account: %w",
						err,
					)
				}

				if tag.RowsAffected() != 1 {
					publicErr =
						ErrStaffActivationStateConflict

					return nil
				}

				tag, err =
					tx.Exec(
						ctx,
						`
							UPDATE staff_invitations
							SET
								status = 'accepted',
								accepted_at = $2,
								updated_at = $2
							WHERE
								id = $1::uuid
								AND status = 'pending'
								AND password_set_at IS NOT NULL
						`,
						record.InvitationID,
						now,
					)
				if err != nil {
					return fmt.Errorf(
						"accept staff invitation: %w",
						err,
					)
				}

				if tag.RowsAffected() != 1 {
					publicErr =
						ErrStaffActivationStateConflict

					return nil
				}

				staffID :=
					record.StaffAccountID

				if err :=
					s.repository.
						InsertSecurityEventTx(
							ctx,
							tx,
							newSecurityEvent(
								SecurityEventStaffActivationCompleted,
								SecurityOutcomeSuccess,
								&staffID,
								nil,
								"",
								metadata,
								map[string]any{
									"invitation_id": record.InvitationID,

									"credential_id": credential.ID,
								},
								now,
							),
						); err != nil {

					return err
				}

				result =
					StaffActivationCompleteResult{
						Activated: true,

						StaffAccountID: record.StaffAccountID,

						Email: record.Email,

						ActivatedAt: now,

						RecoveryCodes: recoveryCodes,

						Next: "login",
					}

				return nil
			},
		)
	if err != nil {
		return StaffActivationCompleteResult{},
			err
	}

	if mfaFailed {
		_ =
			s.lockout.RecordMFAFailure(
				ctx,
				preRecord.StaffAccountID,
				metadata.IPAddress,
			)
	}

	if publicErr != nil {
		return StaffActivationCompleteResult{},
			publicErr
	}

	_ =
		s.lockout.RecordMFASuccess(
			ctx,
			preRecord.StaffAccountID,
		)

	return result,
		nil
}

func (s *Service) getStaffActivation(
	ctx context.Context,
	email string,
	activationToken string,
) (
	staffActivationRecord,
	error,
) {
	var result staffActivationRecord

	err :=
		s.repository.db.QueryRow(
			ctx,
			staffActivationSelectSQL(
				false,
			),
			platformsecurity.HashToken(
				activationToken,
			),
			email,
		).Scan(
			&result.InvitationID,
			&result.StaffAccountID,
			&result.Email,
			&result.InvitationStatus,
			&result.ExpiresAt,
			&result.PasswordSet,
			&result.StaffStatus,
			&result.AccountEmailMatches,
			&result.HasPanelAccess,
		)
	if errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return staffActivationRecord{},
			ErrInvalidStaffActivation
	}

	if err != nil {
		return staffActivationRecord{},
			fmt.Errorf(
				"load staff activation: %w",
				err,
			)
	}

	return result,
		nil
}

func (s *Service) lockStaffActivationTx(
	ctx context.Context,
	tx pgx.Tx,
	email string,
	activationToken string,
) (
	staffActivationRecord,
	error,
) {
	var result staffActivationRecord

	err :=
		tx.QueryRow(
			ctx,
			staffActivationSelectSQL(
				true,
			),
			platformsecurity.HashToken(
				activationToken,
			),
			email,
		).Scan(
			&result.InvitationID,
			&result.StaffAccountID,
			&result.Email,
			&result.InvitationStatus,
			&result.ExpiresAt,
			&result.PasswordSet,
			&result.StaffStatus,
			&result.AccountEmailMatches,
			&result.HasPanelAccess,
		)
	if err != nil {
		return staffActivationRecord{},
			err
	}

	return result,
		nil
}

func staffActivationSelectSQL(
	lock bool,
) string {
	query := `
		SELECT
			si.id::text,
			si.staff_account_id::text,
			lower(si.email),
			si.status,
			si.expires_at,
			si.password_set_at IS NOT NULL,
			sa.status,
			lower(sa.email) = lower(si.email),
			EXISTS (
				SELECT 1
				FROM staff_account_roles sar
				JOIN staff_role_permissions srp
					ON srp.role_id = sar.role_id
				JOIN staff_permissions sp
					ON sp.id = srp.permission_id
				WHERE
					sar.staff_account_id = sa.id
					AND sp.code = 'admin.panel.access'
			)
		FROM staff_invitations si
		JOIN staff_accounts sa
			ON sa.id = si.staff_account_id
		WHERE
			si.token_hash = $1
			AND lower(si.email) = lower($2)
		LIMIT 1
	`

	if lock {
		query += `
			FOR UPDATE OF si, sa
		`
	}

	return query
}

func validateStaffActivationRecord(
	record staffActivationRecord,
	now time.Time,
	requirePassword bool,
) error {
	switch record.InvitationStatus {
	case "pending":

	case "accepted":
		return ErrStaffActivationCompleted

	case "expired":
		return ErrStaffActivationExpired

	case "cancelled":
		return ErrInvalidStaffActivation

	default:
		return ErrInvalidStaffActivation
	}

	if !record.ExpiresAt.After(
		now,
	) {
		return ErrStaffActivationExpired
	}

	if record.StaffStatus !=
		"pending_activation" {

		if record.StaffStatus ==
			"active" {

			return ErrStaffActivationCompleted
		}

		return ErrStaffActivationStateConflict
	}

	if !record.AccountEmailMatches {
		return ErrInvalidStaffActivation
	}

	if !record.HasPanelAccess {
		return ErrAdminPanelAccessRequired
	}

	if requirePassword &&
		!record.PasswordSet {

		return ErrStaffActivationPasswordNotSet
	}

	return nil
}

func markStaffInvitationExpiredTx(
	ctx context.Context,
	tx pgx.Tx,
	invitationID string,
	now time.Time,
) error {
	_, err :=
		tx.Exec(
			ctx,
			`
				UPDATE staff_invitations
				SET
					status = 'expired',
					updated_at = $2
				WHERE
					id = $1::uuid
					AND status = 'pending'
					AND expires_at <= $2
			`,
			invitationID,
			now,
		)
	if err != nil {
		return fmt.Errorf(
			"expire staff invitation: %w",
			err,
		)
	}

	return nil
}

func normalizeStaffActivationCredentials(
	email string,
	activationToken string,
) (
	string,
	string,
	error,
) {
	email =
		strings.ToLower(
			strings.TrimSpace(
				email,
			),
		)

	activationToken =
		strings.TrimSpace(
			activationToken,
		)

	if email == "" ||
		activationToken == "" ||
		len(email) > 255 ||
		len(activationToken) > 512 {

		return "",
			"",
			ErrInvalidStaffActivation
	}

	return email,
		activationToken,
		nil
}
