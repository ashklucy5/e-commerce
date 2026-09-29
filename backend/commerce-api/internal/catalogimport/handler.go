package catalogimport

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

const (
	maxCatalogImportUploadSize int64 = 20 * 1024 * 1024

	maxCatalogImportAssetMapBytes = 2 * 1024 * 1024

	maxCatalogImportAssetMapEntries = 2000
)

type Handler struct {
	service *Service
}

func NewHandler(
	service *Service,
) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) List(
	c *gin.Context,
) {
	limit := queryInt(
		c,
		"limit",
		50,
	)

	offset := queryInt(
		c,
		"offset",
		0,
	)

	result, err := h.service.List(
		c.Request.Context(),
		limit,
		offset,
	)
	if err != nil {
		c.JSON(
			http.StatusInternalServerError,
			gin.H{
				"error": "failed to list catalog imports",
			},
		)

		return
	}

	c.JSON(
		http.StatusOK,
		result,
	)
}

func (h *Handler) Create(
	c *gin.Context,
) {
	fileHeader, err :=
		c.FormFile("file")

	if err != nil {
		c.JSON(
			http.StatusBadRequest,
			gin.H{
				"error": "XLSX file is required in multipart field \"file\"",
			},
		)

		return
	}

	if fileHeader.Size <= 0 {
		c.JSON(
			http.StatusBadRequest,
			gin.H{
				"error": "uploaded file is empty",
			},
		)

		return
	}

	if fileHeader.Size >
		maxCatalogImportUploadSize {
		c.JSON(
			http.StatusRequestEntityTooLarge,
			gin.H{
				"error": "catalog import file cannot exceed 20 MiB",
			},
		)

		return
	}

	filename :=
		strings.TrimSpace(
			fileHeader.Filename,
		)

	if !strings.EqualFold(
		fileExtension(filename),
		".xlsx",
	) {
		c.JSON(
			http.StatusUnsupportedMediaType,
			gin.H{
				"error": "catalog import file must be XLSX",
			},
		)

		return
	}

	/*
		Optional map supplied by the Admin
		ZIP-import frontend.

		Example:

		{
		  "shoe-01.webp":
		    "public/products/images/uploads/.../shoe-01.webp",
		  "shoe-02.webp":
		    "public/products/images/uploads/.../shoe-02.webp"
		}

		Existing XLSX-only imports do not send
		this field and remain fully supported.
	*/
	imageAssets, err :=
		parseImageAssetMap(
			c.PostForm(
				"asset_map",
			),
		)

	if err != nil {
		c.JSON(
			http.StatusBadRequest,
			gin.H{
				"error": err.Error(),
			},
		)

		return
	}

	file, err :=
		fileHeader.Open()

	if err != nil {
		c.JSON(
			http.StatusBadRequest,
			gin.H{
				"error": "failed to open uploaded file",
			},
		)

		return
	}

	defer file.Close()

	data, err :=
		io.ReadAll(
			io.LimitReader(
				file,
				maxCatalogImportUploadSize+1,
			),
		)

	if err != nil {
		c.JSON(
			http.StatusBadRequest,
			gin.H{
				"error": "failed to read uploaded file",
			},
		)

		return
	}

	if int64(len(data)) >
		maxCatalogImportUploadSize {
		c.JSON(
			http.StatusRequestEntityTooLarge,
			gin.H{
				"error": "catalog import file cannot exceed 20 MiB",
			},
		)

		return
	}

	checksum :=
		sha256.Sum256(
			data,
		)

	checksumHex :=
		hex.EncodeToString(
			checksum[:],
		)

	/*
		The workbook itself is currently
		staged from this multipart request.

		For ZIP imports, image files are
		uploaded directly to object storage
		before this endpoint is called.

		The asset_map links workbook filenames
		to those already-uploaded storage keys.
	*/
	storageKey :=
		fmt.Sprintf(
			"catalog-imports/http/%s",
			checksumHex,
		)

	result, err :=
		h.service.Stage(
			c.Request.Context(),
			filename,
			storageKey,
			checksumHex,
			bytes.NewReader(
				data,
			),
			imageAssets,
		)

	if err != nil {
		c.JSON(
			http.StatusUnprocessableEntity,
			gin.H{
				"error": err.Error(),
			},
		)

		return
	}

	status :=
		http.StatusCreated

	if result.Batch.Status ==
		"failed" {
		status =
			http.StatusUnprocessableEntity
	}

	c.JSON(
		status,
		result,
	)
}

