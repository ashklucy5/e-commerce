package analytics

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const analyticsQueryTimeout = 5 * time.Second

type Service struct {
	db *pgxpool.Pool
}

func NewService(
	db *pgxpool.Pool,
) *Service {
	return &Service{
		db: db,
	}
}

func (s *Service) queryContext(
	ctx context.Context,
) (
	context.Context,
	context.CancelFunc,
	error,
) {
	if s == nil ||
		s.db == nil {

		return nil,
			nil,
			fmt.Errorf(
				"analytics database is not configured",
			)
	}

	queryCtx, cancel :=
		context.WithTimeout(
			ctx,
			analyticsQueryTimeout,
		)

	return queryCtx,
		cancel,
		nil
}

func roundedAverageAmount(
	total int64,
	count int64,
) int64 {
	if count <= 0 {
		return 0
	}

	if total >= 0 {
		return (total +
			count/2) /
			count
	}

	return (total -
		count/2) /
		count
}

func basisPoints(
	numerator int64,
	denominator int64,
) int64 {
	if numerator <= 0 ||
		denominator <= 0 {

		return 0
	}

	if numerator >=
		denominator {

		return 10000
	}

	return (numerator*10000 +
		denominator/2) /
		denominator
}
