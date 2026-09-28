package staff

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(
	db *pgxpool.Pool,
) *Repository {
	return &Repository{
		db: db,
	}
}

func (r *Repository) Begin(
	ctx context.Context,
) (pgx.Tx, error) {
	tx, err :=
		r.db.Begin(
			ctx,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"begin staff transaction: %w",
				err,
			)
	}

	return tx, nil
}

func (r *Repository) FindByIdentifier(
	ctx context.Context,
	identifier string,
) (accountRecord, error) {
	const query = `
		SELECT
			id::text,
			staff_code,
			full_name,
			email,
			COALESCE(phone, ''),
			status,
			password_hash
		FROM staff_accounts
		WHERE
			lower(email) = lower($1)
			OR upper(staff_code) = upper($1)
		LIMIT 1
	`

	var result accountRecord

	err :=
		r.db.QueryRow(
			ctx,
			query,
			identifier,
		).Scan(
			&result.ID,
			&result.StaffCode,
			&result.FullName,
			&result.Email,
			&result.Phone,
			&result.Status,
			&result.PasswordHash,
		)
	if err != nil {
		return accountRecord{},
			err
	}

	return result, nil
}

func (r *Repository) GetAccountByID(
	ctx context.Context,
	staffID string,
) (Account, error) {
	const query = `
		SELECT
			id::text,
			staff_code,
			full_name,
			email,
			COALESCE(phone, ''),
			status
		FROM staff_accounts
		WHERE id = $1::uuid
		LIMIT 1
	`

	var result Account

	err :=
		r.db.QueryRow(
			ctx,
			query,
			staffID,
		).Scan(
			&result.ID,
			&result.StaffCode,
			&result.FullName,
			&result.Email,
			&result.Phone,
			&result.Status,
		)
	if err != nil {
		return Account{}, err
	}

	if err :=
		r.loadAuthorization(
			ctx,
			&result,
		); err != nil {
		return Account{}, err
	}

	return result, nil
}

func (r *Repository) GetAccountByAccessTokenHash(
	ctx context.Context,
	accessTokenHash string,
) (Account, time.Time, error) {
	const query = `
		SELECT
			a.id::text,
			a.staff_code,
			a.full_name,
			a.email,
			COALESCE(a.phone, ''),
			a.status,
			s.access_expires_at
		FROM staff_sessions s
		JOIN staff_accounts a
			ON a.id = s.staff_account_id
		WHERE
			s.access_token_hash = $1
			AND s.revoked_at IS NULL
		LIMIT 1
	`

	var result Account
	var accessExpiresAt time.Time

	err :=
		r.db.QueryRow(
			ctx,
			query,
			accessTokenHash,
		).Scan(
			&result.ID,
			&result.StaffCode,
			&result.FullName,
			&result.Email,
			&result.Phone,
			&result.Status,
			&accessExpiresAt,
		)
	if err != nil {
		return Account{},
			time.Time{},
			err
	}

	if err :=
		r.loadAuthorization(
			ctx,
			&result,
		); err != nil {
		return Account{},
			time.Time{},
			err
	}

	return result,
		accessExpiresAt,
		nil
}

