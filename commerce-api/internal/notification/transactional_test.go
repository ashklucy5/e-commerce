package notification

import "testing"

func TestBuildCustomerEventRequests(
	t *testing.T,
) {
	items, err :=
		buildCustomerEventRequests(
			CustomerEventRequest{
				BaseDedupeKey: "order:11111111-1111-4111-8111-111111111111:placed",

				Category: CategoryOrder,

				EventType: EventOrderPlaced,

				CustomerID: "22222222-2222-4222-8222-222222222222",

				OrderID: "11111111-1111-4111-8111-111111111111",

				Phone: "+8801700000000",

				Email: "customer@example.com",

				TemplateKey: TemplateOrderPlaced,

				Message: "We received your order.",

				Payload: map[string]any{
					"order_number": "ORD-TEST",
				},
			},
		)
	if err != nil {
		t.Fatalf(
			"build requests: %v",
			err,
		)
	}

	if len(items) != 2 {
		t.Fatalf(
			"expected 2 channel rows, got %d",
			len(items),
		)
	}

	if items[0].Channel !=
		ChannelSMS {

		t.Fatalf(
			"expected SMS first, got %q",
			items[0].Channel,
		)
	}

	if items[1].Channel !=
		ChannelEmail {

		t.Fatalf(
			"expected email second, got %q",
			items[1].Channel,
		)
	}

	if items[0].DedupeKey !=
		"order:11111111-1111-4111-8111-111111111111:placed:sms" {

		t.Fatalf(
			"unexpected SMS dedupe key %q",
			items[0].DedupeKey,
		)
	}

	if items[1].DedupeKey !=
		"order:11111111-1111-4111-8111-111111111111:placed:email" {

		t.Fatalf(
			"unexpected email dedupe key %q",
			items[1].DedupeKey,
		)
	}

	if items[0].Payload["message"] !=
		"We received your order." {

		t.Fatalf(
			"unexpected message %#v",
			items[0].Payload["message"],
		)
	}
}

func TestBuildCustomerEventRequestsOverridesPayloadMessage(
	t *testing.T,
) {
	items, err :=
		buildCustomerEventRequests(
			CustomerEventRequest{
				BaseDedupeKey: "order:test:processing",

				Category: CategoryOrder,

				EventType: EventOrderProcessing,

				TemplateKey: TemplateOrderProcessing,

				Message: "Safe authoritative message",

				Payload: map[string]any{
					"message": "caller should not override this",
				},
			},
		)
	if err != nil {
		t.Fatalf(
			"build requests: %v",
			err,
		)
	}

	if items[0].Payload["message"] !=
		"Safe authoritative message" {

		t.Fatalf(
			"payload overrode authoritative message",
		)
	}
}
