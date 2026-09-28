package catalog

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

const productListingCTE = `
	WITH RECURSIVE category_scope AS (
		SELECT
			c.id,
			ARRAY[c.id] AS path

		FROM categories c

		WHERE
			c.slug = $1
			AND c.is_active = true

		UNION ALL

		SELECT
			child.id,
			array_append(
				scope.path,
				child.id
			)

		FROM categories child

		JOIN category_scope scope
			ON child.parent_id =
				scope.id

		WHERE
			child.is_active = true
			AND NOT child.id =
				ANY(scope.path)
	),

	product_candidates AS (
		SELECT
			p.id AS product_id,
			p.id::text AS id,
			p.product_code,
			p.name,
			p.slug,
			p.brand,
			p.short_description,
			p.is_featured,
			p.published_at,
			p.created_at,

			price_variant.price_amount,
			price_variant.currency,

			EXISTS (
				SELECT 1

				FROM product_variants stock_variant

				JOIN inventory i
					ON i.variant_id =
						stock_variant.id

				WHERE
					stock_variant.product_id =
						p.id

					AND stock_variant.is_active =
						true

					AND (
						i.quantity_on_hand
						- i.quantity_reserved
					) > 0
			) AS in_stock

		FROM products p

		JOIN categories product_category
			ON product_category.id =
				p.category_id
			AND product_category.is_active =
				true

		JOIN LATERAL (
			SELECT
				v.price_amount,
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

		WHERE
			p.status = 'active'

			AND (
				$1 = ''

				OR p.category_id IN (
					SELECT id
					FROM category_scope
				)
			)
	),

	filtered AS (
		SELECT *

		FROM product_candidates

		WHERE
			(
				$2::bigint IS NULL
				OR price_amount >= $2
			)

			AND (
				$3::bigint IS NULL
				OR price_amount <= $3
			)

			AND (
				$4::boolean IS NULL
				OR in_stock = $4
			)
	)
`

func (r *Repository) ListActiveProductsPage(
	ctx context.Context,
	input ListProductsQuery,
) (ProductListPage, error) {
	orderBy, err :=
		productListOrderBy(
			input.Sort,
		)
	if err != nil {
		return ProductListPage{},
			err
	}

	countQuery :=
		productListingCTE +
			`
				SELECT count(*)
				FROM filtered
			`

	listQuery :=
		productListingCTE +
			`
				SELECT
					f.id,
					f.product_code,
					f.name,
					f.slug,
					f.brand,
					f.short_description,
					f.is_featured,

					primary_image.url,

					f.price_amount,
					f.currency,
					f.in_stock

				FROM filtered f

				LEFT JOIN LATERAL (
					SELECT
						pi.url

					FROM product_images pi

					WHERE
						pi.product_id =
							f.product_id

						AND pi.is_primary =
							true

					LIMIT 1
				) AS primary_image
					ON true

				CROSS JOIN LATERAL (
					SELECT
						$5::text[]
							AS ranked_product_ids
				) AS ranking_input
			` +
			orderBy +
			`
				LIMIT $6
				OFFSET $7
			`

	minPrice :=
		nullableInt64(
			input.MinPrice,
		)

	maxPrice :=
		nullableInt64(
			input.MaxPrice,
		)

	inStock :=
		nullableBool(
			input.InStock,
		)

	rankedProductIDs :=
		input.RankedProductIDs

	if rankedProductIDs == nil {
		rankedProductIDs =
			make(
				[]string,
				0,
			)
	}

	batch :=
		&pgx.Batch{}

	batch.Queue(
		countQuery,
		input.CategorySlug,
		minPrice,
		maxPrice,
		inStock,
	)

	batch.Queue(
		listQuery,
		input.CategorySlug,
		minPrice,
		maxPrice,
		inStock,
		rankedProductIDs,
		input.Pagination.Limit,
		input.Pagination.Offset(),
	)

	results :=
		r.db.SendBatch(
			ctx,
			batch,
		)

	defer results.Close()

	var total int64

	if err :=
		results.QueryRow().Scan(
			&total,
		); err != nil {

		return ProductListPage{},
			fmt.Errorf(
				"count active products: %w",
				err,
			)
	}

	rows, err :=
		results.Query()
	if err != nil {
		return ProductListPage{},
			fmt.Errorf(
				"query active product page: %w",
				err,
			)
	}

	defer rows.Close()

	products :=
		make(
			[]ProductCard,
			0,
			input.Pagination.Limit,
		)

	for rows.Next() {
		var item ProductCard

		if err :=
			rows.Scan(
				&item.ID,
				&item.ProductCode,
				&item.Name,
				&item.Slug,
				&item.Brand,
				&item.ShortDescription,
				&item.IsFeatured,
				&item.PrimaryImageURL,
				&item.PriceAmount,
				&item.Currency,
				&item.InStock,
			); err != nil {

			return ProductListPage{},
				fmt.Errorf(
					"scan active product page: %w",
					err,
				)
		}

		products =
			append(
				products,
				item,
			)
	}

	if err :=
		rows.Err(); err != nil {

		return ProductListPage{},
			fmt.Errorf(
				"iterate active product page: %w",
				err,
			)
	}

	return ProductListPage{
		Items: products,
		Total: total,
	}, nil
}

func productListOrderBy(
	sort string,
) (string, error) {
	switch sort {

	case ProductSortRecommended:

		return `
			ORDER BY
				CASE
					WHEN array_position(
						ranking_input.ranked_product_ids,
						f.id
					) IS NULL
					THEN 1
					ELSE 0
				END ASC,

				array_position(
					ranking_input.ranked_product_ids,
					f.id
				) ASC NULLS LAST,

				f.is_featured DESC,
				f.published_at DESC NULLS LAST,
				f.created_at DESC,
				f.id ASC
		`, nil

	case ProductSortFeatured:

		return `
			ORDER BY
				f.is_featured DESC,
				f.published_at DESC NULLS LAST,
				f.created_at DESC,
				f.id ASC
		`, nil

	case ProductSortNewest:

		return `
			ORDER BY
				f.published_at DESC NULLS LAST,
				f.created_at DESC,
				f.id ASC
		`, nil

	case ProductSortPriceAsc:

		return `
			ORDER BY
				f.price_amount ASC,
				f.id ASC
		`, nil

	case ProductSortPriceDesc:

		return `
			ORDER BY
				f.price_amount DESC,
				f.id ASC
		`, nil

	case ProductSortNameAsc:

		return `
			ORDER BY
				lower(f.name) ASC,
				f.id ASC
		`, nil

	case ProductSortNameDesc:

		return `
			ORDER BY
				lower(f.name) DESC,
				f.id ASC
		`, nil

	default:

		return "",
			ErrInvalidProductSort
	}
}

func nullableInt64(
	value *int64,
) any {
	if value == nil {
		return nil
	}

	return *value
}

func nullableBool(
	value *bool,
) any {
	if value == nil {
		return nil
	}

	return *value
}
