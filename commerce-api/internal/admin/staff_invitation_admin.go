package admin

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	platformdatabase "project.local/commerce-api/internal/platform/database"
	platformsecurity "project.local/commerce-api/internal/platform/security"
)

const (
	adminEventStaffInvitationReissued  = "staff_invitation_reissued"
	adminEventStaffInvitationCancelled = "staff_invitation_cancelled"
)

var (
	ErrAdminStaffInvitationNotFound = errors.New(
		"staff invitation not found",
	)

	ErrAdminStaffInvitationState = errors.New(
		"staff invitation can only be managed while the account is pending activation",
	)

	ErrAdminStaffInvitationNotPending = errors.New(
		"staff account has no pending invitation to cancel",
	)

	ErrAdminStaffInvitationMFAConflict = errors.New(
		"pending staff account already has active Admin MFA",
	)
)

type AdminStaffInvitation struct {
	ID string `json:"id"`

	Email string `json:"email"`

	DeliveryMode   string `json:"delivery_mode"`
	DeliveryStatus string `json:"delivery_status"`

	// Status is the effective state. A stale persisted "pending" row whose
	// expiry has passed is reported as "expired" without making GET mutative.
	Status string `json:"status"`

	ExpiresAt time.Time `json:"expires_at"`

	PasswordSetAt *time.Time `json:"password_set_at,omitempty"`
	AcceptedAt    *time.Time `json:"accepted_at,omitempty"`
	CancelledAt   *time.Time `json:"cancelled_at,omitempty"`
	DeliveredAt   *time.Time `json:"delivered_at,omitempty"`

	DeliveryError string `json:"delivery_error,omitempty"`

	CreatedByStaffID string `json:"created_by_staff_id"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ReissueStaffInvitationResult struct {
	Invitation AdminStaffInvitation `json:"invitation"`

	// Returned exactly once for manual delivery. Only its SHA-256 hash is
	// persisted in PostgreSQL.
	ActivationToken string `json:"activation_token"`
}

func (s *Service) GetStaffInvitation(
	ctx context.Context,
	staffID string,
) (
	AdminStaffInvitation,
	error,
) {
	queryCtx, cancel := adminReadContext(ctx)
	defer cancel()

	if err := ensureAdminStaffExists(
		queryCtx,
		s.db,
		staffID,
	); err != nil {
		return AdminStaffInvitation{}, err
	}

	result, err := loadLatestStaffInvitation(
		queryCtx,
		s.db,
		staffID,
		time.Now().UTC(),
	)
	if err != nil {
		return AdminStaffInvitation{}, err
	}

	return result, nil
}

func (s *Service) ReissueStaffInvitation(
	ctx context.Context,
	staffID string,
	metadata AdminActionMetadata,
) (
	ReissueStaffInvitationResult,
	error,
) {
	activationToken, err := generateStaffOnboardingSecret(
		staffOnboardingTokenBytes,
	)
	if err != nil {
		return ReissueStaffInvitationResult{}, err
	}

	activationTokenHash := platformsecurity.HashToken(
		activationToken,
	)

	now := time.Now().UTC()
	expiresAt := now.Add(StaffInvitationTTL)

	queryCtx, cancel := adminReadContext(ctx)
	defer cancel()

	var result ReissueStaffInvitationResult

	err = platformdatabase.WithinTxOptions(
		queryCtx,
		s.db,
		pgx.TxOptions{},
		func(
			ctx context.Context,
			tx pgx.Tx,
		) error {
			email, err := lockPendingActivationStaffForInvitationTx(
				ctx,
				tx,
				staffID,
				metadata.StaffAccountID,
			)
			if err != nil {
				return err
			}

			if err := ensureNoActiveAdminMFAForInvitationTx(
				ctx,
				tx,
				staffID,
			); err != nil {
				return err
			}

			// Preserve terminal semantics: invitations which have naturally
			// expired become expired; only still-live pending invitations are
			// cancelled by the reissue.
			if _, err := tx.Exec(
				ctx,
				`
					UPDATE staff_invitations
					SET
						status = 'expired',
						updated_at = $2
					WHERE
						staff_account_id = $1::uuid
						AND status = 'pending'
						AND expires_at <= $2
				`,
				staffID,
				now,
			); err != nil {
				return fmt.Errorf(
					"expire previous staff invitations before reissue: %w",
					err,
				)
			}

			if _, err := tx.Exec(
				ctx,
				`
					UPDATE staff_invitations
					SET
						status = 'cancelled',
						cancelled_at = COALESCE(cancelled_at, $2),
						updated_at = $2
					WHERE
						staff_account_id = $1::uuid
						AND status = 'pending'
				`,
				staffID,
				now,
			); err != nil {
				return fmt.Errorf(
					"cancel previous staff invitations before reissue: %w",
					err,
				)
			}

			// If the previous activation attempt reached MFA enrollment but was
			// never confirmed, remove that pending secret. The replacement
			// invitation starts a clean activation attempt.
			if _, err := tx.Exec(
				ctx,
				`
					DELETE FROM admin_totp_credentials
					WHERE
						staff_account_id = $1::uuid
						AND status = 'pending'
				`,
				staffID,
			); err != nil {
				return fmt.Errorf(
					"clear pending Admin MFA before invitation reissue: %w",
					err,
				)
			}

			var invitationID string

			if err := tx.QueryRow(
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
						'manual',
						'not_requested',
						'pending',
						$4,
						$5::uuid,
						$6,
						$6
					)
					RETURNING id::text
				`,
				staffID,
				email,
				activationTokenHash,
				expiresAt,
				metadata.StaffAccountID,
				now,
			).Scan(&invitationID); err != nil {
				return fmt.Errorf(
					"reissue staff invitation: %w",
					err,
				)
			}

			invitation, err := loadStaffInvitationByID(
				ctx,
				tx,
				invitationID,
				now,
			)
			if err != nil {
				return err
			}

			if err := insertAdminActionAuditTx(
				ctx,
				tx,
				metadata,
				adminEventStaffInvitationReissued,
				map[string]any{
					"staff_account_id": staffID,
					"invitation_id":    invitationID,
					"delivery_mode":    StaffInvitationDeliveryManual,
					"expires_at":       expiresAt,
				},
			); err != nil {
				return err
			}

			result = ReissueStaffInvitationResult{
				Invitation:      invitation,
				ActivationToken: activationToken,
			}

			return nil
		},
	)
	if err != nil {
		return ReissueStaffInvitationResult{}, err
	}

	return result, nil
}

