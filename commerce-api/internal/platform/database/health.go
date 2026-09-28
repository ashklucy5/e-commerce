package database

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

func Healthy(
	ctx context.Context,
	db *pgxpool.Pool,
) bool {
	return db.Ping(ctx) == nil
}
