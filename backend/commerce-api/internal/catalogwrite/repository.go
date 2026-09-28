package catalogwrite

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(
	db *pgxpool.Pool,
) *Repository {
	return &Repository{
		db: db,
	}
}

func (r *Repository) CreateProduct(
	ctx context.Context,
	input ProductInput,
) (ProductResult, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return ProductResult{},
			fmt.Errorf(
				"begin catalog write transaction: %w",
				err,
			)
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	categoryID, err :=
		resolveProductCategoryID(
			ctx,
			tx,
			input,
		)
	if err != nil {
		return ProductResult{}, err
	}

	productCode := strings.TrimSpace(
		input.ProductCode,
	)

	if productCode == "" {
		productCode, err =
			allocateProductCode(
				ctx,
				tx,
				categoryID,
			)
		if err != nil {
			return ProductResult{}, err
		}
	}

	slug := strings.TrimSpace(
		input.Slug,
	)

	if slug == "" {
		slug = slugify(
			input.Name,
		)

		if slug == "" {
			slug = slugify(
				productCode,
			)
		}

		slug, err = uniqueProductSlug(
			ctx,
			tx,
			slug,
		)
		if err != nil {
			return ProductResult{}, err
		}
	}

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
				$2,
				$3,
				$4,
				NULLIF($5, ''),
				NULLIF($6, ''),
				NULLIF($7, ''),
				$8::varchar(30),
				$9,
				CASE
					WHEN $8::varchar(30) = 'active'
						THEN now()
					ELSE NULL
				END,
				now(),
				now()
			)
			RETURNING id::text
		`,
		categoryID,
		productCode,
		input.Name,
		slug,
		input.Brand,
		input.ShortDescription,
		input.Description,
		input.Status,
		input.IsFeatured,
	).Scan(
		&productID,
	)
	if err != nil {
		return ProductResult{},
			fmt.Errorf(
				"create product %q: %w",
				productCode,
				err,
			)
	}

	result := ProductResult{
		ID:          productID,
		ProductCode: productCode,
		Slug:        slug,
		Variants: make(
			[]VariantResult,
			0,
			len(input.Variants),
		),
	}

	for index, variant := range input.Variants {

		sku := strings.TrimSpace(
			variant.SKU,
		)

		if sku == "" {
			sku, err =
				allocateVariantSKU(
					ctx,
					tx,
					productCode,
					productID,
					index+1,
				)
			if err != nil {
				return ProductResult{},
					err
			}
		}

		var variantID string

		err = tx.QueryRow(
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
					barcode,
					weight_grams,
					is_active,
					created_at,
					updated_at
				)
				VALUES (
					$1,
					$2,
					NULLIF($3, ''),
					NULLIF($4, ''),
					NULLIF($5, ''),
					$6,
					$7,
					$8,
					$9,
					$10,
					$11,
					NULLIF($12, ''),
					$13,
					$14,
					now(),
					now()
				)
				RETURNING id::text
			`,
			productID,
			sku,
			variant.ColorName,
			variant.ColorHex,
			variant.Size,
			variant.MinimumOrderQuantity,
			variant.OrderIncrement,
			variant.PriceAmount,
			variant.CompareAtPriceAmount,
			variant.CostAmount,
			variant.Currency,
			variant.Barcode,
			variant.WeightGrams,
			variant.IsActive,
		).Scan(
			&variantID,
		)
		if err != nil {
			return ProductResult{},
				fmt.Errorf(
					"create SKU %q: %w",
					sku,
					err,
				)
		}

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
			`,
			variantID,
			variant.Stock,
			variant.ReorderLevel,
		)
		if err != nil {
			return ProductResult{},
				fmt.Errorf(
					"create inventory for SKU %q: %w",
					sku,
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
				return ProductResult{},
					fmt.Errorf(
						"create price tier for SKU %q at quantity %d: %w",
						sku,
						tier.MinQuantity,
						err,
					)
			}
		}

		result.Variants = append(
			result.Variants,
			VariantResult{
				ID:  variantID,
				SKU: sku,
			},
		)
	}

	if err := tx.Commit(ctx); err != nil {
		return ProductResult{},
			fmt.Errorf(
				"commit catalog write transaction: %w",
				err,
			)
	}

	return result, nil
}

func allocateProductCode(
	ctx context.Context,
	tx pgx.Tx,
	categoryID string,
) (string, error) {
	var prefix string
	var number int64

	err := tx.QueryRow(
		ctx,
		`
			UPDATE product_code_namespaces
			SET
				next_number = next_number + 1,
				updated_at = now()
			WHERE category_id = $1
			RETURNING
				prefix,
				next_number - 1
		`,
		categoryID,
	).Scan(
		&prefix,
		&number,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "",
				fmt.Errorf(
					"product-code namespace is not configured for category %s",
					categoryID,
				)
		}

		return "",
			fmt.Errorf(
				"allocate product code: %w",
				err,
			)
	}

	if number <= 0 ||
		number > 999999 {
		return "",
			fmt.Errorf(
				"product-code sequence exhausted for namespace %s",
				prefix,
			)
	}

	return fmt.Sprintf(
		"%s-%06d",
		prefix,
		number,
	), nil
}