func (h *Handler) Preview(
	c *gin.Context,
) {
	batchID :=
		strings.TrimSpace(
			c.Param("id"),
		)

	limit := queryInt(
		c,
		"limit",
		100,
	)

	offset := queryInt(
		c,
		"offset",
		0,
	)

	result, err :=
		h.service.Preview(
			c.Request.Context(),
			batchID,
			limit,
			offset,
		)

	if err != nil {
		handleCatalogImportReadError(
			c,
			err,
		)

		return
	}

	c.JSON(
		http.StatusOK,
		result,
	)
}

func (h *Handler) Apply(
	c *gin.Context,
) {
	batchID :=
		strings.TrimSpace(
			c.Param("id"),
		)

	result, err :=
		h.service.Apply(
			c.Request.Context(),
			batchID,
		)

	if err != nil {
		handleCatalogImportApplyError(
			c,
			err,
		)

		return
	}

	c.JSON(
		http.StatusOK,
		result,
	)
}

func handleCatalogImportReadError(
	c *gin.Context,
	err error,
) {
	if errors.Is(
		err,
		pgx.ErrNoRows,
	) ||
		strings.Contains(
			strings.ToLower(
				err.Error(),
			),
			"not found",
		) {
		c.JSON(
			http.StatusNotFound,
			gin.H{
				"error": err.Error(),
			},
		)

		return
	}

	c.JSON(
		http.StatusInternalServerError,
		gin.H{
			"error": "failed to load catalog import",
		},
	)
}

func handleCatalogImportApplyError(
	c *gin.Context,
	err error,
) {
	message :=
		strings.ToLower(
			err.Error(),
		)

	switch {
	case strings.Contains(
		message,
		"not found",
	):
		c.JSON(
			http.StatusNotFound,
			gin.H{
				"error": err.Error(),
			},
		)

	case strings.Contains(
		message,
		"must be ready",
	),
		strings.Contains(
			message,
			"invalid row",
		):
		c.JSON(
			http.StatusConflict,
			gin.H{
				"error": err.Error(),
			},
		)

	default:
		c.JSON(
			http.StatusInternalServerError,
			gin.H{
				"error": err.Error(),
			},
		)
	}
}

func parseImageAssetMap(
	raw string,
) (
	map[string]string,
	error,
) {
	raw =
		strings.TrimSpace(
			raw,
		)

	/*
		No asset map means this is a normal
		legacy XLSX import containing URLs.
	*/
	if raw == "" {
		return nil, nil
	}

	if len(raw) >
		maxCatalogImportAssetMapBytes {

		return nil,
			fmt.Errorf(
				"catalog image asset map is too large",
			)
	}

	var assets map[string]string

	if err :=
		json.Unmarshal(
			[]byte(raw),
			&assets,
		); err != nil {

		return nil,
			fmt.Errorf(
				"catalog image asset map must be valid JSON",
			)
	}

	if len(assets) >
		maxCatalogImportAssetMapEntries {

		return nil,
			fmt.Errorf(
				"catalog image asset map cannot contain more than %d entries",
				maxCatalogImportAssetMapEntries,
			)
	}

	clean :=
		make(
			map[string]string,
			len(assets),
		)

	for filename, storageKey := range assets {

		filename =
			strings.TrimSpace(
				filename,
			)

		storageKey =
			strings.TrimSpace(
				storageKey,
			)

		if filename == "" ||
			storageKey == "" {

			return nil,
				fmt.Errorf(
					"catalog image asset map contains an empty filename or storage key",
				)
		}

		clean[filename] =
			storageKey
	}

	return clean, nil
}

func queryInt(
	c *gin.Context,
	name string,
	fallback int,
) int {
	raw :=
		strings.TrimSpace(
			c.Query(name),
		)

	if raw == "" {
		return fallback
	}

	value, err :=
		strconv.Atoi(raw)

	if err != nil {
		return fallback
	}

	return value
}

func fileExtension(
	filename string,
) string {
	index :=
		strings.LastIndex(
			filename,
			".",
		)

	if index < 0 {
		return ""
	}

	return strings.ToLower(
		filename[index:],
	)
}
