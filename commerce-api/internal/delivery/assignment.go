package delivery

import "context"

// Provider is the provider-neutral boundary used by future courier adapters.
//
// Logistics v0 does not call a real courier yet. The manual/admin workflow
// persists provider references and canonical tracking events through Service.
type Provider interface {
	Code() string

	CreateShipment(
		ctx context.Context,
		request ProviderCreateRequest,
	) (ProviderShipment, error)

	CancelShipment(
		ctx context.Context,
		providerShipmentID string,
	) error

	GetShipment(
		ctx context.Context,
		providerShipmentID string,
	) (ProviderShipment, error)
}

type ProviderCreateRequest struct {
	OrderID        string
	ShipmentID     string
	TrackingNumber string
}

type ProviderShipment struct {
	ProviderShipmentID string
	ProviderStatus     string
	TrackingNumber     string
	TrackingURL        string
}
