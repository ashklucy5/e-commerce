package notification

import (
	"context"
	"errors"
	"testing"
)

type fakeSender struct {
	channel string

	readyErr error
}

func (s *fakeSender) Channel() string {
	return s.channel
}

func (s *fakeSender) Ready() error {
	return s.readyErr
}

func (s *fakeSender) Send(
	context.Context,
	SendRequest,
) (
	SendResult,
	error,
) {
	return SendResult{
		ProviderMessageID: "fake-message",
	}, nil
}

func TestChannelDedupeKey(
	t *testing.T,
) {
	value :=
		ChannelDedupeKey(
			"order:123:placed",
			"SMS",
		)

	if value !=
		"order:123:placed:sms" {

		t.Fatalf(
			"unexpected dedupe key %q",
			value,
		)
	}
}

func TestOutboxRejectsSensitivePayload(
	t *testing.T,
) {
	request :=
		EnqueueRequest{
			DedupeKey: "security:test:email",

			Category: CategorySecurity,

			EventType: "security.test",

			Channel: ChannelEmail,

			TemplateKey: TemplateSupportReply,

			Payload: map[string]any{
				"message": "hello",

				"access_token": "must-never-be-stored",
			},
		}

	_, err :=
		normalizeAndValidateEnqueueRequest(
			&request,
		)

	if !errors.Is(
		err,
		ErrInvalidInput,
	) {
		t.Fatalf(
			"expected sensitive payload rejection, got %v",
			err,
		)
	}
}

func TestDefaultTemplateRegistry(
	t *testing.T,
) {
	registry, err :=
		NewDefaultTemplateRegistry()
	if err != nil {
		t.Fatalf(
			"create default templates: %v",
			err,
		)
	}

	rendered, err :=
		registry.Render(
			TemplateDeliveryUpdated,
			map[string]any{
				"message": "Your shipment has arrived in Bangladesh",
			},
		)
	if err != nil {
		t.Fatalf(
			"render template: %v",
			err,
		)
	}

	if rendered.Subject !=
		"Delivery update" {

		t.Fatalf(
			"unexpected subject %q",
			rendered.Subject,
		)
	}

	if rendered.Text !=
		"Your shipment has arrived in Bangladesh" {

		t.Fatalf(
			"unexpected text %q",
			rendered.Text,
		)
	}
}

func TestChannelRegistry(
	t *testing.T,
) {
	registry :=
		NewChannelRegistry()

	if err :=
		registry.Register(
			&fakeSender{
				channel: ChannelSMS,
			},
		); err != nil {

		t.Fatalf(
			"register sender: %v",
			err,
		)
	}

	sender, err :=
		registry.Resolve(
			ChannelSMS,
		)
	if err != nil {
		t.Fatalf(
			"resolve sender: %v",
			err,
		)
	}

	if sender.Channel() !=
		ChannelSMS {

		t.Fatalf(
			"unexpected sender channel %q",
			sender.Channel(),
		)
	}

	_, err =
		registry.Resolve(
			ChannelEmail,
		)

	if !errors.Is(
		err,
		ErrChannelNotConfigured,
	) {
		t.Fatalf(
			"expected unconfigured channel, got %v",
			err,
		)
	}
}
