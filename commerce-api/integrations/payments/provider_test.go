package payments

import (
	"context"
	"errors"
	"net/http"
	"testing"
)

type fakeProvider struct {
	code string

	readyErr error
}

func (p *fakeProvider) Code() string {
	return p.code
}

func (p *fakeProvider) Ready() error {
	return p.readyErr
}

func (p *fakeProvider) CreatePayment(
	context.Context,
	CreatePaymentRequest,
) (CreatePaymentResult, error) {
	return CreatePaymentResult{},
		nil
}

func (p *fakeProvider) QueryPayment(
	context.Context,
	QueryPaymentRequest,
) (QueryPaymentResult, error) {
	return QueryPaymentResult{},
		nil
}

func (p *fakeProvider) VerifyWebhook(
	context.Context,
	WebhookRequest,
) (VerifiedWebhookEvent, error) {
	return VerifiedWebhookEvent{},
		nil
}

func TestRegistryDisabledProvider(
	t *testing.T,
) {
	registry :=
		NewRegistry(
			Enablement{
				BKash: false,
			},
		)

	err :=
		registry.Register(
			&fakeProvider{
				code: ProviderBKash,
			},
		)
	if err != nil {
		t.Fatalf(
			"register provider: %v",
			err,
		)
	}

	_, err =
		registry.Resolve(
			ProviderBKash,
		)

	if !errors.Is(
		err,
		ErrProviderDisabled,
	) {
		t.Fatalf(
			"expected disabled provider, got %v",
			err,
		)
	}
}

func TestRegistryEnabledButMissingAdapter(
	t *testing.T,
) {
	registry :=
		NewRegistry(
			Enablement{
				BKash: true,
			},
		)

	_, err :=
		registry.Resolve(
			ProviderBKash,
		)

	if !errors.Is(
		err,
		ErrProviderUnavailable,
	) {
		t.Fatalf(
			"expected unavailable provider, got %v",
			err,
		)
	}
}

func TestRegistryEnabledButNotConfigured(
	t *testing.T,
) {
	registry :=
		NewRegistry(
			Enablement{
				BKash: true,
			},
		)

	err :=
		registry.Register(
			&fakeProvider{
				code: ProviderBKash,

				readyErr: errors.New(
					"missing credentials",
				),
			},
		)
	if err != nil {
		t.Fatalf(
			"register provider: %v",
			err,
		)
	}

	_, err =
		registry.Resolve(
			ProviderBKash,
		)

	if !errors.Is(
		err,
		ErrProviderNotConfigured,
	) {
		t.Fatalf(
			"expected not-configured provider, got %v",
			err,
		)
	}
}

func TestRegistryAvailableProvider(
	t *testing.T,
) {
	registry :=
		NewRegistry(
			Enablement{
				BKash: true,
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

	resolved, err :=
		registry.Resolve(
			ProviderBKash,
		)
	if err != nil {
		t.Fatalf(
			"resolve provider: %v",
			err,
		)
	}

	if resolved != provider {
		t.Fatal(
			"registry returned wrong provider",
		)
	}

	if !registry.Available(
		ProviderBKash,
	) {
		t.Fatal(
			"provider should be available",
		)
	}
}

func TestRegistryRejectsDuplicateProvider(
	t *testing.T,
) {
	registry :=
		NewRegistry(
			Enablement{},
		)

	provider :=
		&fakeProvider{
			code: ProviderNagad,
		}

	if err :=
		registry.Register(
			provider,
		); err != nil {

		t.Fatalf(
			"first registration: %v",
			err,
		)
	}

	err :=
		registry.Register(
			provider,
		)

	if !errors.Is(
		err,
		ErrProviderAlreadyRegistered,
	) {
		t.Fatalf(
			"expected duplicate error, got %v",
			err,
		)
	}
}

func TestRegistryStatus(
	t *testing.T,
) {
	registry :=
		NewRegistry(
			Enablement{
				BKash: true,

				Nagad: false,
			},
		)

	if err :=
		registry.Register(
			&fakeProvider{
				code: ProviderBKash,
			},
		); err != nil {

		t.Fatalf(
			"register bkash: %v",
			err,
		)
	}

	bkash :=
		registry.Status(
			ProviderBKash,
		)

	if !bkash.Enabled ||
		!bkash.Registered ||
		!bkash.Configured ||
		!bkash.Available {

		t.Fatalf(
			"unexpected bkash status: %#v",
			bkash,
		)
	}

	nagad :=
		registry.Status(
			ProviderNagad,
		)

	if nagad.Enabled ||
		nagad.Registered ||
		nagad.Configured ||
		nagad.Available {

		t.Fatalf(
			"unexpected nagad status: %#v",
			nagad,
		)
	}
}

func TestWebhookRequestSupportsHTTPHeaders(
	t *testing.T,
) {
	request :=
		WebhookRequest{
			Headers: http.Header{
				"X-Test-Signature": []string{
					"signature",
				},
			},

			Body: []byte(
				`{"status":"success"}`,
			),
		}

	if request.Headers.Get(
		"X-Test-Signature",
	) != "signature" {

		t.Fatal(
			"webhook headers not preserved",
		)
	}
}
