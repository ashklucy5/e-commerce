package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

const rollbackTimeout = 5 * time.Second

var (
	ErrNilTransactionStarter = errors.New(
		"database transaction starter is required",
	)

	ErrNilTransactionFunc = errors.New(
		"database transaction function is required",
	)
)

// TransactionStarter is implemented by pgxpool.Pool and other
// pgx types capable of starting a database transaction.
//
// Keeping this as a small interface makes the transaction helper
// usable without coupling callers directly to pgxpool.Pool.
type TransactionStarter interface {
	BeginTx(
		ctx context.Context,
		txOptions pgx.TxOptions,
	) (pgx.Tx, error)
}

// TransactionFunc is the work executed inside a database transaction.
//
// Returning an error causes the transaction to roll back.
// Returning nil causes the transaction to commit.
type TransactionFunc func(
	ctx context.Context,
	tx pgx.Tx,
) error

// WithinTx executes fn inside a transaction using default pgx
// transaction options.
//
// The transaction is:
//
//	begin
//	  -> callback
//	  -> rollback when callback fails
//	  -> commit when callback succeeds
//
// Panics are not swallowed. The helper attempts to roll back first,
// then re-panics with the original value.
func WithinTx(
	ctx context.Context,
	starter TransactionStarter,
	fn TransactionFunc,
) error {
	return WithinTxOptions(
		ctx,
		starter,
		pgx.TxOptions{},
		fn,
	)
}

// WithinTxOptions is the configurable form of WithinTx.
//
// Domain errors returned by fn are preserved so errors.Is/errors.As
// continue to work at higher layers.
func WithinTxOptions(
	ctx context.Context,
	starter TransactionStarter,
	options pgx.TxOptions,
	fn TransactionFunc,
) (err error) {
	if starter == nil {
		return ErrNilTransactionStarter
	}

	if fn == nil {
		return ErrNilTransactionFunc
	}

	tx, err := starter.BeginTx(
		ctx,
		options,
	)
	if err != nil {
		return fmt.Errorf(
			"begin database transaction: %w",
			err,
		)
	}

	defer func() {
		recovered := recover()
		if recovered != nil {
			_ = rollbackTransaction(
				tx,
			)

			panic(recovered)
		}

		if err == nil {
			return
		}

		rollbackErr := rollbackTransaction(
			tx,
		)
		if rollbackErr == nil {
			return
		}

		err = errors.Join(
			err,
			rollbackErr,
		)
	}()

	err = fn(
		ctx,
		tx,
	)
	if err != nil {
		return err
	}

	if err = tx.Commit(ctx); err != nil {
		err = fmt.Errorf(
			"commit database transaction: %w",
			err,
		)

		return err
	}

	return nil
}

// rollbackTransaction uses a short independent context.
//
// The original request context may already be cancelled or timed out.
// We still want PostgreSQL to receive the rollback attempt so the
// connection can return to the pool in a clean state.
func rollbackTransaction(
	tx pgx.Tx,
) error {
	ctx, cancel := context.WithTimeout(
		context.Background(),
		rollbackTimeout,
	)
	defer cancel()

	err := tx.Rollback(
		ctx,
	)
	if err == nil ||
		errors.Is(
			err,
			pgx.ErrTxClosed,
		) {
		return nil
	}

	return fmt.Errorf(
		"rollback database transaction: %w",
		err,
	)
}
