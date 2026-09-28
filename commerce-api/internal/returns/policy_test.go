package returns

import "testing"

func TestOrderEligibleForReturn(
	t *testing.T,
) {
	if !orderEligibleForReturn(
		"delivered",
	) {
		t.Fatal(
			"delivered order should be returnable",
		)
	}

	if !orderEligibleForReturn(
		"completed",
	) {
		t.Fatal(
			"completed order should be returnable",
		)
	}

	for _, status := range []string{
		"pending_payment",
		"confirmed",
		"processing",
		"shipped",
		"cancelled",
		"payment_expired",
	} {
		if orderEligibleForReturn(
			status,
		) {
			t.Fatalf(
				"status %q should not be returnable",
				status,
			)
		}
	}
}

func TestExpectedSourceStatus(
	t *testing.T,
) {
	tests := []struct {
		target string
		source string
		valid  bool
	}{
		{
			target: StatusApproved,
			source: StatusRequested,
			valid:  true,
		},
		{
			target: StatusRejected,
			source: StatusRequested,
			valid:  true,
		},
		{
			target: StatusReceived,
			source: StatusApproved,
			valid:  true,
		},
		{
			target: StatusInspected,
			source: StatusReceived,
			valid:  true,
		},
		{
			target: StatusCompleted,
			source: StatusInspected,
			valid:  true,
		},
	}

	for _, test := range tests {
		source, valid :=
			expectedSourceStatus(
				test.target,
			)

		if valid != test.valid ||
			source != test.source {
			t.Fatalf(
				"target=%q expected source=%q valid=%v; got source=%q valid=%v",
				test.target,
				test.source,
				test.valid,
				source,
				valid,
			)
		}
	}
}
