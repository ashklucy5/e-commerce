package payment

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	paymentintegrations "project.local/commerce-api/integrations/payments"
)

const (
	paymentAttemptRandomBytes = 24

	maxProviderHandoffURLLength = 4096
)

type InitiatePaymentResult struct {
	PaymentID string `json:"payment_id"`

	OrderID string `json:"order_id"`

	Provider string `json:"provider"`

	Status string `json:"status"`

	HandoffURL string `json:"handoff_url,omitempty"`

	ExpiresAt *time.Time `json:"expires_at,omitempty"`

	Reused bool `json:"reused"`
}

type InitiationService struct {
	repository *Repository

	providers *paymentintegrations.Registry
}

func NewInitiationService(
	repository *Repository,
	providers *paymentintegrations.Registry,
) *InitiationService {
	return &InitiationService{
		repository: repository,

		providers: providers,
	}
}

func (s *InitiationService) Initiate(
	ctx context.Context,
	customerID string,
	orderID string,
	checkoutKey string,
) (
	InitiatePaymentResult,
	error,
) {
	orderID =
		strings.TrimSpace(
			orderID,
		)

	if !paymentUUIDPattern.MatchString(
		orderID,
	) {
		return InitiatePaymentResult{},
			ErrInvalidInput
	}

	customerID =
		strings.TrimSpace(
			customerID,
		)

	if customerID != "" &&
		!paymentUUIDPattern.MatchString(
			customerID,
		) {

		// Never let an invalid identity value become a database
		// cast error. Treat it as no valid owner identity.
		customerID = ""
	}

	checkoutKey =
		normalizePaymentCheckoutKey(
			checkoutKey,
		)

	tx, err :=
		s.repository.Begin(
			ctx,
		)
	if err != nil {
		return InitiatePaymentResult{},
			err
	}

	defer func() {
		_ = tx.Rollback(
			ctx,
		)
	}()

	orderSnapshot, err :=
		s.repository.LockOwnedPaymentInitiationOrderTx(
			ctx,
			tx,
			orderID,
			customerID,
			checkoutKey,
		)
	if err != nil {
		return InitiatePaymentResult{},
			err
	}

	if orderSnapshot.PaymentMethod ==
		PaymentMethodCOD {

		return InitiatePaymentResult{},
			ErrPaymentInitiationNotRequired
	}

	if orderSnapshot.TotalAmount <= 0 {
		return InitiatePaymentResult{},
			ErrPaymentInitiationNotRequired
	}

	if orderSnapshot.Status !=
		"pending_payment" {

		return InitiatePaymentResult{},
			ErrPaymentOrderNotPending
	}

	if orderSnapshot.PaymentDueAt != nil &&
		!time.Now().
			UTC().
			Before(
				orderSnapshot.PaymentDueAt.UTC(),
			) {

		return InitiatePaymentResult{},
			ErrPaymentWindowExpired
	}

	if s.providers == nil {
		return InitiatePaymentResult{},
			ErrPaymentMethodUnavailable
	}

	provider, err :=
		s.providers.Resolve(
			orderSnapshot.PaymentMethod,
		)
	if err != nil {
		return InitiatePaymentResult{},
			fmt.Errorf(
				"%w: %v",
				ErrPaymentMethodUnavailable,
				err,
			)
	}

	attempt, found, err :=
		s.repository.GetPendingInitiationByOrderIDTx(
			ctx,
			tx,
			orderSnapshot.ID,
		)
	if err != nil {
		return InitiatePaymentResult{},
			err
	}

	reused :=
		found

	if found {
		if attempt.Provider !=
			orderSnapshot.PaymentMethod ||
			attempt.Amount !=
				orderSnapshot.TotalAmount ||
			attempt.Currency !=
				orderSnapshot.Currency {

			return InitiatePaymentResult{},
				ErrReconciliationRequired
		}

		if attempt.AttemptKey == "" {
			// Legacy pending payment rows created before initiation
			// support cannot safely be sent to a provider because
			// they do not have an idempotency reference.
			return InitiatePaymentResult{},
				ErrReconciliationRequired
		}
	} else {
		attemptKey, err :=
			newPaymentAttemptKey()
		if err != nil {
			return InitiatePaymentResult{},
				err
		}

		attempt, err =
			s.repository.InsertPendingInitiationTx(
				ctx,
				tx,
				orderSnapshot,
				attemptKey,
			)
		if err != nil {
			return InitiatePaymentResult{},
				err
		}
	}

	if err :=
		tx.Commit(
			ctx,
		); err != nil {

		return InitiatePaymentResult{},
			fmt.Errorf(
				"commit pending payment initiation: %w",
				err,
			)
	}

	// A previous request already completed provider initiation.
	// Never call the provider again in that case.
	if attempt.ProviderPaymentID != "" {
		if attempt.HandoffURL != "" ||
			attempt.Provider ==
				paymentintegrations.ProviderBankTransfer {

			return initiationResultFromAttempt(
				attempt,
				true,
			), nil
		}

		// Provider identity exists but local handoff state is
		// incomplete. This is a reconciliation case.
		return InitiatePaymentResult{},
			ErrReconciliationRequired
	}

	if attempt.HandoffURL != "" {
		return InitiatePaymentResult{},
			ErrReconciliationRequired
	}

	providerResult, err :=
		provider.CreatePayment(
			ctx,
			paymentintegrations.CreatePaymentRequest{
				OrderID: orderSnapshot.ID,

				OrderNumber: orderSnapshot.OrderNumber,

				MerchantReference: attempt.AttemptKey,

				Amount: orderSnapshot.TotalAmount,

				Currency: orderSnapshot.Currency,
			},
		)
	if err != nil {
		if errors.Is(
			err,
			paymentintegrations.ErrProviderRequestRejected,
		) {
			if markErr :=
				s.repository.MarkInitiationRejected(
					ctx,
					attempt.ID,
				); markErr != nil {

				return InitiatePaymentResult{},
					ErrReconciliationRequired
			}

			return InitiatePaymentResult{},
				ErrPaymentProviderRejected
		}

		// Timeout, connection reset, caller cancellation or any
		// unclassified provider error is treated as unknown.
		//
		// Do NOT fail the database row because the provider may
		// have actually created the payment.
		return InitiatePaymentResult{},
			ErrPaymentProviderOutcomeUnknown
	}

	providerResult, err =
		normalizeProviderCreateResult(
			attempt.Provider,
			attempt.CreatedAt,
			providerResult,
		)
	if err != nil {
		// The provider may already have created the payment.
		// Preserve the pending attempt for retry/reconciliation.
		return InitiatePaymentResult{},
			ErrPaymentProviderOutcomeUnknown
	}

	updated, err :=
		s.repository.UpdateInitiationProviderHandoff(
			ctx,
			attempt.ID,
			providerResult.ProviderPaymentID,
			providerResult.RedirectURL,
			providerResult.ExpiresAt,
		)
	if err != nil {
		// At this point the provider creation succeeded, so a
		// local write failure must never cause us to create a new
		// unrelated provider payment.
		return InitiatePaymentResult{},
			ErrReconciliationRequired
	}

	return initiationResultFromAttempt(
		updated,
		reused,
	), nil
}

