package checkout

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"project.local/commerce-api/internal/promotion"
)

const defaultCheckoutLifetime = 24 * time.Hour

type promotionEvaluator interface {
	Evaluate(
		ctx context.Context,
		input promotion.EvaluateInput,
	) (promotion.Result, error)
}

type promotionSelection struct {
	ID             string
	Code           string
	DiscountAmount int64
}

type Service struct {
	repository      *Repository
	paymentMethods  PaymentMethodConfig
	deliveryMethods DeliveryMethodConfig
	promotions      promotionEvaluator
}

func NewService(
	repository *Repository,
	paymentMethods PaymentMethodConfig,
	promotionServices ...promotionEvaluator,
) *Service {
	var promotionService promotionEvaluator

	if len(promotionServices) > 0 {
		promotionService =
			promotionServices[0]
	}

	return &Service{
		repository:      repository,
		paymentMethods:  paymentMethods,
		deliveryMethods: DefaultDeliveryMethodConfig(),
		promotions:      promotionService,
	}
}

func (s *Service) PaymentOptions(
	ctx context.Context,
	checkoutKey string,
) ([]PaymentOption, error) {
	session, err :=
		s.Get(
			ctx,
			checkoutKey,
		)
	if err != nil {
		return nil, err
	}

	return EnabledPaymentOptions(
		s.paymentMethods,
		session.Currency,
		session.TotalAmount,
	), nil
}

func (s *Service) PaymentOptionsForCustomer(
	ctx context.Context,
	customerID string,
	checkoutKey string,
) ([]PaymentOption, error) {
	session, err :=
		s.GetForCustomer(
			ctx,
			customerID,
			checkoutKey,
		)
	if err != nil {
		return nil, err
	}

	return EnabledPaymentOptions(
		s.paymentMethods,
		session.Currency,
		session.TotalAmount,
	), nil
}

func (s *Service) GetForCustomer(
	ctx context.Context,
	customerID string,
	checkoutKey string,
) (*Session, error) {
	customerID =
		strings.TrimSpace(
			customerID,
		)

	result, err :=
		s.Get(
			ctx,
			checkoutKey,
		)
	if err != nil {
		return nil, err
	}

	if result.CustomerID != "" &&
		result.CustomerID != customerID {
		return nil,
			ErrCheckoutNotFound
	}

	return result, nil
}

func (s *Service) UpdateForCustomer(
	ctx context.Context,
	customerID string,
	checkoutKey string,
	input UpdateRequest,
) (*Session, error) {
	if _, err :=
		s.GetForCustomer(
			ctx,
			customerID,
			checkoutKey,
		); err != nil {
		return nil, err
	}

	return s.Update(
		ctx,
		checkoutKey,
		input,
	)
}

func (s *Service) CancelForCustomer(
	ctx context.Context,
	customerID string,
	checkoutKey string,
) (*Session, error) {
	if _, err :=
		s.GetForCustomer(
			ctx,
			customerID,
			checkoutKey,
		); err != nil {
		return nil, err
	}

	return s.Cancel(
		ctx,
		checkoutKey,
	)
}

