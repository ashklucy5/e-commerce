package admin

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/xuri/excelize/v2"

	platformdatabase "project.local/commerce-api/internal/platform/database"
	platformpagination "project.local/commerce-api/internal/platform/pagination"
	platformvalidation "project.local/commerce-api/internal/platform/validation"
)

const (
	FinanceImportTemplateVersion = 1
	FinanceImportSheetName       = "Finance"
	FinanceImportMaxRows         = 25000

	FinanceImportStatusValidating = "validating"
	FinanceImportStatusReady      = "ready"
	FinanceImportStatusApplying   = "applying"
	FinanceImportStatusCompleted  = "completed"
	FinanceImportStatusFailed     = "failed"

	FinanceImportRowStatusValid   = "valid"
	FinanceImportRowStatusInvalid = "invalid"
	FinanceImportRowStatusApplied = "applied"
	FinanceImportRowStatusSkipped = "skipped"
	FinanceImportRowStatusFailed  = "failed"

	FinanceImportRecordExpense     = "expense"
	FinanceImportRecordVariantCost = "variant_cost"

	adminEventFinanceImportCreated = "finance_import_created"
	adminEventFinanceImportApplied = "finance_import_applied"
)

var (
	ErrFinanceImportNotFound = errors.New(
		"finance import not found",
	)

	ErrInvalidFinanceImport = errors.New(
		"invalid finance import",
	)

	ErrFinanceImportNotReady = errors.New(
		"finance import is not ready to apply",
	)
)

var financeImportBusinessLocation = time.FixedZone(
	"Asia/Dhaka",
	6*60*60,
)

var financeImportHeaders = []string{
	"record_type",
	"date",
	"category",
	"scope",
	"amount",
	"unit_cost_amount",
	"currency",
	"product_code",
	"sku",
	"order_id",
	"staff_email",
	"warehouse_code",
	"description",
	"reference",
	"set_current",
}

type FinanceImportBatch struct {
	ID string `json:"id"`

	SourceFilename     string `json:"source_filename"`
	FileChecksumSHA256 string `json:"file_checksum_sha256"`
	TemplateVersion    int    `json:"template_version"`
	Status             string `json:"status"`

	TotalRows          int `json:"total_rows"`
	ValidRows          int `json:"valid_rows"`
	InvalidRows        int `json:"invalid_rows"`
	AppliedExpenses    int `json:"applied_expenses"`
	AppliedCostUpdates int `json:"applied_cost_updates"`
	SkippedRows        int `json:"skipped_rows"`

	CreatedByStaffID string `json:"created_by_staff_id"`

	LastError string `json:"last_error,omitempty"`

	AppliedAt *time.Time `json:"applied_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type FinanceImportRow struct {
	ID string `json:"id"`

	BatchID    string `json:"batch_id"`
	RowNumber  int    `json:"row_number"`
	RecordType string `json:"record_type"`
	Status     string `json:"status"`

	RawData        json.RawMessage `json:"raw_data"`
	NormalizedData json.RawMessage `json:"normalized_data,omitempty"`
	ErrorDetails   json.RawMessage `json:"error_details,omitempty"`

	AppliedExpenseID     string `json:"applied_expense_id,omitempty"`
	AppliedCostHistoryID string `json:"applied_cost_history_id,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type FinanceImportDetail struct {
	Batch FinanceImportBatch `json:"batch"`
	Rows  []FinanceImportRow `json:"rows"`

	Meta platformpagination.Meta `json:"meta"`
}

type FinanceImportCreateResult struct {
	Batch FinanceImportBatch `json:"batch"`

	Duplicate bool `json:"duplicate"`
}

type FinanceImportApplyResult struct {
	Batch FinanceImportBatch `json:"batch"`
}

type financeImportParsedRow struct {
	RowNumber int

	Values map[string]string
}

type financeImportPreparedRow struct {
	RowNumber int

	RecordType string
	Status     string

	RawData        json.RawMessage
	NormalizedData json.RawMessage
	ErrorDetails   json.RawMessage
}

type financeImportNormalizedRow struct {
	RecordType string `json:"record_type"`

	Expense *financeImportNormalizedExpense `json:"expense,omitempty"`

	VariantCost *financeImportNormalizedVariantCost `json:"variant_cost,omitempty"`
}

