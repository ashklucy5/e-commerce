package admin

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	platformdatabase "project.local/commerce-api/internal/platform/database"
	platformpagination "project.local/commerce-api/internal/platform/pagination"
)

var (
	ErrAdminCustomerNotFound = errors.New(
		"admin customer not found",
	)

	ErrInvalidAdminCustomerStatus = errors.New(
		"invalid admin customer status",
	)
)

type CustomerReadFilter struct {
	Status string

	Query   string
	QueryID string
}

type CustomerListItem struct {
	ID string `json:"id"`

	Phone    string `json:"phone"`
	Email    string `json:"email,omitempty"`
	FullName string `json:"full_name"`

	Status string `json:"status"`

	OrderCount         int64 `json:"order_count"`
	ReviewCount        int64 `json:"review_count"`
	ActiveSessionCount int64 `json:"active_session_count"`

	LastOrderAt *time.Time `json:"last_order_at,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CustomerAddressRead struct {
	ID string `json:"id"`

	Label string `json:"label"`

	RecipientName string `json:"recipient_name"`
	Phone         string `json:"phone"`

	AddressLine1 string `json:"address_line1"`
	AddressLine2 string `json:"address_line2,omitempty"`

	City       string `json:"city"`
	Area       string `json:"area"`
	PostalCode string `json:"postal_code,omitempty"`

	IsDefault bool `json:"is_default"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CustomerOrderSummary struct {
	ID          string `json:"id"`
	OrderNumber string `json:"order_number"`

	Status        string `json:"status"`
	PaymentStatus string `json:"payment_status"`

	TotalAmount int64  `json:"total_amount"`
	Currency    string `json:"currency"`

	CreatedAt time.Time `json:"created_at"`
}

type CustomerDetail struct {
	CustomerListItem

	DeliveredOrCompletedOrders int64 `json:"delivered_or_completed_orders"`

	Addresses    []CustomerAddressRead  `json:"addresses"`
	RecentOrders []CustomerOrderSummary `json:"recent_orders"`
}

type CustomerListResult struct {
	Items []CustomerListItem
	Meta  platformpagination.Meta
}

func (s *Service) ListCustomers(
	ctx context.Context,
	params platformpagination.Params,
	filter CustomerReadFilter,
) (
	CustomerListResult,
	error,
) {
	queryCtx, cancel :=
		adminReadContext(ctx)
	defer cancel()

	rows, err :=
		s.db.Query(
			queryCtx,
			`
				WITH selected AS (
					SELECT
						c.id,
						c.phone,
						c.email,
						c.full_name,
						c.status,
						c.created_at,
						c.updated_at,
						COUNT(*) OVER()::bigint AS total_count
					FROM customers c
					WHERE
						($1 = '' OR c.status = $1)
						AND (
							$2 = ''
							OR c.phone = $2
							OR c.email = lower($2)
							OR c.id = NULLIF($3, '')::uuid
						)
					ORDER BY
						c.created_at DESC,
						c.id DESC
					LIMIT $4
					OFFSET $5
				)
				SELECT
					s.id::text,
					s.phone,
					COALESCE(s.email, ''),
					s.full_name,
					s.status,
					COALESCE(order_stats.order_count, 0),
					COALESCE(review_stats.review_count, 0),
					COALESCE(session_stats.active_session_count, 0),
					order_stats.last_order_at,
					s.created_at,
					s.updated_at,
					s.total_count
				FROM selected s
				LEFT JOIN LATERAL (
					SELECT
						COUNT(*)::bigint AS order_count,
						MAX(o.created_at) AS last_order_at
					FROM orders o
					WHERE o.customer_id = s.id
				) order_stats
					ON true
				LEFT JOIN LATERAL (
					SELECT
						COUNT(*)::bigint AS review_count
					FROM reviews r
					WHERE
						r.customer_id = s.id
						AND r.deleted_at IS NULL
				) review_stats
					ON true
				LEFT JOIN LATERAL (
					SELECT
						COUNT(*)::bigint AS active_session_count
					FROM auth_sessions a
					WHERE
						a.customer_id = s.id
						AND a.revoked_at IS NULL
						AND a.refresh_expires_at > now()
				) session_stats
					ON true
				ORDER BY
					s.created_at DESC,
					s.id DESC
			`,
			filter.Status,
			filter.Query,
			filter.QueryID,
			params.Limit,
			params.Offset(),
		)
	if err != nil {
		return CustomerListResult{},
			fmt.Errorf(
				"list Admin customers: %w",
				err,
			)
	}
	defer rows.Close()

	items :=
		make(
			[]CustomerListItem,
			0,
			params.Limit,
		)

	var total int64

	for rows.Next() {
		var item CustomerListItem
		var lastOrderAt pgtype.Timestamptz

		if err :=
			rows.Scan(
				&item.ID,
				&item.Phone,
				&item.Email,
				&item.FullName,
				&item.Status,
				&item.OrderCount,
				&item.ReviewCount,
				&item.ActiveSessionCount,
				&lastOrderAt,
				&item.CreatedAt,
				&item.UpdatedAt,
				&total,
			); err != nil {

			return CustomerListResult{},
				fmt.Errorf(
					"scan Admin customer: %w",
					err,
				)
		}

		item.LastOrderAt =
			adminTimePointer(
				lastOrderAt,
			)

		items =
			append(
				items,
				item,
			)
	}

	if err :=
		rows.Err(); err != nil {

		return CustomerListResult{},
			fmt.Errorf(
				"iterate Admin customers: %w",
				err,
			)
	}

	return CustomerListResult{
		Items: items,

		Meta: platformpagination.NewMeta(
			params,
			total,
		),
	}, nil
}

func (s *Service) GetCustomer(
	ctx context.Context,
	customerID string,
) (
	CustomerDetail,
	error,
) {
	queryCtx, cancel :=
		adminReadContext(ctx)
	defer cancel()

	var result CustomerDetail

	err :=
		platformdatabase.WithinTxOptions(
			queryCtx,
			s.db,
			pgx.TxOptions{
				IsoLevel: pgx.RepeatableRead,

				AccessMode: pgx.ReadOnly,
			},
			func(
				ctx context.Context,
				tx pgx.Tx,
			) error {
				item, err :=
					getAdminCustomerDetail(
						ctx,
						tx,
						customerID,
					)
				if err != nil {
					return err
				}

				addresses, err :=
					listAdminCustomerAddresses(
						ctx,
						tx,
						customerID,
					)
				if err != nil {
					return err
				}

				recentOrders, err :=
					listAdminCustomerRecentOrders(
						ctx,
						tx,
						customerID,
					)
				if err != nil {
					return err
				}

				item.Addresses =
					addresses

				item.RecentOrders =
					recentOrders

				result =
					item

				return nil
			},
		)
	if err != nil {
		return CustomerDetail{},
			err
	}

	return result, nil
}

func (s *Service) SetCustomerStatus(
	ctx context.Context,
	customerID string,
	status string,
	metadata AdminActionMetadata,
) (
	CustomerDetail,
	error,
) {
	status =
		strings.ToLower(
			strings.TrimSpace(
				status,
			),
		)

	if status != "active" &&
		status != "disabled" {

		return CustomerDetail{},
			ErrInvalidAdminCustomerStatus
	}

	queryCtx, cancel :=
		adminReadContext(ctx)
	defer cancel()

	var result CustomerDetail

	err :=
		platformdatabase.WithinTxOptions(
			queryCtx,
			s.db,
			pgx.TxOptions{},
			func(
				ctx context.Context,
				tx pgx.Tx,
			) error {
				var previousStatus string

				err :=
					tx.QueryRow(
						ctx,
						`
							SELECT status
							FROM customers
							WHERE id = $1::uuid
							FOR UPDATE
						`,
						customerID,
					).Scan(
						&previousStatus,
					)

				if errors.Is(
					err,
					pgx.ErrNoRows,
				) {
					return ErrAdminCustomerNotFound
				}

				if err != nil {
					return fmt.Errorf(
						"lock Admin customer: %w",
						err,
					)
				}

				if previousStatus !=
					status {

					_, err =
						tx.Exec(
							ctx,
							`
								UPDATE customers
								SET
									status = $2,
									updated_at = now()
								WHERE id = $1::uuid
							`,
							customerID,
							status,
						)
					if err != nil {
						return fmt.Errorf(
							"update customer status: %w",
							err,
						)
					}
				}

				var revokedSessions int64

				if status == "disabled" {
					tag, err :=
						tx.Exec(
							ctx,
							`
								UPDATE auth_sessions
								SET
									revoked_at = now(),
									updated_at = now()
								WHERE
									customer_id = $1::uuid
									AND revoked_at IS NULL
							`,
							customerID,
						)
					if err != nil {
						return fmt.Errorf(
							"revoke disabled customer sessions: %w",
							err,
						)
					}

					revokedSessions =
						tag.RowsAffected()
				}

				if err :=
					insertAdminActionAuditTx(
						ctx,
						tx,
						metadata,
						adminEventCustomerStatusChanged,
						map[string]any{
							"customer_id": customerID,

							"previous_status": previousStatus,

							"new_status": status,

							"revoked_sessions": revokedSessions,
						},
					); err != nil {

					return err
				}

				item, err :=
					getAdminCustomerDetail(
						ctx,
						tx,
						customerID,
					)
				if err != nil {
					return err
				}

				addresses, err :=
					listAdminCustomerAddresses(
						ctx,
						tx,
						customerID,
					)
				if err != nil {
					return err
				}

				recentOrders, err :=
					listAdminCustomerRecentOrders(
						ctx,
						tx,
						customerID,
					)
				if err != nil {
					return err
				}

				item.Addresses =
					addresses

				item.RecentOrders =
					recentOrders

				result =
					item

				return nil
			},
		)
	if err != nil {
		return CustomerDetail{},
			err
	}

	return result, nil
}

func getAdminCustomerDetail(
	ctx context.Context,
	querier adminReadQuerier,
	customerID string,
) (
	CustomerDetail,
	error,
) {
	var result CustomerDetail
	var lastOrderAt pgtype.Timestamptz

	err :=
		querier.QueryRow(
			ctx,
			`
				SELECT
					c.id::text,
					c.phone,
					COALESCE(c.email, ''),
					c.full_name,
					c.status,
					(
						SELECT COUNT(*)::bigint
						FROM orders o
						WHERE o.customer_id = c.id
					),
					(
						SELECT COUNT(*)::bigint
						FROM reviews r
						WHERE
							r.customer_id = c.id
							AND r.deleted_at IS NULL
					),
					(
						SELECT COUNT(*)::bigint
						FROM auth_sessions a
						WHERE
							a.customer_id = c.id
							AND a.revoked_at IS NULL
							AND a.refresh_expires_at > now()
					),
					(
						SELECT MAX(o.created_at)
						FROM orders o
						WHERE o.customer_id = c.id
					),
					c.created_at,
					c.updated_at,
					(
						SELECT COUNT(*)::bigint
						FROM orders o
						WHERE
							o.customer_id = c.id
							AND o.status IN (
								'delivered',
								'completed'
							)
					)
				FROM customers c
				WHERE c.id = $1::uuid
			`,
			customerID,
		).Scan(
			&result.ID,
			&result.Phone,
			&result.Email,
			&result.FullName,
			&result.Status,
			&result.OrderCount,
			&result.ReviewCount,
			&result.ActiveSessionCount,
			&lastOrderAt,
			&result.CreatedAt,
			&result.UpdatedAt,
			&result.DeliveredOrCompletedOrders,
		)

	if errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return CustomerDetail{},
			ErrAdminCustomerNotFound
	}

	if err != nil {
		return CustomerDetail{},
			fmt.Errorf(
				"get Admin customer: %w",
				err,
			)
	}

	result.LastOrderAt =
		adminTimePointer(
			lastOrderAt,
		)

	return result, nil
}