func (s *Service) CancelStaffInvitation(
	ctx context.Context,
	staffID string,
	metadata AdminActionMetadata,
) (
	AdminStaffInvitation,
	error,
) {
	now := time.Now().UTC()

	queryCtx, cancel := adminReadContext(ctx)
	defer cancel()

	var result AdminStaffInvitation

	err := platformdatabase.WithinTxOptions(
		queryCtx,
		s.db,
		pgx.TxOptions{},
		func(
			ctx context.Context,
			tx pgx.Tx,
		) error {
			if _, err := lockPendingActivationStaffForInvitationTx(
				ctx,
				tx,
				staffID,
				metadata.StaffAccountID,
			); err != nil {
				return err
			}

			if err := ensureNoActiveAdminMFAForInvitationTx(
				ctx,
				tx,
				staffID,
			); err != nil {
				return err
			}

			if _, err := tx.Exec(
				ctx,
				`
					UPDATE staff_invitations
					SET
						status = 'expired',
						updated_at = $2
					WHERE
						staff_account_id = $1::uuid
						AND status = 'pending'
						AND expires_at <= $2
				`,
				staffID,
				now,
			); err != nil {
				return fmt.Errorf(
					"expire staff invitations before cancellation: %w",
					err,
				)
			}

			var invitationID string

			err := tx.QueryRow(
				ctx,
				`
					SELECT id::text
					FROM staff_invitations
					WHERE
						staff_account_id = $1::uuid
						AND status = 'pending'
					ORDER BY created_at DESC, id DESC
					LIMIT 1
					FOR UPDATE
				`,
				staffID,
			).Scan(&invitationID)
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrAdminStaffInvitationNotPending
			}
			if err != nil {
				return fmt.Errorf(
					"lock pending staff invitation: %w",
					err,
				)
			}

			if _, err := tx.Exec(
				ctx,
				`
					UPDATE staff_invitations
					SET
						status = 'cancelled',
						cancelled_at = $2,
						updated_at = $2
					WHERE
						staff_account_id = $1::uuid
						AND status = 'pending'
				`,
				staffID,
				now,
			); err != nil {
				return fmt.Errorf(
					"cancel staff invitation: %w",
					err,
				)
			}

			if _, err := tx.Exec(
				ctx,
				`
					DELETE FROM admin_totp_credentials
					WHERE
						staff_account_id = $1::uuid
						AND status = 'pending'
				`,
				staffID,
			); err != nil {
				return fmt.Errorf(
					"clear pending Admin MFA after invitation cancellation: %w",
					err,
				)
			}

			invitation, err := loadStaffInvitationByID(
				ctx,
				tx,
				invitationID,
				now,
			)
			if err != nil {
				return err
			}

			if err := insertAdminActionAuditTx(
				ctx,
				tx,
				metadata,
				adminEventStaffInvitationCancelled,
				map[string]any{
					"staff_account_id": staffID,
					"invitation_id":    invitationID,
				},
			); err != nil {
				return err
			}

			result = invitation
			return nil
		},
	)
	if err != nil {
		return AdminStaffInvitation{}, err
	}

	return result, nil
}

