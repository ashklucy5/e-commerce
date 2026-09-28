package recommendation

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	platformvalidation "project.local/commerce-api/internal/platform/validation"
)

const (
	EventImpression = "impression"
	EventClick      = "click"
	EventAddToCart  = "add_to_cart"

	PlacementCatalog     = "catalog"
	PlacementProductPage = "product_page"
	PlacementHome        = "home"
	PlacementSearch      = "search"
	PlacementCart        = "cart"

	StrategyPersonalized    = "personalized"
	StrategyCategoryRelated = "category_related"
	StrategyRelated         = "related"
	StrategyComplementary   = "complementary"
	StrategyFallback        = "fallback"
	StrategyMerchandising   = "merchandising"
)

var (
	ErrInvalidEventInput = errors.New("invalid recommendation event input")
	ErrProductNotFound   = errors.New("recommendation product not found")
)

type EventInput struct {
	CustomerID   string
	SessionID    string
	ProductID    string
	Placement    string
	EventType    string
	Strategy     string
	RankPosition *int
}

type Event struct {
	ID           string    `json:"id"`
	CustomerID   string    `json:"customer_id,omitempty"`
	SessionID    string    `json:"session_id,omitempty"`
	ProductID    string    `json:"product_id"`
	Placement    string    `json:"placement"`
	EventType    string    `json:"event_type"`
	Strategy     string    `json:"strategy"`
	RankPosition *int      `json:"rank_position,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

type EventStore interface {
	RecordEvent(context.Context, EventInput) (Event, error)
}

type TelemetryService struct {
	store EventStore
}

func NewTelemetryService(store EventStore) *TelemetryService {
	return &TelemetryService{store: store}
}

func (s *TelemetryService) RecordEvent(ctx context.Context, input EventInput) (Event, error) {
	input.CustomerID = strings.TrimSpace(input.CustomerID)
	input.SessionID = strings.TrimSpace(input.SessionID)
	input.ProductID = strings.TrimSpace(input.ProductID)
	input.Placement = strings.ToLower(strings.TrimSpace(input.Placement))
	input.EventType = strings.ToLower(strings.TrimSpace(input.EventType))
	input.Strategy = strings.ToLower(strings.TrimSpace(input.Strategy))

	if input.CustomerID != "" && !platformvalidation.IsUUID(input.CustomerID) {
		return Event{}, fmt.Errorf("%w: customer_id must be a valid UUID", ErrInvalidEventInput)
	}
	if !platformvalidation.IsUUID(input.ProductID) {
		return Event{}, fmt.Errorf("%w: product_id must be a valid UUID", ErrInvalidEventInput)
	}
	if input.CustomerID == "" && input.SessionID == "" {
		return Event{}, fmt.Errorf("%w: session_id is required for anonymous events", ErrInvalidEventInput)
	}
	if len(input.SessionID) > 128 {
		return Event{}, fmt.Errorf("%w: session_id must not exceed 128 characters", ErrInvalidEventInput)
	}
	if !validPlacement(input.Placement) {
		return Event{}, fmt.Errorf("%w: unsupported placement", ErrInvalidEventInput)
	}
	if !validEventType(input.EventType) {
		return Event{}, fmt.Errorf("%w: unsupported event_type", ErrInvalidEventInput)
	}
	if !validStrategy(input.Strategy) {
		return Event{}, fmt.Errorf("%w: unsupported strategy", ErrInvalidEventInput)
	}
	if input.RankPosition != nil && (*input.RankPosition < 0 || *input.RankPosition > 10000) {
		return Event{}, fmt.Errorf("%w: rank_position must be between 0 and 10000", ErrInvalidEventInput)
	}
	if s == nil || s.store == nil {
		return Event{}, errors.New("recommendation telemetry store is not configured")
	}
	return s.store.RecordEvent(ctx, input)
}

func validPlacement(value string) bool {
	switch value {
	case PlacementCatalog, PlacementProductPage, PlacementHome, PlacementSearch, PlacementCart:
		return true
	default:
		return false
	}
}

func validEventType(value string) bool {
	switch value {
	case EventImpression, EventClick, EventAddToCart:
		return true
	default:
		return false
	}
}

func validStrategy(value string) bool {
	switch value {
	case StrategyPersonalized, StrategyCategoryRelated, StrategyRelated, StrategyComplementary, StrategyFallback, StrategyMerchandising:
		return true
	default:
		return false
	}
}
