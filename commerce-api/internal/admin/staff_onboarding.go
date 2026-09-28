package admin

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"

	platformdatabase "project.local/commerce-api/internal/platform/database"
	platformsecurity "project.local/commerce-api/internal/platform/security"
)

const (
	StaffInvitationDeliveryManual = "manual"
	StaffInvitationDeliveryEmail  = "email"

	StaffInvitationTTL = 7 * 24 * time.Hour

	staffOnboardingTokenBytes = 32

	staffOnboardingPlaceholderBytes = 48
)

var (
	ErrAdminInvalidStaffOnboarding = fmt.Errorf(
		"invalid staff onboarding request",
	)

	ErrAdminInvalidInvitationDelivery = fmt.Errorf(
		"invalid staff invitation delivery mode",
	)

	ErrAdminEmailInvitationDisabled = fmt.Errorf(
		"staff email invitation delivery is not configured",
	)

	ErrAdminStaffActivationManagedSeparately = fmt.Errorf(
		"pending staff activation must be completed through the invitation flow",
	)
)

type StaffInvitationDelivery struct {
	ID string `json:"id"`

	DeliveryMode string `json:"delivery_mode"`

	/*
		Returned exactly once for manual delivery.

		Only the SHA-256 token hash is stored in PostgreSQL.
	*/
	ActivationToken string `json:"activation_token"`

	ExpiresAt time.Time `json:"expires_at"`
}

type CreateStaffResult struct {
	Staff AdminStaffDetail `json:"staff"`

	Invitation StaffInvitationDelivery `json:"invitation"`
}

func (
	s *Service,
) createPendingStaff(
	ctx context.Context,
	input CreateStaffInput,
	metadata AdminActionMetadata,
) (
	CreateStaffResult,
	error,
) {
	input.FullName =
		strings.TrimSpace(
			input.FullName,
		)

	input.Email =
		strings.ToLower(
			strings.TrimSpace(
				input.Email,
			),
		)

	input.Phone =
		strings.TrimSpace(
			input.Phone,
		)

	input.DeliveryMode =
		strings.ToLower(
			strings.TrimSpace(
				input.DeliveryMode,
			),
		)

	if input.DeliveryMode == "" {
		input.DeliveryMode =
			StaffInvitationDeliveryManual
	}

	if err :=
		validateCreateStaffOnboardingInput(
			input,
		); err != nil {

		return CreateStaffResult{},
			err
	}

	/*
		The contract deliberately knows about email delivery,
		but no SMTP/provider exists yet.

		Fail closed. Do not silently fall back to manual mode,
		because an administrator who requested email delivery
		must not believe an invitation was sent.
	*/
	if input.DeliveryMode ==
		StaffInvitationDeliveryEmail {

		return CreateStaffResult{},
			ErrAdminEmailInvitationDisabled
	}

	roleCodes :=
		normalizeAdminCodes(
			input.RoleCodes,
		)

	if len(roleCodes) == 0 {
		return CreateStaffResult{},
			ErrAdminInvalidStaffOnboarding
	}

	activationToken, err :=
		generateStaffOnboardingSecret(
			staffOnboardingTokenBytes,
		)
	if err != nil {
		return CreateStaffResult{},
			err
	}

	activationTokenHash :=
		platformsecurity.HashToken(
			activationToken,
		)

	/*
		staff_accounts.password_hash is existing required
		security state.

		The invitee has not selected a password yet, so use a
		cryptographically random, unrecoverable placeholder hash.
		The plaintext material is never returned or persisted.
	*/
	placeholderMaterial, err :=
		generateStaffOnboardingSecret(
			staffOnboardingPlaceholderBytes,
		)
	if err != nil {
		return CreateStaffResult{},
			err
	}

	placeholderPasswordHash, err :=
		platformsecurity.HashPassword(
			placeholderMaterial,
		)
	if err != nil {
		return CreateStaffResult{},
			fmt.Errorf(
				"create pending staff password placeholder: %w",
				err,
			)
	}

	now :=
		time.Now().
			UTC()

	expiresAt :=
		now.Add(
			StaffInvitationTTL,
		)

	queryCtx, cancel :=
		adminReadContext(ctx)
	defer cancel()

	var result CreateStaffResult

	err =
		platformdatabase.WithinTxOptions(
			queryCtx,
			s.db,
			pgx.TxOptions{},
			func(
				ctx context.Context,
				tx pgx.Tx,
			) error {
				if err :=
					validateRoleCodesTx(
						ctx,
						tx,
						roleCodes,
					); err != nil {

					return err
				}

				if err :=
					validateCreateStaffRoleAssignmentTx(
						ctx,
						tx,
						metadata.StaffAccountID,
						roleCodes,
					); err != nil {

					return err
				}

				if _, err :=
					tx.Exec(
						ctx,
						`
							SELECT pg_advisory_xact_lock(
								hashtext(
									'commerce-staff-code-v1'
								)
							)
						`,
					); err != nil {

					return fmt.Errorf(
						"lock staff code allocation: %w",
						err,
					)
				}

				staffCode, err :=
					nextAdminStaffCodeTx(
						ctx,
						tx,
					)
				if err != nil {
					return err
				}

				var staffID string

				err =
					tx.QueryRow(
						ctx,
						`
							INSERT INTO staff_accounts (
								staff_code,
								full_name,
								email,
								phone,
								password_hash,
								status,
								created_at,
								updated_at
							)
							VALUES (
								$1,
								$2,
								$3,
								NULLIF($4, ''),
								$5,
								'pending_activation',
								$6,
								$6
							)
							RETURNING id::text
						`,
						staffCode,
						input.FullName,
						input.Email,
						input.Phone,
						placeholderPasswordHash,
						now,
					).Scan(
						&staffID,
					)
				if err != nil {
					if isAdminUniqueViolation(
						err,
					) {
						return ErrAdminStaffConflict
					}

					return fmt.Errorf(
						"create pending Admin staff account: %w",
						err,
					)
				}

				if err :=
					replaceStaffRolesTx(
						ctx,
						tx,
						staffID,
						roleCodes,
						metadata.StaffAccountID,
					); err != nil {

					return err
				}

				var invitationID string

				err =
					tx.QueryRow(
						ctx,
						`
							INSERT INTO staff_invitations (
								staff_account_id,
								email,
								token_hash,
								delivery_mode,
								delivery_status,
								status,
								expires_at,
								created_by_staff_id,
								created_at,
								updated_at
							)
							VALUES (
								$1::uuid,
								$2,
								$3,
								$4,
								'not_requested',
								'pending',
								$5,
								$6::uuid,
								$7,
								$7
							)
							RETURNING id::text
						`,
						staffID,
						input.Email,
						activationTokenHash,
						input.DeliveryMode,
						expiresAt,
						metadata.StaffAccountID,
						now,
					).Scan(
						&invitationID,
					)
				if err != nil {
					return fmt.Errorf(
						"create staff invitation: %w",
						err,
					)
				}

				/*
					Never put either the activation token or its
					hash into the audit details.
				*/
				if err :=
					insertAdminActionAuditTx(
						ctx,
						tx,
						metadata,
						adminEventStaffCreated,
						map[string]any{
							"staff_account_id": staffID,

							"staff_code": staffCode,

							"role_codes": roleCodes,

							"status": "pending_activation",

							"invitation_id": invitationID,

							"invitation_delivery_mode": input.DeliveryMode,

							"invitation_expires_at": expiresAt,
						},
					); err != nil {

					return err
				}

				staff,
					err :=
					getAdminStaffDetail(
						ctx,
						tx,
						staffID,
					)
				if err != nil {
					return err
				}

				result =
					CreateStaffResult{
						Staff: staff,

						Invitation: StaffInvitationDelivery{
							ID: invitationID,

							DeliveryMode: input.DeliveryMode,

							ActivationToken: activationToken,

							ExpiresAt: expiresAt,
						},
					}

				return nil
			},
		)
	if err != nil {
		return CreateStaffResult{},
			err
	}

	return result,
		nil
}

