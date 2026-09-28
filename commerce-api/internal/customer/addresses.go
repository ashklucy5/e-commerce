package customer

import (
	"context"
	"fmt"
	"strings"
)

func (s *Service) ListAddresses(
	ctx context.Context,
	customerID string,
) ([]Address, error) {
	return s.repository.ListAddresses(
		ctx,
		customerID,
	)
}

func (s *Service) CreateAddress(
	ctx context.Context,
	customerID string,
	request CreateAddressRequest,
) (Address, error) {
	label :=
		strings.TrimSpace(
			request.Label,
		)

	if label == "" {
		label = "Address"
	}

	label, err :=
		requiredAddressText(
			label,
			80,
		)
	if err != nil {
		return Address{},
			err
	}

	recipientName, err :=
		requiredAddressText(
			request.RecipientName,
			160,
		)
	if err != nil {
		return Address{},
			err
	}

	phone, err :=
		normalizePhone(
			request.Phone,
		)
	if err != nil {
		return Address{},
			err
	}

	addressLine1, err :=
		requiredAddressText(
			request.AddressLine1,
			255,
		)
	if err != nil {
		return Address{},
			err
	}

	addressLine2, err :=
		optionalAddressText(
			request.AddressLine2,
			255,
		)
	if err != nil {
		return Address{},
			err
	}

	city, err :=
		requiredAddressText(
			request.City,
			120,
		)
	if err != nil {
		return Address{},
			err
	}

	area, err :=
		requiredAddressText(
			request.Area,
			120,
		)
	if err != nil {
		return Address{},
			err
	}

	postalCode, err :=
		optionalAddressText(
			request.PostalCode,
			30,
		)
	if err != nil {
		return Address{},
			err
	}

	tx, err :=
		s.repository.Begin(
			ctx,
		)
	if err != nil {
		return Address{},
			err
	}

	defer func() {
		_ = tx.Rollback(
			ctx,
		)
	}()

	if err :=
		s.repository.LockCustomerTx(
			ctx,
			tx,
			customerID,
		); err != nil {
		return Address{},
			err
	}

	count, err :=
		s.repository.AddressCountTx(
			ctx,
			tx,
			customerID,
		)
	if err != nil {
		return Address{},
			err
	}

	isDefault :=
		request.IsDefault ||
			count == 0

	if isDefault {
		if err :=
			s.repository.ClearDefaultAddressesTx(
				ctx,
				tx,
				customerID,
			); err != nil {
			return Address{},
				err
		}
	}

	result, err :=
		s.repository.CreateAddressTx(
			ctx,
			tx,
			customerID,
			Address{
				Label:         label,
				RecipientName: recipientName,
				Phone:         phone,
				AddressLine1:  addressLine1,
				AddressLine2:  addressLine2,
				City:          city,
				Area:          area,
				PostalCode:    postalCode,
				IsDefault:     isDefault,
			},
		)
	if err != nil {
		return Address{},
			err
	}

	if err :=
		tx.Commit(
			ctx,
		); err != nil {
		return Address{},
			fmt.Errorf(
				"commit customer address creation: %w",
				err,
			)
	}

	return result, nil
}

