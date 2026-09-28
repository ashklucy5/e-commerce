package admin

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	platformconfig "project.local/commerce-api/internal/platform/config"
	platformdatabase "project.local/commerce-api/internal/platform/database"
	promotiondomain "project.local/commerce-api/internal/promotion"
)

const (
	runPromotionDBTestEnv = "RUN_PROMOTION_DB_TEST"
	promotionTestDBURLEnv = "PROMOTION_TEST_DATABASE_URL"
)

type promotionDBFixture struct {
	t *testing.T

	db *pgxpool.Pool

	adminService     *Service
	promotionService *promotiondomain.Service

	requestID string

	categoryIDs  []string
	productIDs   []string
	variantIDs   []string
	promotionIDs []string
}

func TestPromotionDBEndToEnd(t *testing.T) {
	f := newPromotionDBFixture(t)
	ctx := context.Background()

	catalog := f.seedCatalog()

	now := time.Now().UTC().Truncate(time.Second)
	startsAt := now.Add(-time.Hour)
	endsAt := now.Add(time.Hour)

	twentyPercent := 2000
	flashSale, err := f.adminService.CreatePromotion(
		ctx,
		PromotionConfigInput{
			Name:          "Promotion DB Flash Sale",
			Scope:         promotiondomain.ScopeProduct,
			CampaignType:  promotiondomain.CampaignTypeFlashSale,
			DiscountType:  promotiondomain.DiscountTypePercentage,
			PercentageBPS: &twentyPercent,
			Currency:      "BDT",
			Status:        promotiondomain.StatusActive,
			StartsAt:      &startsAt,
			EndsAt:        &endsAt,
			Targets: []PromotionTargetInput{
				{ProductID: catalog.targetProductID},
			},
		},
		f.metadata(),
	)
	if err != nil {
		t.Fatalf("create flash sale: %v", err)
	}
	f.promotionIDs = append(f.promotionIDs, flashSale.ID)

	if flashSale.Scope != promotiondomain.ScopeProduct ||
		flashSale.CampaignType != promotiondomain.CampaignTypeFlashSale ||
		flashSale.TargetCount != 1 ||
		len(flashSale.Targets) != 1 {
		t.Fatalf("unexpected flash sale: %+v", flashSale)
	}

	fivePercent := 500
	automaticOrderPromotion, err := f.adminService.CreatePromotion(
		ctx,
		PromotionConfigInput{
			Name:          "Promotion DB Automatic Order",
			Scope:         promotiondomain.ScopeOrder,
			CampaignType:  promotiondomain.CampaignTypeStandard,
			DiscountType:  promotiondomain.DiscountTypePercentage,
			PercentageBPS: &fivePercent,
			Currency:      "BDT",
			Status:        promotiondomain.StatusActive,
		},
		f.metadata(),
	)
	if err != nil {
		t.Fatalf("create automatic order promotion: %v", err)
	}
	f.promotionIDs = append(f.promotionIDs, automaticOrderPromotion.ID)

	lowCode := "PROMODB10"
	tenPercent := 1000
	lowCodePromotion, err := f.adminService.CreatePromotion(
		ctx,
		PromotionConfigInput{
			Name:          "Promotion DB Low Code",
			Code:          &lowCode,
			Scope:         promotiondomain.ScopeOrder,
			CampaignType:  promotiondomain.CampaignTypeStandard,
			DiscountType:  promotiondomain.DiscountTypePercentage,
			PercentageBPS: &tenPercent,
			Currency:      "BDT",
			Status:        promotiondomain.StatusActive,
		},
		f.metadata(),
	)
	if err != nil {
		t.Fatalf("create low code promotion: %v", err)
	}
	f.promotionIDs = append(f.promotionIDs, lowCodePromotion.ID)

	highCode := "PROMODB30"
	thirtyPercent := 3000
	highCodePromotion, err := f.adminService.CreatePromotion(
		ctx,
		PromotionConfigInput{
			Name:          "Promotion DB High Code",
			Code:          &highCode,
			Scope:         promotiondomain.ScopeOrder,
			CampaignType:  promotiondomain.CampaignTypeStandard,
			DiscountType:  promotiondomain.DiscountTypePercentage,
			PercentageBPS: &thirtyPercent,
			Currency:      "BDT",
			Status:        promotiondomain.StatusActive,
		},
		f.metadata(),
	)
	if err != nil {
		t.Fatalf("create high code promotion: %v", err)
	}
	f.promotionIDs = append(f.promotionIDs, highCodePromotion.ID)

	lines := []promotiondomain.EvaluateLine{
		{
			VariantID:       catalog.targetVariantID,
			Quantity:        1,
			UnitPriceAmount: 2000,
		},
		{
			VariantID:       catalog.outsideVariantID,
			Quantity:        1,
			UnitPriceAmount: 1000,
		},
	}

	automaticResult, err := f.promotionService.Evaluate(
		ctx,
		promotiondomain.EvaluateInput{
			SubtotalAmount: 3000,
			Currency:       "BDT",
			Lines:          lines,
			Now:            now,
		},
	)
	if err != nil {
		t.Fatalf("evaluate automatic promotions: %v", err)
	}
	if !automaticResult.Applied ||
		automaticResult.PromotionID != flashSale.ID ||
		automaticResult.DiscountAmount != 400 {
		t.Fatalf("automatic promotion result = %+v, want flash sale discount 400", automaticResult)
	}

	lowCodeResult, err := f.promotionService.Evaluate(
		ctx,
		promotiondomain.EvaluateInput{
			Code:           lowCode,
			SubtotalAmount: 3000,
			Currency:       "BDT",
			Lines:          lines,
			Now:            now,
		},
	)
	if err != nil {
		t.Fatalf("evaluate low code against automatic flash sale: %v", err)
	}
	if lowCodeResult.PromotionID != flashSale.ID ||
		lowCodeResult.DiscountAmount != 400 {
		t.Fatalf("low code result = %+v, want better automatic flash sale", lowCodeResult)
	}

	highCodeResult, err := f.promotionService.Evaluate(
		ctx,
		promotiondomain.EvaluateInput{
			Code:           highCode,
			SubtotalAmount: 3000,
			Currency:       "BDT",
			Lines:          lines,
			Now:            now,
		},
	)
	if err != nil {
		t.Fatalf("evaluate high code against automatic flash sale: %v", err)
	}
	if highCodeResult.PromotionID != highCodePromotion.ID ||
		highCodeResult.PromotionCode != highCode ||
		highCodeResult.DiscountAmount != 900 {
		t.Fatalf("high code result = %+v, want explicit code discount 900", highCodeResult)
	}

	storefrontPromotions, err := f.promotionService.ListStorefrontActive(
		ctx,
		"BDT",
		now,
	)
	if err != nil {
		t.Fatalf("list storefront promotions: %v", err)
	}
	if len(storefrontPromotions) != 1 ||
		storefrontPromotions[0].ID != automaticOrderPromotion.ID {
		t.Fatalf(
			"storefront standard promotions = %+v, want only automatic order promotion %s",
			storefrontPromotions,
			automaticOrderPromotion.ID,
		)
	}

	flashSales, err := f.promotionService.ListActiveFlashSales(
		ctx,
		"BDT",
		now,
	)
	if err != nil {
		t.Fatalf("list active flash sales: %v", err)
	}
	if len(flashSales) != 1 || flashSales[0].ID != flashSale.ID {
		t.Fatalf("active flash sales = %+v, want only %s", flashSales, flashSale.ID)
	}
	if len(flashSales[0].Targets) != 2 {
		t.Fatalf("flash sale target count = %d, want 2 product variants", len(flashSales[0].Targets))
	}

	effectivePrices := map[string]int64{}
	for _, target := range flashSales[0].Targets {
		effectivePrices[target.VariantID] = target.EffectivePriceAmount
	}
	if effectivePrices[catalog.targetVariantID] != 1600 {
		t.Fatalf(
			"target variant effective price = %d, want 1600",
			effectivePrices[catalog.targetVariantID],
		)
	}
	if effectivePrices[catalog.secondTargetVariantID] != 800 {
		t.Fatalf(
			"second target variant effective price = %d, want 800",
			effectivePrices[catalog.secondTargetVariantID],
		)
	}
	if _, exists := effectivePrices[catalog.outsideVariantID]; exists {
		t.Fatalf("outside variant %s unexpectedly appeared in flash sale", catalog.outsideVariantID)
	}

	overlapPercent := 2500
	_, err = f.adminService.CreatePromotion(
		ctx,
		PromotionConfigInput{
			Name:          "Promotion DB Conflicting Flash Sale",
			Scope:         promotiondomain.ScopeProduct,
			CampaignType:  promotiondomain.CampaignTypeFlashSale,
			DiscountType:  promotiondomain.DiscountTypePercentage,
			PercentageBPS: &overlapPercent,
			Currency:      "BDT",
			Status:        promotiondomain.StatusActive,
			StartsAt:      &startsAt,
			EndsAt:        &endsAt,
			Targets: []PromotionTargetInput{
				{VariantID: catalog.targetVariantID},
			},
		},
		f.metadata(),
	)
	if !errors.Is(err, ErrAdminPromotionTargetConflict) {
		t.Fatalf("overlapping flash sale error = %v, want %v", err, ErrAdminPromotionTargetConflict)
	}

	var conflictingPromotionCount int
	if err := f.db.QueryRow(
		ctx,
		`SELECT COUNT(*)::integer FROM promotions WHERE name = 'Promotion DB Conflicting Flash Sale'`,
	).Scan(&conflictingPromotionCount); err != nil {
		t.Fatalf("count rolled-back conflicting promotion: %v", err)
	}
	if conflictingPromotionCount != 0 {
		t.Fatalf("conflicting flash sale persisted %d row(s), want 0", conflictingPromotionCount)
	}

	f.testProductDiscountLifecycle(catalog.outsideVariantID)
}

