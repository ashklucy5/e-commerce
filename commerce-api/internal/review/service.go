package review

import (
	"context"
	"encoding/hex"
	"math"
	"strings"
	"unicode/utf8"
)

const (
	maxReviewTitleLength = 160
	maxReviewBodyLength  = 5000
)

type Service struct {
	repository *Repository
}

func NewService(
	repository *Repository,
) *Service {
	return &Service{
		repository: repository,
	}
}

func (s *Service) Create(
	ctx context.Context,
	customerID string,
	request CreateRequest,
) (Review, error) {
	orderItemID :=
		strings.TrimSpace(
			request.OrderItemID,
		)

	if !validReviewUUID(
		orderItemID,
	) {
		return Review{},
			ErrInvalidRequest
	}

	if !validRating(
		request.Rating,
	) {
		return Review{},
			ErrInvalidRequest
	}

	title, err :=
		normalizeReviewText(
			request.Title,
			maxReviewTitleLength,
		)
	if err != nil {
		return Review{},
			err
	}

	body, err :=
		normalizeReviewText(
			request.Body,
			maxReviewBodyLength,
		)
	if err != nil {
		return Review{},
			err
	}

	purchase, err :=
		s.repository.GetPurchaseContext(
			ctx,
			customerID,
			orderItemID,
		)
	if err != nil {
		return Review{}, err
	}

	if purchase.OrderStatus != "delivered" &&
		purchase.OrderStatus != "completed" {
		return Review{},
			ErrOrderNotEligible
	}

	return s.repository.Create(
		ctx,
		customerID,
		purchase,
		request.Rating,
		title,
		body,
	)
}

func (s *Service) Update(
	ctx context.Context,
	customerID string,
	reviewID string,
	request UpdateRequest,
) (Review, error) {
	if !validReviewUUID(
		reviewID,
	) {
		return Review{},
			ErrReviewNotFound
	}

	if request.Rating == nil &&
		request.Title == nil &&
		request.Body == nil {
		return Review{},
			ErrInvalidRequest
	}

	var input updateInput

	if request.Rating != nil {
		if !validRating(
			*request.Rating,
		) {
			return Review{},
				ErrInvalidRequest
		}

		rating :=
			*request.Rating

		input.Rating =
			&rating
	}

	if request.Title != nil {
		title, err :=
			normalizeReviewText(
				request.Title,
				maxReviewTitleLength,
			)
		if err != nil {
			return Review{},
				err
		}

		input.TitleSet = true
		input.Title = title
	}

	if request.Body != nil {
		body, err :=
			normalizeReviewText(
				request.Body,
				maxReviewBodyLength,
			)
		if err != nil {
			return Review{},
				err
		}

		input.BodySet = true
		input.Body = body
	}

	return s.repository.Update(
		ctx,
		customerID,
		reviewID,
		input,
	)
}

func (s *Service) Delete(
	ctx context.Context,
	customerID string,
	reviewID string,
) error {
	if !validReviewUUID(
		reviewID,
	) {
		return ErrReviewNotFound
	}

	return s.repository.Delete(
		ctx,
		customerID,
		reviewID,
	)
}

func (s *Service) ListMine(
	ctx context.Context,
	customerID string,
	limit int,
	offset int,
) ([]Review, int, int, error) {
	limit, offset, err :=
		normalizeReviewPage(
			limit,
			offset,
		)
	if err != nil {
		return nil,
			0,
			0,
			err
	}

	items, err :=
		s.repository.ListCustomerReviews(
			ctx,
			customerID,
			limit,
			offset,
		)
	if err != nil {
		return nil,
			0,
			0,
			err
	}

	return items,
		limit,
		offset,
		nil
}

func (s *Service) ListProduct(
	ctx context.Context,
	productID string,
	limit int,
	offset int,
) ([]PublicReview, int, int, error) {
	productID =
		strings.TrimSpace(
			productID,
		)

	if !validReviewUUID(
		productID,
	) {
		return nil,
			0,
			0,
			ErrProductNotFound
	}

	exists, err :=
		s.repository.ProductExists(
			ctx,
			productID,
		)
	if err != nil {
		return nil,
			0,
			0,
			err
	}

	if !exists {
		return nil,
			0,
			0,
			ErrProductNotFound
	}

	limit, offset, err =
		normalizeReviewPage(
			limit,
			offset,
		)
	if err != nil {
		return nil,
			0,
			0,
			err
	}

	items, err :=
		s.repository.ListProductReviews(
			ctx,
			productID,
			limit,
			offset,
		)
	if err != nil {
		return nil,
			0,
			0,
			err
	}

	return items,
		limit,
		offset,
		nil
}

func (s *Service) Summary(
	ctx context.Context,
	productID string,
) (Summary, error) {
	productID =
		strings.TrimSpace(
			productID,
		)

	if !validReviewUUID(
		productID,
	) {
		return Summary{},
			ErrProductNotFound
	}

	exists, err :=
		s.repository.ProductExists(
			ctx,
			productID,
		)
	if err != nil {
		return Summary{}, err
	}

	if !exists {
		return Summary{},
			ErrProductNotFound
	}

	result, err :=
		s.repository.ProductSummary(
			ctx,
			productID,
		)
	if err != nil {
		return Summary{}, err
	}

	result.AverageRating =
		math.Round(
			result.AverageRating*100,
		) / 100

	return result, nil
}

func normalizeReviewPage(
	limit int,
	offset int,
) (int, int, error) {
	if limit <= 0 {
		limit = 20
	}

	if limit > 100 {
		limit = 100
	}

	if offset < 0 {
		return 0,
			0,
			ErrInvalidRequest
	}

	return limit,
		offset,
		nil
}

func normalizeReviewText(
	value *string,
	maxLength int,
) (*string, error) {
	if value == nil {
		return nil, nil
	}

	normalized :=
		strings.TrimSpace(
			*value,
		)

	if normalized == "" {
		return nil, nil
	}

	if utf8.RuneCountInString(
		normalized,
	) > maxLength {
		return nil,
			ErrInvalidRequest
	}

	return &normalized, nil
}

func validRating(
	rating int,
) bool {
	return rating >= 1 &&
		rating <= 5
}

func validReviewUUID(
	value string,
) bool {
	if len(value) != 36 ||
		value[8] != '-' ||
		value[13] != '-' ||
		value[18] != '-' ||
		value[23] != '-' {
		return false
	}

	compact :=
		strings.ReplaceAll(
			value,
			"-",
			"",
		)

	if len(compact) != 32 {
		return false
	}

	_, err :=
		hex.DecodeString(
			compact,
		)

	return err == nil
}
