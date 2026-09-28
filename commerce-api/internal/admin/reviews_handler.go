package admin

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	platformerrors "project.local/commerce-api/internal/platform/errors"
	platformvalidation "project.local/commerce-api/internal/platform/validation"
)

type reviewStatusRequest struct {
	Status string `json:"status"`
}

func (h *Handler) Reviews(
	c *gin.Context,
) {
	params, ok :=
		adminOperationalPagination(c)
	if !ok {
		return
	}

	status :=
		strings.ToLower(
			strings.TrimSpace(
				c.Query("status"),
			),
		)

	if status != "" &&
		status != "published" &&
		status != "hidden" {

		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_REVIEW_STATUS",
				"status must be published or hidden",
			),
		)

		return
	}

	rating := 0

	ratingValue :=
		strings.TrimSpace(
			c.Query("rating"),
		)

	if ratingValue != "" {
		parsedRating, err :=
			strconv.Atoi(
				ratingValue,
			)
		if err != nil ||
			parsedRating < 1 ||
			parsedRating > 5 {

			platformerrors.Write(
				c,
				platformerrors.BadRequest(
					"INVALID_REVIEW_RATING",
					"rating must be between 1 and 5",
				),
			)

			return
		}

		rating =
			parsedRating
	}

	query, ok :=
		adminOperationalQuery(c)
	if !ok {
		return
	}

	queryID := ""

	if platformvalidation.IsUUID(
		query,
	) {
		queryID =
			query
	}

	result, err :=
		h.service.ListReviews(
			c.Request.Context(),
			params,
			ReviewReadFilter{
				Status: status,

				Rating: rating,

				Query: query,

				QueryID: queryID,
			},
		)
	if err != nil {
		platformerrors.Write(
			c,
			platformerrors.Internal(
				err,
			),
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

func (h *Handler) Review(
	c *gin.Context,
) {
	reviewID :=
		strings.TrimSpace(
			c.Param(
				"review_id",
			),
		)

	if !platformvalidation.IsUUID(
		reviewID,
	) {
		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_REVIEW_ID",
				"review_id must be a valid UUID",
			),
		)

		return
	}

	result, err :=
		h.service.GetReview(
			c.Request.Context(),
			reviewID,
		)

	if errors.Is(
		err,
		ErrAdminReviewNotFound,
	) {
		platformerrors.Write(
			c,
			platformerrors.NotFound(
				"REVIEW_NOT_FOUND",
				"Review not found",
			),
		)

		return
	}

	if err != nil {
		platformerrors.Write(
			c,
			platformerrors.Internal(
				err,
			),
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

func (h *Handler) UpdateReviewStatus(
	c *gin.Context,
) {
	reviewID :=
		strings.TrimSpace(
			c.Param(
				"review_id",
			),
		)

	if !platformvalidation.IsUUID(
		reviewID,
	) {
		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_REVIEW_ID",
				"review_id must be a valid UUID",
			),
		)

		return
	}

	var request reviewStatusRequest

	if err :=
		c.ShouldBindJSON(
			&request,
		); err != nil {

		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_REVIEW_STATUS_REQUEST",
				"Invalid review status request",
			),
		)

		return
	}

	status :=
		strings.ToLower(
			strings.TrimSpace(
				request.Status,
			),
		)

	if status != "published" &&
		status != "hidden" {

		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_REVIEW_STATUS",
				"status must be published or hidden",
			),
		)

		return
	}

	metadata, ok :=
		adminActionMetadataFromContext(c)
	if !ok {
		platformerrors.Write(
			c,
			platformerrors.Unauthorized(
				"ADMIN_AUTH_REQUIRED",
				"Admin authentication required",
			),
		)

		return
	}

	result, err :=
		h.service.SetReviewStatus(
			c.Request.Context(),
			reviewID,
			status,
			metadata,
		)

	switch {
	case errors.Is(
		err,
		ErrAdminReviewNotFound,
	):
		platformerrors.Write(
			c,
			platformerrors.NotFound(
				"REVIEW_NOT_FOUND",
				"Review not found",
			),
		)

		return

	case errors.Is(
		err,
		ErrInvalidAdminReviewStatus,
	):
		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_REVIEW_STATUS",
				"Invalid review status",
			),
		)

		return

	case err != nil:
		platformerrors.Write(
			c,
			platformerrors.Internal(
				err,
			),
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
