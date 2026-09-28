package order

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"project.local/commerce-api/internal/inventory"
	"project.local/commerce-api/internal/platform/config"
	"project.local/commerce-api/internal/platform/database"
)

type expiryIntegrationStock struct {
	OnHand    int
	Reserved  int
	Available int
}

func TestExpirePendingPaymentIntegration(
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
				"PAYMENT_EXPIRY_TEST_ORDER_ID",
			),
		)

	if orderID == "" {
		t.Skip(
			"set PAYMENT_EXPIRY_TEST_ORDER_ID to a fresh pending payment order",
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
		NewRepository(
			pool,
		)

	orderService :=
		NewService(
			orderRepository,
			inventoryService,
		)

	beforeOrder, err :=
		orderService.Get(
			ctx,
			orderID,
		)
	if err != nil {
		t.Fatalf(
			"load expiry test order: %v",
			err,
		)
	}

	if beforeOrder.Status !=
		StatusPendingPayment {
		t.Fatalf(
			"expected order status %q, got %q",
			StatusPendingPayment,
			beforeOrder.Status,
		)
	}

	if beforeOrder.PaymentStatus !=
		PaymentStatusPending {
		t.Fatalf(
			"expected payment status %q, got %q",
			PaymentStatusPending,
			beforeOrder.PaymentStatus,
		)
	}

	if beforeOrder.PaymentMethod !=
		PaymentMethodBKash &&
		beforeOrder.PaymentMethod !=
			PaymentMethodNagad &&
		beforeOrder.PaymentMethod !=
			PaymentMethodRocket &&
		beforeOrder.PaymentMethod !=
			PaymentMethodBankTransfer {
		t.Fatalf(
			"expected pending digital/manual payment method, got %q",
			beforeOrder.PaymentMethod,
		)
	}

	if beforeOrder.PaymentDueAt == nil {
		t.Fatal(
			"pending payment order has no payment_due_at",
		)
	}

	if len(beforeOrder.Items) == 0 {
		t.Fatal(
			"expiry test order has no items",
		)
	}

	beforeStock :=
		make(
			map[string]expiryIntegrationStock,
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
				"load inventory before expiry for variant %s: %v",
				item.VariantID,
				err,
			)
		}

		if stock.QuantityReserved <
			item.Quantity {
			t.Fatalf(
				"variant %s: expected at least %d reserved, got %d",
				item.VariantID,
				item.Quantity,
				stock.QuantityReserved,
			)
		}

		beforeStock[item.VariantID] =
			expiryIntegrationStock{
				OnHand: stock.QuantityOnHand,

				Reserved: stock.QuantityReserved,

				Available: stock.AvailableQuantity,
			}
	}

	// Simulate the scheduler running one second after
	// this Order's real payment deadline.
	expiryNow :=
		beforeOrder.PaymentDueAt.
			Add(
				time.Second,
			)

	expired, err :=
		orderService.ExpirePendingPayments(
			ctx,
			expiryNow,
			100,
		)
	if err != nil {
		t.Fatalf(
			"expire pending payments: %v",
			err,
		)
	}

	if expired < 1 {
		t.Fatalf(
			"expected at least one expired payment order, got %d",
			expired,
		)
	}

	afterOrder, err :=
		orderService.Get(
			ctx,
			orderID,
		)
	if err != nil {
		t.Fatalf(
			"load order after expiry: %v",
			err,
		)
	}

	if afterOrder.Status !=
		StatusPaymentExpired {
		t.Fatalf(
			"expected order status %q, got %q",
			StatusPaymentExpired,
			afterOrder.Status,
		)
	}

	if afterOrder.PaymentStatus !=
		PaymentStatusExpired {
		t.Fatalf(
			"expected payment status %q, got %q",
			PaymentStatusExpired,
			afterOrder.PaymentStatus,
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
				"load inventory after expiry for variant %s: %v",
				item.VariantID,
				err,
			)
		}

		before := beforeStock[item.VariantID]

		expectedReserved :=
			before.Reserved -
				item.Quantity

		expectedAvailable :=
			before.Available +
				item.Quantity

		if stock.QuantityOnHand !=
			before.OnHand {
			t.Fatalf(
				"variant %s: expiry changed on-hand from %d to %d",
				item.VariantID,
				before.OnHand,
				stock.QuantityOnHand,
			)
		}

		if stock.QuantityReserved !=
			expectedReserved {
			t.Fatalf(
				"variant %s: expected reserved %d after expiry, got %d",
				item.VariantID,
				expectedReserved,
				stock.QuantityReserved,
			)
		}

		if stock.AvailableQuantity !=
			expectedAvailable {
			t.Fatalf(
				"variant %s: expected available %d after expiry, got %d",
				item.VariantID,
				expectedAvailable,
				stock.AvailableQuantity,
			)
		}
	}

	var expiryEventCount int

	err =
		pool.QueryRow(
			ctx,
			`
				SELECT count(*)
				FROM order_events
				WHERE
					order_id = $1::uuid
					AND event_type = 'payment_expired'
			`,
			orderID,
		).Scan(
			&expiryEventCount,
		)
	if err != nil {
		t.Fatalf(
			"count payment_expired events: %v",
			err,
		)
	}

	if expiryEventCount != 1 {
		t.Fatalf(
			"expected exactly 1 payment_expired event, got %d",
			expiryEventCount,
		)
	}

	// Running expiry again must be idempotent.
	expiredAgain, err :=
		orderService.ExpirePendingPayments(
			ctx,
			expiryNow.Add(
				time.Minute,
			),
			100,
		)
	if err != nil {
		t.Fatalf(
			"repeat pending payment expiry: %v",
			err,
		)
	}

	// Other test/database orders could theoretically expire in
	// this batch, so don't require expiredAgain == 0 globally.
	_ = expiredAgain

	finalOrder, err :=
		orderService.Get(
			ctx,
			orderID,
		)
	if err != nil {
		t.Fatalf(
			"load order after repeated expiry: %v",
			err,
		)
	}

	if finalOrder.Status !=
		StatusPaymentExpired ||
		finalOrder.PaymentStatus !=
			PaymentStatusExpired {
		t.Fatalf(
			"repeated expiry changed terminal state: status=%q payment_status=%q",
			finalOrder.Status,
			finalOrder.PaymentStatus,
		)
	}

	err =
		pool.QueryRow(
			ctx,
			`
				SELECT count(*)
				FROM order_events
				WHERE
					order_id = $1::uuid
					AND event_type = 'payment_expired'
			`,
			orderID,
		).Scan(
			&expiryEventCount,
		)
	if err != nil {
		t.Fatalf(
			"count payment_expired events after replay: %v",
			err,
		)
	}

	if expiryEventCount != 1 {
		t.Fatalf(
			"expected exactly 1 payment_expired event after replay, got %d",
			expiryEventCount,
		)
	}
}
