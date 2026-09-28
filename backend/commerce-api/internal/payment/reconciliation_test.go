package payment

import (
	"context"
	"errors"
	"testing"
	"time"

	paymentintegrations "project.local/commerce-api/integrations/payments"
)

type fakeReconciliationStore struct {
	candidates []reconciliationCandidate

	deferred []string

	bound []struct {
		PaymentID string

		ProviderPaymentID string

		ProviderTransactionID string
	}

	terminal []reconciliationTerminalUpdate
}

func (s *fakeReconciliationStore) ListPendingReconciliationCandidates(
	context.Context,
	time.Time,
	int,
) ([]reconciliationCandidate, error) {
	return s.candidates,
		nil
}

func (s *fakeReconciliationStore) DeferReconciliation(
	_ context.Context,
	paymentID string,
) error {
	s.deferred =
		append(
			s.deferred,
			paymentID,
		)

	return nil
}

func (s *fakeReconciliationStore) BindReconciliationProviderIdentity(
	_ context.Context,
	paymentID string,
	providerPaymentID string,
	providerTransactionID string,
) error {
	s.bound =
		append(
			s.bound,
			struct {
				PaymentID string

				ProviderPaymentID string

				ProviderTransactionID string
			}{
				PaymentID: paymentID,

				ProviderPaymentID: providerPaymentID,

				ProviderTransactionID: providerTransactionID,
			},
		)

	return nil
}

func (s *fakeReconciliationStore) MarkReconciliationTerminal(
	_ context.Context,
	update reconciliationTerminalUpdate,
) error {
	s.terminal =
		append(
			s.terminal,
			update,
		)

	return nil
}

type fakeReconciliationProvider struct {
	code string

	queryResult paymentintegrations.QueryPaymentResult

	queryErr error

	merchantResult paymentintegrations.QueryPaymentResult

	merchantErr error

	queryCalls int

	merchantCalls int
}

func (p *fakeReconciliationProvider) Code() string {
	return p.code
}

func (p *fakeReconciliationProvider) Ready() error {
	return nil
}

func (p *fakeReconciliationProvider) CreatePayment(
	context.Context,
	paymentintegrations.CreatePaymentRequest,
) (
	paymentintegrations.CreatePaymentResult,
	error,
) {
	return paymentintegrations.CreatePaymentResult{},
		nil
}

func (p *fakeReconciliationProvider) QueryPayment(
	context.Context,
	paymentintegrations.QueryPaymentRequest,
) (
	paymentintegrations.QueryPaymentResult,
	error,
) {
	p.queryCalls++

	return p.queryResult,
		p.queryErr
}

func (p *fakeReconciliationProvider) QueryPaymentByMerchantReference(
	context.Context,
	string,
) (
	paymentintegrations.QueryPaymentResult,
	error,
) {
	p.merchantCalls++

	return p.merchantResult,
		p.merchantErr
}

func (p *fakeReconciliationProvider) VerifyWebhook(
	context.Context,
	paymentintegrations.WebhookRequest,
) (
	paymentintegrations.VerifiedWebhookEvent,
	error,
) {
	return paymentintegrations.VerifiedWebhookEvent{},
		nil
}

type fakeReconciliationConfirmer struct {
	calls int

	input ConfirmVerifiedPaymentInput

	result ConfirmVerifiedPaymentResult

	err error
}

func (c *fakeReconciliationConfirmer) ConfirmVerifiedPayment(
	_ context.Context,
	input ConfirmVerifiedPaymentInput,
) (
	ConfirmVerifiedPaymentResult,
	error,
) {
	c.calls++

	c.input =
		input

	return c.result,
		c.err
}

func reconciliationCandidateFixture() reconciliationCandidate {
	return reconciliationCandidate{
		ID: "11111111-1111-4111-8111-111111111111",

		OrderID: "22222222-2222-4222-8222-222222222222",

		Provider: paymentintegrations.ProviderBKash,

		Status: StatusPending,

		Amount: 250000,

		Currency: "BDT",

		AttemptKey: "pay_test_reference",

		ProviderPaymentID: "provider-payment-1",

		CreatedAt: time.Now().
			Add(
				-time.Minute,
			).
			UTC(),

		UpdatedAt: time.Now().
			Add(
				-time.Minute,
			).
			UTC(),
	}
}

