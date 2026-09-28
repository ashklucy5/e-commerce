package catalogimport

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

func parseMoneyToPoisha(value string) (int64, error) {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, ",", "")
	value = strings.ReplaceAll(value, "৳", "")
	value = strings.TrimSpace(value)

	if value == "" {
		return 0, fmt.Errorf("money value is empty")
	}

	negative := false

	switch value[0] {
	case '-':
		negative = true
		value = value[1:]
	case '+':
		value = value[1:]
	}

	if value == "" {
		return 0, fmt.Errorf("invalid money value")
	}

	parts := strings.Split(value, ".")

	if len(parts) > 2 {
		return 0, fmt.Errorf("invalid decimal value")
	}

	wholeText := parts[0]
	if wholeText == "" {
		wholeText = "0"
	}

	whole, err := strconv.ParseInt(
		wholeText,
		10,
		64,
	)
	if err != nil {
		return 0, fmt.Errorf(
			"invalid whole amount: %w",
			err,
		)
	}

	var fractional int64

	if len(parts) == 2 {
		fractionText := parts[1]

		if len(fractionText) > 2 {
			return 0, fmt.Errorf(
				"money may have at most 2 decimal places",
			)
		}

		switch len(fractionText) {
		case 0:
			fractional = 0

		case 1:
			fractional, err = strconv.ParseInt(
				fractionText+"0",
				10,
				64,
			)

		case 2:
			fractional, err = strconv.ParseInt(
				fractionText,
				10,
				64,
			)
		}

		if err != nil {
			return 0, fmt.Errorf(
				"invalid decimal amount: %w",
				err,
			)
		}
	}

	if whole > (math.MaxInt64-fractional)/100 {
		return 0, fmt.Errorf(
			"money value is too large",
		)
	}

	amount := whole*100 + fractional

	if negative {
		amount = -amount
	}

	return amount, nil
}
