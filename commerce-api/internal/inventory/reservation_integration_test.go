package inventory

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const integrationReferenceType = "inventory_integration_test"

type integrationFixture struct {
	CategoryID string
	ProductID  string
	VariantID  string
}

func TestInventoryReservationLifecycleIntegration(
	t *testing.T,
) {
	pool := newIntegrationPool(t)

	ctx, cancel := context.WithTimeout(
		context.Background(),
		20*time.Second,
	)
	defer cancel()

	fixture := createInventoryFixture(
		t,
		ctx,
		pool,
		70,
	)

	repository := NewRepository(
		pool,
	)

	service := NewService(
		repository,
	)

	// ---------------------------------------------------------
	// Baseline
	// ---------------------------------------------------------

	stock, err := service.Get(
		ctx,
		fixture.VariantID,
	)
	if err != nil {
		t.Fatalf(
			"get initial inventory: %v",
			err,
		)
	}

	assertStock(
		t,
		stock,
		70,
		0,
		70,
	)

	// ---------------------------------------------------------
	// Reserve 6, then release
	// ---------------------------------------------------------

	releaseReference :=
		fmt.Sprintf(
			"release-%d",
			time.Now().UnixNano(),
		)

	reservations, err :=
		service.ReserveReference(
			ctx,
			ReserveReferenceInput{
				ReferenceType: integrationReferenceType,
				ReferenceID:   releaseReference,
				Items: []ReserveItem{
					{
						VariantID: fixture.VariantID,
						Quantity:  6,
					},
				},
				ActorType: "test",
				ActorID:   "inventory-integration",
			},
		)
	if err != nil {
		t.Fatalf(
			"reserve inventory for release: %v",
			err,
		)
	}

	if len(reservations) != 1 {
		t.Fatalf(
			"expected 1 reservation, got %d",
			len(reservations),
		)
	}

	if reservations[0].Status != "active" {
		t.Fatalf(
			"expected active reservation, got %q",
			reservations[0].Status,
		)
	}

	stock, err = service.Get(
		ctx,
		fixture.VariantID,
	)
	if err != nil {
		t.Fatalf(
			"get inventory after reserve: %v",
			err,
		)
	}

	assertStock(
		t,
		stock,
		70,
		6,
		64,
	)

	assertReservationStatus(
		t,
		ctx,
		pool,
		integrationReferenceType,
		releaseReference,
		fixture.VariantID,
		"active",
	)

	assertMovementCount(
		t,
		ctx,
		pool,
		integrationReferenceType,
		releaseReference,
		"reserve",
		1,
	)

	released, err :=
		service.ReleaseReference(
			ctx,
			ReferenceActionInput{
				ReferenceType: integrationReferenceType,
				ReferenceID:   releaseReference,
				Reason:        "integration_test_release",
				ActorType:     "test",
				ActorID:       "inventory-integration",
			},
		)
	if err != nil {
		t.Fatalf(
			"release inventory reservation: %v",
			err,
		)
	}

	if released != 1 {
		t.Fatalf(
			"expected 1 released reservation, got %d",
			released,
		)
	}

	stock, err = service.Get(
		ctx,
		fixture.VariantID,
	)
	if err != nil {
		t.Fatalf(
			"get inventory after release: %v",
			err,
		)
	}

	assertStock(
		t,
		stock,
		70,
		0,
		70,
	)

	assertReservationStatus(
		t,
		ctx,
		pool,
		integrationReferenceType,
		releaseReference,
		fixture.VariantID,
		"released",
	)

	assertMovementCount(
		t,
		ctx,
		pool,
		integrationReferenceType,
		releaseReference,
		"release",
		1,
	)

	// ---------------------------------------------------------
	// Reserve another 6, then commit
	// ---------------------------------------------------------

	commitReference :=
		fmt.Sprintf(
			"commit-%d",
			time.Now().UnixNano(),
		)

	_, err = service.ReserveReference(
		ctx,
		ReserveReferenceInput{
			ReferenceType: integrationReferenceType,
			ReferenceID:   commitReference,
			Items: []ReserveItem{
				{
					VariantID: fixture.VariantID,
					Quantity:  6,
				},
			},
			ActorType: "test",
			ActorID:   "inventory-integration",
		},
	)
	if err != nil {
		t.Fatalf(
			"reserve inventory for commit: %v",
			err,
		)
	}

	stock, err = service.Get(
		ctx,
		fixture.VariantID,
	)
	if err != nil {
		t.Fatalf(
			"get inventory before commit: %v",
			err,
		)
	}

	assertStock(
		t,
		stock,
		70,
		6,
		64,
	)

	committed, err :=
		service.CommitReference(
			ctx,
			ReferenceActionInput{
				ReferenceType: integrationReferenceType,
				ReferenceID:   commitReference,
				Reason:        "integration_test_commit",
				ActorType:     "test",
				ActorID:       "inventory-integration",
			},
		)
	if err != nil {
		t.Fatalf(
			"commit inventory reservation: %v",
			err,
		)
	}

	if committed != 1 {
		t.Fatalf(
			"expected 1 committed reservation, got %d",
			committed,
		)
	}

	stock, err = service.Get(
		ctx,
		fixture.VariantID,
	)
	if err != nil {
		t.Fatalf(
			"get inventory after commit: %v",
			err,
		)
	}

	assertStock(
		t,
		stock,
		64,
		0,
		64,
	)

	assertReservationStatus(
		t,
		ctx,
		pool,
		integrationReferenceType,
		commitReference,
		fixture.VariantID,
		"committed",
	)

	assertMovementCount(
		t,
		ctx,
		pool,
		integrationReferenceType,
		commitReference,
		"commit",
		1,
	)

	// ---------------------------------------------------------
	// Oversell protection
	// ---------------------------------------------------------

	oversellReference :=
		fmt.Sprintf(
			"oversell-%d",
			time.Now().UnixNano(),
		)

	_, err = service.ReserveReference(
		ctx,
		ReserveReferenceInput{
			ReferenceType: integrationReferenceType,
			ReferenceID:   oversellReference,
			Items: []ReserveItem{
				{
					VariantID: fixture.VariantID,
					Quantity:  1000,
				},
			},
		},
	)

	if !errors.Is(
		err,
		ErrInsufficientStock,
	) {
		t.Fatalf(
			"expected ErrInsufficientStock, got %v",
			err,
		)
	}

	// Failed reservation must not mutate inventory.
	stock, err = service.Get(
		ctx,
		fixture.VariantID,
	)
	if err != nil {
		t.Fatalf(
			"get inventory after oversell attempt: %v",
			err,
		)
	}

	assertStock(
		t,
		stock,
		64,
		0,
		64,
	)
}

