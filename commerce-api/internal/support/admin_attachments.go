package support

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"unicode/utf8"

	"project.local/commerce-api/internal/supportattachment"
)

const (
	maxAdminSupportAttachmentsCount = 4
	maxAdminSupportAttachmentsBytes = 6 * 1024 * 1024
	maxAdminSupportAttachmentName   = 255
	maxAdminSupportAttachmentURL    = 4096
)

type AdminReplyWithAttachmentsRequest struct {
	Message     string          `json:"message"`
	Visibility  string          `json:"visibility"`
	Attachments json.RawMessage `json:"attachments,omitempty"`
}

type adminSupportAttachmentInput struct {
	Type string `json:"type,omitempty"`
	Kind string `json:"kind,omitempty"`

	AttachmentID string `json:"attachment_id,omitempty"`

	Name string `json:"name,omitempty"`
	URL  string `json:"url,omitempty"`
}

type adminSupportAttachmentOutput struct {
	Type string `json:"type"`
	Kind string `json:"kind"`

	AttachmentID string `json:"attachment_id,omitempty"`

	Name string `json:"name,omitempty"`

	URL string `json:"url,omitempty"`

	MimeType string `json:"mime_type,omitempty"`
	Size     int64  `json:"size,omitempty"`
}

func (s *AdminService) ReplyWithAttachments(
	ctx context.Context,
	staffID string,
	queueScoped bool,
	caseID string,
	request AdminReplyWithAttachmentsRequest,
) (SupportMessage, error) {
	if !validSupportUUID(caseID) {
		return SupportMessage{},
			ErrCaseNotFound
	}

	body :=
		strings.TrimSpace(
			request.Message,
		)

	attachments, err :=
		parseAdminSupportAttachments(
			request.Attachments,
		)
	if err != nil {
		return SupportMessage{},
			ErrInvalidMessage
	}

	/*
		Attachment-only replies remain valid.

		crm_messages.body stays non-empty so the existing
		message contract does not need to change.
	*/
	if body == "" &&
		len(attachments) > 0 {

		body = "Shared an attachment."
	}

	if body == "" ||
		utf8.RuneCountInString(
			body,
		) > 5000 {

		return SupportMessage{},
			ErrInvalidMessage
	}

	visibility :=
		strings.ToLower(
			strings.TrimSpace(
				request.Visibility,
			),
		)

	if visibility == "" {
		visibility = "customer"
	}

	if visibility != "customer" &&
		visibility != "internal" {

		return SupportMessage{},
			ErrInvalidVisibility
	}

	var actor Actor

	if queueScoped {
		actor, err =
			s.regular.getActiveActor(
				ctx,
				staffID,
			)
	} else {
		actor, err =
			s.ensureAdminActor(
				ctx,
				staffID,
			)
	}

	if err != nil {
		return SupportMessage{}, err
	}

	return s.repo.addSupportMessageWithAttachments(
		ctx,
		actor.ID,
		caseID,
		body,
		visibility,
		attachments,
	)
}