type financeImportNormalizedExpense struct {
	Category string `json:"category"`
	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`

	OccurredAt time.Time `json:"occurred_at"`

	Description string `json:"description"`

	OrderID        string `json:"order_id,omitempty"`
	ProductID      string `json:"product_id,omitempty"`
	VariantID      string `json:"variant_id,omitempty"`
	StaffAccountID string `json:"staff_account_id,omitempty"`
	WarehouseID    string `json:"warehouse_id,omitempty"`

	Reference string `json:"reference,omitempty"`
}

type financeImportNormalizedVariantCost struct {
	VariantID string `json:"variant_id"`
	SKU       string `json:"sku"`

	UnitCostAmount int64  `json:"unit_cost_amount"`
	Currency       string `json:"currency"`

	EffectiveAt time.Time `json:"effective_at"`

	Description string `json:"description,omitempty"`
	Reference   string `json:"reference,omitempty"`

	SetCurrent bool `json:"set_current"`
}

type financeImportResolution struct {
	ProductsByCode map[string]string

	VariantsBySKU map[string]financeImportVariantResolution

	StaffByEmail map[string]string

	WarehousesByCode map[string]string

	OrdersByID map[string]string
}

type financeImportVariantResolution struct {
	ID string

	SKU string

	Currency string
}

type financeImportErrorDetails struct {
	Errors []string `json:"errors"`
}

func (s *Service) CreateFinanceImport(
	ctx context.Context,
	filename string,
	content []byte,
	metadata AdminActionMetadata,
) (
	FinanceImportCreateResult,
	error,
) {
	filename =
		strings.TrimSpace(
			filename,
		)

	if filename == "" ||
		utf8.RuneCountInString(
			filename,
		) > 255 {

		return FinanceImportCreateResult{},
			fmt.Errorf(
				"%w: source filename is required and must be at most 255 characters",
				ErrInvalidFinanceImport,
			)
	}

	if len(
		content,
	) == 0 {

		return FinanceImportCreateResult{},
			fmt.Errorf(
				"%w: workbook is empty",
				ErrInvalidFinanceImport,
			)
	}

	checksumBytes :=
		sha256.Sum256(
			content,
		)

	checksum :=
		hex.EncodeToString(
			checksumBytes[:],
		)

	existing, err :=
		s.getFinanceImportBatchByChecksum(
			ctx,
			checksum,
		)

	if err == nil {
		return FinanceImportCreateResult{
			Batch: existing,

			Duplicate: true,
		}, nil
	}

	if !errors.Is(
		err,
		ErrFinanceImportNotFound,
	) {
		return FinanceImportCreateResult{},
			err
	}

	parsedRows, err :=
		parseFinanceImportWorkbook(
			content,
		)
	if err != nil {
		return FinanceImportCreateResult{},
			err
	}

	resolution, err :=
		s.resolveFinanceImportReferences(
			ctx,
			parsedRows,
		)
	if err != nil {
		return FinanceImportCreateResult{},
			err
	}

	preparedRows :=
		prepareFinanceImportRows(
			parsedRows,
			resolution,
		)

	validRows :=
		0

	invalidRows :=
		0

	for _, row := range preparedRows {

		if row.Status ==
			FinanceImportRowStatusValid {

			validRows++

			continue
		}

		invalidRows++
	}

	status :=
		FinanceImportStatusReady

	lastError :=
		""

	if invalidRows > 0 {
		status =
			FinanceImportStatusFailed

		lastError =
			"Workbook contains invalid rows. Correct the reported rows and upload a new workbook."
	}

	var batchID string

	var duplicateBatchID string

	err =
		platformdatabase.WithinTx(
			ctx,
			s.db,
			func(
				ctx context.Context,
				tx pgx.Tx,
			) error {
				err :=
					tx.QueryRow(
						ctx,
						`
							INSERT INTO finance_import_batches (
								source_filename,
								file_checksum_sha256,
								template_version,
								status,
								total_rows,
								valid_rows,
								invalid_rows,
								applied_expenses,
								applied_cost_updates,
								skipped_rows,
								created_by_staff_id,
								last_error,
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
								0,
								0,
								0,
								$8::uuid,
								NULLIF($9, ''),
								now(),
								now()
							)
							ON CONFLICT (file_checksum_sha256)
							DO NOTHING
							RETURNING id::text
						`,
						filename,
						checksum,
						FinanceImportTemplateVersion,
						status,
						len(
							preparedRows,
						),
						validRows,
						invalidRows,
						metadata.StaffAccountID,
						lastError,
					).Scan(
						&batchID,
					)

				if errors.Is(
					err,
					pgx.ErrNoRows,
				) {
					return tx.QueryRow(
						ctx,
						`
							SELECT id::text
							FROM finance_import_batches
							WHERE file_checksum_sha256 = $1
						`,
						checksum,
					).Scan(
						&duplicateBatchID,
					)
				}

				if err != nil {
					return fmt.Errorf(
						"create finance import batch: %w",
						err,
					)
				}

				for _, row := range preparedRows {

					var normalized any

					if len(
						row.NormalizedData,
					) > 0 {

						normalized =
							string(
								row.NormalizedData,
							)
					}

					var errorDetails any

					if len(
						row.ErrorDetails,
					) > 0 {

						errorDetails =
							string(
								row.ErrorDetails,
							)
					}

					if _, err :=
						tx.Exec(
							ctx,
							`
								INSERT INTO finance_import_rows (
									batch_id,
									row_number,
									record_type,
									status,
									raw_data,
									normalized_data,
									error_details,
									created_at,
									updated_at
								)
								VALUES (
									$1::uuid,
									$2,
									$3,
									$4,
									$5::jsonb,
									$6::jsonb,
									$7::jsonb,
									now(),
									now()
								)
							`,
							batchID,
							row.RowNumber,
							row.RecordType,
							row.Status,
							string(
								row.RawData,
							),
							normalized,
							errorDetails,
						); err != nil {

						return fmt.Errorf(
							"insert finance import row %d: %w",
							row.RowNumber,
							err,
						)
					}
				}

				return insertAdminActionAuditTx(
					ctx,
					tx,
					metadata,
					adminEventFinanceImportCreated,
					map[string]any{
						"finance_import_batch_id": batchID,

						"source_filename": filename,

						"file_checksum_sha256": checksum,

						"total_rows": len(
							preparedRows,
						),

						"valid_rows": validRows,

						"invalid_rows": invalidRows,
					},
				)
			},
		)
	if err != nil {
		return FinanceImportCreateResult{},
			err
	}

	if duplicateBatchID != "" {
		batch, err :=
			s.GetFinanceImportBatch(
				ctx,
				duplicateBatchID,
			)
		if err != nil {
			return FinanceImportCreateResult{},
				err
		}

		return FinanceImportCreateResult{
			Batch: batch,

			Duplicate: true,
		}, nil
	}

	batch, err :=
		s.GetFinanceImportBatch(
			ctx,
			batchID,
		)
	if err != nil {
		return FinanceImportCreateResult{},
			err
	}

	return FinanceImportCreateResult{
		Batch: batch,

		Duplicate: false,
	}, nil
}

func (s *Service) GetFinanceImportBatch(
	ctx context.Context,
	batchID string,
) (
	FinanceImportBatch,
	error,
) {
	batch, err :=
		scanFinanceImportBatch(
			s.db.QueryRow(
				ctx,
				financeImportBatchSelectSQL+
					`
						WHERE id = $1::uuid
					`,
				batchID,
			),
		)

	if errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return FinanceImportBatch{},
			ErrFinanceImportNotFound
	}

	if err != nil {
		return FinanceImportBatch{},
			fmt.Errorf(
				"get finance import batch: %w",
				err,
			)
	}

	return batch,
		nil
}

func (s *Service) GetFinanceImport(
	ctx context.Context,
	batchID string,
	params platformpagination.Params,
) (
	FinanceImportDetail,
	error,
) {
	batch, err :=
		s.GetFinanceImportBatch(
			ctx,
			batchID,
		)
	if err != nil {
		return FinanceImportDetail{},
			err
	}

	var total int64

	if err :=
		s.db.QueryRow(
			ctx,
			`
				SELECT COUNT(*)::bigint
				FROM finance_import_rows
				WHERE batch_id = $1::uuid
			`,
			batchID,
		).Scan(
			&total,
		); err != nil {

		return FinanceImportDetail{},
			fmt.Errorf(
				"count finance import rows: %w",
				err,
			)
	}

	rows, err :=
		s.db.Query(
			ctx,
			`
				SELECT
					id::text,
					batch_id::text,
					row_number,
					record_type,
					status,
					raw_data,
					normalized_data,
					error_details,
					COALESCE(applied_expense_id::text, ''),
					COALESCE(applied_cost_history_id::text, ''),
					created_at,
					updated_at
				FROM finance_import_rows
				WHERE batch_id = $1::uuid
				ORDER BY row_number
				LIMIT $2
				OFFSET $3
			`,
			batchID,
			params.Limit,
			params.Offset(),
		)
	if err != nil {
		return FinanceImportDetail{},
			fmt.Errorf(
				"list finance import rows: %w",
				err,
			)
	}

	defer rows.Close()

	items :=
		make(
			[]FinanceImportRow,
			0,
			params.Limit,
		)

	for rows.Next() {
		var item FinanceImportRow

		if err :=
			rows.Scan(
				&item.ID,
				&item.BatchID,
				&item.RowNumber,
				&item.RecordType,
				&item.Status,
				&item.RawData,
				&item.NormalizedData,
				&item.ErrorDetails,
				&item.AppliedExpenseID,
				&item.AppliedCostHistoryID,
				&item.CreatedAt,
				&item.UpdatedAt,
			); err != nil {

			return FinanceImportDetail{},
				fmt.Errorf(
					"scan finance import row: %w",
					err,
				)
		}

		items =
			append(
				items,
				item,
			)
	}

	if err :=
		rows.Err(); err != nil {

		return FinanceImportDetail{},
			fmt.Errorf(
				"iterate finance import rows: %w",
				err,
			)
	}

	return FinanceImportDetail{
		Batch: batch,

		Rows: items,

		Meta: platformpagination.NewMeta(
			params,
			total,
		),
	}, nil
}

func (s *Service) ApplyFinanceImport(
	ctx context.Context,
	batchID string,
	metadata AdminActionMetadata,
) (
	FinanceImportApplyResult,
	error,
) {
	err :=
		platformdatabase.WithinTx(
			ctx,
			s.db,
			func(
				ctx context.Context,
				tx pgx.Tx,
			) error {
				batch, err :=
					scanFinanceImportBatch(
						tx.QueryRow(
							ctx,
							financeImportBatchSelectSQL+
								`
									WHERE id = $1::uuid
									FOR UPDATE
								`,
							batchID,
						),
					)

				if errors.Is(
					err,
					pgx.ErrNoRows,
				) {
					return ErrFinanceImportNotFound
				}

				if err != nil {
					return fmt.Errorf(
						"lock finance import batch: %w",
						err,
					)
				}

				if batch.Status ==
					FinanceImportStatusCompleted {

					return nil
				}

				if batch.Status !=
					FinanceImportStatusReady ||
					batch.InvalidRows != 0 {

					return ErrFinanceImportNotReady
				}

				if _, err :=
					tx.Exec(
						ctx,
						`
							UPDATE finance_import_batches
							SET
								status = 'applying',
								last_error = NULL,
								updated_at = now()
							WHERE id = $1::uuid
						`,
						batchID,
					); err != nil {

					return fmt.Errorf(
						"mark finance import applying: %w",
						err,
					)
				}

				rows, err :=
					tx.Query(
						ctx,
						`
							SELECT
								id::text,
								row_number,
								record_type,
								normalized_data
							FROM finance_import_rows
							WHERE
								batch_id = $1::uuid
								AND status = 'valid'
							ORDER BY row_number
							FOR UPDATE
						`,
						batchID,
					)
				if err != nil {
					return fmt.Errorf(
						"load finance import rows for apply: %w",
						err,
					)
				}

				type applyRow struct {
					ID string

					RowNumber int

					RecordType string

					Normalized []byte
				}

				applyRows :=
					make(
						[]applyRow,
						0,
						batch.ValidRows,
					)

				for rows.Next() {
					var row applyRow

					if err :=
						rows.Scan(
							&row.ID,
							&row.RowNumber,
							&row.RecordType,
							&row.Normalized,
						); err != nil {

						rows.Close()

						return fmt.Errorf(
							"scan finance import apply row: %w",
							err,
						)
					}

					applyRows =
						append(
							applyRows,
							row,
						)
				}

				if err :=
					rows.Err(); err != nil {

					rows.Close()

					return fmt.Errorf(
						"iterate finance import apply rows: %w",
						err,
					)
				}

				rows.Close()

				if len(
					applyRows,
				) != batch.ValidRows ||
					batch.ValidRows !=
						batch.TotalRows {

					return fmt.Errorf(
						"%w: validated row count changed before apply",
						ErrFinanceImportNotReady,
					)
				}

				appliedExpenses :=
					0

				appliedCosts :=
					0

				for _, row := range applyRows {

					var normalized financeImportNormalizedRow

					if err :=
						json.Unmarshal(
							row.Normalized,
							&normalized,
						); err != nil {

						return fmt.Errorf(
							"decode normalized finance import row %d: %w",
							row.RowNumber,
							err,
						)
					}

					switch normalized.RecordType {
					case FinanceImportRecordExpense:
						if normalized.Expense ==
							nil {

							return fmt.Errorf(
								"%w: row %d has no expense payload",
								ErrInvalidFinanceImport,
								row.RowNumber,
							)
						}

						expenseID, err :=
							createFinanceExpenseImportTx(
								ctx,
								tx,
								batchID,
								row.RowNumber,
								*normalized.Expense,
								metadata,
							)
						if err != nil {
							return err
						}

						if _, err :=
							tx.Exec(
								ctx,
								`
									UPDATE finance_import_rows
									SET
										status = 'applied',
										applied_expense_id = $2::uuid,
										updated_at = now()
									WHERE id = $1::uuid
								`,
								row.ID,
								expenseID,
							); err != nil {

							return fmt.Errorf(
								"mark finance import expense row applied: %w",
								err,
							)
						}

						appliedExpenses++

					case FinanceImportRecordVariantCost:
						if normalized.VariantCost ==
							nil {

							return fmt.Errorf(
								"%w: row %d has no variant cost payload",
								ErrInvalidFinanceImport,
								row.RowNumber,
							)
						}

						costInput :=
							FinanceVariantCostInput{
								VariantID: normalized.
									VariantCost.
									VariantID,

								SKU: normalized.
									VariantCost.
									SKU,

								UnitCostAmount: normalized.
									VariantCost.
									UnitCostAmount,

								Currency: normalized.
									VariantCost.
									Currency,

								EffectiveAt: normalized.
									VariantCost.
									EffectiveAt,

								Source: FinanceVariantCostSourceImport,

								Description: normalized.
									VariantCost.
									Description,

								Reference: normalized.
									VariantCost.
									Reference,

								SetCurrent: normalized.
									VariantCost.
									SetCurrent,
							}

						costID,
							_,
							err :=
							createFinanceVariantCostTx(
								ctx,
								tx,
								costInput,
								metadata,
								false,
							)
						if err != nil {
							return err
						}

						if _, err :=
							tx.Exec(
								ctx,
								`
									UPDATE finance_import_rows
									SET
										status = 'applied',
										applied_cost_history_id = $2::uuid,
										updated_at = now()
									WHERE id = $1::uuid
								`,
								row.ID,
								costID,
							); err != nil {

							return fmt.Errorf(
								"mark finance import cost row applied: %w",
								err,
							)
						}

						appliedCosts++

					default:
						return fmt.Errorf(
							"%w: unsupported record_type %q in row %d",
							ErrInvalidFinanceImport,
							normalized.RecordType,
							row.RowNumber,
						)
					}
				}

				if _, err :=
					tx.Exec(
						ctx,
						`
							UPDATE finance_import_batches
							SET
								status = 'completed',
								applied_expenses = $2,
								applied_cost_updates = $3,
								skipped_rows = 0,
								last_error = NULL,
								applied_at = now(),
								updated_at = now()
							WHERE id = $1::uuid
						`,
						batchID,
						appliedExpenses,
						appliedCosts,
					); err != nil {

					return fmt.Errorf(
						"complete finance import batch: %w",
						err,
					)
				}

				return insertAdminActionAuditTx(
					ctx,
					tx,
					metadata,
					adminEventFinanceImportApplied,
					map[string]any{
						"finance_import_batch_id": batchID,

						"applied_expenses": appliedExpenses,

						"applied_cost_updates": appliedCosts,
					},
				)
			},
		)
	if err != nil {
		return FinanceImportApplyResult{},
			err
	}

	batch, err :=
		s.GetFinanceImportBatch(
			ctx,
			batchID,
		)
	if err != nil {
		return FinanceImportApplyResult{},
			err
	}

	return FinanceImportApplyResult{
		Batch: batch,
	}, nil
}

func parseFinanceImportWorkbook(
	content []byte,
) (
	[]financeImportParsedRow,
	error,
) {
	book, err :=
		excelize.OpenReader(
			bytes.NewReader(
				content,
			),
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"%w: cannot read .xlsx workbook: %v",
				ErrInvalidFinanceImport,
				err,
			)
	}

	defer func() {
		_ = book.Close()
	}()

	rows, err :=
		book.GetRows(
			FinanceImportSheetName,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"%w: sheet %q is required",
				ErrInvalidFinanceImport,
				FinanceImportSheetName,
			)
	}

	if len(
		rows,
	) == 0 {

		return nil,
			fmt.Errorf(
				"%w: sheet %q is empty",
				ErrInvalidFinanceImport,
				FinanceImportSheetName,
			)
	}

	headerIndex, err :=
		validateFinanceImportHeaders(
			rows[0],
		)
	if err != nil {
		return nil,
			err
	}

	result :=
		make(
			[]financeImportParsedRow,
			0,
			len(rows)-1,
		)

	for rowIndex :=
		1; rowIndex < len(rows); rowIndex++ {

		values :=
			make(
				map[string]string,
				len(
					financeImportHeaders,
				),
			)

		blank :=
			true

		for _, header := range financeImportHeaders {

			column :=
				headerIndex[header]

			value :=
				""

			if column <
				len(
					rows[rowIndex],
				) {

				value =
					strings.TrimSpace(
						rows[rowIndex][column],
					)
			}

			values[header] =
				value

			if value != "" {
				blank =
					false
			}
		}

		if blank {
			continue
		}

		if len(
			result,
		) >= FinanceImportMaxRows {

			return nil,
				fmt.Errorf(
					"%w: workbook may contain at most %d non-empty data rows",
					ErrInvalidFinanceImport,
					FinanceImportMaxRows,
				)
		}

		result =
			append(
				result,
				financeImportParsedRow{
					RowNumber: rowIndex + 1,

					Values: values,
				},
			)
	}

	if len(
		result,
	) == 0 {

		return nil,
			fmt.Errorf(
				"%w: workbook contains no finance data rows",
				ErrInvalidFinanceImport,
			)
	}

	return result,
		nil
}

func validateFinanceImportHeaders(
	row []string,
) (
	map[string]int,
	error,
) {
	known :=
		make(
			map[string]struct{},
			len(
				financeImportHeaders,
			),
		)

	for _, header := range financeImportHeaders {

		known[header] =
			struct{}{}
	}

	index :=
		make(
			map[string]int,
			len(
				financeImportHeaders,
			),
		)

	for column, raw := range row {

		header :=
			strings.ToLower(
				strings.TrimSpace(
					raw,
				),
			)

		if header == "" {
			continue
		}

		if _, ok :=
			known[header]; !ok {

			return nil,
				fmt.Errorf(
					"%w: unknown column %q",
					ErrInvalidFinanceImport,
					raw,
				)
		}

		if _, exists :=
			index[header]; exists {

			return nil,
				fmt.Errorf(
					"%w: duplicate column %q",
					ErrInvalidFinanceImport,
					header,
				)
		}

		index[header] =
			column
	}

	for _, header := range financeImportHeaders {

		if _, ok :=
			index[header]; !ok {

			return nil,
				fmt.Errorf(
					"%w: missing required column %q",
					ErrInvalidFinanceImport,
					header,
				)
		}
	}

	return index,
		nil
}

func (s *Service) resolveFinanceImportReferences(
	ctx context.Context,
	rows []financeImportParsedRow,
) (
	financeImportResolution,
	error,
) {
	productCodes :=
		map[string]struct{}{}

	skus :=
		map[string]struct{}{}

	staffEmails :=
		map[string]struct{}{}

	warehouseCodes :=
		map[string]struct{}{}

	orderIDs :=
		map[string]struct{}{}

	for _, row := range rows {

		recordType :=
			strings.ToLower(
				strings.TrimSpace(
					row.Values["record_type"],
				),
			)

		if recordType ==
			FinanceImportRecordVariantCost {

			if sku :=
				strings.TrimSpace(
					row.Values["sku"],
				); sku != "" {

				skus[sku] =
					struct{}{}
			}

			continue
		}

		if recordType !=
			FinanceImportRecordExpense {

			continue
		}

		scope :=
			strings.ToLower(
				strings.TrimSpace(
					row.Values["scope"],
				),
			)

		switch scope {
		case FinanceExpenseScopeProduct:
			if value :=
				strings.TrimSpace(
					row.Values["product_code"],
				); value != "" {

				productCodes[value] =
					struct{}{}
			}

		case FinanceExpenseScopeVariant:
			if value :=
				strings.TrimSpace(
					row.Values["sku"],
				); value != "" {

				skus[value] =
					struct{}{}
			}

		case FinanceExpenseScopeStaff:
			if value :=
				strings.ToLower(
					strings.TrimSpace(
						row.Values["staff_email"],
					),
				); value != "" {

				staffEmails[value] =
					struct{}{}
			}

		case FinanceExpenseScopeWarehouse:
			if value :=
				strings.ToLower(
					strings.TrimSpace(
						row.Values["warehouse_code"],
					),
				); value != "" {

				warehouseCodes[value] =
					struct{}{}
			}

		case FinanceExpenseScopeOrder:
			value :=
				strings.TrimSpace(
					row.Values["order_id"],
				)

			if value != "" &&
				platformvalidation.IsUUID(
					value,
				) {

				orderIDs[value] =
					struct{}{}
			}
		}
	}

	resolution :=
		financeImportResolution{
			ProductsByCode: map[string]string{},

			VariantsBySKU: map[string]financeImportVariantResolution{},

			StaffByEmail: map[string]string{},

			WarehousesByCode: map[string]string{},

			OrdersByID: map[string]string{},
		}

	if err :=
		s.resolveFinanceImportProducts(
			ctx,
			mapKeys(
				productCodes,
			),
			resolution.ProductsByCode,
		); err != nil {

		return financeImportResolution{},
			err
	}

	if err :=
		s.resolveFinanceImportVariants(
			ctx,
			mapKeys(
				skus,
			),
			resolution.VariantsBySKU,
		); err != nil {

		return financeImportResolution{},
			err
	}

	if err :=
		s.resolveFinanceImportStaff(
			ctx,
			mapKeys(
				staffEmails,
			),
			resolution.StaffByEmail,
		); err != nil {

		return financeImportResolution{},
			err
	}

	if err :=
		s.resolveFinanceImportWarehouses(
			ctx,
			mapKeys(
				warehouseCodes,
			),
			resolution.WarehousesByCode,
		); err != nil {

		return financeImportResolution{},
			err
	}

	if err :=
		s.resolveFinanceImportOrders(
			ctx,
			mapKeys(
				orderIDs,
			),
			resolution.OrdersByID,
		); err != nil {

		return financeImportResolution{},
			err
	}

	return resolution,
		nil
}

func (s *Service) resolveFinanceImportProducts(
	ctx context.Context,
	values []string,
	target map[string]string,
) error {
	if len(
		values,
	) == 0 {

		return nil
	}

	rows, err :=
		s.db.Query(
			ctx,
			`
				SELECT
					product_code,
					id::text
				FROM products
				WHERE product_code = ANY($1::text[])
			`,
			values,
		)
	if err != nil {
		return fmt.Errorf(
			"resolve finance import products: %w",
			err,
		)
	}

	defer rows.Close()

	for rows.Next() {
		var key string

		var id string

		if err :=
			rows.Scan(
				&key,
				&id,
			); err != nil {

			return fmt.Errorf(
				"scan finance import product resolution: %w",
				err,
			)
		}

		target[key] =
			id
	}

	if err :=
		rows.Err(); err != nil {

		return fmt.Errorf(
			"iterate finance import product resolution: %w",
			err,
		)
	}

	return nil
}

func (s *Service) resolveFinanceImportVariants(
	ctx context.Context,
	values []string,
	target map[string]financeImportVariantResolution,
) error {
	if len(
		values,
	) == 0 {

		return nil
	}

	rows, err :=
		s.db.Query(
			ctx,
			`
				SELECT
					sku,
					id::text,
					currency
				FROM product_variants
				WHERE sku = ANY($1::text[])
			`,
			values,
		)
	if err != nil {
		return fmt.Errorf(
			"resolve finance import variants: %w",
			err,
		)
	}

	defer rows.Close()

	for rows.Next() {
		var item financeImportVariantResolution

		if err :=
			rows.Scan(
				&item.SKU,
				&item.ID,
				&item.Currency,
			); err != nil {

			return fmt.Errorf(
				"scan finance import variant resolution: %w",
				err,
			)
		}

		target[item.SKU] =
			item
	}

	if err :=
		rows.Err(); err != nil {

		return fmt.Errorf(
			"iterate finance import variant resolution: %w",
			err,
		)
	}

	return nil
}

func (s *Service) resolveFinanceImportStaff(
	ctx context.Context,
	values []string,
	target map[string]string,
) error {
	if len(
		values,
	) == 0 {

		return nil
	}

	rows, err :=
		s.db.Query(
			ctx,
			`
				SELECT
					lower(email),
					id::text
				FROM staff_accounts
				WHERE lower(email) = ANY($1::text[])
			`,
			values,
		)
	if err != nil {
		return fmt.Errorf(
			"resolve finance import staff: %w",
			err,
		)
	}

	defer rows.Close()

	for rows.Next() {
		var key string

		var id string

		if err :=
			rows.Scan(
				&key,
				&id,
			); err != nil {

			return fmt.Errorf(
				"scan finance import staff resolution: %w",
				err,
			)
		}

		target[key] =
			id
	}

	if err :=
		rows.Err(); err != nil {

		return fmt.Errorf(
			"iterate finance import staff resolution: %w",
			err,
		)
	}

	return nil
}

func (s *Service) resolveFinanceImportWarehouses(
	ctx context.Context,
	values []string,
	target map[string]string,
) error {
	if len(
		values,
	) == 0 {

		return nil
	}

	rows, err :=
		s.db.Query(
			ctx,
			`
				SELECT
					lower(code),
					id::text
				FROM warehouses
				WHERE lower(code) = ANY($1::text[])
			`,
			values,
		)
	if err != nil {
		return fmt.Errorf(
			"resolve finance import warehouses: %w",
			err,
		)
	}

	defer rows.Close()

	for rows.Next() {
		var key string

		var id string

		if err :=
			rows.Scan(
				&key,
				&id,
			); err != nil {

			return fmt.Errorf(
				"scan finance import warehouse resolution: %w",
				err,
			)
		}

		target[key] =
			id
	}

	if err :=
		rows.Err(); err != nil {

		return fmt.Errorf(
			"iterate finance import warehouse resolution: %w",
			err,
		)
	}

	return nil
}

func (s *Service) resolveFinanceImportOrders(
	ctx context.Context,
	values []string,
	target map[string]string,
) error {
	if len(
		values,
	) == 0 {

		return nil
	}

	rows, err :=
		s.db.Query(
			ctx,
			`
				SELECT id::text
				FROM orders
				WHERE id::text = ANY($1::text[])
			`,
			values,
		)
	if err != nil {
		return fmt.Errorf(
			"resolve finance import orders: %w",
			err,
		)
	}

	defer rows.Close()

	for rows.Next() {
		var id string

		if err :=
			rows.Scan(
				&id,
			); err != nil {

			return fmt.Errorf(
				"scan finance import order resolution: %w",
				err,
			)
		}

		target[id] =
			id
	}

	if err :=
		rows.Err(); err != nil {

		return fmt.Errorf(
			"iterate finance import order resolution: %w",
			err,
		)
	}

	return nil
}

func prepareFinanceImportRows(
	rows []financeImportParsedRow,
	resolution financeImportResolution,
) []financeImportPreparedRow {
	result :=
		make(
			[]financeImportPreparedRow,
			0,
			len(
				rows,
			),
		)

	for _, row := range rows {

		rawJSON, _ :=
			json.Marshal(
				row.Values,
			)

		normalized,
			validationErrors :=
			normalizeFinanceImportRow(
				row,
				resolution,
			)

		prepared :=
			financeImportPreparedRow{
				RowNumber: row.RowNumber,

				RecordType: strings.ToLower(
					strings.TrimSpace(
						row.Values["record_type"],
					),
				),

				RawData: rawJSON,
			}

		if len(
			validationErrors,
		) > 0 {

			prepared.Status =
				FinanceImportRowStatusInvalid

			errorJSON, _ :=
				json.Marshal(
					financeImportErrorDetails{
						Errors: validationErrors,
					},
				)

			prepared.ErrorDetails =
				errorJSON
		} else {
			prepared.Status =
				FinanceImportRowStatusValid

			normalizedJSON, _ :=
				json.Marshal(
					normalized,
				)

			prepared.NormalizedData =
				normalizedJSON
		}

		result =
			append(
				result,
				prepared,
			)
	}

	return result
}

func normalizeFinanceImportRow(
	row financeImportParsedRow,
	resolution financeImportResolution,
) (
	financeImportNormalizedRow,
	[]string,
) {
	values :=
		row.Values

	recordType :=
		strings.ToLower(
			strings.TrimSpace(
				values["record_type"],
			),
		)

	switch recordType {
	case FinanceImportRecordExpense:
		return normalizeFinanceImportExpenseRow(
			values,
			resolution,
		)

	case FinanceImportRecordVariantCost:
		return normalizeFinanceImportVariantCostRow(
			values,
			resolution,
		)

	default:
		return financeImportNormalizedRow{},
			[]string{
				"record_type must be expense or variant_cost",
			}
	}
}

func normalizeFinanceImportExpenseRow(
	values map[string]string,
	resolution financeImportResolution,
) (
	financeImportNormalizedRow,
	[]string,
) {
	errs :=
		make(
			[]string,
			0,
		)

	category :=
		strings.ToLower(
			strings.TrimSpace(
				values["category"],
			),
		)

	if !validFinanceExpenseCategory(
		category,
	) {
		errs =
			append(
				errs,
				"category is invalid",
			)
	}

	scope :=
		strings.ToLower(
			strings.TrimSpace(
				values["scope"],
			),
		)

	if scope == "" {
		scope =
			FinanceExpenseScopeBusiness
	}

	if !validFinanceExpenseScope(
		scope,
	) {
		errs =
			append(
				errs,
				"scope must be business, order, product, variant, staff, or warehouse",
			)
	}

	amount, err :=
		parseFinanceImportInt64(
			values["amount"],
		)

	if err != nil ||
		amount <= 0 {

		errs =
			append(
				errs,
				"amount must be a positive whole number",
			)
	}

	if strings.TrimSpace(
		values["unit_cost_amount"],
	) != "" {

		errs =
			append(
				errs,
				"unit_cost_amount must be blank for expense rows",
			)
	}

	occurredAt, err :=
		parseFinanceImportDate(
			values["date"],
		)

	if err != nil {
		errs =
			append(
				errs,
				"date must be YYYY-MM-DD, RFC3339, or a valid Excel date",
			)
	}

	currency :=
		strings.ToUpper(
			strings.TrimSpace(
				values["currency"],
			),
		)

	if currency == "" {
		currency =
			"BDT"
	}

	if !validFinanceCurrency(
		currency,
	) {
		errs =
			append(
				errs,
				"currency must be a 3-letter code",
			)
	}

	description :=
		strings.TrimSpace(
			values["description"],
		)

	if description == "" ||
		utf8.RuneCountInString(
			description,
		) > 500 {

		errs =
			append(
				errs,
				"description is required and must be at most 500 characters",
			)
	}

	reference :=
		strings.TrimSpace(
			values["reference"],
		)

	if utf8.RuneCountInString(
		reference,
	) > 160 {

		errs =
			append(
				errs,
				"reference must be at most 160 characters",
			)
	}

	if strings.TrimSpace(
		values["set_current"],
	) != "" {

		errs =
			append(
				errs,
				"set_current must be blank for expense rows",
			)
	}

	productCode :=
		strings.TrimSpace(
			values["product_code"],
		)

	sku :=
		strings.TrimSpace(
			values["sku"],
		)

	orderID :=
		strings.TrimSpace(
			values["order_id"],
		)

	staffEmail :=
		strings.ToLower(
			strings.TrimSpace(
				values["staff_email"],
			),
		)

	warehouseCode :=
		strings.ToLower(
			strings.TrimSpace(
				values["warehouse_code"],
			),
		)

	normalizedExpense :=
		&financeImportNormalizedExpense{
			Category: category,

			Amount: amount,

			Currency: currency,

			OccurredAt: occurredAt,

			Description: description,

			Reference: reference,
		}

	switch scope {
	case FinanceExpenseScopeBusiness:
		if productCode != "" ||
			sku != "" ||
			orderID != "" ||
			staffEmail != "" ||
			warehouseCode != "" {

			errs =
				append(
					errs,
					"business scope cannot include product_code, sku, order_id, staff_email, or warehouse_code",
				)
		}

	case FinanceExpenseScopeOrder:
		if !platformvalidation.IsUUID(
			orderID,
		) {
			errs =
				append(
					errs,
					"order scope requires a valid order_id",
				)
		} else if id :=
			resolution.
				OrdersByID[orderID]; id == "" {

			errs =
				append(
					errs,
					"order_id was not found",
				)
		} else {
			normalizedExpense.OrderID =
				id
		}

		if productCode != "" ||
			sku != "" ||
			staffEmail != "" ||
			warehouseCode != "" {

			errs =
				append(
					errs,
					"order scope may only use order_id",
				)
		}

	case FinanceExpenseScopeProduct:
		if productCode == "" {
			errs =
				append(
					errs,
					"product scope requires product_code",
				)
		} else if id :=
			resolution.
				ProductsByCode[productCode]; id == "" {

			errs =
				append(
					errs,
					"product_code was not found",
				)
		} else {
			normalizedExpense.ProductID =
				id
		}

		if sku != "" ||
			orderID != "" ||
			staffEmail != "" ||
			warehouseCode != "" {

			errs =
				append(
					errs,
					"product scope may only use product_code",
				)
		}

	case FinanceExpenseScopeVariant:
		if sku == "" {
			errs =
				append(
					errs,
					"variant scope requires sku",
				)
		} else if variant, ok :=
			resolution.
				VariantsBySKU[sku]; !ok {

			errs =
				append(
					errs,
					"sku was not found",
				)
		} else {
			normalizedExpense.VariantID =
				variant.ID
		}

		if productCode != "" ||
			orderID != "" ||
			staffEmail != "" ||
			warehouseCode != "" {

			errs =
				append(
					errs,
					"variant scope may only use sku",
				)
		}

	case FinanceExpenseScopeStaff:
		if staffEmail == "" {
			errs =
				append(
					errs,
					"staff scope requires staff_email",
				)
		} else if id :=
			resolution.
				StaffByEmail[staffEmail]; id == "" {

			errs =
				append(
					errs,
					"staff_email was not found",
				)
		} else {
			normalizedExpense.StaffAccountID =
				id
		}

		if productCode != "" ||
			sku != "" ||
			orderID != "" ||
			warehouseCode != "" {

			errs =
				append(
					errs,
					"staff scope may only use staff_email",
				)
		}

	case FinanceExpenseScopeWarehouse:
		if warehouseCode == "" {
			errs =
				append(
					errs,
					"warehouse scope requires warehouse_code",
				)
		} else if id :=
			resolution.
				WarehousesByCode[warehouseCode]; id == "" {

			errs =
				append(
					errs,
					"warehouse_code was not found",
				)
		} else {
			normalizedExpense.WarehouseID =
				id
		}

		if productCode != "" ||
			sku != "" ||
			orderID != "" ||
			staffEmail != "" {

			errs =
				append(
					errs,
					"warehouse scope may only use warehouse_code",
				)
		}
	}

	if len(
		errs,
	) == 0 {

		input :=
			FinanceExpenseInput{
				Category: normalizedExpense.Category,

				Amount: normalizedExpense.Amount,

				Currency: normalizedExpense.Currency,

				OccurredAt: normalizedExpense.OccurredAt,

				Description: normalizedExpense.Description,

				OrderID: normalizedExpense.OrderID,

				ProductID: normalizedExpense.ProductID,

				VariantID: normalizedExpense.VariantID,

				StaffAccountID: normalizedExpense.StaffAccountID,

				WarehouseID: normalizedExpense.WarehouseID,

				Reference: normalizedExpense.Reference,
			}

		if _, err :=
			normalizeFinanceExpenseInput(
				input,
			); err != nil {

			errs =
				append(
					errs,
					err.Error(),
				)
		}
	}

	return financeImportNormalizedRow{
			RecordType: FinanceImportRecordExpense,

			Expense: normalizedExpense,
		},
		errs
}

func normalizeFinanceImportVariantCostRow(
	values map[string]string,
	resolution financeImportResolution,
) (
	financeImportNormalizedRow,
	[]string,
) {
	errs :=
		make(
			[]string,
			0,
		)

	for _, column := range []string{
		"category",
		"scope",
		"amount",
		"product_code",
		"order_id",
		"staff_email",
		"warehouse_code",
	} {

		if strings.TrimSpace(
			values[column],
		) != "" {

			errs =
				append(
					errs,
					column+
						" must be blank for variant_cost rows",
				)
		}
	}

	sku :=
		strings.TrimSpace(
			values["sku"],
		)

	variant, found :=
		resolution.
			VariantsBySKU[sku]

	if sku == "" {
		errs =
			append(
				errs,
				"sku is required for variant_cost rows",
			)
	} else if !found {
		errs =
			append(
				errs,
				"sku was not found",
			)
	}

	unitCost, err :=
		parseFinanceImportInt64(
			values["unit_cost_amount"],
		)

	if err != nil ||
		unitCost < 0 {

		errs =
			append(
				errs,
				"unit_cost_amount must be a non-negative whole number",
			)
	}

	effectiveAt, err :=
		parseFinanceImportDate(
			values["date"],
		)

	if err != nil {
		errs =
			append(
				errs,
				"date must be YYYY-MM-DD, RFC3339, or a valid Excel date",
			)
	}

	currency :=
		strings.ToUpper(
			strings.TrimSpace(
				values["currency"],
			),
		)

	if currency == "" &&
		found {

		currency =
			variant.Currency
	}

	if currency != "" &&
		!validFinanceCurrency(
			currency,
		) {

		errs =
			append(
				errs,
				"currency must be a 3-letter code",
			)
	}

	description :=
		strings.TrimSpace(
			values["description"],
		)

	if utf8.RuneCountInString(
		description,
	) > 500 {

		errs =
			append(
				errs,
				"description must be at most 500 characters",
			)
	}

	reference :=
		strings.TrimSpace(
			values["reference"],
		)

	if utf8.RuneCountInString(
		reference,
	) > 160 {

		errs =
			append(
				errs,
				"reference must be at most 160 characters",
			)
	}

	setCurrent, err :=
		parseFinanceImportBool(
			values["set_current"],
		)

	if err != nil {
		errs =
			append(
				errs,
				"set_current must be TRUE, FALSE, 1, 0, yes, no, or blank",
			)
	}

	if found &&
		setCurrent &&
		currency !=
			variant.Currency {

		errs =
			append(
				errs,
				"set_current requires currency to match the variant currency",
			)
	}

	if setCurrent &&
		!effectiveAt.IsZero() &&
		effectiveAt.After(
			time.Now().
				UTC(),
		) {

		errs =
			append(
				errs,
				"future buying cost cannot be marked set_current",
			)
	}

	normalizedCost :=
		&financeImportNormalizedVariantCost{
			SKU: sku,

			UnitCostAmount: unitCost,

			Currency: currency,

			EffectiveAt: effectiveAt,

			Description: description,

			Reference: reference,

			SetCurrent: setCurrent,
		}

	if found {
		normalizedCost.VariantID =
			variant.ID
	}

	if len(
		errs,
	) == 0 {

		input :=
			FinanceVariantCostInput{
				VariantID: normalizedCost.VariantID,

				SKU: normalizedCost.SKU,

				UnitCostAmount: normalizedCost.UnitCostAmount,

				Currency: normalizedCost.Currency,

				EffectiveAt: normalizedCost.EffectiveAt,

				Source: FinanceVariantCostSourceImport,

				Description: normalizedCost.Description,

				Reference: normalizedCost.Reference,

				SetCurrent: normalizedCost.SetCurrent,
			}

		if _, err :=
			normalizeFinanceVariantCostInput(
				input,
			); err != nil {

			errs =
				append(
					errs,
					err.Error(),
				)
		}
	}

	return financeImportNormalizedRow{
			RecordType: FinanceImportRecordVariantCost,

			VariantCost: normalizedCost,
		},
		errs
}

func parseFinanceImportInt64(
	value string,
) (
	int64,
	error,
) {
	value =
		strings.TrimSpace(
			value,
		)

	if value == "" {
		return 0,
			errors.New(
				"blank integer",
			)
	}

	if strings.ContainsAny(
		value,
		".,",
	) {
		return 0,
			errors.New(
				"amount must be a whole number",
			)
	}

	return strconv.ParseInt(
		value,
		10,
		64,
	)
}

func parseFinanceImportBool(
	value string,
) (
	bool,
	error,
) {
	value =
		strings.ToLower(
			strings.TrimSpace(
				value,
			),
		)

	switch value {
	case "",
		"false",
		"0",
		"no",
		"n":

		return false,
			nil

	case "true",
		"1",
		"yes",
		"y":

		return true,
			nil

	default:
		return false,
			errors.New(
				"invalid boolean",
			)
	}
}

func parseFinanceImportDate(
	value string,
) (
	time.Time,
	error,
) {
	value =
		strings.TrimSpace(
			value,
		)

	if value == "" {
		return time.Time{},
			errors.New(
				"blank date",
			)
	}

	if parsed, err :=
		time.ParseInLocation(
			time.DateOnly,
			value,
			financeImportBusinessLocation,
		); err == nil {

		return parsed.UTC(),
			nil
	}

	if parsed, err :=
		time.Parse(
			time.RFC3339,
			value,
		); err == nil {

		return parsed.UTC(),
			nil
	}

	if parsed, err :=
		time.ParseInLocation(
			"2006-01-02 15:04:05",
			value,
			financeImportBusinessLocation,
		); err == nil {

		return parsed.UTC(),
			nil
	}

	if serial, err :=
		strconv.ParseFloat(
			value,
			64,
		); err == nil {

		if parsed, err :=
			excelize.ExcelDateToTime(
				serial,
				false,
			); err == nil {

			return time.Date(
					parsed.Year(),
					parsed.Month(),
					parsed.Day(),
					parsed.Hour(),
					parsed.Minute(),
					parsed.Second(),
					parsed.Nanosecond(),
					financeImportBusinessLocation,
				).UTC(),
				nil
		}
	}

	return time.Time{},
		errors.New(
			"invalid date",
		)
}

func createFinanceExpenseImportTx(
	ctx context.Context,
	tx pgx.Tx,
	batchID string,
	rowNumber int,
	row financeImportNormalizedExpense,
	metadata AdminActionMetadata,
) (
	string,
	error,
) {
	input, err :=
		normalizeFinanceExpenseInput(
			FinanceExpenseInput{
				Category: row.Category,

				Amount: row.Amount,

				Currency: row.Currency,

				OccurredAt: row.OccurredAt,

				Description: row.Description,

				OrderID: row.OrderID,

				ProductID: row.ProductID,

				VariantID: row.VariantID,

				StaffAccountID: row.StaffAccountID,

				WarehouseID: row.WarehouseID,

				Reference: row.Reference,
			},
		)
	if err != nil {
		return "",
			err
	}

	if err :=
		validateFinanceExpenseTargetTx(
			ctx,
			tx,
			input,
		); err != nil {

		return "",
			err
	}

	keyHash, err :=
		financeExpenseIdempotencyHash(
			metadata.StaffAccountID,
			fmt.Sprintf(
				"finance-import:%s:%d",
				batchID,
				rowNumber,
			),
		)
	if err != nil {
		return "",
			err
	}

	fingerprint, err :=
		financeExpenseRequestFingerprint(
			input,
		)
	if err != nil {
		return "",
			err
	}

	var expenseID string

	err =
		tx.QueryRow(
			ctx,
			`
				INSERT INTO finance_expenses (
					category,
					amount,
					currency,
					occurred_at,
					description,
					order_id,
					product_id,
					variant_id,
					staff_account_id,
					warehouse_id,
					reference,
					status,
					idempotency_key_hash,
					request_fingerprint,
					created_by_staff_id,
					created_at,
					updated_at
				)
				VALUES (
					$1,
					$2,
					$3,
					$4,
					$5,
					NULLIF($6, '')::uuid,
					NULLIF($7, '')::uuid,
					NULLIF($8, '')::uuid,
					NULLIF($9, '')::uuid,
					NULLIF($10, '')::uuid,
					NULLIF($11, ''),
					'active',
					$12,
					$13,
					$14::uuid,
					now(),
					now()
				)
				ON CONFLICT (idempotency_key_hash)
				DO NOTHING
				RETURNING id::text
			`,
			input.Category,
			input.Amount,
			input.Currency,
			input.OccurredAt,
			input.Description,
			input.OrderID,
			input.ProductID,
			input.VariantID,
			input.StaffAccountID,
			input.WarehouseID,
			input.Reference,
			keyHash,
			fingerprint,
			metadata.StaffAccountID,
		).Scan(
			&expenseID,
		)

	if errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		var existingFingerprint string

		if err :=
			tx.QueryRow(
				ctx,
				`
					SELECT
						id::text,
						request_fingerprint
					FROM finance_expenses
					WHERE idempotency_key_hash = $1
				`,
				keyHash,
			).Scan(
				&expenseID,
				&existingFingerprint,
			); err != nil {

			return "",
				fmt.Errorf(
					"load existing imported finance expense: %w",
					err,
				)
		}

		if existingFingerprint !=
			fingerprint {

			return "",
				ErrFinanceExpenseIdempotencyConflict
		}

		return expenseID,
			nil
	}

	if err != nil {
		return "",
			fmt.Errorf(
				"create imported finance expense: %w",
				err,
			)
	}

	return expenseID,
		nil
}

func (s *Service) getFinanceImportBatchByChecksum(
	ctx context.Context,
	checksum string,
) (
	FinanceImportBatch,
	error,
) {
	batch, err :=
		scanFinanceImportBatch(
			s.db.QueryRow(
				ctx,
				financeImportBatchSelectSQL+
					`
						WHERE file_checksum_sha256 = $1
					`,
				checksum,
			),
		)

	if errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return FinanceImportBatch{},
			ErrFinanceImportNotFound
	}

	if err != nil {
		return FinanceImportBatch{},
			fmt.Errorf(
				"find finance import by checksum: %w",
				err,
			)
	}

	return batch,
		nil
}

const financeImportBatchSelectSQL = `
	SELECT
		id::text,
		source_filename,
		file_checksum_sha256,
		template_version,
		status,
		total_rows,
		valid_rows,
		invalid_rows,
		applied_expenses,
		applied_cost_updates,
		skipped_rows,
		created_by_staff_id::text,
		COALESCE(last_error, ''),
		applied_at,
		created_at,
		updated_at
	FROM finance_import_batches
`

type financeImportBatchScanner interface {
	Scan(dest ...any) error
}

func scanFinanceImportBatch(
	scanner financeImportBatchScanner,
) (
	FinanceImportBatch,
	error,
) {
	var item FinanceImportBatch

	if err :=
		scanner.Scan(
			&item.ID,
			&item.SourceFilename,
			&item.FileChecksumSHA256,
			&item.TemplateVersion,
			&item.Status,
			&item.TotalRows,
			&item.ValidRows,
			&item.InvalidRows,
			&item.AppliedExpenses,
			&item.AppliedCostUpdates,
			&item.SkippedRows,
			&item.CreatedByStaffID,
			&item.LastError,
			&item.AppliedAt,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {

		return FinanceImportBatch{},
			err
	}

	return item,
		nil
}

func mapKeys(
	values map[string]struct{},
) []string {
	result :=
		make(
			[]string,
			0,
			len(
				values,
			),
		)

	for value := range values {

		result =
			append(
				result,
				value,
			)
	}

	return result
}

func readFinanceImportBytes(
	reader io.Reader,
	maxBytes int64,
) (
	[]byte,
	error,
) {
	limited :=
		io.LimitReader(
			reader,
			maxBytes+1,
		)

	content, err :=
		io.ReadAll(
			limited,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"read finance import workbook: %w",
				err,
			)
	}

	if int64(
		len(
			content,
		),
	) > maxBytes {

		return nil,
			fmt.Errorf(
				"%w: workbook exceeds the upload size limit",
				ErrInvalidFinanceImport,
			)
	}

	return content,
		nil
}
