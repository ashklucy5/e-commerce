package productrequest

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"project.local/commerce-api/internal/auth"
	platformerrors "project.local/commerce-api/internal/platform/errors"
)

func (h *Handler) ListOffers(
	c *gin.Context,
) {
	customerID, ok :=
		auth.CustomerIDFromContext(
			c,
		)

	if !ok {
		writeOfferCustomerError(
			c,
			auth.ErrInvalidAccessToken,
		)

		return
	}

	result, err :=
		h.service.
			ListCustomerOffers(
				c.Request.Context(),
				customerID,
				strings.TrimSpace(
					c.Param(
						"id",
					),
				),
			)

	if err != nil {
		writeOfferCustomerError(
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

func (h *Handler) GetOffer(
	c *gin.Context,
) {
	customerID, ok :=
		auth.CustomerIDFromContext(
			c,
		)

	if !ok {
		writeOfferCustomerError(
			c,
			auth.ErrInvalidAccessToken,
		)

		return
	}

	result, err :=
		h.service.
			GetCustomerOffer(
				c.Request.Context(),
				customerID,
				strings.TrimSpace(
					c.Param(
						"id",
					),
				),
				strings.TrimSpace(
					c.Param(
						"offerId",
					),
				),
			)

	if err != nil {
		writeOfferCustomerError(
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

func (h *Handler) AcceptOffer(
	c *gin.Context,
) {
	customerID, ok :=
		auth.CustomerIDFromContext(
			c,
		)

	if !ok {
		writeOfferCustomerError(
			c,
			auth.ErrInvalidAccessToken,
		)

		return
	}

	result, err :=
		h.service.
			AcceptCustomerOffer(
				c.Request.Context(),
				customerID,
				strings.TrimSpace(
					c.Param(
						"id",
					),
				),
				strings.TrimSpace(
					c.Param(
						"offerId",
					),
				),
			)

	if err != nil {
		writeOfferCustomerError(
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

func (h *Handler) RejectOffer(
	c *gin.Context,
) {
	customerID, ok :=
		auth.CustomerIDFromContext(
			c,
		)

	if !ok {
		writeOfferCustomerError(
			c,
			auth.ErrInvalidAccessToken,
		)

		return
	}

	result, err :=
		h.service.
			RejectCustomerOffer(
				c.Request.Context(),
				customerID,
				strings.TrimSpace(
					c.Param(
						"id",
					),
				),
				strings.TrimSpace(
					c.Param(
						"offerId",
					),
				),
			)

	if err != nil {
		writeOfferCustomerError(
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

func (h *Handler) GetConfirmation(
	c *gin.Context,
) {
	customerID, ok :=
		auth.CustomerIDFromContext(
			c,
		)

	if !ok {
		writeOfferCustomerError(
			c,
			auth.ErrInvalidAccessToken,
		)

		return
	}

	result, err :=
		h.service.
			GetCustomerConfirmation(
				c.Request.Context(),
				customerID,
				strings.TrimSpace(
					c.Param(
						"id",
					),
				),
			)

	if err != nil {
		writeOfferCustomerError(
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

func (h *AdminHandler) ListOffers(
	c *gin.Context,
) {
	result, err :=
		h.service.
			ListOffers(
				c.Request.Context(),
				strings.TrimSpace(
					c.Param(
						"id",
					),
				),
			)

	if err != nil {
		writeOfferAdminError(
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

func (h *AdminHandler) CreateOffer(
	c *gin.Context,
) {
	staffAccountID, ok :=
		adminStaffAccountID(
			c,
		)

	if !ok {
		return
	}

	var request AdminCreateOfferRequest

	if err :=
		c.ShouldBindJSON(
			&request,
		); err != nil {

		writeOfferAdminError(
			c,
			ErrInvalidRequest,
		)

		return
	}

	result, err :=
		h.service.
			CreateOffer(
				c.Request.Context(),
				staffAccountID,
				strings.TrimSpace(
					c.Param(
						"id",
					),
				),
				request,
			)

	if err != nil {
		writeOfferAdminError(
			c,
			err,
		)

		return
	}

	c.JSON(
		http.StatusCreated,
		gin.H{
			"data": result,
		},
	)
}

func (h *AdminHandler) UpdateOffer(
	c *gin.Context,
) {
	staffAccountID, ok :=
		adminStaffAccountID(
			c,
		)

	if !ok {
		return
	}

	var request AdminUpdateOfferRequest

	if err :=
		c.ShouldBindJSON(
			&request,
		); err != nil {

		writeOfferAdminError(
			c,
			ErrInvalidRequest,
		)

		return
	}

	result, err :=
		h.service.
			UpdateOffer(
				c.Request.Context(),
				staffAccountID,
				strings.TrimSpace(
					c.Param(
						"id",
					),
				),
				strings.TrimSpace(
					c.Param(
						"offerId",
					),
				),
				request,
			)

	if err != nil {
		writeOfferAdminError(
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

func (h *AdminHandler) SendOffer(
	c *gin.Context,
) {
	staffAccountID, ok :=
		adminStaffAccountID(
			c,
		)

	if !ok {
		return
	}

	result, err :=
		h.service.
			SendOffer(
				c.Request.Context(),
				staffAccountID,
				strings.TrimSpace(
					c.Param(
						"id",
					),
				),
				strings.TrimSpace(
					c.Param(
						"offerId",
					),
				),
			)

	if err != nil {
		writeOfferAdminError(
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

func (h *AdminHandler) FinalizeOffer(
	c *gin.Context,
) {
	staffAccountID, ok :=
		adminStaffAccountID(
			c,
		)

	if !ok {
		return
	}

	result, err :=
		h.service.
			FinalizeOffer(
				c.Request.Context(),
				staffAccountID,
				strings.TrimSpace(
					c.Param(
						"id",
					),
				),
				strings.TrimSpace(
					c.Param(
						"offerId",
					),
				),
			)

	if err != nil {
		writeOfferAdminError(
			c,
			err,
		)

		return
	}

	c.JSON(
		http.StatusCreated,
		gin.H{
			"data": result,
		},
	)
}

func (h *AdminHandler) GetConfirmation(
	c *gin.Context,
) {
	result, err :=
		h.service.
			GetConfirmation(
				c.Request.Context(),
				strings.TrimSpace(
					c.Param(
						"id",
					),
				),
			)

	if err != nil {
		writeOfferAdminError(
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

func writeOfferCustomerError(
	c *gin.Context,
	err error,
) {
	switch {
	case errors.Is(
		err,
		auth.ErrInvalidAccessToken,
	):
		writeJSONError(
			c,
			http.StatusUnauthorized,
			"INVALID_ACCESS_TOKEN",
			"Invalid or expired access token",
		)

	case errors.Is(
		err,
		ErrInvalidRequest,
	):
		writeJSONError(
			c,
			http.StatusBadRequest,
			"INVALID_SOURCING_OFFER",
			"Invalid sourcing offer input",
		)

	case errors.Is(
		err,
		ErrNotFound,
	):
		writeJSONError(
			c,
			http.StatusNotFound,
			"PRODUCT_REQUEST_NOT_FOUND",
			"Product request was not found",
		)

	case errors.Is(
		err,
		ErrOfferNotFound,
	):
		writeJSONError(
			c,
			http.StatusNotFound,
			"SOURCING_OFFER_NOT_FOUND",
			"Sourcing offer was not found",
		)

	case errors.Is(
		err,
		ErrConfirmationNotFound,
	):
		writeJSONError(
			c,
			http.StatusNotFound,
			"SOURCING_CONFIRMATION_NOT_FOUND",
			"Sourcing confirmation was not found",
		)

	case errors.Is(
		err,
		ErrOfferExpired,
	):
		writeJSONError(
			c,
			http.StatusConflict,
			"SOURCING_OFFER_EXPIRED",
			"This sourcing offer has expired",
		)

	case errors.Is(
		err,
		ErrOfferConflict,
	),
		errors.Is(
			err,
			ErrOfferNotActionable,
		),
		errors.Is(
			err,
			ErrOfferAlreadyFinalized,
		):

		writeJSONError(
			c,
			http.StatusConflict,
			"SOURCING_OFFER_NOT_ACTIONABLE",
			"This sourcing offer cannot be changed in its current state",
		)

	case errors.Is(
		err,
		ErrConversationClosed,
	):
		writeJSONError(
			c,
			http.StatusConflict,
			"PRODUCT_REQUEST_CONVERSATION_CLOSED",
			"This product request conversation is closed",
		)

	default:
		writeJSONError(
			c,
			http.StatusInternalServerError,
			"SOURCING_OFFER_FAILED",
			"Unable to process sourcing offer",
		)
	}
}

func writeOfferAdminError(
	c *gin.Context,
	err error,
) {
	switch {
	case errors.Is(
		err,
		ErrInvalidRequest,
	):
		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_SOURCING_OFFER",
				"Invalid sourcing offer input",
			),
		)

	case errors.Is(
		err,
		ErrNotFound,
	):
		platformerrors.Write(
			c,
			platformerrors.NotFound(
				"PRODUCT_REQUEST_NOT_FOUND",
				"Product request not found",
			),
		)

	case errors.Is(
		err,
		ErrOfferNotFound,
	):
		platformerrors.Write(
			c,
			platformerrors.NotFound(
				"SOURCING_OFFER_NOT_FOUND",
				"Sourcing offer not found",
			),
		)

	case errors.Is(
		err,
		ErrConfirmationNotFound,
	):
		platformerrors.Write(
			c,
			platformerrors.NotFound(
				"SOURCING_CONFIRMATION_NOT_FOUND",
				"Sourcing confirmation not found",
			),
		)

	case errors.Is(
		err,
		ErrOfferExpired,
	):
		platformerrors.Write(
			c,
			platformerrors.Conflict(
				"SOURCING_OFFER_EXPIRED",
				"Sourcing offer has expired",
			),
		)

	case errors.Is(
		err,
		ErrOfferConflict,
	),
		errors.Is(
			err,
			ErrOfferNotActionable,
		),
		errors.Is(
			err,
			ErrOfferAlreadyFinalized,
		):

		platformerrors.Write(
			c,
			platformerrors.Conflict(
				"SOURCING_OFFER_NOT_ACTIONABLE",
				"Sourcing offer cannot be changed in its current state",
			),
		)

	case errors.Is(
		err,
		ErrConversationClosed,
	):
		platformerrors.Write(
			c,
			platformerrors.Conflict(
				"PRODUCT_REQUEST_CONVERSATION_CLOSED",
				"This product request conversation is closed",
			),
		)

	case errors.Is(
		err,
		ErrAdminActorUnavailable,
	):
		platformerrors.Write(
			c,
			platformerrors.Conflict(
				"ADMIN_SUPPORT_ACTOR_UNAVAILABLE",
				"The authenticated staff account cannot act as a support actor",
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
}
