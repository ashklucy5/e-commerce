package catalogmediawrite

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"project.local/commerce-api/internal/platform/storage"
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

func (h *Handler) CreateImage(
	c *gin.Context,
) {
	var request CreateImageRequest

	if err :=
		c.ShouldBindJSON(
			&request,
		); err != nil {
		c.JSON(
			http.StatusBadRequest,
			gin.H{
				"error": "invalid request body",
			},
		)

		return
	}

	result, err :=
		h.service.CreateImage(
			c.Request.Context(),
			CreateImageInput{
				ProductID: strings.TrimSpace(
					c.Param(
						"product_id",
					),
				),

				VariantID: request.VariantID,

				StorageKey: request.StorageKey,

				AltText: request.AltText,

				SortOrder: request.SortOrder,

				IsPrimary: request.IsPrimary,
			},
		)

	if err != nil {
		handleMediaError(
			c,
			err,
			ErrInvalidImageRequest,
			"failed to attach product image",
		)

		return
	}

	c.JSON(
		http.StatusCreated,
		result,
	)
}

func (h *Handler) Create360Frame(
	c *gin.Context,
) {
	var request Create360FrameRequest

	if err :=
		c.ShouldBindJSON(
			&request,
		); err != nil {
		c.JSON(
			http.StatusBadRequest,
			gin.H{
				"error": "invalid request body",
			},
		)

		return
	}

	result, err :=
		h.service.Create360Frame(
			c.Request.Context(),
			Create360FrameInput{
				ProductID: strings.TrimSpace(
					c.Param(
						"product_id",
					),
				),

				VariantID: request.VariantID,

				StorageKey: request.StorageKey,

				FrameIndex: request.FrameIndex,
			},
		)

	if err != nil {
		handleMediaError(
			c,
			err,
			ErrInvalid360FrameRequest,
			"failed to attach product 360 frame",
		)

		return
	}

	c.JSON(
		http.StatusCreated,
		result,
	)
}

func (h *Handler) Upsert3DModel(
	c *gin.Context,
) {
	var request Upsert3DModelRequest

	if err :=
		c.ShouldBindJSON(
			&request,
		); err != nil {
		c.JSON(
			http.StatusBadRequest,
			gin.H{
				"error": "invalid request body",
			},
		)

		return
	}

	result, err :=
		h.service.Upsert3DModel(
			c.Request.Context(),
			Upsert3DModelInput{
				ProductID: strings.TrimSpace(
					c.Param(
						"product_id",
					),
				),

				VariantID: request.VariantID,

				StorageKey: request.StorageKey,

				PosterURL: request.PosterURL,
			},
		)

	if err != nil {
		handleMediaError(
			c,
			err,
			ErrInvalid3DModelRequest,
			"failed to attach product 3D model",
		)

		return
	}

	c.JSON(
		http.StatusOK,
		result,
	)
}

func handleMediaError(
	c *gin.Context,
	err error,
	validationError error,
	internalMessage string,
) {
	switch {

	case errors.Is(
		err,
		validationError,
	):
		c.JSON(
			http.StatusBadRequest,
			gin.H{
				"error": err.Error(),
			},
		)

	case errors.Is(
		err,
		ErrStorageUnavailable,
	):
		c.JSON(
			http.StatusServiceUnavailable,
			gin.H{
				"error": "storage is unavailable",
			},
		)

	case errors.Is(
		err,
		storage.ErrNotFound,
	):
		c.JSON(
			http.StatusBadRequest,
			gin.H{
				"error": "uploaded object was not found",
			},
		)

	default:
		c.JSON(
			http.StatusInternalServerError,
			gin.H{
				"error": internalMessage,
			},
		)
	}
}
