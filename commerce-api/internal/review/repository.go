package review

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(
	db *pgxpool.Pool,
) *Repository {
	return &Repository{
		db: db,
	}
}

type PurchaseContext struct {
	OrderItemID string
	OrderID     string
	ProductID   string
	VariantID   string

	SKU         string
	ProductName string
	OrderStatus string
}

func (r *Repository) GetPurchaseContext(
	ctx context.Context,
	customerID string,
	orderItemID string,
) (PurchaseContext, error) {
	const query = `
		SELECT
			oi.id::text,
			oi.order_id::text,
			v.product_id::text,
			oi.variant_id::text,
			oi.sku,
			oi.product_name,
			o.status
		FROM order_items oi
		JOIN orders o
			ON o.id = oi.order_id
		JOIN product_variants v
			ON v.id = oi.variant_id
		WHERE
			oi.id = $1::uuid
			AND o.customer_id = $2::uuid
	`

	var result PurchaseContext

	err := r.db.QueryRow(
		ctx,
		query,
		orderItemID,
		customerID,
	).Scan(
		&result.OrderItemID,
		&result.OrderID,
		&result.ProductID,
		&result.VariantID,
		&result.SKU,
		&result.ProductName,
		&result.OrderStatus,
	)
	if errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return PurchaseContext{},
			ErrOrderItemNotFound
	}

	if err != nil {
		return PurchaseContext{},
			fmt.Errorf(
				"load review purchase context: %w",
				err,
			)
	}

	return result, nil
}

func (r *Repository) Create(
	ctx context.Context,
	customerID string,
	purchase PurchaseContext,
	rating int,
	title *string,
	body *string,
) (Review, error) {
	var reviewID string

	err := r.db.QueryRow(
		ctx,
		`
			INSERT INTO reviews (
				customer_id,
				order_item_id,
				rating,
				title,
				body,
				status,
				created_at,
				updated_at
			)
			VALUES (
				$1::uuid,
				$2::uuid,
				$3,
				$4,
				$5,
				'published',
				now(),
				now()
			)
			RETURNING id::text
		`,
		customerID,
		purchase.OrderItemID,
		rating,
		nullableStringArgument(title),
		nullableStringArgument(body),
	).Scan(
		&reviewID,
	)
	if err != nil {
		if reviewUniqueConflict(err) {
			return Review{},
				ErrAlreadyExists
		}

		return Review{},
			fmt.Errorf(
				"create review: %w",
				err,
			)
	}

	return r.GetCustomerReview(
		ctx,
		customerID,
		reviewID,
	)
}

type updateInput struct {
	Rating *int

	TitleSet bool
	Title    *string

	BodySet bool
	Body    *string
}

func (r *Repository) Update(
	ctx context.Context,
	customerID string,
	reviewID string,
	input updateInput,
) (Review, error) {
	tag, err := r.db.Exec(
		ctx,
		`
			UPDATE reviews
			SET
				rating = COALESCE(
					$3::integer,
					rating
				),
				title = CASE
					WHEN $4::boolean
					THEN $5::varchar
					ELSE title
				END,
				body = CASE
					WHEN $6::boolean
					THEN $7::text
					ELSE body
				END,
				updated_at = now()
			WHERE
				id = $1::uuid
				AND customer_id = $2::uuid
				AND deleted_at IS NULL
		`,
		reviewID,
		customerID,
		input.Rating,
		input.TitleSet,
		nullableStringArgument(
			input.Title,
		),
		input.BodySet,
		nullableStringArgument(
			input.Body,
		),
	)
	if err != nil {
		return Review{},
			fmt.Errorf(
				"update review: %w",
				err,
			)
	}

	if tag.RowsAffected() != 1 {
		return Review{},
			ErrReviewNotFound
	}

	return r.GetCustomerReview(
		ctx,
		customerID,
		reviewID,
	)
}

