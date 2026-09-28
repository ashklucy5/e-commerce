package cart

import (
	"math"
)

func calculateLineTotal(
	unitPrice int64,
	quantity int,
) (int64, error) {
	if unitPrice < 0 ||
		quantity < 0 {
		return 0,
			ErrMoneyOverflow
	}

	if quantity == 0 {
		return 0, nil
	}

	if unitPrice >
		math.MaxInt64/int64(quantity) {
		return 0,
			ErrMoneyOverflow
	}

	return unitPrice *
			int64(quantity),
		nil
}

func calculateTotals(
	items []Item,
	currency string,
) (Totals, error) {
	result := Totals{
		Currency:  currency,
		ItemCount: len(items),
	}

	for index := range items {
		lineTotal, err :=
			calculateLineTotal(
				items[index].
					UnitPriceAmount,
				items[index].
					Quantity,
			)
		if err != nil {
			return Totals{},
				err
		}

		items[index].
			LineTotalAmount =
			lineTotal

		if result.SubtotalAmount >
			math.MaxInt64-lineTotal {
			return Totals{},
				ErrMoneyOverflow
		}

		result.SubtotalAmount +=
			lineTotal

		result.QuantityTotal +=
			items[index].
				Quantity
	}

	return result, nil
}
