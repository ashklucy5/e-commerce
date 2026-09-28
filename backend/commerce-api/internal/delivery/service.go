package delivery

import (
	"context"
	"net/url"
	"regexp"
	"strings"
	"unicode/utf8"
)

var uuidPattern = regexp.MustCompile(
	`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`,
)

type Service struct {
	repository *Repository
	providers  map[string]Provider

	etaEstimator ETAEstimator
	pricer       DeliveryPricer
	otpSender    OTPSender
	otpConfig    OTPConfig
}

func NewService(
	repository *Repository,
) *Service {
	return &Service{
		repository: repository,
		providers:  make(map[string]Provider),
		otpConfig:  DefaultOTPConfig(),
	}
}

func (s *Service) RegisterProvider(
	provider Provider,
) {
	if provider == nil {
		return
	}

	code := strings.ToLower(
		strings.TrimSpace(
			provider.Code(),
		),
	)

	if code == "" {
		return
	}

	s.providers[code] = provider
}

func normalizePrepareShipmentRequest(
	request *PrepareShipmentRequest,
) {
	request.DeliveryMode = strings.ToLower(
		strings.TrimSpace(
			request.DeliveryMode,
		),
	)

	request.ProviderCode = strings.ToLower(
		strings.TrimSpace(
			request.ProviderCode,
		),
	)

	request.ProviderShipmentID = strings.TrimSpace(
		request.ProviderShipmentID,
	)

	request.ProviderStatus = strings.TrimSpace(
		request.ProviderStatus,
	)

	request.CourierName = strings.TrimSpace(
		request.CourierName,
	)

	request.CourierReference = strings.TrimSpace(
		request.CourierReference,
	)

	request.RiderReference = strings.TrimSpace(
		request.RiderReference,
	)

	request.TrackingNumber = strings.TrimSpace(
		request.TrackingNumber,
	)

	request.TrackingURL = strings.TrimSpace(
		request.TrackingURL,
	)
}

func validatePrepareShipmentRequest(
	request PrepareShipmentRequest,
) error {
	if !validDeliveryMode(
		request.DeliveryMode,
	) {
		return ErrInvalidInput
	}

	if utf8.RuneCountInString(
		request.ProviderCode,
	) > 60 ||
		utf8.RuneCountInString(
			request.ProviderShipmentID,
		) > 160 ||
		utf8.RuneCountInString(
			request.ProviderStatus,
		) > 80 ||
		utf8.RuneCountInString(
			request.CourierName,
		) > 120 ||
		utf8.RuneCountInString(
			request.CourierReference,
		) > 160 ||
		utf8.RuneCountInString(
			request.RiderReference,
		) > 160 ||
		utf8.RuneCountInString(
			request.TrackingNumber,
		) > 160 ||
		utf8.RuneCountInString(
			request.TrackingURL,
		) > 1000 {
		return ErrInvalidInput
	}

	if request.DeliveryMode ==
		DeliveryModeCourier &&
		request.CourierName == "" &&
		request.ProviderCode == "" {
		return ErrInvalidInput
	}

	if request.DeliveryMode ==
		DeliveryModeCommunityRider &&
		request.RiderReference == "" {
		return ErrInvalidInput
	}

	if request.DeliveryMode ==
		DeliveryModeSelfPickup &&
		(request.ProviderCode != "" ||
			request.ProviderShipmentID != "" ||
			request.RiderReference != "") {
		return ErrInvalidInput
	}

	if request.ProviderShipmentID != "" &&
		request.ProviderCode == "" {
		return ErrInvalidInput
	}

	if request.TrackingURL != "" {
		parsed, err :=
			url.ParseRequestURI(
				request.TrackingURL,
			)

		if err != nil ||
			parsed.Host == "" ||
			(parsed.Scheme != "http" &&
				parsed.Scheme != "https") {
			return ErrInvalidInput
		}
	}

	return nil
}

func normalizeTrackingEventRequest(
	request *TrackingEventRequest,
) {
	request.Source = strings.ToLower(
		strings.TrimSpace(
			request.Source,
		),
	)

	request.EventCode = strings.ToLower(
		strings.TrimSpace(
			request.EventCode,
		),
	)

	request.Status = strings.TrimSpace(
		request.Status,
	)

	request.Message = strings.TrimSpace(
		request.Message,
	)

	request.ExternalEventID = strings.TrimSpace(
		request.ExternalEventID,
	)
}

func validateTrackingEventRequest(
	request TrackingEventRequest,
) error {
	if !validTrackingSource(
		request.Source,
	) ||
		request.EventCode == "" ||
		utf8.RuneCountInString(
			request.EventCode,
		) > 80 ||
		utf8.RuneCountInString(
			request.Status,
		) > 80 ||
		utf8.RuneCountInString(
			request.Message,
		) > 1000 ||
		utf8.RuneCountInString(
			request.ExternalEventID,
		) > 160 {
		return ErrInvalidInput
	}

	if request.Latitude != nil &&
		(*request.Latitude < -90 ||
			*request.Latitude > 90) {
		return ErrInvalidInput
	}

	if request.Longitude != nil &&
		(*request.Longitude < -180 ||
			*request.Longitude > 180) {
		return ErrInvalidInput
	}

	return nil
}

func (s *Service) GetTracking(
	ctx context.Context,
	orderID string,
) (TrackingDetail, error) {
	orderID = strings.TrimSpace(
		orderID,
	)

	if !uuidPattern.MatchString(
		orderID,
	) {
		return TrackingDetail{},
			ErrOrderNotFound
	}

	shipment, err :=
		s.repository.GetShipmentByOrder(
			ctx,
			orderID,
		)
	if err != nil {
		return TrackingDetail{}, err
	}

	events, err :=
		s.repository.ListTrackingEvents(
			ctx,
			shipment.ID,
		)
	if err != nil {
		return TrackingDetail{}, err
	}

	return TrackingDetail{
		Shipment: shipment,
		Events:   events,
	}, nil
}

func (s *Service) GetTrackingForCustomer(
	ctx context.Context,
	customerID string,
	orderID string,
) (TrackingDetail, error) {
	customerID = strings.TrimSpace(
		customerID,
	)

	orderID = strings.TrimSpace(
		orderID,
	)

	if customerID == "" {
		return TrackingDetail{},
			ErrCustomerAuthenticationRequired
	}

	if !uuidPattern.MatchString(
		customerID,
	) ||
		!uuidPattern.MatchString(
			orderID,
		) {
		return TrackingDetail{},
			ErrOrderNotFound
	}

	order, err :=
		s.repository.GetOrderState(
			ctx,
			orderID,
		)
	if err != nil {
		return TrackingDetail{}, err
	}

	if order.CustomerID == "" ||
		order.CustomerID != customerID {
		return TrackingDetail{},
			ErrCustomerOrderMismatch
	}

	return s.GetTracking(
		ctx,
		orderID,
	)
}
