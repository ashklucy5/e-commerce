package catalog

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles PostgreSQL operations for the catalog.
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a catalog repository.
func NewRepository(
	db *pgxpool.Pool,
) *Repository {
	return &Repository{
		db: db,
	}
}

// ListActiveProducts returns lightweight cards for
// customer-visible products.
//
// Only active products with at least one active variant
// are returned.
func (r *Repository) ListActiveProducts(
	ctx context.Context,
) ([]ProductCard, error) {
	const query = `
		SELECT
			p.id,
			p.product_code,
			p.name,
			p.slug,
			p.brand,
			p.short_description,
			p.is_featured,

			primary_image.url,

			price_variant.price_amount,
			price_variant.currency,

			EXISTS (
				SELECT 1
				FROM product_variants stock_variant
				JOIN inventory i
					ON i.variant_id = stock_variant.id
				WHERE
					stock_variant.product_id = p.id
					AND stock_variant.is_active = true
					AND (
						i.quantity_on_hand
						- i.quantity_reserved
					) > 0
			) AS in_stock

		FROM products p

		JOIN LATERAL (
			SELECT
				v.price_amount,
				v.currency
			FROM product_variants v
			WHERE
				v.product_id = p.id
				AND v.is_active = true
			ORDER BY
				v.price_amount ASC,
				v.id ASC
			LIMIT 1
		) AS price_variant
			ON true

		LEFT JOIN LATERAL (
			SELECT pi.url
			FROM product_images pi
			WHERE
				pi.product_id = p.id
				AND pi.is_primary = true
			LIMIT 1
		) AS primary_image
			ON true

		WHERE p.status = 'active'

		ORDER BY
			p.is_featured DESC,
			p.published_at DESC NULLS LAST,
			p.created_at DESC
	`

	rows, err := r.db.Query(
		ctx,
		query,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"query active products: %w",
			err,
		)
	}
	defer rows.Close()

	products := make(
		[]ProductCard,
		0,
	)

	for rows.Next() {
		var item ProductCard

		err := rows.Scan(
			&item.ID,
			&item.ProductCode,
			&item.Name,
			&item.Slug,
			&item.Brand,
			&item.ShortDescription,
			&item.IsFeatured,
			&item.PrimaryImageURL,
			&item.PriceAmount,
			&item.Currency,
			&item.InStock,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"scan product card: %w",
				err,
			)
		}

		products = append(
			products,
			item,
		)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"iterate products: %w",
			err,
		)
	}

	return products, nil
}

// FindActiveProductBySlug returns the core product and its
// category. Images and variants are loaded separately.
func (r *Repository) FindActiveProductBySlug(
	ctx context.Context,
	slug string,
) (ProductDetail, error) {
	const query = `
		SELECT
			p.id,
			p.category_id,
			p.product_code,
			p.name,
			p.slug,
			p.brand,
			p.short_description,
			p.description,
			p.is_featured,
			p.published_at,
			p.created_at,
			p.updated_at,

			c.id,
			c.name,
			c.slug

		FROM products p
		JOIN categories c
			ON c.id = p.category_id

		WHERE
			p.slug = $1
			AND p.status = 'active'
			AND c.is_active = true

		LIMIT 1
	`

	var item ProductDetail

	err := r.db.QueryRow(
		ctx,
		query,
		slug,
	).Scan(
		&item.ID,
		&item.CategoryID,
		&item.ProductCode,
		&item.Name,
		&item.Slug,
		&item.Brand,
		&item.ShortDescription,
		&item.Description,
		&item.IsFeatured,
		&item.PublishedAt,
		&item.CreatedAt,
		&item.UpdatedAt,
		&item.Category.ID,
		&item.Category.Name,
		&item.Category.Slug,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return ProductDetail{}, pgx.ErrNoRows
		}

		return ProductDetail{}, fmt.Errorf(
			"find active product by slug: %w",
			err,
		)
	}

	images, err := r.listProductImages(
		ctx,
		item.ID,
	)
	if err != nil {
		return ProductDetail{}, err
	}

	variants, err := r.listActiveVariants(
		ctx,
		item.ID,
	)
	if err != nil {
		return ProductDetail{}, err
	}

	item.Images = images
	item.Variants = variants

	return item, nil
}

