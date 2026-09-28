package order

import (
	checkoutpolicy "project.local/commerce-api/internal/checkout"
)

func (s *Service) SetDeliveryMethods(
	config checkoutpolicy.DeliveryMethodConfig,
) error {
	if err :=
		config.Validate(); err != nil {

		return err
	}

	s.deliveryMethods =
		config

	return nil
}

func (s *Service) validateCheckoutDelivery(
	checkout checkoutSnapshot,
) error {
	quote, err :=
		s.deliveryMethods.Quote(
			checkout.DeliveryMethod,
			checkout.Currency,
		)
	if err != nil {
		return ErrCheckoutChanged
	}

	if quote.ShippingAmount !=
		checkout.ShippingAmount {

		return ErrCheckoutChanged
	}

	return nil
}