func listAdminCustomerAddresses(
	ctx context.Context,
	querier adminReadQuerier,
	customerID string,
) (
	[]CustomerAddressRead,
	error,
) {
	rows, err :=
		querier.Query(
			ctx,
			`
				SELECT
					id::text,
					label,
					recipient_name,
					phone,
					address_line1,
					COALESCE(address_line2, ''),
					city,
					area,
					COALESCE(postal_code, ''),
					is_default,
					created_at,
					updated_at
				FROM customer_addresses
				WHERE customer_id = $1::uuid
				ORDER BY
					is_default DESC,
					created_at DESC,
					id DESC
			`,
			customerID,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"list Admin customer addresses: %w",
				err,
			)
	}
	defer rows.Close()

	result :=
		make(
			[]CustomerAddressRead,
			0,
		)

	for rows.Next() {
		var item CustomerAddressRead

		if err :=
			rows.Scan(
				&item.ID,
				&item.Label,
				&item.RecipientName,
				&item.Phone,
				&item.AddressLine1,
				&item.AddressLine2,
				&item.City,
				&item.Area,
				&item.PostalCode,
				&item.IsDefault,
				&item.CreatedAt,
				&item.UpdatedAt,
			); err != nil {

			return nil,
				fmt.Errorf(
					"scan Admin customer address: %w",
					err,
				)
		}

		result =
			append(
				result,
				item,
			)
	}

	if err :=
		rows.Err(); err != nil {

		return nil,
			fmt.Errorf(
				"iterate Admin customer addresses: %w",
				err,
			)
	}

	return result, nil
}

