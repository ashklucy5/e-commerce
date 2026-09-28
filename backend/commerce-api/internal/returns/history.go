package returns

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"project.local/commerce-api/internal/platform/pagination"
)

var (
	ErrReturnHistoryAuthenticationRequired = errors.New(
		"customer authentication is required",
	)

	ErrInvalidReturnHistoryStatus = errors.New(
		"invalid return history status",
	)
)

type ReturnHistoryItem struct {
	ID           string `json:"id"`
	ReturnNumber string `json:"return_number"`

	OrderID     string `json:"order_id"`
	OrderNumber string `json:"order_number"`

	Status string `json:"status"`

	Currency        string `json:"currency"`
	RequestedAmount int64  `json:"requested_amount"`

	ItemCount        int64  `json:"item_count"`
	QuantityTotal    int64  `json:"quantity_total"`
	FirstProductName string `json:"first_product_name,omitempty"`

	RequestedAt time.Time `json:"requested_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type ReturnHistoryResult struct {
	Items []ReturnHistoryItem
	Meta  pagination.Meta
}

func NormalizeReturnHistoryStatus(
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
	case StatusRequested,
		StatusApproved,
		StatusRejected,
		StatusReceived,
		StatusInspected,
		StatusCompleted,
		StatusCancelled:

		return value, nil

	default:
		return "",
			fmt.Errorf(
				"%w: %q",
				ErrInvalidReturnHistoryStatus,
				value,
			)
	}
}

func (s *Service) ListHistoryForCustomer(
	ctx context.Context,
	customerID string,
	params pagination.Params,
	status string,
) (ReturnHistoryResult, error) {
	customerID =
		strings.TrimSpace(
			customerID,
		)

	if customerID == "" {
		return ReturnHistoryResult{},
			ErrReturnHistoryAuthenticationRequired
	}

	status, err :=
		NormalizeReturnHistoryStatus(
			status,
		)
	if err != nil {
		return ReturnHistoryResult{}, err
	}

	result, err :=
		s.repository.ListHistoryForCustomer(
			ctx,
			customerID,
			params,
			status,
		)
	if err != nil {
		return ReturnHistoryResult{},
			fmt.Errorf(
				"list customer return history: %w",
				err,
			)
	}

	if result.Items == nil {
		result.Items =
			make(
				[]ReturnHistoryItem,
				0,
			)
	}

	return result, nil
}
