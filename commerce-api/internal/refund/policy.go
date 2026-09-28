package refund

import (
	"math/big"
	"strings"
)

func providerForPaymentMethod(
	paymentMethod string,
) (string, bool) {
	switch strings.ToLower(
		strings.TrimSpace(
			paymentMethod,
		),
	) {
	case "cod":
		return ProviderManual, true

	case ProviderBKash:
		return ProviderBKash, true

	case ProviderNagad:
		return ProviderNagad, true

	case ProviderRocket:
		return ProviderRocket, true

	case ProviderBankTransfer:
		return ProviderBankTransfer, true

	default:
		return "", false
	}
}

func paymentStateEligibleForRefund(
	paymentMethod string,
	paymentStatus string,
) bool {
	switch paymentMethod {
	case "cod":
		return paymentStatus == "cod_collected"

	case ProviderBKash,
		ProviderNagad,
		ProviderRocket,
		ProviderBankTransfer:
		return paymentStatus == "paid"

	default:
		return false
	}
}

func calculateReturnRefundAmount(
	grossReturnAmount int64,
	subtotalAmount int64,
	discountAmount int64,
) (int64, error) {
	if grossReturnAmount <= 0 ||
		subtotalAmount <= 0 ||
		discountAmount < 0 ||
		discountAmount > subtotalAmount {
		return 0,
			ErrNothingToRefund
	}

	netMerchandise :=
		subtotalAmount -
			discountAmount

	if netMerchandise <= 0 {
		return 0,
			ErrNothingToRefund
	}

	// grossReturnAmount * netMerchandise / subtotalAmount,
	// using big.Int so money arithmetic cannot overflow int64.
	numerator :=
		new(big.Int).Mul(
			big.NewInt(
				grossReturnAmount,
			),
			big.NewInt(
				netMerchandise,
			),
		)

	result :=
		new(big.Int).Quo(
			numerator,
			big.NewInt(
				subtotalAmount,
			),
		)

	if !result.IsInt64() {
		return 0,
			ErrRefundAmountExceeded
	}

	amount :=
		result.Int64()

	if amount <= 0 {
		return 0,
			ErrNothingToRefund
	}

	return amount, nil
}
