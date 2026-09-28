package wishlist

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"project.local/commerce-api/internal/platform/pagination"
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

const wishlistItemSelect = `
	SELECT
		w.id::text,
		w.product_id::text,
		w.created_at,

		p.product_code,
		p.name,
		p.slug,
		p.brand,

		primary_image.url,

		price_variant.price_amount,
		price_variant.compare_at_price_amount,
		price_variant.currency,

		EXISTS (
			SELECT 1
			FROM product_variants stock_variant
			JOIN inventory i
				ON i.variant_id = stock_variant.id
			WHERE
				stock_variant.product_id = p.id
				AND stock_variant.is_active = true
				AND (
					i.quantity_on_hand
					- i.quantity_reserved
				) > 0
		) AS in_stock,

		(
			p.status = 'active'
			AND price_variant.id IS NOT NULL
		) AS is_available

	FROM customer_wishlist_items w

	JOIN products p
		ON p.id = w.product_id

	LEFT JOIN LATERAL (
		SELECT
			v.id,
			v.price_amount,
			v.compare_at_price_amount,
			v.currency
		FROM product_variants v
		WHERE
			v.product_id = p.id
			AND v.is_active = true
		ORDER BY
			v.price_amount ASC,
			v.id ASC
		LIMIT 1
	) AS price_variant
		ON true

	LEFT JOIN LATERAL (
		SELECT pi.url
		FROM product_images pi
		WHERE pi.product_id = p.id
		ORDER BY
			pi.is_primary DESC,
			pi.sort_order ASC,
			pi.id ASC
		LIMIT 1
	) AS primary_image
		ON true
`

type rowScanner interface {
	Scan(dest ...any) error
}

func scanItem(
	row rowScanner,
) (Item, error) {
	var result Item

	var price pgtype.Int8
	var compareAt pgtype.Int8
	var currency pgtype.Text

	if err := row.Scan(
		&result.ID,
		&result.ProductID,
		&result.AddedAt,
		&result.ProductCode,
		&result.Name,
		&result.Slug,
		&result.Brand,
		&result.PrimaryImageURL,
		&price,
		&compareAt,
		&currency,
		&result.InStock,
		&result.IsAvailable,
	); err != nil {
		return Item{}, err
	}

	if price.Valid {
		value := price.Int64

		result.PriceAmount =
			&value
	}

	if compareAt.Valid {
		value := compareAt.Int64

		result.CompareAtPriceAmount =
			&value
	}

	if currency.Valid {
		result.Currency =
			currency.String
	}

	return result, nil
}

func (r *Repository) Add(
	ctx context.Context,
	customerID string,
	productID string,
) (Item, error) {
	const insertQuery = `
		INSERT INTO customer_wishlist_items (
			customer_id,
			product_id
		)
		SELECT
			$1::uuid,
			p.id
		FROM products p
		WHERE
			p.id = $2::uuid
			AND p.status = 'active'
		ON CONFLICT (customer_id, product_id)
		DO UPDATE SET
			customer_id = EXCLUDED.customer_id
		RETURNING id::text
	`

	var wishlistID string

	if err := r.db.QueryRow(
		ctx,
		insertQuery,
		customerID,
		productID,
	).Scan(
		&wishlistID,
	); err != nil {

		if errors.Is(
			err,
			pgx.ErrNoRows,
		) {
			return Item{},
				ErrProductNotFound
		}

		return Item{},
			fmt.Errorf(
				"add wishlist item: %w",
				err,
			)
	}

	result, err := r.Get(
		ctx,
		customerID,
		productID,
	)
	if err != nil {
		return Item{}, err
	}

	return result, nil
}

func (r *Repository) Get(
	ctx context.Context,
	customerID string,
	productID string,
) (Item, error) {
	result, err := scanItem(
		r.db.QueryRow(
			ctx,
			wishlistItemSelect+`
				WHERE
					w.customer_id = $1::uuid
					AND w.product_id = $2::uuid
			`,
			customerID,
			productID,
		),
	)

	if errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return Item{},
			ErrProductNotFound
	}

	if err != nil {
		return Item{},
			fmt.Errorf(
				"get wishlist item: %w",
				err,
			)
	}

	return result, nil
}

func (r *Repository) Remove(
	ctx context.Context,
	customerID string,
	productID string,
) error {
	_, err := r.db.Exec(
		ctx,
		`
			DELETE FROM customer_wishlist_items
			WHERE
				customer_id = $1::uuid
				AND product_id = $2::uuid
		`,
		customerID,
		productID,
	)
	if err != nil {
		return fmt.Errorf(
			"remove wishlist item: %w",
			err,
		)
	}

	return nil
}

func (r *Repository) Contains(
	ctx context.Context,
	customerID string,
	productID string,
) (bool, error) {
	var exists bool

	if err := r.db.QueryRow(
		ctx,
		`
			SELECT EXISTS (
				SELECT 1
				FROM customer_wishlist_items
				WHERE
					customer_id = $1::uuid
					AND product_id = $2::uuid
			)
		`,
		customerID,
		productID,
	).Scan(
		&exists,
	); err != nil {

		return false,
			fmt.Errorf(
				"check wishlist item: %w",
				err,
			)
	}

	return exists, nil
}

func (r *Repository) Count(
	ctx context.Context,
	customerID string,
) (int64, error) {
	var total int64

	if err := r.db.QueryRow(
		ctx,
		`
			SELECT count(*)
			FROM customer_wishlist_items
			WHERE customer_id = $1::uuid
		`,
		customerID,
	).Scan(
		&total,
	); err != nil {

		return 0,
			fmt.Errorf(
				"count wishlist items: %w",
				err,
			)
	}

	return total, nil
}

func (r *Repository) List(
	ctx context.Context,
	customerID string,
	params pagination.Params,
) (ListResult, error) {
	const countQuery = `
		SELECT count(*)
		FROM customer_wishlist_items
		WHERE customer_id = $1::uuid
	`

	listQuery :=
		wishlistItemSelect +
			`
				WHERE w.customer_id = $1::uuid
				ORDER BY
					w.created_at DESC,
					w.id DESC
				LIMIT $2
				OFFSET $3
			`

	batch :=
		&pgx.Batch{}

	batch.Queue(
		countQuery,
		customerID,
	)

	batch.Queue(
		listQuery,
		customerID,
		params.Limit,
		params.Offset(),
	)

	results :=
		r.db.SendBatch(
			ctx,
			batch,
		)

	defer results.Close()

	var total int64

	if err := results.QueryRow().Scan(
		&total,
	); err != nil {

		return ListResult{},
			fmt.Errorf(
				"count wishlist page: %w",
				err,
			)
	}

	rows, err :=
		results.Query()
	if err != nil {
		return ListResult{},
			fmt.Errorf(
				"query wishlist page: %w",
				err,
			)
	}

	defer rows.Close()

	items :=
		make(
			[]Item,
			0,
			params.Limit,
		)

	for rows.Next() {
		item, err :=
			scanItem(
				rows,
			)
		if err != nil {
			return ListResult{},
				fmt.Errorf(
					"scan wishlist item: %w",
					err,
				)
		}

		items =
			append(
				items,
				item,
			)
	}

	if err :=
		rows.Err(); err != nil {

		return ListResult{},
			fmt.Errorf(
				"iterate wishlist items: %w",
				err,
			)
	}

	return ListResult{
		Items: items,

		Meta: pagination.NewMeta(
			params,
			total,
		),
	}, nil
}