func parseAdminSupportAttachments(
	raw json.RawMessage,
) ([]adminSupportAttachmentInput, error) {
	trimmed :=
		bytes.TrimSpace(
			raw,
		)

	if len(trimmed) == 0 ||
		bytes.Equal(
			trimmed,
			[]byte("null"),
		) {

		return nil, nil
	}

	if len(trimmed) >
		maxAdminSupportAttachmentsBytes ||
		!json.Valid(trimmed) ||
		trimmed[0] != '[' {

		return nil,
			ErrInvalidMessage
	}

	var items []adminSupportAttachmentInput

	if err :=
		json.Unmarshal(
			trimmed,
			&items,
		); err != nil {

		return nil,
			ErrInvalidMessage
	}

	if len(items) == 0 {
		return nil, nil
	}

	if len(items) >
		maxAdminSupportAttachmentsCount {

		return nil,
			ErrInvalidMessage
	}

	seenAttachmentIDs :=
		make(
			map[string]struct{},
			len(items),
		)

	for index := range items {
		item :=
			&items[index]

		item.Type =
			strings.ToLower(
				strings.TrimSpace(
					item.Type,
				),
			)

		item.Kind =
			strings.ToLower(
				strings.TrimSpace(
					item.Kind,
				),
			)

		item.AttachmentID =
			strings.ToLower(
				strings.TrimSpace(
					item.AttachmentID,
				),
			)

		item.Name =
			strings.TrimSpace(
				item.Name,
			)

		item.URL =
			strings.TrimSpace(
				item.URL,
			)

		kind := item.Kind

		if kind == "" {
			kind = item.Type
		}

		if kind == "" &&
			item.AttachmentID != "" {

			kind = "image"
		}

		if kind == "" &&
			item.URL != "" {

			kind = "link"
		}

		switch kind {
		case "image":
			if !validSupportUUID(
				item.AttachmentID,
			) {
				return nil,
					ErrInvalidMessage
			}

			/*
				New support screenshots must be referenced
				by attachment ID only.

				Base64 and arbitrary image URLs are no
				longer accepted for new writes.
			*/
			if item.URL != "" {
				return nil,
					ErrInvalidMessage
			}

			if item.Name != "" &&
				utf8.RuneCountInString(
					item.Name,
				) >
					maxAdminSupportAttachmentName {

				return nil,
					ErrInvalidMessage
			}

			if _, exists :=
				seenAttachmentIDs[item.AttachmentID]; exists {

				return nil,
					ErrInvalidMessage
			}

			seenAttachmentIDs[item.AttachmentID] = struct{}{}

			item.Type = "image"
			item.Kind = "image"

		case "link":
			if item.AttachmentID != "" {
				return nil,
					ErrInvalidMessage
			}

			if item.URL == "" ||
				utf8.RuneCountInString(
					item.URL,
				) >
					maxAdminSupportAttachmentURL {

				return nil,
					ErrInvalidMessage
			}

			parsed, err :=
				url.Parse(
					item.URL,
				)

			if err != nil ||
				parsed.Host == "" ||
				(parsed.Scheme != "https" &&
					parsed.Scheme != "http") {

				return nil,
					ErrInvalidMessage
			}

			if item.Name != "" &&
				utf8.RuneCountInString(
					item.Name,
				) >
					maxAdminSupportAttachmentName {

				return nil,
					ErrInvalidMessage
			}

			item.Type = "link"
			item.Kind = "link"

		default:
			return nil,
				ErrInvalidMessage
		}
	}

	return items, nil
}

