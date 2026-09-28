package customer

import "errors"

var (
	ErrCustomerNotFound = errors.New(
		"customer not found",
	)

	ErrAddressNotFound = errors.New(
		"customer address not found",
	)

	ErrInvalidPhone = errors.New(
		"invalid phone number",
	)

	ErrInvalidEmail = errors.New(
		"invalid email address",
	)

	ErrInvalidFullName = errors.New(
		"invalid full name",
	)

	ErrInvalidAddress = errors.New(
		"invalid customer address",
	)

	ErrPhoneInUse = errors.New(
		"phone number is already registered",
	)

	ErrEmailInUse = errors.New(
		"email address is already registered",
	)

	ErrNoChanges = errors.New(
		"no changes supplied",
	)
)
