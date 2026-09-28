package catalogimport

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type BatchListItem struct {
	BatchSummary

	SourceFilename string    `json:"source_filename"`
	StorageKey     string    `json:"storage_key"`
	ChecksumSHA256 string    `json:"checksum_sha256,omitempty"`
	LastError      string    `json:"last_error,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type BatchListResult struct {
	Items  []BatchListItem `json:"items"`
	Limit  int             `json:"limit"`
	Offset int             `json:"offset"`
}

type RowPreview struct {
	ID          string            `json:"id"`
	SheetName   string            `json:"sheet_name"`
	RowNumber   int               `json:"row_number"`
	ProductCode string            `json:"product_code,omitempty"`
	SKU         string            `json:"sku,omitempty"`
	Action      string            `json:"action,omitempty"`
	Status      string            `json:"status"`
	RawData     json.RawMessage   `json:"raw_data"`
	Errors      []ValidationError `json:"errors,omitempty"`
}

type PreviewResult struct {
	Batch BatchSummary `json:"batch"`
	Rows  []RowPreview `json:"rows"`
}

func (s *Service) List(
	ctx context.Context,
	limit int,
	offset int,
) (*BatchListResult, error) {
	if limit <= 0 {
		limit = 50
	}

	if limit > 200 {
		limit = 200
	}

	if offset < 0 {
		offset = 0
	}

	items, err := s.repository.ListBatches(
		ctx,
		limit,
		offset,
	)
	if err != nil {
		return nil, err
	}

	return &BatchListResult{
		Items:  items,
		Limit:  limit,
		Offset: offset,
	}, nil
}

func (s *Service) Preview(
	ctx context.Context,
	batchID string,
	limit int,
	offset int,
) (*PreviewResult, error) {
	batchID = strings.TrimSpace(
		batchID,
	)

	if batchID == "" {
		return nil,
			fmt.Errorf(
				"batch ID is required",
			)
	}

	if limit <= 0 {
		limit = 100
	}

	if limit > 500 {
		limit = 500
	}

	if offset < 0 {
		offset = 0
	}

	batch, err := s.repository.GetBatch(
		ctx,
		batchID,
	)
	if err != nil {
		return nil, err
	}

	rows, err := s.repository.ListRows(
		ctx,
		batchID,
		limit,
		offset,
	)
	if err != nil {
		return nil, err
	}

	return &PreviewResult{
		Batch: batch,
		Rows:  rows,
	}, nil
}

func (r *Repository) ListBatches(
	ctx context.Context,
	limit int,
	offset int,
) ([]BatchListItem, error) {
	const query = `
		SELECT
			id::text,
			status,
			total_rows,
			valid_rows,
			failed_rows,
			created_products,
			updated_products,
			created_variants,
			updated_variants,
			created_categories,
			source_filename,
			file_storage_key,
			COALESCE(file_checksum_sha256, ''),
			COALESCE(last_error, ''),
			created_at,
			updated_at
		FROM catalog_import_batches
		ORDER BY created_at DESC
		LIMIT $1
		OFFSET $2
	`

	rows, err := r.db.Query(
		ctx,
		query,
		limit,
		offset,
	)
	if err != nil {
		return nil,
			fmt.Errorf(
				"list catalog import batches: %w",
				err,
			)
	}

	defer rows.Close()

	result := make(
		[]BatchListItem,
		0,
	)

	for rows.Next() {
		var item BatchListItem

		if err := rows.Scan(
			&item.ID,
			&item.Status,
			&item.TotalRows,
			&item.ValidRows,
			&item.FailedRows,
			&item.CreatedProducts,
			&item.UpdatedProducts,
			&item.CreatedVariants,
			&item.UpdatedVariants,
			&item.CreatedCategories,
			&item.SourceFilename,
			&item.StorageKey,
			&item.ChecksumSHA256,
			&item.LastError,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil,
				fmt.Errorf(
					"scan catalog import batch: %w",
					err,
				)
		}

		result = append(
			result,
			item,
		)
	}

	if err := rows.Err(); err != nil {
		return nil,
			fmt.Errorf(
				"iterate catalog import batches: %w",
				err,
			)
	}

	return result, nil
}

func (r *Repository) GetBatch(
	ctx context.Context,
	batchID string,
) (BatchSummary, error) {
	const query = `
		SELECT
			id::text,
			status,
			total_rows,
			valid_rows,
			failed_rows,
			created_products,
			updated_products,
			created_variants,
			updated_variants,
			created_categories
		FROM catalog_import_batches
		WHERE id = $1
	`

	var batch BatchSummary

	err := r.db.QueryRow(
		ctx,
		query,
		batchID,
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
				"get catalog import batch: %w",
				err,
			)
	}

	return batch, nil
}

func (r *Repository) ListRows(
	ctx context.Context,
	batchID string,
	limit int,
	offset int,
) ([]RowPreview, error) {
	const query = `
		SELECT
			id::text,
			sheet_name,
			row_number,
			COALESCE(product_code, ''),
			COALESCE(sku, ''),
			COALESCE(action, ''),
			status,
			raw_data,
			COALESCE(
				error_details,
				'[]'::jsonb
			)
		FROM catalog_import_rows
		WHERE batch_id = $1
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
		LIMIT $2
		OFFSET $3
	`

	rows, err := r.db.Query(
		ctx,
		query,
		batchID,
		limit,
		offset,
	)
	if err != nil {
		return nil,
			fmt.Errorf(
				"list catalog import rows: %w",
				err,
			)
	}

	defer rows.Close()

	result := make(
		[]RowPreview,
		0,
	)

	for rows.Next() {
		var row RowPreview
		var rawData []byte
		var errorData []byte

		if err := rows.Scan(
			&row.ID,
			&row.SheetName,
			&row.RowNumber,
			&row.ProductCode,
			&row.SKU,
			&row.Action,
			&row.Status,
			&rawData,
			&errorData,
		); err != nil {
			return nil,
				fmt.Errorf(
					"scan catalog import row: %w",
					err,
				)
		}

		row.RawData =
			json.RawMessage(
				append(
					[]byte(nil),
					rawData...,
				),
			)

		if len(errorData) > 0 {
			if err := json.Unmarshal(
				errorData,
				&row.Errors,
			); err != nil {
				return nil,
					fmt.Errorf(
						"decode catalog import row errors: %w",
						err,
					)
			}
		}

		result = append(
			result,
			row,
		)
	}

	if err := rows.Err(); err != nil {
		return nil,
			fmt.Errorf(
				"iterate catalog import rows: %w",
				err,
			)
	}

	return result, nil
}