func reconciliationRegistry(
	t *testing.T,
	provider paymentintegrations.Provider,
) *paymentintegrations.Registry {
	t.Helper()

	/*
		All NEW-payment enable flags deliberately remain false.

		ResolveForProcessing must still allow reconciliation of an
		already-created payment.
	*/
	registry :=
		paymentintegrations.NewRegistry(
			paymentintegrations.Enablement{},
		)

	if provider != nil {
		if err :=
			registry.Register(
				provider,
			); err != nil {

			t.Fatalf(
				"register reconciliation provider: %v",
				err,
			)
		}
	}

	return registry
}

func TestReconciliationSucceedsThroughProcessingRegistry(
	t *testing.T,
) {
	candidate :=
		reconciliationCandidateFixture()

	paidAt :=
		time.Now().
			Add(
				-time.Minute,
			).
			UTC()

	provider :=
		&fakeReconciliationProvider{
			code: paymentintegrations.ProviderBKash,

			queryResult: paymentintegrations.QueryPaymentResult{
				ProviderPaymentID: candidate.ProviderPaymentID,

				ProviderTransactionID: "transaction-1",

				Status: paymentintegrations.StatusSucceeded,

				Amount: candidate.Amount,

				Currency: candidate.Currency,

				PaidAt: &paidAt,
			},
		}

	store :=
		&fakeReconciliationStore{
			candidates: []reconciliationCandidate{
				candidate,
			},
		}

	confirmer :=
		&fakeReconciliationConfirmer{
			result: ConfirmVerifiedPaymentResult{
				Payment: Payment{
					ID: "33333333-3333-4333-8333-333333333333",

					Status: StatusSucceeded,
				},

				Processed: true,
			},
		}

	service :=
		&ReconciliationService{
			repository: store,

			providers: reconciliationRegistry(
				t,
				provider,
			),

			confirmer: confirmer,

			minAge: 0,

			queryTimeout: time.Second,
		}

	result, err :=
		service.ReconcilePending(
			context.Background(),
			time.Now(),
			100,
		)
	if err != nil {
		t.Fatalf(
			"reconcile payment: %v",
			err,
		)
	}

	if result.Scanned != 1 ||
		result.Succeeded != 1 {

		t.Fatalf(
			"unexpected reconciliation result: %#v",
			result,
		)
	}

	if confirmer.calls != 1 {
		t.Fatalf(
			"expected one confirmation call, got %d",
			confirmer.calls,
		)
	}

	if len(
		confirmer.input.PayloadSHA256,
	) != 64 {

		t.Fatalf(
			"unexpected reconciliation payload hash %q",
			confirmer.input.PayloadSHA256,
		)
	}

	if confirmer.input.EventType !=
		"payment.reconciliation.succeeded" {

		t.Fatalf(
			"unexpected event type %q",
			confirmer.input.EventType,
		)
	}
}

func TestReconciliationRecoversProviderIdentityByMerchantReference(
	t *testing.T,
) {
	candidate :=
		reconciliationCandidateFixture()

	candidate.ProviderPaymentID =
		""

	provider :=
		&fakeReconciliationProvider{
			code: paymentintegrations.ProviderBKash,

			merchantResult: paymentintegrations.QueryPaymentResult{
				ProviderPaymentID: "recovered-provider-payment",

				Status: paymentintegrations.StatusPending,
			},
		}

	store :=
		&fakeReconciliationStore{
			candidates: []reconciliationCandidate{
				candidate,
			},
		}

	service :=
		&ReconciliationService{
			repository: store,

			providers: reconciliationRegistry(
				t,
				provider,
			),

			confirmer: &fakeReconciliationConfirmer{},

			minAge: 0,

			queryTimeout: time.Second,
		}

	result, err :=
		service.ReconcilePending(
			context.Background(),
			time.Now(),
			100,
		)
	if err != nil {
		t.Fatalf(
			"reconcile merchant reference: %v",
			err,
		)
	}

	if result.Pending != 1 {
		t.Fatalf(
			"unexpected reconciliation result: %#v",
			result,
		)
	}

	if provider.merchantCalls != 1 {
		t.Fatalf(
			"expected merchant reference lookup",
		)
	}

	if len(
		store.bound,
	) != 1 ||
		store.bound[0].
			ProviderPaymentID !=
			"recovered-provider-payment" {

		t.Fatalf(
			"provider identity was not recovered: %#v",
			store.bound,
		)
	}
}

