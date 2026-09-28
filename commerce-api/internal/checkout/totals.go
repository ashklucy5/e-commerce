package checkout

import "math"

func buildCheckoutItems(
	snapshot cartSnapshot,
) ([]Item, int64, int, error) {
	items := make(
		[]Item,
		0,
		len(snapshot.Items),
	)

	var subtotal int64
	var quantityTotal int

	for _, source := range snapshot.Items {
		if source.UnitPriceAmount < 0 ||
			source.Quantity <= 0 {
			return nil,
				0,
				0,
				ErrMoneyOverflow
		}

		if source.UnitPriceAmount >
			math.MaxInt64/int64(source.Quantity) {
			return nil,
				0,
				0,
				ErrMoneyOverflow
		}

		lineTotal :=
			source.UnitPriceAmount *
				int64(source.Quantity)

		if subtotal >
			math.MaxInt64-lineTotal {
			return nil,
				0,
				0,
				ErrMoneyOverflow
		}

		subtotal += lineTotal

		quantityTotal +=
			source.Quantity

		items = append(
			items,
			Item{
				VariantID: source.VariantID,

				SKU: source.SKU,

				ProductName: source.ProductName,

				Quantity: source.Quantity,

				MinimumOrderQuantity: source.MinimumOrderQuantity,

				UnitPriceAmount: source.UnitPriceAmount,

				LineTotalAmount: lineTotal,

				Currency: source.Currency,
			},
		)
	}

	return items,
		subtotal,
		quantityTotal,
		nil
}
