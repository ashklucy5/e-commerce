package admin

import (
	"errors"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"

	platformerrors "project.local/commerce-api/internal/platform/errors"
	platformpagination "project.local/commerce-api/internal/platform/pagination"
	platformvalidation "project.local/commerce-api/internal/platform/validation"
)

const financeImportMaxUploadBytes int64 = 20 << 20

func (h *Handler) CreateFinanceImport(
	c *gin.Context,
) {
	c.Request.Body =
		http.MaxBytesReader(
			c.Writer,
			c.Request.Body,
			financeImportMaxUploadBytes+
				(1<<20),
		)

	if err :=
		c.Request.ParseMultipartForm(
			financeImportMaxUploadBytes,
		); err != nil {

		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_FINANCE_IMPORT_UPLOAD",
				"Invalid finance import upload",
			),
		)

		return
	}

	fileHeader, err :=
		c.FormFile(
			"file",
		)
	if err != nil {
		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"FINANCE_IMPORT_FILE_REQUIRED",
				"A .xlsx file is required in multipart field 'file'",
			),
		)

		return
	}

	if fileHeader.Size <= 0 ||
		fileHeader.Size >
			financeImportMaxUploadBytes {

		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_FINANCE_IMPORT_SIZE",
				"Finance import workbook must be between 1 byte and 20 MB",
			),
		)

		return
	}

	filename :=
		filepath.Base(
			strings.TrimSpace(
				fileHeader.Filename,
			),
		)

	if strings.ToLower(
		filepath.Ext(
			filename,
		),
	) != ".xlsx" {

		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_FINANCE_IMPORT_FILE_TYPE",
				"Finance import must be an .xlsx workbook",
			),
		)

		return
	}

	file, err :=
		fileHeader.Open()
	if err != nil {
		platformerrors.Write(
			c,
			platformerrors.Internal(
				err,
			),
		)

		return
	}

	defer file.Close()

	content, err :=
		readFinanceImportBytes(
			file,
			financeImportMaxUploadBytes,
		)
	if err != nil {
		writeFinanceImportError(
			c,
			err,
		)

		return
	}

	metadata, ok :=
		adminMutationMetadata(
			c,
		)
	if !ok {
		return
	}

	result, err :=
		h.service.CreateFinanceImport(
			c.Request.Context(),
			filename,
			content,
			metadata,
		)

	if writeFinanceImportError(
		c,
		err,
	) {
		return
	}

	status :=
		http.StatusCreated

	if result.Duplicate {
		status =
			http.StatusOK
	}

	c.JSON(
		status,
		gin.H{
			"data": result,
		},
	)
}

func (h *Handler) FinanceImport(
	c *gin.Context,
) {
	batchID, ok :=
		financeImportIDParam(
			c,
		)
	if !ok {
		return
	}

	params, err :=
		platformpagination.Parse(
			c.Query(
				"page",
			),
			c.Query(
				"limit",
			),
		)
	if err != nil {
		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_PAGINATION",
				"Invalid pagination",
			),
		)

		return
	}

	result, err :=
		h.service.GetFinanceImport(
			c.Request.Context(),
			batchID,
			params,
		)

	if writeFinanceImportError(
		c,
		err,
	) {
		return
	}

	c.JSON(
		http.StatusOK,
		gin.H{
			"data": gin.H{
				"batch": result.Batch,

				"rows": result.Rows,
			},

			"meta": result.Meta,
		},
	)
}

func (h *Handler) ApplyFinanceImport(
	c *gin.Context,
) {
	batchID, ok :=
		financeImportIDParam(
			c,
		)
	if !ok {
		return
	}

	metadata, ok :=
		adminMutationMetadata(
			c,
		)
	if !ok {
		return
	}

	result, err :=
		h.service.ApplyFinanceImport(
			c.Request.Context(),
			batchID,
			metadata,
		)

	if writeFinanceImportError(
		c,
		err,
	) {
		return
	}

	c.JSON(
		http.StatusOK,
		gin.H{
			"data": result,
		},
	)
}

func (h *Handler) FinanceImportTemplate(
	c *gin.Context,
) {
	content, err :=
		BuildFinanceImportTemplate()
	if err != nil {
		platformerrors.Write(
			c,
			platformerrors.Internal(
				err,
			),
		)

		return
	}

	c.Header(
		"Content-Disposition",
		`attachment; filename="finance-import-template-v1.xlsx"`,
	)

	c.Data(
		http.StatusOK,
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		content,
	)
}

func financeImportIDParam(
	c *gin.Context,
) (
	string,
	bool,
) {
	batchID :=
		strings.TrimSpace(
			c.Param(
				"import_id",
			),
		)

	if !platformvalidation.IsUUID(
		batchID,
	) {
		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_FINANCE_IMPORT_ID",
				"import_id must be a valid UUID",
			),
		)

		return "",
			false
	}

	return batchID,
		true
}

func writeFinanceImportError(
	c *gin.Context,
	err error,
) bool {
	if err == nil {
		return false
	}

	switch {
	case errors.Is(
		err,
		ErrFinanceImportNotFound,
	):
		platformerrors.Write(
			c,
			platformerrors.NotFound(
				"FINANCE_IMPORT_NOT_FOUND",
				"Finance import not found",
			),
		)

	case errors.Is(
		err,
		ErrFinanceImportNotReady,
	):
		platformerrors.Write(
			c,
			platformerrors.Conflict(
				"FINANCE_IMPORT_NOT_READY",
				"Finance import is not ready to apply. Correct invalid rows and upload a new workbook, or use the already completed import.",
			),
		)

	case errors.Is(
		err,
		ErrInvalidFinanceImport,
	),
		errors.Is(
			err,
			ErrInvalidFinanceExpense,
		),
		errors.Is(
			err,
			ErrInvalidFinanceVariantCost,
		):

		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_FINANCE_IMPORT",
				err.Error(),
			),
		)

	case errors.Is(
		err,
		ErrFinanceExpenseOrderNotFound,
	),
		errors.Is(
			err,
			ErrFinanceExpenseProductNotFound,
		),
		errors.Is(
			err,
			ErrFinanceExpenseVariantNotFound,
		),
		errors.Is(
			err,
			ErrFinanceExpenseStaffNotFound,
		),
		errors.Is(
			err,
			ErrFinanceExpenseWarehouseNotFound,
		),
		errors.Is(
			err,
			ErrFinanceVariantNotFound,
		):

		platformerrors.Write(
			c,
			platformerrors.Conflict(
				"FINANCE_IMPORT_REFERENCE_CHANGED",
				"A referenced business record changed after validation. Upload a fresh workbook preview before applying.",
			),
		)

	case errors.Is(
		err,
		ErrFinanceVariantCostCurrencyMismatch,
	):
		platformerrors.Write(
			c,
			platformerrors.Conflict(
				"FINANCE_IMPORT_COST_CURRENCY_MISMATCH",
				err.Error(),
			),
		)

	case errors.Is(
		err,
		ErrFinanceExpenseIdempotencyConflict,
	):
		platformerrors.Write(
			c,
			platformerrors.Conflict(
				"FINANCE_IMPORT_IDEMPOTENCY_CONFLICT",
				"Finance import expense idempotency conflict",
			),
		)

	default:
		platformerrors.Write(
			c,
			platformerrors.Internal(
				err,
			),
		)
	}

	return true
}