func (r *Repository) Delete(
	ctx context.Context,
	customerID string,
	reviewID string,
) error {
	tag, err := r.db.Exec(
		ctx,
		`
			UPDATE reviews
			SET
				deleted_at = now(),
				updated_at = now()
			WHERE
				id = $1::uuid
				AND customer_id = $2::uuid
				AND deleted_at IS NULL
		`,
		reviewID,
		customerID,
	)
	if err != nil {
		return fmt.Errorf(
			"delete review: %w",
			err,
		)
	}

	if tag.RowsAffected() != 1 {
		return ErrReviewNotFound
	}

	return nil
}

const customerReviewSelect = `
	SELECT
		r.id::text,
		r.order_item_id::text,
		v.product_id::text,
		oi.variant_id::text,
		oi.sku,
		oi.product_name,
		r.rating,
		r.title,
		r.body,
		r.status,
		r.created_at,
		r.updated_at
	FROM reviews r
	JOIN order_items oi
		ON oi.id = r.order_item_id
	JOIN product_variants v
		ON v.id = oi.variant_id
`

type reviewScanner interface {
	Scan(
		dest ...any,
	) error
}

func scanCustomerReview(
	row reviewScanner,
) (Review, error) {
	var result Review

	var title pgtype.Text
	var body pgtype.Text

	err := row.Scan(
		&result.ID,
		&result.OrderItemID,
		&result.ProductID,
		&result.VariantID,
		&result.SKU,
		&result.ProductName,
		&result.Rating,
		&title,
		&body,
		&result.Status,
		&result.CreatedAt,
		&result.UpdatedAt,
	)
	if err != nil {
		return Review{}, err
	}

	result.Title =
		textPointer(
			title,
		)

	result.Body =
		textPointer(
			body,
		)

	result.VerifiedPurchase = true

	return result, nil
}

func (r *Repository) GetCustomerReview(
	ctx context.Context,
	customerID string,
	reviewID string,
) (Review, error) {
	result, err :=
		scanCustomerReview(
			r.db.QueryRow(
				ctx,
				customerReviewSelect+`
					WHERE
						r.id = $1::uuid
						AND r.customer_id = $2::uuid
						AND r.deleted_at IS NULL
				`,
				reviewID,
				customerID,
			),
		)

	if errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return Review{},
			ErrReviewNotFound
	}

	if err != nil {
		return Review{},
			fmt.Errorf(
				"get customer review: %w",
				err,
			)
	}

	return result, nil
}

func (r *Repository) ListCustomerReviews(
	ctx context.Context,
	customerID string,
	limit int,
	offset int,
) ([]Review, error) {
	rows, err := r.db.Query(
		ctx,
		customerReviewSelect+`
			WHERE
				r.customer_id = $1::uuid
				AND r.deleted_at IS NULL
			ORDER BY
				r.created_at DESC,
				r.id DESC
			LIMIT $2
			OFFSET $3
		`,
		customerID,
		limit,
		offset,
	)
	if err != nil {
		return nil,
			fmt.Errorf(
				"list customer reviews: %w",
				err,
			)
	}

	defer rows.Close()

	result := make(
		[]Review,
		0,
	)

	for rows.Next() {
		item, err :=
			scanCustomerReview(
				rows,
			)
		if err != nil {
			return nil,
				fmt.Errorf(
					"scan customer review: %w",
					err,
				)
		}

		result = append(
			result,
			item,
		)
	}

	if err := rows.Err(); err != nil {
		return nil,
			fmt.Errorf(
				"iterate customer reviews: %w",
				err,
			)
	}

	return result, nil
}

func (r *Repository) ProductExists(
	ctx context.Context,
	productID string,
) (bool, error) {
	var exists bool

	err := r.db.QueryRow(
		ctx,
		`
			SELECT EXISTS (
				SELECT 1
				FROM products
				WHERE
					id = $1::uuid
					AND status = 'active'
			)
		`,
		productID,
	).Scan(
		&exists,
	)
	if err != nil {
		return false,
			fmt.Errorf(
				"check review product: %w",
				err,
			)
	}

	return exists, nil
}

