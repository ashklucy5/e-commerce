package security

import (
	"testing"
)

func TestCSRFTokenLifecycle(
	t *testing.T,
) {
	token, err :=
		NewCSRFToken()
	if err != nil {
		t.Fatalf(
			"generate CSRF token: %v",
			err,
		)
	}

	hash, err :=
		HashCSRFToken(
			token,
		)
	if err != nil {
		t.Fatalf(
			"hash CSRF token: %v",
			err,
		)
	}

	if !VerifyCSRFToken(
		token,
		hash,
	) {
		t.Fatal(
			"expected CSRF token verification to succeed",
		)
	}

	if VerifyCSRFToken(
		"different-token",
		hash,
	) {
		t.Fatal(
			"expected different CSRF token to fail",
		)
	}
}

func TestCSRFUsesIndependentRandomValues(
	t *testing.T,
) {
	first, err :=
		NewCSRFToken()
	if err != nil {
		t.Fatalf(
			"generate first CSRF token: %v",
			err,
		)
	}

	second, err :=
		NewCSRFToken()
	if err != nil {
		t.Fatalf(
			"generate second CSRF token: %v",
			err,
		)
	}

	if first == second {
		t.Fatal(
			"expected independently generated CSRF tokens",
		)
	}
}
