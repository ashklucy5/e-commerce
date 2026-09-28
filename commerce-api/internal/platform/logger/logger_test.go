package logger

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
)

func TestParseLevel(
	t *testing.T,
) {
	cases :=
		map[string]slog.Level{
			"debug": slog.LevelDebug,

			"INFO": slog.LevelInfo,

			"warning": slog.LevelWarn,

			"error": slog.LevelError,

			"unknown": slog.LevelInfo,
		}

	for input, expected := range cases {

		actual :=
			ParseLevel(
				input,
			)

		if actual != expected {
			t.Fatalf(
				"ParseLevel(%q): expected %v, got %v",
				input,
				expected,
				actual,
			)
		}
	}
}

func TestContextLogger(
	t *testing.T,
) {
	var buffer bytes.Buffer

	value :=
		New(
			Config{
				Level: "debug",

				Format: "json",

				Writer: &buffer,
			},
		)

	ctx :=
		IntoContext(
			context.Background(),
			value,
		)

	FromContext(
		ctx,
	).Info(
		"context logger works",
	)

	if !strings.Contains(
		buffer.String(),
		"context logger works",
	) {
		t.Fatalf(
			"expected log output, got %q",
			buffer.String(),
		)
	}
}
