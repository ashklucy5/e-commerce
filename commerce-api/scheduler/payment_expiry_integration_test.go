package scheduler

import (
	"context"
	"io"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"project.local/commerce-api/internal/inventory"
	orderpkg "project.local/commerce-api/internal/order"
	"project.local/commerce-api/internal/platform/config"
	"project.local/commerce-api/internal/platform/database"
)

type schedulerIntegrationStock struct {
	OnHand    int
	Reserved  int
	Available int
}

func TestPaymentExpirySchedulerRunOnceIntegration(
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
				"SCHEDULER_PAYMENT_EXPIRY_TEST_ORDER_ID",
			),
		)

	if orderID == "" {
		t.Skip(
			"set SCHEDULER_PAYMENT_EXPIRY_TEST_ORDER_ID to a fresh pending bKash/Nagad order",
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

	logger :=
		log.New(
			io.Discard,
			"",
			0,
		)

	runner :=
		New(
			orderService,
			time.Hour,
			100,
			logger,
		)

	// ---------------------------------------------------------
	// Verify target Order starts as a fresh pending online order.
	// ---------------------------------------------------------

	beforeOrder, err :=
		orderService.Get(
			ctx,
			orderID,
		)
	if err != nil {
		t.Fatalf(
			"load scheduler test order: %v",
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
			"pending online order has no payment_due_at",
		)
	}

	if len(beforeOrder.Items) == 0 {
		t.Fatal(
			"scheduler test order has no items",
		)
	}

	// ---------------------------------------------------------
	// Capture reserved inventory before scheduler expiry.
	// ---------------------------------------------------------

	beforeStock :=
		make(
			map[string]schedulerIntegrationStock,
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
				"load inventory before scheduler run for variant %s: %v",
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
			schedulerIntegrationStock{
				OnHand: stock.QuantityOnHand,

				Reserved: stock.QuantityReserved,

				Available: stock.AvailableQuantity,
			}
	}

	// ---------------------------------------------------------
	// Make ONLY the supplied target Order due immediately.
	//
	// This lets Scheduler.RunOnce() use its real time.Now().UTC()
	// without waiting for the normal 15-minute payment window.
	// ---------------------------------------------------------

	forcedDueAt :=
		time.Now().
			UTC().
			Add(
				-time.Second,
			)

	result, err :=
		pool.Exec(
			ctx,
			`
				UPDATE orders
				SET
					payment_due_at = $2,
					updated_at = now()
				WHERE
					id = $1::uuid
					AND status = 'pending_payment'
					AND payment_status = 'pending'
			`,
			orderID,
			forcedDueAt,
		)
	if err != nil {
		t.Fatalf(
			"force scheduler test payment deadline: %v",
			err,
		)
	}

	if result.RowsAffected() != 1 {
		t.Fatalf(
			"expected to update exactly 1 pending payment order, updated %d",
			result.RowsAffected(),
		)
	}

	// ---------------------------------------------------------
	// Run the actual Scheduler.RunOnce().
	// ---------------------------------------------------------

	expired, err :=
		runner.RunOnce(
			ctx,
		)
	if err != nil {
		t.Fatalf(
			"scheduler RunOnce: %v",
			err,
		)
	}

	// Other genuinely overdue orders may also be processed,
	// therefore we only require that at least one was expired.
	if expired < 1 {
		t.Fatalf(
			"expected scheduler to expire at least 1 order, got %d",
			expired,
		)
	}

	// ---------------------------------------------------------
	// Target Order must now be expired.
	// ---------------------------------------------------------

	afterOrder, err :=
		orderService.Get(
			ctx,
			orderID,
		)
	if err != nil {
		t.Fatalf(
			"load order after scheduler expiry: %v",
			err,
		)
	}

	if afterOrder.Status !=
		orderpkg.StatusPaymentExpired {
		t.Fatalf(
			"expected order status %q, got %q",
			orderpkg.StatusPaymentExpired,
			afterOrder.Status,
		)
	}

	if afterOrder.PaymentStatus !=
		orderpkg.PaymentStatusExpired {
		t.Fatalf(
			"expected payment status %q, got %q",
			orderpkg.PaymentStatusExpired,
			afterOrder.PaymentStatus,
		)
	}

	// ---------------------------------------------------------
	// Scheduler expiry must release reservation.
	//
	// on_hand stays unchanged
	// reserved decreases
	// available increases
	// ---------------------------------------------------------

	for _, item := range afterOrder.Items {
		stock, err :=
			inventoryService.Get(
				ctx,
				item.VariantID,
			)
		if err != nil {
			t.Fatalf(
				"load inventory after scheduler expiry for variant %s: %v",
				item.VariantID,
				err,
			)
		}

		before :=
			beforeStock[item.VariantID]

		expectedReserved :=
			before.Reserved -
				item.Quantity

		expectedAvailable :=
			before.Available +
				item.Quantity

		if stock.QuantityOnHand !=
			before.OnHand {
			t.Fatalf(
				"variant %s: scheduler expiry changed on-hand from %d to %d",
				item.VariantID,
				before.OnHand,
				stock.QuantityOnHand,
			)
		}

		if stock.QuantityReserved !=
			expectedReserved {
			t.Fatalf(
				"variant %s: expected reserved %d after scheduler expiry, got %d",
				item.VariantID,
				expectedReserved,
				stock.QuantityReserved,
			)
		}

		if stock.AvailableQuantity !=
			expectedAvailable {
			t.Fatalf(
				"variant %s: expected available %d after scheduler expiry, got %d",
				item.VariantID,
				expectedAvailable,
				stock.AvailableQuantity,
			)
		}
	}

	// ---------------------------------------------------------
	// Exactly one payment_expired Order event for target.
	// ---------------------------------------------------------

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

	// ---------------------------------------------------------
	// Run Scheduler.RunOnce() again.
	//
	// The target Order must not be processed twice.
	// ---------------------------------------------------------

	_, err =
		runner.RunOnce(
			ctx,
		)
	if err != nil {
		t.Fatalf(
			"second scheduler RunOnce: %v",
			err,
		)
	}

	finalOrder, err :=
		orderService.Get(
			ctx,
			orderID,
		)
	if err != nil {
		t.Fatalf(
			"load order after second scheduler run: %v",
			err,
		)
	}

	if finalOrder.Status !=
		orderpkg.StatusPaymentExpired ||
		finalOrder.PaymentStatus !=
			orderpkg.PaymentStatusExpired {
		t.Fatalf(
			"second scheduler run changed terminal state: status=%q payment_status=%q",
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
			"count payment_expired events after second scheduler run: %v",
			err,
		)
	}

	if expiryEventCount != 1 {
		t.Fatalf(
			"expected exactly 1 payment_expired event after second scheduler run, got %d",
			expiryEventCount,
		)
	}
}
