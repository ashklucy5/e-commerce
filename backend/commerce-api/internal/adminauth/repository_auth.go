package adminauth

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"project.local/commerce-api/internal/staff"
)

func (r *Repository) FindLoginAccount(
	ctx context.Context,
	identifier string,
) (
	LoginAccount,
	error,
) {
	var result LoginAccount

	err :=
		r.db.QueryRow(
			ctx,
			`
				SELECT
					sa.id::text,
					sa.staff_code,
					sa.full_name,
					sa.email,
					COALESCE(sa.phone, ''),
					sa.status,
					sa.password_hash,
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
					),
					EXISTS (
						SELECT 1
						FROM admin_totp_credentials atc
						WHERE
							atc.staff_account_id = sa.id
							AND atc.status = 'active'
					)
				FROM staff_accounts sa
				WHERE
					lower(sa.email) = lower($1)
					OR upper(sa.staff_code) = upper($1)
				LIMIT 1
			`,
			identifier,
		).Scan(
			&result.ID,
			&result.StaffCode,
			&result.FullName,
			&result.Email,
			&result.Phone,
			&result.Status,
			&result.PasswordHash,
			&result.HasPanelAccess,
			&result.HasActiveMFA,
		)
	if err != nil {
		return LoginAccount{},
			err
	}

	if err :=
		r.loadAdminAuthorization(
			ctx,
			&result.Account,
		); err != nil {

		return LoginAccount{},
			err
	}

	return result,
		nil
}

func (r *Repository) GetStaffAccountByID(
	ctx context.Context,
	staffAccountID string,
) (
	staff.Account,
	error,
) {
	var result staff.Account

	err :=
		r.db.QueryRow(
			ctx,
			`
				SELECT
					id::text,
					staff_code,
					full_name,
					email,
					COALESCE(phone, ''),
					status
				FROM staff_accounts
				WHERE id = $1::uuid
			`,
			staffAccountID,
		).Scan(
			&result.ID,
			&result.StaffCode,
			&result.FullName,
			&result.Email,
			&result.Phone,
			&result.Status,
		)
	if err != nil {
		return staff.Account{},
			err
	}

	if err :=
		r.loadAdminAuthorization(
			ctx,
			&result,
		); err != nil {

		return staff.Account{},
			err
	}

	return result,
		nil
}

func (r *Repository) loadAdminAuthorization(
	ctx context.Context,
	account *staff.Account,
) error {
	account.Roles =
		make(
			[]string,
			0,
		)

	roleRows, err :=
		r.db.Query(
			ctx,
			`
				SELECT r.code
				FROM staff_account_roles sar
				JOIN staff_roles r
					ON r.id = sar.role_id
				WHERE sar.staff_account_id = $1::uuid
				ORDER BY r.code
			`,
			account.ID,
		)
	if err != nil {
		return fmt.Errorf(
			"load Admin staff roles: %w",
			err,
		)
	}

	for roleRows.Next() {
		var role string

		if err :=
			roleRows.Scan(
				&role,
			); err != nil {

			roleRows.Close()

			return fmt.Errorf(
				"scan Admin staff role: %w",
				err,
			)
		}

		account.Roles =
			append(
				account.Roles,
				role,
			)
	}

	if err :=
		roleRows.Err(); err != nil {

		roleRows.Close()

		return fmt.Errorf(
			"iterate Admin staff roles: %w",
			err,
		)
	}

	roleRows.Close()

	account.Permissions =
		make(
			[]string,
			0,
		)

	permissionRows, err :=
		r.db.Query(
			ctx,
			`
				SELECT DISTINCT p.code
				FROM staff_account_roles sar
				JOIN staff_role_permissions srp
					ON srp.role_id = sar.role_id
				JOIN staff_permissions p
					ON p.id = srp.permission_id
				WHERE sar.staff_account_id = $1::uuid
				ORDER BY p.code
			`,
			account.ID,
		)
	if err != nil {
		return fmt.Errorf(
			"load Admin staff permissions: %w",
			err,
		)
	}

	for permissionRows.Next() {
		var permission string

		if err :=
			permissionRows.Scan(
				&permission,
			); err != nil {

			permissionRows.Close()

			return fmt.Errorf(
				"scan Admin staff permission: %w",
				err,
			)
		}

		account.Permissions =
			append(
				account.Permissions,
				permission,
			)
	}

	if err :=
		permissionRows.Err(); err != nil {

		permissionRows.Close()

		return fmt.Errorf(
			"iterate Admin staff permissions: %w",
			err,
		)
	}

	permissionRows.Close()

	return nil
}

