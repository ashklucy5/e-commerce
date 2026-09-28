package catalogimport

import (
	"context"
	"encoding/json"
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

func (r *Repository) CreateBatch(
	ctx context.Context,
	sourceFilename string,
	storageKey string,
	checksum string,
) (BatchSummary, error) {
	const query = `
		INSERT INTO catalog_import_batches (
			source_filename,
			file_storage_key,
			file_checksum_sha256,
			status,
			started_at
		)
		VALUES (
			$1,
			$2,
			NULLIF($3, ''),
			'parsing',
			now()
		)
		RETURNING
			id,
			status,
			total_rows,
			valid_rows,
			failed_rows,
			created_products,
			updated_products,
			created_variants,
			updated_variants,
			created_categories
	`

	var batch BatchSummary

	err := r.db.QueryRow(
		ctx,
		query,
		sourceFilename,
		storageKey,
		checksum,
	).Scan(
		&batch.ID,
		&batch.Status,
		&batch.TotalRows,
		&batch.ValidRows,
		&batch.FailedRows,
		&batch.CreatedProducts,
		&batch.UpdatedProducts,
		&batch.CreatedVariants,
		&batch.UpdatedVariants,
		&batch.CreatedCategories,
	)
	if err != nil {
		return BatchSummary{},
			fmt.Errorf(
				"create catalog import batch: %w",
				err,
			)
	}

	return batch, nil
}

func (r *Repository) SaveValidation(
	ctx context.Context,
	batchID string,
	result *ParseResult,
) (BatchSummary, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return BatchSummary{},
			fmt.Errorf(
				"begin catalog import staging transaction: %w",
				err,
			)
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	plan, err := r.buildActionPlan(
		ctx,
		tx,
		result,
	)
	if err != nil {
		return BatchSummary{},
			fmt.Errorf(
				"build import action plan: %w",
				err,
			)
	}

	errorsByRow := groupErrorsByRow(
		result.Errors,
	)

	rows := collectStagedRows(
		result,
	)

	failedRows := 0

	for _, row := range rows {
		rowErrors := errorsByRow[rowKey(row.Source)]

		status := "valid"
		action := ""

		if len(rowErrors) > 0 {
			status = "invalid"
			failedRows++
		} else {
			action = plan.Actions[rowKey(row.Source)]
		}

		rawJSON, err := json.Marshal(
			row.Raw,
		)
		if err != nil {
			return BatchSummary{},
				fmt.Errorf(
					"marshal %s row %d: %w",
					row.Source.Sheet,
					row.Source.Row,
					err,
				)
		}

		var errorJSON *string

		if len(rowErrors) > 0 {
			encodedErrors, err := json.Marshal(
				rowErrors,
			)
			if err != nil {
				return BatchSummary{},
					fmt.Errorf(
						"marshal errors for %s row %d: %w",
						row.Source.Sheet,
						row.Source.Row,
						err,
					)
			}

			value := string(encodedErrors)
			errorJSON = &value
		}

		const insertRowQuery = `
			INSERT INTO catalog_import_rows (
				batch_id,
				sheet_name,
				row_number,
				product_code,
				sku,
				action,
				status,
				raw_data,
				error_details
			)
			VALUES (
				$1,
				$2,
				$3,
				NULLIF($4, ''),
				NULLIF($5, ''),
				NULLIF($6, ''),
				$7,
				$8::jsonb,
				$9::jsonb
			)
		`

		_, err = tx.Exec(
			ctx,
			insertRowQuery,
			batchID,
			row.Source.Sheet,
			row.Source.Row,
			row.ProductCode,
			row.SKU,
			action,
			status,
			string(rawJSON),
			errorJSON,
		)
		if err != nil {
			return BatchSummary{},
				fmt.Errorf(
					"insert %s row %d: %w",
					row.Source.Sheet,
					row.Source.Row,
					err,
				)
		}
	}

	totalRows := len(rows)
	validRows := totalRows - failedRows

	batchStatus := "ready"

	var lastError *string

	if len(result.Errors) > 0 {
		batchStatus = "failed"

		message := fmt.Sprintf(
			"validation failed with %d error(s)",
			len(result.Errors),
		)

		lastError = &message
	}

	const updateBatchQuery = `
		UPDATE catalog_import_batches
		SET
			status = $2::varchar(30),
			total_rows = $3,
			valid_rows = $4,
			failed_rows = $5,
			last_error = $6,
			created_products = $7,
			updated_products = $8,
			created_variants = $9,
			updated_variants = $10,
			created_categories = $11,
			completed_at = CASE
				WHEN $2::varchar(30) = 'failed'
					THEN now()
				ELSE NULL
			END,
			updated_at = now()
		WHERE id = $1
		RETURNING
			id,
			status,
			total_rows,
			valid_rows,
			failed_rows,
			created_products,
			updated_products,
			created_variants,
			updated_variants,
			created_categories
	`

	var batch BatchSummary

	err = tx.QueryRow(
		ctx,
		updateBatchQuery,
		batchID,
		batchStatus,
		totalRows,
		validRows,
		failedRows,
		lastError,
		plan.CreatedProducts,
		plan.UpdatedProducts,
		plan.CreatedVariants,
		plan.UpdatedVariants,
		plan.CreatedCategories,
	).Scan(
		&batch.ID,
		&batch.Status,
		&batch.TotalRows,
		&batch.ValidRows,
		&batch.FailedRows,
		&batch.CreatedProducts,
		&batch.UpdatedProducts,
		&batch.CreatedVariants,
		&batch.UpdatedVariants,
		&batch.CreatedCategories,
	)
	if err != nil {
		return BatchSummary{},
			fmt.Errorf(
				"update catalog import batch: %w",
				err,
			)
	}

	if err := tx.Commit(ctx); err != nil {
		return BatchSummary{},
			fmt.Errorf(
				"commit catalog import staging transaction: %w",
				err,
			)
	}

	return batch, nil
}

func (r *Repository) FailBatch(
	ctx context.Context,
	batchID string,
	message string,
) error {
	const query = `
		UPDATE catalog_import_batches
		SET
			status = 'failed',
			last_error = $2,
			completed_at = now(),
			updated_at = now()
		WHERE id = $1
	`

	_, err := r.db.Exec(
		ctx,
		query,
		batchID,
		message,
	)
	if err != nil {
		return fmt.Errorf(
			"mark catalog import batch failed: %w",
			err,
		)
	}

	return nil
}

type stagedRow struct {
	Source      RowSource
	ProductCode string
	SKU         string
	Raw         any
}

type actionPlan struct {
	Actions           map[string]string
	CreatedProducts   int
	UpdatedProducts   int
	CreatedVariants   int
	UpdatedVariants   int
	CreatedCategories int
}

func (r *Repository) buildActionPlan(
	ctx context.Context,
	tx pgx.Tx,
	result *ParseResult,
) (actionPlan, error) {
	plan := actionPlan{
		Actions: make(map[string]string),
	}

	invalidRows := make(map[string]struct{})

	for _, validationError := range result.Errors {
		key := fmt.Sprintf(
			"%s:%d",
			validationError.Sheet,
			validationError.Row,
		)

		invalidRows[key] = struct{}{}
	}

	productCodes := make(
		[]string,
		0,
		len(result.Workbook.Products)+len(result.Workbook.Images),
	)

	for _, row := range result.Workbook.Products {
		if rowIsInvalid(
			invalidRows,
			row.Source,
		) {
			continue
		}

		code := normalizeIdentifier(
			row.ProductCode,
		)

		if code != "" {
			productCodes = append(
				productCodes,
				code,
			)
		}
	}

	for _, row := range result.Workbook.Images {
		if rowIsInvalid(
			invalidRows,
			row.Source,
		) {
			continue
		}

		code := normalizeIdentifier(
			row.ProductCode,
		)

		if code != "" {
			productCodes = append(
				productCodes,
				code,
			)
		}
	}

	skus := make(
		[]string,
		0,
		len(result.Workbook.Variants)+len(result.Workbook.Images),
	)

	for _, row := range result.Workbook.Variants {
		if rowIsInvalid(
			invalidRows,
			row.Source,
		) {
			continue
		}

		sku := normalizeIdentifier(
			row.SKU,
		)

		if sku != "" {
			skus = append(
				skus,
				sku,
			)
		}
	}

	for _, row := range result.Workbook.Images {
		if rowIsInvalid(
			invalidRows,
			row.Source,
		) {
			continue
		}

		sku := normalizeIdentifier(
			row.SKU,
		)

		if sku != "" {
			skus = append(
				skus,
				sku,
			)
		}
	}

	productCodes = uniqueStrings(
		productCodes,
	)

	skus = uniqueStrings(
		skus,
	)

	existingProducts, err :=
		loadExistingProductSnapshots(
			ctx,
			tx,
			productCodes,
		)
	if err != nil {
		return actionPlan{},
			fmt.Errorf(
				"load existing products: %w",
				err,
			)
	}

	existingVariants, err :=
		loadExistingVariantSnapshots(
			ctx,
			tx,
			skus,
		)
	if err != nil {
		return actionPlan{},
			fmt.Errorf(
				"load existing variants: %w",
				err,
			)
	}

	existingPriceTiers, err :=
		loadExistingPriceTierSnapshots(
			ctx,
			tx,
			skus,
		)
	if err != nil {
		return actionPlan{},
			fmt.Errorf(
				"load existing price tiers: %w",
				err,
			)
	}

	existingImages, err :=
		loadExistingImageSnapshots(
			ctx,
			tx,
			productCodes,
		)
	if err != nil {
		return actionPlan{},
			fmt.Errorf(
				"load existing images: %w",
				err,
			)
	}

	existingCategoryPaths, err :=
		loadExistingCategoryPaths(
			ctx,
			tx,
		)
	if err != nil {
		return actionPlan{},
			fmt.Errorf(
				"load existing categories: %w",
				err,
			)
	}

	for _, row := range result.Workbook.Products {
		if rowIsInvalid(
			invalidRows,
			row.Source,
		) {
			continue
		}

		code := normalizeIdentifier(
			row.ProductCode,
		)

		existing, exists :=
			existingProducts[code]

		if !exists {
			plan.Actions[rowKey(row.Source)] = "create"
			plan.CreatedProducts++
			continue
		}

		if productRowChanged(
			row,
			existing,
		) {
			plan.Actions[rowKey(row.Source)] = "update"
			plan.UpdatedProducts++
			continue
		}

		plan.Actions[rowKey(row.Source)] = "skip"
	}

	for _, row := range result.Workbook.Variants {
		if rowIsInvalid(
			invalidRows,
			row.Source,
		) {
			continue
		}

		sku := normalizeIdentifier(
			row.SKU,
		)

		existing, exists :=
			existingVariants[sku]

		if !exists {
			plan.Actions[rowKey(row.Source)] = "create"
			plan.CreatedVariants++
			continue
		}

		if variantRowChanged(
			row,
			existing,
		) {
			plan.Actions[rowKey(row.Source)] = "update"
			plan.UpdatedVariants++
			continue
		}

		plan.Actions[rowKey(row.Source)] = "skip"
	}

	for _, row := range result.Workbook.PriceTiers {
		if rowIsInvalid(
			invalidRows,
			row.Source,
		) {
			continue
		}

		key := priceTierKey(
			row.SKU,
			row.MinQuantity,
		)

		existingPrice, exists :=
			existingPriceTiers[key]

		if !exists {
			plan.Actions[rowKey(row.Source)] = "create"
			continue
		}

		if existingPrice != row.UnitPriceAmount {
			plan.Actions[rowKey(row.Source)] = "update"
			continue
		}

		plan.Actions[rowKey(row.Source)] = "skip"
	}

	// Images:
	// identity is product_code + optional SKU + image_url.
	// URL-backed images are created when missing, updated when metadata
	// changed, and skipped when the database already matches the workbook.
	// image_file rows remain skipped until archive/object-storage import is wired.
	for _, row := range result.Workbook.Images {
		if rowIsInvalid(
			invalidRows,
			row.Source,
		) {
			continue
		}

		imageURL := strings.TrimSpace(
			row.ImageURL,
		)

		if imageURL == "" {
			plan.Actions[rowKey(row.Source)] = "skip"
			continue
		}

		key := imageIdentityKey(
			row.ProductCode,
			row.SKU,
			imageURL,
		)

		existing, exists :=
			existingImages[key]

		if !exists {
			plan.Actions[rowKey(row.Source)] = "create"
			continue
		}

		if imageRowChanged(
			row,
			existing,
		) {
			plan.Actions[rowKey(row.Source)] = "update"
			continue
		}

		plan.Actions[rowKey(row.Source)] = "skip"
	}

	for _, row := range result.Workbook.Categories {
		if rowIsInvalid(
			invalidRows,
			row.Source,
		) {
			continue
		}

		key := categoryPathKey(
			row.CategoryPath,
		)

		if _, exists :=
			existingCategoryPaths[key]; exists {

			plan.Actions[rowKey(row.Source)] = "skip"
			continue
		}

		plan.Actions[rowKey(row.Source)] = "create"
	}

	plan.CreatedCategories =
		countMissingCategories(
			result,
			invalidRows,
			existingCategoryPaths,
		)

	return plan, nil
}

type imageSnapshot struct {
	ProductCode string
	SKU         string
	URL         string
	AltText     string
	SortOrder   int
	IsPrimary   bool
}

func loadExistingImageSnapshots(
	ctx context.Context,
	tx pgx.Tx,
	productCodes []string,
) (map[string]imageSnapshot, error) {
	result := make(
		map[string]imageSnapshot,
	)

	if len(productCodes) == 0 {
		return result, nil
	}

	const query = `
		SELECT
			UPPER(TRIM(p.product_code)),
			COALESCE(UPPER(TRIM(v.sku)), ''),
			TRIM(pi.url),
			COALESCE(pi.alt_text, ''),
			pi.sort_order,
			pi.is_primary
		FROM product_images pi
		JOIN products p
			ON p.id = pi.product_id
		LEFT JOIN product_variants v
			ON v.id = pi.variant_id
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
		var snapshot imageSnapshot

		if err := rows.Scan(
			&snapshot.ProductCode,
			&snapshot.SKU,
			&snapshot.URL,
			&snapshot.AltText,
			&snapshot.SortOrder,
			&snapshot.IsPrimary,
		); err != nil {
			return nil, err
		}

		result[imageIdentityKey(
			snapshot.ProductCode,
			snapshot.SKU,
			snapshot.URL,
		)] = snapshot
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func imageIdentityKey(
	productCode string,
	sku string,
	imageURL string,
) string {
	return fmt.Sprintf(
		"%s|%s|%s",
		normalizeIdentifier(productCode),
		normalizeIdentifier(sku),
		strings.TrimSpace(imageURL),
	)
}

func imageRowChanged(
	row ImageRow,
	existing imageSnapshot,
) bool {
	if normalizeIdentifier(row.ProductCode) !=
		normalizeIdentifier(existing.ProductCode) {
		return true
	}

	if normalizeIdentifier(row.SKU) !=
		normalizeIdentifier(existing.SKU) {
		return true
	}

	if strings.TrimSpace(row.ImageURL) !=
		strings.TrimSpace(existing.URL) {
		return true
	}

	if strings.TrimSpace(row.AltText) !=
		strings.TrimSpace(existing.AltText) {
		return true
	}

	if row.SortOrder != existing.SortOrder {
		return true
	}

	return row.IsPrimary != existing.IsPrimary
}

type categoryNode struct {
	ID       string
	ParentID *string
	Name     string
}

func loadExistingCategoryPaths(
	ctx context.Context,
	tx pgx.Tx,
) (map[string]struct{}, error) {
	const query = `
		SELECT
			id::text,
			parent_id::text,
			name
		FROM categories
	`

	rows, err := tx.Query(
		ctx,
		query,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	nodes := make(
		map[string]categoryNode,
	)

	for rows.Next() {
		var node categoryNode

		if err := rows.Scan(
			&node.ID,
			&node.ParentID,
			&node.Name,
		); err != nil {
			return nil, err
		}

		nodes[node.ID] = node
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	paths := make(
		map[string]struct{},
	)

	for id := range nodes {
		path, err :=
			buildCategoryPath(
				id,
				nodes,
			)
		if err != nil {
			return nil, err
		}

		if path == "" {
			continue
		}

		paths[categoryPathKey(path)] = struct{}{}
	}

	return paths, nil
}

func buildCategoryPath(
	id string,
	nodes map[string]categoryNode,
) (string, error) {
	segments := make(
		[]string,
		0,
		4,
	)

	visited := make(
		map[string]struct{},
	)

	currentID := id

	for currentID != "" {
		if _, exists :=
			visited[currentID]; exists {

			return "",
				fmt.Errorf(
					"category hierarchy cycle detected at %s",
					currentID,
				)
		}

		visited[currentID] = struct{}{}

		node, exists :=
			nodes[currentID]

		if !exists {
			return "",
				fmt.Errorf(
					"category %s references missing parent",
					currentID,
				)
		}

		name := strings.TrimSpace(
			node.Name,
		)

		if name != "" {
			segments = append(
				segments,
				name,
			)
		}

		if node.ParentID == nil {
			break
		}

		currentID = *node.ParentID
	}

	for left, right :=
		0, len(segments)-1; left < right; left, right =
		left+1, right-1 {

		segments[left],
			segments[right] =
			segments[right],
			segments[left]
	}

	return normalizeCategoryPath(
		strings.Join(
			segments,
			" > ",
		),
	), nil
}

func countMissingCategories(
	result *ParseResult,
	invalidRows map[string]struct{},
	existing map[string]struct{},
) int {
	planned := make(
		map[string]struct{},
	)

	addPath := func(path string) {
		for _, prefix := range categoryPathPrefixes(path) {
			key := categoryPathKey(
				prefix,
			)

			if key == "" {
				continue
			}

			if _, exists :=
				existing[key]; exists {
				continue
			}

			if _, exists :=
				planned[key]; exists {
				continue
			}

			planned[key] = struct{}{}
		}
	}

	for _, row := range result.Workbook.Products {
		if rowIsInvalid(
			invalidRows,
			row.Source,
		) {
			continue
		}

		addPath(
			row.CategoryPath,
		)
	}

	for _, row := range result.Workbook.Categories {
		if rowIsInvalid(
			invalidRows,
			row.Source,
		) {
			continue
		}

		addPath(
			row.CategoryPath,
		)
	}

	return len(planned)
}

func categoryPathPrefixes(
	path string,
) []string {
	parts := strings.Split(
		path,
		">",
	)

	prefixes := make(
		[]string,
		0,
		len(parts),
	)

	clean := make(
		[]string,
		0,
		len(parts),
	)

	for _, part := range parts {
		part = strings.TrimSpace(
			part,
		)

		if part == "" {
			continue
		}

		clean = append(
			clean,
			part,
		)

		prefixes = append(
			prefixes,
			normalizeCategoryPath(
				strings.Join(
					clean,
					" > ",
				),
			),
		)
	}

	return prefixes
}

func normalizeIdentifier(
	value string,
) string {
	return strings.ToUpper(
		strings.TrimSpace(value),
	)
}

func categoryPathKey(
	path string,
) string {
	return strings.ToLower(
		normalizeCategoryPath(
			path,
		),
	)
}

func priceTierKey(
	sku string,
	minQuantity int,
) string {
	return fmt.Sprintf(
		"%s:%d",
		normalizeIdentifier(sku),
		minQuantity,
	)
}

func uniqueStrings(
	values []string,
) []string {
	seen := make(
		map[string]struct{},
		len(values),
	)

	result := make(
		[]string,
		0,
		len(values),
	)

	for _, value := range values {
		value = strings.TrimSpace(
			value,
		)

		if value == "" {
			continue
		}

		if _, exists :=
			seen[value]; exists {
			continue
		}

		seen[value] = struct{}{}

		result = append(
			result,
			value,
		)
	}

	return result
}

func rowIsInvalid(
	invalidRows map[string]struct{},
	source RowSource,
) bool {
	_, exists :=
		invalidRows[rowKey(source)]

	return exists
}

func collectStagedRows(
	result *ParseResult,
) []stagedRow {
	total := len(result.Workbook.Products) +
		len(result.Workbook.Variants) +
		len(result.Workbook.Images) +
		len(result.Workbook.PriceTiers) +
		len(result.Workbook.Categories)

	rows := make(
		[]stagedRow,
		0,
		total,
	)

	for _, row := range result.Workbook.Products {
		rows = append(
			rows,
			stagedRow{
				Source:      row.Source,
				ProductCode: row.ProductCode,
				Raw:         row,
			},
		)
	}

	for _, row := range result.Workbook.Variants {
		rows = append(
			rows,
			stagedRow{
				Source:      row.Source,
				ProductCode: row.ProductCode,
				SKU:         row.SKU,
				Raw:         row,
			},
		)
	}

	for _, row := range result.Workbook.Images {
		rows = append(
			rows,
			stagedRow{
				Source:      row.Source,
				ProductCode: row.ProductCode,
				SKU:         row.SKU,
				Raw:         row,
			},
		)
	}

	for _, row := range result.Workbook.PriceTiers {
		rows = append(
			rows,
			stagedRow{
				Source: row.Source,
				SKU:    row.SKU,
				Raw:    row,
			},
		)
	}

	for _, row := range result.Workbook.Categories {
		rows = append(
			rows,
			stagedRow{
				Source: row.Source,
				Raw:    row,
			},
		)
	}

	return rows
}

func groupErrorsByRow(
	errors []ValidationError,
) map[string][]ValidationError {
	grouped := make(
		map[string][]ValidationError,
	)

	for _, validationError := range errors {
		key := fmt.Sprintf(
			"%s:%d",
			validationError.Sheet,
			validationError.Row,
		)

		grouped[key] = append(
			grouped[key],
			validationError,
		)
	}

	return grouped
}

func rowKey(
	source RowSource,
) string {
	return fmt.Sprintf(
		"%s:%d",
		source.Sheet,
		source.Row,
	)
}
