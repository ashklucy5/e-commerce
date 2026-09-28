package admin

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	platformdatabase "project.local/commerce-api/internal/platform/database"
	platformpagination "project.local/commerce-api/internal/platform/pagination"
)

var (
	ErrAdminReviewNotFound = errors.New(
		"admin review not found",
	)

	ErrInvalidAdminReviewStatus = errors.New(
		"invalid admin review status",
	)
)

type ReviewReadFilter struct {
	Status string
	Rating int

	Query   string
	QueryID string
}

type AdminReview struct {
	ID string `json:"id"`

	CustomerID    string `json:"customer_id"`
	CustomerName  string `json:"customer_name"`
	CustomerPhone string `json:"customer_phone"`

	OrderID     string `json:"order_id"`
	OrderNumber string `json:"order_number"`

	OrderItemID string `json:"order_item_id"`

	ProductID string `json:"product_id"`
	VariantID string `json:"variant_id"`

	SKU         string `json:"sku"`
	ProductName string `json:"product_name"`

	Rating int `json:"rating"`

	Title string `json:"title,omitempty"`
	Body  string `json:"body,omitempty"`

	Status string `json:"status"`

	VerifiedPurchase bool `json:"verified_purchase"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ReviewListResult struct {
	Items []AdminReview
	Meta  platformpagination.Meta
}

func (s *Service) ListReviews(
	ctx context.Context,
	params platformpagination.Params,
	filter ReviewReadFilter,
) (
	ReviewListResult,
	error,
) {
	queryCtx, cancel :=
		adminReadContext(ctx)
	defer cancel()

	rows, err :=
		s.db.Query(
			queryCtx,
			`
				SELECT
					r.id::text,
					r.customer_id::text,
					c.full_name,
					c.phone,
					o.id::text,
					o.order_number,
					r.order_item_id::text,
					v.product_id::text,
					oi.variant_id::text,
					oi.sku,
					oi.product_name,
					r.rating,
					COALESCE(r.title, ''),
					COALESCE(r.body, ''),
					r.status,
					r.created_at,
					r.updated_at,
					COUNT(*) OVER()::bigint
				FROM reviews r
				JOIN customers c
					ON c.id = r.customer_id
				JOIN order_items oi
					ON oi.id = r.order_item_id
				JOIN orders o
					ON o.id = oi.order_id
				JOIN product_variants v
					ON v.id = oi.variant_id
				WHERE
					r.deleted_at IS NULL
					AND ($1 = '' OR r.status = $1)
					AND ($2 = 0 OR r.rating = $2)
					AND (
						$3 = ''
						OR r.id = NULLIF($4, '')::uuid
						OR r.customer_id = NULLIF($4, '')::uuid
						OR r.order_item_id = NULLIF($4, '')::uuid
						OR o.id = NULLIF($4, '')::uuid
						OR oi.variant_id = NULLIF($4, '')::uuid
						OR v.product_id = NULLIF($4, '')::uuid
						OR o.order_number = upper($3)
						OR oi.sku = $3
						OR c.phone = $3
					)
				ORDER BY
					r.created_at DESC,
					r.id DESC
				LIMIT $5
				OFFSET $6
			`,
			filter.Status,
			filter.Rating,
			filter.Query,
			filter.QueryID,
			params.Limit,
			params.Offset(),
		)
	if err != nil {
		return ReviewListResult{},
			fmt.Errorf(
				"list Admin reviews: %w",
				err,
			)
	}
	defer rows.Close()

	items :=
		make(
			[]AdminReview,
			0,
			params.Limit,
		)

	var total int64

	for rows.Next() {
		var item AdminReview

		if err :=
			rows.Scan(
				&item.ID,
				&item.CustomerID,
				&item.CustomerName,
				&item.CustomerPhone,
				&item.OrderID,
				&item.OrderNumber,
				&item.OrderItemID,
				&item.ProductID,
				&item.VariantID,
				&item.SKU,
				&item.ProductName,
				&item.Rating,
				&item.Title,
				&item.Body,
				&item.Status,
				&item.CreatedAt,
				&item.UpdatedAt,
				&total,
			); err != nil {

			return ReviewListResult{},
				fmt.Errorf(
					"scan Admin review: %w",
					err,
				)
		}

		item.VerifiedPurchase =
			true

		items =
			append(
				items,
				item,
			)
	}

	if err :=
		rows.Err(); err != nil {

		return ReviewListResult{},
			fmt.Errorf(
				"iterate Admin reviews: %w",
				err,
			)
	}

	return ReviewListResult{
		Items: items,

		Meta: platformpagination.NewMeta(
			params,
			total,
		),
	}, nil
}

func (s *Service) GetReview(
	ctx context.Context,
	reviewID string,
) (
	AdminReview,
	error,
) {
	queryCtx, cancel :=
		adminReadContext(ctx)
	defer cancel()

	return getAdminReview(
		queryCtx,
		s.db,
		reviewID,
	)
}

func (s *Service) SetReviewStatus(
	ctx context.Context,
	reviewID string,
	status string,
	metadata AdminActionMetadata,
) (
	AdminReview,
	error,
) {
	status =
		strings.ToLower(
			strings.TrimSpace(
				status,
			),
		)

	if status != "published" &&
		status != "hidden" {

		return AdminReview{},
			ErrInvalidAdminReviewStatus
	}

	queryCtx, cancel :=
		adminReadContext(ctx)
	defer cancel()

	var result AdminReview

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
							FROM reviews
							WHERE
								id = $1::uuid
								AND deleted_at IS NULL
							FOR UPDATE
						`,
						reviewID,
					).Scan(
						&previousStatus,
					)

				if errors.Is(
					err,
					pgx.ErrNoRows,
				) {
					return ErrAdminReviewNotFound
				}

				if err != nil {
					return fmt.Errorf(
						"lock Admin review: %w",
						err,
					)
				}

				if previousStatus !=
					status {

					_, err =
						tx.Exec(
							ctx,
							`
								UPDATE reviews
								SET
									status = $2,
									updated_at = now()
								WHERE id = $1::uuid
							`,
							reviewID,
							status,
						)
					if err != nil {
						return fmt.Errorf(
							"update review moderation status: %w",
							err,
						)
					}
				}

				if err :=
					insertAdminActionAuditTx(
						ctx,
						tx,
						metadata,
						adminEventReviewStatusChanged,
						map[string]any{
							"review_id": reviewID,

							"previous_status": previousStatus,

							"new_status": status,
						},
					); err != nil {

					return err
				}

				item, err :=
					getAdminReview(
						ctx,
						tx,
						reviewID,
					)
				if err != nil {
					return err
				}

				result =
					item

				return nil
			},
		)
	if err != nil {
		return AdminReview{},
			err
	}

	return result, nil
}

