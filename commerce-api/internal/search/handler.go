package search

import (
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"project.local/commerce-api/internal/auth"
)

const imageSearchFileField = "image"

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

func (h *Handler) Search(
	c *gin.Context,
) {
	page, err :=
		optionalQueryInt(
			c.Query(
				"page",
			),
		)
	if err != nil {
		writeSearchError(
			c,
			ErrInvalidPage,
		)
		return
	}

	limit, err :=
		optionalQueryInt(
			c.Query(
				"limit",
			),
		)
	if err != nil {
		writeSearchError(
			c,
			ErrInvalidLimit,
		)
		return
	}

	customerID, _ :=
		auth.CustomerIDFromContext(
			c,
		)

	result, err :=
		h.service.Search(
			c.Request.Context(),
			customerID,
			Input{
				Query: c.Query(
					"q",
				),

				Page: page,

				Limit: limit,
			},
		)
	if err != nil {
		writeSearchError(
			c,
			err,
		)
		return
	}

	c.JSON(
		http.StatusOK,
		gin.H{
			"data": result.Items,

			"meta": result.Meta,
		},
	)
}

func (h *Handler) SearchImage(
	c *gin.Context,
) {
	page, err :=
		optionalQueryInt(
			c.Query(
				"page",
			),
		)
	if err != nil {
		writeSearchError(
			c,
			ErrInvalidPage,
		)
		return
	}

	limit, err :=
		optionalQueryInt(
			c.Query(
				"limit",
			),
		)
	if err != nil {
		writeSearchError(
			c,
			ErrInvalidLimit,
		)
		return
	}

	c.Request.Body =
		http.MaxBytesReader(
			c.Writer,
			c.Request.Body,
			maxSearchImageRequestBytes,
		)

	fileHeader, err :=
		c.FormFile(
			imageSearchFileField,
		)
	if err != nil {
		var maxBytesError *http.MaxBytesError

		if errors.As(
			err,
			&maxBytesError,
		) ||
			errors.Is(
				err,
				multipart.ErrMessageTooLarge,
			) {
			writeSearchError(
				c,
				ErrImageTooLarge,
			)
			return
		}

		writeSearchError(
			c,
			ErrInvalidImage,
		)
		return
	}

	if c.Request.MultipartForm != nil {
		defer c.Request.
			MultipartForm.
			RemoveAll()
	}

	file, err :=
		fileHeader.Open()
	if err != nil {
		writeSearchError(
			c,
			ErrInvalidImage,
		)
		return
	}

	defer file.Close()

	data, err :=
		io.ReadAll(
			io.LimitReader(
				file,
				maxSearchImageBytes+1,
			),
		)
	if err != nil {
		writeSearchError(
			c,
			ErrInvalidImage,
		)
		return
	}

	if len(data) == 0 {
		writeSearchError(
			c,
			ErrInvalidImage,
		)
		return
	}

	if int64(len(data)) >
		maxSearchImageBytes {
		writeSearchError(
			c,
			ErrImageTooLarge,
		)
		return
	}

	mediaType :=
		detectSearchImageMediaType(
			data,
		)

	if !isSupportedSearchImageMediaType(
		mediaType,
	) {
		writeSearchError(
			c,
			ErrUnsupportedImageType,
		)
		return
	}

	customerID, _ :=
		auth.CustomerIDFromContext(
			c,
		)

	result, err :=
		h.service.SearchImage(
			c.Request.Context(),
			customerID,
			ImageInput{
				MediaType: mediaType,

				Data: data,

				Page: page,

				Limit: limit,
			},
		)
	if err != nil {
		writeSearchError(
			c,
			err,
		)
		return
	}

	c.JSON(
		http.StatusOK,
		gin.H{
			"data": result.Items,

			"meta": result.Meta,
		},
	)
}

func detectSearchImageMediaType(
	data []byte,
) string {
	if len(data) >= 12 &&
		string(
			data[0:4],
		) == "RIFF" &&
		string(
			data[8:12],
		) == "WEBP" {
		return "image/webp"
	}

	return http.DetectContentType(
		data,
	)
}

func optionalQueryInt(
	value string,
) (int, error) {
	value =
		strings.TrimSpace(
			value,
		)

	if value == "" {
		return 0, nil
	}

	parsed, err :=
		strconv.Atoi(
			value,
		)
	if err != nil {
		return 0, err
	}

	return parsed, nil
}

func writeSearchError(
	c *gin.Context,
	err error,
) {
	switch {
	case errors.Is(
		err,
		ErrInvalidQuery,
	):
		writeSearchJSONError(
			c,
			http.StatusBadRequest,
			"INVALID_SEARCH_QUERY",
			"Search query is required and must be at most 120 characters",
		)

	case errors.Is(
		err,
		ErrInvalidPage,
	):
		writeSearchJSONError(
			c,
			http.StatusBadRequest,
			"INVALID_SEARCH_PAGE",
			"Search page must be between 1 and 100000",
		)

	case errors.Is(
		err,
		ErrInvalidLimit,
	):
		writeSearchJSONError(
			c,
			http.StatusBadRequest,
			"INVALID_SEARCH_LIMIT",
			"Search limit must be between 1 and 100",
		)

	case errors.Is(
		err,
		ErrInvalidImage,
	):
		writeSearchJSONError(
			c,
			http.StatusBadRequest,
			"INVALID_SEARCH_IMAGE",
			"A non-empty image file is required in multipart field 'image'",
		)

	case errors.Is(
		err,
		ErrImageTooLarge,
	):
		writeSearchJSONError(
			c,
			http.StatusRequestEntityTooLarge,
			"SEARCH_IMAGE_TOO_LARGE",
			"Search image must be at most 10 MB",
		)

	case errors.Is(
		err,
		ErrUnsupportedImageType,
	):
		writeSearchJSONError(
			c,
			http.StatusUnsupportedMediaType,
			"UNSUPPORTED_SEARCH_IMAGE_TYPE",
			"Search image must be JPEG, PNG, WEBP, or GIF",
		)

	case errors.Is(
		err,
		ErrImageSearchUnavailable,
	):
		writeSearchJSONError(
			c,
			http.StatusServiceUnavailable,
			"IMAGE_SEARCH_UNAVAILABLE",
			"Image search is temporarily unavailable",
		)

	default:
		writeSearchJSONError(
			c,
			http.StatusInternalServerError,
			"SEARCH_FAILED",
			"Unable to search products",
		)
	}
}

func writeSearchJSONError(
	c *gin.Context,
	status int,
	code string,
	message string,
) {
	c.JSON(
		status,
		gin.H{
			"error": gin.H{
				"code": code,

				"message": message,
			},
		},
	)
}
