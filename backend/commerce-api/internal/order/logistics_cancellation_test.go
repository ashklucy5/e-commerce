package order

import "testing"

func TestShipmentStatusAllowsOrderCancellation(
	t *testing.T,
) {
	tests :=
		[]struct {
			status  string
			allowed bool
		}{
			{
				status:  "pending",
				allowed: true,
			},
			{
				status:  "cancelled",
				allowed: true,
			},
			{
				status:  "shipped",
				allowed: false,
			},
			{
				status:  "awaiting_confirmation",
				allowed: false,
			},
			{
				status:  "delivered",
				allowed: false,
			},
		}

	for _, test := range tests {

		actual :=
			shipmentStatusAllowsOrderCancellation(
				test.status,
			)

		if actual !=
			test.allowed {
			t.Fatalf(
				"status %q: expected allowed=%v, got %v",
				test.status,
				test.allowed,
				actual,
			)
		}
	}
}

func TestWarehouseFulfillmentStatusAllowsOrderCancellation(
	t *testing.T,
) {
	for _, status := range []string{
		"allocated",
		"waiting_inbound",
		"received",
		"picking",
		"packed",
		"ready_for_handoff",
	} {

		if !warehouseFulfillmentStatusAllowsOrderCancellation(
			status,
		) {
			t.Fatalf(
				"expected warehouse status %q to be cancellable",
				status,
			)
		}
	}

	for _, status := range []string{
		"handed_off",
		"cancelled",
		"unknown",
	} {

		if warehouseFulfillmentStatusAllowsOrderCancellation(
			status,
		) {
			t.Fatalf(
				"expected warehouse status %q not to be cancellable",
				status,
			)
		}
	}
}
