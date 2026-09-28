package catalogimport

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"unicode"

	"github.com/jackc/pgx/v5"
)

type ApplyResult struct {
	BatchID     string `json:"batch_id"`
	Status      string `json:"status"`
	AppliedRows int    `json:"applied_rows"`
	SkippedRows int    `json:"skipped_rows"`
}

type applyRow struct {
	ID        string
	SheetName string
	RowNumber int
	Action    string
	RawData   []byte
}

func (s *Service) Apply(
	ctx context.Context,
	batchID string,
) (*ApplyResult, error) {
	batchID = strings.TrimSpace(batchID)

	if batchID == "" {
		return nil, fmt.Errorf(
			"batch ID is required",
		)
	}

	result, err :=
		s.repository.ApplyReadyBatch(
			ctx,
			batchID,
		)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func (r *Repository) ApplyReadyBatch(
	ctx context.Context,
	batchID string,
) (ApplyResult, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return ApplyResult{},
			fmt.Errorf(
				"begin import apply transaction: %w",
				err,
			)
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var status string

	const lockBatchQuery = `
		SELECT status
		FROM catalog_import_batches
		WHERE id = $1
		FOR UPDATE
	`

	err = tx.QueryRow(
		ctx,
		lockBatchQuery,
		batchID,
	).Scan(
		&status,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return ApplyResult{},
				fmt.Errorf(
					"catalog import batch %s not found",
					batchID,
				)
		}

		return ApplyResult{},
			fmt.Errorf(
				"lock catalog import batch: %w",
				err,
			)
	}

	if status != "ready" {
		return ApplyResult{},
			fmt.Errorf(
				"catalog import batch must be ready; current status is %q",
				status,
			)
	}

	var invalidRows int

	err = tx.QueryRow(
		ctx,
		`
			SELECT COUNT(*)
			FROM catalog_import_rows
			WHERE
				batch_id = $1
				AND status = 'invalid'
		`,
		batchID,
	).Scan(
		&invalidRows,
	)
	if err != nil {
		return ApplyResult{},
			fmt.Errorf(
				"check invalid import rows: %w",
				err,
			)
	}

	if invalidRows > 0 {
		return ApplyResult{},
			fmt.Errorf(
				"catalog import batch contains %d invalid row(s)",
				invalidRows,
			)
	}

	_, err = tx.Exec(
		ctx,
		`
			UPDATE catalog_import_batches
			SET
				status = 'applying',
				updated_at = now()
			WHERE id = $1
		`,
		batchID,
	)
	if err != nil {
		return ApplyResult{},
			fmt.Errorf(
				"mark catalog import batch applying: %w",
				err,
			)
	}

	rows, err := loadRowsForApply(
		ctx,
		tx,
		batchID,
	)
	if err != nil {
		return ApplyResult{}, err
	}

	result := ApplyResult{
		BatchID: batchID,
		Status:  "applying",
	}

	for _, row := range rows {
		if row.Action == "" ||
			row.Action == "skip" {

			if err := markImportRow(
				ctx,
				tx,
				row.ID,
				"skipped",
			); err != nil {
				return ApplyResult{}, err
			}

			result.SkippedRows++

			continue
		}

		switch row.SheetName {
		case sheetCategories:
			var parsed CategoryRow

			if err := json.Unmarshal(
				row.RawData,
				&parsed,
			); err != nil {
				return ApplyResult{},
					fmt.Errorf(
						"decode Categories row %d: %w",
						row.RowNumber,
						err,
					)
			}

			if row.Action == "create" {
				if err := applyCategoryCreate(
					ctx,
					tx,
					parsed,
				); err != nil {
					return ApplyResult{},
						fmt.Errorf(
							"apply Categories row %d: %w",
							row.RowNumber,
							err,
						)
				}
			}

		case sheetProducts:
			var parsed ProductRow

			if err := json.Unmarshal(
				row.RawData,
				&parsed,
			); err != nil {
				return ApplyResult{},
					fmt.Errorf(
						"decode Products row %d: %w",
						row.RowNumber,
						err,
					)
			}

			if row.Action == "create" {
				if err := applyProductCreate(
					ctx,
					tx,
					parsed,
				); err != nil {
					return ApplyResult{},
						fmt.Errorf(
							"apply Products row %d: %w",
							row.RowNumber,
							err,
						)
				}
			} else {
				if err := applyProductUpdate(
					ctx,
					tx,
					parsed,
				); err != nil {
					return ApplyResult{},
						fmt.Errorf(
							"apply Products row %d: %w",
							row.RowNumber,
							err,
						)
				}
			}

		case sheetVariants:
			var parsed VariantRow

			if err := json.Unmarshal(
				row.RawData,
				&parsed,
			); err != nil {
				return ApplyResult{},
					fmt.Errorf(
						"decode Variants row %d: %w",
						row.RowNumber,
						err,
					)
			}

			if row.Action == "create" {
				if err := applyVariantCreate(
					ctx,
					tx,
					parsed,
				); err != nil {
					return ApplyResult{},
						fmt.Errorf(
							"apply Variants row %d: %w",
							row.RowNumber,
							err,
						)
				}
			} else {
				if err := applyVariantUpdate(
					ctx,
					tx,
					parsed,
				); err != nil {
					return ApplyResult{},
						fmt.Errorf(
							"apply Variants row %d: %w",
							row.RowNumber,
							err,
						)
				}
			}

		case sheetPriceTiers:
			var parsed PriceTierRow

			if err := json.Unmarshal(
				row.RawData,
				&parsed,
			); err != nil {
				return ApplyResult{},
					fmt.Errorf(
						"decode PriceTiers row %d: %w",
						row.RowNumber,
						err,
					)
			}

			if err := applyPriceTier(
				ctx,
				tx,
				parsed,
			); err != nil {
				return ApplyResult{},
					fmt.Errorf(
						"apply PriceTiers row %d: %w",
						row.RowNumber,
						err,
					)
			}

		case sheetImages:
			var parsed ImageRow

			if err := json.Unmarshal(
				row.RawData,
				&parsed,
			); err != nil {
				return ApplyResult{},
					fmt.Errorf(
						"decode Images row %d: %w",
						row.RowNumber,
						err,
					)
			}

			if strings.TrimSpace(parsed.ImageURL) == "" {
				if err := markImportRow(
					ctx,
					tx,
					row.ID,
					"skipped",
				); err != nil {
					return ApplyResult{}, err
				}

				result.SkippedRows++
				continue
			}

			if err := applyImageURL(
				ctx,
				tx,
				parsed,
			); err != nil {
				return ApplyResult{},
					fmt.Errorf(
						"apply Images row %d: %w",
						row.RowNumber,
						err,
					)
			}

		default:
			return ApplyResult{},
				fmt.Errorf(
					"unsupported import sheet %q",
					row.SheetName,
				)
		}

		if err := markImportRow(
			ctx,
			tx,
			row.ID,
			"applied",
		); err != nil {
			return ApplyResult{}, err
		}

		result.AppliedRows++
	}

	_, err = tx.Exec(
		ctx,
		`
			UPDATE catalog_import_batches
			SET
				status = 'completed',
				completed_at = now(),
				updated_at = now()
			WHERE id = $1
		`,
		batchID,
	)
	if err != nil {
		return ApplyResult{},
			fmt.Errorf(
				"complete catalog import batch: %w",
				err,
			)
	}

	if err := tx.Commit(ctx); err != nil {
		return ApplyResult{},
			fmt.Errorf(
				"commit catalog import apply transaction: %w",
				err,
			)
	}

	result.Status = "completed"

	return result, nil
}