func listAdminCustomerRecentOrders(
	ctx context.Context,
	querier adminReadQuerier,
	customerID string,
) (
	[]CustomerOrderSummary,
	error,
) {
	rows, err :=
		querier.Query(
			ctx,
			`
				SELECT
					id::text,
					order_number,
					status,
					payment_status,
					total_amount,
					currency,
					created_at
				FROM orders
				WHERE customer_id = $1::uuid
				ORDER BY
					created_at DESC,
					id DESC
				LIMIT 20
			`,
			customerID,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"list Admin customer recent orders: %w",
				err,
			)
	}
	defer rows.Close()

	result :=
		make(
			[]CustomerOrderSummary,
			0,
			20,
		)

	for rows.Next() {
		var item CustomerOrderSummary

		if err :=
			rows.Scan(
				&item.ID,
				&item.OrderNumber,
				&item.Status,
				&item.PaymentStatus,
				&item.TotalAmount,
				&item.Currency,
				&item.CreatedAt,
			); err != nil {

			return nil,
				fmt.Errorf(
					"scan Admin customer order: %w",
					err,
				)
		}

		result =
			append(
				result,
				item,
			)
	}

	if err :=
		rows.Err(); err != nil {

		return nil,
			fmt.Errorf(
				"iterate Admin customer orders: %w",
				err,
			)
	}

	return result, nil
}
