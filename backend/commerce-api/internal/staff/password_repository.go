package staff

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

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
			"update staff password hash: %w",
			err,
		)
	}

	if tag.RowsAffected() != 1 {
		return fmt.Errorf(
			"update staff password hash: expected one staff account, updated %d",
			tag.RowsAffected(),
		)
	}

	return nil
}
