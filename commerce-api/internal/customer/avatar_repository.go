package customer

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

type avatarState struct {
	StorageKey string
	UpdatedAt  *time.Time
}

func (r *Repository) GetAvatarState(
	ctx context.Context,
	customerID string,
) (avatarState, error) {
	var result avatarState
	var updatedAt *time.Time

	err :=
		r.db.QueryRow(
			ctx,
			`
				SELECT
					COALESCE(avatar_storage_key, ''),
					avatar_updated_at
				FROM customers
				WHERE id = $1::uuid
			`,
			customerID,
		).Scan(
			&result.StorageKey,
			&updatedAt,
		)
	if err != nil {
		if errors.Is(
			err,
			pgx.ErrNoRows,
		) {
			return avatarState{},
				ErrCustomerNotFound
		}

		return avatarState{},
			fmt.Errorf(
				"get customer avatar state: %w",
				err,
			)
	}

	result.UpdatedAt =
		updatedAt

	return result, nil
}

func (r *Repository) SetAvatarTx(
	ctx context.Context,
	tx pgx.Tx,
	customerID string,
	storageKey string,
) (string, time.Time, error) {
	var previousStorageKey string

	err :=
		tx.QueryRow(
			ctx,
			`
				SELECT
					COALESCE(avatar_storage_key, '')
				FROM customers
				WHERE id = $1::uuid
				FOR UPDATE
			`,
			customerID,
		).Scan(
			&previousStorageKey,
		)
	if err != nil {
		if errors.Is(
			err,
			pgx.ErrNoRows,
		) {
			return "",
				time.Time{},
				ErrCustomerNotFound
		}

		return "",
			time.Time{},
			fmt.Errorf(
				"lock customer avatar: %w",
				err,
			)
	}

	var updatedAt time.Time

	err =
		tx.QueryRow(
			ctx,
			`
				UPDATE customers
				SET
					avatar_storage_key = $2,
					avatar_updated_at = now(),
					updated_at = now()
				WHERE id = $1::uuid
				RETURNING avatar_updated_at
			`,
			customerID,
			storageKey,
		).Scan(
			&updatedAt,
		)
	if err != nil {
		return "",
			time.Time{},
			fmt.Errorf(
				"set customer avatar: %w",
				err,
			)
	}

	return previousStorageKey,
		updatedAt,
		nil
}

func (r *Repository) ClearAvatarTx(
	ctx context.Context,
	tx pgx.Tx,
	customerID string,
) (string, error) {
	var previousStorageKey string

	err :=
		tx.QueryRow(
			ctx,
			`
				SELECT
					COALESCE(avatar_storage_key, '')
				FROM customers
				WHERE id = $1::uuid
				FOR UPDATE
			`,
			customerID,
		).Scan(
			&previousStorageKey,
		)
	if err != nil {
		if errors.Is(
			err,
			pgx.ErrNoRows,
		) {
			return "",
				ErrCustomerNotFound
		}

		return "",
			fmt.Errorf(
				"lock customer avatar for deletion: %w",
				err,
			)
	}

	if previousStorageKey == "" {
		return "", nil
	}

	_, err =
		tx.Exec(
			ctx,
			`
				UPDATE customers
				SET
					avatar_storage_key = NULL,
					avatar_updated_at = NULL,
					updated_at = now()
				WHERE id = $1::uuid
			`,
			customerID,
		)
	if err != nil {
		return "",
			fmt.Errorf(
				"clear customer avatar: %w",
				err,
			)
	}

	return previousStorageKey,
		nil
}
