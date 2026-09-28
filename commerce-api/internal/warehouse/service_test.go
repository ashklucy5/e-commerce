package warehouse

import "testing"

func TestAllowedInboundTransition(
	t *testing.T,
) {
	tests := []struct {
		name string
		from string
		to   string
		want bool
	}{
		{
			name: "created to supplier ready",
			from: InboundStatusCreated,
			to:   InboundStatusSupplierReady,
			want: true,
		},
		{
			name: "cannot skip supplier ready",
			from: InboundStatusCreated,
			to:   InboundStatusDepartedChina,
			want: false,
		},
		{
			name: "customs released to received",
			from: InboundStatusCustomsReleased,
			to:   InboundStatusReceivedAtWarehouse,
			want: true,
		},
		{
			name: "can cancel before receipt",
			from: InboundStatusInInternationalTransit,
			to:   InboundStatusCancelled,
			want: true,
		},
		{
			name: "cannot cancel after receipt",
			from: InboundStatusReceivedAtWarehouse,
			to:   InboundStatusCancelled,
			want: false,
		},
	}

	for _, test := range tests {
		t.Run(
			test.name,
			func(t *testing.T) {
				got :=
					allowedInboundTransition(
						test.from,
						test.to,
					)

				if got !=
					test.want {
					t.Fatalf(
						"allowedInboundTransition(%q, %q) = %v, want %v",
						test.from,
						test.to,
						got,
						test.want,
					)
				}
			},
		)
	}
}

func TestAllowedManualFulfillmentTransition(
	t *testing.T,
) {
	tests := []struct {
		name string
		from string
		to   string
		want bool
	}{
		{
			name: "local allocated to picking",
			from: FulfillmentStatusAllocated,
			to:   FulfillmentStatusPicking,
			want: true,
		},
		{
			name: "received inbound to picking",
			from: FulfillmentStatusReceived,
			to:   FulfillmentStatusPicking,
			want: true,
		},
		{
			name: "picking to packed",
			from: FulfillmentStatusPicking,
			to:   FulfillmentStatusPacked,
			want: true,
		},
		{
			name: "packed to ready",
			from: FulfillmentStatusPacked,
			to:   FulfillmentStatusReadyForHandoff,
			want: true,
		},
		{
			name: "waiting inbound cannot be manually received",
			from: FulfillmentStatusWaitingInbound,
			to:   FulfillmentStatusReceived,
			want: false,
		},
		{
			name: "ready cannot be manually handed off",
			from: FulfillmentStatusReadyForHandoff,
			to:   FulfillmentStatusHandedOff,
			want: false,
		},
		{
			name: "waiting inbound can be cancelled",
			from: FulfillmentStatusWaitingInbound,
			to:   FulfillmentStatusCancelled,
			want: true,
		},
	}

	for _, test := range tests {
		t.Run(
			test.name,
			func(t *testing.T) {
				got :=
					allowedManualFulfillmentTransition(
						test.from,
						test.to,
					)

				if got !=
					test.want {
					t.Fatalf(
						"allowedManualFulfillmentTransition(%q, %q) = %v, want %v",
						test.from,
						test.to,
						got,
						test.want,
					)
				}
			},
		)
	}
}
