package customer

import (
	"context"
	"errors"
	"fmt"

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
				"begin customer transaction: %w",
				err,
			)
	}

	return tx, nil
}

func (r *Repository) GetCustomer(
	ctx context.Context,
	customerID string,
) (Customer, error) {
	var result Customer

	err :=
		r.db.QueryRow(
			ctx,
			`
				SELECT
					id::text,
					phone,
					COALESCE(email, ''),
					full_name,
					status,
					created_at,
					updated_at
				FROM customers
				WHERE id = $1::uuid
			`,
			customerID,
		).Scan(
			&result.ID,
			&result.Phone,
			&result.Email,
			&result.FullName,
			&result.Status,
			&result.CreatedAt,
			&result.UpdatedAt,
		)
	if err != nil {
		if errors.Is(
			err,
			pgx.ErrNoRows,
		) {
			return Customer{},
				ErrCustomerNotFound
		}

		return Customer{},
			fmt.Errorf(
				"get customer: %w",
				err,
			)
	}

	return result, nil
}

func (r *Repository) LockCustomerTx(
	ctx context.Context,
	tx pgx.Tx,
	customerID string,
) error {
	var id string

	err :=
		tx.QueryRow(
			ctx,
			`
				SELECT id::text
				FROM customers
				WHERE id = $1::uuid
				FOR UPDATE
			`,
			customerID,
		).Scan(
			&id,
		)
	if err != nil {
		if errors.Is(
			err,
			pgx.ErrNoRows,
		) {
			return ErrCustomerNotFound
		}

		return fmt.Errorf(
			"lock customer: %w",
			err,
		)
	}

	return nil
}

func (r *Repository) UpdateCustomerTx(
	ctx context.Context,
	tx pgx.Tx,
	customerID string,
	phone string,
	email string,
	fullName string,
) (Customer, error) {
	var result Customer

	err :=
		tx.QueryRow(
			ctx,
			`
				UPDATE customers
				SET
					phone = $2,
					email = NULLIF($3, ''),
					full_name = $4,
					updated_at = now()
				WHERE id = $1::uuid
				RETURNING
					id::text,
					phone,
					COALESCE(email, ''),
					full_name,
					status,
					created_at,
					updated_at
			`,
			customerID,
			phone,
			email,
			fullName,
		).Scan(
			&result.ID,
			&result.Phone,
			&result.Email,
			&result.FullName,
			&result.Status,
			&result.CreatedAt,
			&result.UpdatedAt,
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

		if errors.Is(
			err,
			pgx.ErrNoRows,
		) {
			return Customer{},
				ErrCustomerNotFound
		}

		return Customer{},
			fmt.Errorf(
				"update customer: %w",
				err,
			)
	}

	return result, nil
}

func (r *Repository) ListAddresses(
	ctx context.Context,
	customerID string,
) ([]Address, error) {
	rows, err :=
		r.db.Query(
			ctx,
			`
				SELECT
					id::text,
					customer_id::text,
					label,
					recipient_name,
					phone,
					address_line1,
					COALESCE(address_line2, ''),
					city,
					area,
					COALESCE(postal_code, ''),
					is_default,
					created_at,
					updated_at
				FROM customer_addresses
				WHERE customer_id = $1::uuid
				ORDER BY
					is_default DESC,
					created_at ASC
			`,
			customerID,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"list customer addresses: %w",
				err,
			)
	}
	defer rows.Close()

	results :=
		make(
			[]Address,
			0,
		)

	for rows.Next() {
		address, err :=
			scanAddress(
				rows,
			)
		if err != nil {
			return nil,
				fmt.Errorf(
					"scan customer address: %w",
					err,
				)
		}

		results =
			append(
				results,
				address,
			)
	}

	if err :=
		rows.Err(); err != nil {
		return nil,
			fmt.Errorf(
				"iterate customer addresses: %w",
				err,
			)
	}

	return results, nil
}

func (r *Repository) LockAddressTx(
	ctx context.Context,
	tx pgx.Tx,
	customerID string,
	addressID string,
) (Address, error) {
	row :=
		tx.QueryRow(
			ctx,
			`
				SELECT
					id::text,
					customer_id::text,
					label,
					recipient_name,
					phone,
					address_line1,
					COALESCE(address_line2, ''),
					city,
					area,
					COALESCE(postal_code, ''),
					is_default,
					created_at,
					updated_at
				FROM customer_addresses
				WHERE
					id = $2::uuid
					AND customer_id = $1::uuid
				FOR UPDATE
			`,
			customerID,
			addressID,
		)

	result, err :=
		scanAddress(
			row,
		)
	if err != nil {
		if errors.Is(
			err,
			pgx.ErrNoRows,
		) {
			return Address{},
				ErrAddressNotFound
		}

		return Address{},
			fmt.Errorf(
				"lock customer address: %w",
				err,
			)
	}

	return result, nil
}

func (r *Repository) AddressCountTx(
	ctx context.Context,
	tx pgx.Tx,
	customerID string,
) (int, error) {
	var count int

	err :=
		tx.QueryRow(
			ctx,
			`
				SELECT COUNT(*)::integer
				FROM customer_addresses
				WHERE customer_id = $1::uuid
			`,
			customerID,
		).Scan(
			&count,
		)
	if err != nil {
		return 0,
			fmt.Errorf(
				"count customer addresses: %w",
				err,
			)
	}

	return count, nil
}

