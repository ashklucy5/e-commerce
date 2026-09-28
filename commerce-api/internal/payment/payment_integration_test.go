package payment

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"project.local/commerce-api/internal/inventory"
	orderpkg "project.local/commerce-api/internal/order"
	"project.local/commerce-api/internal/platform/config"
	"project.local/commerce-api/internal/platform/database"
)

type paymentIntegrationStock struct {
	OnHand    int
	Reserved  int
	Available int
}

func TestConfirmVerifiedPaymentIntegration(
	t *testing.T,
) {
	if os.Getenv(
		"RUN_INTEGRATION_TESTS",
	) != "1" {
		t.Skip(
			"set RUN_INTEGRATION_TESTS=1 to run PostgreSQL integration tests",
		)
	}

	orderID :=
		strings.TrimSpace(
			os.Getenv(
				"PAYMENT_TEST_ORDER_ID",
			),
		)

	if orderID == "" {
		t.Skip(
			"set PAYMENT_TEST_ORDER_ID to a fresh pending bKash/Nagad order",
		)
	}

	ctx, cancel :=
		context.WithTimeout(
			context.Background(),
			30*time.Second,
		)
	defer cancel()

	cfg, err :=
		config.Load()
	if err != nil {
		t.Fatalf(
			"load integration config: %v",
			err,
		)
	}

	pool, err :=
		database.NewPostgres(
			ctx,
			cfg,
		)
	if err != nil {
		t.Fatalf(
			"open integration postgres: %v",
			err,
		)
	}

	t.Cleanup(
		func() {
			pool.Close()
		},
	)

	inventoryRepository :=
		inventory.NewRepository(
			pool,
		)

	inventoryService :=
		inventory.NewService(
			inventoryRepository,
		)

	orderRepository :=
		orderpkg.NewRepository(
			pool,
		)

	orderService :=
		orderpkg.NewService(
			orderRepository,
			inventoryService,
		)

	paymentRepository :=
		NewRepository(
			pool,
		)

	paymentService :=
		NewService(
			paymentRepository,
			orderService,
		)

	// ---------------------------------------------------------
	// Load fresh pending-payment Order
	// ---------------------------------------------------------

	beforeOrder, err :=
		orderService.Get(
			ctx,
			orderID,
		)
	if err != nil {
		t.Fatalf(
			"load payment test order: %v",
			err,
		)
	}

	if beforeOrder.Status !=
		orderpkg.StatusPendingPayment {
		t.Fatalf(
			"expected order status %q, got %q",
			orderpkg.StatusPendingPayment,
			beforeOrder.Status,
		)
	}

	if beforeOrder.PaymentStatus !=
		orderpkg.PaymentStatusPending {
		t.Fatalf(
			"expected payment status %q, got %q",
			orderpkg.PaymentStatusPending,
			beforeOrder.PaymentStatus,
		)
	}

	if beforeOrder.PaymentMethod !=
		orderpkg.PaymentMethodBKash &&
		beforeOrder.PaymentMethod !=
			orderpkg.PaymentMethodNagad {
		t.Fatalf(
			"expected online payment method, got %q",
			beforeOrder.PaymentMethod,
		)
	}

	if beforeOrder.PaymentDueAt == nil {
		t.Fatal(
			"pending online order has no payment deadline",
		)
	}

	now :=
		time.Now().UTC()

	if !now.Before(
		*beforeOrder.PaymentDueAt,
	) {
		t.Fatalf(
			"payment test order already expired at %s",
			beforeOrder.PaymentDueAt.UTC().Format(
				time.RFC3339Nano,
			),
		)
	}

	if len(beforeOrder.Items) == 0 {
		t.Fatal(
			"payment test order has no items",
		)
	}

	// ---------------------------------------------------------
	// Capture inventory before payment
	// ---------------------------------------------------------

	beforeStock :=
		make(
			map[string]paymentIntegrationStock,
			len(beforeOrder.Items),
		)

	for _, item := range beforeOrder.Items {
		stock, err :=
			inventoryService.Get(
				ctx,
				item.VariantID,
			)
		if err != nil {
			t.Fatalf(
				"load inventory before payment for variant %s: %v",
				item.VariantID,
				err,
			)
		}

		if stock.QuantityReserved <
			item.Quantity {
			t.Fatalf(
				"expected at least %d reserved for variant %s, got %d",
				item.Quantity,
				item.VariantID,
				stock.QuantityReserved,
			)
		}

		beforeStock[item.VariantID] =
			paymentIntegrationStock{
				OnHand: stock.QuantityOnHand,

				Reserved: stock.QuantityReserved,

				Available: stock.AvailableQuantity,
			}
	}

	// ---------------------------------------------------------
	// Simulate a result that has already been VERIFIED by
	// the bKash/Nagad provider adapter.
	//
	// This does NOT pretend to verify the external provider.
	// We are testing the provider-neutral payment transaction.
	// ---------------------------------------------------------

	providerEventID :=
		fmt.Sprintf(
			"integration-event-%s",
			orderID,
		)

	providerPaymentID :=
		fmt.Sprintf(
			"integration-payment-%s",
			orderID,
		)

	providerTransactionID :=
		fmt.Sprintf(
			"integration-transaction-%s",
			orderID,
		)

	payload :=
		fmt.Appendf(
			nil,
			"provider=%s;order=%s;payment=%s",
			beforeOrder.PaymentMethod,
			orderID,
			providerPaymentID,
		)

	paidAt :=
		time.Now().UTC()

	input :=
		ConfirmVerifiedPaymentInput{
			OrderID: orderID,

			Provider: beforeOrder.PaymentMethod,

			ProviderEventID: providerEventID,

			EventType: "payment_succeeded",

			ProviderPaymentID: providerPaymentID,

			ProviderTransactionID: providerTransactionID,

			Amount: beforeOrder.TotalAmount,

			Currency: beforeOrder.Currency,

			PaidAt: paidAt,

			PayloadSHA256: PayloadSHA256(
				payload,
			),
		}

	result, err :=
		paymentService.ConfirmVerifiedPayment(
			ctx,
			input,
		)
	if err != nil {
		t.Fatalf(
			"confirm verified payment: %v",
			err,
		)
	}

	if !result.Processed {
		t.Fatal(
			"expected first verified payment event to be processed",
		)
	}

	if result.Payment.ID == "" {
		t.Fatal(
			"expected persisted payment id",
		)
	}

	if result.Payment.OrderID !=
		orderID {
		t.Fatalf(
			"expected payment order %q, got %q",
			orderID,
			result.Payment.OrderID,
		)
	}

	if result.Payment.Provider !=
		beforeOrder.PaymentMethod {
		t.Fatalf(
			"expected provider %q, got %q",
			beforeOrder.PaymentMethod,
			result.Payment.Provider,
		)
	}

	if result.Payment.Status !=
		StatusSucceeded {
		t.Fatalf(
			"expected payment status %q, got %q",
			StatusSucceeded,
			result.Payment.Status,
		)
	}

	if result.Payment.Amount !=
		beforeOrder.TotalAmount {
		t.Fatalf(
			"expected amount %d, got %d",
			beforeOrder.TotalAmount,
			result.Payment.Amount,
		)
	}

	if result.Payment.Currency !=
		beforeOrder.Currency {
		t.Fatalf(
			"expected currency %q, got %q",
			beforeOrder.Currency,
			result.Payment.Currency,
		)
	}

	if result.Payment.ProviderPaymentID !=
		providerPaymentID {
		t.Fatalf(
			"expected provider payment id %q, got %q",
			providerPaymentID,
			result.Payment.ProviderPaymentID,
		)
	}

	// ---------------------------------------------------------
	// Order must now be confirmed and paid.
	// ---------------------------------------------------------

	afterOrder, err :=
		orderService.Get(
			ctx,
			orderID,
		)
	if err != nil {
		t.Fatalf(
			"load order after payment: %v",
			err,
		)
	}

	if afterOrder.Status !=
		orderpkg.StatusConfirmed {
		t.Fatalf(
			"expected order status %q, got %q",
			orderpkg.StatusConfirmed,
			afterOrder.Status,
		)
	}

	if afterOrder.PaymentStatus !=
		orderpkg.PaymentStatusPaid {
		t.Fatalf(
			"expected payment status %q, got %q",
			orderpkg.PaymentStatusPaid,
			afterOrder.PaymentStatus,
		)
	}

	if afterOrder.PaidAt == nil {
		t.Fatal(
			"expected paid_at to be populated",
		)
	}

	if afterOrder.ConfirmedAt == nil {
		t.Fatal(
			"expected confirmed_at to be populated",
		)
	}

	// ---------------------------------------------------------
	// Reservation must have been committed.
	//
	// on_hand decreases by ordered quantity
	// reserved decreases by ordered quantity
	// ---------------------------------------------------------

	afterFirstStock :=
		make(
			map[string]paymentIntegrationStock,
			len(afterOrder.Items),
		)

	for _, item := range afterOrder.Items {
		stock, err :=
			inventoryService.Get(
				ctx,
				item.VariantID,
			)
		if err != nil {
			t.Fatalf(
				"load inventory after payment for variant %s: %v",
				item.VariantID,
				err,
			)
		}

		before := beforeStock[item.VariantID]

		expectedOnHand :=
			before.OnHand -
				item.Quantity

		expectedReserved :=
			before.Reserved -
				item.Quantity

		if stock.QuantityOnHand !=
			expectedOnHand {
			t.Fatalf(
				"variant %s: expected on-hand %d after payment, got %d",
				item.VariantID,
				expectedOnHand,
				stock.QuantityOnHand,
			)
		}

		if stock.QuantityReserved !=
			expectedReserved {
			t.Fatalf(
				"variant %s: expected reserved %d after payment, got %d",
				item.VariantID,
				expectedReserved,
				stock.QuantityReserved,
			)
		}

		afterFirstStock[item.VariantID] =
			paymentIntegrationStock{
				OnHand: stock.QuantityOnHand,

				Reserved: stock.QuantityReserved,

				Available: stock.AvailableQuantity,
			}
	}

	// ---------------------------------------------------------
	// Verify durable records.
	// ---------------------------------------------------------

	var paymentCount int

	err =
		pool.QueryRow(
			ctx,
			`
				SELECT count(*)
				FROM payments
				WHERE
					provider = $1
					AND provider_payment_id = $2
			`,
			input.Provider,
			input.ProviderPaymentID,
		).Scan(
			&paymentCount,
		)
	if err != nil {
		t.Fatalf(
			"count persisted payment: %v",
			err,
		)
	}

	if paymentCount != 1 {
		t.Fatalf(
			"expected exactly 1 payment record, got %d",
			paymentCount,
		)
	}

	var paymentEventCount int

	err =
		pool.QueryRow(
			ctx,
			`
				SELECT count(*)
				FROM payment_events
				WHERE
					provider = $1
					AND provider_event_id = $2
			`,
			input.Provider,
			input.ProviderEventID,
		).Scan(
			&paymentEventCount,
		)
	if err != nil {
		t.Fatalf(
			"count payment events: %v",
			err,
		)
	}

	if paymentEventCount != 1 {
		t.Fatalf(
			"expected exactly 1 provider event, got %d",
			paymentEventCount,
		)
	}

	var orderPaymentEventCount int

	err =
		pool.QueryRow(
			ctx,
			`
				SELECT count(*)
				FROM order_events
				WHERE
					order_id = $1::uuid
					AND event_type = 'payment_confirmed'
			`,
			orderID,
		).Scan(
			&orderPaymentEventCount,
		)
	if err != nil {
		t.Fatalf(
			"count order payment events: %v",
			err,
		)
	}

	if orderPaymentEventCount != 1 {
		t.Fatalf(
			"expected exactly 1 payment_confirmed order event, got %d",
			orderPaymentEventCount,
		)
	}

	// ---------------------------------------------------------
	// Replay EXACT same provider event.
	//
	// Must not:
	// - create another payment
	// - commit inventory again
	// - add another order payment event
	// ---------------------------------------------------------

	replay, err :=
		paymentService.ConfirmVerifiedPayment(
			ctx,
			input,
		)
	if err != nil {
		t.Fatalf(
			"replay verified payment: %v",
			err,
		)
	}

	if replay.Processed {
		t.Fatal(
			"expected replayed provider event to be idempotent",
		)
	}

	if replay.Payment.ID !=
		result.Payment.ID {
		t.Fatalf(
			"expected replay payment id %q, got %q",
			result.Payment.ID,
			replay.Payment.ID,
		)
	}

	for _, item := range afterOrder.Items {
		stock, err :=
			inventoryService.Get(
				ctx,
				item.VariantID,
			)
		if err != nil {
			t.Fatalf(
				"load inventory after replay for variant %s: %v",
				item.VariantID,
				err,
			)
		}

		expected := afterFirstStock[item.VariantID]

		if stock.QuantityOnHand !=
			expected.OnHand {
			t.Fatalf(
				"variant %s: replay changed on-hand from %d to %d",
				item.VariantID,
				expected.OnHand,
				stock.QuantityOnHand,
			)
		}

		if stock.QuantityReserved !=
			expected.Reserved {
			t.Fatalf(
				"variant %s: replay changed reserved from %d to %d",
				item.VariantID,
				expected.Reserved,
				stock.QuantityReserved,
			)
		}

		if stock.AvailableQuantity !=
			expected.Available {
			t.Fatalf(
				"variant %s: replay changed available from %d to %d",
				item.VariantID,
				expected.Available,
				stock.AvailableQuantity,
			)
		}
	}

	err =
		pool.QueryRow(
			ctx,
			`
				SELECT count(*)
				FROM order_events
				WHERE
					order_id = $1::uuid
					AND event_type = 'payment_confirmed'
			`,
			orderID,
		).Scan(
			&orderPaymentEventCount,
		)
	if err != nil {
		t.Fatalf(
			"count order payment events after replay: %v",
			err,
		)
	}

	if orderPaymentEventCount != 1 {
		t.Fatalf(
			"expected exactly 1 payment_confirmed event after replay, got %d",
			orderPaymentEventCount,
		)
	}
}
