package config

import (
	"testing"
	"time"
)

func TestLoadRuntimeTickConfigDisabledDefaults(
	t *testing.T,
) {
	t.Setenv(
		"RUNTIME_TICK_ENABLED",
		"false",
	)

	t.Setenv(
		"RUNTIME_TICK_TOKEN",
		"",
	)

	cfg, err :=
		LoadRuntimeTickConfig()
	if err != nil {
		t.Fatalf(
			"load runtime tick config: %v",
			err,
		)
	}

	if cfg.Enabled {
		t.Fatal(
			"expected runtime tick to be disabled",
		)
	}

	if cfg.MaxMessages !=
		DefaultRuntimeTickMaxMessages {

		t.Fatalf(
			"unexpected max messages %d",
			cfg.MaxMessages,
		)
	}

	if cfg.Timeout !=
		DefaultRuntimeTickTimeout {

		t.Fatalf(
			"unexpected timeout %s",
			cfg.Timeout,
		)
	}

	if cfg.LockTTL !=
		DefaultRuntimeTickLockTTL {

		t.Fatalf(
			"unexpected lock TTL %s",
			cfg.LockTTL,
		)
	}
}

func TestLoadRuntimeTickConfigEnabled(
	t *testing.T,
) {
	t.Setenv(
		"RUNTIME_TICK_ENABLED",
		"true",
	)

	t.Setenv(
		"RUNTIME_TICK_TOKEN",
		"0123456789abcdef0123456789abcdef",
	)

	t.Setenv(
		"RUNTIME_TICK_MAX_MESSAGES",
		"24",
	)

	t.Setenv(
		"RUNTIME_TICK_TIMEOUT",
		"15s",
	)

	t.Setenv(
		"RUNTIME_TICK_LOCK_TTL",
		"25s",
	)

	cfg, err :=
		LoadRuntimeTickConfig()
	if err != nil {
		t.Fatalf(
			"load runtime tick config: %v",
			err,
		)
	}

	if !cfg.Enabled {
		t.Fatal(
			"expected runtime tick to be enabled",
		)
	}

	if cfg.MaxMessages != 24 {
		t.Fatalf(
			"unexpected max messages %d",
			cfg.MaxMessages,
		)
	}

	if cfg.Timeout !=
		15*time.Second {

		t.Fatalf(
			"unexpected timeout %s",
			cfg.Timeout,
		)
	}

	if cfg.LockTTL !=
		25*time.Second {

		t.Fatalf(
			"unexpected lock TTL %s",
			cfg.LockTTL,
		)
	}
}

func TestLoadRuntimeTickConfigRequiresStrongToken(
	t *testing.T,
) {
	t.Setenv(
		"RUNTIME_TICK_ENABLED",
		"true",
	)

	t.Setenv(
		"RUNTIME_TICK_TOKEN",
		"too-short",
	)

	if _, err :=
		LoadRuntimeTickConfig(); err == nil {

		t.Fatal(
			"expected runtime tick token validation error",
		)
	}
}

func TestLoadRuntimeTickConfigRejectsInvalidMessageLimit(
	t *testing.T,
) {
	t.Setenv(
		"RUNTIME_TICK_ENABLED",
		"false",
	)

	t.Setenv(
		"RUNTIME_TICK_MAX_MESSAGES",
		"257",
	)

	if _, err :=
		LoadRuntimeTickConfig(); err == nil {

		t.Fatal(
			"expected max-message validation error",
		)
	}
}

func TestLoadRuntimeTickConfigRejectsExcessiveTimeout(
	t *testing.T,
) {
	t.Setenv(
		"RUNTIME_TICK_ENABLED",
		"false",
	)

	t.Setenv(
		"RUNTIME_TICK_TIMEOUT",
		"26s",
	)

	if _, err :=
		LoadRuntimeTickConfig(); err == nil {

		t.Fatal(
			"expected timeout validation error",
		)
	}
}

func TestLoadRuntimeTickConfigRequiresLockTTLAboveTimeout(
	t *testing.T,
) {
	t.Setenv(
		"RUNTIME_TICK_ENABLED",
		"false",
	)

	t.Setenv(
		"RUNTIME_TICK_TIMEOUT",
		"20s",
	)

	t.Setenv(
		"RUNTIME_TICK_LOCK_TTL",
		"20s",
	)

	if _, err :=
		LoadRuntimeTickConfig(); err == nil {

		t.Fatal(
			"expected lock TTL validation error",
		)
	}
}