func loadRowsForApply(
	ctx context.Context,
	tx pgx.Tx,
	batchID string,
) ([]applyRow, error) {
	const query = `
		SELECT
			id::text,
			sheet_name,
			row_number,
			COALESCE(action, ''),
			raw_data
		FROM catalog_import_rows
		WHERE
			batch_id = $1
			AND status = 'valid'
		ORDER BY
			CASE sheet_name
				WHEN 'Categories' THEN 1
				WHEN 'Products' THEN 2
				WHEN 'Variants' THEN 3
				WHEN 'PriceTiers' THEN 4
				WHEN 'Images' THEN 5
				ELSE 99
			END,
			row_number
	`

	rows, err := tx.Query(
		ctx,
		query,
		batchID,
	)
	if err != nil {
		return nil,
			fmt.Errorf(
				"load staged import rows: %w",
				err,
			)
	}

	defer rows.Close()

	result := make(
		[]applyRow,
		0,
	)

	for rows.Next() {
		var row applyRow

		if err := rows.Scan(
			&row.ID,
			&row.SheetName,
			&row.RowNumber,
			&row.Action,
			&row.RawData,
		); err != nil {
			return nil,
				fmt.Errorf(
					"scan staged import row: %w",
					err,
				)
		}

		result = append(
			result,
			row,
		)
	}

	if err := rows.Err(); err != nil {
		return nil,
			fmt.Errorf(
				"iterate staged import rows: %w",
				err,
			)
	}

	return result, nil
}

