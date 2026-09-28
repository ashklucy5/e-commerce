package warehouse

import "testing"

func TestInboundNotificationStage(
	t *testing.T,
) {
	tests :=
		[]struct {
			status string

			stage string

			notify bool
		}{
			{
				status: InboundStatusPackedInChina,

				stage: "packed_in_china",

				notify: true,
			},
			{
				status: InboundStatusDepartedChina,

				stage: "international_transit",

				notify: true,
			},
			{
				status: InboundStatusInInternationalTransit,

				stage: "international_transit",

				notify: true,
			},
			{
				status: InboundStatusArrivedBangladesh,

				stage: "arrived_bangladesh",

				notify: true,
			},
			{
				status: InboundStatusReceivedAtWarehouse,

				stage: "warehouse_received",

				notify: true,
			},

			// Operational/internal progression should not
			// generate extra SMS/email noise.
			{
				status: InboundStatusSupplierReady,

				notify: false,
			},
			{
				status: InboundStatusPickedUpInChina,

				notify: false,
			},
			{
				status: InboundStatusCustomsProcessing,

				notify: false,
			},
			{
				status: InboundStatusCustomsReleased,

				notify: false,
			},
		}

	for _, test := range tests {

		stage, notify :=
			inboundNotificationStage(
				test.status,
			)

		if notify !=
			test.notify {

			t.Fatalf(
				"status %q: expected notify=%v, got %v",
				test.status,
				test.notify,
				notify,
			)
		}

		if stage !=
			test.stage {

			t.Fatalf(
				"status %q: expected stage %q, got %q",
				test.status,
				test.stage,
				stage,
			)
		}
	}
}
