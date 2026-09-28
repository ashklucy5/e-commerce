package order

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	checkoutpolicy "project.local/commerce-api/internal/checkout"
	"project.local/commerce-api/internal/inventory"
)

const maxInt64Value int64 = 9223372036854775807

var uuidPattern = regexp.MustCompile(
	`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`,
)

type Service struct {
	repository      *Repository
	inventory       *inventory.Service
	paymentMethods  PaymentMethodConfig
	deliveryMethods checkoutpolicy.DeliveryMethodConfig
	promotions      promotionEvaluator

	cancellationRefunds cancellationRefundLifecycle
}

func NewService(
	repository *Repository,
	inventoryService *inventory.Service,
	paymentMethods ...PaymentMethodConfig,
) *Service {
	config :=
		AllPaymentMethodsEnabled()

	if len(paymentMethods) > 0 {
		config =
			paymentMethods[0]
	}

	return &Service{
		repository:      repository,
		inventory:       inventoryService,
		paymentMethods:  config,
		deliveryMethods: checkoutpolicy.DefaultDeliveryMethodConfig(),
	}
}

func (s *Service) PlaceOrder(
	ctx context.Context,
	customerID string,
	checkoutKey string,
) (PlaceOrderResult, error) {
	customerID =
		strings.TrimSpace(
			customerID,
		)

	checkoutKey =
		strings.TrimSpace(
			checkoutKey,
		)

	if checkoutKey == "" {
		return PlaceOrderResult{},
			ErrInvalidCheckoutKey
	}

	tx, err :=
		s.repository.Begin(
			ctx,
		)
	if err != nil {
		return PlaceOrderResult{}, err
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	checkout, err :=
		s.repository.LockCheckoutForPlaceOrder(
			ctx,
			tx,
			checkoutKey,
		)
	if err != nil {
		return PlaceOrderResult{}, err
	}

	// If this checkout already belongs to a customer,
	// only that authenticated customer may place or
	// retrieve the idempotent result for it.
	//
	// A guest request has an empty customerID, so it
	// cannot use a customer-owned checkout.
	if checkout.CustomerID != "" &&
		checkout.CustomerID != customerID {
		return PlaceOrderResult{},
			ErrCheckoutNotFound
	}

	if checkout.Status == "completed" {
		existing, err :=
			s.repository.GetOrderByCheckoutIDTx(
				ctx,
				tx,
				checkout.ID,
			)
		if err != nil {
			return PlaceOrderResult{},
				fmt.Errorf(
					"load existing order: %w",
					err,
				)
		}

		return PlaceOrderResult{
			Order:   existing,
			Created: false,
		}, nil
	}

	if checkout.Status != "active" {
		return PlaceOrderResult{},
			ErrCheckoutNotActive
	}

	// A guest checkout may be claimed when the customer
	// authenticates before placing the order.
	if checkout.CustomerID == "" &&
		customerID != "" {
		if err :=
			s.repository.AttachCheckoutCustomerTx(
				ctx,
				tx,
				checkout.ID,
				customerID,
			); err != nil {
			return PlaceOrderResult{},
				err
		}

		checkout.CustomerID =
			customerID
	}

	now :=
		time.Now().UTC()

	if !checkout.ExpiresAt.After(
		now,
	) {
		if err :=
			s.repository.MarkCheckoutExpiredTx(
				ctx,
				tx,
				checkout.ID,
			); err != nil {
			return PlaceOrderResult{}, err
		}

		if err :=
			tx.Commit(
				ctx,
			); err != nil {
			return PlaceOrderResult{},
				fmt.Errorf(
					"commit checkout expiry: %w",
					err,
				)
		}

		return PlaceOrderResult{},
			ErrCheckoutExpired
	}

	normalizeCheckoutSnapshot(
		&checkout,
	)

	if err :=
		validateCheckoutForOrder(
			checkout,
		); err != nil {
		return PlaceOrderResult{}, err
	}

	if err :=
		s.validateCheckoutDelivery(
			checkout,
		); err != nil {
		return PlaceOrderResult{}, err
	}

	if !s.paymentMethods.IsEnabled(
		checkout.PaymentMethod,
	) {
		return PlaceOrderResult{},
			ErrInvalidPaymentMethod
	}

	items, err :=
		s.repository.GetCheckoutItemsTx(
			ctx,
			tx,
			checkout.ID,
		)
	if err != nil {
		return PlaceOrderResult{}, err
	}

	if err :=
		validateCheckoutItemsForOrder(
			checkout,
			items,
		); err != nil {
		return PlaceOrderResult{}, err
	}

	if err :=
		s.validateCheckoutPromotion(
			ctx,
			checkout,
			items,
			now,
		); err != nil {
		return PlaceOrderResult{}, err
	}

	state, err :=
		buildInitialPaymentState(
			checkout.PaymentMethod,
			now,
		)
	if err != nil {
		return PlaceOrderResult{}, err
	}

	orderNumber, err :=
		generateOrderNumber(
			now,
		)
	if err != nil {
		return PlaceOrderResult{},
			fmt.Errorf(
				"generate order number: %w",
				err,
			)
	}

	orderID, err :=
		s.repository.CreateOrderTx(
			ctx,
			tx,
			createOrderInput{
				OrderNumber: orderNumber,

				Checkout: checkout,

				Status: state.OrderStatus,

				PaymentStatus: state.PaymentStatus,

				PaymentDueAt: state.PaymentDueAt,

				ConfirmedAt: state.ConfirmedAt,
			},
		)
	if err != nil {
		return PlaceOrderResult{}, err
	}

	if err :=
		s.repository.InsertOrderItemsTx(
			ctx,
			tx,
			orderID,
			items,
		); err != nil {
		return PlaceOrderResult{}, err
	}

	reserveItems :=
		make(
			[]inventory.ReserveItem,
			0,
			len(items),
		)

	for _, item := range items {
		reserveItems =
			append(
				reserveItems,
				inventory.ReserveItem{
					VariantID: item.VariantID,
					Quantity:  item.Quantity,
				},
			)
	}

	reservations, err :=
		s.inventory.ReserveReferenceTx(
			ctx,
			tx,
			inventory.ReserveReferenceInput{
				ReferenceType: inventoryReferenceType,

				ReferenceID: orderID,

				Items: reserveItems,

				ExpiresAt: state.PaymentDueAt,

				ActorType: "order",

				ActorID: orderID,
			},
		)
	if err != nil {
		if errors.Is(
			err,
			inventory.ErrInsufficientStock,
		) {
			return PlaceOrderResult{},
				ErrInsufficientStock
		}

		return PlaceOrderResult{},
			fmt.Errorf(
				"reserve order inventory: %w",
				err,
			)
	}

	if len(reservations) != len(items) {
		return PlaceOrderResult{},
			ErrInventoryReservationMismatch
	}

	if checkout.PaymentMethod ==
		PaymentMethodCOD {
		committed, err :=
			s.inventory.CommitReferenceTx(
				ctx,
				tx,
				inventory.ReferenceActionInput{
					ReferenceType: inventoryReferenceType,

					ReferenceID: orderID,

					Reason: "cod_order_confirmed",

					ActorType: "order",

					ActorID: orderID,
				},
			)
		if err != nil {
			return PlaceOrderResult{},
				fmt.Errorf(
					"commit COD inventory: %w",
					err,
				)
		}

		if committed != len(items) {
			return PlaceOrderResult{},
				ErrInventoryReservationMismatch
		}
	}

	if err :=
		s.repository.MarkCheckoutCompletedTx(
			ctx,
			tx,
			checkout.ID,
			now,
		); err != nil {
		return PlaceOrderResult{}, err
	}

	if err :=
		s.repository.MarkCartConvertedTx(
			ctx,
			tx,
			checkout.CartID,
		); err != nil {
		return PlaceOrderResult{}, err
	}

	if err :=
		s.repository.InsertEventTx(
			ctx,
			tx,
			orderID,
			eventInsert{
				EventType: EventOrderPlaced,

				ToStatus: state.OrderStatus,

				Message: "Order placed",

				ActorType: "customer",

				ActorID: customerID,
			},
		); err != nil {
		return PlaceOrderResult{}, err
	}

	if err :=
		tx.Commit(
			ctx,
		); err != nil {
		return PlaceOrderResult{},
			fmt.Errorf(
				"commit place order transaction: %w",
				err,
			)
	}

	result, err :=
		s.repository.GetByID(
			ctx,
			orderID,
		)
	if err != nil {
		return PlaceOrderResult{}, err
	}

	return PlaceOrderResult{
		Order:   result,
		Created: true,
	}, nil
}

func (s *Service) Get(
	ctx context.Context,
	orderID string,
) (Order, error) {
	orderID =
		strings.TrimSpace(
			orderID,
		)

	if !uuidPattern.MatchString(
		orderID,
	) {
		return Order{},
			ErrInvalidOrderID
	}

	return s.repository.GetByID(
		ctx,
		orderID,
	)
}

func (s *Service) GetForCustomer(
	ctx context.Context,
	customerID string,
	orderID string,
) (Order, error) {
	customerID =
		strings.TrimSpace(
			customerID,
		)

	result, err :=
		s.Get(
			ctx,
			orderID,
		)
	if err != nil {
		return Order{}, err
	}

	if result.CustomerID != "" &&
		result.CustomerID != customerID {
		return Order{},
			ErrOrderNotFound
	}

	return result, nil
}

func (s *Service) MarkPaid(
	ctx context.Context,
	orderID string,
	paidAt time.Time,
) (Order, error) {
	orderID =
		strings.TrimSpace(
			orderID,
		)

	if !uuidPattern.MatchString(
		orderID,
	) {
		return Order{},
			ErrInvalidOrderID
	}

	now :=
		time.Now().UTC()

	if paidAt.IsZero() ||
		paidAt.After(now) {
		paidAt = now
	} else {
		paidAt =
			paidAt.UTC()
	}

	tx, err :=
		s.repository.Begin(
			ctx,
		)
	if err != nil {
		return Order{}, err
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	current, err :=
		s.repository.LockOrderByIDTx(
			ctx,
			tx,
			orderID,
		)
	if err != nil {
		return Order{}, err
	}

	if current.PaymentStatus ==
		PaymentStatusPaid {
		return current, nil
	}

	if current.Status !=
		StatusPendingPayment ||
		current.PaymentStatus !=
			PaymentStatusPending {
		return Order{},
			ErrOrderNotPendingPayment
	}

	if current.PaymentDueAt == nil ||
		!now.Before(
			*current.PaymentDueAt,
		) {
		return Order{},
			ErrPaymentWindowExpired
	}

	targetStatus :=
		StatusConfirmed

	if current.OrderType ==
		OrderTypeSourcing {

		targetStatus =
			StatusAwaitingProcurement
	} else {
		committed, err :=
			s.inventory.CommitReferenceTx(
				ctx,
				tx,
				inventory.ReferenceActionInput{
					ReferenceType: inventoryReferenceType,

					ReferenceID: orderID,

					Reason: "payment_confirmed",

					ActorType: "order",

					ActorID: orderID,
				},
			)
		if err != nil {
			return Order{},
				fmt.Errorf(
					"commit paid order inventory: %w",
					err,
				)
		}

		if committed !=
			current.ItemCount {
			return Order{},
				ErrInventoryReservationMismatch
		}
	}

	if err :=
		s.repository.MarkPaidTx(
			ctx,
			tx,
			orderID,
			paidAt,
			targetStatus,
		); err != nil {
		return Order{}, err
	}

	if err :=
		s.repository.InsertEventTx(
			ctx,
			tx,
			orderID,
			eventInsert{
				EventType: EventPaymentConfirmed,

				FromStatus: StatusPendingPayment,

				ToStatus: targetStatus,

				Message: "Payment confirmed",

				ActorType: "payment",
			},
		); err != nil {
		return Order{}, err
	}

	if err :=
		tx.Commit(
			ctx,
		); err != nil {
		return Order{},
			fmt.Errorf(
				"commit paid order: %w",
				err,
			)
	}

	return s.repository.GetByID(
		ctx,
		orderID,
	)
}
func (s *Service) ExpirePendingPayments(
	ctx context.Context,
	now time.Time,
	limit int,
) (int, error) {
	if now.IsZero() {
		now =
			time.Now().UTC()
	} else {
		now =
			now.UTC()
	}

	if limit <= 0 {
		limit = 100
	}

	if limit > 1000 {
		limit = 1000
	}

	tx, err :=
		s.repository.Begin(
			ctx,
		)
	if err != nil {
		return 0, err
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	orders, err :=
		s.repository.ListDuePendingOrdersTx(
			ctx,
			tx,
			now,
			limit,
		)
	if err != nil {
		return 0, err
	}

	expired := 0

	for _, current := range orders {
		// Standard checkout orders reserve stock while payment is
		// pending, so expiry must release that reservation.
		//
		// Sourcing orders deliberately have no inventory reservation
		// before procurement, so there is nothing to release.
		if current.OrderType !=
			OrderTypeSourcing {

			_, err :=
				s.inventory.ReleaseReferenceTx(
					ctx,
					tx,
					inventory.ReferenceActionInput{
						ReferenceType: inventoryReferenceType,

						ReferenceID: current.ID,

						Reason: "payment_window_expired",

						ActorType: "order",

						ActorID: current.ID,
					},
				)
			if err != nil {
				return 0,
					fmt.Errorf(
						"release expired order inventory: %w",
						err,
					)
			}
		}

		if err :=
			s.repository.MarkPaymentExpiredTx(
				ctx,
				tx,
				current.ID,
			); err != nil {
			return 0, err
		}

		if err :=
			s.repository.InsertEventTx(
				ctx,
				tx,
				current.ID,
				eventInsert{
					EventType: EventPaymentExpired,

					FromStatus: StatusPendingPayment,

					ToStatus: StatusPaymentExpired,

					Message: "Payment window expired",

					ActorType: "system",
				},
			); err != nil {
			return 0, err
		}

		expired++
	}

	if err :=
		tx.Commit(
			ctx,
		); err != nil {
		return 0,
			fmt.Errorf(
				"commit payment expiry batch: %w",
				err,
			)
	}

	return expired, nil
}

func normalizeCheckoutSnapshot(
	checkout *checkoutSnapshot,
) {
	checkout.CustomerName =
		strings.TrimSpace(
			checkout.CustomerName,
		)

	checkout.CustomerPhone =
		strings.TrimSpace(
			checkout.CustomerPhone,
		)

	checkout.CustomerEmail =
		strings.TrimSpace(
			checkout.CustomerEmail,
		)

	checkout.ShippingAddressLine1 =
		strings.TrimSpace(
			checkout.ShippingAddressLine1,
		)

	checkout.ShippingAddressLine2 =
		strings.TrimSpace(
			checkout.ShippingAddressLine2,
		)

	checkout.ShippingCity =
		strings.TrimSpace(
			checkout.ShippingCity,
		)

	checkout.ShippingArea =
		strings.TrimSpace(
			checkout.ShippingArea,
		)

	checkout.ShippingPostalCode =
		strings.TrimSpace(
			checkout.ShippingPostalCode,
		)

	checkout.DeliveryMethod =
		strings.TrimSpace(
			checkout.DeliveryMethod,
		)

	checkout.PaymentMethod =
		strings.ToLower(
			strings.TrimSpace(
				checkout.PaymentMethod,
			),
		)

	checkout.Currency =
		strings.ToUpper(
			strings.TrimSpace(
				checkout.Currency,
			),
		)
}

func validateCheckoutForOrder(
	checkout checkoutSnapshot,
) error {
	if checkout.CartStatus != "active" {
		return ErrCartNotActive
	}

	if checkout.CustomerName == "" ||
		checkout.CustomerPhone == "" {
		return ErrCustomerDetailsRequired
	}

	if checkout.ShippingAddressLine1 == "" ||
		checkout.ShippingCity == "" ||
		checkout.ShippingArea == "" ||
		checkout.DeliveryMethod == "" {
		return ErrShippingDetailsRequired
	}

	switch checkout.PaymentMethod {
	case PaymentMethodCOD,
		PaymentMethodBKash,
		PaymentMethodNagad,
		PaymentMethodRocket,
		PaymentMethodBankTransfer:
	default:
		return ErrInvalidPaymentMethod
	}

	if checkout.SubtotalAmount < 0 ||
		checkout.DiscountAmount < 0 ||
		checkout.ShippingAmount < 0 ||
		checkout.TotalAmount < 0 {
		return ErrCheckoutChanged
	}

	if checkout.DiscountAmount >
		checkout.SubtotalAmount {
		return ErrCheckoutChanged
	}

	base :=
		checkout.SubtotalAmount -
			checkout.DiscountAmount

	if checkout.ShippingAmount >
		maxInt64Value-base {
		return ErrMoneyOverflow
	}

	if base+checkout.ShippingAmount !=
		checkout.TotalAmount {
		return ErrCheckoutChanged
	}

	return nil
}

func validateCheckoutItemsForOrder(
	checkout checkoutSnapshot,
	items []checkoutItemSnapshot,
) error {
	if len(items) == 0 {
		return ErrEmptyCheckout
	}

	var subtotal int64

	for _, item := range items {
		if !item.VariantActive ||
			!item.ProductActive {
			return ErrItemUnavailable
		}

		if item.Quantity <= 0 ||
			item.MinimumOrderQuantity <= 0 ||
			item.CurrentMinimumOrderQuantity <= 0 {
			return ErrCheckoutChanged
		}

		if item.CurrentMinimumOrderQuantity !=
			item.MinimumOrderQuantity {
			return ErrCheckoutChanged
		}

		if item.Quantity <
			item.CurrentMinimumOrderQuantity {
			return ErrBelowMinimumOrderQuantity
		}

		if item.CurrentCurrency !=
			item.Currency ||
			item.Currency !=
				checkout.Currency {
			return ErrCheckoutChanged
		}

		if item.CurrentUnitPriceAmount !=
			item.UnitPriceAmount {
			return ErrCheckoutChanged
		}

		if item.UnitPriceAmount < 0 ||
			item.LineTotalAmount < 0 {
			return ErrCheckoutChanged
		}

		quantity64 :=
			int64(
				item.Quantity,
			)

		if quantity64 > 0 &&
			item.UnitPriceAmount >
				maxInt64Value/quantity64 {
			return ErrMoneyOverflow
		}

		expected :=
			item.UnitPriceAmount *
				quantity64

		if expected !=
			item.LineTotalAmount {
			return ErrCheckoutChanged
		}

		if subtotal >
			maxInt64Value-item.LineTotalAmount {
			return ErrMoneyOverflow
		}

		subtotal +=
			item.LineTotalAmount
	}

	if subtotal !=
		checkout.SubtotalAmount {
		return ErrCheckoutChanged
	}

	return nil
}

func generateOrderNumber(
	now time.Time,
) (string, error) {
	randomBytes :=
		make(
			[]byte,
			8,
		)

	if _, err :=
		rand.Read(
			randomBytes,
		); err != nil {
		return "", err
	}

	randomPart :=
		strings.ToUpper(
			hex.EncodeToString(
				randomBytes,
			),
		)

	return fmt.Sprintf(
		"ORD-%s-%s",
		now.UTC().Format(
			"20060102",
		),
		randomPart,
	), nil
}
