package payments

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"
)

const (
	ProviderBKash        = "bkash"
	ProviderNagad        = "nagad"
	ProviderRocket       = "rocket"
	ProviderBankTransfer = "bank_transfer"
)

const (
	StatusPending   = "pending"
	StatusSucceeded = "succeeded"
	StatusFailed    = "failed"
	StatusCancelled = "cancelled"
	StatusExpired   = "expired"
)

var (
	ErrUnsupportedProvider = errors.New(
		"unsupported payment provider",
	)

	ErrProviderDisabled = errors.New(
		"payment provider disabled",
	)

	ErrProviderUnavailable = errors.New(
		"payment provider unavailable",
	)

	ErrProviderNotConfigured = errors.New(
		"payment provider not configured",
	)

	ErrProviderAlreadyRegistered = errors.New(
		"payment provider already registered",
	)

	ErrInvalidProvider = errors.New(
		"invalid payment provider",
	)

	ErrInvalidPaymentRequest = errors.New(
		"invalid payment request",
	)

	ErrInvalidWebhook = errors.New(
		"invalid payment webhook",
	)

	ErrWebhookVerificationFailed = errors.New(
		"payment webhook verification failed",
	)
)

type Enablement struct {
	BKash bool

	Nagad bool

	Rocket bool

	BankTransfer bool
}

func (e Enablement) Enabled(
	code string,
) bool {
	switch normalizeProviderCode(
		code,
	) {
	case ProviderBKash:
		return e.BKash

	case ProviderNagad:
		return e.Nagad

	case ProviderRocket:
		return e.Rocket

	case ProviderBankTransfer:
		return e.BankTransfer

	default:
		return false
	}
}

type CreatePaymentRequest struct {
	OrderID string

	OrderNumber string

	// MerchantReference must be used by the adapter as the
	// provider-side idempotency/merchant reference whenever
	// the provider supports one.
	//
	// For this application it is the payments.attempt_key.
	MerchantReference string

	Amount int64

	Currency string

	CustomerName string

	CustomerPhone string

	CustomerEmail string

	ReturnURL string

	CancelURL string
}

type CreatePaymentResult struct {
	ProviderPaymentID string

	RedirectURL string

	ExpiresAt *time.Time
}

type QueryPaymentRequest struct {
	ProviderPaymentID string
}

type QueryPaymentResult struct {
	ProviderPaymentID string

	ProviderTransactionID string

	Status string

	Amount int64

	Currency string

	PaidAt *time.Time
}

type WebhookRequest struct {
	Headers http.Header

	Body []byte
}

type VerifiedWebhookEvent struct {
	ProviderEventID string

	EventType string

	OrderID string

	ProviderPaymentID string

	ProviderTransactionID string

	Status string

	Amount int64

	Currency string

	PaidAt *time.Time
}

type RefundRequest struct {
	ProviderPaymentID string

	ProviderTransactionID string

	Amount int64

	Currency string

	Reason string
}

type RefundResult struct {
	ProviderRefundID string

	Status string
}

type Provider interface {
	Code() string

	// Ready reports whether this adapter has all mandatory
	// configuration required for safe real provider calls.
	Ready() error

	CreatePayment(
		ctx context.Context,
		request CreatePaymentRequest,
	) (CreatePaymentResult, error)

	QueryPayment(
		ctx context.Context,
		request QueryPaymentRequest,
	) (QueryPaymentResult, error)

	VerifyWebhook(
		ctx context.Context,
		request WebhookRequest,
	) (VerifiedWebhookEvent, error)
}

type Refunder interface {
	Refund(
		ctx context.Context,
		request RefundRequest,
	) (RefundResult, error)
}

type ProviderStatus struct {
	Code string `json:"code"`

	Enabled bool `json:"enabled"`

	Registered bool `json:"registered"`

	Configured bool `json:"configured"`

	Available bool `json:"available"`
}

type Registry struct {
	enablement Enablement

	providers map[string]Provider
}