func initiationResultFromAttempt(
	attempt paymentInitiationAttempt,
	reused bool,
) InitiatePaymentResult {
	return InitiatePaymentResult{
		PaymentID: attempt.ID,

		OrderID: attempt.OrderID,

		Provider: attempt.Provider,

		Status: attempt.Status,

		HandoffURL: attempt.HandoffURL,

		ExpiresAt: attempt.ProviderExpiresAt,

		Reused: reused,
	}
}

func newPaymentAttemptKey() (
	string,
	error,
) {
	buffer :=
		make(
			[]byte,
			paymentAttemptRandomBytes,
		)

	if _, err :=
		rand.Read(
			buffer,
		); err != nil {

		return "",
			fmt.Errorf(
				"generate payment attempt key: %w",
				err,
			)
	}

	return "pay_" +
			base64.RawURLEncoding.
				EncodeToString(
					buffer,
				),
		nil
}

func normalizeProviderCreateResult(
	provider string,
	attemptCreatedAt time.Time,
	result paymentintegrations.CreatePaymentResult,
) (
	paymentintegrations.CreatePaymentResult,
	error,
) {
	result.ProviderPaymentID =
		strings.TrimSpace(
			result.ProviderPaymentID,
		)

	result.RedirectURL =
		strings.TrimSpace(
			result.RedirectURL,
		)

	if result.ProviderPaymentID == "" ||
		len(
			result.ProviderPaymentID,
		) > 160 {

		return paymentintegrations.CreatePaymentResult{},
			ErrInvalidInput
	}

	switch provider {
	case paymentintegrations.ProviderBKash,
		paymentintegrations.ProviderNagad,
		paymentintegrations.ProviderRocket:

		if result.RedirectURL == "" {
			return paymentintegrations.CreatePaymentResult{},
				ErrInvalidInput
		}
	}

	if len(
		result.RedirectURL,
	) > maxProviderHandoffURLLength {

		return paymentintegrations.CreatePaymentResult{},
			ErrInvalidInput
	}

	if result.RedirectURL != "" {
		parsed, err :=
			url.Parse(
				result.RedirectURL,
			)
		if err != nil {
			return paymentintegrations.CreatePaymentResult{},
				ErrInvalidInput
		}

		if parsed.Host == "" {
			return paymentintegrations.CreatePaymentResult{},
				ErrInvalidInput
		}

		switch strings.ToLower(
			parsed.Scheme,
		) {
		case "http", "https":

		default:
			return paymentintegrations.CreatePaymentResult{},
				ErrInvalidInput
		}
	}

	if result.ExpiresAt != nil {
		value :=
			result.ExpiresAt.UTC()

		if !value.After(
			attemptCreatedAt.UTC(),
		) {
			return paymentintegrations.CreatePaymentResult{},
				ErrInvalidInput
		}

		result.ExpiresAt =
			&value
	}

	return result,
		nil
}
