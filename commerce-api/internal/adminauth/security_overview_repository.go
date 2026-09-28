package adminauth

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

func (r *Repository) GetSelfSecurityOverview(
	ctx context.Context,
	staffAccountID string,
	currentSessionID string,
	now time.Time,
) (
	SelfSecurityMFAOverview,
	[]SelfSecuritySessionOverview,
	error,
) {
	var mfa SelfSecurityMFAOverview

	if err :=
		r.db.QueryRow(
			ctx,
			`
				SELECT
					COALESCE(atc.status = 'active', false),
					COALESCE(atc.label, ''),
					atc.verified_at,
					atc.last_used_at,
					(
						SELECT count(*)
						FROM admin_mfa_recovery_codes rc
						WHERE
							rc.staff_account_id = sa.id
							AND rc.used_at IS NULL
					)
				FROM staff_accounts sa
				LEFT JOIN admin_totp_credentials atc
					ON atc.staff_account_id = sa.id
				WHERE sa.id = $1::uuid
			`,
			staffAccountID,
		).Scan(
			&mfa.Enabled,
			&mfa.Label,
			&mfa.VerifiedAt,
			&mfa.LastUsedAt,
			&mfa.RecoveryCodesRemaining,
		); err != nil {

		return SelfSecurityMFAOverview{},
			nil,
			fmt.Errorf(
				"load Admin self security MFA overview: %w",
				err,
			)
	}

	rows, err :=
		r.db.Query(
			ctx,
			`
				SELECT
					id::text,
					authenticated_at,
					mfa_verified_at,
					last_used_at,
					last_rotated_at,
					COALESCE(created_ip, ''),
					COALESCE(last_ip, ''),
					COALESCE(user_agent, ''),
					refresh_expires_at
				FROM admin_sessions
				WHERE
					staff_account_id = $1::uuid
					AND revoked_at IS NULL
					AND refresh_expires_at > $2
				ORDER BY authenticated_at DESC, id DESC
			`,
			staffAccountID,
			now,
		)
	if err != nil {
		return SelfSecurityMFAOverview{},
			nil,
			fmt.Errorf(
				"list active Admin self security sessions: %w",
				err,
			)
	}
	defer rows.Close()

	sessions :=
		make(
			[]SelfSecuritySessionOverview,
			0,
		)

	for rows.Next() {
		var session SelfSecuritySessionOverview

		if err :=
			rows.Scan(
				&session.ID,
				&session.AuthenticatedAt,
				&session.MFAVerifiedAt,
				&session.LastUsedAt,
				&session.LastRotatedAt,
				&session.CreatedIP,
				&session.LastIP,
				&session.UserAgent,
				&session.RefreshExpiresAt,
			); err != nil {

			return SelfSecurityMFAOverview{},
				nil,
				fmt.Errorf(
					"scan active Admin self security session: %w",
					err,
				)
		}

		session.Current =
			session.ID == currentSessionID

		sessions =
			append(
				sessions,
				session,
			)
	}

	if err := rows.Err(); err != nil {
		return SelfSecurityMFAOverview{},
			nil,
			fmt.Errorf(
				"iterate active Admin self security sessions: %w",
				err,
			)
	}

	return mfa,
		sessions,
		nil
}

func (r *Repository) RevokeOwnedAdminSessionTx(
	ctx context.Context,
	tx pgx.Tx,
	staffAccountID string,
	targetSessionID string,
	currentSessionID string,
	revokedAt time.Time,
) (
	string,
	error,
) {
	var revokedSessionID string

	if err :=
		tx.QueryRow(
			ctx,
			`
				UPDATE admin_sessions
				SET
					revoked_at = $4,
					revoke_reason = 'self_session_revoke',
					updated_at = now()
				WHERE
					staff_account_id = $1::uuid
					AND id::text = $2
					AND id::text <> $3
					AND revoked_at IS NULL
					AND refresh_expires_at > $4
				RETURNING id::text
			`,
			staffAccountID,
			targetSessionID,
			currentSessionID,
			revokedAt,
		).Scan(
			&revokedSessionID,
		); err != nil {

		return "",
			err
	}

	return revokedSessionID,
		nil
}

func (r *Repository) RevokeOtherOwnedAdminSessionsTx(
	ctx context.Context,
	tx pgx.Tx,
	staffAccountID string,
	currentSessionID string,
	revokedAt time.Time,
) (
	int64,
	error,
) {
	tag, err :=
		tx.Exec(
			ctx,
			`
				UPDATE admin_sessions
				SET
					revoked_at = $3,
					revoke_reason = 'self_revoke_other_sessions',
					updated_at = now()
				WHERE
					staff_account_id = $1::uuid
					AND id::text <> $2
					AND revoked_at IS NULL
					AND refresh_expires_at > $3
			`,
			staffAccountID,
			currentSessionID,
			revokedAt,
		)
	if err != nil {
		return 0,
			fmt.Errorf(
				"revoke other active Admin self security sessions: %w",
				err,
			)
	}

	return tag.RowsAffected(),
		nil
}
