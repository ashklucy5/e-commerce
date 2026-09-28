package notification

import (
	"context"
	"errors"
	"testing"
	"time"
)

type dispatcherTestStore struct {
	items []OutboxItem

	preferenceEnabled bool

	sent []string

	skipped map[string]string

	retried []string

	dead []string
}

func (
	s *dispatcherTestStore,
) MarkExhausted(
	context.Context,
	time.Time,
	time.Time,
) (
	int64,
	error,
) {
	return 0,
		nil
}

func (
	s *dispatcherTestStore,
) ClaimDue(
	context.Context,
	time.Time,
	time.Time,
	int,
) (
	[]OutboxItem,
	error,
) {
	return s.items,
		nil
}

func (
	s *dispatcherTestStore,
) PreferenceEnabled(
	context.Context,
	OutboxItem,
) (
	bool,
	error,
) {
	return s.preferenceEnabled,
		nil
}

func (
	s *dispatcherTestStore,
) MarkSent(
	_ context.Context,
	id string,
	_ string,
	_ time.Time,
) error {
	s.sent =
		append(
			s.sent,
			id,
		)

	return nil
}

func (
	s *dispatcherTestStore,
) MarkSkipped(
	_ context.Context,
	id string,
	reason string,
	_ time.Time,
) error {
	if s.skipped == nil {
		s.skipped =
			make(
				map[string]string,
			)
	}

	s.skipped[id] =
		reason

	return nil
}

func (
	s *dispatcherTestStore,
) MarkRetry(
	_ context.Context,
	id string,
	_ string,
	_ time.Time,
	_ time.Time,
) error {
	s.retried =
		append(
			s.retried,
			id,
		)

	return nil
}

func (
	s *dispatcherTestStore,
) MarkDead(
	_ context.Context,
	id string,
	_ string,
	_ time.Time,
) error {
	s.dead =
		append(
			s.dead,
			id,
		)

	return nil
}

type dispatcherTestSender struct {
	channel string

	sendErr error

	calls int
}

func (
	s *dispatcherTestSender,
) Channel() string {
	return s.channel
}

func (
	s *dispatcherTestSender,
) Ready() error {
	return nil
}

func (
	s *dispatcherTestSender,
) Send(
	context.Context,
	SendRequest,
) (
	SendResult,
	error,
) {
	s.calls++

	if s.sendErr != nil {
		return SendResult{},
			s.sendErr
	}

	return SendResult{
		ProviderMessageID: "provider-message-1",
	}, nil
}

func dispatcherFixture() OutboxItem {
	return OutboxItem{
		ID: "11111111-1111-4111-8111-111111111111",

		DedupeKey: "order:test:placed:sms",

		Category: CategoryOrder,

		EventType: EventOrderPlaced,

		Channel: ChannelSMS,

		Recipient: "+8801700000000",

		TemplateKey: TemplateOrderPlaced,

		Payload: []byte(
			`{"message":"Your order has been confirmed"}`,
		),

		Status: StatusProcessing,

		AttemptCount: 1,

		MaxAttempts: 5,
	}
}

func dispatcherTemplates(
	t *testing.T,
) *TemplateRegistry {
	t.Helper()

	registry, err :=
		NewDefaultTemplateRegistry()
	if err != nil {
		t.Fatalf(
			"create templates: %v",
			err,
		)
	}

	return registry
}

func TestDispatcherSendsConfiguredChannel(
	t *testing.T,
) {
	item :=
		dispatcherFixture()

	store :=
		&dispatcherTestStore{
			items: []OutboxItem{
				item,
			},

			preferenceEnabled: true,
		}

	sender :=
		&dispatcherTestSender{
			channel: ChannelSMS,
		}

	channels :=
		NewChannelRegistry()

	if err :=
		channels.Register(
			sender,
		); err != nil {

		t.Fatalf(
			"register sender: %v",
			err,
		)
	}

	dispatcher :=
		&Dispatcher{
			repository: store,

			channels: channels,

			templates: dispatcherTemplates(
				t,
			),

			lockTimeout: time.Minute,
		}

	result, err :=
		dispatcher.DispatchDue(
			context.Background(),
			time.Now(),
			100,
		)
	if err != nil {
		t.Fatalf(
			"dispatch notifications: %v",
			err,
		)
	}

	if result.Sent != 1 ||
		len(
			store.sent,
		) != 1 ||
		sender.calls != 1 {

		t.Fatalf(
			"unexpected send result: %#v",
			result,
		)
	}
}