func (r *Repository) ClearDefaultAddressesTx(
	ctx context.Context,
	tx pgx.Tx,
	customerID string,
) error {
	_, err :=
		tx.Exec(
			ctx,
			`
				UPDATE customer_addresses
				SET
					is_default = false,
					updated_at = now()
				WHERE
					customer_id = $1::uuid
					AND is_default = true
			`,
			customerID,
		)
	if err != nil {
		return fmt.Errorf(
			"clear default customer address: %w",
			err,
		)
	}

	return nil
}

func (r *Repository) EnsureDefaultAddressTx(
	ctx context.Context,
	tx pgx.Tx,
	customerID string,
) error {
	_, err :=
		tx.Exec(
			ctx,
			`
				UPDATE customer_addresses
				SET
					is_default = true,
					updated_at = now()
				WHERE
					id = (
						SELECT id
						FROM customer_addresses
						WHERE customer_id = $1::uuid
						ORDER BY
							created_at ASC,
							id ASC
						LIMIT 1
					)
					AND NOT EXISTS (
						SELECT 1
						FROM customer_addresses
						WHERE
							customer_id = $1::uuid
							AND is_default = true
					)
			`,
			customerID,
		)
	if err != nil {
		return fmt.Errorf(
			"ensure default customer address: %w",
			err,
		)
	}

	return nil
}

func (r *Repository) CreateAddressTx(
	ctx context.Context,
	tx pgx.Tx,
	customerID string,
	address Address,
) (Address, error) {
	row :=
		tx.QueryRow(
			ctx,
			`
				INSERT INTO customer_addresses (
					customer_id,
					label,
					recipient_name,
					phone,
					address_line1,
					address_line2,
					city,
					area,
					postal_code,
					is_default,
					created_at,
					updated_at
				)
				VALUES (
					$1::uuid,
					$2,
					$3,
					$4,
					$5,
					NULLIF($6, ''),
					$7,
					$8,
					NULLIF($9, ''),
					$10,
					now(),
					now()
				)
				RETURNING
					id::text,
					customer_id::text,
					label,
					recipient_name,
					phone,
					address_line1,
					COALESCE(address_line2, ''),
					city,
					area,
					COALESCE(postal_code, ''),
					is_default,
					created_at,
					updated_at
			`,
			customerID,
			address.Label,
			address.RecipientName,
			address.Phone,
			address.AddressLine1,
			address.AddressLine2,
			address.City,
			address.Area,
			address.PostalCode,
			address.IsDefault,
		)

	result, err :=
		scanAddress(
			row,
		)
	if err != nil {
		return Address{},
			fmt.Errorf(
				"create customer address: %w",
				err,
			)
	}

	return result, nil
}

func (r *Repository) UpdateAddressTx(
	ctx context.Context,
	tx pgx.Tx,
	address Address,
) (Address, error) {
	row :=
		tx.QueryRow(
			ctx,
			`
				UPDATE customer_addresses
				SET
					label = $3,
					recipient_name = $4,
					phone = $5,
					address_line1 = $6,
					address_line2 = NULLIF($7, ''),
					city = $8,
					area = $9,
					postal_code = NULLIF($10, ''),
					is_default = $11,
					updated_at = now()
				WHERE
					id = $2::uuid
					AND customer_id = $1::uuid
				RETURNING
					id::text,
					customer_id::text,
					label,
					recipient_name,
					phone,
					address_line1,
					COALESCE(address_line2, ''),
					city,
					area,
					COALESCE(postal_code, ''),
					is_default,
					created_at,
					updated_at
			`,
			address.CustomerID,
			address.ID,
			address.Label,
			address.RecipientName,
			address.Phone,
			address.AddressLine1,
			address.AddressLine2,
			address.City,
			address.Area,
			address.PostalCode,
			address.IsDefault,
		)

	result, err :=
		scanAddress(
			row,
		)
	if err != nil {
		if errors.Is(
			err,
			pgx.ErrNoRows,
		) {
			return Address{},
				ErrAddressNotFound
		}

		return Address{},
			fmt.Errorf(
				"update customer address: %w",
				err,
			)
	}

	return result, nil
}

func (r *Repository) DeleteAddressTx(
	ctx context.Context,
	tx pgx.Tx,
	customerID string,
	addressID string,
) error {
	tag, err :=
		tx.Exec(
			ctx,
			`
				DELETE FROM customer_addresses
				WHERE
					id = $2::uuid
					AND customer_id = $1::uuid
			`,
			customerID,
			addressID,
		)
	if err != nil {
		return fmt.Errorf(
			"delete customer address: %w",
			err,
		)
	}

	if tag.RowsAffected() == 0 {
		return ErrAddressNotFound
	}

	return nil
}

func scanAddress(
	row pgx.Row,
) (Address, error) {
	var result Address

	err :=
		row.Scan(
			&result.ID,
			&result.CustomerID,
			&result.Label,
			&result.RecipientName,
			&result.Phone,
			&result.AddressLine1,
			&result.AddressLine2,
			&result.City,
			&result.Area,
			&result.PostalCode,
			&result.IsDefault,
			&result.CreatedAt,
			&result.UpdatedAt,
		)
	if err != nil {
		return Address{},
			err
	}

	return result, nil
}
