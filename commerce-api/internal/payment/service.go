package payment

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"project.local/commerce-api/internal/order"
)

var paymentUUIDPattern = regexp.MustCompile(
	`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`,
)

var sha256Pattern = regexp.MustCompile(
	`^[0-9a-f]{64}$`,
)

type orderPaymentLifecycle interface {
	MarkPaidTx(
		ctx context.Context,
		tx pgx.Tx,
		orderID string,
		paidAt time.Time,
	) error
}

type Service struct {
	repository *Repository
	orders     orderPaymentLifecycle
}

func NewService(
	repository *Repository,
	orderService orderPaymentLifecycle,
) *Service {
	return &Service{
		repository: repository,
		orders:     orderService,
	}
}

// ConfirmVerifiedPayment must only be called after a provider
// adapter has authenticated and verified the provider response.
//
// It deliberately does not trust a browser/client to mark an
// Order as paid.
func (s *Service) ConfirmVerifiedPayment(
	ctx context.Context,
	input ConfirmVerifiedPaymentInput,
) (ConfirmVerifiedPaymentResult, error) {
	if err :=
		normalizeAndValidateVerifiedPayment(
			&input,
		); err != nil {
		return ConfirmVerifiedPaymentResult{},
			err
	}

	tx, err :=
		s.repository.Begin(
			ctx,
		)
	if err != nil {
		return ConfirmVerifiedPaymentResult{},
			err
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	eventID, inserted, err :=
		s.repository.InsertReceivedEventTx(
			ctx,
			tx,
			input,
		)
	if err != nil {
		return ConfirmVerifiedPaymentResult{},
			err
	}

	// Exact provider-event replay.
	if !inserted {
		existing, found, err :=
			s.repository.GetByProviderPaymentIDTx(
				ctx,
				tx,
				input.Provider,
				input.ProviderPaymentID,
			)
		if err != nil {
			return ConfirmVerifiedPaymentResult{},
				err
		}

		if !found {
			return ConfirmVerifiedPaymentResult{
				Processed: false,
			}, nil
		}

		return ConfirmVerifiedPaymentResult{
			Payment:   existing,
			Processed: false,
		}, nil
	}

	orderSnapshot, err :=
		s.repository.LockOrderPaymentSnapshotTx(
			ctx,
			tx,
			input.OrderID,
		)
	if err != nil {
		return ConfirmVerifiedPaymentResult{},
			err
	}

	if orderSnapshot.PaymentMethod !=
		input.Provider {
		return ConfirmVerifiedPaymentResult{},
			s.failVerifiedEvent(
				ctx,
				tx,
				eventID,
				ErrPaymentMethodMismatch,
			)
	}

	if orderSnapshot.TotalAmount !=
		input.Amount {
		return ConfirmVerifiedPaymentResult{},
			s.failVerifiedEvent(
				ctx,
				tx,
				eventID,
				ErrPaymentAmountMismatch,
			)
	}

	if orderSnapshot.Currency !=
		input.Currency {
		return ConfirmVerifiedPaymentResult{},
			s.failVerifiedEvent(
				ctx,
				tx,
				eventID,
				ErrPaymentCurrencyMismatch,
			)
	}

	successfulPayment,
		hasSuccessfulPayment,
		err :=
		s.repository.GetSucceededByOrderIDTx(
			ctx,
			tx,
			input.OrderID,
		)
	if err != nil {
		return ConfirmVerifiedPaymentResult{},
			err
	}

	if hasSuccessfulPayment &&
		(successfulPayment.Provider !=
			input.Provider ||
			successfulPayment.ProviderPaymentID !=
				input.ProviderPaymentID) {

		err :=
			s.failVerifiedEvent(
				ctx,
				tx,
				eventID,
				ErrReconciliationRequired,
			)

		return ConfirmVerifiedPaymentResult{
			Payment: successfulPayment,
		}, err
	}

	existing, found, err :=
		s.repository.GetByProviderPaymentIDTx(
			ctx,
			tx,
			input.Provider,
			input.ProviderPaymentID,
		)
	if err != nil {
		return ConfirmVerifiedPaymentResult{},
			err
	}

	var result Payment

	if found {
		if existing.OrderID !=
			input.OrderID ||
			existing.Amount !=
				input.Amount ||
			existing.Currency !=
				input.Currency {

			return ConfirmVerifiedPaymentResult{},
				s.failVerifiedEvent(
					ctx,
					tx,
					eventID,
					ErrPaymentReferenceConflict,
				)
		}

		if existing.Status ==
			StatusRefunded {

			return ConfirmVerifiedPaymentResult{},
				s.failVerifiedEvent(
					ctx,
					tx,
					eventID,
					ErrPaymentReferenceConflict,
				)
		}

		result, err =
			s.repository.UpdateSucceededPaymentTx(
				ctx,
				tx,
				existing.ID,
				input,
			)
		if err != nil {
			return ConfirmVerifiedPaymentResult{},
				err
		}
	} else {
		// A local initiation attempt may already exist before the
		// provider has supplied provider_payment_id.
		//
		// Bind the verified provider result to that pending row
		// rather than inserting a second payment.
		pending,
			hasPending,
			pendingErr :=
			s.repository.GetPendingByOrderIDTx(
				ctx,
				tx,
				input.OrderID,
			)
		if pendingErr != nil {
			return ConfirmVerifiedPaymentResult{},
				pendingErr
		}

		if hasPending {
			if pending.Provider !=
				input.Provider ||
				pending.Amount !=
					input.Amount ||
				pending.Currency !=
					input.Currency {

				return ConfirmVerifiedPaymentResult{},
					s.failVerifiedEvent(
						ctx,
						tx,
						eventID,
						ErrPaymentReferenceConflict,
					)
			}

			if pending.ProviderPaymentID != "" &&
				pending.ProviderPaymentID !=
					input.ProviderPaymentID {

				return ConfirmVerifiedPaymentResult{},
					s.failVerifiedEvent(
						ctx,
						tx,
						eventID,
						ErrPaymentReferenceConflict,
					)
			}

			result, err =
				s.repository.PromotePendingToSucceededTx(
					ctx,
					tx,
					pending.ID,
					input,
				)
			if err != nil {
				return ConfirmVerifiedPaymentResult{},
					err
			}
		} else {
			result, err =
				s.repository.InsertSucceededPaymentTx(
					ctx,
					tx,
					input,
				)
			if err != nil {
				return ConfirmVerifiedPaymentResult{},
					err
			}
		}
	}

	err =
		s.orders.MarkPaidTx(
			ctx,
			tx,
			input.OrderID,
			input.PaidAt,
		)

	if errors.Is(
		err,
		order.ErrPaymentWindowExpired,
	) ||
		errors.Is(
			err,
			order.ErrOrderNotPendingPayment,
		) {

		if failureErr :=
			s.repository.MarkEventFailedTx(
				ctx,
				tx,
				eventID,
				ErrReconciliationRequired.Error(),
			); failureErr != nil {

			return ConfirmVerifiedPaymentResult{},
				failureErr
		}

		if commitErr :=
			tx.Commit(
				ctx,
			); commitErr != nil {

			return ConfirmVerifiedPaymentResult{},
				fmt.Errorf(
					"commit payment reconciliation state: %w",
					commitErr,
				)
		}

		return ConfirmVerifiedPaymentResult{
				Payment: result,

				Processed: true,
			},
			ErrReconciliationRequired
	}

	if err != nil {
		return ConfirmVerifiedPaymentResult{},
			err
	}

	if err :=
		s.repository.MarkEventProcessedTx(
			ctx,
			tx,
			eventID,
			result.ID,
		); err != nil {

		return ConfirmVerifiedPaymentResult{},
			err
	}

	if err :=
		tx.Commit(
			ctx,
		); err != nil {

		return ConfirmVerifiedPaymentResult{},
			fmt.Errorf(
				"commit verified payment: %w",
				err,
			)
	}

	return ConfirmVerifiedPaymentResult{
		Payment:   result,
		Processed: true,
	}, nil
}

func (s *Service) GetLatestByOrderID(
	ctx context.Context,
	orderID string,
) (Payment, error) {
	orderID =
		strings.TrimSpace(
			orderID,
		)

	if !paymentUUIDPattern.MatchString(
		orderID,
	) {
		return Payment{},
			ErrInvalidInput
	}

	return s.repository.GetLatestByOrderID(
		ctx,
		orderID,
	)
}

func (s *Service) failVerifiedEvent(
	ctx context.Context,
	tx pgx.Tx,
	eventID string,
	cause error,
) error {
	if err :=
		s.repository.MarkEventFailedTx(
			ctx,
			tx,
			eventID,
			cause.Error(),
		); err != nil {

		return err
	}

	if err :=
		tx.Commit(
			ctx,
		); err != nil {

		return fmt.Errorf(
			"commit failed payment event: %w",
			err,
		)
	}

	return cause
}

func normalizeAndValidateVerifiedPayment(
	input *ConfirmVerifiedPaymentInput,
) error {
	input.OrderID =
		strings.TrimSpace(
			input.OrderID,
		)

	input.Provider =
		strings.ToLower(
			strings.TrimSpace(
				input.Provider,
			),
		)

	input.ProviderEventID =
		strings.TrimSpace(
			input.ProviderEventID,
		)

	input.EventType =
		strings.TrimSpace(
			input.EventType,
		)

	input.ProviderPaymentID =
		strings.TrimSpace(
			input.ProviderPaymentID,
		)

	input.ProviderTransactionID =
		strings.TrimSpace(
			input.ProviderTransactionID,
		)

	input.Currency =
		strings.ToUpper(
			strings.TrimSpace(
				input.Currency,
			),
		)

	input.PayloadSHA256 =
		strings.ToLower(
			strings.TrimSpace(
				input.PayloadSHA256,
			),
		)

	if !paymentUUIDPattern.MatchString(
		input.OrderID,
	) {
		return ErrInvalidInput
	}

	switch input.Provider {
	case ProviderBKash,
		ProviderNagad,
		ProviderRocket,
		ProviderBankTransfer:
	default:
		return ErrUnsupportedProvider
	}

	if input.ProviderEventID == "" ||
		len(input.ProviderEventID) > 200 {

		return ErrInvalidInput
	}

	if input.EventType == "" ||
		len(input.EventType) > 100 {

		return ErrInvalidInput
	}

	if input.ProviderPaymentID == "" ||
		len(input.ProviderPaymentID) > 160 {

		return ErrInvalidInput
	}

	if len(
		input.ProviderTransactionID,
	) > 160 {

		return ErrInvalidInput
	}

	if input.Amount < 0 {
		return ErrInvalidInput
	}

	if len(
		input.Currency,
	) != 3 {

		return ErrInvalidInput
	}

	if !sha256Pattern.MatchString(
		input.PayloadSHA256,
	) {
		return ErrInvalidInput
	}

	now :=
		time.Now().UTC()

	if input.PaidAt.IsZero() ||
		input.PaidAt.After(
			now,
		) {

		input.PaidAt =
			now
	} else {
		input.PaidAt =
			input.PaidAt.UTC()
	}

	return nil
}
