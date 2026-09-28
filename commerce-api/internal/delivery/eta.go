package delivery

import (
	"context"
	"strings"
	"time"
)

type ETAEstimateInput struct {
	OrderID           string    `json:"order_id,omitempty"`
	ShipmentID        string    `json:"shipment_id,omitempty"`
	OriginWarehouseID string    `json:"origin_warehouse_id,omitempty"`
	DeliveryMode      string    `json:"delivery_mode"`
	ProviderCode      string    `json:"provider_code,omitempty"`
	DestinationCity   string    `json:"destination_city,omitempty"`
	DestinationArea   string    `json:"destination_area,omitempty"`
	ServiceLevel      string    `json:"service_level,omitempty"`
	DispatchedAt      time.Time `json:"dispatched_at,omitempty"`
}

type ETAEstimate struct {
	EarliestAt   time.Time `json:"earliest_at"`
	LatestAt     time.Time `json:"latest_at"`
	EstimatedAt  time.Time `json:"estimated_at"`
	Source       string    `json:"source"`
	ProviderCode string    `json:"provider_code,omitempty"`
	ServiceLevel string    `json:"service_level,omitempty"`
	Note         string    `json:"note,omitempty"`
}

type ETAEstimator interface {
	EstimateETA(
		ctx context.Context,
		input ETAEstimateInput,
	) (ETAEstimate, error)
}

// FixedETAEstimator is useful for a manual/default policy.
//
// It is intentionally simple. Later a Pathao/eCourier/other provider
// adapter can implement ETAEstimator without changing Delivery Service.
type FixedETAEstimator struct {
	EarliestAfter time.Duration
	LatestAfter   time.Duration
	Source        string
	Note          string
}

func NewFixedETAEstimator(
	earliestAfter time.Duration,
	latestAfter time.Duration,
	source string,
) *FixedETAEstimator {
	return &FixedETAEstimator{
		EarliestAfter: earliestAfter,
		LatestAfter:   latestAfter,
		Source:        strings.TrimSpace(source),
	}
}

func (e *FixedETAEstimator) EstimateETA(
	ctx context.Context,
	input ETAEstimateInput,
) (ETAEstimate, error) {
	_ = ctx

	if e == nil ||
		e.EarliestAfter < 0 ||
		e.LatestAfter < e.EarliestAfter {
		return ETAEstimate{},
			ErrInvalidInput
	}

	base := input.DispatchedAt

	if base.IsZero() {
		base = time.Now().UTC()
	} else {
		base = base.UTC()
	}

	source := strings.TrimSpace(
		e.Source,
	)

	if source == "" {
		source = "manual"
	}

	return ETAEstimate{
		EarliestAt: base.Add(
			e.EarliestAfter,
		),

		LatestAt: base.Add(
			e.LatestAfter,
		),

		EstimatedAt: time.Now().UTC(),

		Source: source,

		ProviderCode: strings.ToLower(
			strings.TrimSpace(
				input.ProviderCode,
			),
		),

		ServiceLevel: strings.TrimSpace(
			input.ServiceLevel,
		),

		Note: strings.TrimSpace(
			e.Note,
		),
	}, nil
}

func (s *Service) SetETAEstimator(
	estimator ETAEstimator,
) {
	s.etaEstimator = estimator
}

func (s *Service) EstimateETA(
	ctx context.Context,
	input ETAEstimateInput,
) (ETAEstimate, error) {
	input.OrderID =
		strings.TrimSpace(
			input.OrderID,
		)

	input.ShipmentID =
		strings.TrimSpace(
			input.ShipmentID,
		)

	input.OriginWarehouseID =
		strings.TrimSpace(
			input.OriginWarehouseID,
		)

	input.DeliveryMode =
		strings.ToLower(
			strings.TrimSpace(
				input.DeliveryMode,
			),
		)

	input.ProviderCode =
		strings.ToLower(
			strings.TrimSpace(
				input.ProviderCode,
			),
		)

	input.DestinationCity =
		strings.TrimSpace(
			input.DestinationCity,
		)

	input.DestinationArea =
		strings.TrimSpace(
			input.DestinationArea,
		)

	input.ServiceLevel =
		strings.TrimSpace(
			input.ServiceLevel,
		)

	if !validDeliveryMode(
		input.DeliveryMode,
	) ||
		(input.OrderID != "" &&
			!uuidPattern.MatchString(
				input.OrderID,
			)) ||
		(input.ShipmentID != "" &&
			!uuidPattern.MatchString(
				input.ShipmentID,
			)) ||
		(input.OriginWarehouseID != "" &&
			!uuidPattern.MatchString(
				input.OriginWarehouseID,
			)) {
		return ETAEstimate{},
			ErrInvalidInput
	}

	if s.etaEstimator == nil {
		return ETAEstimate{},
			ErrETAEstimatorUnavailable
	}

	result, err :=
		s.etaEstimator.EstimateETA(
			ctx,
			input,
		)
	if err != nil {
		return ETAEstimate{}, err
	}

	if result.EarliestAt.IsZero() ||
		result.LatestAt.IsZero() ||
		result.LatestAt.Before(
			result.EarliestAt,
		) {
		return ETAEstimate{},
			ErrInvalidInput
	}

	if result.EstimatedAt.IsZero() {
		result.EstimatedAt =
			time.Now().UTC()
	}

	if strings.TrimSpace(
		result.Source,
	) == "" {
		result.Source =
			"manual"
	}

	return result, nil
}
