package delivery

import "testing"

func TestJourneyInboundStageMapping(
	t *testing.T,
) {
	tests :=
		[]struct {
			Status string
			Stage  string
			Rank   int
		}{
			{
				Status: "created",
				Stage:  JourneyStageInChina,
				Rank:   1,
			},
			{
				Status: "packed_in_china",
				Stage:  JourneyStagePackedInChina,
				Rank:   2,
			},
			{
				Status: "departed_china",
				Stage:  JourneyStageInternationalTransit,
				Rank:   3,
			},
			{
				Status: "arrived_bangladesh",
				Stage:  JourneyStageArrivedBangladesh,
				Rank:   4,
			},
			{
				Status: "received_at_warehouse",
				Stage:  JourneyStageWarehouseReceived,
				Rank:   5,
			},
		}

	for _, test := range tests {

		stage :=
			journeyStageForInboundStatus(
				test.Status,
			)

		if stage !=
			test.Stage {

			t.Fatalf(
				"status %q: expected stage %q, got %q",
				test.Status,
				test.Stage,
				stage,
			)
		}

		if journeyStageRank(
			stage,
		) != test.Rank {

			t.Fatalf(
				"stage %q: expected rank %d",
				stage,
				test.Rank,
			)
		}
	}
}

func TestProviderDeliveredIsNotFinalDelivery(
	t *testing.T,
) {
	stage :=
		journeyStageForDeliveryEvent(
			EventProviderDelivered,
			ShipmentStatusAwaitingConfirmation,
		)

	if stage !=
		JourneyStageDelivering {

		t.Fatalf(
			"provider delivered must remain delivering until receipt confirmation; got %q",
			stage,
		)
	}

	if journeyStageRank(
		stage,
	) >= 7 {

		t.Fatal(
			"provider-delivered incorrectly completed final delivery milestone",
		)
	}
}

func TestReceiptConfirmationCompletesDelivery(
	t *testing.T,
) {
	stage :=
		journeyStageForDeliveryEvent(
			EventReceiptConfirmed,
			ShipmentStatusDelivered,
		)

	if stage !=
		JourneyStageDelivered {

		t.Fatalf(
			"expected delivered, got %q",
			stage,
		)
	}

	if journeyStageRank(
		stage,
	) != 7 {

		t.Fatal(
			"receipt confirmation did not complete journey",
		)
	}
}

func TestDeliveryJourneyCoordinatePair(
	t *testing.T,
) {
	latitude :=
		23.8103

	longitude :=
		90.4125

	if !validDeliveryJourneyCoordinatePair(
		&latitude,
		&longitude,
	) {
		t.Fatal(
			"valid coordinate pair rejected",
		)
	}

	if validDeliveryJourneyCoordinatePair(
		&latitude,
		nil,
	) {
		t.Fatal(
			"partial coordinate pair accepted",
		)
	}
}
