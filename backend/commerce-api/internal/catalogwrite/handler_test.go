package catalogwrite

import (
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestIsCatalogWriteConflict(
	t *testing.T,
) {
	t.Run(
		"unique violation",
		func(
			t *testing.T,
		) {
			err := fmt.Errorf(
				"create product: %w",
				&pgconn.PgError{
					Code: "23505",
				},
			)

			if !isCatalogWriteConflict(
				err,
			) {
				t.Fatal(
					"expected PostgreSQL unique violation to be a catalog conflict",
				)
			}
		},
	)

	t.Run(
		"check violation",
		func(
			t *testing.T,
		) {
			err := &pgconn.PgError{
				Code: "23514",
			}

			if isCatalogWriteConflict(
				err,
			) {
				t.Fatal(
					"did not expect PostgreSQL check violation to be a catalog conflict",
				)
			}
		},
	)
}
