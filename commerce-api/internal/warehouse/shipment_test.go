package warehouse

import "testing"

func TestNormalizeHandoffItemsRejectsDuplicates(
	t *testing.T,
) {
	id :=
		"11111111-1111-1111-1111-111111111111"

	_, ok :=
		normalizeHandoffItems(
			[]HandoffFulfillmentRequest{
				{
					FulfillmentID: id,
					Quantity:      1,
				},
				{
					FulfillmentID: id,
					Quantity:      1,
				},
			},
		)

	if ok {
		t.Fatal(
			"expected duplicate fulfillment IDs to be rejected",
		)
	}
}

func TestValidHandoffType(
	t *testing.T,
) {
	for _, value := range []string{
		HandoffTypeCourier,
		HandoffTypeSelfPickup,
		HandoffTypeCommunityRider,
	} {
		if !validHandoffType(
			value,
		) {
			t.Fatalf(
				"expected %q to be a valid handoff type",
				value,
			)
		}
	}

	if validHandoffType(
		"drone",
	) {
		t.Fatal(
			"unexpected handoff type accepted",
		)
	}
}
