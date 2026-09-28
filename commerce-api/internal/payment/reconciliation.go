package payment

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	paymentintegrations "project.local/commerce-api/integrations/payments"
)

const (
	defaultPaymentReconciliationBatchSize = 100

	maxPaymentReconciliationBatchSize = 1000

	defaultPaymentReconciliationMinAge = 15 * time.Second

	defaultPaymentProviderQueryTimeout = 8 * time.Second
)

type ReconciliationBatchResult struct {
	Scanned int `json:"scanned"`

	Succeeded int `json:"succeeded"`

	Pending int `json:"pending"`

	Failed int `json:"failed"`

	Expired int `json:"expired"`

	Deferred int `json:"deferred"`

	ReconciliationRequired int `json:"reconciliation_required"`
}

type reconciliationStore interface {
	ListPendingReconciliationCandidates(
		ctx context.Context,
		before time.Time,
		limit int,
	) ([]reconciliationCandidate, error)

	DeferReconciliation(
		ctx context.Context,
		paymentID string,
	) error

	BindReconciliationProviderIdentity(
		ctx context.Context,
		paymentID string,
		providerPaymentID string,
		providerTransactionID string,
	) error

	MarkReconciliationTerminal(
		ctx context.Context,
		update reconciliationTerminalUpdate,
	) error
}

type reconciliationPaymentConfirmer interface {
	ConfirmVerifiedPayment(
		ctx context.Context,
		input ConfirmVerifiedPaymentInput,
	) (ConfirmVerifiedPaymentResult, error)
}

type ReconciliationService struct {
	repository reconciliationStore

	providers *paymentintegrations.Registry

	confirmer reconciliationPaymentConfirmer

	minAge time.Duration

	queryTimeout time.Duration
}

func NewReconciliationService(
	repository *Repository,
	providers *paymentintegrations.Registry,
	confirmer reconciliationPaymentConfirmer,
) *ReconciliationService {
	return &ReconciliationService{
		repository: repository,

		providers: providers,

		confirmer: confirmer,

		minAge: defaultPaymentReconciliationMinAge,

		queryTimeout: defaultPaymentProviderQueryTimeout,
	}
}

func (s *ReconciliationService) ReconcilePending(
	ctx context.Context,
	now time.Time,
	limit int,
) (
	ReconciliationBatchResult,
	error,
) {
	if s == nil ||
		s.repository == nil ||
		s.providers == nil ||
		s.confirmer == nil {

		return ReconciliationBatchResult{},
			fmt.Errorf(
				"payment reconciliation service is not configured",
			)
	}

	if now.IsZero() {
		now =
			time.Now().
				UTC()
	} else {
		now =
			now.UTC()
	}

	limit =
		normalizePaymentReconciliationBatchSize(
			limit,
		)

	before :=
		now.Add(
			-s.minAge,
		)

	candidates, err :=
		s.repository.
			ListPendingReconciliationCandidates(
				ctx,
				before,
				limit,
			)
	if err != nil {
		return ReconciliationBatchResult{},
			err
	}

	result :=
		ReconciliationBatchResult{
			Scanned: len(
				candidates,
			),
		}

	for _, candidate := range candidates {

		if err :=
			ctx.Err(); err != nil {

			return result,
				err
		}

		provider, err :=
			s.providers.
				ResolveForProcessing(
					candidate.Provider,
				)
		if err != nil {
			if deferErr :=
				s.repository.
					DeferReconciliation(
						ctx,
						candidate.ID,
					); deferErr != nil {

				return result,
					deferErr
			}

			result.Deferred++

			continue
		}

		queryResult, err :=
			s.queryProvider(
				ctx,
				provider,
				candidate,
			)
		if err != nil {
			if ctx.Err() != nil {
				return result,
					ctx.Err()
			}

			if deferErr :=
				s.repository.
					DeferReconciliation(
						ctx,
						candidate.ID,
					); deferErr != nil {

				return result,
					deferErr
			}

			result.Deferred++

			continue
		}

		queryResult, err =
			normalizeReconciliationQueryResult(
				candidate,
				queryResult,
			)
		if err != nil {
			if deferErr :=
				s.repository.
					DeferReconciliation(
						ctx,
						candidate.ID,
					); deferErr != nil {

				return result,
					deferErr
			}

			result.ReconciliationRequired++

			continue
		}

		switch queryResult.Status {
		case paymentintegrations.StatusPending:
			err :=
				s.repository.
					BindReconciliationProviderIdentity(
						ctx,
						candidate.ID,
						queryResult.ProviderPaymentID,
						queryResult.ProviderTransactionID,
					)

			if errors.Is(
				err,
				ErrPaymentReferenceConflict,
			) {
				if deferErr :=
					s.repository.
						DeferReconciliation(
							ctx,
							candidate.ID,
						); deferErr != nil {

					return result,
						deferErr
				}

				result.ReconciliationRequired++

				continue
			}

			if err != nil {
				return result,
					err
			}

			result.Pending++

		case paymentintegrations.StatusSucceeded:
			confirmed, err :=
				s.confirmSucceeded(
					ctx,
					now,
					candidate,
					queryResult,
				)

			if err != nil {
				if isPaymentReconciliationConflict(
					err,
				) {
					if confirmed.Payment.Status ==
						StatusSucceeded {

						result.Succeeded++
					} else {
						if deferErr :=
							s.repository.
								DeferReconciliation(
									ctx,
									candidate.ID,
								); deferErr != nil {

							return result,
								deferErr
						}
					}

					result.ReconciliationRequired++

					continue
				}

				return result,
					err
			}

			result.Succeeded++

		case paymentintegrations.StatusFailed:
			err :=
				s.markTerminal(
					ctx,
					candidate,
					queryResult,
					StatusFailed,
					"provider_failed",
					"payment provider reports payment failed",
				)
			if err != nil {
				if errors.Is(
					err,
					ErrPaymentReferenceConflict,
				) {
					if deferErr :=
						s.repository.
							DeferReconciliation(
								ctx,
								candidate.ID,
							); deferErr != nil {

						return result,
							deferErr
					}

					result.ReconciliationRequired++

					continue
				}

				return result,
					err
			}

			result.Failed++

		case paymentintegrations.StatusCancelled:
			err :=
				s.markTerminal(
					ctx,
					candidate,
					queryResult,
					StatusFailed,
					"provider_cancelled",
					"payment provider reports payment cancelled",
				)
			if err != nil {
				if errors.Is(
					err,
					ErrPaymentReferenceConflict,
				) {
					if deferErr :=
						s.repository.
							DeferReconciliation(
								ctx,
								candidate.ID,
							); deferErr != nil {

						return result,
							deferErr
					}

					result.ReconciliationRequired++

					continue
				}

				return result,
					err
			}

			result.Failed++

		case paymentintegrations.StatusExpired:
			err :=
				s.markTerminal(
					ctx,
					candidate,
					queryResult,
					StatusExpired,
					"provider_expired",
					"payment provider reports payment expired",
				)
			if err != nil {
				if errors.Is(
					err,
					ErrPaymentReferenceConflict,
				) {
					if deferErr :=
						s.repository.
							DeferReconciliation(
								ctx,
								candidate.ID,
							); deferErr != nil {

						return result,
							deferErr
					}

					result.ReconciliationRequired++

					continue
				}

				return result,
					err
			}

			result.Expired++
		}
	}

	return result,
		nil
}

