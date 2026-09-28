package wishlist

import (
	"context"
	"regexp"
	"strings"

	"project.local/commerce-api/internal/platform/pagination"
)

var productIDPattern = regexp.MustCompile(
	`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`,
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

func normalizeProductID(
	productID string,
) (string, error) {
	productID =
		strings.TrimSpace(
			productID,
		)

	if !productIDPattern.MatchString(
		productID,
	) {
		return "",
			ErrInvalidProductID
	}

	return strings.ToLower(
		productID,
	), nil
}

func requireCustomerID(
	customerID string,
) (string, error) {
	customerID =
		strings.TrimSpace(
			customerID,
		)

	if customerID == "" {
		return "",
			ErrAuthenticationRequired
	}

	return customerID, nil
}

func (s *Service) Add(
	ctx context.Context,
	customerID string,
	productID string,
) (Item, error) {
	customerID, err :=
		requireCustomerID(
			customerID,
		)
	if err != nil {
		return Item{}, err
	}

	productID, err =
		normalizeProductID(
			productID,
		)
	if err != nil {
		return Item{}, err
	}

	return s.repository.Add(
		ctx,
		customerID,
		productID,
	)
}

func (s *Service) Remove(
	ctx context.Context,
	customerID string,
	productID string,
) error {
	customerID, err :=
		requireCustomerID(
			customerID,
		)
	if err != nil {
		return err
	}

	productID, err =
		normalizeProductID(
			productID,
		)
	if err != nil {
		return err
	}

	return s.repository.Remove(
		ctx,
		customerID,
		productID,
	)
}

func (s *Service) State(
	ctx context.Context,
	customerID string,
	productID string,
) (State, error) {
	customerID, err :=
		requireCustomerID(
			customerID,
		)
	if err != nil {
		return State{}, err
	}

	productID, err =
		normalizeProductID(
			productID,
		)
	if err != nil {
		return State{}, err
	}

	exists, err :=
		s.repository.Contains(
			ctx,
			customerID,
			productID,
		)
	if err != nil {
		return State{}, err
	}

	return State{
		ProductID: productID,

		Wishlisted: exists,
	}, nil
}

func (s *Service) Count(
	ctx context.Context,
	customerID string,
) (Count, error) {
	customerID, err :=
		requireCustomerID(
			customerID,
		)
	if err != nil {
		return Count{}, err
	}

	total, err :=
		s.repository.Count(
			ctx,
			customerID,
		)
	if err != nil {
		return Count{}, err
	}

	return Count{
		Total: total,
	}, nil
}

func (s *Service) List(
	ctx context.Context,
	customerID string,
	params pagination.Params,
) (ListResult, error) {
	customerID, err :=
		requireCustomerID(
			customerID,
		)
	if err != nil {
		return ListResult{}, err
	}

	return s.repository.List(
		ctx,
		customerID,
		params,
	)
}