type promotionDBCatalog struct {
	targetProductID       string
	targetVariantID       string
	secondTargetVariantID string
	outsideVariantID      string
}

func newPromotionDBFixture(t *testing.T) *promotionDBFixture {
	t.Helper()

	db := openPromotionTestDB(t)
	repository := promotiondomain.NewRepository(db)

	fixture := &promotionDBFixture{
		t:                t,
		db:               db,
		adminService:     NewService(db),
		promotionService: promotiondomain.NewService(repository),
		requestID: fmt.Sprintf(
			"promotion-db-test-%d",
			time.Now().UTC().UnixNano(),
		),
	}

	t.Cleanup(fixture.cleanup)
	return fixture
}

func openPromotionTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	if os.Getenv(runPromotionDBTestEnv) != "1" {
		t.Skipf(
			"set %s=1 and %s to a dedicated PostgreSQL test database",
			runPromotionDBTestEnv,
			promotionTestDBURLEnv,
		)
	}

	rawURL := strings.TrimSpace(os.Getenv(promotionTestDBURLEnv))
	if rawURL == "" {
		t.Fatalf("%s is required when %s=1", promotionTestDBURLEnv, runPromotionDBTestEnv)
	}

	cfg, databaseName := promotionTestConfigFromURL(t, rawURL)
	if !strings.Contains(strings.ToLower(databaseName), "test") {
		t.Fatalf(
			"refusing promotion integration test against database %q: name must contain 'test'",
			databaseName,
		)
	}

	migrationCtx, migrationCancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer migrationCancel()

	if _, err := platformdatabase.ApplyMigrations(migrationCtx, cfg); err != nil {
		t.Fatalf("apply migrations to dedicated promotion test database: %v", err)
	}

	poolConfig, err := pgxpool.ParseConfig(rawURL)
	if err != nil {
		t.Fatalf("parse promotion test database URL: %v", err)
	}
	poolConfig.MaxConns = 6
	poolConfig.MinConns = 0

	db, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		t.Fatalf("open promotion test database: %v", err)
	}
	t.Cleanup(db.Close)

	pingCtx, pingCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer pingCancel()
	if err := db.Ping(pingCtx); err != nil {
		t.Fatalf("ping promotion test database: %v", err)
	}

	var scopeColumn bool
	var campaignTypeColumn bool
	var targetTable bool
	if err := db.QueryRow(
		context.Background(),
		`
			SELECT
				EXISTS (
					SELECT 1
					FROM information_schema.columns
					WHERE table_schema = 'public' AND table_name = 'promotions' AND column_name = 'scope'
				),
				EXISTS (
					SELECT 1
					FROM information_schema.columns
					WHERE table_schema = 'public' AND table_name = 'promotions' AND column_name = 'campaign_type'
				),
				to_regclass('public.promotion_targets') IS NOT NULL
		`,
	).Scan(&scopeColumn, &campaignTypeColumn, &targetTable); err != nil {
		t.Fatalf("verify promotion migration schema: %v", err)
	}
	if !scopeColumn || !campaignTypeColumn || !targetTable {
		t.Fatalf(
			"promotion migration incomplete: scope=%t campaign_type=%t promotion_targets=%t",
			scopeColumn,
			campaignTypeColumn,
			targetTable,
		)
	}

	return db
}

