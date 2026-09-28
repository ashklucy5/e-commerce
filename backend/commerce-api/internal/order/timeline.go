package order

import "context"

type Timeline struct {
	OrderID     string  `json:"order_id"`
	OrderNumber string  `json:"order_number"`
	Status      string  `json:"status"`
	Events      []Event `json:"events"`
}

func (s *Service) Timeline(
	ctx context.Context,
	orderID string,
) (Timeline, error) {
	order, err :=
		s.Get(
			ctx,
			orderID,
		)
	if err != nil {
		return Timeline{}, err
	}

	return s.timelineForOrder(
		ctx,
		order,
	)
}

func (s *Service) TimelineForCustomer(
	ctx context.Context,
	customerID string,
	orderID string,
) (Timeline, error) {
	order, err :=
		s.GetForCustomer(
			ctx,
			customerID,
			orderID,
		)
	if err != nil {
		return Timeline{}, err
	}

	return s.timelineForOrder(
		ctx,
		order,
	)
}

func (s *Service) timelineForOrder(
	ctx context.Context,
	order Order,
) (Timeline, error) {
	events, err :=
		s.repository.ListEvents(
			ctx,
			order.ID,
		)
	if err != nil {
		return Timeline{}, err
	}

	return Timeline{
		OrderID:     order.ID,
		OrderNumber: order.OrderNumber,
		Status:      order.Status,
		Events:      events,
	}, nil
}
