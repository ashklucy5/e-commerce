package productrequest

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"project.local/commerce-api/internal/order"
	"project.local/commerce-api/internal/support"
)

const (
	productRequestQueueCode = "bulk_sales"

	maxDescriptionRunes  = 5000
	maxMessageRunes      = 5000
	maxExternalURLRunes  = 2048
	maxRequestedQuantity = 2147483647

	maxRequirementsBytes = 64 * 1024
	maxAttachmentsBytes  = 128 * 1024
)

type Service struct {
	repository   *Repository
	orderService *order.Service
}

func NewService(
	repository *Repository,
	orderServices ...*order.Service,
) *Service {
	result :=
		&Service{
			repository: repository,
		}

	if len(orderServices) > 0 {
		result.orderService =
			orderServices[0]
	}

	return result
}

func (s *Service) Create(
	ctx context.Context,
	customerID string,
	request CreateRequest,
) (Request, error) {
	productName :=
		strings.Join(
			strings.Fields(
				request.RequestedProductName,
			),
			" ",
		)

	description :=
		strings.TrimSpace(
			request.Description,
		)

	if productName == "" ||
		utf8.RuneCountInString(
			productName,
		) > 180 ||
		description == "" ||
		utf8.RuneCountInString(
			description,
		) > maxDescriptionRunes ||
		request.RequestedQuantity <= 0 ||
		request.RequestedQuantity >
			maxRequestedQuantity {

		return Request{},
			ErrInvalidRequest
	}

	requirements, err :=
		normalizeJSONShape(
			request.CustomerRequirements,
			'{',
			maxRequirementsBytes,
		)
	if err != nil {
		return Request{},
			ErrInvalidRequest
	}

	attachments, err :=
		normalizeJSONShape(
			request.Attachments,
			'[',
			maxAttachmentsBytes,
		)
	if err != nil {
		return Request{},
			ErrInvalidRequest
	}

	externalURL, err :=
		normalizeExternalURL(
			request.ExternalURL,
		)
	if err != nil {
		return Request{},
			ErrInvalidRequest
	}

	caseNumber, err :=
		newNumber(
			"CRM",
		)
	if err != nil {
		return Request{},
			err
	}

	requestNumber, err :=
		newNumber(
			"SRC",
		)
	if err != nil {
		return Request{},
			err
	}

	contextSnapshot, err :=
		json.Marshal(
			map[string]any{
				"product_request": map[string]any{
					"request_number": requestNumber,

					"requested_product_name": productName,

					"requested_quantity": request.RequestedQuantity,

					"external_url": externalURL,
				},
			},
		)
	if err != nil {
		return Request{},
			fmt.Errorf(
				"encode product request CRM context snapshot: %w",
				err,
			)
	}

	tx, err :=
		s.repository.Begin(
			ctx,
		)
	if err != nil {
		return Request{},
			err
	}

	defer func() {
		_ = tx.Rollback(
			ctx,
		)
	}()

	caseID, err :=
		s.repository.CreateCaseTx(
			ctx,
			tx,
			createCaseInput{
				CaseNumber: caseNumber,

				CustomerID: customerID,

				Subject: productName,

				RequestedQuantity: request.RequestedQuantity,

				ContextSnapshot: contextSnapshot,
			},
		)
	if err != nil {
		return Request{},
			err
	}

	requestID, err :=
		s.repository.
			CreateSourcingRequestTx(
				ctx,
				tx,
				createSourcingRequestInput{
					CaseID: caseID,

					RequestNumber: requestNumber,

					RequestedProductName: productName,

					Description: description,

					CustomerRequirements: requirements,

					ExternalURL: externalURL,

					Attachments: attachments,
				},
			)
	if err != nil {
		return Request{},
			err
	}

	if _, err :=
		s.repository.
			InsertCustomerMessageTx(
				ctx,
				tx,
				caseID,
				description,
				attachments,
			); err != nil {

		return Request{},
			err
	}

	if err :=
		s.repository.
			InsertCustomerEventTx(
				ctx,
				tx,
				caseID,
				"case_created",
				"",
				"waiting_support",
				map[string]any{
					"queue_code": productRequestQueueCode,

					"source": "product_request",

					"product_request_id": requestID,

					"request_number": requestNumber,
				},
			); err != nil {

		return Request{},
			err
	}

	if err :=
		support.AssignInitialCaseTx(
			ctx,
			tx,
			caseID,
			productRequestQueueCode,
		); err != nil {

		return Request{},
			err
	}

	if err :=
		s.repository.
			enqueueStaffProductRequestCreatedTx(
				ctx,
				tx,
				requestID,
				caseID,
				requestNumber,
				productName,
				request.RequestedQuantity,
			); err != nil {

		return Request{},
			err
	}

	if err :=
		tx.Commit(
			ctx,
		); err != nil {

		return Request{},
			fmt.Errorf(
				"commit product request creation: %w",
				err,
			)
	}

	return s.repository.
		GetCustomerRequest(
			ctx,
			customerID,
			requestID,
		)
}

func (s *Service) List(
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
		return ListResult{},
			ErrInvalidRequest
	}

	items, err :=
		s.repository.
			ListCustomerRequests(
				ctx,
				customerID,
				limit,
				offset,
			)
	if err != nil {
		return ListResult{},
			err
	}

	return ListResult{
		Items: items,

		Limit: limit,

		Offset: offset,
	}, nil
}

func (s *Service) Get(
	ctx context.Context,
	customerID string,
	requestID string,
) (Request, error) {
	if !validUUID(
		requestID,
	) {
		return Request{},
			ErrNotFound
	}

	return s.repository.
		GetCustomerRequest(
			ctx,
			customerID,
			requestID,
		)
}