func TestInventoryConcurrentReservationPreventsOversellIntegration(
	t *testing.T,
) {
	pool := newIntegrationPool(t)

	ctx, cancel := context.WithTimeout(
		context.Background(),
		20*time.Second,
	)
	defer cancel()

	fixture := createInventoryFixture(
		t,
		ctx,
		pool,
		10,
	)

	service := NewService(
		NewRepository(
			pool,
		),
	)

	type result struct {
		err error
	}

	start := make(
		chan struct{},
	)

	results := make(
		chan result,
		2,
	)

	runReserve := func(
		referenceID string,
	) {
		<-start

		_, err :=
			service.ReserveReference(
				ctx,
				ReserveReferenceInput{
					ReferenceType: integrationReferenceType,
					ReferenceID:   referenceID,
					Items: []ReserveItem{
						{
							VariantID: fixture.VariantID,
							Quantity:  6,
						},
					},
				},
			)

		results <- result{
			err: err,
		}
	}

	go runReserve(
		fmt.Sprintf(
			"concurrent-a-%d",
			time.Now().UnixNano(),
		),
	)

	go runReserve(
		fmt.Sprintf(
			"concurrent-b-%d",
			time.Now().UnixNano(),
		),
	)

	close(start)

	first := <-results
	second := <-results

	successes := 0
	insufficient := 0

	for _, item := range []result{
		first,
		second,
	} {
		switch {
		case item.err == nil:
			successes++

		case errors.Is(
			item.err,
			ErrInsufficientStock,
		):
			insufficient++

		default:
			t.Fatalf(
				"unexpected concurrent reservation error: %v",
				item.err,
			)
		}
	}

	if successes != 1 {
		t.Fatalf(
			"expected exactly 1 successful reservation, got %d",
			successes,
		)
	}

	if insufficient != 1 {
		t.Fatalf(
			"expected exactly 1 insufficient-stock result, got %d",
			insufficient,
		)
	}

	stock, err := service.Get(
		ctx,
		fixture.VariantID,
	)
	if err != nil {
		t.Fatalf(
			"get concurrent inventory result: %v",
			err,
		)
	}

	assertStock(
		t,
		stock,
		10,
		6,
		4,
	)
}

