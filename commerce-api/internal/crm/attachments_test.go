package crm

import (
	"encoding/json"
	"testing"
)

func TestValidateCustomerAttachmentEnvelopeAcceptsImageID(
	t *testing.T,
) {
	raw := json.RawMessage(`
		[
			{
				"type": "image",
				"kind": "image",
				"attachment_id": "11111111-1111-4111-8111-111111111111"
			}
		]
	`)

	hasAttachments, err :=
		validateCustomerAttachmentEnvelope(
			raw,
		)
	if err != nil {
		t.Fatalf(
			"validateCustomerAttachmentEnvelope returned error: %v",
			err,
		)
	}

	if !hasAttachments {
		t.Fatal(
			"expected attachments to be present",
		)
	}
}

func TestValidateCustomerAttachmentEnvelopeAcceptsImageIDWithoutKind(
	t *testing.T,
) {
	raw := json.RawMessage(`
		[
			{
				"attachment_id": "11111111-1111-4111-8111-111111111111"
			}
		]
	`)

	hasAttachments, err :=
		validateCustomerAttachmentEnvelope(
			raw,
		)
	if err != nil {
		t.Fatalf(
			"validateCustomerAttachmentEnvelope returned error: %v",
			err,
		)
	}

	if !hasAttachments {
		t.Fatal(
			"expected attachments to be present",
		)
	}
}

func TestValidateCustomerAttachmentEnvelopeRejectsLegacyBase64Write(
	t *testing.T,
) {
	raw := json.RawMessage(`
		[
			{
				"type": "image",
				"kind": "image",
				"name": "damage.jpg",
				"url": "data:image/jpeg;base64,AAAA"
			}
		]
	`)

	_, err :=
		validateCustomerAttachmentEnvelope(
			raw,
		)

	if err == nil {
		t.Fatal(
			"expected Base64 image write to be rejected",
		)
	}
}

func TestValidateCustomerAttachmentEnvelopeAcceptsHTTPSLink(
	t *testing.T,
) {
	raw := json.RawMessage(`
		[
			{
				"type": "link",
				"kind": "link",
				"name": "Order reference",
				"url": "https://example.com/orders/123"
			}
		]
	`)

	hasAttachments, err :=
		validateCustomerAttachmentEnvelope(
			raw,
		)
	if err != nil {
		t.Fatalf(
			"validateCustomerAttachmentEnvelope returned error: %v",
			err,
		)
	}

	if !hasAttachments {
		t.Fatal(
			"expected link attachment to be present",
		)
	}
}

func TestValidateCustomerAttachmentEnvelopeRejectsUnsafeLink(
	t *testing.T,
) {
	raw := json.RawMessage(`
		[
			{
				"type": "link",
				"url": "javascript:alert(1)"
			}
		]
	`)

	_, err :=
		validateCustomerAttachmentEnvelope(
			raw,
		)

	if err == nil {
		t.Fatal(
			"expected unsafe link to be rejected",
		)
	}
}

func TestValidateCustomerAttachmentEnvelopeRejectsDuplicateImageID(
	t *testing.T,
) {
	raw := json.RawMessage(`
		[
			{
				"type": "image",
				"attachment_id": "11111111-1111-4111-8111-111111111111"
			},
			{
				"type": "image",
				"attachment_id": "11111111-1111-4111-8111-111111111111"
			}
		]
	`)

	_, err :=
		validateCustomerAttachmentEnvelope(
			raw,
		)

	if err == nil {
		t.Fatal(
			"expected duplicate attachment id to be rejected",
		)
	}
}