func TestDispatcherSkipsUnconfiguredProvider(
	t *testing.T,
) {
	item :=
		dispatcherFixture()

	store :=
		&dispatcherTestStore{
			items: []OutboxItem{
				item,
			},

			preferenceEnabled: true,
		}

	dispatcher :=
		&Dispatcher{
			repository: store,

			channels: NewChannelRegistry(),

			templates: dispatcherTemplates(
				t,
			),

			lockTimeout: time.Minute,
		}

	result, err :=
		dispatcher.DispatchDue(
			context.Background(),
			time.Now(),
			100,
		)
	if err != nil {
		t.Fatalf(
			"dispatch notifications: %v",
			err,
		)
	}

	if result.Skipped != 1 {
		t.Fatalf(
			"expected skipped notification: %#v",
			result,
		)
	}

	if store.skipped[item.ID] !=
		"provider_unavailable" {

		t.Fatalf(
			"unexpected skip reason %q",
			store.skipped[item.ID],
		)
	}
}

func TestDispatcherHonorsPreference(
	t *testing.T,
) {
	item :=
		dispatcherFixture()

	store :=
		&dispatcherTestStore{
			items: []OutboxItem{
				item,
			},

			preferenceEnabled: false,
		}

	sender :=
		&dispatcherTestSender{
			channel: ChannelSMS,
		}

	channels :=
		NewChannelRegistry()

	if err :=
		channels.Register(
			sender,
		); err != nil {

		t.Fatalf(
			"register sender: %v",
			err,
		)
	}

	dispatcher :=
		&Dispatcher{
			repository: store,

			channels: channels,

			templates: dispatcherTemplates(
				t,
			),

			lockTimeout: time.Minute,
		}

	result, err :=
		dispatcher.DispatchDue(
			context.Background(),
			time.Now(),
			100,
		)
	if err != nil {
		t.Fatalf(
			"dispatch notifications: %v",
			err,
		)
	}

	if result.Skipped != 1 ||
		sender.calls != 0 {

		t.Fatalf(
			"preference was not respected: %#v",
			result,
		)
	}

	if store.skipped[item.ID] !=
		"preference_disabled" {

		t.Fatalf(
			"unexpected preference skip reason %q",
			store.skipped[item.ID],
		)
	}
}

func TestDispatcherRetriesTransientSendFailure(
	t *testing.T,
) {
	item :=
		dispatcherFixture()

	store :=
		&dispatcherTestStore{
			items: []OutboxItem{
				item,
			},

			preferenceEnabled: true,
		}

	sender :=
		&dispatcherTestSender{
			channel: ChannelSMS,

			sendErr: errors.New(
				"temporary provider failure",
			),
		}

	channels :=
		NewChannelRegistry()

	if err :=
		channels.Register(
			sender,
		); err != nil {

		t.Fatalf(
			"register sender: %v",
			err,
		)
	}

	dispatcher :=
		&Dispatcher{
			repository: store,

			channels: channels,

			templates: dispatcherTemplates(
				t,
			),

			lockTimeout: time.Minute,
		}

	result, err :=
		dispatcher.DispatchDue(
			context.Background(),
			time.Now(),
			100,
		)
	if err != nil {
		t.Fatalf(
			"dispatch notifications: %v",
			err,
		)
	}

	if result.Retried != 1 ||
		len(
			store.retried,
		) != 1 {

		t.Fatalf(
			"expected retry: %#v",
			result,
		)
	}
}

func TestDispatcherDeadAfterMaxAttempts(
	t *testing.T,
) {
	item :=
		dispatcherFixture()

	item.AttemptCount =
		item.MaxAttempts

	store :=
		&dispatcherTestStore{
			items: []OutboxItem{
				item,
			},

			preferenceEnabled: true,
		}

	sender :=
		&dispatcherTestSender{
			channel: ChannelSMS,

			sendErr: errors.New(
				"provider remains unavailable",
			),
		}

	channels :=
		NewChannelRegistry()

	if err :=
		channels.Register(
			sender,
		); err != nil {

		t.Fatalf(
			"register sender: %v",
			err,
		)
	}

	dispatcher :=
		&Dispatcher{
			repository: store,

			channels: channels,

			templates: dispatcherTemplates(
				t,
			),

			lockTimeout: time.Minute,
		}

	result, err :=
		dispatcher.DispatchDue(
			context.Background(),
			time.Now(),
			100,
		)
	if err != nil {
		t.Fatalf(
			"dispatch notifications: %v",
			err,
		)
	}

	if result.Dead != 1 ||
		len(
			store.dead,
		) != 1 {

		t.Fatalf(
			"expected terminal failure: %#v",
			result,
		)
	}
}