// listProductImages returns all images belonging to a product.
func (r *Repository) listProductImages(
	ctx context.Context,
	productID string,
) ([]ProductImage, error) {
	const query = `
		SELECT
			id,
			variant_id,
			url,
			alt_text,
			sort_order,
			is_primary
		FROM product_images
		WHERE product_id = $1
		ORDER BY
			is_primary DESC,
			sort_order ASC,
			id ASC
	`

	rows, err := r.db.Query(
		ctx,
		query,
		productID,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"query product images: %w",
			err,
		)
	}
	defer rows.Close()

	images := make(
		[]ProductImage,
		0,
	)

	for rows.Next() {
		var image ProductImage

		err := rows.Scan(
			&image.ID,
			&image.VariantID,
			&image.URL,
			&image.AltText,
			&image.SortOrder,
			&image.IsPrimary,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"scan product image: %w",
				err,
			)
		}

		images = append(
			images,
			image,
		)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"iterate product images: %w",
			err,
		)
	}

	return images, nil
}

// listActiveVariants returns all active sellable variants
// together with available inventory and B2B pricing rules.
func (r *Repository) listActiveVariants(
	ctx context.Context,
	productID string,
) ([]ProductVariant, error) {
	const query = `
		SELECT
			v.id,
			v.sku,
			v.color_name,
			v.color_hex,
			v.size,
			v.minimum_order_quantity,
			v.order_increment,
			v.price_amount,
			v.compare_at_price_amount,
			v.currency,
			v.weight_grams,

			COALESCE(
				i.quantity_on_hand
				- i.quantity_reserved,
				0
			) AS available_quantity

		FROM product_variants v

		LEFT JOIN inventory i
			ON i.variant_id = v.id

		WHERE
			v.product_id = $1
			AND v.is_active = true

		ORDER BY
			v.price_amount ASC,
			v.color_name ASC NULLS LAST,
			v.size ASC NULLS LAST,
			v.id ASC
	`

	rows, err := r.db.Query(
		ctx,
		query,
		productID,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"query product variants: %w",
			err,
		)
	}
	defer rows.Close()

	variants := make(
		[]ProductVariant,
		0,
	)

	for rows.Next() {
		var item ProductVariant

		err := rows.Scan(
			&item.ID,
			&item.SKU,
			&item.ColorName,
			&item.ColorHex,
			&item.Size,
			&item.MinimumOrderQuantity,
			&item.OrderIncrement,
			&item.PriceAmount,
			&item.CompareAtPriceAmount,
			&item.Currency,
			&item.WeightGrams,
			&item.AvailableQuantity,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"scan product variant: %w",
				err,
			)
		}

		item.InStock =
			item.AvailableQuantity > 0

		variants = append(
			variants,
			item,
		)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"iterate product variants: %w",
			err,
		)
	}

	// We have finished consuming the variant rows.
	rows.Close()

	priceTiers, err := r.listPriceTiers(
		ctx,
		productID,
	)
	if err != nil {
		return nil, err
	}

	for i := range variants {
		tiers, exists := priceTiers[variants[i].ID]

		if !exists {
			// Return [] instead of null in JSON when the
			// variant has no wholesale price tiers.
			variants[i].PriceTiers = make(
				[]PriceTier,
				0,
			)

			continue
		}

		variants[i].PriceTiers = tiers
	}

	return variants, nil
}

// listPriceTiers loads wholesale quantity-based prices
// for all active variants belonging to a product.
func (r *Repository) listPriceTiers(
	ctx context.Context,
	productID string,
) (map[string][]PriceTier, error) {
	const query = `
		SELECT
			pt.variant_id,
			pt.min_quantity,
			pt.unit_price_amount

		FROM product_variant_price_tiers pt

		JOIN product_variants v
			ON v.id = pt.variant_id

		WHERE
			v.product_id = $1
			AND v.is_active = true

		ORDER BY
			pt.variant_id,
			pt.min_quantity ASC
	`

	rows, err := r.db.Query(
		ctx,
		query,
		productID,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"query product price tiers: %w",
			err,
		)
	}
	defer rows.Close()

	priceTiers := make(
		map[string][]PriceTier,
	)

	for rows.Next() {
		var variantID string
		var tier PriceTier

		err := rows.Scan(
			&variantID,
			&tier.MinQuantity,
			&tier.UnitPriceAmount,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"scan product price tier: %w",
				err,
			)
		}

		priceTiers[variantID] = append(
			priceTiers[variantID],
			tier,
		)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"iterate product price tiers: %w",
			err,
		)
	}

	return priceTiers, nil
}
