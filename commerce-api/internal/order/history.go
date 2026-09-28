package order

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"project.local/commerce-api/internal/platform/pagination"
)

var (
	ErrCustomerAuthenticationRequired = errors.New(
		"customer authentication is required",
	)

	ErrInvalidOrderHistoryStatus = errors.New(
		"invalid order history status",
	)
)

type OrderHistoryItem struct {
	ID          string `json:"id"`
	OrderNumber string `json:"order_number"`
	OrderType   string `json:"order_type"`

	Status        string `json:"status"`
	PaymentStatus string `json:"payment_status"`
	PaymentMethod string `json:"payment_method"`

	Currency       string `json:"currency"`
	SubtotalAmount int64  `json:"subtotal_amount"`
	DiscountAmount int64  `json:"discount_amount"`
	ShippingAmount int64  `json:"shipping_amount"`
	TotalAmount    int64  `json:"total_amount"`

	DeliveryMethod string `json:"delivery_method"`

	ItemCount        int64  `json:"item_count"`
	QuantityTotal    int64  `json:"quantity_total"`
	FirstProductName string `json:"first_product_name,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type OrderHistoryResult struct {
	Items []OrderHistoryItem
	Meta  pagination.Meta
}

func NormalizeOrderHistoryStatus(
	value string,
) (string, error) {
	value =
		strings.ToLower(
			strings.TrimSpace(
				value,
			),
		)

	if value == "" ||
		value == "all" {

		return "", nil
	}

	switch value {
	case StatusPendingPayment,
		StatusAwaitingProcurement,
		StatusConfirmed,
		StatusProcessing,
		StatusShipped,
		StatusDelivered,
		StatusCompleted,
		StatusPaymentExpired,
		StatusCancelled:

		return value, nil

	default:
		return "",
			fmt.Errorf(
				"%w: %q",
				ErrInvalidOrderHistoryStatus,
				value,
			)
	}
}

func (s *Service) ListForCustomer(
	ctx context.Context,
	customerID string,
	params pagination.Params,
	status string,
) (OrderHistoryResult, error) {
	customerID =
		strings.TrimSpace(
			customerID,
		)

	if customerID == "" {
		return OrderHistoryResult{},
			ErrCustomerAuthenticationRequired
	}

	status, err :=
		NormalizeOrderHistoryStatus(
			status,
		)
	if err != nil {
		return OrderHistoryResult{}, err
	}

	result, err :=
		s.repository.ListForCustomer(
			ctx,
			customerID,
			params,
			status,
		)
	if err != nil {
		return OrderHistoryResult{},
			fmt.Errorf(
				"list customer orders: %w",
				err,
			)
	}

	if result.Items == nil {
		result.Items =
			make(
				[]OrderHistoryItem,
				0,
			)
	}

	return result, nil
}