func (s *Service) Start(
	ctx context.Context,
	customerID string,
	input StartRequest,
) (*Session, error) {
	customerID =
		strings.TrimSpace(
			customerID,
		)

	cartKey, err :=
		normalizeCartKey(
			input.CartKey,
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

	snapshot, err :=
		s.repository.LockCartSnapshot(
			ctx,
			tx,
			cartKey,
		)
	if err != nil {
		return nil, err
	}

	now :=
		time.Now().UTC()

	if err :=
		validateCartSnapshot(
			snapshot,
			now,
		); err != nil {
		return nil, err
	}

	items,
		subtotal,
		_,
		err :=
		buildCheckoutItems(
			snapshot,
		)
	if err != nil {
		return nil, err
	}

	active, err :=
		s.repository.GetActiveForCartTx(
			ctx,
			tx,
			snapshot.ID,
		)
	if err != nil {
		return nil, err
	}

	if active != nil &&
		!active.ExpiresAt.After(
			now,
		) {
		if err :=
			s.repository.MarkExpiredTx(
				ctx,
				tx,
				active.ID,
			); err != nil {
			return nil, err
		}

		active = nil
	}

	if active != nil &&
		active.CustomerID != "" &&
		active.CustomerID != customerID {
		return nil,
			ErrCheckoutNotFound
	}

	if active != nil &&
		active.CustomerID == "" &&
		customerID != "" {
		if err :=
			s.repository.AttachCustomerTx(
				ctx,
				tx,
				active.ID,
				customerID,
			); err != nil {
			return nil, err
		}

		active.CustomerID =
			customerID
	}

	requestedCode := ""

	explicitPromotion :=
		input.PromotionCode != nil

	if explicitPromotion {
		requestedCode =
			*input.PromotionCode
	} else if active != nil {
		requestedCode =
			active.PromotionCode
	}

	promotionSelection, err :=
		s.resolvePromotion(
			ctx,
			requestedCode,
			explicitPromotion,
			subtotal,
			snapshot.Currency,
			items,
			now,
		)
	if err != nil {
		return nil, err
	}

	expiresAt :=
		now.Add(
			defaultCheckoutLifetime,
		)

	if snapshot.ExpiresAt != nil &&
		snapshot.ExpiresAt.Before(
			expiresAt,
		) {
		expiresAt =
			*snapshot.ExpiresAt
	}

	var checkoutID string
	var checkoutKey string

	if active == nil {
		checkoutKey, err =
			generateCheckoutKey()
		if err != nil {
			return nil, err
		}

		session, err :=
			s.repository.CreateSessionTx(
				ctx,
				tx,
				checkoutKey,
				snapshot.ID,
				customerID,
				snapshot.Currency,
				subtotal,
				promotionSelection.DiscountAmount,
				promotionSelection.ID,
				promotionSelection.Code,
				expiresAt,
			)
		if err != nil {
			return nil, err
		}

		checkoutID =
			session.ID
	} else {
		checkoutID =
			active.ID

		checkoutKey =
			active.CheckoutKey

		if err :=
			s.repository.RefreshSessionTx(
				ctx,
				tx,
				active.ID,
				snapshot.Currency,
				subtotal,
				promotionSelection.DiscountAmount,
				promotionSelection.ID,
				promotionSelection.Code,
				expiresAt,
			); err != nil {
			return nil, err
		}
	}

	if err :=
		s.repository.ReplaceItemsTx(
			ctx,
			tx,
			checkoutID,
			items,
		); err != nil {
		return nil, err
	}

	if err :=
		tx.Commit(
			ctx,
		); err != nil {
		return nil,
			fmt.Errorf(
				"commit checkout session: %w",
				err,
			)
	}

	return s.Get(
		ctx,
		checkoutKey,
	)
}

func (s *Service) Get(
	ctx context.Context,
	checkoutKey string,
) (*Session, error) {
	checkoutKey, err :=
		normalizeCheckoutKey(
			checkoutKey,
		)
	if err != nil {
		return nil, err
	}

	result, err :=
		s.repository.GetByKey(
			ctx,
			checkoutKey,
		)
	if err != nil {
		return nil, err
	}

	if result.Status == "active" &&
		!result.ExpiresAt.After(
			time.Now(),
		) {
		if err :=
			s.repository.MarkExpired(
				ctx,
				result.ID,
			); err != nil {
			return nil, err
		}

		result, err =
			s.repository.GetByKey(
				ctx,
				checkoutKey,
			)
		if err != nil {
			return nil, err
		}
	}

	return &result, nil
}

func (s *Service) Update(
	ctx context.Context,
	checkoutKey string,
	input UpdateRequest,
) (*Session, error) {
	checkoutKey, err :=
		normalizeCheckoutKey(
			checkoutKey,
		)
	if err != nil {
		return nil, err
	}

	current, err :=
		s.repository.GetByKey(
			ctx,
			checkoutKey,
		)
	if err != nil {
		return nil, err
	}

	if current.Status != "active" {
		return nil,
			ErrCheckoutNotActive
	}

	now :=
		time.Now().UTC()

	if !current.ExpiresAt.After(
		now,
	) {
		if err :=
			s.repository.MarkExpired(
				ctx,
				current.ID,
			); err != nil {
			return nil, err
		}

		return nil,
			ErrCheckoutNotActive
	}

	requestedCode :=
		current.PromotionCode

	explicitPromotion :=
		input.PromotionCode != nil

	if explicitPromotion {
		requestedCode =
			*input.PromotionCode
	}

	promotionSelection, err :=
		s.resolvePromotion(
			ctx,
			requestedCode,
			explicitPromotion,
			current.SubtotalAmount,
			current.Currency,
			current.Items,
			now,
		)
	if err != nil {
		return nil, err
	}

	if err :=
		applyPromotionSelection(
			&current,
			promotionSelection,
		); err != nil {
		return nil, err
	}

	if input.CustomerName != nil {
		current.CustomerName, err =
			normalizeCustomerValue(
				*input.CustomerName,
				160,
			)
		if err != nil {
			return nil, err
		}
	}

	if input.CustomerPhone != nil {
		current.CustomerPhone, err =
			normalizeCustomerValue(
				*input.CustomerPhone,
				40,
			)
		if err != nil {
			return nil, err
		}
	}

	if input.CustomerEmail != nil {
		current.CustomerEmail, err =
			normalizeCustomerValue(
				*input.CustomerEmail,
				255,
			)
		if err != nil {
			return nil, err
		}
	}

	if input.ShippingAddressLine1 != nil {
		current.ShippingAddressLine1, err =
			normalizeShippingValue(
				*input.ShippingAddressLine1,
				255,
			)
		if err != nil {
			return nil, err
		}
	}

	if input.ShippingAddressLine2 != nil {
		current.ShippingAddressLine2, err =
			normalizeShippingValue(
				*input.ShippingAddressLine2,
				255,
			)
		if err != nil {
			return nil, err
		}
	}

	if input.ShippingCity != nil {
		current.ShippingCity, err =
			normalizeShippingValue(
				*input.ShippingCity,
				120,
			)
		if err != nil {
			return nil, err
		}
	}

	if input.ShippingArea != nil {
		current.ShippingArea, err =
			normalizeShippingValue(
				*input.ShippingArea,
				120,
			)
		if err != nil {
			return nil, err
		}
	}

	if input.ShippingPostalCode != nil {
		current.ShippingPostalCode, err =
			normalizeShippingValue(
				*input.ShippingPostalCode,
				30,
			)
		if err != nil {
			return nil, err
		}
	}

	if input.DeliveryMethod != nil {
		current.DeliveryMethod, err =
			normalizeDeliveryMethod(
				*input.DeliveryMethod,
			)
		if err != nil {
			return nil, err
		}

		if err :=
			s.applyDeliveryMethod(
				&current,
			); err != nil {

			return nil, err
		}
	}

	if input.PaymentMethod != nil {
		paymentMethod, normalizeErr :=
			normalizePaymentMethod(
				*input.PaymentMethod,
			)
		if normalizeErr != nil {
			return nil, normalizeErr
		}

		if paymentMethod != "" &&
			!s.paymentMethods.IsEnabled(
				paymentMethod,
			) {
			return nil,
				ErrInvalidPaymentMethod
		}

		current.PaymentMethod =
			paymentMethod
	}

	if err :=
		s.repository.UpdateDetails(
			ctx,
			current,
		); err != nil {
		return nil, err
	}

	return s.Get(
		ctx,
		checkoutKey,
	)
}

func (s *Service) Cancel(
	ctx context.Context,
	checkoutKey string,
) (*Session, error) {
	checkoutKey, err :=
		normalizeCheckoutKey(
			checkoutKey,
		)
	if err != nil {
		return nil, err
	}

	current, err :=
		s.repository.GetByKey(
			ctx,
			checkoutKey,
		)
	if err != nil {
		return nil, err
	}

	switch current.Status {
	case "cancelled",
		"expired":
		return &current, nil

	case "completed":
		return nil,
			ErrCheckoutNotActive
	}

	if err :=
		s.repository.MarkCancelled(
			ctx,
			current.ID,
		); err != nil {
		return nil, err
	}

	return s.Get(
		ctx,
		checkoutKey,
	)
}

func (s *Service) resolvePromotion(
	ctx context.Context,
	code string,
	explicit bool,
	subtotalAmount int64,
	currency string,
	items []Item,
	now time.Time,
) (promotionSelection, error) {
	result, err :=
		s.evaluatePromotion(
			ctx,
			code,
			subtotalAmount,
			currency,
			items,
			now,
		)
	if err == nil {
		return result, nil
	}

	if !explicit &&
		strings.TrimSpace(code) != "" &&
		(errors.Is(
			err,
			ErrPromotionNotFound,
		) ||
			errors.Is(
				err,
				ErrPromotionNotApplicable,
			)) {
		return s.evaluatePromotion(
			ctx,
			"",
			subtotalAmount,
			currency,
			items,
			now,
		)
	}

	return promotionSelection{},
		err
}

func (s *Service) evaluatePromotion(
	ctx context.Context,
	code string,
	subtotalAmount int64,
	currency string,
	items []Item,
	now time.Time,
) (promotionSelection, error) {
	code =
		strings.TrimSpace(
			code,
		)

	if s.promotions == nil {
		if code != "" {
			return promotionSelection{},
				ErrPromotionUnavailable
		}

		return promotionSelection{},
			nil
	}

	result, err :=
		s.promotions.Evaluate(
			ctx,
			promotion.EvaluateInput{
				Code: code,

				SubtotalAmount: subtotalAmount,

				Currency: currency,

				Lines: promotionLinesFromCheckoutItems(
					items,
				),

				Now: now,
			},
		)
	if err != nil {
		switch {
		case errors.Is(
			err,
			promotion.ErrInvalidCode,
		):
			return promotionSelection{},
				ErrInvalidPromotionCode

		case errors.Is(
			err,
			promotion.ErrPromotionNotFound,
		):
			return promotionSelection{},
				ErrPromotionNotFound

		case errors.Is(
			err,
			promotion.ErrPromotionNotApplicable,
		):
			return promotionSelection{},
				ErrPromotionNotApplicable

		default:
			return promotionSelection{},
				err
		}
	}

	if !result.Applied {
		return promotionSelection{},
			nil
	}

	return promotionSelection{
		ID: result.PromotionID,

		Code: result.PromotionCode,

		DiscountAmount: result.DiscountAmount,
	}, nil
}

func promotionLinesFromCheckoutItems(
	items []Item,
) []promotion.EvaluateLine {
	result :=
		make(
			[]promotion.EvaluateLine,
			0,
			len(items),
		)

	for _, item := range items {
		result =
			append(
				result,
				promotion.EvaluateLine{
					VariantID: item.VariantID,

					Quantity: item.Quantity,

					UnitPriceAmount: item.UnitPriceAmount,
				},
			)
	}

	return result
}

func applyPromotionSelection(
	session *Session,
	selection promotionSelection,
) error {
	if selection.DiscountAmount < 0 ||
		selection.DiscountAmount >
			session.SubtotalAmount {
		return ErrMoneyOverflow
	}

	base :=
		session.SubtotalAmount -
			selection.DiscountAmount

	if session.ShippingAmount < 0 ||
		base >
			math.MaxInt64-
				session.ShippingAmount {
		return ErrMoneyOverflow
	}

	session.PromotionID =
		selection.ID

	session.PromotionCode =
		selection.Code

	session.DiscountAmount =
		selection.DiscountAmount

	session.TotalAmount =
		base +
			session.ShippingAmount

	return nil
}

func generateCheckoutKey() (
	string,
	error,
) {
	buffer :=
		make(
			[]byte,
			32,
		)

	if _, err :=
		rand.Read(
			buffer,
		); err != nil {
		return "",
			fmt.Errorf(
				"generate checkout key: %w",
				err,
			)
	}

	return "chk_" +
			base64.RawURLEncoding.
				EncodeToString(
					buffer,
				),
		nil
}
