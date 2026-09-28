package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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
				"begin auth transaction: %w",
				err,
			)
	}

	return tx, nil
}

func (r *Repository) IsPhoneRegistered(
	ctx context.Context,
	phone string,
) (
	bool,
	error,
) {
	var exists bool

	err :=
		r.db.QueryRow(
			ctx,
			`
				SELECT EXISTS (
					SELECT 1
					FROM customers
					WHERE phone = $1
				)
			`,
			phone,
		).Scan(
			&exists,
		)
	if err != nil {
		return false,
			fmt.Errorf(
				"check customer phone registration: %w",
				err,
			)
	}

	return exists, nil
}

func (r *Repository) CreateCustomerTx(
	ctx context.Context,
	tx pgx.Tx,
	phone string,
	email string,
	passwordHash string,
	fullName string,
) (Customer, error) {
	var customer Customer

	err :=
		tx.QueryRow(
			ctx,
			`
				INSERT INTO customers (
					phone,
					email,
					password_hash,
					full_name,
					status,
					created_at,
					updated_at
				)
				VALUES (
					$1,
					NULLIF($2, ''),
					$3,
					$4,
					'active',
					now(),
					now()
				)
				RETURNING
					id::text,
					phone,
					COALESCE(email, ''),
					password_hash,
					full_name,
					status,
					created_at,
					updated_at
			`,
			phone,
			email,
			passwordHash,
			fullName,
		).Scan(
			&customer.ID,
			&customer.Phone,
			&customer.Email,
			&customer.PasswordHash,
			&customer.FullName,
			&customer.Status,
			&customer.CreatedAt,
			&customer.UpdatedAt,
		)
	if err != nil {
		var pgErr *pgconn.PgError

		if errors.As(
			err,
			&pgErr,
		) &&
			pgErr.Code == "23505" {

			switch pgErr.ConstraintName {
			case "customers_phone_key":
				return Customer{},
					ErrPhoneInUse

			case "customers_email_key":
				return Customer{},
					ErrEmailInUse
			}
		}

		return Customer{},
			fmt.Errorf(
				"create customer: %w",
				err,
			)
	}

	return customer, nil
}

func (r *Repository) GetCustomerByPhone(
	ctx context.Context,
	phone string,
) (Customer, error) {
	var customer Customer

	err :=
		r.db.QueryRow(
			ctx,
			`
				SELECT
					id::text,
					phone,
					COALESCE(email, ''),
					password_hash,
					full_name,
					status,
					created_at,
					updated_at
				FROM customers
				WHERE phone = $1
			`,
			phone,
		).Scan(
			&customer.ID,
			&customer.Phone,
			&customer.Email,
			&customer.PasswordHash,
			&customer.FullName,
			&customer.Status,
			&customer.CreatedAt,
			&customer.UpdatedAt,
		)
	if err != nil {
		if errors.Is(
			err,
			pgx.ErrNoRows,
		) {
			return Customer{},
				ErrInvalidCredentials
		}

		return Customer{},
			fmt.Errorf(
				"get customer by phone: %w",
				err,
			)
	}

	return customer, nil
}

func (r *Repository) CreateSessionTx(
	ctx context.Context,
	tx pgx.Tx,
	customerID string,
	material tokenMaterial,
) error {
	_, err :=
		tx.Exec(
			ctx,
			`
				INSERT INTO auth_sessions (
					customer_id,
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
			customerID,
			material.AccessTokenHash,
			material.RefreshTokenHash,
			material.AccessExpiresAt,
			material.RefreshExpiresAt,
		)
	if err != nil {
		return fmt.Errorf(
			"create auth session: %w",
			err,
		)
	}

	return nil
}

func (r *Repository) LockSessionByRefreshTokenTx(
	ctx context.Context,
	tx pgx.Tx,
	refreshTokenHash string,
) (sessionRecord, error) {
	var result sessionRecord

	err :=
		tx.QueryRow(
			ctx,
			`
				SELECT
					s.id::text,
					s.customer_id::text,
					s.refresh_expires_at,
					c.id::text,
					c.phone,
					COALESCE(c.email, ''),
					c.password_hash,
					c.full_name,
					c.status,
					c.created_at,
					c.updated_at
				FROM auth_sessions s
				JOIN customers c
					ON c.id = s.customer_id
				WHERE
					s.refresh_token_hash = $1
					AND s.revoked_at IS NULL
				FOR UPDATE OF s
			`,
			refreshTokenHash,
		).Scan(
			&result.ID,
			&result.CustomerID,
			&result.RefreshExpiresAt,
			&result.Customer.ID,
			&result.Customer.Phone,
			&result.Customer.Email,
			&result.Customer.PasswordHash,
			&result.Customer.FullName,
			&result.Customer.Status,
			&result.Customer.CreatedAt,
			&result.Customer.UpdatedAt,
		)
	if err != nil {
		if errors.Is(
			err,
			pgx.ErrNoRows,
		) {
			return sessionRecord{},
				ErrInvalidRefreshToken
		}

		return sessionRecord{},
			fmt.Errorf(
				"lock auth session by refresh token: %w",
				err,
			)
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
				UPDATE auth_sessions
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
			"rotate auth session: %w",
			err,
		)
	}

	if tag.RowsAffected() == 0 {
		return ErrInvalidRefreshToken
	}

	return nil
}

func (r *Repository) GetCustomerByAccessTokenHash(
	ctx context.Context,
	accessTokenHash string,
) (Customer, error) {
	var customer Customer
	var accessExpiresAt time.Time

	err :=
		r.db.QueryRow(
			ctx,
			`
				SELECT
					c.id::text,
					c.phone,
					COALESCE(c.email, ''),
					c.password_hash,
					c.full_name,
					c.status,
					c.created_at,
					c.updated_at,
					s.access_expires_at
				FROM auth_sessions s
				JOIN customers c
					ON c.id = s.customer_id
				WHERE
					s.access_token_hash = $1
					AND s.revoked_at IS NULL
			`,
			accessTokenHash,
		).Scan(
			&customer.ID,
			&customer.Phone,
			&customer.Email,
			&customer.PasswordHash,
			&customer.FullName,
			&customer.Status,
			&customer.CreatedAt,
			&customer.UpdatedAt,
			&accessExpiresAt,
		)
	if err != nil {
		if errors.Is(
			err,
			pgx.ErrNoRows,
		) {
			return Customer{},
				ErrInvalidAccessToken
		}

		return Customer{},
			fmt.Errorf(
				"get customer by access token: %w",
				err,
			)
	}

	if !accessExpiresAt.After(
		time.Now(),
	) {
		return Customer{},
			ErrInvalidAccessToken
	}

	return customer, nil
}

func (r *Repository) RevokeSessionByAccessTokenHash(
	ctx context.Context,
	accessTokenHash string,
) error {
	_, err :=
		r.db.Exec(
			ctx,
			`
				UPDATE auth_sessions
				SET
					revoked_at =
						COALESCE(
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
			"revoke auth session: %w",
			err,
		)
	}

	return nil
}
