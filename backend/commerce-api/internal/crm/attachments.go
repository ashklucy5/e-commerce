package crm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"strings"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"

	"project.local/commerce-api/internal/supportattachment"
)

type customerAttachmentInput struct {
	Type string `json:"type,omitempty"`
	Kind string `json:"kind,omitempty"`

	AttachmentID string `json:"attachment_id,omitempty"`

	Name string `json:"name,omitempty"`
	URL  string `json:"url,omitempty"`
}

type customerAttachmentOutput struct {
	Type string `json:"type"`
	Kind string `json:"kind"`

	AttachmentID string `json:"attachment_id,omitempty"`

	Name string `json:"name,omitempty"`

	URL string `json:"url,omitempty"`

	MimeType string `json:"mime_type,omitempty"`
	Size     int64  `json:"size,omitempty"`
}

func validateCustomerAttachmentEnvelope(
	raw json.RawMessage,
) (bool, error) {
	items, err := parseCustomerAttachmentInputs(raw)
	if err != nil {
		return false, err
	}

	return len(items) > 0, nil
}

func parseCustomerAttachmentInputs(
	raw json.RawMessage,
) ([]customerAttachmentInput, error) {
	trimmed := bytes.TrimSpace(raw)

	if len(trimmed) == 0 ||
		bytes.Equal(
			trimmed,
			[]byte("null"),
		) {
		return nil, nil
	}

	if len(trimmed) > maxCRMAttachmentsBytes {
		return nil, ErrInvalidRequest
	}

	if !json.Valid(trimmed) ||
		trimmed[0] != '[' {
		return nil, ErrInvalidRequest
	}

	var items []customerAttachmentInput

	if err := json.Unmarshal(
		trimmed,
		&items,
	); err != nil {
		return nil, ErrInvalidRequest
	}

	if len(items) == 0 {
		return nil, nil
	}

	if len(items) > maxCRMAttachmentsCount {
		return nil, ErrInvalidRequest
	}

	seenAttachmentIDs := make(
		map[string]struct{},
		len(items),
	)

	for index := range items {
		item := &items[index]

		item.Type = strings.ToLower(
			strings.TrimSpace(
				item.Type,
			),
		)

		item.Kind = strings.ToLower(
			strings.TrimSpace(
				item.Kind,
			),
		)

		item.AttachmentID = strings.ToLower(
			strings.TrimSpace(
				item.AttachmentID,
			),
		)

		item.Name = strings.TrimSpace(
			item.Name,
		)

		item.URL = strings.TrimSpace(
			item.URL,
		)

		kind := item.Kind

		if kind == "" {
			kind = item.Type
		}

		/*
			Allow attachment_id itself to imply an image.

			This keeps the API tolerant if a frontend sends:

				{
					"attachment_id": "..."
				}

			while still canonicalizing the stored JSON later.
		*/
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
			if !validUUID(
				item.AttachmentID,
			) {
				return nil, ErrInvalidRequest
			}

			/*
				New image writes must never contain
				a URL or data:image payload.

				The URL is generated only after an
				authorized download request.
			*/
			if item.URL != "" {
				return nil, ErrInvalidRequest
			}

			if item.Name != "" &&
				utf8.RuneCountInString(
					item.Name,
				) > maxCRMAttachmentNameRunes {
				return nil, ErrInvalidRequest
			}

			if _, exists :=
				seenAttachmentIDs[item.AttachmentID]; exists {
				return nil, ErrInvalidRequest
			}

			seenAttachmentIDs[item.AttachmentID] = struct{}{}

			item.Type = "image"
			item.Kind = "image"

		case "link":
			if item.AttachmentID != "" {
				return nil, ErrInvalidRequest
			}

			if item.URL == "" ||
				utf8.RuneCountInString(
					item.URL,
				) > maxCRMAttachmentURLRunes {
				return nil, ErrInvalidRequest
			}

			parsed, err := url.Parse(
				item.URL,
			)
			if err != nil {
				return nil, ErrInvalidRequest
			}

			if parsed.Host == "" ||
				(parsed.Scheme != "https" &&
					parsed.Scheme != "http") {
				return nil, ErrInvalidRequest
			}

			if item.Name != "" &&
				utf8.RuneCountInString(
					item.Name,
				) > maxCRMAttachmentNameRunes {
				return nil, ErrInvalidRequest
			}

			item.Type = "link"
			item.Kind = "link"

		default:
			return nil, ErrInvalidRequest
		}
	}

	return items, nil
}

func (s *Service) prepareCustomerAttachmentsTx(
	ctx context.Context,
	tx pgx.Tx,
	customerID string,
	caseID string,
	raw json.RawMessage,
) (
	json.RawMessage,
	[]string,
	error,
) {
	items, err :=
		parseCustomerAttachmentInputs(
			raw,
		)
	if err != nil {
		return nil, nil, err
	}

	if len(items) == 0 {
		return nil, nil, nil
	}

	attachmentRepository :=
		supportattachment.NewRepository(
			s.repository.db,
		)

	output :=
		make(
			[]customerAttachmentOutput,
			0,
			len(items),
		)

	attachmentIDs :=
		make(
			[]string,
			0,
			len(items),
		)

	for _, item := range items {
		switch item.Kind {
		case "image":
			attachment, err :=
				attachmentRepository.
					LockReadyOwnedTx(
						ctx,
						tx,
						item.AttachmentID,
						supportattachment.
							UploaderCustomer,
						customerID,
						caseID,
					)
			if err != nil {
				/*
					Do not reveal whether an attachment
					exists or belongs to somebody else.
				*/
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
					return nil,
						nil,
						ErrInvalidRequest
				}

				return nil, nil, err
			}

			output = append(
				output,
				customerAttachmentOutput{
					Type: "image",
					Kind: "image",

					AttachmentID: attachment.ID,

					Name: attachment.
						OriginalFilename,

					MimeType: attachment.MimeType,

					Size: attachment.ByteSize,
				},
			)

			attachmentIDs = append(
				attachmentIDs,
				attachment.ID,
			)

		case "link":
			outputItem :=
				customerAttachmentOutput{
					Type: "link",
					Kind: "link",

					URL: item.URL,
				}

			if item.Name != "" {
				outputItem.Name =
					item.Name
			}

			output = append(
				output,
				outputItem,
			)

		default:
			return nil,
				nil,
				ErrInvalidRequest
		}
	}

	encoded, err := json.Marshal(
		output,
	)
	if err != nil {
		return nil, nil, err
	}

	return json.RawMessage(encoded),
		attachmentIDs,
		nil
}

func (s *Service) bindCustomerAttachmentsTx(
	ctx context.Context,
	tx pgx.Tx,
	customerID string,
	caseID string,
	messageID string,
	attachmentIDs []string,
) error {
	if len(attachmentIDs) == 0 {
		return nil
	}

	attachmentRepository :=
		supportattachment.NewRepository(
			s.repository.db,
		)

	for _, attachmentID := range attachmentIDs {

		if _, err :=
			attachmentRepository.
				BindReadyToMessageTx(
					ctx,
					tx,
					attachmentID,
					supportattachment.
						UploaderCustomer,
					customerID,
					caseID,
					messageID,
				); err != nil {

			return err
		}
	}

	return nil
}
