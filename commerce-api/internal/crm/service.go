package crm

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"project.local/commerce-api/internal/support"
)

const (
	maxCRMAttachmentsCount = 4
	maxCRMAttachmentsBytes = 6 * 1024 * 1024

	maxCRMAttachmentNameRunes = 255
	maxCRMAttachmentURLRunes  = 4096
)

type Service struct {
	repository *Repository
}

func NewService(repository *Repository) *Service {
	return &Service{
		repository: repository,
	}
}

func (s *Service) CreateCase(
	ctx context.Context,
	customerID string,
	request CreateCaseRequest,
) (Case, error) {
	caseType := strings.ToLower(
		strings.TrimSpace(
			request.CaseType,
		),
	)

	subject := strings.Join(
		strings.Fields(
			request.Subject,
		),
		" ",
	)

	message := strings.TrimSpace(
		request.Message,
	)

	if !validCaseType(caseType) {
		return Case{}, ErrInvalidCaseType
	}

	hasAttachments, err :=
		validateCustomerAttachmentEnvelope(
			request.Attachments,
		)
	if err != nil {
		return Case{}, ErrInvalidRequest
	}

	/*
		A conversation may contain only an attachment.

		crm_messages.body is still populated so the
		existing CRM model does not need to change.
	*/
	if message == "" && hasAttachments {
		message = "Shared an attachment."
	}

	if subject == "" ||
		utf8.RuneCountInString(subject) > 180 ||
		message == "" ||
		utf8.RuneCountInString(message) > 5000 {

		return Case{}, ErrInvalidRequest
	}

	productID := strings.TrimSpace(
		request.ProductID,
	)

	variantID := strings.TrimSpace(
		request.VariantID,
	)

	orderID := strings.TrimSpace(
		request.OrderID,
	)

	for _, value := range []string{
		productID,
		variantID,
		orderID,
	} {
		if value != "" &&
			!validUUID(value) {

			return Case{}, ErrInvalidRequest
		}
	}

	if caseType != "bulk_stock_request" &&
		request.RequestedQuantity != 0 {

		return Case{}, ErrInvalidRequestedQuantity
	}

	tx, err := s.repository.Begin(ctx)
	if err != nil {
		return Case{}, err
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	snapshot := ContextSnapshot{}

	var requestedQuantity *int
	var availableQuantity *int

	if variantID != "" {
		product, err := s.repository.GetVariantContextTx(
			ctx,
			tx,
			variantID,
		)
		if err != nil {
			return Case{}, err
		}

		if productID != "" &&
			productID != product.ProductID {

			return Case{}, ErrContextMismatch
		}

		productID = product.ProductID

		snapshot.Product = &product
	} else if productID != "" {
		product, err := s.repository.GetProductContextTx(
			ctx,
			tx,
			productID,
		)
		if err != nil {
			return Case{}, err
		}

		snapshot.Product = &product
	}

	if caseType == "bulk_stock_request" {
		if variantID == "" ||
			request.RequestedQuantity <= 0 ||
			snapshot.Product == nil {

			return Case{}, ErrInvalidRequestedQuantity
		}

		if request.RequestedQuantity <=
			snapshot.Product.AvailableQuantity {

			return Case{}, ErrBulkStockAvailable
		}

		requested := request.RequestedQuantity

		available :=
			snapshot.Product.AvailableQuantity

		requestedQuantity = &requested
		availableQuantity = &available
	}

	if orderID != "" {
		order, err := s.repository.GetOrderContextTx(
			ctx,
			tx,
			customerID,
			orderID,
		)
		if err != nil {
			return Case{}, err
		}

		snapshot.Order = &order
	}

	contextJSON, err := json.Marshal(
		snapshot,
	)
	if err != nil {
		return Case{}, fmt.Errorf(
			"encode CRM context snapshot: %w",
			err,
		)
	}

	caseNumber, err := newCaseNumber()
	if err != nil {
		return Case{}, err
	}

	priority := support.DefaultPriority(
		caseType,
	)

	queueCode := support.DefaultQueueCode(
		caseType,
	)

	caseID, err := s.repository.CreateCaseTx(
		ctx,
		tx,
		createCaseInput{
			CaseNumber: caseNumber,
			CustomerID: customerID,
			CaseType:   caseType,
			Subject:    subject,
			Priority:   priority,

			ProductID: productID,
			VariantID: variantID,
			OrderID:   orderID,

			RequestedQuantity: requestedQuantity,

			AvailableQuantitySnapshot: availableQuantity,

			ContextSnapshot: contextJSON,
		},
	)
	if err != nil {
		return Case{}, err
	}

	attachments, attachmentIDs, err :=
		s.prepareCustomerAttachmentsTx(
			ctx,
			tx,
			customerID,
			caseID,
			request.Attachments,
		)
	if err != nil {
		return Case{}, err
	}

	createdMessage, err :=
		s.repository.InsertCustomerMessageTx(
			ctx,
			tx,
			caseID,
			message,
			attachments,
		)
	if err != nil {
		return Case{}, err
	}

	if err :=
		s.bindCustomerAttachmentsTx(
			ctx,
			tx,
			customerID,
			caseID,
			createdMessage.ID,
			attachmentIDs,
		); err != nil {
		return Case{}, err
	}

	if err := s.repository.InsertCustomerEventTx(
		ctx,
		tx,
		caseID,
		"case_created",
		"",
		StatusWaitingSupport,
		map[string]any{
			"queue_code": queueCode,
		},
	); err != nil {
		return Case{}, err
	}

	if err := support.AssignInitialCaseTx(
		ctx,
		tx,
		caseID,
		queueCode,
	); err != nil {
		return Case{}, err
	}

	if err := s.repository.enqueueStaffCaseCreatedTx(
		ctx,
		tx,
		caseID,
		caseNumber,
		caseType,
		subject,
		priority,
	); err != nil {
		return Case{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Case{}, fmt.Errorf(
			"commit CRM case creation: %w",
			err,
		)
	}

	return s.repository.GetCustomerCase(
		ctx,
		customerID,
		caseID,
	)
}

func (s *Service) ListCases(
	ctx context.Context,
	customerID string,
	limit int,
	offset int,
) (ListResult, error) {
	if limit <= 0 {
		limit = 20
	}

	if limit > 100 {
		limit = 100
	}

	if offset < 0 {
		return ListResult{}, ErrInvalidRequest
	}

	items, err := s.repository.ListCustomerCases(
		ctx,
		customerID,
		limit,
		offset,
	)
	if err != nil {
		return ListResult{}, err
	}

	return ListResult{
		Items:  items,
		Limit:  limit,
		Offset: offset,
	}, nil
}

func (s *Service) GetCase(
	ctx context.Context,
	customerID string,
	caseID string,
) (Case, error) {
	if !validUUID(caseID) {
		return Case{}, ErrCaseNotFound
	}

	return s.repository.GetCustomerCase(
		ctx,
		customerID,
		caseID,
	)
}

func (s *Service) ListMessages(
	ctx context.Context,
	customerID string,
	caseID string,
	limit int,
	offset int,
) (MessageListResult, error) {
	if !validUUID(caseID) {
		return MessageListResult{},
			ErrCaseNotFound
	}

	if limit <= 0 {
		limit = 100
	}

	if limit > 200 {
		limit = 200
	}

	if offset < 0 {
		return MessageListResult{},
			ErrInvalidRequest
	}

	items, err := s.repository.ListCustomerMessages(
		ctx,
		customerID,
		caseID,
		limit,
		offset,
	)
	if err != nil {
		return MessageListResult{}, err
	}

	return MessageListResult{
		Items:  items,
		Limit:  limit,
		Offset: offset,
	}, nil
}

func (s *Service) AddCustomerMessage(
	ctx context.Context,
	customerID string,
	caseID string,
	request AddMessageRequest,
) (Message, error) {
	if !validUUID(caseID) {
		return Message{}, ErrCaseNotFound
	}

	body := strings.TrimSpace(
		request.Message,
	)

	hasAttachments, err :=
		validateCustomerAttachmentEnvelope(
			request.Attachments,
		)
	if err != nil {
		return Message{}, ErrInvalidRequest
	}

	/*
		Allow image/link-only chat messages while
		keeping crm_messages.body populated.
	*/
	if body == "" && hasAttachments {
		body = "Shared an attachment."
	}

	if body == "" ||
		utf8.RuneCountInString(body) > 5000 {

		return Message{}, ErrInvalidRequest
	}

	tx, err := s.repository.Begin(ctx)
	if err != nil {
		return Message{}, err
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	current, err := s.repository.LockCustomerCaseTx(
		ctx,
		tx,
		customerID,
		caseID,
	)
	if err != nil {
		return Message{}, err
	}

	if current.Status == StatusClosed {
		return Message{}, ErrCaseClosed
	}

	attachments, attachmentIDs, err :=
		s.prepareCustomerAttachmentsTx(
			ctx,
			tx,
			customerID,
			caseID,
			request.Attachments,
		)
	if err != nil {
		return Message{}, err
	}

	message, err :=
		s.repository.InsertCustomerMessageTx(
			ctx,
			tx,
			caseID,
			body,
			attachments,
		)
	if err != nil {
		return Message{}, err
	}

	if err :=
		s.bindCustomerAttachmentsTx(
			ctx,
			tx,
			customerID,
			caseID,
			message.ID,
			attachmentIDs,
		); err != nil {
		return Message{}, err
	}

	clearResolved :=
		current.Status == StatusResolved

	if err := s.repository.MarkCustomerMessageTx(
		ctx,
		tx,
		caseID,
		clearResolved,
	); err != nil {
		return Message{}, err
	}

	if err := s.repository.InsertCustomerEventTx(
		ctx,
		tx,
		caseID,
		"customer_message_added",
		current.Status,
		StatusWaitingSupport,
		map[string]any{},
	); err != nil {
		return Message{}, err
	}

	if err := s.repository.enqueueStaffCustomerReplyTx(
		ctx,
		tx,
		caseID,
		message.ID,
	); err != nil {
		return Message{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Message{}, fmt.Errorf(
			"commit CRM customer message: %w",
			err,
		)
	}

	return message, nil
}

func newCaseNumber() (string, error) {
	random := make(
		[]byte,
		6,
	)

	if _, err := rand.Read(
		random,
	); err != nil {
		return "", fmt.Errorf(
			"generate CRM case number: %w",
			err,
		)
	}

	return fmt.Sprintf(
		"CRM-%s-%s",
		time.Now().
			UTC().
			Format("20060102"),
		strings.ToUpper(
			hex.EncodeToString(
				random,
			),
		),
	), nil
}

func validUUID(
	value string,
) bool {
	if len(value) != 36 ||
		value[8] != '-' ||
		value[13] != '-' ||
		value[18] != '-' ||
		value[23] != '-' {

		return false
	}

	compact :=
		strings.ReplaceAll(
			value,
			"-",
			"",
		)

	if len(compact) != 32 {
		return false
	}

	_, err := hex.DecodeString(
		compact,
	)

	return err == nil
}
