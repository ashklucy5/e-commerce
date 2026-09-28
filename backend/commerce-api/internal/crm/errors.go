package crm

import "errors"

var (
	ErrInvalidRequest  = errors.New("invalid CRM request")
	ErrInvalidCaseType = errors.New("invalid CRM case type")
	ErrCaseNotFound    = errors.New("CRM case not found")
	ErrCaseClosed      = errors.New("CRM case is closed")

	ErrProductNotFound = errors.New("CRM product not found")
	ErrVariantNotFound = errors.New("CRM variant not found")
	ErrOrderNotFound   = errors.New("CRM order not found")

	ErrContextMismatch = errors.New(
		"CRM product and variant context do not match",
	)

	ErrInvalidRequestedQuantity = errors.New(
		"invalid requested quantity",
	)

	ErrBulkStockAvailable = errors.New(
		"requested quantity does not exceed currently available stock",
	)
)
