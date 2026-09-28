package returns

import "strings"

func validateReceiveRequest(
	current Return,
	request *ReceiveRequest,
) error {
	if len(request.Items) !=
		len(current.Items) {
		return ErrInvalidReceivedQuantity
	}

	existing := make(
		map[string]Item,
		len(current.Items),
	)

	for _, item := range current.Items {
		existing[item.OrderItemID] =
			item
	}

	seen := make(
		map[string]struct{},
	)

	for index := range request.Items {
		item := &request.Items[index]

		item.OrderItemID =
			strings.TrimSpace(
				item.OrderItemID,
			)

		currentItem, ok :=
			existing[item.OrderItemID]
		if !ok {
			return ErrReturnItemNotFound
		}

		if _, exists :=
			seen[item.OrderItemID]; exists {
			return ErrDuplicateReturnItem
		}

		seen[item.OrderItemID] =
			struct{}{}

		if item.ReceivedQuantity < 0 ||
			item.ReceivedQuantity >
				currentItem.Quantity {
			return ErrInvalidReceivedQuantity
		}
	}

	return nil
}

func validateInspectRequest(
	current Return,
	request *InspectRequest,
) error {
	if len(request.Items) !=
		len(current.Items) {
		return ErrInvalidInspection
	}

	existing := make(
		map[string]Item,
		len(current.Items),
	)

	for _, item := range current.Items {
		existing[item.OrderItemID] =
			item
	}

	seen := make(
		map[string]struct{},
	)

	for index := range request.Items {
		item := &request.Items[index]

		item.OrderItemID =
			strings.TrimSpace(
				item.OrderItemID,
			)

		item.InspectionStatus =
			strings.ToLower(
				strings.TrimSpace(
					item.InspectionStatus,
				),
			)

		item.InspectionNote =
			strings.TrimSpace(
				item.InspectionNote,
			)

		currentItem, ok :=
			existing[item.OrderItemID]
		if !ok {
			return ErrReturnItemNotFound
		}

		if _, exists :=
			seen[item.OrderItemID]; exists {
			return ErrDuplicateReturnItem
		}

		seen[item.OrderItemID] =
			struct{}{}

		switch item.InspectionStatus {
		case InspectionRestockable:
			if item.RestockQuantity < 0 ||
				item.RestockQuantity >
					currentItem.ReceivedQuantity {
				return ErrInvalidInspection
			}

		case InspectionDamaged,
			InspectionNonRestockable:
			if item.RestockQuantity != 0 {
				return ErrInvalidInspection
			}

		default:
			return ErrInvalidInspection
		}

		if len(
			item.InspectionNote,
		) > 500 {
			return ErrInvalidInspection
		}
	}

	return nil
}