func (s *Service) ListMessages(
	ctx context.Context,
	customerID string,
	requestID string,
	limit int,
	offset int,
) (
	MessageListResult,
	error,
) {
	if !validUUID(
		requestID,
	) {
		return MessageListResult{},
			ErrNotFound
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

	items, err :=
		s.repository.
			ListCustomerMessages(
				ctx,
				customerID,
				requestID,
				limit,
				offset,
			)
	if err != nil {
		return MessageListResult{},
			err
	}

	return MessageListResult{
		Items: items,

		Limit: limit,

		Offset: offset,
	}, nil
}

func (s *Service) AddCustomerMessage(
	ctx context.Context,
	customerID string,
	requestID string,
	request AddMessageRequest,
) (Message, error) {
	if !validUUID(
		requestID,
	) {
		return Message{},
			ErrNotFound
	}

	body :=
		strings.TrimSpace(
			request.Message,
		)

	if body == "" ||
		utf8.RuneCountInString(
			body,
		) > maxMessageRunes {

		return Message{},
			ErrInvalidRequest
	}

	attachments, err :=
		normalizeJSONShape(
			request.Attachments,
			'[',
			maxAttachmentsBytes,
		)
	if err != nil {
		return Message{},
			ErrInvalidRequest
	}

	tx, err :=
		s.repository.Begin(
			ctx,
		)
	if err != nil {
		return Message{},
			err
	}

	defer func() {
		_ = tx.Rollback(
			ctx,
		)
	}()

	current, err :=
		s.repository.
			LockCustomerRequestTx(
				ctx,
				tx,
				customerID,
				requestID,
			)
	if err != nil {
		return Message{},
			err
	}

	if current.ConversationStatus ==
		"closed" ||
		current.RequestStatus ==
			"cancelled" ||
		current.RequestStatus ==
			"converted_to_order" {

		return Message{},
			ErrConversationClosed
	}

	message, err :=
		s.repository.
			InsertCustomerMessageTx(
				ctx,
				tx,
				current.CaseID,
				body,
				attachments,
			)
	if err != nil {
		return Message{},
			err
	}

	clearResolved :=
		current.ConversationStatus ==
			"resolved"

	if err :=
		s.repository.
			MarkCustomerMessageTx(
				ctx,
				tx,
				current.CaseID,
				requestID,
				clearResolved,
			); err != nil {

		return Message{},
			err
	}

	if err :=
		s.repository.
			InsertCustomerEventTx(
				ctx,
				tx,
				current.CaseID,
				"customer_message_added",
				current.ConversationStatus,
				"waiting_support",
				map[string]any{
					"source": "product_request",

					"product_request_id": requestID,
				},
			); err != nil {

		return Message{},
			err
	}

	if err :=
		s.repository.
			enqueueStaffProductRequestCustomerMessageTx(
				ctx,
				tx,
				requestID,
				current.CaseID,
				message.ID,
			); err != nil {

		return Message{},
			err
	}

	if err :=
		tx.Commit(
			ctx,
		); err != nil {

		return Message{},
			fmt.Errorf(
				"commit product request customer message: %w",
				err,
			)
	}

	return message, nil
}

func normalizeJSONShape(
	raw json.RawMessage,
	openingByte byte,
	maxBytes int,
) (
	json.RawMessage,
	error,
) {
	if len(raw) == 0 {
		return nil,
			nil
	}

	if len(raw) >
		maxBytes ||
		!json.Valid(
			raw,
		) {

		return nil,
			ErrInvalidRequest
	}

	trimmed :=
		strings.TrimSpace(
			string(
				raw,
			),
		)

	if trimmed ==
		"null" {

		return nil,
			nil
	}

	if trimmed == "" ||
		trimmed[0] !=
			openingByte {

		return nil,
			ErrInvalidRequest
	}

	var value any

	if err :=
		json.Unmarshal(
			[]byte(
				trimmed,
			),
			&value,
		); err != nil {

		return nil,
			ErrInvalidRequest
	}

	switch openingByte {
	case '{':
		if _, ok :=
			value.(map[string]any); !ok {

			return nil,
				ErrInvalidRequest
		}

	case '[':
		if _, ok :=
			value.([]any); !ok {

			return nil,
				ErrInvalidRequest
		}

	default:
		return nil,
			ErrInvalidRequest
	}

	return json.RawMessage(
			trimmed,
		),
		nil
}

func normalizeExternalURL(
	value string,
) (
	string,
	error,
) {
	value =
		strings.TrimSpace(
			value,
		)

	if value == "" {
		return "",
			nil
	}

	if utf8.RuneCountInString(
		value,
	) > maxExternalURLRunes {

		return "",
			ErrInvalidRequest
	}

	parsed, err :=
		url.ParseRequestURI(
			value,
		)

	if err != nil ||
		(parsed.Scheme != "http" &&
			parsed.Scheme != "https") ||
		parsed.Host == "" {

		return "",
			ErrInvalidRequest
	}

	return parsed.String(),
		nil
}

func newNumber(
	prefix string,
) (
	string,
	error,
) {
	random :=
		make(
			[]byte,
			6,
		)

	if _, err :=
		rand.Read(
			random,
		); err != nil {

		return "",
			fmt.Errorf(
				"generate product request number: %w",
				err,
			)
	}

	return fmt.Sprintf(
			"%s-%s-%s",
			prefix,
			time.Now().
				UTC().
				Format(
					"20060102",
				),
			strings.ToUpper(
				hex.EncodeToString(
					random,
				),
			),
		),
		nil
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

	_, err :=
		hex.DecodeString(
			compact,
		)

	return err == nil
}