func newIntegrationPool(
	t *testing.T,
) *pgxpool.Pool {
	t.Helper()

	if os.Getenv(
		"RUN_INTEGRATION_TESTS",
	) != "1" {
		t.Skip(
			"set RUN_INTEGRATION_TESTS=1 to run PostgreSQL integration tests",
		)
	}

	required := []string{
		"POSTGRES_HOST",
		"POSTGRES_PORT",
		"POSTGRES_DB",
		"POSTGRES_USER",
		"POSTGRES_PASSWORD",
	}

	for _, name := range required {
		if os.Getenv(name) == "" {
			t.Fatalf(
				"missing environment variable %s",
				name,
			)
		}
	}

	sslMode :=
		os.Getenv(
			"POSTGRES_SSLMODE",
		)

	if sslMode == "" {
		sslMode = "disable"
	}

	databaseURL := &url.URL{
		Scheme: "postgres",
		User: url.UserPassword(
			os.Getenv(
				"POSTGRES_USER",
			),
			os.Getenv(
				"POSTGRES_PASSWORD",
			),
		),
		Host: net.JoinHostPort(
			os.Getenv(
				"POSTGRES_HOST",
			),
			os.Getenv(
				"POSTGRES_PORT",
			),
		),
		Path: os.Getenv(
			"POSTGRES_DB",
		),
	}

	query :=
		databaseURL.Query()

	query.Set(
		"sslmode",
		sslMode,
	)

	query.Set(
		"search_path",
		"public",
	)

	databaseURL.RawQuery =
		query.Encode()

	ctx, cancel :=
		context.WithTimeout(
			context.Background(),
			5*time.Second,
		)
	defer cancel()

	pool, err :=
		pgxpool.New(
			ctx,
			databaseURL.String(),
		)
	if err != nil {
		t.Fatalf(
			"create integration postgres pool: %v",
			err,
		)
	}

	if err := pool.Ping(
		ctx,
	); err != nil {
		pool.Close()

		t.Fatalf(
			"ping integration postgres: %v",
			err,
		)
	}

	t.Cleanup(
		func() {
			pool.Close()
		},
	)

	return pool
}

func createInventoryFixture(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	quantityOnHand int,
) integrationFixture {
	t.Helper()

	suffix :=
		fmt.Sprintf(
			"%d",
			time.Now().UnixNano(),
		)

	var fixture integrationFixture

	err := pool.QueryRow(
		ctx,
		`
			INSERT INTO categories (
				name,
				slug,
				is_active
			)
			VALUES (
				$1,
				$2,
				true
			)
			RETURNING id::text
		`,
		"Inventory Integration "+suffix,
		"inventory-integration-"+suffix,
	).Scan(
		&fixture.CategoryID,
	)
	if err != nil {
		t.Fatalf(
			"create integration category: %v",
			err,
		)
	}

	err = pool.QueryRow(
		ctx,
		`
			INSERT INTO products (
				category_id,
				product_code,
				name,
				slug,
				status,
				published_at
			)
			VALUES (
				$1::uuid,
				$2,
				$3,
				$4,
				'active',
				now()
			)
			RETURNING id::text
		`,
		fixture.CategoryID,
		"INV-TEST-"+suffix,
		"Inventory Integration Product "+suffix,
		"inventory-integration-product-"+suffix,
	).Scan(
		&fixture.ProductID,
	)
	if err != nil {
		cleanupInventoryFixture(
			context.Background(),
			pool,
			fixture,
		)

		t.Fatalf(
			"create integration product: %v",
			err,
		)
	}

	err = pool.QueryRow(
		ctx,
		`
			INSERT INTO product_variants (
				product_id,
				sku,
				minimum_order_quantity,
				order_increment,
				price_amount,
				currency,
				is_active
			)
			VALUES (
				$1::uuid,
				$2,
				1,
				1,
				10000,
				'BDT',
				true
			)
			RETURNING id::text
		`,
		fixture.ProductID,
		"INV-TEST-SKU-"+suffix,
	).Scan(
		&fixture.VariantID,
	)
	if err != nil {
		cleanupInventoryFixture(
			context.Background(),
			pool,
			fixture,
		)

		t.Fatalf(
			"create integration variant: %v",
			err,
		)
	}

	_, err = pool.Exec(
		ctx,
		`
			INSERT INTO inventory (
				variant_id,
				quantity_on_hand,
				quantity_reserved,
				reorder_level
			)
			VALUES (
				$1::uuid,
				$2,
				0,
				5
			)
		`,
		fixture.VariantID,
		quantityOnHand,
	)
	if err != nil {
		cleanupInventoryFixture(
			context.Background(),
			pool,
			fixture,
		)

		t.Fatalf(
			"create integration inventory: %v",
			err,
		)
	}

	t.Cleanup(
		func() {
			cleanupInventoryFixture(
				context.Background(),
				pool,
				fixture,
			)
		},
	)

	return fixture
}

