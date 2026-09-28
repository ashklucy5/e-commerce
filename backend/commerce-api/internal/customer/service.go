package customer

import (
	"context"
	"fmt"

	"project.local/commerce-api/internal/platform/storage"
)

type Service struct {
	repository *Repository
	storage    *storage.Gateway
}

func NewService(
	repository *Repository,
	storageGateway *storage.Gateway,
) *Service {
	if storageGateway == nil {
		storageGateway =
			storage.NewGateway(
				nil,
			)
	}

	return &Service{
		repository: repository,
		storage:    storageGateway,
	}
}

func (s *Service) GetProfile(
	ctx context.Context,
	customerID string,
) (Customer, error) {
	result, err :=
		s.repository.GetCustomer(
			ctx,
			customerID,
		)
	if err != nil {
		return Customer{},
			err
	}

	if err :=
		s.attachAvatar(
			ctx,
			customerID,
			&result,
		); err != nil {
		return Customer{},
			fmt.Errorf(
				"attach customer avatar: %w",
				err,
			)
	}

	return result, nil
}

func (s *Service) UpdateProfile(
	ctx context.Context,
	customerID string,
	request UpdateProfileRequest,
) (Customer, error) {
	if request.Phone == nil &&
		request.Email == nil &&
		request.FullName == nil {
		return Customer{},
			ErrNoChanges
	}

	current, err :=
		s.repository.GetCustomer(
			ctx,
			customerID,
		)
	if err != nil {
		return Customer{},
			err
	}

	phone :=
		current.Phone

	email :=
		current.Email

	fullName :=
		current.FullName

	if request.Phone != nil {
		phone, err =
			normalizePhone(
				*request.Phone,
			)
		if err != nil {
			return Customer{},
				err
		}
	}

	if request.Email != nil {
		email, err =
			normalizeEmail(
				*request.Email,
			)
		if err != nil {
			return Customer{},
				err
		}
	}

	if request.FullName != nil {
		fullName, err =
			normalizeFullName(
				*request.FullName,
			)
		if err != nil {
			return Customer{},
				err
		}
	}

	tx, err :=
		s.repository.Begin(
			ctx,
		)
	if err != nil {
		return Customer{},
			err
	}

	defer func() {
		_ = tx.Rollback(
			ctx,
		)
	}()

	_, err =
		s.repository.UpdateCustomerTx(
			ctx,
			tx,
			customerID,
			phone,
			email,
			fullName,
		)
	if err != nil {
		return Customer{},
			err
	}

	if err :=
		tx.Commit(
			ctx,
		); err != nil {
		return Customer{},
			fmt.Errorf(
				"commit customer profile update: %w",
				err,
			)
	}

	return s.GetProfile(
		ctx,
		customerID,
	)
}
