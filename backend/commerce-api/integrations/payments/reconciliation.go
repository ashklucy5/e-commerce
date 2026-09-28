package payments

import "context"

// MerchantReferenceQuerier is optional.
//
// Some providers can recover a payment created during an uncertain
// CreatePayment outcome by looking it up through the merchant-side
// idempotency/reference value.
//
// Providers that do not support this capability do not need to
// implement this interface.
type MerchantReferenceQuerier interface {
	QueryPaymentByMerchantReference(
		ctx context.Context,
		merchantReference string,
	) (QueryPaymentResult, error)
}
