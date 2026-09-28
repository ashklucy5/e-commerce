package auth

import (
	"context"
	"log"
)

type PasswordResetOTPSender interface {
	SendPasswordResetOTP(
		ctx context.Context,
		phone string,
		code string,
	) error
}

/*
The existing development registration sender now also
supports password-reset OTP delivery.

Later the Alpha/Tencent/etc. adapter only needs to
implement both OTP sender interfaces.
*/
func (
	*developmentRegistrationOTPSender,
) SendPasswordResetOTP(
	_ context.Context,
	phone string,
	code string,
) error {
	log.Printf(
		"[customer-auth][development] password-reset OTP phone=%s code=%s",
		phone,
		code,
	)

	return nil
}

func (
	*unavailableRegistrationOTPSender,
) SendPasswordResetOTP(
	_ context.Context,
	_ string,
	_ string,
) error {
	return ErrOTPDeliveryUnavailable
}