func TestReconciliationMapsProviderCancellationToFailed(
	t *testing.T,
) {
	candidate :=
		reconciliationCandidateFixture()

	provider :=
		&fakeReconciliationProvider{
			code: paymentintegrations.ProviderBKash,

			queryResult: paymentintegrations.QueryPaymentResult{
				ProviderPaymentID: candidate.ProviderPaymentID,

				Status: paymentintegrations.StatusCancelled,
			},
		}

	store :=
		&fakeReconciliationStore{
			candidates: []reconciliationCandidate{
				candidate,
			},
		}

	service :=
		&ReconciliationService{
			repository: store,

			providers: reconciliationRegistry(
				t,
				provider,
			),

			confirmer: &fakeReconciliationConfirmer{},

			minAge: 0,

			queryTimeout: time.Second,
		}

	result, err :=
		service.ReconcilePending(
			context.Background(),
			time.Now(),
			100,
		)
	if err != nil {
		t.Fatalf(
			"reconcile cancelled payment: %v",
			err,
		)
	}

	if result.Failed != 1 {
		t.Fatalf(
			"unexpected reconciliation result: %#v",
			result,
		)
	}

	if len(
		store.terminal,
	) != 1 {

		t.Fatalf(
			"expected one terminal update",
		)
	}

	if store.terminal[0].Status !=
		StatusFailed ||
		store.terminal[0].FailureCode !=
			"provider_cancelled" {

		t.Fatalf(
			"unexpected terminal update: %#v",
			store.terminal[0],
		)
	}
}

func TestReconciliationDefersUnavailableProvider(
	t *testing.T,
) {
	candidate :=
		reconciliationCandidateFixture()

	store :=
		&fakeReconciliationStore{
			candidates: []reconciliationCandidate{
				candidate,
			},
		}

	service :=
		&ReconciliationService{
			repository: store,

			providers: reconciliationRegistry(
				t,
				nil,
			),

			confirmer: &fakeReconciliationConfirmer{},

			minAge: 0,

			queryTimeout: time.Second,
		}

	result, err :=
		service.ReconcilePending(
			context.Background(),
			time.Now(),
			100,
		)
	if err != nil {
		t.Fatalf(
			"defer unavailable provider: %v",
			err,
		)
	}

	if result.Deferred != 1 ||
		len(
			store.deferred,
		) != 1 {

		t.Fatalf(
			"unexpected deferred result: %#v deferred=%#v",
			result,
			store.deferred,
		)
	}
}

func TestReconciliationDefersMismatchedSuccessfulAmount(
	t *testing.T,
) {
	candidate :=
		reconciliationCandidateFixture()

	provider :=
		&fakeReconciliationProvider{
			code: paymentintegrations.ProviderBKash,

			queryResult: paymentintegrations.QueryPaymentResult{
				ProviderPaymentID: candidate.ProviderPaymentID,

				Status: paymentintegrations.StatusSucceeded,

				Amount: candidate.Amount + 1,

				Currency: candidate.Currency,
			},
		}

	store :=
		&fakeReconciliationStore{
			candidates: []reconciliationCandidate{
				candidate,
			},
		}

	confirmer :=
		&fakeReconciliationConfirmer{}

	service :=
		&ReconciliationService{
			repository: store,

			providers: reconciliationRegistry(
				t,
				provider,
			),

			confirmer: confirmer,

			minAge: 0,

			queryTimeout: time.Second,
		}

	result, err :=
		service.ReconcilePending(
			context.Background(),
			time.Now(),
			100,
		)
	if err != nil {
		t.Fatalf(
			"reconcile mismatched amount: %v",
			err,
		)
	}

	if result.ReconciliationRequired !=
		1 {

		t.Fatalf(
			"unexpected reconciliation result: %#v",
			result,
		)
	}

	if confirmer.calls != 0 {
		t.Fatalf(
			"mismatched provider result reached payment confirmation",
		)
	}

	if len(
		store.deferred,
	) != 1 {

		t.Fatalf(
			"mismatched payment was not deferred",
		)
	}
}

func TestReconciliationConflictClassification(
	t *testing.T,
) {
	if !isPaymentReconciliationConflict(
		ErrReconciliationRequired,
	) {
		t.Fatal(
			"expected reconciliation error classification",
		)
	}

	if isPaymentReconciliationConflict(
		errors.New(
			"database unavailable",
		),
	) {
		t.Fatal(
			"unexpected generic error classification",
		)
	}
}
