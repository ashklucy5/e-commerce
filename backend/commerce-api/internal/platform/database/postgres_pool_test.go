package database

import "testing"

func TestPostgresPoolLimitsDefault(
	t *testing.T,
) {
	maxConns, minConns :=
		postgresPoolLimits(
			false,
		)

	if maxConns !=
		defaultPostgresMaxConns {

		t.Fatalf(
			"max connections = %d, want %d",
			maxConns,
			defaultPostgresMaxConns,
		)
	}

	if minConns !=
		defaultPostgresMinConns {

		t.Fatalf(
			"min connections = %d, want %d",
			minConns,
			defaultPostgresMinConns,
		)
	}
}

func TestPostgresPoolLimitsServerless(
	t *testing.T,
) {
	maxConns, minConns :=
		postgresPoolLimits(
			true,
		)

	if maxConns !=
		serverlessPostgresMaxConns {

		t.Fatalf(
			"max connections = %d, want %d",
			maxConns,
			serverlessPostgresMaxConns,
		)
	}

	if minConns !=
		serverlessPostgresMinConns {

		t.Fatalf(
			"min connections = %d, want %d",
			minConns,
			serverlessPostgresMinConns,
		)
	}
}