package recommendation

import (
	"context"
	"errors"
	"testing"
)

type fakeEventStore struct {
	input EventInput
	event Event
	err   error
}

func (f *fakeEventStore) RecordEvent(_ context.Context, input EventInput) (Event, error) {
	f.input = input
	if f.err != nil {
		return Event{}, f.err
	}
	return f.event, nil
}

func TestTelemetryServiceNormalizesAndRecordsAuthenticatedEvent(t *testing.T) {
	rank := 3
	store := &fakeEventStore{event: Event{ID: "event-1"}}
	service := NewTelemetryService(store)

	_, err := service.RecordEvent(context.Background(), EventInput{
		CustomerID:   "11111111-1111-4111-8111-111111111111",
		SessionID:    "  session-1  ",
		ProductID:    "22222222-2222-4222-8222-222222222222",
		Placement:    "  CATALOG ",
		EventType:    " CLICK ",
		Strategy:     " PERSONALIZED ",
		RankPosition: &rank,
	})
	if err != nil {
		t.Fatalf("record event: %v", err)
	}

	if store.input.SessionID != "session-1" || store.input.Placement != PlacementCatalog || store.input.EventType != EventClick || store.input.Strategy != StrategyPersonalized {
		t.Fatalf("event was not normalized: %#v", store.input)
	}
}

func TestTelemetryServiceRequiresAnonymousSession(t *testing.T) {
	service := NewTelemetryService(&fakeEventStore{})
	_, err := service.RecordEvent(context.Background(), EventInput{
		ProductID: "22222222-2222-4222-8222-222222222222",
		Placement: PlacementHome,
		EventType: EventImpression,
		Strategy:  StrategyMerchandising,
	})
	if !errors.Is(err, ErrInvalidEventInput) {
		t.Fatalf("expected invalid input, got %v", err)
	}
}

func TestTelemetryServiceRejectsUnknownDimensions(t *testing.T) {
	service := NewTelemetryService(&fakeEventStore{})
	for _, input := range []EventInput{
		{SessionID: "s", ProductID: "22222222-2222-4222-8222-222222222222", Placement: "unknown", EventType: EventImpression, Strategy: StrategyFallback},
		{SessionID: "s", ProductID: "22222222-2222-4222-8222-222222222222", Placement: PlacementHome, EventType: "unknown", Strategy: StrategyFallback},
		{SessionID: "s", ProductID: "22222222-2222-4222-8222-222222222222", Placement: PlacementHome, EventType: EventImpression, Strategy: "unknown"},
	} {
		if _, err := service.RecordEvent(context.Background(), input); !errors.Is(err, ErrInvalidEventInput) {
			t.Fatalf("expected invalid input for %#v, got %v", input, err)
		}
	}
}
