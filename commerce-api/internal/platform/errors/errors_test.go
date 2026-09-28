package errors

import (
	stderrors "errors"
	"net/http"
	"testing"
)

func TestUnknownErrorBecomesSafeInternalError(
	t *testing.T,
) {
	internal := stderrors.New(
		"database password should never reach client",
	)

	status, response := ResponseFor(
		internal,
	)

	if status != http.StatusInternalServerError {
		t.Fatalf(
			"expected status 500, got %d",
			status,
		)
	}

	if response.Error.Code != CodeInternal {
		t.Fatalf(
			"expected code %q, got %q",
			CodeInternal,
			response.Error.Code,
		)
	}

	if response.Error.Message != "Internal server error" {
		t.Fatalf(
			"unexpected public message %q",
			response.Error.Message,
		)
	}
}

func TestNormalizePreservesAPIError(
	t *testing.T,
) {
	expected := Conflict(
		"ORDER_CONFLICT",
		"Order can no longer be changed",
	)

	actual := Normalize(
		expected,
	)

	if actual != expected {
		t.Fatal(
			"expected existing APIError to be preserved",
		)
	}
}

func TestWithDetailsDoesNotMutateOriginal(
	t *testing.T,
) {
	base := BadRequest(
		"INVALID_REQUEST",
		"Invalid request",
	)

	withDetails := base.WithDetails(
		map[string]string{
			"field": "name",
		},
	)

	if base.Details != nil {
		t.Fatal(
			"expected original error to remain unchanged",
		)
	}

	if withDetails.Details == nil {
		t.Fatal(
			"expected copied error to contain details",
		)
	}
}
