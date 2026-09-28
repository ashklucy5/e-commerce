package config

import "testing"

func TestRuntimeHTTPAddrUsesDefault(
	t *testing.T,
) {
	t.Setenv(
		"PORT",
		"",
	)

	t.Setenv(
		"HTTP_ADDR",
		"",
	)

	if actual :=
		runtimeHTTPAddr(); actual !=
		":8080" {

		t.Fatalf(
			"expected :8080, got %q",
			actual,
		)
	}
}

func TestRuntimeHTTPAddrUsesHTTPAddr(
	t *testing.T,
) {
	t.Setenv(
		"PORT",
		"",
	)

	t.Setenv(
		"HTTP_ADDR",
		":9090",
	)

	if actual :=
		runtimeHTTPAddr(); actual !=
		":9090" {

		t.Fatalf(
			"expected :9090, got %q",
			actual,
		)
	}
}

func TestRuntimeHTTPAddrPrefersPort(
	t *testing.T,
) {
	t.Setenv(
		"PORT",
		"3210",
	)

	t.Setenv(
		"HTTP_ADDR",
		":9090",
	)

	if actual :=
		runtimeHTTPAddr(); actual !=
		":3210" {

		t.Fatalf(
			"expected :3210, got %q",
			actual,
		)
	}
}