func (s *Service) UpdateAddress(
	ctx context.Context,
	customerID string,
	addressID string,
	request UpdateAddressRequest,
) (Address, error) {
	if request.Label == nil &&
		request.RecipientName == nil &&
		request.Phone == nil &&
		request.AddressLine1 == nil &&
		request.AddressLine2 == nil &&
		request.City == nil &&
		request.Area == nil &&
		request.PostalCode == nil &&
		request.IsDefault == nil {
		return Address{},
			ErrNoChanges
	}

	tx, err :=
		s.repository.Begin(
			ctx,
		)
	if err != nil {
		return Address{},
			err
	}

	defer func() {
		_ = tx.Rollback(
			ctx,
		)
	}()

	if err :=
		s.repository.LockCustomerTx(
			ctx,
			tx,
			customerID,
		); err != nil {
		return Address{},
			err
	}

	address, err :=
		s.repository.LockAddressTx(
			ctx,
			tx,
			customerID,
			addressID,
		)
	if err != nil {
		return Address{},
			err
	}

	if request.Label != nil {
		address.Label, err =
			requiredAddressText(
				*request.Label,
				80,
			)
		if err != nil {
			return Address{},
				err
		}
	}

	if request.RecipientName != nil {
		address.RecipientName, err =
			requiredAddressText(
				*request.RecipientName,
				160,
			)
		if err != nil {
			return Address{},
				err
		}
	}

	if request.Phone != nil {
		address.Phone, err =
			normalizePhone(
				*request.Phone,
			)
		if err != nil {
			return Address{},
				err
		}
	}

	if request.AddressLine1 != nil {
		address.AddressLine1, err =
			requiredAddressText(
				*request.AddressLine1,
				255,
			)
		if err != nil {
			return Address{},
				err
		}
	}

	if request.AddressLine2 != nil {
		address.AddressLine2, err =
			optionalAddressText(
				*request.AddressLine2,
				255,
			)
		if err != nil {
			return Address{},
				err
		}
	}

	if request.City != nil {
		address.City, err =
			requiredAddressText(
				*request.City,
				120,
			)
		if err != nil {
			return Address{},
				err
		}
	}

	if request.Area != nil {
		address.Area, err =
			requiredAddressText(
				*request.Area,
				120,
			)
		if err != nil {
			return Address{},
				err
		}
	}

	if request.PostalCode != nil {
		address.PostalCode, err =
			optionalAddressText(
				*request.PostalCode,
				30,
			)
		if err != nil {
			return Address{},
				err
		}
	}

	if request.IsDefault != nil {
		address.IsDefault =
			*request.IsDefault
	}

	if address.IsDefault {
		if err :=
			s.repository.ClearDefaultAddressesTx(
				ctx,
				tx,
				customerID,
			); err != nil {
			return Address{},
				err
		}
	}

	result, err :=
		s.repository.UpdateAddressTx(
			ctx,
			tx,
			address,
		)
	if err != nil {
		return Address{},
			err
	}

	if !result.IsDefault {
		if err :=
			s.repository.EnsureDefaultAddressTx(
				ctx,
				tx,
				customerID,
			); err != nil {
			return Address{},
				err
		}
	}

	if err :=
		tx.Commit(
			ctx,
		); err != nil {
		return Address{},
			fmt.Errorf(
				"commit customer address update: %w",
				err,
			)
	}

	if !result.IsDefault {
		updated, err :=
			s.repository.ListAddresses(
				ctx,
				customerID,
			)
		if err == nil {
			for _, candidate := range updated {
				if candidate.ID ==
					result.ID {
					result =
						candidate

					break
				}
			}
		}
	}

	return result, nil
}

func (s *Service) DeleteAddress(
	ctx context.Context,
	customerID string,
	addressID string,
) error {
	tx, err :=
		s.repository.Begin(
			ctx,
		)
	if err != nil {
		return err
	}

	defer func() {
		_ = tx.Rollback(
			ctx,
		)
	}()

	if err :=
		s.repository.LockCustomerTx(
			ctx,
			tx,
			customerID,
		); err != nil {
		return err
	}

	address, err :=
		s.repository.LockAddressTx(
			ctx,
			tx,
			customerID,
			addressID,
		)
	if err != nil {
		return err
	}

	if err :=
		s.repository.DeleteAddressTx(
			ctx,
			tx,
			customerID,
			addressID,
		); err != nil {
		return err
	}

	if address.IsDefault {
		if err :=
			s.repository.EnsureDefaultAddressTx(
				ctx,
				tx,
				customerID,
			); err != nil {
			return err
		}
	}

	if err :=
		tx.Commit(
			ctx,
		); err != nil {
		return fmt.Errorf(
			"commit customer address deletion: %w",
			err,
		)
	}

	return nil
}
