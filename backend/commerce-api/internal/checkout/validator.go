package checkout

import (
	"fmt"
	"strings"
	"time"
)

func normalizeCartKey(
	value string,
) (string, error) {
	value = strings.TrimSpace(
		value,
	)

	if value == "" ||
		len(value) > 160 {
		return "",
			ErrInvalidCartKey
	}

	return value, nil
}

func normalizeCheckoutKey(
	value string,
) (string, error) {
	value = strings.TrimSpace(
		value,
	)

	if value == "" ||
		len(value) > 160 {
		return "",
			ErrInvalidCheckoutKey
	}

	return value, nil
}

func validateCartSnapshot(
	snapshot cartSnapshot,
	now time.Time,
) error {
	if snapshot.Status != "active" {
		return ErrCartInactive
	}

	if snapshot.ExpiresAt != nil &&
		!snapshot.ExpiresAt.After(now) {
		return ErrCartExpired
	}

	if len(snapshot.Items) == 0 {
		return ErrEmptyCart
	}

	for _, item := range snapshot.Items {
		if !item.VariantActive ||
			!item.ProductActive {
			return fmt.Errorf(
				"%w: SKU %s",
				ErrCartItemUnavailable,
				item.SKU,
			)
		}

		if !strings.EqualFold(
			snapshot.Currency,
			item.Currency,
		) {
			return fmt.Errorf(
				"%w: SKU %s",
				ErrCurrencyMismatch,
				item.SKU,
			)
		}

		if item.MinimumOrderQuantity <= 0 {
			return fmt.Errorf(
				"%w: invalid MOQ for SKU %s",
				ErrCartItemUnavailable,
				item.SKU,
			)
		}

		if item.Quantity <
			item.MinimumOrderQuantity {
			return fmt.Errorf(
				"%w: SKU %s requires at least %d",
				ErrBelowMinimumOrderQuantity,
				item.SKU,
				item.MinimumOrderQuantity,
			)
		}

		if item.Quantity >
			item.AvailableQuantity {
			return fmt.Errorf(
				"%w: SKU %s requested %d, available %d",
				ErrInsufficientStock,
				item.SKU,
				item.Quantity,
				item.AvailableQuantity,
			)
		}
	}

	return nil
}

func normalizeCustomerValue(
	value string,
	maxLength int,
) (string, error) {
	value = strings.TrimSpace(
		value,
	)

	if len(value) > maxLength {
		return "",
			ErrInvalidCustomerDetails
	}

	return value, nil
}

func normalizeShippingValue(
	value string,
	maxLength int,
) (string, error) {
	value = strings.TrimSpace(
		value,
	)

	if len(value) > maxLength {
		return "",
			ErrInvalidShippingDetails
	}

	return value, nil
}
