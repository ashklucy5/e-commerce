package order

import (
	"errors"
	"testing"
)

func TestNormalizeOrderHistoryStatus(
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
				name:     "normalizes confirmed",
				input:    " Confirmed ",
				expected: StatusConfirmed,
			},
			{
				name:     "accepts completed",
				input:    StatusCompleted,
				expected: StatusCompleted,
			},
		}

	for _, test := range tests {

		t.Run(
			test.name,
			func(
				t *testing.T,
			) {
				actual, err :=
					NormalizeOrderHistoryStatus(
						test.input,
					)
				if err != nil {
					t.Fatalf(
						"normalize order status: %v",
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

func TestNormalizeOrderHistoryStatusRejectsInvalid(
	t *testing.T,
) {
	_, err :=
		NormalizeOrderHistoryStatus(
			"definitely_not_an_order_status",
		)

	if !errors.Is(
		err,
		ErrInvalidOrderHistoryStatus,
	) {
		t.Fatalf(
			"expected ErrInvalidOrderHistoryStatus, got %v",
			err,
		)
	}
}