func applyCategoryCreate(
	ctx context.Context,
	tx pgx.Tx,
	row CategoryRow,
) error {
	categoryID, err := ensureCategoryPath(
		ctx,
		tx,
		row.CategoryPath,
	)
	if err != nil {
		return err
	}

	const query = `
		UPDATE categories
		SET
			description =
				NULLIF(trim($2::text), ''),
			image_url =
				NULLIF(trim($3::text), ''),
			icon_url =
				NULLIF(trim($4::text), ''),
			sort_order = $5,
			is_active = $6,
			updated_at = now()
		WHERE id = $1
	`

	tag, err := tx.Exec(
		ctx,
		query,
		categoryID,
		row.Description,
		row.ImageURL,
		row.IconURL,
		row.SortOrder,
		row.IsActive,
	)
	if err != nil {
		return fmt.Errorf(
			"update created category metadata: %w",
			err,
		)
	}

	if tag.RowsAffected() != 1 {
		return fmt.Errorf(
			"created category %q could not be updated",
			row.CategoryPath,
		)
	}

	return nil
}

func applyProductCreate(
	ctx context.Context,
	tx pgx.Tx,
	row ProductRow,
) error {
	categoryID, err := ensureCategoryPath(
		ctx,
		tx,
		row.CategoryPath,
	)
	if err != nil {
		return err
	}

	slug := strings.TrimSpace(
		row.Slug,
	)

	if slug == "" {
		slug = slugify(
			row.ProductName,
		)

		if slug == "" {
			slug = slugify(
				row.ProductCode,
			)
		}

		slug, err = uniqueProductSlug(
			ctx,
			tx,
			slug,
		)
		if err != nil {
			return err
		}
	}

	const query = `
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
			trim($2::text),
			trim($3::text),
			trim($4::text),
			NULLIF(trim($5::text), ''),
			NULLIF(trim($6::text), ''),
			NULLIF(trim($7::text), ''),
			lower(trim($8::text)),
			$9,
			CASE
				WHEN lower(trim($8::text)) = 'active'
					THEN now()
				ELSE NULL
			END,
			now(),
			now()
		)
	`

	_, err = tx.Exec(
		ctx,
		query,
		categoryID,
		row.ProductCode,
		row.ProductName,
		slug,
		row.Brand,
		row.ShortDescription,
		row.Description,
		row.Status,
		row.IsFeatured,
	)
	if err != nil {
		return fmt.Errorf(
			"create product_code %q: %w",
			row.ProductCode,
			err,
		)
	}

	return nil
}

