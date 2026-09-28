package payment

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	providerpayments "project.local/commerce-api/integrations/payments"
)

type fakeWebhookProvider struct {
	code string

	event providerpayments.VerifiedWebhookEvent

	verifyErr error

	calls int
}

func (p *fakeWebhookProvider) Code() string {
	return p.code
}

func (p *fakeWebhookProvider) Ready() error {
	return nil
}

func (p *fakeWebhookProvider) CreatePayment(
	context.Context,
	providerpayments.CreatePaymentRequest,
) (providerpayments.CreatePaymentResult, error) {
	return providerpayments.CreatePaymentResult{},
		nil
}

func (p *fakeWebhookProvider) QueryPayment(
	context.Context,
	providerpayments.QueryPaymentRequest,
) (providerpayments.QueryPaymentResult, error) {
	return providerpayments.QueryPaymentResult{},
		nil
}

func (p *fakeWebhookProvider) VerifyWebhook(
	context.Context,
	providerpayments.WebhookRequest,
) (providerpayments.VerifiedWebhookEvent, error) {
	p.calls++

	if p.verifyErr != nil {
		return providerpayments.VerifiedWebhookEvent{},
			p.verifyErr
	}

	return p.event,
		nil
}

type fakePaymentConfirmer struct {
	calls int

	input ConfirmVerifiedPaymentInput

	result ConfirmVerifiedPaymentResult

	err error
}

func (c *fakePaymentConfirmer) ConfirmVerifiedPayment(
	_ context.Context,
	input ConfirmVerifiedPaymentInput,
) (ConfirmVerifiedPaymentResult, error) {
	c.calls++

	c.input =
		input

	return c.result,
		c.err
}

func testVerifiedSuccessEvent() providerpayments.VerifiedWebhookEvent {
	paidAt :=
		time.Date(
			2026,
			8,
			28,
			8,
			0,
			0,
			0,
			time.UTC,
		)

	return providerpayments.VerifiedWebhookEvent{
		ProviderEventID: "evt-1",

		EventType: "payment.succeeded",

		OrderID: "11111111-1111-4111-8111-111111111111",

		ProviderPaymentID: "provider-payment-1",

		ProviderTransactionID: "provider-transaction-1",

		Status: providerpayments.StatusSucceeded,

		Amount: 250000,

		Currency: "BDT",

		PaidAt: &paidAt,
	}
}

func registryWithProvider(
	t *testing.T,
	provider *fakeWebhookProvider,
) *providerpayments.Registry {
	t.Helper()

	/*
		All enablement stays false deliberately.

		The webhook path must still process a previously initiated
		payment through ResolveForProcessing().
	*/
	registry :=
		providerpayments.NewRegistry(
			providerpayments.Enablement{},
		)

	if err :=
		registry.Register(
			provider,
		); err != nil {

		t.Fatalf(
			"register provider: %v",
			err,
		)
	}

	return registry
}

func TestWebhookUsesProcessingResolutionAndHashesRawBody(
	t *testing.T,
) {
	provider :=
		&fakeWebhookProvider{
			code: providerpayments.ProviderBKash,

			event: testVerifiedSuccessEvent(),
		}

	confirmer :=
		&fakePaymentConfirmer{
			result: ConfirmVerifiedPaymentResult{
				Processed: true,

				Payment: Payment{
					ID: "payment-1",
				},
			},
		}

	service :=
		NewWebhookService(
			registryWithProvider(
				t,
				provider,
			),
			confirmer,
		)

	body :=
		[]byte(
			`{"payment":"verified"}`,
		)

	result, err :=
		service.Process(
			context.Background(),
			"bkash",
			http.Header{},
			body,
		)
	if err != nil {
		t.Fatalf(
			"process webhook: %v",
			err,
		)
	}

	if provider.calls != 1 {
		t.Fatalf(
			"expected one verify call, got %d",
			provider.calls,
		)
	}

	if confirmer.calls != 1 {
		t.Fatalf(
			"expected one confirm call, got %d",
			confirmer.calls,
		)
	}

	if confirmer.input.PayloadSHA256 !=
		PayloadSHA256(
			body,
		) {

		t.Fatalf(
			"unexpected payload hash %q",
			confirmer.input.PayloadSHA256,
		)
	}

	if !result.Acknowledged ||
		!result.Processed ||
		result.Replayed {

		t.Fatalf(
			"unexpected result: %#v",
			result,
		)
	}
}

func TestWebhookNeverConfirmsBeforeVerification(
	t *testing.T,
) {
	provider :=
		&fakeWebhookProvider{
			code: providerpayments.ProviderNagad,

			verifyErr: providerpayments.ErrWebhookVerificationFailed,
		}

	confirmer :=
		&fakePaymentConfirmer{}

	service :=
		NewWebhookService(
			registryWithProvider(
				t,
				provider,
			),
			confirmer,
		)

	_, err :=
		service.Process(
			context.Background(),
			"nagad",
			http.Header{},
			[]byte(
				`{"event":"x"}`,
			),
		)

	if !errors.Is(
		err,
		providerpayments.ErrWebhookVerificationFailed,
	) {
		t.Fatalf(
			"expected verification failure, got %v",
			err,
		)
	}

	if confirmer.calls != 0 {
		t.Fatalf(
			"unverified webhook reached payment confirmer",
		)
	}
}

func TestWebhookAcknowledgesVerifiedNonSuccessWithoutMarkingPaid(
	t *testing.T,
) {
	event :=
		testVerifiedSuccessEvent()

	event.Status =
		providerpayments.StatusFailed

	provider :=
		&fakeWebhookProvider{
			code: providerpayments.ProviderRocket,

			event: event,
		}

	confirmer :=
		&fakePaymentConfirmer{}

	service :=
		NewWebhookService(
			registryWithProvider(
				t,
				provider,
			),
			confirmer,
		)

	result, err :=
		service.Process(
			context.Background(),
			"rocket",
			http.Header{},
			[]byte(
				`{"event":"failed"}`,
			),
		)
	if err != nil {
		t.Fatalf(
			"process webhook: %v",
			err,
		)
	}

	if !result.Acknowledged ||
		!result.Ignored ||
		result.Processed {

		t.Fatalf(
			"unexpected result: %#v",
			result,
		)
	}

	if confirmer.calls != 0 {
		t.Fatalf(
			"non-success webhook marked payment as paid",
		)
	}
}

func TestWebhookAcknowledgesDeterministicConflictForReconciliation(
	t *testing.T,
) {
	provider :=
		&fakeWebhookProvider{
			code: providerpayments.ProviderBankTransfer,

			event: testVerifiedSuccessEvent(),
		}

	confirmer :=
		&fakePaymentConfirmer{
			err: ErrPaymentAmountMismatch,
		}

	service :=
		NewWebhookService(
			registryWithProvider(
				t,
				provider,
			),
			confirmer,
		)

	result, err :=
		service.Process(
			context.Background(),
			"bank_transfer",
			http.Header{},
			[]byte(
				`{"event":"mismatch"}`,
			),
		)
	if err != nil {
		t.Fatalf(
			"deterministic conflict should be acknowledged: %v",
			err,
		)
	}

	if !result.Acknowledged ||
		!result.ReconciliationRequired ||
		result.Replayed {

		t.Fatalf(
			"unexpected result: %#v",
			result,
		)
	}
}
