package adminauth

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

type RecoverySessionRecord struct {
	ID string

	StaffAccountID string

	CSRFTokenHash string

	AccessExpiresAt time.Time

	RecoveryVerifiedAt *time.Time

	RevokedAt *time.Time

	StaffStatus string

	HasPanelAccess bool
}

func (r *Repository) LockRecoverySessionByAccessTokenHashTx(
	ctx context.Context,
	tx pgx.Tx,
	accessTokenHash string,
) (
	RecoverySessionRecord,
	error,
) {
	var result RecoverySessionRecord

	err :=
		tx.QueryRow(
			ctx,
			`
				SELECT
					s.id::text,
					s.staff_account_id::text,
					s.csrf_token_hash,
					s.access_expires_at,
					(
						SELECT max(e.occurred_at)
						FROM admin_security_events e
						WHERE
							e.admin_session_id = s.id
							AND e.event_type = $2
							AND e.outcome = $3
					),
					s.revoked_at,
					sa.status,
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
				FROM admin_sessions s
				JOIN staff_accounts sa
					ON sa.id = s.staff_account_id
				WHERE s.access_token_hash = $1
				FOR UPDATE OF s
			`,
			accessTokenHash,
			SecurityEventRecoveryCodeUsed,
			SecurityOutcomeSuccess,
		).Scan(
			&result.ID,
			&result.StaffAccountID,
			&result.CSRFTokenHash,
			&result.AccessExpiresAt,
			&result.RecoveryVerifiedAt,
			&result.RevokedAt,
			&result.StaffStatus,
			&result.HasPanelAccess,
		)
	if err != nil {
		return RecoverySessionRecord{},
			err
	}

	return result,
		nil
}

func (r *Repository) UpdatePasswordHashTx(
	ctx context.Context,
	tx pgx.Tx,
	staffAccountID string,
	passwordHash string,
) error {
	tag, err :=
		tx.Exec(
			ctx,
			`
				UPDATE staff_accounts
				SET
					password_hash = $2,
					updated_at = now()
				WHERE id = $1::uuid
			`,
			staffAccountID,
			passwordHash,
		)
	if err != nil {
		return fmt.Errorf(
			"update Admin recovery password hash: %w",
			err,
		)
	}

	if tag.RowsAffected() != 1 {
		return ErrAdminAccountNotFound
	}

	return nil
}

func (r *Repository) DisableMFAForRecoveryTx(
	ctx context.Context,
	tx pgx.Tx,
	staffAccountID string,
	disabledAt time.Time,
) error {
	tag, err :=
		tx.Exec(
			ctx,
			`
				UPDATE admin_totp_credentials
				SET
					status = 'disabled',
					last_accepted_step = NULL,
					disabled_at = $2,
					updated_at = now()
				WHERE
					staff_account_id = $1::uuid
					AND status = 'active'
			`,
			staffAccountID,
			disabledAt,
		)
	if err != nil {
		return fmt.Errorf(
			"disable Admin MFA after recovery password reset: %w",
			err,
		)
	}

	// A recovery-code authenticated session can only exist if active MFA
	// existed when the recovery code was consumed. Treat a missing active
	// credential here as a state conflict instead of silently continuing.
	if tag.RowsAffected() != 1 {
		return ErrMFANotEnrolled
	}

	_, err =
		tx.Exec(
			ctx,
			`
				DELETE FROM admin_mfa_recovery_codes
				WHERE staff_account_id = $1::uuid
			`,
			staffAccountID,
		)
	if err != nil {
		return fmt.Errorf(
			"delete Admin recovery codes after password reset: %w",
			err,
		)
	}

	return nil
}

func (r *Repository) CancelAllPendingChallengesTx(
	ctx context.Context,
	tx pgx.Tx,
	staffAccountID string,
) error {
	_, err :=
		tx.Exec(
			ctx,
			`
				UPDATE admin_login_challenges
				SET
					status = 'cancelled',
					updated_at = now()
				WHERE
					staff_account_id = $1::uuid
					AND status = 'pending'
			`,
			staffAccountID,
		)
	if err != nil {
		return fmt.Errorf(
			"cancel Admin challenges for recovery password reset: %w",
			err,
		)
	}

	return nil
}

func (r *Repository) RevokeAllStaffSecuritySessionsTx(
	ctx context.Context,
	tx pgx.Tx,
	staffAccountID string,
	reason string,
	revokedAt time.Time,
) error {
	_, err :=
		tx.Exec(
			ctx,
			`
				UPDATE staff_sessions
				SET
					revoked_at = $2,
					updated_at = now()
				WHERE
					staff_account_id = $1::uuid
					AND revoked_at IS NULL
			`,
			staffAccountID,
			revokedAt,
		)
	if err != nil {
		return fmt.Errorf(
			"revoke staff sessions after recovery password reset: %w",
			err,
		)
	}

	_, err =
		tx.Exec(
			ctx,
			`
				UPDATE admin_sessions
				SET
					revoked_at = $3,
					revoke_reason = $2,
					updated_at = now()
				WHERE
					staff_account_id = $1::uuid
					AND revoked_at IS NULL
			`,
			staffAccountID,
			reason,
			revokedAt,
		)
	if err != nil {
		return fmt.Errorf(
			"revoke Admin sessions after recovery password reset: %w",
			err,
		)
	}

	return nil
}