func NewRegistry(
	enablement Enablement,
) *Registry {
	return &Registry{
		enablement: enablement,

		providers: make(
			map[string]Provider,
		),
	}
}

func (r *Registry) Register(
	provider Provider,
) error {
	if provider == nil {
		return ErrInvalidProvider
	}

	code :=
		normalizeProviderCode(
			provider.Code(),
		)

	if !supportedProvider(
		code,
	) {
		return ErrUnsupportedProvider
	}

	if _, exists :=
		r.providers[code]; exists {

		return fmt.Errorf(
			"%w: %s",
			ErrProviderAlreadyRegistered,
			code,
		)
	}

	r.providers[code] =
		provider

	return nil
}

// Resolve is for NEW customer payment initiation.
//
// Environment enablement is mandatory.
func (r *Registry) Resolve(
	code string,
) (Provider, error) {
	code =
		normalizeProviderCode(
			code,
		)

	if !supportedProvider(
		code,
	) {
		return nil,
			ErrUnsupportedProvider
	}

	if !r.enablement.Enabled(
		code,
	) {
		return nil,
			fmt.Errorf(
				"%w: %s",
				ErrProviderDisabled,
				code,
			)
	}

	return r.resolveConfigured(
		code,
	)
}

// ResolveForProcessing deliberately ignores the enable flag.
//
// This is intended for later webhook/reconciliation processing.
//
// Example:
// A merchant disables bKash at 12:00, but a payment initiated
// at 11:59 delivers its verified callback at 12:01.
// That callback must still be processed safely.
func (r *Registry) ResolveForProcessing(
	code string,
) (Provider, error) {
	code =
		normalizeProviderCode(
			code,
		)

	if !supportedProvider(
		code,
	) {
		return nil,
			ErrUnsupportedProvider
	}

	return r.resolveConfigured(
		code,
	)
}

func (r *Registry) resolveConfigured(
	code string,
) (Provider, error) {
	provider, exists :=
		r.providers[code]

	if !exists {
		return nil,
			fmt.Errorf(
				"%w: %s",
				ErrProviderUnavailable,
				code,
			)
	}

	if err :=
		provider.Ready(); err != nil {

		return nil,
			fmt.Errorf(
				"%w: %s: %v",
				ErrProviderNotConfigured,
				code,
				err,
			)
	}

	return provider,
		nil
}

func (r *Registry) Available(
	code string,
) bool {
	_, err :=
		r.Resolve(
			code,
		)

	return err == nil
}

func (r *Registry) Registered(
	code string,
) bool {
	code =
		normalizeProviderCode(
			code,
		)

	_, exists :=
		r.providers[code]

	return exists
}

func (r *Registry) Status(
	code string,
) ProviderStatus {
	code =
		normalizeProviderCode(
			code,
		)

	status :=
		ProviderStatus{
			Code: code,

			Enabled: r.enablement.Enabled(
				code,
			),

			Registered: r.Registered(
				code,
			),
		}

	if provider, exists :=
		r.providers[code]; exists {

		status.Configured =
			provider.Ready() == nil
	}

	status.Available =
		status.Enabled &&
			status.Registered &&
			status.Configured

	return status
}

func (r *Registry) Statuses() []ProviderStatus {
	codes :=
		[]string{
			ProviderBKash,
			ProviderNagad,
			ProviderRocket,
			ProviderBankTransfer,
		}

	result :=
		make(
			[]ProviderStatus,
			0,
			len(codes),
		)

	for _, code := range codes {

		result =
			append(
				result,
				r.Status(
					code,
				),
			)
	}

	sort.SliceStable(
		result,
		func(
			i int,
			j int,
		) bool {
			return result[i].Code <
				result[j].Code
		},
	)

	return result
}

func supportedProvider(
	code string,
) bool {
	switch code {
	case ProviderBKash,
		ProviderNagad,
		ProviderRocket,
		ProviderBankTransfer:

		return true

	default:
		return false
	}
}

func normalizeProviderCode(
	value string,
) string {
	return strings.ToLower(
		strings.TrimSpace(
			value,
		),
	)
}