func promotionTestConfigFromURL(t *testing.T, rawURL string) (platformconfig.Config, string) {
	t.Helper()

	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("parse %s: %v", promotionTestDBURLEnv, err)
	}
	if parsed.Scheme != "postgres" && parsed.Scheme != "postgresql" {
		t.Fatalf("%s must use postgres:// or postgresql://", promotionTestDBURLEnv)
	}
	if parsed.User == nil {
		t.Fatalf("%s must include a database user", promotionTestDBURLEnv)
	}

	password, _ := parsed.User.Password()
	port := uint16(5432)
	if value := parsed.Port(); value != "" {
		parsedPort, parseErr := strconv.ParseUint(value, 10, 16)
		if parseErr != nil {
			t.Fatalf("invalid PostgreSQL port %q: %v", value, parseErr)
		}
		port = uint16(parsedPort)
	}

	databaseName, err := url.PathUnescape(strings.TrimPrefix(parsed.Path, "/"))
	if err != nil || databaseName == "" {
		t.Fatalf("invalid test database name in %s", promotionTestDBURLEnv)
	}

	sslMode := strings.TrimSpace(parsed.Query().Get("sslmode"))
	if sslMode == "" {
		sslMode = "disable"
	}

	return platformconfig.Config{
		PostgresHost:     parsed.Hostname(),
		PostgresPort:     port,
		PostgresDB:       databaseName,
		PostgresUser:     parsed.User.Username(),
		PostgresPassword: password,
		PostgresSSLMode:  sslMode,
	}, databaseName
}

