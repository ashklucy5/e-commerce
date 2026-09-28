package wishlist

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"project.local/commerce-api/internal/auth"
	platformerrors "project.local/commerce-api/internal/platform/errors"
	"project.local/commerce-api/internal/platform/pagination"
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

func customerIDFromContext(
	c *gin.Context,
) (string, bool) {
	customerID, ok :=
		auth.CustomerIDFromContext(
			c,
		)
	if ok {
		return customerID, true
	}

	platformerrors.Write(
		c,
		platformerrors.New(
			http.StatusUnauthorized,
			"AUTHENTICATION_REQUIRED",
			"Customer authentication is required",
		),
	)

	return "", false
}

func (h *Handler) List(
	c *gin.Context,
) {
	customerID, ok :=
		customerIDFromContext(
			c,
		)
	if !ok {
		return
	}

	params, err :=
		pagination.Parse(
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
			platformerrors.New(
				http.StatusBadRequest,
				"INVALID_PAGINATION",
				"Invalid wishlist pagination",
			),
		)

		return
	}

	result, err :=
		h.service.List(
			c.Request.Context(),
			customerID,
			params,
		)
	if err != nil {
		writeError(
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

func (h *Handler) Add(
	c *gin.Context,
) {
	customerID, ok :=
		customerIDFromContext(
			c,
		)
	if !ok {
		return
	}

	result, err :=
		h.service.Add(
			c.Request.Context(),
			customerID,
			c.Param(
				"product_id",
			),
		)
	if err != nil {
		writeError(
			c,
			err,
		)

		return
	}

	c.JSON(
		http.StatusOK,
		gin.H{
			"data": result,
		},
	)
}

func (h *Handler) Remove(
	c *gin.Context,
) {
	customerID, ok :=
		customerIDFromContext(
			c,
		)
	if !ok {
		return
	}

	if err :=
		h.service.Remove(
			c.Request.Context(),
			customerID,
			c.Param(
				"product_id",
			),
		); err != nil {

		writeError(
			c,
			err,
		)

		return
	}

	c.Status(
		http.StatusNoContent,
	)
}

func (h *Handler) State(
	c *gin.Context,
) {
	customerID, ok :=
		customerIDFromContext(
			c,
		)
	if !ok {
		return
	}

	result, err :=
		h.service.State(
			c.Request.Context(),
			customerID,
			c.Param(
				"product_id",
			),
		)
	if err != nil {
		writeError(
			c,
			err,
		)

		return
	}

	c.JSON(
		http.StatusOK,
		gin.H{
			"data": result,
		},
	)
}

func (h *Handler) Count(
	c *gin.Context,
) {
	customerID, ok :=
		customerIDFromContext(
			c,
		)
	if !ok {
		return
	}

	result, err :=
		h.service.Count(
			c.Request.Context(),
			customerID,
		)
	if err != nil {
		writeError(
			c,
			err,
		)

		return
	}

	c.JSON(
		http.StatusOK,
		gin.H{
			"data": result,
		},
	)
}

func writeError(
	c *gin.Context,
	err error,
) {
	switch {
	case errors.Is(
		err,
		ErrAuthenticationRequired,
	):
		platformerrors.Write(
			c,
			platformerrors.New(
				http.StatusUnauthorized,
				"AUTHENTICATION_REQUIRED",
				"Customer authentication is required",
			),
		)

	case errors.Is(
		err,
		ErrInvalidProductID,
	):
		platformerrors.Write(
			c,
			platformerrors.New(
				http.StatusBadRequest,
				"INVALID_PRODUCT_ID",
				"Invalid product id",
			),
		)

	case errors.Is(
		err,
		ErrProductNotFound,
	):
		platformerrors.Write(
			c,
			platformerrors.New(
				http.StatusNotFound,
				"PRODUCT_NOT_FOUND",
				"Product not found",
			),
		)

	default:
		platformerrors.Write(
			c,
			platformerrors.New(
				http.StatusInternalServerError,
				platformerrors.CodeInternal,
				"Unable to process wishlist request",
			),
		)
	}
}