func (r *PostgresRepository) addSupportMessageWithAttachments(
	ctx context.Context,
	actorID string,
	caseID string,
	body string,
	visibility string,
	inputs []adminSupportAttachmentInput,
) (SupportMessage, error) {
	tx, err :=
		r.db.Begin(
			ctx,
		)
	if err != nil {
		return SupportMessage{},
			fmt.Errorf(
				"begin support attachment message transaction: %w",
				err,
			)
	}

	defer func() {
		_ =
			tx.Rollback(
				ctx,
			)
	}()

	assignment, err :=
		r.lockActiveAssignmentTx(
			ctx,
			tx,
			caseID,
		)
	if err != nil {
		return SupportMessage{}, err
	}

	if assignment.CaseStatus ==
		"closed" {

		return SupportMessage{},
			ErrCaseClosed
	}

	if assignment.SupportActorID !=
		actorID {

		return SupportMessage{},
			ErrCaseNotOwned
	}

	attachmentRepository :=
		supportattachment.NewRepository(
			r.db,
		)

	output :=
		make(
			[]adminSupportAttachmentOutput,
			0,
			len(inputs),
		)

	attachmentIDs :=
		make(
			[]string,
			0,
			len(inputs),
		)

	/*
		Lock all ready image attachments before inserting
		the CRM message.

		This guarantees that an attachment cannot be reused
		concurrently in multiple support messages.
	*/
	for _, item := range inputs {

		switch item.Kind {
		case "image":
			attachment, err :=
				attachmentRepository.
					LockReadyOwnedTx(
						ctx,
						tx,
						item.AttachmentID,
						supportattachment.
							UploaderSupport,
						actorID,
						caseID,
					)
			if err != nil {
				if errors.Is(
					err,
					supportattachment.
						ErrInvalidState,
				) ||
					errors.Is(
						err,
						supportattachment.
							ErrNotFound,
					) {

					return SupportMessage{},
						ErrInvalidMessage
				}

				return SupportMessage{},
					err
			}

			output =
				append(
					output,
					adminSupportAttachmentOutput{
						Type: "image",

						Kind: "image",

						AttachmentID: attachment.ID,

						Name: attachment.
							OriginalFilename,

						MimeType: attachment.
							MimeType,

						Size: attachment.
							ByteSize,
					},
				)

			attachmentIDs =
				append(
					attachmentIDs,
					attachment.ID,
				)

		case "link":
			itemOutput :=
				adminSupportAttachmentOutput{
					Type: "link",

					Kind: "link",

					URL: item.URL,
				}

			if item.Name != "" {
				itemOutput.Name =
					item.Name
			}

			output =
				append(
					output,
					itemOutput,
				)

		default:
			return SupportMessage{},
				ErrInvalidMessage
		}
	}

	var normalizedAttachments json.RawMessage
	var attachmentsArgument any

	if len(output) > 0 {
		encoded, err :=
			json.Marshal(
				output,
			)
		if err != nil {
			return SupportMessage{},
				fmt.Errorf(
					"encode support attachments: %w",
					err,
				)
		}

		normalizedAttachments =
			json.RawMessage(
				encoded,
			)

		attachmentsArgument =
			string(
				normalizedAttachments,
			)
	}

	var result SupportMessage

	err =
		tx.QueryRow(
			ctx,
			`
				INSERT INTO crm_messages (
					case_id,
					author_type,
					support_actor_id,
					visibility,
					body,
					attachments,
					created_at
				)
				VALUES (
					$1::uuid,
					'support',
					$2::uuid,
					$3,
					$4,
					$5::jsonb,
					now()
				)
				RETURNING
					id::text,
					case_id::text,
					author_type,
					visibility,
					body,
					created_at
			`,
			caseID,
			actorID,
			visibility,
			body,
			attachmentsArgument,
		).Scan(
			&result.ID,
			&result.CaseID,
			&result.AuthorType,
			&result.Visibility,
			&result.Body,
			&result.CreatedAt,
		)

	if err != nil {
		return SupportMessage{},
			fmt.Errorf(
				"insert support attachment message: %w",
				err,
			)
	}

	/*
		Only after the CRM message exists do we bind each
		ready B2 object to that message.

		The insert and every attachment bind live inside the
		same PostgreSQL transaction.
	*/
	for _, attachmentID := range attachmentIDs {

		if _, err :=
			attachmentRepository.
				BindReadyToMessageTx(
					ctx,
					tx,
					attachmentID,
					supportattachment.
						UploaderSupport,
					actorID,
					caseID,
					result.ID,
				); err != nil {

			return SupportMessage{},
				err
		}
	}

	if len(normalizedAttachments) > 0 {
		result.Attachments =
			make(
				json.RawMessage,
				len(
					normalizedAttachments,
				),
			)

		copy(
			result.Attachments,
			normalizedAttachments,
		)
	}

	var actor TicketActor

	err =
		tx.QueryRow(
			ctx,
			`
				SELECT
					id::text,
					actor_code,
					actor_type,
					display_name
				FROM support_actors
				WHERE id = $1::uuid
			`,
			actorID,
		).Scan(
			&actor.ID,
			&actor.ActorCode,
			&actor.ActorType,
			&actor.DisplayName,
		)

	if err != nil {
		return SupportMessage{},
			fmt.Errorf(
				"load support attachment message actor: %w",
				err,
			)
	}

	result.SupportActor =
		&actor

	nextStatus :=
		assignment.CaseStatus

	eventType :=
		"internal_note_added"

	if visibility ==
		"customer" {

		nextStatus =
			"waiting_customer"

		eventType =
			"support_message_added"

		_, err =
			tx.Exec(
				ctx,
				`
					UPDATE crm_cases
					SET
						status = 'waiting_customer',
						last_message_at = now(),
						last_support_message_at = now(),
						resolved_at = NULL,
						updated_at = now()
					WHERE id = $1::uuid
				`,
				caseID,
			)
	} else {
		_, err =
			tx.Exec(
				ctx,
				`
					UPDATE crm_cases
					SET
						last_message_at = now(),
						last_support_message_at = now(),
						updated_at = now()
					WHERE id = $1::uuid
				`,
				caseID,
			)
	}

	if err != nil {
		return SupportMessage{},
			fmt.Errorf(
				"update CRM support attachment message state: %w",
				err,
			)
	}

	if err :=
		insertSupportEventTx(
			ctx,
			tx,
			caseID,
			actorID,
			eventType,
			assignment.CaseStatus,
			nextStatus,
			map[string]any{
				"visibility": visibility,

				"attachment_count": len(output),
			},
		); err != nil {

		return SupportMessage{},
			err
	}

	/*
		Internal notes never notify customers.

		Customer-visible support replies retain the
		existing notification behavior.
	*/
	if visibility ==
		"customer" {

		if err :=
			r.enqueueSupportReplyNotificationTx(
				ctx,
				tx,
				caseID,
				result.ID,
			); err != nil {

			return SupportMessage{},
				err
		}
	}

	if err :=
		tx.Commit(
			ctx,
		); err != nil {

		return SupportMessage{},
			fmt.Errorf(
				"commit support attachment message: %w",
				err,
			)
	}

	return result, nil
}
