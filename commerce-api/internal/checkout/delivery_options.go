package checkout

import (
	"context"
	"math"
)

func (s *Service) SetDeliveryMethods(
	config DeliveryMethodConfig,
) error {
	if err :=
		config.Validate(); err != nil {

		return err
	}

	s.deliveryMethods =
		config

	return nil
}

func (s *Service) DeliveryOptionsForCustomer(
	ctx context.Context,
	customerID string,
	checkoutKey string,
) ([]DeliveryOption, error) {
	session, err :=
		s.GetForCustomer(
			ctx,
			customerID,
			checkoutKey,
		)
	if err != nil {
		return nil, err
	}

	return s.deliveryMethods.Options(
		session.Currency,
		session.DeliveryMethod,
	)
}

func (s *Service) applyDeliveryMethod(
	session *Session,
) error {
	if session.DeliveryMethod == "" {
		session.ShippingAmount =
			0

		return recalculateCheckoutTotal(
			session,
		)
	}

	quote, err :=
		s.deliveryMethods.Quote(
			session.DeliveryMethod,
			session.Currency,
		)
	if err != nil {
		return err
	}

	session.ShippingAmount =
		quote.ShippingAmount

	return recalculateCheckoutTotal(
		session,
	)
}

func recalculateCheckoutTotal(
	session *Session,
) error {
	if session.SubtotalAmount < 0 ||
		session.DiscountAmount < 0 ||
		session.DiscountAmount >
			session.SubtotalAmount ||
		session.ShippingAmount < 0 {

		return ErrMoneyOverflow
	}

	base :=
		session.SubtotalAmount -
			session.DiscountAmount

	if base >
		math.MaxInt64-
			session.ShippingAmount {

		return ErrMoneyOverflow
	}

	session.TotalAmount =
		base +
			session.ShippingAmount

	return nil
}
