package payments

import (
	"errors"
	"testing"
)

func TestDisabledProviderCanStillProcessExistingPayments(
	t *testing.T,
) {
	registry :=
		NewRegistry(
			Enablement{
				BKash: false,
			},
		)

	provider :=
		&fakeProvider{
			code: ProviderBKash,
		}

	if err :=
		registry.Register(
			provider,
		); err != nil {

		t.Fatalf(
			"register provider: %v",
			err,
		)
	}

	_, err :=
		registry.Resolve(
			ProviderBKash,
		)

	if !errors.Is(
		err,
		ErrProviderDisabled,
	) {
		t.Fatalf(
			"new initiation should be disabled, got %v",
			err,
		)
	}

	resolved, err :=
		registry.ResolveForProcessing(
			ProviderBKash,
		)
	if err != nil {
		t.Fatalf(
			"existing payment processing should remain possible: %v",
			err,
		)
	}

	if resolved != provider {
		t.Fatal(
			"wrong provider returned",
		)
	}
}