func (r *Repository) UpdatePasswordHash(
	ctx context.Context,
	staffAccountID string,
	passwordHash string,
) error {
	tag, err :=
		r.db.Exec(
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
			"update Admin password hash: %w",
			err,
		)
	}

	if tag.RowsAffected() != 1 {
		return ErrAdminAccountNotFound
	}

	return nil
}

func (r *Repository) UpdateLastLoginTx(
	ctx context.Context,
	tx pgx.Tx,
	staffAccountID string,
	at time.Time,
) error {
	_, err :=
		tx.Exec(
			ctx,
			`
				UPDATE staff_accounts
				SET
					last_login_at = $2,
					updated_at = now()
				WHERE id = $1::uuid
			`,
			staffAccountID,
			at,
		)
	if err != nil {
		return fmt.Errorf(
			"update Admin last login: %w",
			err,
		)
	}

	return nil
}

func (r *Repository) ConsumeRecoveryCodeTx(
	ctx context.Context,
	tx pgx.Tx,
	staffAccountID string,
	codeHash string,
	usedAt time.Time,
) (
	bool,
	error,
) {
	tag, err :=
		tx.Exec(
			ctx,
			`
				UPDATE admin_mfa_recovery_codes
				SET used_at = $3
				WHERE
					staff_account_id = $1::uuid
					AND code_hash = $2
					AND used_at IS NULL
			`,
			staffAccountID,
			codeHash,
			usedAt,
		)
	if err != nil {
		return false,
			fmt.Errorf(
				"consume Admin recovery code: %w",
				err,
			)
	}

	return tag.RowsAffected() == 1,
		nil
}