func (s *ReconciliationService) queryProvider(
	ctx context.Context,
	provider paymentintegrations.Provider,
	candidate reconciliationCandidate,
) (
	paymentintegrations.QueryPaymentResult,
	error,
) {
	queryCtx, cancel :=
		context.WithTimeout(
			ctx,
			s.queryTimeout,
		)

	defer cancel()

	if candidate.ProviderPaymentID != "" {
		return provider.QueryPayment(
			queryCtx,
			paymentintegrations.QueryPaymentRequest{
				ProviderPaymentID: candidate.ProviderPaymentID,
			},
		)
	}

	if candidate.AttemptKey == "" {
		return paymentintegrations.QueryPaymentResult{},
			paymentintegrations.ErrInvalidPaymentRequest
	}

	merchantQuerier, ok :=
		provider.(paymentintegrations.MerchantReferenceQuerier)

	if !ok {
		return paymentintegrations.QueryPaymentResult{},
			paymentintegrations.ErrProviderOutcomeUnknown
	}

	return merchantQuerier.
		QueryPaymentByMerchantReference(
			queryCtx,
			candidate.AttemptKey,
		)
}

func (s *ReconciliationService) confirmSucceeded(
	ctx context.Context,
	now time.Time,
	candidate reconciliationCandidate,
	queryResult paymentintegrations.QueryPaymentResult,
) (
	ConfirmVerifiedPaymentResult,
	error,
) {
	eventID,
		eventType,
		payloadHash :=
		reconciliationObservation(
			candidate,
			queryResult,
		)

	paidAt :=
		now

	if queryResult.PaidAt != nil {
		paidAt =
			queryResult.PaidAt.UTC()
	}

	return s.confirmer.
		ConfirmVerifiedPayment(
			ctx,
			ConfirmVerifiedPaymentInput{
				OrderID: candidate.OrderID,

				Provider: candidate.Provider,

				ProviderEventID: eventID,

				EventType: eventType,

				ProviderPaymentID: queryResult.ProviderPaymentID,

				ProviderTransactionID: queryResult.ProviderTransactionID,

				Amount: queryResult.Amount,

				Currency: queryResult.Currency,

				PaidAt: paidAt,

				PayloadSHA256: payloadHash,
			},
		)
}

func (s *ReconciliationService) markTerminal(
	ctx context.Context,
	candidate reconciliationCandidate,
	queryResult paymentintegrations.QueryPaymentResult,
	internalStatus string,
	failureCode string,
	failureMessage string,
) error {
	eventID,
		eventType,
		payloadHash :=
		reconciliationObservation(
			candidate,
			queryResult,
		)

	return s.repository.
		MarkReconciliationTerminal(
			ctx,
			reconciliationTerminalUpdate{
				PaymentID: candidate.ID,

				Status: internalStatus,

				ProviderPaymentID: queryResult.ProviderPaymentID,

				ProviderTransactionID: queryResult.ProviderTransactionID,

				FailureCode: failureCode,

				FailureMessage: failureMessage,

				EventID: eventID,

				EventType: eventType,

				PayloadSHA256: payloadHash,
			},
		)
}

