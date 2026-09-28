package delivery

import (
	"context"
	"strings"
	"time"
)

type DeliveryQuoteInput struct {
	OrderID           string `json:"order_id,omitempty"`
	ShipmentID        string `json:"shipment_id,omitempty"`
	OriginWarehouseID string `json:"origin_warehouse_id,omitempty"`

	DeliveryMode string `json:"delivery_mode"`
	ProviderCode string `json:"provider_code,omitempty"`
	ServiceLevel string `json:"service_level,omitempty"`

	DestinationCity string `json:"destination_city,omitempty"`
	DestinationArea string `json:"destination_area,omitempty"`

	ParcelCount       int   `json:"parcel_count,omitempty"`
	ParcelWeightGrams int64 `json:"parcel_weight_grams,omitempty"`

	// CODAmount is allowed as a delivery-pricing input because some
	// delivery providers charge a COD handling fee.
	//
	// It does not couple this package to product pricing.
	CODAmount int64 `json:"cod_amount,omitempty"`

	Currency string `json:"currency"`
}

type DeliveryQuote struct {
	CustomerChargeAmount int64 `json:"customer_charge_amount"`

	// ProviderCostAmount is the operational provider cost.
	// CustomerChargeAmount is what the commerce layer may charge
	// to the customer.
	ProviderCostAmount int64 `json:"provider_cost_amount"`

	Currency string `json:"currency"`

	Source       string `json:"source"`
	ProviderCode string `json:"provider_code,omitempty"`
	ServiceLevel string `json:"service_level,omitempty"`

	QuotedAt  time.Time  `json:"quoted_at"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`

	Note string `json:"note,omitempty"`
}

type DeliveryPricer interface {
	QuoteDelivery(
		ctx context.Context,
		input DeliveryQuoteInput,
	) (DeliveryQuote, error)
}

// FixedDeliveryPricer provides a manual/default quote implementation.
//
// A real provider later only needs to implement DeliveryPricer.
type FixedDeliveryPricer struct {
	CustomerChargeAmount int64
	ProviderCostAmount   int64

	Currency string
	Source   string

	TTL time.Duration

	Note string
}

func NewFixedDeliveryPricer(
	customerChargeAmount int64,
	providerCostAmount int64,
	currency string,
	source string,
) *FixedDeliveryPricer {
	return &FixedDeliveryPricer{
		CustomerChargeAmount: customerChargeAmount,

		ProviderCostAmount: providerCostAmount,

		Currency: strings.ToUpper(
			strings.TrimSpace(
				currency,
			),
		),

		Source: strings.TrimSpace(
			source,
		),
	}
}

func (p *FixedDeliveryPricer) QuoteDelivery(
	ctx context.Context,
	input DeliveryQuoteInput,
) (DeliveryQuote, error) {
	_ = ctx

	if p == nil ||
		p.CustomerChargeAmount < 0 ||
		p.ProviderCostAmount < 0 {
		return DeliveryQuote{},
			ErrInvalidInput
	}

	currency :=
		strings.ToUpper(
			strings.TrimSpace(
				p.Currency,
			),
		)

	if currency == "" {
		currency =
			strings.ToUpper(
				strings.TrimSpace(
					input.Currency,
				),
			)
	}

	if len(
		currency,
	) != 3 {
		return DeliveryQuote{},
			ErrInvalidInput
	}

	source :=
		strings.TrimSpace(
			p.Source,
		)

	if source == "" {
		source =
			"manual"
	}

	now :=
		time.Now().UTC()

	result :=
		DeliveryQuote{
			CustomerChargeAmount: p.CustomerChargeAmount,

			ProviderCostAmount: p.ProviderCostAmount,

			Currency: currency,

			Source: source,

			ProviderCode: strings.ToLower(
				strings.TrimSpace(
					input.ProviderCode,
				),
			),

			ServiceLevel: strings.TrimSpace(
				input.ServiceLevel,
			),

			QuotedAt: now,

			Note: strings.TrimSpace(
				p.Note,
			),
		}

	if p.TTL > 0 {
		expiresAt :=
			now.Add(
				p.TTL,
			)

		result.ExpiresAt =
			&expiresAt
	}

	return result, nil
}

func (s *Service) SetDeliveryPricer(
	pricer DeliveryPricer,
) {
	s.pricer = pricer
}

func (s *Service) QuoteDelivery(
	ctx context.Context,
	input DeliveryQuoteInput,
) (DeliveryQuote, error) {
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

	input.Currency =
		strings.ToUpper(
			strings.TrimSpace(
				input.Currency,
			),
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
			)) ||
		input.ParcelCount < 0 ||
		input.ParcelWeightGrams < 0 ||
		input.CODAmount < 0 ||
		len(
			input.Currency,
		) != 3 {
		return DeliveryQuote{},
			ErrInvalidInput
	}

	if s.pricer == nil {
		return DeliveryQuote{},
			ErrDeliveryPricerUnavailable
	}

	result, err :=
		s.pricer.QuoteDelivery(
			ctx,
			input,
		)
	if err != nil {
		return DeliveryQuote{}, err
	}

	result.Currency =
		strings.ToUpper(
			strings.TrimSpace(
				result.Currency,
			),
		)

	result.Source =
		strings.TrimSpace(
			result.Source,
		)

	if result.CustomerChargeAmount < 0 ||
		result.ProviderCostAmount < 0 ||
		len(
			result.Currency,
		) != 3 ||
		result.Source == "" {
		return DeliveryQuote{},
			ErrInvalidInput
	}

	if result.QuotedAt.IsZero() {
		result.QuotedAt =
			time.Now().UTC()
	}

	return result, nil
}