func applyProductUpdate(
	ctx context.Context,
	tx pgx.Tx,
	row ProductRow,
) error {
	categoryID, err := findCategoryIDByPath(
		ctx,
		tx,
		row.CategoryPath,
	)
	if err != nil {
		return err
	}

	const query = `
		UPDATE products
		SET
			category_id = $2,
			name = $3,
			slug = CASE
				WHEN trim($4::text) = ''
					THEN slug
				ELSE trim($4::text)
			END,
			brand = NULLIF(trim($5::text), ''),
			short_description =
				NULLIF(trim($6::text), ''),
			description =
				NULLIF(trim($7::text), ''),
			status = lower(trim($8::text)),
			is_featured = $9,
			published_at = CASE
				WHEN
					lower(trim($8::text)) = 'active'
					AND published_at IS NULL
				THEN now()
				ELSE published_at
			END,
			updated_at = now()
		WHERE product_code = $1
	`

	tag, err := tx.Exec(
		ctx,
		query,
		row.ProductCode,
		categoryID,
		row.ProductName,
		row.Slug,
		row.Brand,
		row.ShortDescription,
		row.Description,
		row.Status,
		row.IsFeatured,
	)
	if err != nil {
		return err
	}

	if tag.RowsAffected() != 1 {
		return fmt.Errorf(
			"product_code %q was not found",
			row.ProductCode,
		)
	}

	return nil
}

func applyVariantCreate(
	ctx context.Context,
	tx pgx.Tx,
	row VariantRow,
) error {
	var productID string

	err := tx.QueryRow(
		ctx,
		`
			SELECT id::text
			FROM products
			WHERE product_code = $1
		`,
		strings.TrimSpace(
			row.ProductCode,
		),
	).Scan(
		&productID,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return fmt.Errorf(
				"product_code %q does not exist for SKU %q",
				row.ProductCode,
				row.SKU,
			)
		}

		return fmt.Errorf(
			"find product for SKU %q: %w",
			row.SKU,
			err,
		)
	}

	var variantID string

	const insertVariantQuery = `
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
			trim($2::text),
			NULLIF(trim($3::text), ''),
			NULLIF(trim($4::text), ''),
			NULLIF(trim($5::text), ''),
			$6,
			$7,
			$8,
			$9,
			$10,
			upper(trim($11::text)),
			NULLIF(trim($12::text), ''),
			$13,
			$14,
			now(),
			now()
		)
		RETURNING id::text
	`

	err = tx.QueryRow(
		ctx,
		insertVariantQuery,
		productID,
		row.SKU,
		row.ColorName,
		row.ColorHex,
		row.Size,
		row.MinimumOrderQuantity,
		row.OrderIncrement,
		row.PriceAmount,
		row.CompareAtPriceAmount,
		row.CostAmount,
		row.Currency,
		row.Barcode,
		row.WeightGrams,
		row.IsActive,
	).Scan(
		&variantID,
	)
	if err != nil {
		return fmt.Errorf(
			"create SKU %q: %w",
			row.SKU,
			err,
		)
	}

	const inventoryQuery = `
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
	`

	_, err = tx.Exec(
		ctx,
		inventoryQuery,
		variantID,
		row.Stock,
		row.ReorderLevel,
	)
	if err != nil {
		return fmt.Errorf(
			"create inventory for SKU %q: %w",
			row.SKU,
			err,
		)
	}

	return nil
}

