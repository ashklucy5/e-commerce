package delivery

import "testing"

func TestValidDeliveryMode(
	t *testing.T,
) {
	for _, value := range []string{
		DeliveryModeCourier,
		DeliveryModeSelfPickup,
		DeliveryModeCommunityRider,
	} {
		if !validDeliveryMode(
			value,
		) {
			t.Fatalf(
				"expected %q to be a valid delivery mode",
				value,
			)
		}
	}

	if validDeliveryMode(
		"drone",
	) {
		t.Fatal(
			"unexpected delivery mode accepted",
		)
	}
}

func TestValidatePrepareShipmentRequest(
	t *testing.T,
) {
	tests :=
		[]struct {
			name    string
			request PrepareShipmentRequest
			valid   bool
		}{
			{
				name: "manual courier",
				request: PrepareShipmentRequest{
					DeliveryMode: DeliveryModeCourier,
					CourierName:  "Development Courier",
				},
				valid: true,
			},
			{
				name: "provider courier",
				request: PrepareShipmentRequest{
					DeliveryMode: DeliveryModeCourier,
					ProviderCode: "future_provider",
				},
				valid: true,
			},
			{
				name: "self pickup",
				request: PrepareShipmentRequest{
					DeliveryMode: DeliveryModeSelfPickup,
				},
				valid: true,
			},
			{
				name: "rider requires reference",
				request: PrepareShipmentRequest{
					DeliveryMode: DeliveryModeCommunityRider,
				},
				valid: false,
			},
			{
				name: "provider shipment requires provider",
				request: PrepareShipmentRequest{
					DeliveryMode: DeliveryModeCourier,
					CourierName:  "Development Courier",

					ProviderShipmentID: "external-1",
				},
				valid: false,
			},
		}

	for _, test := range tests {
		t.Run(
			test.name,
			func(
				t *testing.T,
			) {
				request :=
					test.request

				normalizePrepareShipmentRequest(
					&request,
				)

				err :=
					validatePrepareShipmentRequest(
						request,
					)

				if test.valid &&
					err != nil {
					t.Fatalf(
						"expected valid request: %v",
						err,
					)
				}

				if !test.valid &&
					err == nil {
					t.Fatal(
						"expected request validation failure",
					)
				}
			},
		)
	}
}

func TestPaymentReadyForDelivery(
	t *testing.T,
) {
	if !paymentReadyForDelivery(
		orderState{
			PaymentMethod: "cod",
			PaymentStatus: "cod_pending",
		},
	) {
		t.Fatal(
			"expected pending COD to be delivery-ready",
		)
	}

	if !paymentReadyForDelivery(
		orderState{
			PaymentMethod: "bkash",
			PaymentStatus: "paid",
		},
	) {
		t.Fatal(
			"expected paid online order to be delivery-ready",
		)
	}

	if paymentReadyForDelivery(
		orderState{
			PaymentMethod: "bkash",
			PaymentStatus: "pending",
		},
	) {
		t.Fatal(
			"unexpected unpaid online order accepted",
		)
	}
}