func (f *promotionDBFixture) seedCatalog() promotionDBCatalog {
	f.t.Helper()
	ctx := context.Background()
	tag := fmt.Sprintf("%d", time.Now().UTC().UnixNano())

	categoryID := f.insertCategory("promotion-db-" + tag)
	targetProductID := f.insertProduct(categoryID, "PROMO-TARGET-"+tag, "Promotion Target", "promotion-target-"+tag)
	outsideProductID := f.insertProduct(categoryID, "PROMO-OUTSIDE-"+tag, "Promotion Outside", "promotion-outside-"+tag)

	targetVariantID := f.insertVariant(targetProductID, "PROMO-TARGET-A-"+tag, 2000)
	secondTargetVariantID := f.insertVariant(targetProductID, "PROMO-TARGET-B-"+tag, 1000)
	outsideVariantID := f.insertVariant(outsideProductID, "PROMO-OUTSIDE-A-"+tag, 1000)

	if _, err := f.db.Exec(
		ctx,
		`
			INSERT INTO product_images (
				product_id, variant_id, url, alt_text, sort_order, is_primary, created_at
			)
			VALUES ($1::uuid, NULL, $2, 'Promotion DB test image', 0, true, now())
		`,
		targetProductID,
		"https://example.invalid/promotion-db-test-"+tag+".jpg",
	); err != nil {
		f.t.Fatalf("insert promotion test product image: %v", err)
	}

	return promotionDBCatalog{
		targetProductID:       targetProductID,
		targetVariantID:       targetVariantID,
		secondTargetVariantID: secondTargetVariantID,
		outsideVariantID:      outsideVariantID,
	}
}

func (f *promotionDBFixture) insertCategory(slug string) string {
	f.t.Helper()

	var id string
	if err := f.db.QueryRow(
		context.Background(),
		`
			INSERT INTO categories (name, slug, is_active, created_at, updated_at)
			VALUES ('Promotion DB Category', $1, true, now(), now())
			RETURNING id::text
		`,
		slug,
	).Scan(&id); err != nil {
		f.t.Fatalf("insert promotion test category: %v", err)
	}
	f.categoryIDs = append(f.categoryIDs, id)
	return id
}

