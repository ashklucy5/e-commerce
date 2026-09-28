package logger

import (
	"context"
	"log/slog"
)

type contextKey struct{}

func IntoContext(
	ctx context.Context,
	value *slog.Logger,
) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}

	if value == nil {
		value = slog.Default()
	}

	return context.WithValue(
		ctx,
		contextKey{},
		value,
	)
}

func FromContext(
	ctx context.Context,
) *slog.Logger {
	if ctx == nil {
		return slog.Default()
	}

	value, ok :=
		ctx.Value(
			contextKey{},
		).(*slog.Logger)

	if !ok ||
		value == nil {
		return slog.Default()
	}

	return value
}

func With(
	ctx context.Context,
	args ...any,
) *slog.Logger {
	return FromContext(
		ctx,
	).With(
		args...,
	)
}