func (r *Repository) CancelPendingChallengesTx(
	ctx context.Context,
	tx pgx.Tx,
	staffAccountID string,
	purpose string,
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
					AND purpose = $2
					AND status = 'pending'
			`,
			staffAccountID,
			purpose,
		)
	if err != nil {
		return fmt.Errorf(
			"cancel prior Admin login challenges: %w",
			err,
		)
	}

	return nil
}

func (r *Repository) CreateLoginChallengeTx(
	ctx context.Context,
	tx pgx.Tx,
	staffAccountID string,
	tokenHash string,
	purpose string,
	expiresAt time.Time,
	maxAttempts int,
	metadata ClientMetadata,
) (
	LoginChallenge,
	error,
) {
	var result LoginChallenge

	err :=
		tx.QueryRow(
			ctx,
			`
				INSERT INTO admin_login_challenges (
					staff_account_id,
					challenge_token_hash,
					purpose,
					status,
					failed_attempts,
					max_attempts,
					expires_at,
					verified_at,
					consumed_at,
					client_ip,
					user_agent,
					request_id,
					created_at,
					updated_at
				)
				VALUES (
					$1::uuid,
					$2,
					$3,
					'pending',
					0,
					$4,
					$5,
					NULL,
					NULL,
					NULLIF($6, ''),
					NULLIF($7, ''),
					NULLIF($8, ''),
					now(),
					now()
				)
				RETURNING
					id::text,
					staff_account_id::text,
					challenge_token_hash,
					purpose,
					status,
					failed_attempts,
					max_attempts,
					expires_at,
					verified_at,
					consumed_at,
					COALESCE(client_ip, ''),
					COALESCE(user_agent, ''),
					COALESCE(request_id, ''),
					created_at,
					updated_at
			`,
			staffAccountID,
			tokenHash,
			purpose,
			maxAttempts,
			expiresAt,
			metadata.IPAddress,
			metadata.UserAgent,
			metadata.RequestID,
		).Scan(
			&result.ID,
			&result.StaffAccountID,
			&result.ChallengeTokenHash,
			&result.Purpose,
			&result.Status,
			&result.FailedAttempts,
			&result.MaxAttempts,
			&result.ExpiresAt,
			&result.VerifiedAt,
			&result.ConsumedAt,
			&result.ClientIP,
			&result.UserAgent,
			&result.RequestID,
			&result.CreatedAt,
			&result.UpdatedAt,
		)
	if err != nil {
		return LoginChallenge{},
			fmt.Errorf(
				"create Admin login challenge: %w",
				err,
			)
	}

	return result,
		nil
}

func (r *Repository) LockChallengeByTokenHashTx(
	ctx context.Context,
	tx pgx.Tx,
	tokenHash string,
) (
	LoginChallenge,
	error,
) {
	var result LoginChallenge

	err :=
		tx.QueryRow(
			ctx,
			`
				SELECT
					id::text,
					staff_account_id::text,
					challenge_token_hash,
					purpose,
					status,
					failed_attempts,
					max_attempts,
					expires_at,
					verified_at,
					consumed_at,
					COALESCE(client_ip, ''),
					COALESCE(user_agent, ''),
					COALESCE(request_id, ''),
					created_at,
					updated_at
				FROM admin_login_challenges
				WHERE challenge_token_hash = $1
				FOR UPDATE
			`,
			tokenHash,
		).Scan(
			&result.ID,
			&result.StaffAccountID,
			&result.ChallengeTokenHash,
			&result.Purpose,
			&result.Status,
			&result.FailedAttempts,
			&result.MaxAttempts,
			&result.ExpiresAt,
			&result.VerifiedAt,
			&result.ConsumedAt,
			&result.ClientIP,
			&result.UserAgent,
			&result.RequestID,
			&result.CreatedAt,
			&result.UpdatedAt,
		)
	if err != nil {
		return LoginChallenge{},
			err
	}

	return result,
		nil
}

func (r *Repository) IncrementChallengeFailureTx(
	ctx context.Context,
	tx pgx.Tx,
	challenge LoginChallenge,
) (
	string,
	error,
) {
	newAttempts :=
		challenge.FailedAttempts + 1

	newStatus :=
		ChallengeStatusPending

	if newAttempts >=
		challenge.MaxAttempts {

		newAttempts =
			challenge.MaxAttempts

		newStatus =
			ChallengeStatusLocked
	}

	_, err :=
		tx.Exec(
			ctx,
			`
				UPDATE admin_login_challenges
				SET
					failed_attempts = $2,
					status = $3,
					updated_at = now()
				WHERE id = $1::uuid
			`,
			challenge.ID,
			newAttempts,
			newStatus,
		)
	if err != nil {
		return "",
			fmt.Errorf(
				"record Admin challenge failure: %w",
				err,
			)
	}

	return newStatus,
		nil
}

func (r *Repository) MarkChallengeExpiredTx(
	ctx context.Context,
	tx pgx.Tx,
	challengeID string,
) error {
	_, err :=
		tx.Exec(
			ctx,
			`
				UPDATE admin_login_challenges
				SET
					status = 'expired',
					updated_at = now()
				WHERE
					id = $1::uuid
					AND status = 'pending'
			`,
			challengeID,
		)
	if err != nil {
		return fmt.Errorf(
			"expire Admin login challenge: %w",
			err,
		)
	}

	return nil
}

func (r *Repository) ConsumeChallengeTx(
	ctx context.Context,
	tx pgx.Tx,
	challengeID string,
	at time.Time,
) error {
	tag, err :=
		tx.Exec(
			ctx,
			`
				UPDATE admin_login_challenges
				SET
					status = 'consumed',
					verified_at = COALESCE(
						verified_at,
						$2
					),
					consumed_at = $2,
					updated_at = now()
				WHERE
					id = $1::uuid
					AND status = 'pending'
			`,
			challengeID,
			at,
		)
	if err != nil {
		return fmt.Errorf(
			"consume Admin login challenge: %w",
			err,
		)
	}

	if tag.RowsAffected() != 1 {
		return ErrChallengeConsumed
	}

	return nil
}

func (r *Repository) CreateAdminSessionTx(
	ctx context.Context,
	tx pgx.Tx,
	staffAccountID string,
	challengeID string,
	material AdminSessionMaterial,
	authenticatedAt time.Time,
	mfaVerifiedAt time.Time,
	metadata ClientMetadata,
) (
	string,
	error,
) {
	var sessionID string

	err :=
		tx.QueryRow(
			ctx,
			`
				INSERT INTO admin_sessions (
					staff_account_id,
					login_challenge_id,
					access_token_hash,
					refresh_token_hash,
					csrf_token_hash,
					refresh_generation,
					access_expires_at,
					refresh_expires_at,
					authenticated_at,
					mfa_verified_at,
					last_used_at,
					last_rotated_at,
					created_ip,
					last_ip,
					user_agent,
					revoked_at,
					revoke_reason,
					created_at,
					updated_at
				)
				VALUES (
					$1::uuid,
					$2::uuid,
					$3,
					$4,
					$5,
					0,
					$6,
					$7,
					$8,
					$9,
					$8,
					NULL,
					NULLIF($10, ''),
					NULLIF($10, ''),
					NULLIF($11, ''),
					NULL,
					NULL,
					now(),
					now()
				)
				RETURNING id::text
			`,
			staffAccountID,
			challengeID,
			material.AccessTokenHash,
			material.RefreshTokenHash,
			material.CSRFTokenHash,
			material.AccessExpiresAt,
			material.RefreshExpiresAt,
			authenticatedAt,
			mfaVerifiedAt,
			metadata.IPAddress,
			metadata.UserAgent,
		).Scan(
			&sessionID,
		)
	if err != nil {
		return "",
			fmt.Errorf(
				"create Admin session: %w",
				err,
			)
	}

	return sessionID,
		nil
}

func (r *Repository) LockSessionByRefreshTokenHashTx(
	ctx context.Context,
	tx pgx.Tx,
	refreshTokenHash string,
) (
	AdminSessionRecord,
	error,
) {
	var result AdminSessionRecord

	err :=
		tx.QueryRow(
			ctx,
			`
				SELECT
					s.id::text,
					s.staff_account_id::text,
					COALESCE(s.login_challenge_id::text, ''),
					s.access_token_hash,
					s.refresh_token_hash,
					s.csrf_token_hash,
					s.refresh_generation,
					s.access_expires_at,
					s.refresh_expires_at,
					s.authenticated_at,
					s.mfa_verified_at,
					s.last_used_at,
					s.last_rotated_at,
					COALESCE(s.created_ip, ''),
					COALESCE(s.last_ip, ''),
					COALESCE(s.user_agent, ''),
					s.revoked_at,
					COALESCE(s.revoke_reason, ''),
					sa.status
				FROM admin_sessions s
				JOIN staff_accounts sa
					ON sa.id = s.staff_account_id
				WHERE s.refresh_token_hash = $1
				FOR UPDATE OF s
			`,
			refreshTokenHash,
		).Scan(
			&result.ID,
			&result.StaffAccountID,
			&result.LoginChallengeID,
			&result.AccessTokenHash,
			&result.RefreshTokenHash,
			&result.CSRFTokenHash,
			&result.RefreshGeneration,
			&result.AccessExpiresAt,
			&result.RefreshExpiresAt,
			&result.AuthenticatedAt,
			&result.MFAVerifiedAt,
			&result.LastUsedAt,
			&result.LastRotatedAt,
			&result.CreatedIP,
			&result.LastIP,
			&result.UserAgent,
			&result.RevokedAt,
			&result.RevokeReason,
			&result.StaffStatus,
		)
	if err != nil {
		return AdminSessionRecord{},
			err
	}

	return result,
		nil
}

func (r *Repository) RotateAdminSessionTx(
	ctx context.Context,
	tx pgx.Tx,
	sessionID string,
	material AdminSessionMaterial,
	lastIP string,
	userAgent string,
	at time.Time,
) error {
	tag, err :=
		tx.Exec(
			ctx,
			`
				UPDATE admin_sessions
				SET
					access_token_hash = $2,
					refresh_token_hash = $3,
					csrf_token_hash = $4,
					refresh_generation = refresh_generation + 1,
					access_expires_at = $5,
					last_used_at = $6,
					last_rotated_at = $6,
					last_ip = NULLIF($7, ''),
					user_agent = COALESCE(
						NULLIF($8, ''),
						user_agent
					),
					updated_at = now()
				WHERE
					id = $1::uuid
					AND revoked_at IS NULL
			`,
			sessionID,
			material.AccessTokenHash,
			material.RefreshTokenHash,
			material.CSRFTokenHash,
			material.AccessExpiresAt,
			at,
			lastIP,
			userAgent,
		)
	if err != nil {
		return fmt.Errorf(
			"rotate Admin session: %w",
			err,
		)
	}

	if tag.RowsAffected() != 1 {
		return ErrSessionRevoked
	}

	return nil
}

func (r *Repository) RevokeSessionByAccessTokenHash(
	ctx context.Context,
	accessTokenHash string,
	reason string,
	at time.Time,
) (
	string,
	*string,
	error,
) {
	var sessionID string

	var staffAccountID string

	err :=
		r.db.QueryRow(
			ctx,
			`
				UPDATE admin_sessions
				SET
					revoked_at = COALESCE(
						revoked_at,
						$3
					),
					revoke_reason = COALESCE(
						revoke_reason,
						NULLIF($2, '')
					),
					updated_at = now()
				WHERE access_token_hash = $1
				RETURNING
					id::text,
					staff_account_id::text
			`,
			accessTokenHash,
			reason,
			at,
		).Scan(
			&sessionID,
			&staffAccountID,
		)
	if err != nil {
		return "",
			nil,
			err
	}

	return sessionID,
		&staffAccountID,
		nil
}

func (r *Repository) GetPrincipalByAccessTokenHash(
	ctx context.Context,
	accessTokenHash string,
) (
	AdminPrincipal,
	string,
	*time.Time,
	error,
) {
	var principal AdminPrincipal

	var csrfTokenHash string

	var revokedAt *time.Time

	err :=
		r.db.QueryRow(
			ctx,
			`
				SELECT
					s.id::text,
					sa.id::text,
					sa.staff_code,
					sa.full_name,
					sa.email,
					COALESCE(sa.phone, ''),
					sa.status,
					s.authenticated_at,
					s.mfa_verified_at,
					s.access_expires_at,
					s.csrf_token_hash,
					s.revoked_at
				FROM admin_sessions s
				JOIN staff_accounts sa
					ON sa.id = s.staff_account_id
				WHERE s.access_token_hash = $1
				LIMIT 1
			`,
			accessTokenHash,
		).Scan(
			&principal.SessionID,
			&principal.Staff.ID,
			&principal.Staff.StaffCode,
			&principal.Staff.FullName,
			&principal.Staff.Email,
			&principal.Staff.Phone,
			&principal.Staff.Status,
			&principal.AuthenticatedAt,
			&principal.MFAVerifiedAt,
			&principal.AccessExpiresAt,
			&csrfTokenHash,
			&revokedAt,
		)
	if err != nil {
		return AdminPrincipal{},
			"",
			nil,
			err
	}

	if err :=
		r.loadAdminAuthorization(
			ctx,
			&principal.Staff,
		); err != nil {

		return AdminPrincipal{},
			"",
			nil,
			err
	}

	return principal,
		csrfTokenHash,
		revokedAt,
		nil
}

func (r *Repository) InsertSecurityEventTx(
	ctx context.Context,
	tx pgx.Tx,
	event SecurityEvent,
) error {
	details, err :=
		marshalAdminSecurityEventDetails(
			event.Details,
		)
	if err != nil {
		return err
	}

	occurredAt :=
		event.OccurredAt

	if occurredAt.IsZero() {
		occurredAt =
			time.Now().
				UTC()
	}

	_, err =
		tx.Exec(
			ctx,
			`
				INSERT INTO admin_security_events (
					staff_account_id,
					admin_session_id,
					event_type,
					outcome,
					identifier_hash,
					request_id,
					ip_address,
					user_agent,
					details,
					occurred_at,
					created_at
				)
				VALUES (
					$1::uuid,
					$2::uuid,
					$3,
					$4,
					$5,
					NULLIF($6, ''),
					NULLIF($7, ''),
					NULLIF($8, ''),
					$9::jsonb,
					$10,
					now()
				)
			`,
			event.StaffAccountID,
			event.AdminSessionID,
			event.EventType,
			event.Outcome,
			event.IdentifierHash,
			event.RequestID,
			event.IPAddress,
			event.UserAgent,
			details,
			occurredAt,
		)
	if err != nil {
		return fmt.Errorf(
			"insert Admin security event: %w",
			err,
		)
	}

	return nil
}

func (r *Repository) InsertSecurityEvent(
	ctx context.Context,
	event SecurityEvent,
) error {
	details, err :=
		marshalAdminSecurityEventDetails(
			event.Details,
		)
	if err != nil {
		return err
	}

	occurredAt :=
		event.OccurredAt

	if occurredAt.IsZero() {
		occurredAt =
			time.Now().
				UTC()
	}

	_, err =
		r.db.Exec(
			ctx,
			`
				INSERT INTO admin_security_events (
					staff_account_id,
					admin_session_id,
					event_type,
					outcome,
					identifier_hash,
					request_id,
					ip_address,
					user_agent,
					details,
					occurred_at,
					created_at
				)
				VALUES (
					$1::uuid,
					$2::uuid,
					$3,
					$4,
					$5,
					NULLIF($6, ''),
					NULLIF($7, ''),
					NULLIF($8, ''),
					$9::jsonb,
					$10,
					now()
				)
			`,
			event.StaffAccountID,
			event.AdminSessionID,
			event.EventType,
			event.Outcome,
			event.IdentifierHash,
			event.RequestID,
			event.IPAddress,
			event.UserAgent,
			details,
			occurredAt,
		)
	if err != nil {
		return fmt.Errorf(
			"insert Admin security event: %w",
			err,
		)
	}

	return nil
}

func marshalAdminSecurityEventDetails(
	details map[string]any,
) (
	[]byte,
	error,
) {
	if len(details) == 0 {
		return []byte(
				"null",
			),
			nil
	}

	value, err :=
		json.Marshal(
			details,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"marshal Admin security event details: %w",
				err,
			)
	}

	return value,
		nil
}
