package catalog

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"project.local/commerce-api/internal/platform/pagination"
)

const HomeFeedSort = "homepage"

type HomeProductCard struct {
	ID               string  `json:"id"`
	ProductCode      string  `json:"product_code"`
	Name             string  `json:"name"`
	Slug             string  `json:"slug"`
	Brand            *string `json:"brand,omitempty"`
	ShortDescription *string `json:"short_description,omitempty"`
	IsFeatured       bool    `json:"is_featured"`

	PrimaryImageURL *string `json:"primary_image_url,omitempty"`

	PriceAmount          int64  `json:"price_amount"`
	CompareAtPriceAmount *int64 `json:"compare_at_price_amount,omitempty"`
	Currency             string `json:"currency"`

	InStock bool `json:"in_stock"`

	PublishedAt *time.Time `json:"published_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`

	SoldQuantity       int64  `json:"sold_quantity"`
	MerchandisingGroup string `json:"merchandising_group"`
}

type HomeProductListPage struct {
	Items []HomeProductCard
	Total int64
}

type HomeProductListMeta struct {
	pagination.Meta

	Sort string `json:"sort"`
}

type HomeProductListResult struct {
	Items []HomeProductCard
	Meta  HomeProductListMeta
}

type homeFeedStore interface {
	ListHomeProductsPage(
		ctx context.Context,
		params pagination.Params,
	) (HomeProductListPage, error)
}

func (s *Service) ListHomeProducts(
	ctx context.Context,
	page int,
	limit int,
) (HomeProductListResult, error) {
	params, err := pagination.New(
		page,
		limit,
	)
	if err != nil {
		return HomeProductListResult{}, err
	}

	store, ok := s.repository.(homeFeedStore)
	if !ok {
		return HomeProductListResult{},
			fmt.Errorf(
				"homepage catalog repository is unavailable",
			)
	}

	result, err := store.ListHomeProductsPage(
		ctx,
		params,
	)
	if err != nil {
		return HomeProductListResult{},
			fmt.Errorf(
				"list homepage product page: %w",
				err,
			)
	}

	if result.Items == nil {
		result.Items = make(
			[]HomeProductCard,
			0,
		)
	}

	return HomeProductListResult{
		Items: result.Items,
		Meta: HomeProductListMeta{
			Meta: pagination.NewMeta(
				params,
				result.Total,
			),
			Sort: HomeFeedSort,
		},
	}, nil
}

const homeProductCandidatesQuery = `
	WITH candidates AS (
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

			COALESCE(
				sales.sold_quantity,
				0
			)::bigint AS sold_quantity

		FROM products p

		JOIN categories product_category
			ON product_category.id = p.category_id
			AND product_category.is_active = true

		JOIN LATERAL (
			SELECT
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

		LEFT JOIN (
			SELECT
				sold_variant.product_id,
				SUM(oi.quantity)::bigint AS sold_quantity
			FROM product_variants sold_variant
			JOIN order_items oi
				ON oi.variant_id = sold_variant.id
			JOIN orders o
				ON o.id = oi.order_id
			WHERE
				o.payment_status IN (
					'paid',
					'cod_collected'
				)
				AND o.status NOT IN (
					'cancelled',
					'payment_expired'
				)
			GROUP BY sold_variant.product_id
		) AS sales
			ON sales.product_id = p.id

		WHERE p.status = 'active'
	),

	bucketed AS (
		SELECT
			candidates.*,

			CASE
				WHEN
					published_at IS NOT NULL
					AND published_at >= now() - interval '30 days'
					AND NOT (
						compare_at_price_amount IS NOT NULL
						AND compare_at_price_amount > price_amount
					)
					THEN 'new'

				WHEN
					compare_at_price_amount IS NOT NULL
					AND compare_at_price_amount > price_amount
					THEN 'discount'

				WHEN sold_quantity > 0
					THEN 'popular'

				ELSE 'catalog'
			END AS merchandising_group,

			CASE
				WHEN
					published_at IS NOT NULL
					AND published_at >= now() - interval '30 days'
					AND NOT (
						compare_at_price_amount IS NOT NULL
						AND compare_at_price_amount > price_amount
					)
					THEN 0

				WHEN
					compare_at_price_amount IS NOT NULL
					AND compare_at_price_amount > price_amount
					THEN 1

				WHEN sold_quantity > 0
					THEN 2

				ELSE 3
			END AS merchandising_rank,

			CASE
				WHEN
					compare_at_price_amount IS NOT NULL
					AND compare_at_price_amount > price_amount
					AND compare_at_price_amount > 0
				THEN (
					(compare_at_price_amount - price_amount)
					* 10000
					/ compare_at_price_amount
				)::bigint
				ELSE 0::bigint
			END AS discount_bps

		FROM candidates
	)
`

