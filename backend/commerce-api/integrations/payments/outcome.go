package payments

import "errors"

var (
	// ErrProviderRequestRejected means the adapter knows the
	// provider did NOT create the payment.
	//
	// A new initiation attempt may therefore be created later.
	ErrProviderRequestRejected = errors.New(
		"payment provider rejected request",
	)

	// ErrProviderOutcomeUnknown means the request may have reached
	// the provider, but the application cannot determine whether
	// the provider created a payment.
	//
	// The local pending attempt must be preserved for retry or
	// reconciliation.
	ErrProviderOutcomeUnknown = errors.New(
		"payment provider outcome unknown",
	)
)