func getAdminReview(
	ctx context.Context,
	querier adminReadQuerier,
	reviewID string,
) (
	AdminReview,
	error,
) {
	var result AdminReview

	err :=
		querier.QueryRow(
			ctx,
			`
				SELECT
					r.id::text,
					r.customer_id::text,
					c.full_name,
					c.phone,
					o.id::text,
					o.order_number,
					r.order_item_id::text,
					v.product_id::text,
					oi.variant_id::text,
					oi.sku,
					oi.product_name,
					r.rating,
					COALESCE(r.title, ''),
					COALESCE(r.body, ''),
					r.status,
					r.created_at,
					r.updated_at
				FROM reviews r
				JOIN customers c
					ON c.id = r.customer_id
				JOIN order_items oi
					ON oi.id = r.order_item_id
				JOIN orders o
					ON o.id = oi.order_id
				JOIN product_variants v
					ON v.id = oi.variant_id
				WHERE
					r.id = $1::uuid
					AND r.deleted_at IS NULL
			`,
			reviewID,
		).Scan(
			&result.ID,
			&result.CustomerID,
			&result.CustomerName,
			&result.CustomerPhone,
			&result.OrderID,
			&result.OrderNumber,
			&result.OrderItemID,
			&result.ProductID,
			&result.VariantID,
			&result.SKU,
			&result.ProductName,
			&result.Rating,
			&result.Title,
			&result.Body,
			&result.Status,
			&result.CreatedAt,
			&result.UpdatedAt,
		)

	if errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return AdminReview{},
			ErrAdminReviewNotFound
	}

	if err != nil {
		return AdminReview{},
			fmt.Errorf(
				"get Admin review: %w",
				err,
			)
	}

	result.VerifiedPurchase =
		true

	return result, nil
}
