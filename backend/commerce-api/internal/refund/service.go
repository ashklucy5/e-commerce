package refund

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

var uuidPattern = regexp.MustCompile(
	`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`,
)

type Service struct {
	repository *Repository
}

func NewService(
	repository *Repository,
) *Service {
	return &Service{
		repository: repository,
	}
}

func (s *Service) RequestForReturn(
	ctx context.Context,
	returnID string,
	reason string,
	actorID string,
) (Refund, error) {
	returnID =
		strings.TrimSpace(
			returnID,
		)

	if !uuidPattern.MatchString(
		returnID,
	) {
		return Refund{},
			ErrInvalidInput
	}

	reason =
		strings.TrimSpace(
			reason,
		)

	if reason == "" {
		reason = "Refund for returned items"
	}

	if len(reason) > 500 {
		return Refund{},
			ErrInvalidInput
	}

	actorID =
		normalizeActorID(
			actorID,
		)

	tx, err :=
		s.repository.Begin(
			ctx,
		)
	if err != nil {
		return Refund{}, err
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	returnSnapshot, err :=
		s.repository.LockReturnTx(
			ctx,
			tx,
			returnID,
		)
	if err != nil {
		return Refund{}, err
	}

	if returnSnapshot.Status !=
		"inspected" {
		return Refund{},
			ErrReturnNotReady
	}

	if existing, found, err :=
		s.repository.ExistingByReturnIDTx(
			ctx,
			tx,
			returnID,
		); err != nil {
		return Refund{}, err
	} else if found {
		return existing, nil
	}

	if !paymentStateEligibleForRefund(
		returnSnapshot.PaymentMethod,
		returnSnapshot.PaymentStatus,
	) {
		return Refund{},
			ErrRefundNotAllowed
	}

	provider, ok :=
		providerForPaymentMethod(
			returnSnapshot.PaymentMethod,
		)
	if !ok {
		return Refund{},
			ErrRefundNotAllowed
	}

	grossReturnAmount, err :=
		s.repository.ReturnGrossAmountTx(
			ctx,
			tx,
			returnID,
		)
	if err != nil {
		return Refund{}, err
	}

	refundAmount, err :=
		calculateReturnRefundAmount(
			grossReturnAmount,
			returnSnapshot.SubtotalAmount,
			returnSnapshot.DiscountAmount,
		)
	if err != nil {
		return Refund{}, err
	}

	committedAmount, err :=
		s.repository.CommittedRefundAmountTx(
			ctx,
			tx,
			returnSnapshot.OrderID,
		)
	if err != nil {
		return Refund{}, err
	}

	remaining :=
		returnSnapshot.TotalAmount -
			committedAmount

	if remaining <= 0 ||
		refundAmount > remaining {
		return Refund{},
			ErrRefundAmountExceeded
	}

	paymentID := ""

	if provider != ProviderManual {
		paymentSnapshot, err :=
			s.repository.LockPaymentForOrderTx(
				ctx,
				tx,
				returnSnapshot.OrderID,
			)
		if err != nil {
			return Refund{}, err
		}

		if paymentSnapshot.Provider != provider ||
			paymentSnapshot.Currency !=
				returnSnapshot.Currency {
			return Refund{},
				ErrRefundNotAllowed
		}

		paymentID =
			paymentSnapshot.ID
	}

	refundNumber, err :=
		generateRefundNumber(
			time.Now().UTC(),
		)
	if err != nil {
		return Refund{},
			fmt.Errorf(
				"generate refund number: %w",
				err,
			)
	}

	refundID, err :=
		s.repository.CreateReturnRefundTx(
			ctx,
			tx,
			refundNumber,
			returnSnapshot,
			paymentID,
			provider,
			refundAmount,
			reason,
			actorID,
		)
	if err != nil {
		return Refund{}, err
	}

	if err :=
		s.repository.InsertEventTx(
			ctx,
			tx,
			refundID,
			eventInsert{
				EventType: EventRequested,

				ToStatus: StatusRequested,

				Message: reason,

				ActorType: "admin",

				ActorID: actorID,
			},
		); err != nil {
		return Refund{}, err
	}

	if err :=
		tx.Commit(
			ctx,
		); err != nil {
		return Refund{},
			fmt.Errorf(
				"commit refund request: %w",
				err,
			)
	}

	return s.repository.GetByID(
		ctx,
		refundID,
	)
}

func (s *Service) RequestCancellationTx(
	ctx context.Context,
	tx pgx.Tx,
	orderID string,
	reason string,
	actorID string,
) (string, error) {
	if tx == nil {
		return "",
			fmt.Errorf(
				"cancellation refund transaction is required",
			)
	}

	orderID =
		strings.TrimSpace(
			orderID,
		)

	if !uuidPattern.MatchString(
		orderID,
	) {
		return "",
			ErrInvalidInput
	}

	reason =
		strings.TrimSpace(
			reason,
		)

	if reason == "" ||
		len(reason) > 500 {
		return "",
			ErrInvalidInput
	}

	actorID =
		strings.TrimSpace(
			actorID,
		)

	if actorID == "" {
		actorID = "customer"
	}

	orderSnapshot, err :=
		s.repository.LockCancellationOrderTx(
			ctx,
			tx,
			orderID,
		)
	if err != nil {
		return "", err
	}

	if orderSnapshot.Status != "confirmed" &&
		orderSnapshot.Status != "processing" {
		return "",
			ErrRefundNotAllowed
	}

	if orderSnapshot.PaymentStatus !=
		"paid" {
		return "",
			ErrRefundNotAllowed
	}

	provider, ok :=
		providerForPaymentMethod(
			orderSnapshot.PaymentMethod,
		)
	if !ok ||
		provider == ProviderManual {
		return "",
			ErrRefundNotAllowed
	}

	if existing, found, err :=
		s.repository.ExistingCancellationByOrderIDTx(
			ctx,
			tx,
			orderID,
		); err != nil {
		return "", err
	} else if found {
		return existing.ID, nil
	}

	paymentSnapshot, err :=
		s.repository.LockPaymentForOrderTx(
			ctx,
			tx,
			orderID,
		)
	if err != nil {
		return "", err
	}

	if paymentSnapshot.Status !=
		"succeeded" {
		return "",
			ErrRefundNotAllowed
	}

	if paymentSnapshot.Provider != provider ||
		paymentSnapshot.Currency !=
			orderSnapshot.Currency {
		return "",
			ErrRefundNotAllowed
	}

	committedAmount, err :=
		s.repository.CommittedRefundAmountTx(
			ctx,
			tx,
			orderID,
		)
	if err != nil {
		return "", err
	}

	refundAmount :=
		orderSnapshot.TotalAmount -
			committedAmount

	if refundAmount <= 0 {
		return "",
			ErrNothingToRefund
	}

	refundNumber, err :=
		generateRefundNumber(
			time.Now().UTC(),
		)
	if err != nil {
		return "",
			fmt.Errorf(
				"generate cancellation refund number: %w",
				err,
			)
	}

	refundID, err :=
		s.repository.CreateCancellationRefundTx(
			ctx,
			tx,
			refundNumber,
			orderID,
			paymentSnapshot.ID,
			refundAmount,
			orderSnapshot.Currency,
			provider,
			reason,
			actorID,
		)
	if err != nil {
		return "", err
	}

	if err :=
		s.repository.InsertEventTx(
			ctx,
			tx,
			refundID,
			eventInsert{
				EventType: EventRequested,

				ToStatus: StatusRequested,

				Message: reason,

				ActorType: "customer",

				ActorID: actorID,
			},
		); err != nil {
		return "", err
	}

	return refundID, nil
}

func (s *Service) Get(
	ctx context.Context,
	refundID string,
) (Refund, error) {
	refundID =
		strings.TrimSpace(
			refundID,
		)

	if !uuidPattern.MatchString(
		refundID,
	) {
		return Refund{},
			ErrInvalidInput
	}

	return s.repository.GetByID(
		ctx,
		refundID,
	)
}

func (s *Service) Timeline(
	ctx context.Context,
	refundID string,
) ([]Event, error) {
	refundID =
		strings.TrimSpace(
			refundID,
		)

	if !uuidPattern.MatchString(
		refundID,
	) {
		return nil,
			ErrInvalidInput
	}

	if _, err :=
		s.repository.GetByID(
			ctx,
			refundID,
		); err != nil {
		return nil, err
	}

	return s.repository.ListEvents(
		ctx,
		refundID,
	)
}

func (s *Service) Approve(
	ctx context.Context,
	refundID string,
	actorID string,
) (Refund, error) {
	refundID =
		strings.TrimSpace(
			refundID,
		)

	if !uuidPattern.MatchString(
		refundID,
	) {
		return Refund{},
			ErrInvalidInput
	}

	actorID =
		normalizeActorID(
			actorID,
		)

	tx, err :=
		s.repository.Begin(
			ctx,
		)
	if err != nil {
		return Refund{}, err
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	current, err :=
		s.repository.LockByIDTx(
			ctx,
			tx,
			refundID,
		)
	if err != nil {
		return Refund{}, err
	}

	if current.Status ==
		StatusApproved {
		return current, nil
	}

	if current.Status !=
		StatusRequested {
		return Refund{},
			ErrInvalidRefundTransition
	}

	if err :=
		s.repository.MarkApprovedTx(
			ctx,
			tx,
			refundID,
			actorID,
		); err != nil {
		return Refund{}, err
	}

	if err :=
		s.repository.InsertEventTx(
			ctx,
			tx,
			refundID,
			eventInsert{
				EventType: EventApproved,

				FromStatus: StatusRequested,

				ToStatus: StatusApproved,

				Message: "Refund approved",

				ActorType: "admin",

				ActorID: actorID,
			},
		); err != nil {
		return Refund{}, err
	}

	if err :=
		tx.Commit(
			ctx,
		); err != nil {
		return Refund{},
			fmt.Errorf(
				"commit refund approval: %w",
				err,
			)
	}

	return s.repository.GetByID(
		ctx,
		refundID,
	)
}

func (s *Service) StartProcessing(
	ctx context.Context,
	refundID string,
	actorID string,
) (Refund, error) {
	refundID =
		strings.TrimSpace(
			refundID,
		)

	if !uuidPattern.MatchString(
		refundID,
	) {
		return Refund{},
			ErrInvalidInput
	}

	actorID =
		normalizeActorID(
			actorID,
		)

	tx, err :=
		s.repository.Begin(
			ctx,
		)
	if err != nil {
		return Refund{}, err
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	current, err :=
		s.repository.LockByIDTx(
			ctx,
			tx,
			refundID,
		)
	if err != nil {
		return Refund{}, err
	}

	if current.Status ==
		StatusProcessing {
		return current, nil
	}

	fromStatus :=
		current.Status

	if fromStatus !=
		StatusApproved &&
		fromStatus !=
			StatusFailed {
		return Refund{},
			ErrInvalidRefundTransition
	}

	if err :=
		s.repository.MarkProcessingTx(
			ctx,
			tx,
			refundID,
		); err != nil {
		return Refund{}, err
	}

	if err :=
		s.repository.InsertEventTx(
			ctx,
			tx,
			refundID,
			eventInsert{
				EventType: EventProcessing,

				FromStatus: fromStatus,

				ToStatus: StatusProcessing,

				Message: "Refund processing started",

				ActorType: "admin",

				ActorID: actorID,
			},
		); err != nil {
		return Refund{}, err
	}

	if err :=
		tx.Commit(
			ctx,
		); err != nil {
		return Refund{},
			fmt.Errorf(
				"commit refund processing: %w",
				err,
			)
	}

	return s.repository.GetByID(
		ctx,
		refundID,
	)
}

func (s *Service) Fail(
	ctx context.Context,
	refundID string,
	request FailureRequest,
	actorID string,
) (Refund, error) {
	refundID =
		strings.TrimSpace(
			refundID,
		)

	if !uuidPattern.MatchString(
		refundID,
	) {
		return Refund{},
			ErrInvalidInput
	}

	if err :=
		normalizeFailureRequest(
			&request,
		); err != nil {
		return Refund{}, err
	}

	actorID =
		normalizeActorID(
			actorID,
		)

	tx, err :=
		s.repository.Begin(
			ctx,
		)
	if err != nil {
		return Refund{}, err
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	current, err :=
		s.repository.LockByIDTx(
			ctx,
			tx,
			refundID,
		)
	if err != nil {
		return Refund{}, err
	}

	if current.Status ==
		StatusFailed {
		return current, nil
	}

	if current.Status !=
		StatusProcessing {
		return Refund{},
			ErrInvalidRefundTransition
	}

	if err :=
		s.repository.MarkFailedTx(
			ctx,
			tx,
			refundID,
			request,
		); err != nil {
		return Refund{}, err
	}

	if err :=
		s.repository.InsertEventTx(
			ctx,
			tx,
			refundID,
			eventInsert{
				EventType: EventFailed,

				FromStatus: StatusProcessing,

				ToStatus: StatusFailed,

				Message: request.FailureMessage,

				ActorType: "admin",

				ActorID: actorID,
			},
		); err != nil {
		return Refund{}, err
	}

	if err :=
		tx.Commit(
			ctx,
		); err != nil {
		return Refund{},
			fmt.Errorf(
				"commit failed refund: %w",
				err,
			)
	}

	return s.repository.GetByID(
		ctx,
		refundID,
	)
}

func (s *Service) Succeed(
	ctx context.Context,
	refundID string,
	request SuccessRequest,
	actorID string,
) (Refund, error) {
	refundID =
		strings.TrimSpace(
			refundID,
		)

	if !uuidPattern.MatchString(
		refundID,
	) {
		return Refund{},
			ErrInvalidInput
	}

	if err :=
		normalizeSuccessRequest(
			&request,
		); err != nil {
		return Refund{}, err
	}

	actorID =
		normalizeActorID(
			actorID,
		)

	tx, err :=
		s.repository.Begin(
			ctx,
		)
	if err != nil {
		return Refund{}, err
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	current, err :=
		s.repository.LockByIDTx(
			ctx,
			tx,
			refundID,
		)
	if err != nil {
		return Refund{}, err
	}

	if current.Status ==
		StatusSucceeded {
		return current, nil
	}

	if current.Status !=
		StatusProcessing {
		return Refund{},
			ErrInvalidRefundTransition
	}

	if err :=
		s.repository.MarkSucceededTx(
			ctx,
			tx,
			refundID,
			request.ProviderRefundID,
		); err != nil {
		return Refund{}, err
	}

	successfulAmount, err :=
		s.repository.SuccessfulRefundAmountTx(
			ctx,
			tx,
			current.OrderID,
		)
	if err != nil {
		return Refund{}, err
	}

	var orderTotal int64

	const orderQuery = `
		SELECT total_amount
		FROM orders
		WHERE id = $1::uuid
		FOR UPDATE
	`

	if err :=
		tx.QueryRow(
			ctx,
			orderQuery,
			current.OrderID,
		).Scan(
			&orderTotal,
		); err != nil {
		return Refund{},
			fmt.Errorf(
				"load refund order total: %w",
				err,
			)
	}

	// Partial refunds keep the original payment state.
	// Only a cumulative full refund marks the order/payment refunded.
	if successfulAmount >=
		orderTotal {
		if err :=
			s.repository.MarkOrderFullyRefundedTx(
				ctx,
				tx,
				current.OrderID,
			); err != nil {
			return Refund{}, err
		}

		if current.PaymentID != "" {
			if err :=
				s.repository.MarkPaymentFullyRefundedTx(
					ctx,
					tx,
					current.PaymentID,
				); err != nil {
				return Refund{}, err
			}
		}
	}

	if current.ReturnID != "" {
		if err :=
			s.repository.CompleteReturnTx(
				ctx,
				tx,
				current.ReturnID,
				current.ID,
			); err != nil {
			return Refund{}, err
		}
	}

	if err :=
		s.repository.InsertEventTx(
			ctx,
			tx,
			refundID,
			eventInsert{
				EventType: EventSucceeded,

				FromStatus: StatusProcessing,

				ToStatus: StatusSucceeded,

				Message: "Refund succeeded",

				ActorType: "admin",

				ActorID: actorID,
			},
		); err != nil {
		return Refund{}, err
	}

	if err :=
		tx.Commit(
			ctx,
		); err != nil {
		return Refund{},
			fmt.Errorf(
				"commit successful refund: %w",
				err,
			)
	}

	return s.repository.GetByID(
		ctx,
		refundID,
	)
}

func normalizeActorID(
	actorID string,
) string {
	actorID =
		strings.TrimSpace(
			actorID,
		)

	if actorID == "" {
		return "development-admin"
	}

	return actorID
}

func generateRefundNumber(
	now time.Time,
) (string, error) {
	buffer :=
		make(
			[]byte,
			6,
		)

	if _, err :=
		rand.Read(
			buffer,
		); err != nil {
		return "", err
	}

	return fmt.Sprintf(
		"REF-%s-%s",
		now.UTC().Format(
			"20060102",
		),
		strings.ToUpper(
			hex.EncodeToString(
				buffer,
			),
		),
	), nil
}
