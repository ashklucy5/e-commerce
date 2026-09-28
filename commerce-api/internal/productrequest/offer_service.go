package productrequest

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	maxOfferDescriptionRunes    = 8000
	maxOfferAttachmentsBytes    = 64 * 1024
	maxOfferSpecificationsBytes = 128 * 1024
)

func (s *AdminService) ListOffers(
	ctx context.Context,
	requestID string,
) (
	[]SourcingOffer,
	error,
) {
	if !validUUID(
		requestID,
	) {
		return nil,
			ErrNotFound
	}

	items, err :=
		s.repository.
			ListAdminOffers(
				ctx,
				requestID,
			)
	if err != nil {
		return nil,
			err
	}

	for i := range items {

		items[i] =
			effectiveOffer(
				items[i],
			)
	}

	return items,
		nil
}

func (s *AdminService) CreateOffer(
	ctx context.Context,
	staffAccountID string,
	requestID string,
	request AdminCreateOfferRequest,
) (
	SourcingOffer,
	error,
) {
	if !validUUID(
		staffAccountID,
	) ||
		!validUUID(
			requestID,
		) {

		return SourcingOffer{},
			ErrInvalidRequest
	}

	input, err :=
		normalizeOfferCreate(
			request,
		)
	if err != nil {
		return SourcingOffer{},
			err
	}

	tx, err :=
		s.repository.Begin(
			ctx,
		)
	if err != nil {
		return SourcingOffer{},
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
		return SourcingOffer{},
			err
	}

	if current.CRMStatus ==
		"closed" {

		return SourcingOffer{},
			ErrConversationClosed
	}

	if current.RequestStatus !=
		"accepted" &&
		current.RequestStatus !=
			"negotiating" {

		return SourcingOffer{},
			ErrOfferConflict
	}

	hasAccepted, err :=
		s.repository.
			HasAcceptedOrFinalizedOfferTx(
				ctx,
				tx,
				requestID,
			)
	if err != nil {
		return SourcingOffer{},
			err
	}

	if hasAccepted {
		return SourcingOffer{},
			ErrOfferConflict
	}

	hasDraft, err :=
		s.repository.
			HasDraftOfferTx(
				ctx,
				tx,
				requestID,
			)
	if err != nil {
		return SourcingOffer{},
			err
	}

	if hasDraft {
		return SourcingOffer{},
			ErrOfferConflict
	}

	actor, err :=
		s.repository.
			EnsureAdminSupportActorTx(
				ctx,
				tx,
				staffAccountID,
			)
	if err != nil {
		return SourcingOffer{},
			err
	}

	offer, err :=
		s.repository.
			CreateOfferTx(
				ctx,
				tx,
				requestID,
				actor.ID,
				input,
			)
	if err != nil {
		return SourcingOffer{},
			err
	}

	if err :=
		s.repository.
			InsertAdminEventTx(
				ctx,
				tx,
				current.CaseID,
				actor.ID,
				"product_sourcing_offer_created",
				current.CRMStatus,
				current.CRMStatus,
				map[string]any{
					"source": "admin_product_request",

					"product_request_id": requestID,

					"offer_id": offer.ID,

					"offer_status": "draft",
				},
			); err != nil {

		return SourcingOffer{},
			err
	}

	if err :=
		tx.Commit(
			ctx,
		); err != nil {

		return SourcingOffer{},
			fmt.Errorf(
				"commit sourcing offer creation: %w",
				err,
			)
	}

	return offer,
		nil
}

