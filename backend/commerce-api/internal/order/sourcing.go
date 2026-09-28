package order

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
)

const sourcingDeliveryMethod = "standard"

type SourcingOrderInput struct {
	ConfirmationID string
	CustomerID     string

	ProductName string

	Quantity             int
	MinimumOrderQuantity int

	UnitPriceAmount int64
	ShippingAmount  int64
	TotalAmount     int64
	Currency        string

	CustomerName  string
	CustomerPhone string
	CustomerEmail string

	ShippingAddressLine1 string
	ShippingAddressLine2 string
	ShippingCity         string
	ShippingArea         string
	ShippingPostalCode   string

	PaymentMethod string
}

type SourcingOrderCreation struct {
	OrderID string

	ProductID string
	VariantID string
	SKU       string
}

func (s *Service) CreateSourcingOrderTx(
	ctx context.Context,
	tx pgx.Tx,
	input SourcingOrderInput,
) (SourcingOrderCreation, error) {
	if tx == nil {
		return SourcingOrderCreation{},
			fmt.Errorf("create sourcing order transaction is required")
	}

	normalizeSourcingOrderInput(&input)

	subtotalAmount, err := validateSourcingOrderInput(input)
	if err != nil {
		return SourcingOrderCreation{}, err
	}

	if !s.paymentMethods.IsEnabled(input.PaymentMethod) {
		return SourcingOrderCreation{}, ErrInvalidPaymentMethod
	}

	now := time.Now().UTC()

	state, err := buildInitialSourcingPaymentState(
		input.PaymentMethod,
		now,
	)
	if err != nil {
		return SourcingOrderCreation{}, err
	}

	token := strings.ToUpper(
		strings.ReplaceAll(
			input.ConfirmationID,
			"-",
			"",
		),
	)

	productCode := "SRC-" + token
	sku := "SRC-" + token
	slug := "sourcing-" + strings.ToLower(token)

	catalog, err := s.repository.CreateSourcingCatalogTx(
		ctx,
		tx,
		sourcingCatalogInput{
			ProductCode:          productCode,
			ProductName:          input.ProductName,
			ProductSlug:          slug,
			SKU:                  sku,
			MinimumOrderQuantity: input.MinimumOrderQuantity,
			UnitPriceAmount:      input.UnitPriceAmount,
			Currency:             input.Currency,
		},
	)
	if err != nil {
		return SourcingOrderCreation{}, err
	}

	orderNumber, err := generateOrderNumber(now)
	if err != nil {
		return SourcingOrderCreation{},
			fmt.Errorf("generate sourcing order number: %w", err)
	}

	orderID, err := s.repository.CreateSourcingOrderTx(
		ctx,
		tx,
		createSourcingOrderInput{
			OrderNumber: orderNumber,
			CustomerID:  input.CustomerID,

			Status:        state.OrderStatus,
			PaymentStatus: state.PaymentStatus,
			PaymentMethod: input.PaymentMethod,

			Currency:       input.Currency,
			SubtotalAmount: subtotalAmount,
			ShippingAmount: input.ShippingAmount,
			TotalAmount:    input.TotalAmount,

			CustomerName:  input.CustomerName,
			CustomerPhone: input.CustomerPhone,
			CustomerEmail: input.CustomerEmail,

			ShippingAddressLine1: input.ShippingAddressLine1,
			ShippingAddressLine2: input.ShippingAddressLine2,
			ShippingCity:         input.ShippingCity,
			ShippingArea:         input.ShippingArea,
			ShippingPostalCode:   input.ShippingPostalCode,

			DeliveryMethod: sourcingDeliveryMethod,
			PaymentDueAt:   state.PaymentDueAt,
		},
	)
	if err != nil {
		return SourcingOrderCreation{}, err
	}

	if err := s.repository.InsertOrderItemsTx(
		ctx,
		tx,
		orderID,
		[]checkoutItemSnapshot{
			{
				VariantID:            catalog.VariantID,
				SKU:                  catalog.SKU,
				ProductName:          input.ProductName,
				Quantity:             input.Quantity,
				MinimumOrderQuantity: input.MinimumOrderQuantity,
				UnitPriceAmount:      input.UnitPriceAmount,
				LineTotalAmount:      subtotalAmount,
				Currency:             input.Currency,
			},
		},
	); err != nil {
		return SourcingOrderCreation{}, err
	}

	if err := s.repository.InsertEventTx(
		ctx,
		tx,
		orderID,
		eventInsert{
			EventType: EventOrderPlaced,
			ToStatus:  state.OrderStatus,
			Message:   "Sourcing order placed",
			ActorType: "customer",
			ActorID:   input.CustomerID,
		},
	); err != nil {
		return SourcingOrderCreation{}, err
	}

	return SourcingOrderCreation{
		OrderID:   orderID,
		ProductID: catalog.ProductID,
		VariantID: catalog.VariantID,
		SKU:       catalog.SKU,
	}, nil
}