func cleanupInventoryFixture(
	ctx context.Context,
	pool *pgxpool.Pool,
	fixture integrationFixture,
) {
	if fixture.VariantID != "" {
		_, _ = pool.Exec(
			ctx,
			`
				DELETE FROM inventory_movements
				WHERE variant_id = $1::uuid
			`,
			fixture.VariantID,
		)

		_, _ = pool.Exec(
			ctx,
			`
				DELETE FROM inventory_reservations
				WHERE variant_id = $1::uuid
			`,
			fixture.VariantID,
		)

		_, _ = pool.Exec(
			ctx,
			`
				DELETE FROM inventory
				WHERE variant_id = $1::uuid
			`,
			fixture.VariantID,
		)

		_, _ = pool.Exec(
			ctx,
			`
				DELETE FROM product_variants
				WHERE id = $1::uuid
			`,
			fixture.VariantID,
		)
	}

	if fixture.ProductID != "" {
		_, _ = pool.Exec(
			ctx,
			`
				DELETE FROM products
				WHERE id = $1::uuid
			`,
			fixture.ProductID,
		)
	}

	if fixture.CategoryID != "" {
		_, _ = pool.Exec(
			ctx,
			`
				DELETE FROM categories
				WHERE id = $1::uuid
			`,
			fixture.CategoryID,
		)
	}
}

func assertStock(
	t *testing.T,
	stock StockItem,
	onHand int,
	reserved int,
	available int,
) {
	t.Helper()

	if stock.QuantityOnHand != onHand {
		t.Fatalf(
			"expected on-hand %d, got %d",
			onHand,
			stock.QuantityOnHand,
		)
	}

	if stock.QuantityReserved != reserved {
		t.Fatalf(
			"expected reserved %d, got %d",
			reserved,
			stock.QuantityReserved,
		)
	}

	if stock.AvailableQuantity != available {
		t.Fatalf(
			"expected available %d, got %d",
			available,
			stock.AvailableQuantity,
		)
	}
}

func assertReservationStatus(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	referenceType string,
	referenceID string,
	variantID string,
	expected string,
) {
	t.Helper()

	var status string

	err := pool.QueryRow(
		ctx,
		`
			SELECT status
			FROM inventory_reservations
			WHERE
				reference_type = $1
				AND reference_id = $2
				AND variant_id = $3::uuid
		`,
		referenceType,
		referenceID,
		variantID,
	).Scan(
		&status,
	)
	if err != nil {
		t.Fatalf(
			"load reservation status: %v",
			err,
		)
	}

	if status != expected {
		t.Fatalf(
			"expected reservation status %q, got %q",
			expected,
			status,
		)
	}
}

func assertMovementCount(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	referenceType string,
	referenceID string,
	movementType string,
	expected int,
) {
	t.Helper()

	var count int

	err := pool.QueryRow(
		ctx,
		`
			SELECT count(*)
			FROM inventory_movements
			WHERE
				reference_type = $1
				AND reference_id = $2
				AND movement_type = $3
		`,
		referenceType,
		referenceID,
		movementType,
	).Scan(
		&count,
	)
	if err != nil {
		t.Fatalf(
			"count inventory movements: %v",
			err,
		)
	}

	if count != expected {
		t.Fatalf(
			"expected %d %q movement(s), got %d",
			expected,
			movementType,
			count,
		)
	}
}
