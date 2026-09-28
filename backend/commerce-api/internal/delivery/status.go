package delivery

import "strings"

func validDeliveryMode(
	value string,
) bool {
	switch value {
	case DeliveryModeCourier,
		DeliveryModeSelfPickup,
		DeliveryModeCommunityRider:
		return true

	default:
		return false
	}
}

func validTrackingSource(
	value string,
) bool {
	switch value {
	case "manual",
		"provider",
		"rider",
		"customer",
		"support",
		"admin",
		"system":
		return true

	default:
		return false
	}
}

func paymentReadyForDelivery(
	order orderState,
) bool {
	if order.PaymentMethod == "cod" {
		return order.PaymentStatus == "cod_pending"
	}

	return order.PaymentStatus == "paid"
}

func normalizeActorID(
	value string,
) string {
	value = strings.TrimSpace(
		value,
	)

	if value == "" {
		return "development-admin"
	}

	return value
}
