package delivery

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

const GuestCheckoutKeyHeader = "X-Checkout-Key"

const maxGuestCheckoutKeyLength = 160

type trackingAccess struct {
	CustomerID string

	CheckoutKey string
}

func (r *Repository) GetTrackingAccess(
	ctx context.Context,
	orderID string,
) (trackingAccess, error) {
	var result trackingAccess

	err :=
		r.db.QueryRow(
			ctx,
			`
				SELECT
					COALESCE(
						o.customer_id::text,
						''
					),
					COALESCE(cs.checkout_key, '')

				FROM orders o

				LEFT JOIN checkout_sessions cs
					ON cs.id =
						o.checkout_id

				WHERE
					o.id =
						$1::uuid
			`,
			orderID,
		).Scan(
			&result.CustomerID,
			&result.CheckoutKey,
		)

	if errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return trackingAccess{},
			ErrOrderNotFound
	}

	if err != nil {
		return trackingAccess{},
			fmt.Errorf(
				"load tracking access: %w",
				err,
			)
	}

	return result, nil
}

func (s *Service) GetTrackingForViewer(
	ctx context.Context,
	customerID string,
	checkoutKey string,
	orderID string,
) (TrackingDetail, error) {
	customerID =
		strings.TrimSpace(
			customerID,
		)

	checkoutKey =
		normalizeGuestCheckoutKey(
			checkoutKey,
		)

	orderID =
		strings.TrimSpace(
			orderID,
		)

	if !uuidPattern.MatchString(
		orderID,
	) {
		return TrackingDetail{},
			ErrOrderNotFound
	}

	access, err :=
		s.repository.GetTrackingAccess(
			ctx,
			orderID,
		)
	if err != nil {
		return TrackingDetail{}, err
	}

	if access.CustomerID != "" {
		if customerID == "" ||
			!uuidPattern.MatchString(
				customerID,
			) ||
			access.CustomerID !=
				customerID {

			return TrackingDetail{},
				ErrCustomerOrderMismatch
		}
	} else {
		if checkoutKey == "" ||
			!constantTimeEqual(
				checkoutKey,
				access.CheckoutKey,
			) {

			return TrackingDetail{},
				ErrCustomerOrderMismatch
		}
	}

	return s.GetTracking(
		ctx,
		orderID,
	)
}

func normalizeGuestCheckoutKey(
	value string,
) string {
	value =
		strings.TrimSpace(
			value,
		)

	if value == "" ||
		len(value) >
			maxGuestCheckoutKeyLength ||
		!strings.HasPrefix(
			value,
			"chk_",
		) {

		return ""
	}

	return value
}

func constantTimeEqual(
	left string,
	right string,
) bool {
	if len(left) !=
		len(right) {

		return false
	}

	return subtle.ConstantTimeCompare(
		[]byte(
			left,
		),
		[]byte(
			right,
		),
	) == 1
}