func (s *AdminService) UpdateOffer(
	ctx context.Context,
	staffAccountID string,
	requestID string,
	offerID string,
	request AdminUpdateOfferRequest,
) (
	SourcingOffer,
	error,
) {
	if !validUUID(
		staffAccountID,
	) ||
		!validUUID(
			requestID,
		) ||
		!validUUID(
			offerID,
		) {

		return SourcingOffer{},
			ErrInvalidRequest
	}

	if !hasOfferPatch(
		request,
	) {
		return SourcingOffer{},
			ErrInvalidRequest
	}

	tx, err :=
		s.repository.Begin(
			ctx,
		)
	if err != nil {
		return SourcingOffer{},
			err
	}

	defer func() {
		_ = tx.Rollback(
			ctx,
		)
	}()

	currentRequest, err :=
		s.repository.
			LockAdminRequestTx(
				ctx,
				tx,
				requestID,
			)
	if err != nil {
		return SourcingOffer{},
			err
	}

	if currentRequest.CRMStatus ==
		"closed" {

		return SourcingOffer{},
			ErrConversationClosed
	}

	if currentRequest.RequestStatus !=
		"accepted" &&
		currentRequest.RequestStatus !=
			"negotiating" {

		return SourcingOffer{},
			ErrOfferConflict
	}

	currentOffer, err :=
		s.repository.
			LockOfferTx(
				ctx,
				tx,
				requestID,
				offerID,
			)
	if err != nil {
		return SourcingOffer{},
			err
	}

	if currentOffer.Status !=
		"draft" {

		return SourcingOffer{},
			ErrOfferNotActionable
	}

	input, err :=
		normalizeOfferPatch(
			currentOffer,
			request,
		)
	if err != nil {
		return SourcingOffer{},
			err
	}

	actor, err :=
		s.repository.
			EnsureAdminSupportActorTx(
				ctx,
				tx,
				staffAccountID,
			)
	if err != nil {
		return SourcingOffer{},
			err
	}

	offer, err :=
		s.repository.
			UpdateDraftOfferTx(
				ctx,
				tx,
				requestID,
				offerID,
				input,
			)
	if err != nil {
		return SourcingOffer{},
			err
	}

	if err :=
		s.repository.
			InsertAdminEventTx(
				ctx,
				tx,
				currentRequest.CaseID,
				actor.ID,
				"product_sourcing_offer_updated",
				currentRequest.CRMStatus,
				currentRequest.CRMStatus,
				map[string]any{
					"source": "admin_product_request",

					"product_request_id": requestID,

					"offer_id": offerID,
				},
			); err != nil {

		return SourcingOffer{},
			err
	}

	if err :=
		tx.Commit(
			ctx,
		); err != nil {

		return SourcingOffer{},
			fmt.Errorf(
				"commit sourcing offer update: %w",
				err,
			)
	}

	return offer,
		nil
}