func lockPendingActivationStaffForInvitationTx(
	ctx context.Context,
	tx pgx.Tx,
	staffID string,
	actorStaffID string,
) (
	string,
	error,
) {
	var status string
	var email string

	err := tx.QueryRow(
		ctx,
		`
			SELECT status, lower(email)
			FROM staff_accounts
			WHERE id = $1::uuid
			FOR UPDATE
		`,
		staffID,
	).Scan(
		&status,
		&email,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrAdminStaffNotFound
	}
	if err != nil {
		return "", fmt.Errorf(
			"lock staff account for invitation management: %w",
			err,
		)
	}

	if status != "pending_activation" {
		return "", ErrAdminStaffInvitationState
	}

	targetRoles, err := adminStaffRoleCodesTx(
		ctx,
		tx,
		staffID,
	)
	if err != nil {
		return "", err
	}

	if err := validateProtectedStaffMutationTx(
		ctx,
		tx,
		actorStaffID,
		targetRoles,
	); err != nil {
		return "", err
	}

	return email, nil
}

func ensureNoActiveAdminMFAForInvitationTx(
	ctx context.Context,
	tx pgx.Tx,
	staffID string,
) error {
	var activeCount int

	if err := tx.QueryRow(
		ctx,
		`
			SELECT COUNT(*)::integer
			FROM admin_totp_credentials
			WHERE
				staff_account_id = $1::uuid
				AND status = 'active'
		`,
		staffID,
	).Scan(&activeCount); err != nil {
		return fmt.Errorf(
			"check active Admin MFA before invitation management: %w",
			err,
		)
	}

	if activeCount != 0 {
		return ErrAdminStaffInvitationMFAConflict
	}

	return nil
}

func ensureAdminStaffExists(
	ctx context.Context,
	querier adminReadQuerier,
	staffID string,
) error {
	var exists bool

	if err := querier.QueryRow(
		ctx,
		`SELECT EXISTS(SELECT 1 FROM staff_accounts WHERE id = $1::uuid)`,
		staffID,
	).Scan(&exists); err != nil {
		return fmt.Errorf(
			"check Admin staff account: %w",
			err,
		)
	}

	if !exists {
		return ErrAdminStaffNotFound
	}

	return nil
}

func loadLatestStaffInvitation(
	ctx context.Context,
	querier adminReadQuerier,
	staffID string,
	now time.Time,
) (
	AdminStaffInvitation,
	error,
) {
	return scanAdminStaffInvitation(
		querier.QueryRow(
			ctx,
			staffInvitationSelectSQL+`
				WHERE staff_account_id = $1::uuid
				ORDER BY created_at DESC, id DESC
				LIMIT 1
			`,
			staffID,
		),
		now,
	)
}

func loadStaffInvitationByID(
	ctx context.Context,
	querier adminReadQuerier,
	invitationID string,
	now time.Time,
) (
	AdminStaffInvitation,
	error,
) {
	return scanAdminStaffInvitation(
		querier.QueryRow(
			ctx,
			staffInvitationSelectSQL+`
				WHERE id = $1::uuid
			`,
			invitationID,
		),
		now,
	)
}

const staffInvitationSelectSQL = `
	SELECT
		id::text,
		email,
		delivery_mode,
		delivery_status,
		status,
		expires_at,
		password_set_at,
		accepted_at,
		cancelled_at,
		delivered_at,
		COALESCE(delivery_error, ''),
		created_by_staff_id::text,
		created_at,
		updated_at
	FROM staff_invitations
`

func scanAdminStaffInvitation(
	row pgx.Row,
	now time.Time,
) (
	AdminStaffInvitation,
	error,
) {
	var result AdminStaffInvitation

	var passwordSetAt pgtype.Timestamptz
	var acceptedAt pgtype.Timestamptz
	var cancelledAt pgtype.Timestamptz
	var deliveredAt pgtype.Timestamptz

	err := row.Scan(
		&result.ID,
		&result.Email,
		&result.DeliveryMode,
		&result.DeliveryStatus,
		&result.Status,
		&result.ExpiresAt,
		&passwordSetAt,
		&acceptedAt,
		&cancelledAt,
		&deliveredAt,
		&result.DeliveryError,
		&result.CreatedByStaffID,
		&result.CreatedAt,
		&result.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return AdminStaffInvitation{}, ErrAdminStaffInvitationNotFound
	}
	if err != nil {
		return AdminStaffInvitation{}, fmt.Errorf(
			"load staff invitation: %w",
			err,
		)
	}

	if result.Status == "pending" && !result.ExpiresAt.After(now) {
		result.Status = "expired"
	}

	result.PasswordSetAt = adminTimePointer(passwordSetAt)
	result.AcceptedAt = adminTimePointer(acceptedAt)
	result.CancelledAt = adminTimePointer(cancelledAt)
	result.DeliveredAt = adminTimePointer(deliveredAt)

	result.Email = strings.ToLower(strings.TrimSpace(result.Email))

	return result, nil
}
