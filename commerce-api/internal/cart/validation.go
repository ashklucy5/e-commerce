package cart

import (
	"encoding/hex"
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

func validateUUID(
	value string,
	invalidError error,
) (string, error) {
	value = strings.TrimSpace(
		value,
	)

	if len(value) != 36 {
		return "",
			invalidError
	}

	if value[8] != '-' ||
		value[13] != '-' ||
		value[18] != '-' ||
		value[23] != '-' {
		return "",
			invalidError
	}

	compact := strings.ReplaceAll(
		value,
		"-",
		"",
	)

	if len(compact) != 32 {
		return "",
			invalidError
	}

	if _, err := hex.DecodeString(
		compact,
	); err != nil {
		return "",
			invalidError
	}

	return value, nil
}

func validateCartWritable(
	cart Cart,
	now time.Time,
) error {
	if cart.Status != "active" {
		return ErrCartInactive
	}

	if cart.ExpiresAt != nil &&
		!cart.ExpiresAt.After(now) {
		return ErrCartExpired
	}

	return nil
}

func validateRequestedQuantity(
	quantity int,
	variant purchasableVariant,
) error {
	if quantity <= 0 {
		return ErrInvalidQuantity
	}

	// The seller/admin controls the MOQ for each variant.
	//
	// Example:
	//   Machine       -> MOQ 1
	//   Belt          -> MOQ 5
	//   Phone case    -> MOQ 10
	//
	// Once the MOQ is satisfied, the customer may choose
	// any whole-number quantity above it. There is no
	// order-increment restriction.
	if quantity <
		variant.MinimumOrderQuantity {
		return fmt.Errorf(
			"%w: minimum quantity is %d",
			ErrBelowMinimumOrderQuantity,
			variant.MinimumOrderQuantity,
		)
	}

	// Available inventory is the effective upper limit.
	// There is no seller-configured maximum order quantity.
	if quantity >
		variant.AvailableQuantity {
		return fmt.Errorf(
			"%w: requested %d, available %d",
			ErrInsufficientStock,
			quantity,
			variant.AvailableQuantity,
		)
	}

	return nil
}

func quantityMatchesRules(
	quantity int,
	minimum int,
) bool {
	if quantity <= 0 {
		return false
	}

	if minimum <= 0 {
		return false
	}

	return quantity >= minimum
}