func (r *Repository) ListHomeProductsPage(
	ctx context.Context,
	params pagination.Params,
) (HomeProductListPage, error) {
	const countQuery = `
		SELECT count(*)
		FROM products p
		JOIN categories c
			ON c.id = p.category_id
			AND c.is_active = true
		WHERE
			p.status = 'active'
			AND EXISTS (
				SELECT 1
				FROM product_variants v
				WHERE
					v.product_id = p.id
					AND v.is_active = true
			)
	`

	listQuery := homeProductCandidatesQuery + `
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
			f.compare_at_price_amount,
			f.currency,
			f.in_stock,
			f.published_at,
			f.created_at,
			f.sold_quantity,
			f.merchandising_group

		FROM bucketed f

		LEFT JOIN LATERAL (
			SELECT pi.url
			FROM product_images pi
			WHERE
				pi.product_id = f.product_id
				AND pi.is_primary = true
			ORDER BY
				pi.sort_order ASC,
				pi.id ASC
			LIMIT 1
		) AS primary_image
			ON true

		ORDER BY
			f.merchandising_rank ASC,

			CASE
				WHEN f.merchandising_rank = 0
					THEN f.published_at
			END DESC NULLS LAST,

			CASE
				WHEN f.merchandising_rank = 1
					THEN f.discount_bps
			END DESC NULLS LAST,

			CASE
				WHEN f.merchandising_rank = 2
					THEN f.sold_quantity
			END DESC NULLS LAST,

			f.is_featured DESC,
			f.published_at DESC NULLS LAST,
			f.created_at DESC,
			f.id ASC

		LIMIT $1
		OFFSET $2
	`

	batch := &pgx.Batch{}

	batch.Queue(countQuery)

	batch.Queue(
		listQuery,
		params.Limit,
		params.Offset(),
	)

	results := r.db.SendBatch(
		ctx,
		batch,
	)
	defer results.Close()

	var total int64

	if err := results.QueryRow().Scan(
		&total,
	); err != nil {
		return HomeProductListPage{},
			fmt.Errorf(
				"count homepage products: %w",
				err,
			)
	}

	rows, err := results.Query()
	if err != nil {
		return HomeProductListPage{},
			fmt.Errorf(
				"query homepage products: %w",
				err,
			)
	}
	defer rows.Close()

	items := make(
		[]HomeProductCard,
		0,
		params.Limit,
	)

	for rows.Next() {
		var item HomeProductCard
		var compareAt pgtype.Int8
		var publishedAt pgtype.Timestamptz

		if err := rows.Scan(
			&item.ID,
			&item.ProductCode,
			&item.Name,
			&item.Slug,
			&item.Brand,
			&item.ShortDescription,
			&item.IsFeatured,
			&item.PrimaryImageURL,
			&item.PriceAmount,
			&compareAt,
			&item.Currency,
			&item.InStock,
			&publishedAt,
			&item.CreatedAt,
			&item.SoldQuantity,
			&item.MerchandisingGroup,
		); err != nil {
			return HomeProductListPage{},
				fmt.Errorf(
					"scan homepage product: %w",
					err,
				)
		}

		if compareAt.Valid {
			value := compareAt.Int64
			item.CompareAtPriceAmount = &value
		}

		if publishedAt.Valid {
			value := publishedAt.Time
			item.PublishedAt = &value
		}

		items = append(
			items,
			item,
		)
	}

	if err := rows.Err(); err != nil {
		return HomeProductListPage{},
			fmt.Errorf(
				"iterate homepage products: %w",
				err,
			)
	}

	return HomeProductListPage{
		Items: items,
		Total: total,
	}, nil
}