func validateCreateStaffOnboardingInput(
	input CreateStaffInput,
) error {
	if input.FullName == "" ||
		utf8.RuneCountInString(
			input.FullName,
		) > 160 {

		return ErrAdminInvalidStaffOnboarding
	}

	if input.Email == "" ||
		utf8.RuneCountInString(
			input.Email,
		) > 255 {

		return ErrAdminInvalidStaffOnboarding
	}

	parsed, err :=
		mail.ParseAddress(
			input.Email,
		)
	if err != nil ||
		!strings.EqualFold(
			parsed.Address,
			input.Email,
		) {

		return ErrAdminInvalidStaffOnboarding
	}

	if utf8.RuneCountInString(
		input.Phone,
	) > 40 {

		return ErrAdminInvalidStaffOnboarding
	}

	if len(input.RoleCodes) == 0 ||
		len(input.RoleCodes) > 32 {

		return ErrAdminInvalidStaffOnboarding
	}

	switch input.DeliveryMode {
	case StaffInvitationDeliveryManual,
		StaffInvitationDeliveryEmail:

	default:
		return ErrAdminInvalidInvitationDelivery
	}

	return nil
}

func generateStaffOnboardingSecret(
	byteCount int,
) (
	string,
	error,
) {
	if byteCount <= 0 {
		return "",
			fmt.Errorf(
				"invalid onboarding secret length",
			)
	}

	material, err :=
		platformsecurity.RandomBytes(
			byteCount,
		)
	if err != nil {
		return "",
			fmt.Errorf(
				"generate staff onboarding secret: %w",
				err,
			)
	}

	return base64.RawURLEncoding.
			EncodeToString(
				material,
			),
		nil
}

/*
ensureStaffOnboardingSecurityMutationAllowedTx prevents an
administrator from bypassing invitation ownership by using the normal
password-reset or MFA-reset endpoints while the invitee has not yet
activated the account.

The caller must already hold the staff row lock.
*/
func ensureStaffOnboardingSecurityMutationAllowedTx(
	ctx context.Context,
	tx pgx.Tx,
	staffID string,
) error {
	var status string

	err :=
		tx.QueryRow(
			ctx,
			`
				SELECT status
				FROM staff_accounts
				WHERE id = $1::uuid
			`,
			staffID,
		).Scan(
			&status,
		)
	if err != nil {
		return fmt.Errorf(
			"load staff onboarding status: %w",
			err,
		)
	}

	switch status {
	case "pending_activation":
		return ErrAdminStaffActivationManagedSeparately

	case "banned":
		return ErrAdminStaffBanManagedSeparately

	case "deleted":
		return ErrAdminStaffDeleted

	default:
		return nil
	}
}

func cancelPendingStaffInvitationsTx(
	ctx context.Context,
	tx pgx.Tx,
	staffID string,
) error {
	_, err :=
		tx.Exec(
			ctx,
			`
				UPDATE staff_invitations
				SET
					status = 'cancelled',
					cancelled_at =
						COALESCE(
							cancelled_at,
							now()
						),
					updated_at = now()
				WHERE
					staff_account_id = $1::uuid
					AND status = 'pending'
			`,
			staffID,
		)
	if err != nil {
		return fmt.Errorf(
			"cancel pending staff invitation: %w",
			err,
		)
	}

	return nil
}
