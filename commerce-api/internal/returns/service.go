package returns

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
	"time"

	"project.local/commerce-api/internal/inventory"
)

var uuidPattern = regexp.MustCompile(
	`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`,
)

type Service struct {
	repository *Repository
	inventory  *inventory.Service
}

func NewService(
	repository *Repository,
	inventoryService *inventory.Service,
) *Service {
	return &Service{
		repository: repository,
		inventory:  inventoryService,
	}
}

func (s *Service) Create(
	ctx context.Context,
	orderID string,
	request CreateRequest,
) (Return, error) {
	orderID = strings.TrimSpace(
		orderID,
	)

	if !uuidPattern.MatchString(
		orderID,
	) {
		return Return{},
			ErrInvalidInput
	}

	if err := normalizeCreateRequest(
		&request,
	); err != nil {
		return Return{}, err
	}

	tx, err := s.repository.Begin(
		ctx,
	)
	if err != nil {
		return Return{}, err
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	order, err := s.repository.LockOrderTx(
		ctx,
		tx,
		orderID,
	)
	if err != nil {
		return Return{}, err
	}

	if !orderEligibleForReturn(
		order.Status,
	) {
		return Return{},
			ErrReturnNotAllowed
	}

	orderItems, err := s.repository.ListOrderItemsTx(
		ctx,
		tx,
		orderID,
	)
	if err != nil {
		return Return{}, err
	}

	itemMap := make(
		map[string]orderItemSnapshot,
		len(orderItems),
	)

	for _, item := range orderItems {
		itemMap[item.ID] = item
	}

	existingQuantities, err := s.repository.ExistingReturnQuantitiesTx(
		ctx,
		tx,
		orderID,
	)
	if err != nil {
		return Return{}, err
	}

	for _, requestItem := range request.Items {
		orderItem, exists := itemMap[requestItem.OrderItemID]

		if !exists {
			return Return{},
				ErrReturnItemNotFound
		}

		alreadyReturned := existingQuantities[requestItem.OrderItemID]

		remainingQuantity :=
			orderItem.Quantity -
				alreadyReturned

		if requestItem.Quantity >
			remainingQuantity {
			return Return{},
				ErrReturnQuantityExceeded
		}
	}

	returnNumber, err := generateReturnNumber(
		time.Now().UTC(),
	)
	if err != nil {
		return Return{},
			fmt.Errorf(
				"generate return number: %w",
				err,
			)
	}

	returnID, err := s.repository.CreateTx(
		ctx,
		tx,
		returnNumber,
		orderID,
		request.CustomerNote,
		"customer",
	)
	if err != nil {
		return Return{}, err
	}

	if err := s.repository.InsertItemsTx(
		ctx,
		tx,
		returnID,
		request.Items,
	); err != nil {
		return Return{}, err
	}

	if err := s.repository.InsertEventTx(
		ctx,
		tx,
		returnID,
		eventInsert{
			EventType: EventRequested,

			ToStatus: StatusRequested,

			Message: "Return requested",

			ActorType: "customer",
		},
	); err != nil {
		return Return{}, err
	}

	if err := tx.Commit(
		ctx,
	); err != nil {
		return Return{},
			fmt.Errorf(
				"commit return request: %w",
				err,
			)
	}

	return s.repository.GetByID(
		ctx,
		returnID,
	)
}

func (s *Service) Get(
	ctx context.Context,
	returnID string,
) (Return, error) {
	returnID = strings.TrimSpace(
		returnID,
	)

	if !uuidPattern.MatchString(
		returnID,
	) {
		return Return{},
			ErrInvalidInput
	}

	return s.repository.GetByID(
		ctx,
		returnID,
	)
}

func (s *Service) Timeline(
	ctx context.Context,
	returnID string,
) ([]Event, error) {
	returnID = strings.TrimSpace(
		returnID,
	)

	if !uuidPattern.MatchString(
		returnID,
	) {
		return nil,
			ErrInvalidInput
	}

	if _, err := s.repository.GetByID(
		ctx,
		returnID,
	); err != nil {
		return nil, err
	}

	return s.repository.ListEvents(
		ctx,
		returnID,
	)
}

func (s *Service) Cancel(
	ctx context.Context,
	returnID string,
) (Return, error) {
	returnID = strings.TrimSpace(
		returnID,
	)

	if !uuidPattern.MatchString(
		returnID,
	) {
		return Return{},
			ErrInvalidInput
	}

	tx, err := s.repository.Begin(
		ctx,
	)
	if err != nil {
		return Return{}, err
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	current, err := s.repository.LockByIDTx(
		ctx,
		tx,
		returnID,
	)
	if err != nil {
		return Return{}, err
	}

	// Idempotent cancellation.
	if current.Status ==
		StatusCancelled {
		return current, nil
	}

	// Customer cancellation is only allowed
	// while the return is still requested.
	if current.Status !=
		StatusRequested {
		return Return{},
			ErrInvalidReturnTransition
	}

	if err := s.repository.MarkCancelledTx(
		ctx,
		tx,
		returnID,
	); err != nil {
		return Return{}, err
	}

	if err := s.repository.InsertEventTx(
		ctx,
		tx,
		returnID,
		eventInsert{
			EventType: EventCancelled,

			FromStatus: StatusRequested,

			ToStatus: StatusCancelled,

			Message: "Return cancelled",

			ActorType: "customer",
		},
	); err != nil {
		return Return{}, err
	}

	if err := tx.Commit(
		ctx,
	); err != nil {
		return Return{},
			fmt.Errorf(
				"commit return cancellation: %w",
				err,
			)
	}

	return s.repository.GetByID(
		ctx,
		returnID,
	)
}

func (s *Service) Approve(
	ctx context.Context,
	returnID string,
	actorID string,
) (Return, error) {
	returnID = strings.TrimSpace(
		returnID,
	)

	if !uuidPattern.MatchString(
		returnID,
	) {
		return Return{},
			ErrInvalidInput
	}

	actorID = normalizeActorID(
		actorID,
	)

	tx, err := s.repository.Begin(
		ctx,
	)
	if err != nil {
		return Return{}, err
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	current, err := s.repository.LockByIDTx(
		ctx,
		tx,
		returnID,
	)
	if err != nil {
		return Return{}, err
	}

	// Idempotent approval.
	if current.Status ==
		StatusApproved {
		return current, nil
	}

	if current.Status !=
		StatusRequested {
		return Return{},
			ErrInvalidReturnTransition
	}

	if err := s.repository.MarkApprovedTx(
		ctx,
		tx,
		returnID,
		actorID,
	); err != nil {
		return Return{}, err
	}

	if err := s.repository.InsertEventTx(
		ctx,
		tx,
		returnID,
		eventInsert{
			EventType: EventApproved,

			FromStatus: StatusRequested,

			ToStatus: StatusApproved,

			Message: "Return approved",

			ActorType: "admin",

			ActorID: actorID,
		},
	); err != nil {
		return Return{}, err
	}

	if err := tx.Commit(
		ctx,
	); err != nil {
		return Return{},
			fmt.Errorf(
				"commit return approval: %w",
				err,
			)
	}

	return s.repository.GetByID(
		ctx,
		returnID,
	)
}

func (s *Service) Reject(
	ctx context.Context,
	returnID string,
	request RejectRequest,
	actorID string,
) (Return, error) {
	returnID = strings.TrimSpace(
		returnID,
	)

	if !uuidPattern.MatchString(
		returnID,
	) {
		return Return{},
			ErrInvalidInput
	}

	request.Reason = strings.TrimSpace(
		request.Reason,
	)

	if request.Reason == "" ||
		len(request.Reason) > 500 {
		return Return{},
			ErrRejectionReasonRequired
	}

	actorID = normalizeActorID(
		actorID,
	)

	tx, err := s.repository.Begin(
		ctx,
	)
	if err != nil {
		return Return{}, err
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	current, err := s.repository.LockByIDTx(
		ctx,
		tx,
		returnID,
	)
	if err != nil {
		return Return{}, err
	}

	// Idempotent rejection.
	if current.Status ==
		StatusRejected {
		return current, nil
	}

	if current.Status !=
		StatusRequested {
		return Return{},
			ErrInvalidReturnTransition
	}

	if err := s.repository.MarkRejectedTx(
		ctx,
		tx,
		returnID,
		actorID,
		request.Reason,
	); err != nil {
		return Return{}, err
	}

	if err := s.repository.InsertEventTx(
		ctx,
		tx,
		returnID,
		eventInsert{
			EventType: EventRejected,

			FromStatus: StatusRequested,

			ToStatus: StatusRejected,

			Message: request.Reason,

			ActorType: "admin",

			ActorID: actorID,
		},
	); err != nil {
		return Return{}, err
	}

	if err := tx.Commit(
		ctx,
	); err != nil {
		return Return{},
			fmt.Errorf(
				"commit return rejection: %w",
				err,
			)
	}

	return s.repository.GetByID(
		ctx,
		returnID,
	)
}

func (s *Service) Receive(
	ctx context.Context,
	returnID string,
	request ReceiveRequest,
	actorID string,
) (Return, error) {
	returnID = strings.TrimSpace(
		returnID,
	)

	if !uuidPattern.MatchString(
		returnID,
	) {
		return Return{},
			ErrInvalidInput
	}

	actorID = normalizeActorID(
		actorID,
	)

	tx, err := s.repository.Begin(
		ctx,
	)
	if err != nil {
		return Return{}, err
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	current, err := s.repository.LockByIDTx(
		ctx,
		tx,
		returnID,
	)
	if err != nil {
		return Return{}, err
	}

	// Idempotent receipt.
	if current.Status ==
		StatusReceived {
		return current, nil
	}

	if current.Status !=
		StatusApproved {
		return Return{},
			ErrInvalidReturnTransition
	}

	if err := validateReceiveRequest(
		current,
		&request,
	); err != nil {
		return Return{}, err
	}

	if err := s.repository.UpdateReceivedItemsTx(
		ctx,
		tx,
		returnID,
		request.Items,
	); err != nil {
		return Return{}, err
	}

	if err := s.repository.MarkReceivedTx(
		ctx,
		tx,
		returnID,
	); err != nil {
		return Return{}, err
	}

	if err := s.repository.InsertEventTx(
		ctx,
		tx,
		returnID,
		eventInsert{
			EventType: EventReceived,

			FromStatus: StatusApproved,

			ToStatus: StatusReceived,

			Message: "Returned items received",

			ActorType: "admin",

			ActorID: actorID,
		},
	); err != nil {
		return Return{}, err
	}

	if err := tx.Commit(
		ctx,
	); err != nil {
		return Return{},
			fmt.Errorf(
				"commit return receipt: %w",
				err,
			)
	}

	return s.repository.GetByID(
		ctx,
		returnID,
	)
}

func (s *Service) Inspect(
	ctx context.Context,
	returnID string,
	request InspectRequest,
	actorID string,
) (Return, error) {
	returnID = strings.TrimSpace(
		returnID,
	)

	if !uuidPattern.MatchString(
		returnID,
	) {
		return Return{},
			ErrInvalidInput
	}

	actorID = normalizeActorID(
		actorID,
	)

	tx, err := s.repository.Begin(
		ctx,
	)
	if err != nil {
		return Return{}, err
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	current, err := s.repository.LockByIDTx(
		ctx,
		tx,
		returnID,
	)
	if err != nil {
		return Return{}, err
	}

	// Prevent duplicate inventory restocking on retries.
	if current.Status ==
		StatusInspected {
		return current, nil
	}

	if current.Status !=
		StatusReceived {
		return Return{},
			ErrInvalidReturnTransition
	}

	if err := validateInspectRequest(
		current,
		&request,
	); err != nil {
		return Return{}, err
	}

	itemMap := make(
		map[string]Item,
		len(current.Items),
	)

	for _, item := range current.Items {
		itemMap[item.OrderItemID] =
			item
	}

	for _, inspection := range request.Items {
		currentItem, exists := itemMap[inspection.OrderItemID]

		if !exists {
			return Return{},
				ErrReturnItemNotFound
		}

		if inspection.RestockQuantity > 0 {
			if err := s.inventory.AdjustReferenceTx(
				ctx,
				tx,
				inventory.AdjustReferenceInput{
					VariantID: currentItem.VariantID,

					QuantityDelta: inspection.RestockQuantity,

					ReferenceType: "return",

					ReferenceID: returnID,

					Reason: "return_restock",

					Note: inspection.InspectionNote,

					ActorType: "return",

					ActorID: returnID,
				},
			); err != nil {
				return Return{},
					fmt.Errorf(
						"%w: %v",
						ErrInventoryRestockFailed,
						err,
					)
			}
		}

		if err := s.repository.UpdateInspectedItemTx(
			ctx,
			tx,
			returnID,
			inspection,
		); err != nil {
			return Return{}, err
		}
	}

	if err := s.repository.MarkInspectedTx(
		ctx,
		tx,
		returnID,
	); err != nil {
		return Return{}, err
	}

	if err := s.repository.InsertEventTx(
		ctx,
		tx,
		returnID,
		eventInsert{
			EventType: EventInspected,

			FromStatus: StatusReceived,

			ToStatus: StatusInspected,

			Message: "Returned items inspected",

			ActorType: "admin",

			ActorID: actorID,
		},
	); err != nil {
		return Return{}, err
	}

	if err := tx.Commit(
		ctx,
	); err != nil {
		return Return{},
			fmt.Errorf(
				"commit return inspection: %w",
				err,
			)
	}

	return s.repository.GetByID(
		ctx,
		returnID,
	)
}

func normalizeCreateRequest(
	request *CreateRequest,
) error {
	request.CustomerNote = strings.TrimSpace(
		request.CustomerNote,
	)

	if len(
		request.CustomerNote,
	) > 1000 {
		return ErrInvalidInput
	}

	if len(
		request.Items,
	) == 0 {
		return ErrInvalidInput
	}

	seen := make(
		map[string]struct{},
	)

	for index := range request.Items {
		item := &request.Items[index]

		item.OrderItemID = strings.TrimSpace(
			item.OrderItemID,
		)

		item.ReasonCode = strings.ToLower(
			strings.TrimSpace(
				item.ReasonCode,
			),
		)

		item.ReasonNote = strings.TrimSpace(
			item.ReasonNote,
		)

		if !uuidPattern.MatchString(
			item.OrderItemID,
		) ||
			item.Quantity <= 0 ||
			!validReasonCode(
				item.ReasonCode,
			) ||
			len(
				item.ReasonNote,
			) > 500 {
			return ErrInvalidInput
		}

		if _, exists := seen[item.OrderItemID]; exists {
			return ErrDuplicateReturnItem
		}

		seen[item.OrderItemID] =
			struct{}{}
	}

	return nil
}

func normalizeActorID(
	actorID string,
) string {
	actorID = strings.TrimSpace(
		actorID,
	)

	if actorID == "" {
		return "development-admin"
	}

	return actorID
}

func generateReturnNumber(
	now time.Time,
) (string, error) {
	buffer := make(
		[]byte,
		6,
	)

	if _, err := rand.Read(
		buffer,
	); err != nil {
		return "", err
	}

	return fmt.Sprintf(
		"RET-%s-%s",
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