func normalizeReconciliationQueryResult(
	candidate reconciliationCandidate,
	result paymentintegrations.QueryPaymentResult,
) (
	paymentintegrations.QueryPaymentResult,
	error,
) {
	result.ProviderPaymentID =
		strings.TrimSpace(
			result.ProviderPaymentID,
		)

	result.ProviderTransactionID =
		strings.TrimSpace(
			result.ProviderTransactionID,
		)

	result.Status =
		strings.ToLower(
			strings.TrimSpace(
				result.Status,
			),
		)

	result.Currency =
		strings.ToUpper(
			strings.TrimSpace(
				result.Currency,
			),
		)

	if result.ProviderPaymentID == "" {
		result.ProviderPaymentID =
			candidate.ProviderPaymentID
	}

	if candidate.ProviderPaymentID != "" &&
		result.ProviderPaymentID != "" &&
		candidate.ProviderPaymentID !=
			result.ProviderPaymentID {

		return paymentintegrations.QueryPaymentResult{},
			ErrPaymentReferenceConflict
	}

	if len(
		result.ProviderPaymentID,
	) > 160 ||
		len(
			result.ProviderTransactionID,
		) > 160 {

		return paymentintegrations.QueryPaymentResult{},
			ErrInvalidInput
	}

	switch result.Status {
	case paymentintegrations.StatusPending,
		paymentintegrations.StatusSucceeded,
		paymentintegrations.StatusFailed,
		paymentintegrations.StatusCancelled,
		paymentintegrations.StatusExpired:

	default:
		return paymentintegrations.QueryPaymentResult{},
			ErrInvalidInput
	}

	/*
		Some provider query endpoints omit amount/currency for
		non-successful states.

		If they are supplied, however, they must agree with the
		local payment attempt.
	*/
	if result.Amount != 0 &&
		result.Amount !=
			candidate.Amount {

		return paymentintegrations.QueryPaymentResult{},
			ErrPaymentAmountMismatch
	}

	if result.Currency != "" &&
		result.Currency !=
			candidate.Currency {

		return paymentintegrations.QueryPaymentResult{},
			ErrPaymentCurrencyMismatch
	}

	/*
		A successful result must carry enough authoritative data to
		enter ConfirmVerifiedPayment().
	*/
	if result.Status ==
		paymentintegrations.StatusSucceeded {

		if result.ProviderPaymentID == "" {
			return paymentintegrations.QueryPaymentResult{},
				ErrInvalidInput
		}

		if result.Amount !=
			candidate.Amount {

			return paymentintegrations.QueryPaymentResult{},
				ErrPaymentAmountMismatch
		}

		if result.Currency !=
			candidate.Currency {

			return paymentintegrations.QueryPaymentResult{},
				ErrPaymentCurrencyMismatch
		}
	}

	if result.PaidAt != nil {
		value :=
			result.PaidAt.UTC()

		result.PaidAt =
			&value
	}

	return result,
		nil
}

func reconciliationObservation(
	candidate reconciliationCandidate,
	result paymentintegrations.QueryPaymentResult,
) (
	string,
	string,
	string,
) {
	paidAt :=
		""

	if result.PaidAt != nil {
		paidAt =
			result.PaidAt.
				UTC().
				Format(
					time.RFC3339Nano,
				)
	}

	canonical :=
		strings.Join(
			[]string{
				"payment-reconciliation-v1",

				candidate.ID,
				candidate.OrderID,
				candidate.Provider,
				candidate.AttemptKey,

				result.ProviderPaymentID,
				result.ProviderTransactionID,
				result.Status,

				fmt.Sprintf(
					"%d",
					result.Amount,
				),

				result.Currency,
				paidAt,
			},
			"|",
		)

	sum :=
		sha256.Sum256(
			[]byte(
				canonical,
			),
		)

	hash :=
		hex.EncodeToString(
			sum[:],
		)

	return "reconcile_" + hash,
		"payment.reconciliation." + result.Status,
		hash
}

func isPaymentReconciliationConflict(
	err error,
) bool {
	return errors.Is(
		err,
		ErrPaymentMethodMismatch,
	) ||
		errors.Is(
			err,
			ErrPaymentAmountMismatch,
		) ||
		errors.Is(
			err,
			ErrPaymentCurrencyMismatch,
		) ||
		errors.Is(
			err,
			ErrPaymentReferenceConflict,
		) ||
		errors.Is(
			err,
			ErrReconciliationRequired,
		)
}

func normalizePaymentReconciliationBatchSize(
	limit int,
) int {
	if limit <= 0 {
		return defaultPaymentReconciliationBatchSize
	}

	if limit >
		maxPaymentReconciliationBatchSize {

		return maxPaymentReconciliationBatchSize
	}

	return limit
}
