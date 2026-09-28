package seeds

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SeedProducts creates development B2B catalog data.
//
// It is intentionally idempotent:
// - product is identified by product_code
// - variants are identified by SKU
// - inventory is upserted
// - price tiers are replaced for the seeded variants
// - product images are replaced
func SeedProducts(
	ctx context.Context,
	db *pgxpool.Pool,
) error {
	tx, err := db.Begin(ctx)
	if err != nil {
		return fmt.Errorf(
			"begin product seed transaction: %w",
			err,
		)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	// Our category seeder already creates:
	//
	// Men
	//   └── Shirts
	//
	// Products must reference an existing category.
	var categoryID string

	err = tx.QueryRow(
		ctx,
		`
		SELECT child.id
		FROM categories child
		JOIN categories parent
			ON parent.id = child.parent_id
		WHERE
			lower(trim(parent.name)) = 'men'
			AND lower(trim(child.name)) = 'shirts'
			AND parent.is_active = true
			AND child.is_active = true
		LIMIT 1
	`,
	).Scan(&categoryID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return fmt.Errorf(
				"shirts category not found; run category seed first",
			)
		}

		return fmt.Errorf(
			"find shirts category: %w",
			err,
		)
	}

	// ------------------------------------------------------------
	// Product
	// ------------------------------------------------------------

	var productID string

	err = tx.QueryRow(
		ctx,
		`
			INSERT INTO products (
				category_id,
				product_code,
				name,
				slug,
				brand,
				short_description,
				description,
				status,
				is_featured,
				published_at,
				created_at,
				updated_at
			)
			VALUES (
				$1,
				'P10001',
				'Premium Oxford Shirt',
				'premium-oxford-shirt',
				'StyleNest',
				'Wholesale premium Oxford shirt for business buyers.',
				'Premium Oxford shirt suitable for retail stores, corporate buyers, resellers, and wholesale customers.',
				'active',
				true,
				now(),
				now(),
				now()
			)
			ON CONFLICT (product_code)
			DO UPDATE SET
				category_id = EXCLUDED.category_id,
				name = EXCLUDED.name,
				slug = EXCLUDED.slug,
				brand = EXCLUDED.brand,
				short_description = EXCLUDED.short_description,
				description = EXCLUDED.description,
				status = EXCLUDED.status,
				is_featured = EXCLUDED.is_featured,
				published_at = EXCLUDED.published_at,
				updated_at = now()
			RETURNING id
		`,
		categoryID,
	).Scan(&productID)
	if err != nil {
		return fmt.Errorf(
			"upsert premium oxford shirt: %w",
			err,
		)
	}

	type variantSeed struct {
		SKU                  string
		ColorName            string
		ColorHex             string
		Size                 string
		MinimumOrderQuantity int
		OrderIncrement       int
		PriceAmount          int64
		CompareAtPriceAmount int64
		CostAmount           int64
		WeightGrams          int
		QuantityOnHand       int
		ReorderLevel         int
		PriceTiers           []priceTierSeed
	}

	variants := []variantSeed{
		{
			SKU:                  "OXF-BLK-M",
			ColorName:            "Black",
			ColorHex:             "#000000",
			Size:                 "M",
			MinimumOrderQuantity: 12,
			OrderIncrement:       6,
			PriceAmount:          149000,
			CompareAtPriceAmount: 169000,
			CostAmount:           90000,
			WeightGrams:          300,
			QuantityOnHand:       120,
			ReorderLevel:         24,
			PriceTiers: []priceTierSeed{
				{
					MinQuantity:     50,
					UnitPriceAmount: 139000,
				},
				{
					MinQuantity:     100,
					UnitPriceAmount: 129000,
				},
			},
		},
		{
			SKU:                  "OXF-BLK-L",
			ColorName:            "Black",
			ColorHex:             "#000000",
			Size:                 "L",
			MinimumOrderQuantity: 12,
			OrderIncrement:       6,
			PriceAmount:          149000,
			CompareAtPriceAmount: 169000,
			CostAmount:           90000,
			WeightGrams:          310,
			QuantityOnHand:       80,
			ReorderLevel:         24,
			PriceTiers: []priceTierSeed{
				{
					MinQuantity:     50,
					UnitPriceAmount: 139000,
				},
				{
					MinQuantity:     100,
					UnitPriceAmount: 129000,
				},
			},
		},
		{
			SKU:                  "OXF-WHT-M",
			ColorName:            "White",
			ColorHex:             "#FFFFFF",
			Size:                 "M",
			MinimumOrderQuantity: 12,
			OrderIncrement:       6,
			PriceAmount:          159000,
			CompareAtPriceAmount: 179000,
			CostAmount:           95000,
			WeightGrams:          300,
			QuantityOnHand:       70,
			ReorderLevel:         24,
			PriceTiers: []priceTierSeed{
				{
					MinQuantity:     50,
					UnitPriceAmount: 149000,
				},
				{
					MinQuantity:     100,
					UnitPriceAmount: 139000,
				},
			},
		},
		{
			SKU:                  "OXF-WHT-L",
			ColorName:            "White",
			ColorHex:             "#FFFFFF",
			Size:                 "L",
			MinimumOrderQuantity: 12,
			OrderIncrement:       6,
			PriceAmount:          159000,
			CompareAtPriceAmount: 179000,
			CostAmount:           95000,
			WeightGrams:          310,
			QuantityOnHand:       0,
			ReorderLevel:         24,
			PriceTiers: []priceTierSeed{
				{
					MinQuantity:     50,
					UnitPriceAmount: 149000,
				},
				{
					MinQuantity:     100,
					UnitPriceAmount: 139000,
				},
			},
		},
	}

	for _, variant := range variants {
		var variantID string

		err := tx.QueryRow(
			ctx,
			`
				INSERT INTO product_variants (
					product_id,
					sku,
					color_name,
					color_hex,
					size,
					minimum_order_quantity,
					order_increment,
					price_amount,
					compare_at_price_amount,
					cost_amount,
					currency,
					weight_grams,
					is_active,
					created_at,
					updated_at
				)
				VALUES (
					$1,
					$2,
					$3,
					$4,
					$5,
					$6,
					$7,
					$8,
					$9,
					$10,
					'BDT',
					$11,
					true,
					now(),
					now()
				)
				ON CONFLICT (sku)
				DO UPDATE SET
					product_id = EXCLUDED.product_id,
					color_name = EXCLUDED.color_name,
					color_hex = EXCLUDED.color_hex,
					size = EXCLUDED.size,
					minimum_order_quantity =
						EXCLUDED.minimum_order_quantity,
					order_increment =
						EXCLUDED.order_increment,
					price_amount =
						EXCLUDED.price_amount,
					compare_at_price_amount =
						EXCLUDED.compare_at_price_amount,
					cost_amount =
						EXCLUDED.cost_amount,
					currency =
						EXCLUDED.currency,
					weight_grams =
						EXCLUDED.weight_grams,
					is_active = true,
					updated_at = now()
				RETURNING id
			`,
			productID,
			variant.SKU,
			variant.ColorName,
			variant.ColorHex,
			variant.Size,
			variant.MinimumOrderQuantity,
			variant.OrderIncrement,
			variant.PriceAmount,
			variant.CompareAtPriceAmount,
			variant.CostAmount,
			variant.WeightGrams,
		).Scan(&variantID)
		if err != nil {
			return fmt.Errorf(
				"upsert variant %s: %w",
				variant.SKU,
				err,
			)
		}

		// Inventory is reset to deterministic development values
		// whenever this seed runs.
		_, err = tx.Exec(
			ctx,
			`
				INSERT INTO inventory (
					variant_id,
					quantity_on_hand,
					quantity_reserved,
					reorder_level,
					updated_at
				)
				VALUES (
					$1,
					$2,
					0,
					$3,
					now()
				)
				ON CONFLICT (variant_id)
				DO UPDATE SET
					quantity_on_hand =
						EXCLUDED.quantity_on_hand,
					quantity_reserved = 0,
					reorder_level =
						EXCLUDED.reorder_level,
					updated_at = now()
			`,
			variantID,
			variant.QuantityOnHand,
			variant.ReorderLevel,
		)
		if err != nil {
			return fmt.Errorf(
				"upsert inventory for %s: %w",
				variant.SKU,
				err,
			)
		}

		// Replace this variant's wholesale tiers so re-running
		// the seed doesn't produce duplicates or stale tiers.
		_, err = tx.Exec(
			ctx,
			`
				DELETE FROM product_variant_price_tiers
				WHERE variant_id = $1
			`,
			variantID,
		)
		if err != nil {
			return fmt.Errorf(
				"delete existing price tiers for %s: %w",
				variant.SKU,
				err,
			)
		}

		for _, tier := range variant.PriceTiers {
			_, err = tx.Exec(
				ctx,
				`
					INSERT INTO product_variant_price_tiers (
						variant_id,
						min_quantity,
						unit_price_amount,
						created_at,
						updated_at
					)
					VALUES (
						$1,
						$2,
						$3,
						now(),
						now()
					)
				`,
				variantID,
				tier.MinQuantity,
				tier.UnitPriceAmount,
			)
			if err != nil {
				return fmt.Errorf(
					"insert price tier for %s: %w",
					variant.SKU,
					err,
				)
			}
		}
	}

	// Replace product-level images to keep the development
	// seed deterministic when it is run multiple times.
	_, err = tx.Exec(
		ctx,
		`
			DELETE FROM product_images
			WHERE product_id = $1
		`,
		productID,
	)
	if err != nil {
		return fmt.Errorf(
			"delete existing product images: %w",
			err,
		)
	}

	images := []struct {
		URL       string
		AltText   string
		SortOrder int
		IsPrimary bool
	}{
		{
			URL:       "https://placehold.co/1200x1500/png?text=Oxford+Shirt+Front",
			AltText:   "Premium Oxford Shirt front view",
			SortOrder: 0,
			IsPrimary: true,
		},
		{
			URL:       "https://placehold.co/1200x1500/png?text=Oxford+Shirt+Back",
			AltText:   "Premium Oxford Shirt back view",
			SortOrder: 1,
			IsPrimary: false,
		},
		{
			URL:       "https://placehold.co/1200x1500/png?text=Oxford+Shirt+Detail",
			AltText:   "Premium Oxford Shirt fabric detail",
			SortOrder: 2,
			IsPrimary: false,
		},
	}

	for _, image := range images {
		_, err = tx.Exec(
			ctx,
			`
				INSERT INTO product_images (
					product_id,
					variant_id,
					url,
					alt_text,
					sort_order,
					is_primary,
					created_at
				)
				VALUES (
					$1,
					NULL,
					$2,
					$3,
					$4,
					$5,
					now()
				)
			`,
			productID,
			image.URL,
			image.AltText,
			image.SortOrder,
			image.IsPrimary,
		)
		if err != nil {
			return fmt.Errorf(
				"insert product image: %w",
				err,
			)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf(
			"commit product seed transaction: %w",
			err,
		)
	}

	return nil
}

type priceTierSeed struct {
	MinQuantity     int
	UnitPriceAmount int64
}
