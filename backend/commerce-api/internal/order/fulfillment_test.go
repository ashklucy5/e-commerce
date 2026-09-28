package order

import (
	"errors"
	"testing"
)

func TestExpectedFulfillmentSource(
	t *testing.T,
) {
	tests :=
		[]struct {
			target   string
			expected string
			valid    bool
		}{
			{
				target:   StatusConfirmed,
				expected: StatusAwaitingProcurement,
				valid:    true,
			},
			{
				target:   StatusProcessing,
				expected: StatusConfirmed,
				valid:    true,
			},
			{
				target: StatusShipped,
				valid:  false,
			},
			{
				target: StatusDelivered,
				valid:  false,
			},
			{
				target:   StatusCompleted,
				expected: StatusDelivered,
				valid:    true,
			},
			{
				target: StatusCancelled,
				valid:  false,
			},
		}

	for _, test := range tests {
		actual, valid :=
			expectedFulfillmentSource(
				test.target,
			)

		if valid !=
			test.valid {
			t.Fatalf(
				"target %q: expected valid=%v, got %v",
				test.target,
				test.valid,
				valid,
			)
		}

		if actual !=
			test.expected {
			t.Fatalf(
				"target %q: expected source %q, got %q",
				test.target,
				test.expected,
				actual,
			)
		}
	}
}

func TestValidateFulfillmentPaymentCOD(
	t *testing.T,
) {
	order :=
		Order{
			PaymentMethod: PaymentMethodCOD,
			PaymentStatus: PaymentStatusCODPending,
		}

	for _, target := range []string{
		StatusConfirmed,
		StatusProcessing,
	} {
		if err :=
			validateFulfillmentPayment(
				order,
				target,
			); err != nil {
			t.Fatalf(
				"expected COD order %s to be allowed: %v",
				target,
				err,
			)
		}
	}

	err :=
		validateFulfillmentPayment(
			order,
			StatusCompleted,
		)

	if !errors.Is(
		err,
		ErrPaymentNotReadyForFulfillment,
	) {
		t.Fatalf(
			"expected COD completion to require collection, got %v",
			err,
		)
	}

	order.PaymentStatus =
		PaymentStatusCODCollected

	if err :=
		validateFulfillmentPayment(
			order,
			StatusCompleted,
		); err != nil {
		t.Fatalf(
			"expected collected COD to complete: %v",
			err,
		)
	}
}

func TestValidateFulfillmentPaymentOnline(
	t *testing.T,
) {
	order :=
		Order{
			PaymentMethod: PaymentMethodBKash,
			PaymentStatus: PaymentStatusPaid,
		}

	for _, target := range []string{
		StatusConfirmed,
		StatusProcessing,
		StatusCompleted,
	} {
		if err :=
			validateFulfillmentPayment(
				order,
				target,
			); err != nil {
			t.Fatalf(
				"expected paid online order %s to be allowed: %v",
				target,
				err,
			)
		}
	}

	order.PaymentStatus =
		PaymentStatusPending

	err :=
		validateFulfillmentPayment(
			order,
			StatusProcessing,
		)

	if !errors.Is(
		err,
		ErrPaymentNotReadyForFulfillment,
	) {
		t.Fatalf(
			"expected unpaid online order to be rejected, got %v",
			err,
		)
	}
}