func (r *Repository) loadAuthorization(
	ctx context.Context,
	account *Account,
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
				FROM staff_account_roles ar
				JOIN staff_roles r
					ON r.id = ar.role_id
				WHERE ar.staff_account_id = $1::uuid
				ORDER BY r.code
			`,
			account.ID,
		)
	if err != nil {
		return fmt.Errorf(
			"load staff roles: %w",
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
				"scan staff role: %w",
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
			"iterate staff roles: %w",
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
				FROM staff_account_roles ar
				JOIN staff_role_permissions rp
					ON rp.role_id = ar.role_id
				JOIN staff_permissions p
					ON p.id = rp.permission_id
				WHERE ar.staff_account_id = $1::uuid
				ORDER BY p.code
			`,
			account.ID,
		)
	if err != nil {
		return fmt.Errorf(
			"load staff permissions: %w",
			err,
		)
	}

	defer permissionRows.Close()

	for permissionRows.Next() {
		var permission string

		if err :=
			permissionRows.Scan(
				&permission,
			); err != nil {
			return fmt.Errorf(
				"scan staff permission: %w",
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
		return fmt.Errorf(
			"iterate staff permissions: %w",
			err,
		)
	}

	return nil
}

func (r *Repository) CreateSessionTx(
	ctx context.Context,
	tx pgx.Tx,
	staffAccountID string,
	material tokenMaterial,
) error {
	_, err :=
		tx.Exec(
			ctx,
			`
				INSERT INTO staff_sessions (
					staff_account_id,
					access_token_hash,
					refresh_token_hash,
					access_expires_at,
					refresh_expires_at,
					created_at,
					updated_at
				)
				VALUES (
					$1::uuid,
					$2,
					$3,
					$4,
					$5,
					now(),
					now()
				)
			`,
			staffAccountID,
			material.AccessTokenHash,
			material.RefreshTokenHash,
			material.AccessExpiresAt,
			material.RefreshExpiresAt,
		)
	if err != nil {
		return fmt.Errorf(
			"create staff session: %w",
			err,
		)
	}

	return nil
}

func (r *Repository) UpdateLastLoginTx(
	ctx context.Context,
	tx pgx.Tx,
	staffAccountID string,
) error {
	_, err :=
		tx.Exec(
			ctx,
			`
				UPDATE staff_accounts
				SET
					last_login_at = now(),
					updated_at = now()
				WHERE id = $1::uuid
			`,
			staffAccountID,
		)
	if err != nil {
		return fmt.Errorf(
			"update staff last login: %w",
			err,
		)
	}

	return nil
}

func (r *Repository) LockSessionByRefreshTokenTx(
	ctx context.Context,
	tx pgx.Tx,
	refreshTokenHash string,
) (refreshSessionRecord, error) {
	const query = `
		SELECT
			s.id::text,
			s.staff_account_id::text,
			s.refresh_expires_at,
			a.status
		FROM staff_sessions s
		JOIN staff_accounts a
			ON a.id = s.staff_account_id
		WHERE
			s.refresh_token_hash = $1
			AND s.revoked_at IS NULL
		FOR UPDATE OF s
	`

	var result refreshSessionRecord

	err :=
		tx.QueryRow(
			ctx,
			query,
			refreshTokenHash,
		).Scan(
			&result.ID,
			&result.StaffAccountID,
			&result.RefreshExpiresAt,
			&result.StaffStatus,
		)
	if err != nil {
		return refreshSessionRecord{},
			err
	}

	return result, nil
}

func (r *Repository) RotateSessionTx(
	ctx context.Context,
	tx pgx.Tx,
	sessionID string,
	material tokenMaterial,
) error {
	tag, err :=
		tx.Exec(
			ctx,
			`
				UPDATE staff_sessions
				SET
					access_token_hash = $2,
					refresh_token_hash = $3,
					access_expires_at = $4,
					refresh_expires_at = $5,
					last_used_at = now(),
					updated_at = now()
				WHERE
					id = $1::uuid
					AND revoked_at IS NULL
			`,
			sessionID,
			material.AccessTokenHash,
			material.RefreshTokenHash,
			material.AccessExpiresAt,
			material.RefreshExpiresAt,
		)
	if err != nil {
		return fmt.Errorf(
			"rotate staff session: %w",
			err,
		)
	}

	if tag.RowsAffected() == 0 {
		return ErrInvalidRefreshToken
	}

	return nil
}

func (r *Repository) RevokeSessionByAccessTokenHash(
	ctx context.Context,
	accessTokenHash string,
) error {
	_, err :=
		r.db.Exec(
			ctx,
			`
				UPDATE staff_sessions
				SET
					revoked_at = COALESCE(
						revoked_at,
						now()
					),
					updated_at = now()
				WHERE access_token_hash = $1
			`,
			accessTokenHash,
		)
	if err != nil {
		return fmt.Errorf(
			"revoke staff session: %w",
			err,
		)
	}

	return nil
}
