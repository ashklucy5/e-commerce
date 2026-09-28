package delivery

import (
	"testing"

	"project.local/commerce-api/internal/notification"
)

func TestDeliveryNotificationSpec(
	t *testing.T,
) {
	shipped, ok :=
		deliveryNotificationSpec(
			"order_shipped",
		)

	if !ok {
		t.Fatal(
			"order_shipped should generate a notification",
		)
	}

	if shipped.EventType !=
		notification.EventDeliverySent {

		t.Fatalf(
			"unexpected shipped event %q",
			shipped.EventType,
		)
	}

	if shipped.TemplateKey !=
		notification.TemplateDeliveryDispatched {

		t.Fatalf(
			"unexpected shipped template %q",
			shipped.TemplateKey,
		)
	}

	delivered, ok :=
		deliveryNotificationSpec(
			"order_delivered",
		)

	if !ok {
		t.Fatal(
			"order_delivered should generate a notification",
		)
	}

	if delivered.EventType !=
		notification.EventDeliveryComplete {

		t.Fatalf(
			"unexpected delivered event %q",
			delivered.EventType,
		)
	}

	if delivered.TemplateKey !=
		notification.TemplateDeliveryDelivered {

		t.Fatalf(
			"unexpected delivered template %q",
			delivered.TemplateKey,
		)
	}

	/*
		Courier/provider delivery is deliberately not the final
		customer delivery notification.
	*/
	if _, ok :=
		deliveryNotificationSpec(
			"provider_delivered",
		); ok {

		t.Fatal(
			"provider_delivered must not generate final delivery notification",
		)
	}
}
