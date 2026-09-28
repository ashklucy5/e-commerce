package productrequest

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"
)

type AdminService struct {
	repository *Repository
}

func NewAdminService(
	repository *Repository,
) *AdminService {
	return &AdminService{
		repository: repository,
	}
}

func (s *AdminService) List(
	ctx context.Context,
	filter AdminListFilter,
) (
	AdminListResult,
	error,
) {
	filter.Status =
		strings.ToLower(
			strings.TrimSpace(
				filter.Status,
			),
		)

	filter.Query =
		strings.TrimSpace(
			filter.Query,
		)

	if filter.Status != "" &&
		!validProductRequestStatus(
			filter.Status,
		) {

		return AdminListResult{},
			ErrInvalidRequest
	}

	if utf8.RuneCountInString(
		filter.Query,
	) > 200 {

		return AdminListResult{},
			ErrInvalidRequest
	}

	if filter.Limit <= 0 {
		filter.Limit = 20
	}

	if filter.Limit > 100 {
		filter.Limit = 100
	}

	if filter.Offset < 0 {
		return AdminListResult{},
			ErrInvalidRequest
	}

	items, err :=
		s.repository.
			ListAdminRequests(
				ctx,
				filter,
			)
	if err != nil {
		return AdminListResult{},
			err
	}

	return AdminListResult{
		Items: items,

		Limit: filter.Limit,

		Offset: filter.Offset,
	}, nil
}

func (s *AdminService) Get(
	ctx context.Context,
	requestID string,
) (
	AdminRequest,
	error,
) {
	if !validUUID(
		requestID,
	) {
		return AdminRequest{},
			ErrNotFound
	}

	return s.repository.
		GetAdminRequest(
			ctx,
			requestID,
		)
}

func (s *AdminService) ListMessages(
	ctx context.Context,
	requestID string,
	limit int,
	offset int,
) (
	AdminMessageListResult,
	error,
) {
	if !validUUID(
		requestID,
	) {
		return AdminMessageListResult{},
			ErrNotFound
	}

	if limit <= 0 {
		limit = 100
	}

	if limit > 200 {
		limit = 200
	}

	if offset < 0 {
		return AdminMessageListResult{},
			ErrInvalidRequest
	}

	items, err :=
		s.repository.
			ListAdminMessages(
				ctx,
				requestID,
				limit,
				offset,
			)
	if err != nil {
		return AdminMessageListResult{},
			err
	}

	return AdminMessageListResult{
		Items: items,

		Limit: limit,

		Offset: offset,
	}, nil
}

func (s *AdminService) AddMessage(
	ctx context.Context,
	staffAccountID string,
	requestID string,
	request AdminAddMessageRequest,
) (
	AdminMessage,
	error,
) {
	if !validUUID(
		requestID,
	) ||
		!validUUID(
			staffAccountID,
		) {

		return AdminMessage{},
			ErrInvalidRequest
	}

	body :=
		strings.TrimSpace(
			request.Message,
		)

	if body == "" ||
		utf8.RuneCountInString(
			body,
		) > maxMessageRunes {

		return AdminMessage{},
			ErrInvalidRequest
	}

	visibility :=
		strings.ToLower(
			strings.TrimSpace(
				request.Visibility,
			),
		)

	if visibility == "" {
		visibility =
			"customer"
	}

	if visibility != "customer" &&
		visibility != "internal" {

		return AdminMessage{},
			ErrInvalidRequest
	}

	attachments, err :=
		normalizeJSONShape(
			request.Attachments,
			'[',
			maxAttachmentsBytes,
		)
	if err != nil {
		return AdminMessage{},
			ErrInvalidRequest
	}

	tx, err :=
		s.repository.Begin(
			ctx,
		)
	if err != nil {
		return AdminMessage{},
			err
	}

	defer func() {
		_ = tx.Rollback(
			ctx,
		)
	}()

	current, err :=
		s.repository.
			LockAdminRequestTx(
				ctx,
				tx,
				requestID,
			)
	if err != nil {
		return AdminMessage{},
			err
	}

	if current.CRMStatus ==
		"closed" ||
		current.RequestStatus ==
			"cancelled" ||
		current.RequestStatus ==
			"converted_to_order" {

		return AdminMessage{},
			ErrConversationClosed
	}

	actor, err :=
		s.repository.
			EnsureAdminSupportActorTx(
				ctx,
				tx,
				staffAccountID,
			)
	if err != nil {
		return AdminMessage{},
			err
	}

	message, err :=
		s.repository.
			InsertAdminMessageTx(
				ctx,
				tx,
				current.CaseID,
				actor,
				body,
				visibility,
				attachments,
			)
	if err != nil {
		return AdminMessage{},
			err
	}

	nextCRMStatus, err :=
		s.repository.
			MarkAdminMessageTx(
				ctx,
				tx,
				current.CaseID,
				requestID,
				visibility,
				current.CRMStatus,
			)
	if err != nil {
		return AdminMessage{},
			err
	}

	eventType :=
		"internal_note_added"

	if visibility == "customer" {
		eventType =
			"support_message_added"
	}

	if err :=
		s.repository.
			InsertAdminEventTx(
				ctx,
				tx,
				current.CaseID,
				actor.ID,
				eventType,
				current.CRMStatus,
				nextCRMStatus,
				map[string]any{
					"source": "admin_product_request",

					"product_request_id": requestID,

					"visibility": visibility,
				},
			); err != nil {

		return AdminMessage{},
			err
	}

	if visibility == "customer" {
		if err :=
			s.repository.
				EnqueueAdminReplyNotificationTx(
					ctx,
					tx,
					current.CaseID,
					message.ID,
				); err != nil {

			return AdminMessage{},
				err
		}
	}

	if err :=
		tx.Commit(
			ctx,
		); err != nil {

		return AdminMessage{},
			fmt.Errorf(
				"commit Admin product request message: %w",
				err,
			)
	}

	return message,
		nil
}

