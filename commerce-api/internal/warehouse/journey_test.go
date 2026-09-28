package warehouse

import "testing"

func TestJourneyInboundTransitionIncludesChinaPacking(
	t *testing.T,
) {
	if !allowedJourneyInboundTransition(
		InboundStatusSupplierReady,
		InboundStatusPackedInChina,
	) {
		t.Fatal(
			"supplier_ready -> packed_in_china should be allowed",
		)
	}

	if allowedJourneyInboundTransition(
		InboundStatusSupplierReady,
		InboundStatusPickedUpInChina,
	) {
		t.Fatal(
			"supplier_ready must not skip packed_in_china",
		)
	}

	if !allowedJourneyInboundTransition(
		InboundStatusPackedInChina,
		InboundStatusPickedUpInChina,
	) {
		t.Fatal(
			"packed_in_china -> picked_up_in_china should be allowed",
		)
	}
}

func TestJourneyCoordinatePairValidation(
	t *testing.T,
) {
	latitude := 22.5431
	longitude := 114.0579

	if !validJourneyCoordinatePair(
		&latitude,
		&longitude,
	) {
		t.Fatal(
			"valid coordinate pair rejected",
		)
	}

	if validJourneyCoordinatePair(
		&latitude,
		nil,
	) {
		t.Fatal(
			"partial coordinate pair accepted",
		)
	}
}
