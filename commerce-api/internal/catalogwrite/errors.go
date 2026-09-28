package catalogwrite

import "errors"

var (
	ErrProductCategoryNotFound = errors.New(
		"product category not found",
	)

	ErrProductCategoryInactive = errors.New(
		"product category is inactive",
	)
)

type ProductValidationError struct {
	message string
}

func newProductValidationError(
	err error,
) *ProductValidationError {
	if err == nil {
		return nil
	}

	return &ProductValidationError{
		message: err.Error(),
	}
}

func (e *ProductValidationError) Error() string {
	if e == nil {
		return ""
	}

	return e.message
}