func (s *AdminService) UpdateStatus(
	ctx context.Context,
	staffAccountID string,
	requestID string,
	request AdminUpdateStatusRequest,
) (
	AdminRequest,
	error,
) {
	if !validUUID(
		requestID,
	) ||
		!validUUID(
			staffAccountID,
		) {

		return AdminRequest{},
			ErrInvalidRequest
	}

	targetStatus :=
		strings.ToLower(
			strings.TrimSpace(
				request.Status,
			),
		)

	reason :=
		strings.TrimSpace(
			request.Reason,
		)

	if targetStatus != "on_hold" &&
		targetStatus != "accepted" &&
		targetStatus != "cancelled" {

		return AdminRequest{},
			ErrInvalidRequest
	}

	if (targetStatus == "on_hold" ||
		targetStatus == "cancelled") &&
		reason == "" {

		return AdminRequest{},
			ErrInvalidRequest
	}

	if utf8.RuneCountInString(
		reason,
	) > 2000 {

		return AdminRequest{},
			ErrInvalidRequest
	}

	tx, err :=
		s.repository.Begin(
			ctx,
		)
	if err != nil {
		return AdminRequest{},
			err
	}

	defer func() {
		_ = tx.Rollback(
			ctx,
		)
	}()

	current, err :=
		s.repository.
			LockAdminRequestTx(
				ctx,
				tx,
				requestID,
			)
	if err != nil {
		return AdminRequest{},
			err
	}

	if current.CRMStatus ==
		"closed" {

		return AdminRequest{},
			ErrConversationClosed
	}

	if !allowedReviewTransition(
		current.RequestStatus,
		targetStatus,
	) {
		return AdminRequest{},
			ErrInvalidStatusTransition
	}

	actor, err :=
		s.repository.
			EnsureAdminSupportActorTx(
				ctx,
				tx,
				staffAccountID,
			)
	if err != nil {
		return AdminRequest{},
			err
	}

	if err :=
		s.repository.
			UpdateAdminRequestStatusTx(
				ctx,
				tx,
				requestID,
				targetStatus,
				reason,
				actor.ID,
			); err != nil {

		return AdminRequest{},
			err
	}

	nextCRMStatus :=
		current.CRMStatus

	if targetStatus == "cancelled" {
		nextCRMStatus =
			"closed"
	} else if current.CRMStatus ==
		"resolved" {

		nextCRMStatus =
			"waiting_support"
	}

	if nextCRMStatus !=
		current.CRMStatus {

		if err :=
			s.repository.
				UpdateAdminCRMStatusTx(
					ctx,
					tx,
					current.CaseID,
					nextCRMStatus,
				); err != nil {

			return AdminRequest{},
				err
		}
	}

	if err :=
		s.repository.
			InsertAdminEventTx(
				ctx,
				tx,
				current.CaseID,
				actor.ID,
				"product_request_status_changed",
				current.CRMStatus,
				nextCRMStatus,
				map[string]any{
					"source": "admin_product_request",

					"product_request_id": requestID,

					"from_sourcing_status": current.RequestStatus,

					"to_sourcing_status": targetStatus,

					"status_reason": reason,
				},
			); err != nil {

		return AdminRequest{},
			err
	}

	if err :=
		tx.Commit(
			ctx,
		); err != nil {

		return AdminRequest{},
			fmt.Errorf(
				"commit Admin product request status: %w",
				err,
			)
	}

	return s.repository.
		GetAdminRequest(
			ctx,
			requestID,
		)
}

func validProductRequestStatus(
	status string,
) bool {
	switch status {
	case "pending_review",
		"on_hold",
		"accepted",
		"negotiating",
		"agreed",
		"converted_to_order",
		"cancelled":

		return true

	default:
		return false
	}
}

func allowedReviewTransition(
	from string,
	to string,
) bool {
	switch from {
	case "pending_review":
		return to ==
			"on_hold" ||
			to ==
				"accepted" ||
			to ==
				"cancelled"

	case "on_hold":
		return to ==
			"on_hold" ||
			to ==
				"accepted" ||
			to ==
				"cancelled"

	default:
		return false
	}
}
