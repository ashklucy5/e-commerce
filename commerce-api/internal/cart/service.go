package cart

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"
	"time"
)

const (
	defaultCartCurrency = "BDT"

	defaultCartLifetime = 30 * 24 * time.Hour
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
) (*Cart, error) {
	cartKey, err :=
		generateCartKey()
	if err != nil {
		return nil, err
	}

	result, err :=
		s.repository.CreateCart(
			ctx,
			cartKey,
			defaultCartCurrency,
			time.Now().
				Add(defaultCartLifetime),
		)
	if err != nil {
		return nil, err
	}

	result.Items = make(
		[]Item,
		0,
	)

	result.Totals = Totals{
		Currency: result.Currency,
	}

	return &result, nil
}

func (s *Service) Get(
	ctx context.Context,
	cartKey string,
) (*Cart, error) {
	cartKey, err :=
		normalizeCartKey(
			cartKey,
		)
	if err != nil {
		return nil, err
	}

	result, err :=
		s.repository.GetCart(
			ctx,
			cartKey,
		)
	if err != nil {
		return nil, err
	}

	if err := validateCartWritable(
		result,
		time.Now(),
	); err != nil {
		return nil, err
	}

	items, err :=
		s.repository.GetItems(
			ctx,
			result.ID,
		)
	if err != nil {
		return nil, err
	}

	for index := range items {
		lineTotal, err :=
			calculateLineTotal(
				items[index].
					UnitPriceAmount,
				items[index].
					Quantity,
			)
		if err != nil {
			return nil, err
		}

		items[index].
			LineTotalAmount =
			lineTotal

		items[index].
			IsAvailable =
			items[index].
				variantActive &&
				items[index].
					productActive &&
				quantityMatchesRules(
					items[index].
						Quantity,
					items[index].
						MinimumOrderQuantity,
				) &&
				items[index].
					Quantity <=
					items[index].
						AvailableQuantity
	}

	totals, err :=
		calculateTotals(
			items,
			result.Currency,
		)
	if err != nil {
		return nil, err
	}

	result.Items = items
	result.Totals = totals

	return &result, nil
}

func (s *Service) AddItem(
	ctx context.Context,
	cartKey string,
	input AddItemRequest,
) (*Cart, error) {
	cartKey, err :=
		normalizeCartKey(
			cartKey,
		)
	if err != nil {
		return nil, err
	}

	variantID, err :=
		validateUUID(
			input.VariantID,
			ErrInvalidVariantID,
		)
	if err != nil {
		return nil, err
	}

	if input.Quantity <= 0 {
		return nil,
			ErrInvalidQuantity
	}

	tx, err :=
		s.repository.Begin(
			ctx,
		)
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	currentCart, err :=
		s.repository.LockCart(
			ctx,
			tx,
			cartKey,
		)
	if err != nil {
		return nil, err
	}

	if err := validateCartWritable(
		currentCart,
		time.Now(),
	); err != nil {
		return nil, err
	}

	variant, err :=
		s.repository.
			GetPurchasableVariant(
				ctx,
				tx,
				variantID,
			)
	if err != nil {
		return nil, err
	}

	if !strings.EqualFold(
		currentCart.Currency,
		variant.Currency,
	) {
		return nil,
			ErrCurrencyMismatch
	}

	if err := validateRequestedQuantity(
		input.Quantity,
		variant,
	); err != nil {
		return nil, err
	}

	if err := s.repository.UpsertItem(
		ctx,
		tx,
		currentCart.ID,
		variantID,
		input.Quantity,
	); err != nil {
		return nil, err
	}

	if err := s.repository.TouchCart(
		ctx,
		tx,
		currentCart.ID,
	); err != nil {
		return nil, err
	}

	if err := tx.Commit(
		ctx,
	); err != nil {
		return nil,
			fmt.Errorf(
				"commit cart item: %w",
				err,
			)
	}

	return s.Get(
		ctx,
		cartKey,
	)
}

func (s *Service) UpdateItem(
	ctx context.Context,
	cartKey string,
	itemID string,
	input UpdateItemRequest,
) (*Cart, error) {
	cartKey, err :=
		normalizeCartKey(
			cartKey,
		)
	if err != nil {
		return nil, err
	}

	itemID, err =
		validateUUID(
			itemID,
			ErrInvalidItemID,
		)
	if err != nil {
		return nil, err
	}

	if input.Quantity <= 0 {
		return nil,
			ErrInvalidQuantity
	}

	tx, err :=
		s.repository.Begin(
			ctx,
		)
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	currentCart, err :=
		s.repository.LockCart(
			ctx,
			tx,
			cartKey,
		)
	if err != nil {
		return nil, err
	}

	if err := validateCartWritable(
		currentCart,
		time.Now(),
	); err != nil {
		return nil, err
	}

	variantID, err :=
		s.repository.GetItemVariantID(
			ctx,
			tx,
			currentCart.ID,
			itemID,
		)
	if err != nil {
		return nil, err
	}

	variant, err :=
		s.repository.
			GetPurchasableVariant(
				ctx,
				tx,
				variantID,
			)
	if err != nil {
		return nil, err
	}

	if !strings.EqualFold(
		currentCart.Currency,
		variant.Currency,
	) {
		return nil,
			ErrCurrencyMismatch
	}

	if err := validateRequestedQuantity(
		input.Quantity,
		variant,
	); err != nil {
		return nil, err
	}

	if err := s.repository.UpdateItem(
		ctx,
		tx,
		currentCart.ID,
		itemID,
		input.Quantity,
	); err != nil {
		return nil, err
	}

	if err := s.repository.TouchCart(
		ctx,
		tx,
		currentCart.ID,
	); err != nil {
		return nil, err
	}

	if err := tx.Commit(
		ctx,
	); err != nil {
		return nil,
			fmt.Errorf(
				"commit cart item update: %w",
				err,
			)
	}

	return s.Get(
		ctx,
		cartKey,
	)
}

func (s *Service) RemoveItem(
	ctx context.Context,
	cartKey string,
	itemID string,
) (*Cart, error) {
	cartKey, err :=
		normalizeCartKey(
			cartKey,
		)
	if err != nil {
		return nil, err
	}

	itemID, err =
		validateUUID(
			itemID,
			ErrInvalidItemID,
		)
	if err != nil {
		return nil, err
	}

	tx, err :=
		s.repository.Begin(
			ctx,
		)
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	currentCart, err :=
		s.repository.LockCart(
			ctx,
			tx,
			cartKey,
		)
	if err != nil {
		return nil, err
	}

	if err := validateCartWritable(
		currentCart,
		time.Now(),
	); err != nil {
		return nil, err
	}

	if err := s.repository.DeleteItem(
		ctx,
		tx,
		currentCart.ID,
		itemID,
	); err != nil {
		return nil, err
	}

	if err := s.repository.TouchCart(
		ctx,
		tx,
		currentCart.ID,
	); err != nil {
		return nil, err
	}

	if err := tx.Commit(
		ctx,
	); err != nil {
		return nil,
			fmt.Errorf(
				"commit cart item removal: %w",
				err,
			)
	}

	return s.Get(
		ctx,
		cartKey,
	)
}

func generateCartKey() (
	string,
	error,
) {
	buffer := make(
		[]byte,
		32,
	)

	if _, err := rand.Read(
		buffer,
	); err != nil {
		return "",
			fmt.Errorf(
				"generate cart key: %w",
				err,
			)
	}

	return "cart_" +
			base64.RawURLEncoding.
				EncodeToString(
					buffer,
				),
		nil
}