func (f *promotionDBFixture) insertProduct(categoryID, code, name, slug string) string {
	f.t.Helper()

	var id string
	if err := f.db.QueryRow(
		context.Background(),
		`
			INSERT INTO products (
				category_id, product_code, name, slug, status, is_featured,
				published_at, created_at, updated_at
			)
			VALUES ($1::uuid, $2, $3, $4, 'active', false, now(), now(), now())
			RETURNING id::text
		`,
		categoryID,
		code,
		name,
		slug,
	).Scan(&id); err != nil {
		f.t.Fatalf("insert promotion test product %s: %v", code, err)
	}
	f.productIDs = append(f.productIDs, id)
	return id
}

func (f *promotionDBFixture) insertVariant(productID, sku string, price int64) string {
	f.t.Helper()

	var id string
	if err := f.db.QueryRow(
		context.Background(),
		`
			INSERT INTO product_variants (
				product_id, sku, minimum_order_quantity, order_increment,
				price_amount, compare_at_price_amount, currency, is_active,
				created_at, updated_at
			)
			VALUES ($1::uuid, $2, 1, 1, $3, NULL, 'BDT', true, now(), now())
			RETURNING id::text
		`,
		productID,
		sku,
		price,
	).Scan(&id); err != nil {
		f.t.Fatalf("insert promotion test variant %s: %v", sku, err)
	}
	f.variantIDs = append(f.variantIDs, id)
	return id
}

func (f *promotionDBFixture) metadata() AdminActionMetadata {
	return AdminActionMetadata{
		RequestID: f.requestID,
		IPAddress: "127.0.0.1",
		UserAgent: "promotion-db-integration-test",
	}
}

func (f *promotionDBFixture) testProductDiscountLifecycle(variantID string) {
	f.t.Helper()
	ctx := context.Background()

	fifteenPercent := 1500
	applied, err := f.adminService.ApplyProductDiscount(
		ctx,
		variantID,
		ProductDiscountConfigInput{
			DiscountType:  promotiondomain.DiscountTypePercentage,
			PercentageBPS: &fifteenPercent,
		},
		f.metadata(),
	)
	if err != nil {
		f.t.Fatalf("apply product discount: %v", err)
	}
	if !applied.DiscountActive ||
		applied.RegularPriceAmount != 1000 ||
		applied.SalePriceAmount != 850 ||
		applied.DiscountAmount != 150 ||
		applied.DiscountBPS != 1500 {
		f.t.Fatalf("unexpected applied product discount: %+v", applied)
	}

	cleared, err := f.adminService.ClearProductDiscount(
		ctx,
		variantID,
		f.metadata(),
	)
	if err != nil {
		f.t.Fatalf("clear product discount: %v", err)
	}
	if cleared.DiscountActive ||
		cleared.RegularPriceAmount != 1000 ||
		cleared.SalePriceAmount != 1000 ||
		cleared.DiscountAmount != 0 {
		f.t.Fatalf("unexpected cleared product discount: %+v", cleared)
	}
}

func (f *promotionDBFixture) cleanup() {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	exec := func(label, query string, args ...any) {
		if _, err := f.db.Exec(ctx, query, args...); err != nil {
			f.t.Errorf("promotion DB cleanup %s: %v", label, err)
		}
	}

	exec(
		"audit events",
		`DELETE FROM admin_security_events WHERE request_id = $1`,
		f.requestID,
	)

	for index := len(f.promotionIDs) - 1; index >= 0; index-- {
		exec(
			"promotion",
			`DELETE FROM promotions WHERE id = $1::uuid`,
			f.promotionIDs[index],
		)
	}

	for index := len(f.productIDs) - 1; index >= 0; index-- {
		exec(
			"product",
			`DELETE FROM products WHERE id = $1::uuid`,
			f.productIDs[index],
		)
	}

	for index := len(f.categoryIDs) - 1; index >= 0; index-- {
		exec(
			"category",
			`DELETE FROM categories WHERE id = $1::uuid`,
			f.categoryIDs[index],
		)
	}
}