func applyVariantUpdate(
	ctx context.Context,
	tx pgx.Tx,
	row VariantRow,
) error {
	const findVariantQuery = `
		SELECT
			v.id::text,
			p.product_code
		FROM product_variants v
		JOIN products p
			ON p.id = v.product_id
		WHERE v.sku = $1
		FOR UPDATE OF v
	`

	var variantID string
	var currentProductCode string

	err := tx.QueryRow(
		ctx,
		findVariantQuery,
		row.SKU,
	).Scan(
		&variantID,
		&currentProductCode,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return fmt.Errorf(
				"SKU %q was not found",
				row.SKU,
			)
		}

		return err
	}

	if normalizeIdentifier(
		currentProductCode,
	) != normalizeIdentifier(
		row.ProductCode,
	) {
		return fmt.Errorf(
			"moving SKU %q between products is not supported",
			row.SKU,
		)
	}

	const updateVariantQuery = `
		UPDATE product_variants
		SET
			color_name =
				NULLIF(trim($2::text), ''),
			color_hex =
				NULLIF(trim($3::text), ''),
			size =
				NULLIF(trim($4::text), ''),
			minimum_order_quantity = $5,
			order_increment = $6,
			price_amount = $7,
			compare_at_price_amount = $8,
			cost_amount = $9,
			currency = upper(trim($10::text)),
			barcode =
				NULLIF(trim($11::text), ''),
			weight_grams = $12,
			is_active = $13,
			updated_at = now()
		WHERE id = $1
	`

	tag, err := tx.Exec(
		ctx,
		updateVariantQuery,
		variantID,
		row.ColorName,
		row.ColorHex,
		row.Size,
		row.MinimumOrderQuantity,
		row.OrderIncrement,
		row.PriceAmount,
		row.CompareAtPriceAmount,
		row.CostAmount,
		row.Currency,
		row.Barcode,
		row.WeightGrams,
		row.IsActive,
	)
	if err != nil {
		return err
	}

	if tag.RowsAffected() != 1 {
		return fmt.Errorf(
			"failed to update SKU %q",
			row.SKU,
		)
	}

	const inventoryQuery = `
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
			reorder_level =
				EXCLUDED.reorder_level,
			updated_at = now()
		WHERE
			inventory.quantity_reserved
			<= EXCLUDED.quantity_on_hand
	`

	tag, err = tx.Exec(
		ctx,
		inventoryQuery,
		variantID,
		row.Stock,
		row.ReorderLevel,
	)
	if err != nil {
		return err
	}

	if tag.RowsAffected() != 1 {
		return fmt.Errorf(
			"cannot set stock for SKU %q below its currently reserved quantity",
			row.SKU,
		)
	}

	return nil
}

func applyPriceTier(
	ctx context.Context,
	tx pgx.Tx,
	row PriceTierRow,
) error {
	var variantID string

	err := tx.QueryRow(
		ctx,
		`
			SELECT id::text
			FROM product_variants
			WHERE sku = $1
		`,
		row.SKU,
	).Scan(
		&variantID,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return fmt.Errorf(
				"price tier SKU %q was not found",
				row.SKU,
			)
		}

		return err
	}

	_, err = tx.Exec(
		ctx,
		`
			INSERT INTO product_variant_price_tiers (
				variant_id,
				min_quantity,
				unit_price_amount,
				updated_at
			)
			VALUES (
				$1,
				$2,
				$3,
				now()
			)
			ON CONFLICT (
				variant_id,
				min_quantity
			)
			DO UPDATE SET
				unit_price_amount =
					EXCLUDED.unit_price_amount,
				updated_at = now()
		`,
		variantID,
		row.MinQuantity,
		row.UnitPriceAmount,
	)
	if err != nil {
		return err
	}

	return nil
}

func ensureCategoryPath(
	ctx context.Context,
	tx pgx.Tx,
	path string,
) (string, error) {
	parts := splitCategoryPath(
		path,
	)

	if len(parts) == 0 {
		return "",
			fmt.Errorf(
				"category path is empty",
			)
	}

	var parentID *string
	var currentID string

	for index, name := range parts {
		existingID, found, err :=
			findCategoryChild(
				ctx,
				tx,
				parentID,
				name,
			)
		if err != nil {
			return "", err
		}

		if found {
			currentID = existingID
			parentID = &currentID

			continue
		}

		fullPath := strings.Join(
			parts[:index+1],
			" > ",
		)

		slugBase := slugify(
			strings.Join(
				parts[:index+1],
				"-",
			),
		)

		if slugBase == "" {
			slugBase = "category"
		}

		slug, err :=
			uniqueCategorySlug(
				ctx,
				tx,
				slugBase,
			)
		if err != nil {
			return "", err
		}

		err = tx.QueryRow(
			ctx,
			`
				INSERT INTO categories (
					parent_id,
					name,
					slug,
					sort_order,
					is_active,
					created_at,
					updated_at
				)
				VALUES (
					$1,
					$2,
					$3,
					0,
					true,
					now(),
					now()
				)
				RETURNING id::text
			`,
			parentID,
			name,
			slug,
		).Scan(
			&currentID,
		)
		if err != nil {
			return "",
				fmt.Errorf(
					"create category %q: %w",
					fullPath,
					err,
				)
		}

		parentID = &currentID
	}

	return currentID, nil
}

