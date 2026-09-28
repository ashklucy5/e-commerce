package returns

import (
	"errors"
	"testing"
)

func TestNormalizeReturnHistoryStatus(
	t *testing.T,
) {
	tests :=
		[]struct {
			name     string
			input    string
			expected string
		}{
			{
				name:     "empty means all",
				input:    "",
				expected: "",
			},
			{
				name:     "explicit all",
				input:    "all",
				expected: "",
			},
			{
				name:     "normalizes requested",
				input:    " Requested ",
				expected: StatusRequested,
			},
			{
				name:     "accepts completed",
				input:    StatusCompleted,
				expected: StatusCompleted,
			},
			{
				name:     "accepts cancelled",
				input:    StatusCancelled,
				expected: StatusCancelled,
			},
		}

	for _, test := range tests {

		t.Run(
			test.name,
			func(
				t *testing.T,
			) {
				actual, err :=
					NormalizeReturnHistoryStatus(
						test.input,
					)
				if err != nil {
					t.Fatalf(
						"normalize return status: %v",
						err,
					)
				}

				if actual !=
					test.expected {

					t.Fatalf(
						"expected %q, got %q",
						test.expected,
						actual,
					)
				}
			},
		)
	}
}

func TestNormalizeReturnHistoryStatusRejectsInvalid(
	t *testing.T,
) {
	_, err :=
		NormalizeReturnHistoryStatus(
			"definitely_not_a_return_status",
		)

	if !errors.Is(
		err,
		ErrInvalidReturnHistoryStatus,
	) {
		t.Fatalf(
			"expected ErrInvalidReturnHistoryStatus, got %v",
			err,
		)
	}
}
