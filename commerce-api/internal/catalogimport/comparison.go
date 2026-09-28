package catalogimport

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type productSnapshot struct {
	ProductCode      string
	Name             string
	CategoryPath     string
	Slug             string
	Brand            string
	ShortDescription string
	Description      string
	Status           string
	IsFeatured       bool
}

type variantSnapshot struct {
	SKU                  string
	ProductCode          string
	ColorName            string
	ColorHex             string
	Size                 string
	MinimumOrderQuantity int
	OrderIncrement       int
	PriceAmount          int64
	CompareAtPriceAmount *int64
	CostAmount           *int64
	Currency             string
	Barcode              string
	WeightGrams          *int
	IsActive             bool
	Stock                int
	ReorderLevel         int
	InventoryExists      bool
}

func loadExistingProductSnapshots(
	ctx context.Context,
	tx pgx.Tx,
	productCodes []string,
) (map[string]productSnapshot, error) {
	result := make(
		map[string]productSnapshot,
	)

	if len(productCodes) == 0 {
		return result, nil
	}

	const query = `
		WITH RECURSIVE category_paths AS (
			SELECT
				id,
				parent_id,
				name,
				name::text AS path
			FROM categories
			WHERE parent_id IS NULL

			UNION ALL

			SELECT
				c.id,
				c.parent_id,
				c.name,
				cp.path || ' > ' || c.name
			FROM categories c
			JOIN category_paths cp
				ON cp.id = c.parent_id
		)
		SELECT
			UPPER(TRIM(p.product_code)),
			p.name,
			cp.path,
			p.slug,
			COALESCE(p.brand, ''),
			COALESCE(p.short_description, ''),
			COALESCE(p.description, ''),
			p.status,
			p.is_featured
		FROM products p
		JOIN category_paths cp
			ON cp.id = p.category_id
		WHERE
			UPPER(TRIM(p.product_code))
			= ANY($1::text[])
	`

	rows, err := tx.Query(
		ctx,
		query,
		productCodes,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var snapshot productSnapshot

		if err := rows.Scan(
			&snapshot.ProductCode,
			&snapshot.Name,
			&snapshot.CategoryPath,
			&snapshot.Slug,
			&snapshot.Brand,
			&snapshot.ShortDescription,
			&snapshot.Description,
			&snapshot.Status,
			&snapshot.IsFeatured,
		); err != nil {
			return nil, err
		}

		result[normalizeIdentifier(
			snapshot.ProductCode,
		)] = snapshot
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func loadExistingVariantSnapshots(
	ctx context.Context,
	tx pgx.Tx,
	skus []string,
) (map[string]variantSnapshot, error) {
	result := make(
		map[string]variantSnapshot,
	)

	if len(skus) == 0 {
		return result, nil
	}

	const query = `
		SELECT
			UPPER(TRIM(v.sku)),
			UPPER(TRIM(p.product_code)),
			COALESCE(v.color_name, ''),
			COALESCE(v.color_hex, ''),
			COALESCE(v.size, ''),
			v.minimum_order_quantity,
			v.order_increment,
			v.price_amount,
			v.compare_at_price_amount,
			v.cost_amount,
			v.currency,
			COALESCE(v.barcode, ''),
			v.weight_grams,
			v.is_active,
			COALESCE(i.quantity_on_hand, 0),
			COALESCE(i.reorder_level, 0),
			(i.variant_id IS NOT NULL)
		FROM product_variants v
		JOIN products p
			ON p.id = v.product_id
		LEFT JOIN inventory i
			ON i.variant_id = v.id
		WHERE
			UPPER(TRIM(v.sku))
			= ANY($1::text[])
	`

	rows, err := tx.Query(
		ctx,
		query,
		skus,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var snapshot variantSnapshot

		var compareAt pgtype.Int8
		var cost pgtype.Int8
		var weight pgtype.Int4

		if err := rows.Scan(
			&snapshot.SKU,
			&snapshot.ProductCode,
			&snapshot.ColorName,
			&snapshot.ColorHex,
			&snapshot.Size,
			&snapshot.MinimumOrderQuantity,
			&snapshot.OrderIncrement,
			&snapshot.PriceAmount,
			&compareAt,
			&cost,
			&snapshot.Currency,
			&snapshot.Barcode,
			&weight,
			&snapshot.IsActive,
			&snapshot.Stock,
			&snapshot.ReorderLevel,
			&snapshot.InventoryExists,
		); err != nil {
			return nil, err
		}

		if compareAt.Valid {
			value := compareAt.Int64
			snapshot.CompareAtPriceAmount = &value
		}

		if cost.Valid {
			value := cost.Int64
			snapshot.CostAmount = &value
		}

		if weight.Valid {
			value := int(weight.Int32)
			snapshot.WeightGrams = &value
		}

		result[normalizeIdentifier(
			snapshot.SKU,
		)] = snapshot
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func loadExistingPriceTierSnapshots(
	ctx context.Context,
	tx pgx.Tx,
	skus []string,
) (map[string]int64, error) {
	result := make(
		map[string]int64,
	)

	if len(skus) == 0 {
		return result, nil
	}

	const query = `
		SELECT
			UPPER(TRIM(v.sku)),
			t.min_quantity,
			t.unit_price_amount
		FROM product_variant_price_tiers t
		JOIN product_variants v
			ON v.id = t.variant_id
		WHERE
			UPPER(TRIM(v.sku))
			= ANY($1::text[])
	`

	rows, err := tx.Query(
		ctx,
		query,
		skus,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var sku string
		var minQuantity int
		var unitPriceAmount int64

		if err := rows.Scan(
			&sku,
			&minQuantity,
			&unitPriceAmount,
		); err != nil {
			return nil, err
		}

		result[priceTierKey(
			sku,
			minQuantity,
		)] = unitPriceAmount
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func productRowChanged(
	row ProductRow,
	existing productSnapshot,
) bool {
	if cleanText(row.ProductName) !=
		cleanText(existing.Name) {
		return true
	}

	if categoryPathKey(row.CategoryPath) !=
		categoryPathKey(existing.CategoryPath) {
		return true
	}

	// Blank slug means "preserve/generate automatically".
	// It should not force an update to an existing product.
	if strings.TrimSpace(row.Slug) != "" &&
		normalizeSlug(row.Slug) !=
			normalizeSlug(existing.Slug) {
		return true
	}

	if cleanText(row.Brand) !=
		cleanText(existing.Brand) {
		return true
	}

	if cleanText(row.ShortDescription) !=
		cleanText(existing.ShortDescription) {
		return true
	}

	if cleanText(row.Description) !=
		cleanText(existing.Description) {
		return true
	}

	if !strings.EqualFold(
		strings.TrimSpace(existing.Status),
		strings.TrimSpace(row.Status),
	) {
		return true
	}

	if row.IsFeatured != existing.IsFeatured {
		return true
	}

	return false
}

func variantRowChanged(
	row VariantRow,
	existing variantSnapshot,
) bool {
	if normalizeIdentifier(row.ProductCode) !=
		normalizeIdentifier(existing.ProductCode) {
		return true
	}

	if cleanText(row.ColorName) !=
		cleanText(existing.ColorName) {
		return true
	}

	if !strings.EqualFold(
		strings.TrimSpace(existing.ColorHex),
		strings.TrimSpace(row.ColorHex),
	) {
		return true
	}

	if cleanText(row.Size) !=
		cleanText(existing.Size) {
		return true
	}

	if row.MinimumOrderQuantity !=
		existing.MinimumOrderQuantity {
		return true
	}

	if row.OrderIncrement !=
		existing.OrderIncrement {
		return true
	}

	if row.PriceAmount !=
		existing.PriceAmount {
		return true
	}

	if !equalOptionalInt64(
		row.CompareAtPriceAmount,
		existing.CompareAtPriceAmount,
	) {
		return true
	}

	if !equalOptionalInt64(
		row.CostAmount,
		existing.CostAmount,
	) {
		return true
	}

	if !strings.EqualFold(
		strings.TrimSpace(existing.Currency),
		strings.TrimSpace(row.Currency),
	) {
		return true
	}

	if cleanText(row.Barcode) !=
		cleanText(existing.Barcode) {
		return true
	}

	if !equalOptionalInt(
		row.WeightGrams,
		existing.WeightGrams,
	) {
		return true
	}

	if row.IsActive != existing.IsActive {
		return true
	}

	// A missing inventory row must be repaired later
	// even when the requested stock is zero.
	if !existing.InventoryExists {
		return true
	}

	if row.Stock != existing.Stock {
		return true
	}

	if row.ReorderLevel != existing.ReorderLevel {
		return true
	}

	return false
}

func equalOptionalInt64(
	left *int64,
	right *int64,
) bool {
	if left == nil && right == nil {
		return true
	}

	if left == nil || right == nil {
		return false
	}

	return *left == *right
}

func equalOptionalInt(
	left *int,
	right *int,
) bool {
	if left == nil && right == nil {
		return true
	}

	if left == nil || right == nil {
		return false
	}

	return *left == *right
}

func cleanText(
	value string,
) string {
	return strings.TrimSpace(value)
}

func normalizeSlug(
	value string,
) string {
	return strings.ToLower(
		strings.TrimSpace(value),
	)
}
