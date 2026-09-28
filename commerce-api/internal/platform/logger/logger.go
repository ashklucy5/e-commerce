package logger

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

type Config struct {
	Level     string
	Format    string
	AddSource bool
	Writer    io.Writer
}

func New(
	cfg Config,
) *slog.Logger {
	writer := cfg.Writer
	if writer == nil {
		writer = os.Stdout
	}

	options := &slog.HandlerOptions{
		AddSource: cfg.AddSource,
		Level:     ParseLevel(cfg.Level),
	}

	var handler slog.Handler

	switch strings.ToLower(
		strings.TrimSpace(
			cfg.Format,
		),
	) {
	case "text":
		handler = slog.NewTextHandler(
			writer,
			options,
		)

	default:
		handler = slog.NewJSONHandler(
			writer,
			options,
		)
	}

	return slog.New(
		handler,
	)
}

func ParseLevel(
	value string,
) slog.Level {
	switch strings.ToLower(
		strings.TrimSpace(
			value,
		),
	) {
	case "debug":
		return slog.LevelDebug

	case "warn",
		"warning":
		return slog.LevelWarn

	case "error":
		return slog.LevelError

	default:
		return slog.LevelInfo
	}
}
