package adminauth

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

type SelfSecuritySessionRecord struct {
	ID string

	StaffAccountID string

	Email string

	PasswordHash string

	CSRFTokenHash string

	AccessExpiresAt time.Time

	RefreshExpiresAt time.Time

	AuthenticatedAt time.Time

	MFAVerifiedAt time.Time

	RevokedAt *time.Time

	StaffStatus string

	HasPanelAccess bool
}

func (r *Repository) LockSelfSecuritySessionByAccessTokenHashTx(
	ctx context.Context,
	tx pgx.Tx,
	accessTokenHash string,
) (
	SelfSecuritySessionRecord,
	error,
) {
	var result SelfSecuritySessionRecord

	err := tx.QueryRow(
		ctx,
		`
			SELECT
				s.id::text,
				s.staff_account_id::text,
				sa.email,
				sa.password_hash,
				s.csrf_token_hash,
				s.access_expires_at,
				s.refresh_expires_at,
				s.authenticated_at,
				s.mfa_verified_at,
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
			FOR UPDATE OF s, sa
		`,
		accessTokenHash,
	).Scan(
		&result.ID,
		&result.StaffAccountID,
		&result.Email,
		&result.PasswordHash,
		&result.CSRFTokenHash,
		&result.AccessExpiresAt,
		&result.RefreshExpiresAt,
		&result.AuthenticatedAt,
		&result.MFAVerifiedAt,
		&result.RevokedAt,
		&result.StaffStatus,
		&result.HasPanelAccess,
	)
	if err != nil {
		return SelfSecuritySessionRecord{}, err
	}

	return result, nil
}

func (r *Repository) RotateActiveTOTPCredentialTx(
	ctx context.Context,
	tx pgx.Tx,
	staffAccountID string,
	pending pendingMFARotation,
	acceptedStep int64,
	verifiedAt time.Time,
) error {
	tag, err := tx.Exec(
		ctx,
		`
			UPDATE admin_totp_credentials
			SET
				label = $3,
				secret_ciphertext = $4,
				encryption_key_id = $5,
				algorithm = $6,
				digits = $7,
				period_seconds = $8,
				status = 'active',
				last_accepted_step = $9,
				verified_at = $10,
				last_used_at = $10,
				disabled_at = NULL,
				updated_at = now()
			WHERE
				staff_account_id = $1::uuid
				AND id = $2::uuid
				AND status = 'active'
		`,
		staffAccountID,
		pending.CredentialID,
		pending.Label,
		pending.SecretCiphertext,
		pending.EncryptionKeyID,
		pending.Algorithm,
		pending.Digits,
		pending.PeriodSeconds,
		acceptedStep,
		verifiedAt,
	)
	if err != nil {
		return fmt.Errorf("rotate active Admin TOTP credential: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return ErrInvalidMFARotation
	}

	return nil
}

func (r *Repository) RevokeOtherStaffSecuritySessionsTx(
	ctx context.Context,
	tx pgx.Tx,
	staffAccountID string,
	currentAdminSessionID string,
	reason string,
	revokedAt time.Time,
) error {
	if _, err := tx.Exec(
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
	); err != nil {
		return fmt.Errorf("revoke staff sessions after self security change: %w", err)
	}

	if _, err := tx.Exec(
		ctx,
		`
			UPDATE admin_sessions
			SET
				revoked_at = $4,
				revoke_reason = $3,
				updated_at = now()
			WHERE
				staff_account_id = $1::uuid
				AND id <> $2::uuid
				AND revoked_at IS NULL
		`,
		staffAccountID,
		currentAdminSessionID,
		reason,
		revokedAt,
	); err != nil {
		return fmt.Errorf("revoke other Admin sessions after self security change: %w", err)
	}

	return nil
}

func (r *Repository) UpdateAdminSessionMFAVerifiedAtTx(
	ctx context.Context,
	tx pgx.Tx,
	sessionID string,
	verifiedAt time.Time,
) error {
	tag, err := tx.Exec(
		ctx,
		`
			UPDATE admin_sessions
			SET
				mfa_verified_at = $2,
				updated_at = now()
			WHERE
				id = $1::uuid
				AND revoked_at IS NULL
		`,
		sessionID,
		verifiedAt,
	)
	if err != nil {
		return fmt.Errorf("update Admin session MFA verification time: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return ErrSessionRevoked
	}

	return nil
}
