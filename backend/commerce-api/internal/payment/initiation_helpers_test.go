package payment

import (
	"testing"
	"time"

	paymentintegrations "project.local/commerce-api/integrations/payments"
)

func TestPaymentAttemptKeysAreUnique(
	t *testing.T,
) {
	seen :=
		make(
			map[string]struct{},
			100,
		)

	for i :=
		0; i < 100; i++ {

		key, err :=
			newPaymentAttemptKey()
		if err != nil {
			t.Fatalf(
				"generate attempt key: %v",
				err,
			)
		}

		if len(key) > 100 {
			t.Fatalf(
				"attempt key too long: %d",
				len(key),
			)
		}

		if _, exists :=
			seen[key]; exists {

			t.Fatalf(
				"duplicate attempt key: %s",
				key,
			)
		}

		seen[key] =
			struct{}{}
	}
}

func TestWalletProviderRequiresHandoffURL(
	t *testing.T,
) {
	now :=
		time.Now().
			UTC()

	_, err :=
		normalizeProviderCreateResult(
			paymentintegrations.ProviderBKash,
			now,
			paymentintegrations.CreatePaymentResult{
				ProviderPaymentID: "provider-payment-1",
			},
		)

	if err == nil {
		t.Fatal(
			"wallet payment without handoff URL should fail",
		)
	}
}

func TestProviderCreateResultValidation(
	t *testing.T,
) {
	now :=
		time.Now().
			UTC()

	expiry :=
		now.Add(
			15 * time.Minute,
		)

	result, err :=
		normalizeProviderCreateResult(
			paymentintegrations.ProviderBKash,
			now,
			paymentintegrations.CreatePaymentResult{
				ProviderPaymentID: "provider-payment-1",

				RedirectURL: "https://example.com/pay/123",

				ExpiresAt: &expiry,
			},
		)
	if err != nil {
		t.Fatalf(
			"validate provider result: %v",
			err,
		)
	}

	if result.ProviderPaymentID !=
		"provider-payment-1" {

		t.Fatal(
			"provider payment ID mismatch",
		)
	}
}