func findCategoryChild(
	ctx context.Context,
	tx pgx.Tx,
	parentID *string,
	name string,
) (string, bool, error) {
	var id string
	var err error

	if parentID == nil {
		err = tx.QueryRow(
			ctx,
			`
				SELECT id::text
				FROM categories
				WHERE
					parent_id IS NULL
					AND lower(trim(name)) =
						lower(trim($1::text))
				LIMIT 1
			`,
			name,
		).Scan(
			&id,
		)
	} else {
		err = tx.QueryRow(
			ctx,
			`
				SELECT id::text
				FROM categories
				WHERE
					parent_id = $1
					AND lower(trim(name)) =
						lower(trim($2::text))
				LIMIT 1
			`,
			*parentID,
			name,
		).Scan(
			&id,
		)
	}

	if err == pgx.ErrNoRows {
		return "", false, nil
	}

	if err != nil {
		return "",
			false,
			fmt.Errorf(
				"find category %q: %w",
				name,
				err,
			)
	}

	return id, true, nil
}

func findCategoryIDByPath(
	ctx context.Context,
	tx pgx.Tx,
	path string,
) (string, error) {
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
		SELECT id::text
		FROM category_paths
		WHERE lower(trim(path)) =
			lower(trim($1::text))
		LIMIT 1
	`

	var categoryID string

	err := tx.QueryRow(
		ctx,
		query,
		normalizeCategoryPath(path),
	).Scan(
		&categoryID,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "",
				fmt.Errorf(
					"category path %q does not exist",
					path,
				)
		}

		return "", err
	}

	return categoryID, nil
}

func splitCategoryPath(
	path string,
) []string {
	rawParts := strings.Split(
		path,
		">",
	)

	parts := make(
		[]string,
		0,
		len(rawParts),
	)

	for _, raw := range rawParts {
		part := strings.TrimSpace(
			raw,
		)

		if part == "" {
			continue
		}

		parts = append(
			parts,
			part,
		)
	}

	return parts
}

func uniqueCategorySlug(
	ctx context.Context,
	tx pgx.Tx,
	base string,
) (string, error) {
	base = strings.Trim(
		base,
		"-",
	)

	if base == "" {
		base = "category"
	}

	for suffix := 1; suffix <= 10000; suffix++ {
		candidate := base

		if suffix > 1 {
			candidate = fmt.Sprintf(
				"%s-%d",
				base,
				suffix,
			)
		}

		var exists bool

		err := tx.QueryRow(
			ctx,
			`
				SELECT EXISTS (
					SELECT 1
					FROM categories
					WHERE slug = $1
				)
			`,
			candidate,
		).Scan(
			&exists,
		)
		if err != nil {
			return "",
				fmt.Errorf(
					"check category slug %q: %w",
					candidate,
					err,
				)
		}

		if !exists {
			return candidate, nil
		}
	}

	return "",
		fmt.Errorf(
			"could not generate a unique category slug for %q",
			base,
		)
}

func uniqueProductSlug(
	ctx context.Context,
	tx pgx.Tx,
	base string,
) (string, error) {
	base = strings.Trim(
		base,
		"-",
	)

	if base == "" {
		base = "product"
	}

	for suffix := 1; suffix <= 10000; suffix++ {
		candidate := base

		if suffix > 1 {
			candidate = fmt.Sprintf(
				"%s-%d",
				base,
				suffix,
			)
		}

		var exists bool

		err := tx.QueryRow(
			ctx,
			`
				SELECT EXISTS (
					SELECT 1
					FROM products
					WHERE slug = $1
				)
			`,
			candidate,
		).Scan(
			&exists,
		)
		if err != nil {
			return "",
				fmt.Errorf(
					"check product slug %q: %w",
					candidate,
					err,
				)
		}

		if !exists {
			return candidate, nil
		}
	}

	return "",
		fmt.Errorf(
			"could not generate a unique product slug for %q",
			base,
		)
}

func slugify(
	value string,
) string {
	value = strings.ToLower(
		strings.TrimSpace(value),
	)

	var builder strings.Builder
	needsDash := false

	for _, r := range value {
		if unicode.IsLetter(r) ||
			unicode.IsDigit(r) {

			if needsDash &&
				builder.Len() > 0 {

				builder.WriteByte('-')
			}

			builder.WriteRune(r)
			needsDash = false

			continue
		}

		if builder.Len() > 0 {
			needsDash = true
		}
	}

	return strings.Trim(
		builder.String(),
		"-",
	)
}

func applyImageURL(
	ctx context.Context,
	tx pgx.Tx,
	row ImageRow,
) error {
	var productID string

	err := tx.QueryRow(
		ctx,
		`
			SELECT id::text
			FROM products
			WHERE UPPER(TRIM(product_code)) = UPPER(TRIM($1::text))
		`,
		row.ProductCode,
	).Scan(
		&productID,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return fmt.Errorf(
				"product_code %q was not found",
				row.ProductCode,
			)
		}

		return fmt.Errorf(
			"resolve product %q: %w",
			row.ProductCode,
			err,
		)
	}

	variantID := ""

	if strings.TrimSpace(row.SKU) != "" {
		err = tx.QueryRow(
			ctx,
			`
				SELECT id::text
				FROM product_variants
				WHERE
					product_id = $1
					AND UPPER(TRIM(sku)) = UPPER(TRIM($2::text))
			`,
			productID,
			row.SKU,
		).Scan(
			&variantID,
		)
		if err != nil {
			if err == pgx.ErrNoRows {
				return fmt.Errorf(
					"SKU %q was not found for product_code %q",
					row.SKU,
					row.ProductCode,
				)
			}

			return fmt.Errorf(
				"resolve SKU %q: %w",
				row.SKU,
				err,
			)
		}
	}

	if row.IsPrimary && variantID != "" {
		return fmt.Errorf(
			"variant image %q cannot be the primary product image",
			row.SKU,
		)
	}

	if row.IsPrimary {
		_, err = tx.Exec(
			ctx,
			`
				UPDATE product_images
				SET is_primary = false
				WHERE
					product_id = $1
					AND is_primary = true
			`,
			productID,
		)
		if err != nil {
			return fmt.Errorf(
				"clear existing primary image: %w",
				err,
			)
		}
	}

	imageURL := strings.TrimSpace(row.ImageURL)

	tag, err := tx.Exec(
		ctx,
		`
			UPDATE product_images
			SET
				variant_id = NULLIF($2::text, '')::uuid,
				alt_text = NULLIF(TRIM($4::text), ''),
				sort_order = $5,
				is_primary = $6
			WHERE
				product_id = $1
				AND url = $3
				AND (
					(
						variant_id IS NULL
						AND NULLIF($2::text, '')::uuid IS NULL
					)
					OR variant_id = NULLIF($2::text, '')::uuid
				)
		`,
		productID,
		variantID,
		imageURL,
		row.AltText,
		row.SortOrder,
		row.IsPrimary,
	)
	if err != nil {
		return fmt.Errorf(
			"update product image: %w",
			err,
		)
	}

	if tag.RowsAffected() > 0 {
		return nil
	}

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
				NULLIF($2::text, '')::uuid,
				$3,
				NULLIF(TRIM($4::text), ''),
				$5,
				$6,
				now()
			)
		`,
		productID,
		variantID,
		imageURL,
		row.AltText,
		row.SortOrder,
		row.IsPrimary,
	)
	if err != nil {
		return fmt.Errorf(
			"insert product image: %w",
			err,
		)
	}

	return nil
}

func markImportRow(
	ctx context.Context,
	tx pgx.Tx,
	rowID string,
	status string,
) error {
	_, err := tx.Exec(
		ctx,
		`
			UPDATE catalog_import_rows
			SET
				status = $2,
				updated_at = now()
			WHERE id = $1
		`,
		rowID,
		status,
	)
	if err != nil {
		return fmt.Errorf(
			"mark import row %s: %w",
			status,
			err,
		)
	}

	return nil
}
