package adminauth

import "errors"

var (
	ErrAdminAccountNotFound = errors.New(
		"Admin account not found",
	)

	ErrAdminAccountDisabled = errors.New(
		"Admin account is disabled",
	)

	ErrAdminPanelAccessRequired = errors.New(
		"Admin panel permission is required",
	)

	ErrMFAAlreadyEnrolled = errors.New(
		"Admin MFA is already enrolled",
	)

	ErrMFANotEnrolled = errors.New(
		"Admin MFA is not enrolled",
	)

	ErrMFAEnrollmentNotPending = errors.New(
		"Admin MFA enrollment is not pending",
	)

	ErrInvalidMFACode = errors.New(
		"invalid MFA code",
	)

	ErrMFAConfiguration = errors.New(
		"invalid Admin MFA configuration",
	)
)
