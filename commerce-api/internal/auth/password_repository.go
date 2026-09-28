package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

func (r *Repository) UpdatePasswordHashTx(
	ctx context.Context,
	tx pgx.Tx,
	customerID string,
	passwordHash string,
) error {
	tag, err :=
		tx.Exec(
			ctx,
			`
				UPDATE customers
				SET
					password_hash = $2,
					updated_at = now()
				WHERE id = $1::uuid
			`,
			customerID,
			passwordHash,
		)
	if err != nil {
		return fmt.Errorf(
			"update customer password hash: %w",
			err,
		)
	}

	if tag.RowsAffected() != 1 {
		return ErrInvalidCredentials
	}

	return nil
}

func (r *Repository) GetCustomerByIDTx(
	ctx context.Context,
	tx pgx.Tx,
	customerID string,
) (
	Customer,
	error,
) {
	var customer Customer

	err :=
		tx.QueryRow(
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
				WHERE id = $1::uuid
				FOR UPDATE
			`,
			customerID,
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
				"get customer by id: %w",
				err,
			)
	}

	return customer, nil
}

func (r *Repository) RevokeAllCustomerSessionsTx(
	ctx context.Context,
	tx pgx.Tx,
	customerID string,
) error {
	_, err :=
		tx.Exec(
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
				WHERE
					customer_id = $1::uuid
					AND revoked_at IS NULL
			`,
			customerID,
		)
	if err != nil {
		return fmt.Errorf(
			"revoke all customer sessions: %w",
			err,
		)
	}

	return nil
}

func (r *Repository) RevokeOtherCustomerSessionsTx(
	ctx context.Context,
	tx pgx.Tx,
	customerID string,
	currentAccessTokenHash string,
) error {
	_, err :=
		tx.Exec(
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
				WHERE
					customer_id = $1::uuid
					AND access_token_hash <> $2
					AND revoked_at IS NULL
			`,
			customerID,
			currentAccessTokenHash,
		)
	if err != nil {
		return fmt.Errorf(
			"revoke other customer sessions: %w",
			err,
		)
	}

	return nil
}
