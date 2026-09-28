package adminupload

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
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

func (h *Handler) CreateUpload(
	c *gin.Context,
) {
	var request CreateUploadRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(
			http.StatusBadRequest,
			gin.H{
				"error": "invalid request body",
			},
		)
		return
	}

	response, err := h.service.CreateUpload(
		c.Request.Context(),
		request,
	)
	if err != nil {
		handleCreateUploadError(c, err)
		return
	}

	c.JSON(
		http.StatusCreated,
		response,
	)
}

func handleCreateUploadError(
	c *gin.Context,
	err error,
) {
	switch {
	case errors.Is(err, ErrInvalidUploadRequest):
		c.JSON(
			http.StatusBadRequest,
			gin.H{
				"error": err.Error(),
			},
		)

	case errors.Is(err, ErrUnsupportedPurpose):
		c.JSON(
			http.StatusBadRequest,
			gin.H{
				"error": err.Error(),
			},
		)

	case errors.Is(err, ErrUnsupportedFileType):
		c.JSON(
			http.StatusUnsupportedMediaType,
			gin.H{
				"error": err.Error(),
			},
		)

	case errors.Is(err, ErrFileTooLarge):
		c.JSON(
			http.StatusRequestEntityTooLarge,
			gin.H{
				"error": err.Error(),
			},
		)

	case errors.Is(err, ErrStorageUnavailable):
		c.JSON(
			http.StatusServiceUnavailable,
			gin.H{
				"error": "storage is unavailable",
			},
		)

	default:
		c.JSON(
			http.StatusInternalServerError,
			gin.H{
				"error": "failed to create upload target",
			},
		)
	}
}