func (s *AdminService) SendOffer(
	ctx context.Context,
	staffAccountID string,
	requestID string,
	offerID string,
) (
	SourcingOffer,
	error,
) {
	if !validUUID(
		staffAccountID,
	) ||
		!validUUID(
			requestID,
		) ||
		!validUUID(
			offerID,
		) {

		return SourcingOffer{},
			ErrInvalidRequest
	}

	tx, err :=
		s.repository.Begin(
			ctx,
		)
	if err != nil {
		return SourcingOffer{},
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
		return SourcingOffer{},
			err
	}

	if current.CRMStatus ==
		"closed" {

		return SourcingOffer{},
			ErrConversationClosed
	}

	if current.RequestStatus !=
		"accepted" &&
		current.RequestStatus !=
			"negotiating" {

		return SourcingOffer{},
			ErrOfferConflict
	}

	if err :=
		s.repository.
			ExpireRequestOffersTx(
				ctx,
				tx,
				requestID,
			); err != nil {

		return SourcingOffer{},
			err
	}

	hasAccepted, err :=
		s.repository.
			HasAcceptedOrFinalizedOfferTx(
				ctx,
				tx,
				requestID,
			)
	if err != nil {
		return SourcingOffer{},
			err
	}

	if hasAccepted {
		return SourcingOffer{},
			ErrOfferConflict
	}

	offer, err :=
		s.repository.
			LockOfferTx(
				ctx,
				tx,
				requestID,
				offerID,
			)
	if err != nil {
		return SourcingOffer{},
			err
	}

	if offer.Status !=
		"draft" {

		return SourcingOffer{},
			ErrOfferNotActionable
	}

	if err :=
		validateOfferForSend(
			offer,
		); err != nil {

		return SourcingOffer{},
			err
	}

	actor, err :=
		s.repository.
			EnsureAdminSupportActorTx(
				ctx,
				tx,
				staffAccountID,
			)
	if err != nil {
		return SourcingOffer{},
			err
	}

	if err :=
		s.repository.
			SupersedeSentOffersTx(
				ctx,
				tx,
				requestID,
				offerID,
			); err != nil {

		return SourcingOffer{},
			err
	}

	sent, err :=
		s.repository.
			SendOfferTx(
				ctx,
				tx,
				requestID,
				offerID,
			)
	if err != nil {
		return SourcingOffer{},
			err
	}

	nextCRMStatus :=
		"waiting_customer"

	if current.CRMStatus !=
		nextCRMStatus {

		if err :=
			s.repository.
				UpdateAdminCRMStatusTx(
					ctx,
					tx,
					current.CaseID,
					nextCRMStatus,
				); err != nil {

			return SourcingOffer{},
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
				"product_sourcing_offer_sent",
				current.CRMStatus,
				nextCRMStatus,
				map[string]any{
					"source": "admin_product_request",

					"product_request_id": requestID,

					"offer_id": offerID,

					"quoted_quantity": sent.QuotedQuantity,

					"minimum_order_quantity": sent.MinimumOrderQuantity,

					"currency": sent.Currency,
				},
			); err != nil {

		return SourcingOffer{},
			err
	}

	if err :=
		tx.Commit(
			ctx,
		); err != nil {

		return SourcingOffer{},
			fmt.Errorf(
				"commit sourcing offer send: %w",
				err,
			)
	}

	return sent,
		nil
}

func (s *AdminService) FinalizeOffer(
	ctx context.Context,
	staffAccountID string,
	requestID string,
	offerID string,
) (
	SourcingConfirmation,
	error,
) {
	if !validUUID(
		staffAccountID,
	) ||
		!validUUID(
			requestID,
		) ||
		!validUUID(
			offerID,
		) {

		return SourcingConfirmation{},
			ErrInvalidRequest
	}

	tx, err :=
		s.repository.Begin(
			ctx,
		)
	if err != nil {
		return SourcingConfirmation{},
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
		return SourcingConfirmation{},
			err
	}

	if current.CRMStatus ==
		"closed" {

		return SourcingConfirmation{},
			ErrConversationClosed
	}

	if current.RequestStatus !=
		"negotiating" {

		return SourcingConfirmation{},
			ErrOfferConflict
	}

	exists, err :=
		s.repository.
			ConfirmationExistsTx(
				ctx,
				tx,
				requestID,
			)
	if err != nil {
		return SourcingConfirmation{},
			err
	}

	if exists {
		return SourcingConfirmation{},
			ErrOfferAlreadyFinalized
	}

	offer, err :=
		s.repository.
			LockOfferTx(
				ctx,
				tx,
				requestID,
				offerID,
			)
	if err != nil {
		return SourcingConfirmation{},
			err
	}

	if offer.Status !=
		"customer_accepted" ||
		offer.QuotedQuantity ==
			nil ||
		offer.MinimumOrderQuantity ==
			nil {

		return SourcingConfirmation{},
			ErrOfferNotActionable
	}

	totalAmount, ok :=
		safeOfferTotal(
			offer.UnitPrice,
			*offer.QuotedQuantity,
			offer.ShippingPrice,
		)

	if !ok {
		return SourcingConfirmation{},
			ErrInvalidRequest
	}

	actor, err :=
		s.repository.
			EnsureAdminSupportActorTx(
				ctx,
				tx,
				staffAccountID,
			)
	if err != nil {
		return SourcingConfirmation{},
			err
	}

	confirmation, err :=
		s.repository.
			FinalizeOfferTx(
				ctx,
				tx,
				requestID,
				offerID,
				actor.ID,
				*offer.QuotedQuantity,
				*offer.MinimumOrderQuantity,
				offer.ProductName,
				offer.OfferedSpecifications,
				offer.UnitPrice,
				offer.ShippingPrice,
				offer.Currency,
				totalAmount,
			)
	if err != nil {
		return SourcingConfirmation{},
			err
	}

	if err :=
		s.repository.
			InsertAdminEventTx(
				ctx,
				tx,
				current.CaseID,
				actor.ID,
				"product_sourcing_offer_finalized",
				current.CRMStatus,
				current.CRMStatus,
				map[string]any{
					"source": "admin_product_request",

					"product_request_id": requestID,

					"offer_id": offerID,

					"confirmation_id": confirmation.ID,

					"total_amount": totalAmount,

					"currency": offer.Currency,
				},
			); err != nil {

		return SourcingConfirmation{},
			err
	}

	if err :=
		tx.Commit(
			ctx,
		); err != nil {

		return SourcingConfirmation{},
			fmt.Errorf(
				"commit sourcing offer finalization: %w",
				err,
			)
	}

	return confirmation,
		nil
}

func (s *AdminService) GetConfirmation(
	ctx context.Context,
	requestID string,
) (
	SourcingConfirmation,
	error,
) {
	if !validUUID(
		requestID,
	) {
		return SourcingConfirmation{},
			ErrNotFound
	}

	return s.repository.
		GetAdminConfirmation(
			ctx,
			requestID,
		)
}

func (s *Service) ListCustomerOffers(
	ctx context.Context,
	customerID string,
	requestID string,
) (
	[]SourcingOffer,
	error,
) {
	if !validUUID(
		customerID,
	) ||
		!validUUID(
			requestID,
		) {

		return nil,
			ErrNotFound
	}

	items, err :=
		s.repository.
			ListCustomerOffers(
				ctx,
				customerID,
				requestID,
			)
	if err != nil {
		return nil,
			err
	}

	for i := range items {

		items[i] =
			effectiveOffer(
				items[i],
			)
	}

	return items,
		nil
}

func (s *Service) GetCustomerOffer(
	ctx context.Context,
	customerID string,
	requestID string,
	offerID string,
) (
	SourcingOffer,
	error,
) {
	if !validUUID(
		customerID,
	) ||
		!validUUID(
			requestID,
		) ||
		!validUUID(
			offerID,
		) {

		return SourcingOffer{},
			ErrOfferNotFound
	}

	offer, err :=
		s.repository.
			GetCustomerOffer(
				ctx,
				customerID,
				requestID,
				offerID,
			)
	if err != nil {
		return SourcingOffer{},
			err
	}

	return effectiveOffer(
			offer,
		),
		nil
}

func (s *Service) AcceptCustomerOffer(
	ctx context.Context,
	customerID string,
	requestID string,
	offerID string,
) (
	SourcingOffer,
	error,
) {
	return s.respondToCustomerOffer(
		ctx,
		customerID,
		requestID,
		offerID,
		"customer_accepted",
	)
}

func (s *Service) RejectCustomerOffer(
	ctx context.Context,
	customerID string,
	requestID string,
	offerID string,
) (
	SourcingOffer,
	error,
) {
	return s.respondToCustomerOffer(
		ctx,
		customerID,
		requestID,
		offerID,
		"customer_rejected",
	)
}

func (s *Service) respondToCustomerOffer(
	ctx context.Context,
	customerID string,
	requestID string,
	offerID string,
	targetStatus string,
) (
	SourcingOffer,
	error,
) {
	if !validUUID(
		customerID,
	) ||
		!validUUID(
			requestID,
		) ||
		!validUUID(
			offerID,
		) {

		return SourcingOffer{},
			ErrOfferNotFound
	}

	tx, err :=
		s.repository.Begin(
			ctx,
		)
	if err != nil {
		return SourcingOffer{},
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
		return SourcingOffer{},
			err
	}

	if current.ConversationStatus ==
		"closed" ||
		current.RequestStatus ==
			"cancelled" ||
		current.RequestStatus ==
			"converted_to_order" {

		return SourcingOffer{},
			ErrConversationClosed
	}

	if current.RequestStatus !=
		"negotiating" {

		return SourcingOffer{},
			ErrOfferConflict
	}

	offer, err :=
		s.repository.
			LockOfferTx(
				ctx,
				tx,
				requestID,
				offerID,
			)
	if err != nil {
		return SourcingOffer{},
			err
	}

	if offer.Status ==
		"sent" &&
		offer.ExpiresAt != nil &&
		!offer.ExpiresAt.After(
			time.Now(),
		) {

		if err :=
			s.repository.
				MarkOfferExpiredTx(
					ctx,
					tx,
					requestID,
					offerID,
				); err != nil {

			return SourcingOffer{},
				err
		}

		if err :=
			tx.Commit(
				ctx,
			); err != nil {

			return SourcingOffer{},
				fmt.Errorf(
					"commit expired sourcing offer: %w",
					err,
				)
		}

		return SourcingOffer{},
			ErrOfferExpired
	}

	if offer.Status ==
		"expired" {

		return SourcingOffer{},
			ErrOfferExpired
	}

	if offer.Status !=
		"sent" {

		return SourcingOffer{},
			ErrOfferNotActionable
	}

	updated, err :=
		s.repository.
			SetCustomerOfferResponseTx(
				ctx,
				tx,
				requestID,
				offerID,
				targetStatus,
			)
	if err != nil {
		return SourcingOffer{},
			err
	}

	if targetStatus ==
		"customer_accepted" {

		if err :=
			s.repository.
				SupersedeOtherOpenOffersTx(
					ctx,
					tx,
					requestID,
					offerID,
				); err != nil {

			return SourcingOffer{},
				err
		}
	}

	if err :=
		s.repository.
			MarkCustomerOfferResponseCRMTx(
				ctx,
				tx,
				current.CaseID,
			); err != nil {

		return SourcingOffer{},
			err
	}

	eventType :=
		"product_sourcing_offer_rejected"

	if targetStatus ==
		"customer_accepted" {

		eventType =
			"product_sourcing_offer_accepted"
	}

	if err :=
		s.repository.
			InsertCustomerEventTx(
				ctx,
				tx,
				current.CaseID,
				eventType,
				current.ConversationStatus,
				"waiting_support",
				map[string]any{
					"source": "product_request_offer",

					"product_request_id": requestID,

					"offer_id": offerID,
				},
			); err != nil {

		return SourcingOffer{},
			err
	}

	if err :=
		tx.Commit(
			ctx,
		); err != nil {

		return SourcingOffer{},
			fmt.Errorf(
				"commit customer sourcing offer response: %w",
				err,
			)
	}

	updated.CreatedBy =
		nil

	return updated,
		nil
}

func (s *Service) GetCustomerConfirmation(
	ctx context.Context,
	customerID string,
	requestID string,
) (
	SourcingConfirmation,
	error,
) {
	if !validUUID(
		customerID,
	) ||
		!validUUID(
			requestID,
		) {

		return SourcingConfirmation{},
			ErrNotFound
	}

	return s.repository.
		GetCustomerConfirmation(
			ctx,
			customerID,
			requestID,
		)
}

func normalizeOfferCreate(
	request AdminCreateOfferRequest,
) (
	offerMutation,
	error,
) {
	input :=
		offerMutation{
			ProductName: strings.TrimSpace(
				request.ProductName,
			),

			Description: strings.TrimSpace(
				request.Description,
			),

			UnitPrice: request.UnitPrice,

			ShippingPrice: request.ShippingPrice,

			Currency: strings.ToUpper(
				strings.TrimSpace(
					request.Currency,
				),
			),

			QuotedQuantity: cloneOfferInt(
				request.QuotedQuantity,
			),

			MinimumOrderQuantity: cloneOfferInt(
				request.MinimumOrderQuantity,
			),

			ExpiresAt: cloneOfferTime(
				request.ExpiresAt,
			),
		}

	attachments, err :=
		normalizeJSONShape(
			request.Attachments,
			'[',
			maxOfferAttachmentsBytes,
		)
	if err != nil {
		return offerMutation{},
			ErrInvalidRequest
	}

	specifications, err :=
		normalizeJSONShape(
			request.OfferedSpecifications,
			'{',
			maxOfferSpecificationsBytes,
		)
	if err != nil {
		return offerMutation{},
			ErrInvalidRequest
	}

	input.Attachments =
		attachments

	input.OfferedSpecifications =
		specifications

	if err :=
		validateOfferMutation(
			input,
		); err != nil {

		return offerMutation{},
			err
	}

	return input,
		nil
}

func normalizeOfferPatch(
	current SourcingOffer,
	request AdminUpdateOfferRequest,
) (
	offerMutation,
	error,
) {
	input :=
		offerMutation{
			ProductName: current.ProductName,

			Description: current.Description,

			Attachments: cloneRawMessage(
				current.Attachments,
			),

			OfferedSpecifications: cloneRawMessage(
				current.OfferedSpecifications,
			),

			UnitPrice: current.UnitPrice,

			ShippingPrice: current.ShippingPrice,

			Currency: current.Currency,

			QuotedQuantity: cloneOfferInt(
				current.QuotedQuantity,
			),

			MinimumOrderQuantity: cloneOfferInt(
				current.MinimumOrderQuantity,
			),

			ExpiresAt: cloneOfferTime(
				current.ExpiresAt,
			),
		}

	if request.ProductName != nil {
		input.ProductName =
			strings.TrimSpace(
				*request.ProductName,
			)
	}

	if request.Description != nil {
		input.Description =
			strings.TrimSpace(
				*request.Description,
			)
	}

	if request.Attachments != nil {
		value, err :=
			normalizeJSONShape(
				*request.Attachments,
				'[',
				maxOfferAttachmentsBytes,
			)
		if err != nil {
			return offerMutation{},
				ErrInvalidRequest
		}

		input.Attachments =
			value
	}

	if request.OfferedSpecifications !=
		nil {

		value, err :=
			normalizeJSONShape(
				*request.OfferedSpecifications,
				'{',
				maxOfferSpecificationsBytes,
			)
		if err != nil {
			return offerMutation{},
				ErrInvalidRequest
		}

		input.OfferedSpecifications =
			value
	}

	if request.UnitPrice != nil {
		input.UnitPrice =
			*request.UnitPrice
	}

	if request.ShippingPrice != nil {
		input.ShippingPrice =
			*request.ShippingPrice
	}

	if request.Currency != nil {
		input.Currency =
			strings.ToUpper(
				strings.TrimSpace(
					*request.Currency,
				),
			)
	}

	if request.QuotedQuantity != nil {
		input.QuotedQuantity =
			cloneOfferInt(
				request.QuotedQuantity,
			)
	}

	if request.MinimumOrderQuantity !=
		nil {

		input.MinimumOrderQuantity =
			cloneOfferInt(
				request.MinimumOrderQuantity,
			)
	}

	if request.ExpiresAt != nil {
		input.ExpiresAt =
			cloneOfferTime(
				request.ExpiresAt,
			)
	}

	if err :=
		validateOfferMutation(
			input,
		); err != nil {

		return offerMutation{},
			err
	}

	return input,
		nil
}

func validateOfferMutation(
	input offerMutation,
) error {
	if input.ProductName ==
		"" ||
		utf8.RuneCountInString(
			input.ProductName,
		) > 180 {

		return ErrInvalidRequest
	}

	if input.Description ==
		"" ||
		utf8.RuneCountInString(
			input.Description,
		) >
			maxOfferDescriptionRunes {

		return ErrInvalidRequest
	}

	if len(
		input.Currency,
	) != 3 {

		return ErrInvalidRequest
	}

	for _, r := range input.Currency {

		if r <
			'A' ||
			r >
				'Z' {

			return ErrInvalidRequest
		}
	}

	if input.UnitPrice <
		0 ||
		input.ShippingPrice <
			0 {

		return ErrInvalidRequest
	}

	if input.QuotedQuantity !=
		nil &&
		*input.QuotedQuantity <=
			0 {

		return ErrInvalidRequest
	}

	if input.MinimumOrderQuantity !=
		nil &&
		*input.MinimumOrderQuantity <=
			0 {

		return ErrInvalidRequest
	}

	if input.QuotedQuantity !=
		nil &&
		input.MinimumOrderQuantity !=
			nil &&
		*input.QuotedQuantity <
			*input.MinimumOrderQuantity {

		return ErrInvalidRequest
	}

	if input.ExpiresAt !=
		nil &&
		!input.ExpiresAt.After(
			time.Now(),
		) {

		return ErrInvalidRequest
	}

	return nil
}

func validateOfferForSend(
	offer SourcingOffer,
) error {
	if offer.QuotedQuantity ==
		nil ||
		offer.MinimumOrderQuantity ==
			nil {

		return ErrInvalidRequest
	}

	if *offer.QuotedQuantity <=
		0 ||
		*offer.MinimumOrderQuantity <=
			0 ||
		*offer.QuotedQuantity <
			*offer.MinimumOrderQuantity {

		return ErrInvalidRequest
	}

	if offer.ExpiresAt !=
		nil &&
		!offer.ExpiresAt.After(
			time.Now(),
		) {

		return ErrOfferExpired
	}

	return nil
}

func hasOfferPatch(
	request AdminUpdateOfferRequest,
) bool {
	return request.ProductName != nil ||
		request.Description != nil ||
		request.Attachments != nil ||
		request.OfferedSpecifications != nil ||
		request.UnitPrice != nil ||
		request.ShippingPrice != nil ||
		request.Currency != nil ||
		request.QuotedQuantity != nil ||
		request.MinimumOrderQuantity != nil ||
		request.ExpiresAt != nil
}

func effectiveOffer(
	offer SourcingOffer,
) SourcingOffer {
	if offer.Status ==
		"sent" &&
		offer.ExpiresAt != nil &&
		!offer.ExpiresAt.After(
			time.Now(),
		) {

		offer.Status =
			"expired"
	}

	return offer
}

func cloneOfferInt(
	value *int,
) *int {
	if value == nil {
		return nil
	}

	result :=
		*value

	return &result
}

func cloneOfferTime(
	value *time.Time,
) *time.Time {
	if value == nil {
		return nil
	}

	result :=
		*value

	return &result
}

func safeOfferTotal(
	unitPrice int64,
	quantity int,
	shippingPrice int64,
) (
	int64,
	bool,
) {
	if unitPrice <
		0 ||
		shippingPrice <
			0 ||
		quantity <=
			0 {

		return 0,
			false
	}

	q :=
		int64(
			quantity,
		)

	if unitPrice !=
		0 &&
		q >
			math.MaxInt64/
				unitPrice {

		return 0,
			false
	}

	subtotal :=
		unitPrice *
			q

	if shippingPrice >
		math.MaxInt64-
			subtotal {

		return 0,
			false
	}

	return subtotal +
			shippingPrice,
		true
}