func buildInitialSourcingPaymentState(
	paymentMethod string,
	now time.Time,
) (initialPaymentState, error) {
	state, err := buildInitialPaymentState(
		paymentMethod,
		now,
	)
	if err != nil {
		return initialPaymentState{}, err
	}

	if strings.EqualFold(
		paymentMethod,
		PaymentMethodCOD,
	) {
		state.OrderStatus = StatusAwaitingProcurement
		state.ConfirmedAt = nil
	}

	return state, nil
}

func normalizeSourcingOrderInput(
	input *SourcingOrderInput,
) {
	input.ConfirmationID =
		strings.TrimSpace(input.ConfirmationID)

	input.CustomerID =
		strings.TrimSpace(input.CustomerID)

	input.ProductName =
		strings.TrimSpace(input.ProductName)

	input.Currency =
		strings.ToUpper(
			strings.TrimSpace(input.Currency),
		)

	input.CustomerName =
		strings.TrimSpace(input.CustomerName)

	input.CustomerPhone =
		strings.TrimSpace(input.CustomerPhone)

	input.CustomerEmail =
		strings.TrimSpace(input.CustomerEmail)

	input.ShippingAddressLine1 =
		strings.TrimSpace(input.ShippingAddressLine1)

	input.ShippingAddressLine2 =
		strings.TrimSpace(input.ShippingAddressLine2)

	input.ShippingCity =
		strings.TrimSpace(input.ShippingCity)

	input.ShippingArea =
		strings.TrimSpace(input.ShippingArea)

	input.ShippingPostalCode =
		strings.TrimSpace(input.ShippingPostalCode)

	input.PaymentMethod =
		strings.ToLower(
			strings.TrimSpace(input.PaymentMethod),
		)
}

func validateSourcingOrderInput(
	input SourcingOrderInput,
) (int64, error) {
	if !uuidPattern.MatchString(input.ConfirmationID) ||
		!uuidPattern.MatchString(input.CustomerID) {
		return 0, ErrInvalidOrderID
	}

	if input.ProductName == "" ||
		utf8.RuneCountInString(input.ProductName) > 180 {
		return 0, ErrCheckoutChanged
	}

	if input.Quantity <= 0 ||
		input.MinimumOrderQuantity <= 0 ||
		input.Quantity < input.MinimumOrderQuantity {
		return 0, ErrBelowMinimumOrderQuantity
	}

	if input.UnitPriceAmount < 0 ||
		input.ShippingAmount < 0 ||
		input.TotalAmount < 0 {
		return 0, ErrCheckoutChanged
	}

	if len(input.Currency) != 3 {
		return 0, ErrCheckoutChanged
	}

	if input.UnitPriceAmount != 0 &&
		int64(input.Quantity) >
			maxInt64Value/input.UnitPriceAmount {
		return 0, ErrMoneyOverflow
	}

	subtotalAmount :=
		input.UnitPriceAmount *
			int64(input.Quantity)

	if input.ShippingAmount >
		maxInt64Value-subtotalAmount {
		return 0, ErrMoneyOverflow
	}

	if subtotalAmount+
		input.ShippingAmount !=
		input.TotalAmount {
		return 0, ErrCheckoutChanged
	}

	if input.CustomerName == "" ||
		input.CustomerPhone == "" ||
		utf8.RuneCountInString(input.CustomerName) > 160 ||
		utf8.RuneCountInString(input.CustomerPhone) > 40 ||
		utf8.RuneCountInString(input.CustomerEmail) > 255 {
		return 0, ErrCustomerDetailsRequired
	}

	if input.ShippingAddressLine1 == "" ||
		input.ShippingCity == "" ||
		input.ShippingArea == "" ||
		utf8.RuneCountInString(
			input.ShippingAddressLine1,
		) > 255 ||
		utf8.RuneCountInString(
			input.ShippingAddressLine2,
		) > 255 ||
		utf8.RuneCountInString(
			input.ShippingCity,
		) > 120 ||
		utf8.RuneCountInString(
			input.ShippingArea,
		) > 120 ||
		utf8.RuneCountInString(
			input.ShippingPostalCode,
		) > 30 {
		return 0, ErrShippingDetailsRequired
	}

	switch input.PaymentMethod {
	case PaymentMethodCOD,
		PaymentMethodBKash,
		PaymentMethodNagad,
		PaymentMethodRocket,
		PaymentMethodBankTransfer:
	default:
		return 0, ErrInvalidPaymentMethod
	}

	return subtotalAmount, nil
}