func (r *Repository) ListProductReviews(
	ctx context.Context,
	productID string,
	limit int,
	offset int,
) ([]PublicReview, error) {
	rows, err := r.db.Query(
		ctx,
		`
			SELECT
				r.id::text,
				oi.variant_id::text,
				oi.sku,
				r.rating,
				r.title,
				r.body,
				r.created_at,
				r.updated_at
			FROM reviews r
			JOIN order_items oi
				ON oi.id = r.order_item_id
			JOIN product_variants v
				ON v.id = oi.variant_id
			WHERE
				v.product_id = $1::uuid
				AND r.status = 'published'
				AND r.deleted_at IS NULL
			ORDER BY
				r.created_at DESC,
				r.id DESC
			LIMIT $2
			OFFSET $3
		`,
		productID,
		limit,
		offset,
	)
	if err != nil {
		return nil,
			fmt.Errorf(
				"list product reviews: %w",
				err,
			)
	}

	defer rows.Close()

	result := make(
		[]PublicReview,
		0,
	)

	for rows.Next() {
		var item PublicReview

		var title pgtype.Text
		var body pgtype.Text

		if err := rows.Scan(
			&item.ID,
			&item.VariantID,
			&item.SKU,
			&item.Rating,
			&title,
			&body,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil,
				fmt.Errorf(
					"scan product review: %w",
					err,
				)
		}

		item.Title =
			textPointer(
				title,
			)

		item.Body =
			textPointer(
				body,
			)

		item.ReviewerName =
			"Verified Buyer"

		item.VerifiedPurchase =
			true

		result = append(
			result,
			item,
		)
	}

	if err := rows.Err(); err != nil {
		return nil,
			fmt.Errorf(
				"iterate product reviews: %w",
				err,
			)
	}

	return result, nil
}

func (r *Repository) ProductSummary(
	ctx context.Context,
	productID string,
) (Summary, error) {
	var result Summary

	result.ProductID =
		productID

	err := r.db.QueryRow(
		ctx,
		`
			SELECT
				COUNT(*)::bigint,
				COALESCE(
					AVG(r.rating)::float8,
					0
				),
				COUNT(*) FILTER (
					WHERE r.rating = 1
				)::bigint,
				COUNT(*) FILTER (
					WHERE r.rating = 2
				)::bigint,
				COUNT(*) FILTER (
					WHERE r.rating = 3
				)::bigint,
				COUNT(*) FILTER (
					WHERE r.rating = 4
				)::bigint,
				COUNT(*) FILTER (
					WHERE r.rating = 5
				)::bigint
			FROM reviews r
			JOIN order_items oi
				ON oi.id = r.order_item_id
			JOIN product_variants v
				ON v.id = oi.variant_id
			WHERE
				v.product_id = $1::uuid
				AND r.status = 'published'
				AND r.deleted_at IS NULL
		`,
		productID,
	).Scan(
		&result.TotalReviews,
		&result.AverageRating,
		&result.Breakdown.One,
		&result.Breakdown.Two,
		&result.Breakdown.Three,
		&result.Breakdown.Four,
		&result.Breakdown.Five,
	)
	if err != nil {
		return Summary{},
			fmt.Errorf(
				"load product review summary: %w",
				err,
			)
	}

	result.VerifiedPurchaseCount =
		result.TotalReviews

	return result, nil
}

func nullableStringArgument(
	value *string,
) any {
	if value == nil {
		return nil
	}

	return *value
}

func textPointer(
	value pgtype.Text,
) *string {
	if !value.Valid {
		return nil
	}

	result := value.String

	return &result
}

func reviewUniqueConflict(
	err error,
) bool {
	var pgErr *pgconn.PgError

	if !errors.As(
		err,
		&pgErr,
	) {
		return false
	}

	return pgErr.Code == "23505" &&
		pgErr.ConstraintName ==
			"reviews_order_item_active_key"
}
