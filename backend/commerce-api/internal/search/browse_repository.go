package search

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

const browseSearchBase = `
	WITH RECURSIVE
	input AS (
		SELECT lower(
			regexp_replace(
				trim($1::text),
				'\s+',
				' ',
				'g'
			)
		) AS q
	),

	category_scope(id, path) AS (
		SELECT
			c.id,
			ARRAY[c.id]

		FROM categories c

		WHERE
			$2 <> ''
			AND c.slug = $2
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

	product_base AS (
		SELECT
			p.id,
			p.product_code,
			p.name,
			p.slug,
			p.brand,
			p.short_description,
			p.is_featured,
			p.published_at,
			p.created_at,
			p.category_id,

			c.name AS category_name,

			c.slug AS category_slug,

			primary_image.url
				AS primary_image_url,

			price.price_amount,

			price.currency,

			COALESCE(
				variant_text.value,
				''
			) AS variant_text,

			matched_variant.sku
				AS matched_sku,

			EXISTS (
				SELECT 1

				FROM product_variants stock_variant

				LEFT JOIN inventory stock_inventory
					ON stock_inventory.variant_id =
						stock_variant.id

				WHERE
					stock_variant.product_id =
						p.id

					AND stock_variant.is_active =
						true

					AND (
						COALESCE(
							stock_inventory.quantity_on_hand,
							0
						)
						-
						COALESCE(
							stock_inventory.quantity_reserved,
							0
						)
					) > 0
			) AS in_stock

		FROM products p

		JOIN categories c
			ON c.id =
				p.category_id
			AND c.is_active =
				true

		CROSS JOIN input i

		JOIN LATERAL (
			SELECT
				v.price_amount,
				v.currency

			FROM product_variants v

			WHERE
				v.product_id =
					p.id

				AND v.is_active =
					true

			ORDER BY
				v.price_amount ASC,
				v.id ASC

			LIMIT 1
		) price
			ON true

		LEFT JOIN LATERAL (
			SELECT
				pi.url

			FROM product_images pi

			WHERE
				pi.product_id =
					p.id

				AND pi.is_primary =
					true

			ORDER BY
				pi.sort_order ASC,
				pi.id ASC

			LIMIT 1
		) primary_image
			ON true

		LEFT JOIN LATERAL (
			SELECT
				string_agg(
					trim(
						concat_ws(
							' ',
							NULLIF(
								v.sku,
								''
							),
							NULLIF(
								v.color_name,
								''
							),
							NULLIF(
								v.size,
								''
							)
						)
					),
					' '
				) AS value

			FROM product_variants v

			WHERE
				v.product_id =
					p.id

				AND v.is_active =
					true
		) variant_text
			ON true

		LEFT JOIN LATERAL (
			SELECT
				v.sku

			FROM product_variants v

			WHERE
				v.product_id =
					p.id

				AND v.is_active =
					true

				AND (
					lower(v.sku) =
						i.q

					OR similarity(
						lower(v.sku),
						i.q
					) >= 0.60
				)

			ORDER BY
				(
					lower(v.sku) =
						i.q
				) DESC,

				similarity(
					lower(v.sku),
					i.q
				) DESC,

				v.sku ASC

			LIMIT 1
		) matched_variant
			ON true

		WHERE
			p.status = 'active'

			AND (
				$2 = ''

				OR p.category_id IN (
					SELECT id
					FROM category_scope
				)
			)
	),

	scored AS (
		SELECT
			pb.*,

			CASE
				WHEN lower(
					pb.product_code
				) = i.q

					OR lower(
						COALESCE(
							pb.matched_sku,
							''
						)
					) = i.q

				THEN 1000::real

				WHEN lower(
					pb.name
				) = i.q

				THEN 950::real

				WHEN to_tsvector(
					'simple',
					concat_ws(
						' ',
						pb.name,
						pb.brand,
						pb.short_description,
						pb.category_name,
						pb.variant_text
					)
				) @@ plainto_tsquery(
					'simple',
					i.q
				)

				THEN
					800::real
					+
					ts_rank_cd(
						to_tsvector(
							'simple',
							concat_ws(
								' ',
								pb.name,
								pb.brand,
								pb.short_description,
								pb.category_name,
								pb.variant_text
							)
						),
						plainto_tsquery(
							'simple',
							i.q
						)
					) * 100

				WHEN similarity(
					lower(pb.name),
					i.q
				) >= 0.30

				THEN
					650::real
					+
					similarity(
						lower(pb.name),
						i.q
					) * 100

				WHEN pb.brand IS NOT NULL

					AND similarity(
						lower(pb.brand),
						i.q
					) >= 0.40

				THEN
					575::real
					+
					similarity(
						lower(pb.brand),
						i.q
					) * 100

				WHEN lower(
					pb.category_name
				) = i.q

					OR lower(
						replace(
							pb.category_slug,
							'-',
							' '
						)
					) = i.q

				THEN 550::real

				WHEN similarity(
					lower(
						pb.category_name
					),
					i.q
				) >= 0.35

				THEN
					500::real
					+
					similarity(
						lower(
							pb.category_name
						),
						i.q
					) * 100

				ELSE 0::real
			END AS search_score

		FROM product_base pb

		CROSS JOIN input i
	),

	filtered AS (
		SELECT *

		FROM scored

		WHERE
			search_score > 0

			AND (
				$3 = ''

				OR lower(
					trim(
						COALESCE(
							brand,
							''
						)
					)
				) =
					lower(
						trim(
							$3
						)
					)
			)

			AND (
				$4::bigint IS NULL

				OR price_amount >=
					$4
			)

			AND (
				$5::bigint IS NULL

				OR price_amount <=
					$5
			)

			AND (
				$6::boolean IS NULL

				OR in_stock =
					$6
			)
	)
`

func (r *Repository) SearchBrowse(
	ctx context.Context,
	normalizedQuery string,
	page int,
	limit int,
	filters Filters,
) (BrowsePage, error) {
	offset :=
		(page - 1) *
			limit

	orderBy, err :=
		browseOrderBy(
			filters.Sort,
		)
	if err != nil {
		return BrowsePage{},
			err
	}

	listQuery :=
		browseSearchBase +
			`
				SELECT
					f.id::text,
					f.product_code,
					f.name,
					f.slug,
					f.brand,
					f.short_description,
					f.is_featured,
					f.primary_image_url,
					f.price_amount,
					f.currency,
					f.in_stock,
					f.matched_sku,
					f.category_id::text,
					f.category_name,
					f.category_slug,

					CASE
						WHEN f.search_score >= 950
						THEN 'exact'

						ELSE 'primary'
					END AS match_type,

					1 AS search_tier,

					count(*) OVER ()
						AS total_count

				FROM filtered f
			` +
			orderBy +
			`
				LIMIT $7
				OFFSET $8
			`

	facetQuery :=
		browseSearchBase +
			`
				SELECT
					COALESCE(
						(
							SELECT jsonb_agg(
								jsonb_build_object(
									'id',
									cf.category_id::text,

									'name',
									cf.category_name,

									'slug',
									cf.category_slug,

									'count',
									cf.item_count
								)
								ORDER BY
									cf.item_count DESC,
									cf.category_name ASC
							)

							FROM (
								SELECT
									category_id,
									category_name,
									category_slug,

									count(*)::bigint
										AS item_count

								FROM filtered

								GROUP BY
									category_id,
									category_name,
									category_slug

								ORDER BY
									item_count DESC,
									category_name ASC

								LIMIT 50
							) cf
						),
						'[]'::jsonb
					) AS categories,

					COALESCE(
						(
							SELECT jsonb_agg(
								jsonb_build_object(
									'name',
									bf.brand,

									'count',
									bf.item_count
								)
								ORDER BY
									bf.item_count DESC,
									bf.brand ASC
							)

							FROM (
								SELECT
									trim(brand)
										AS brand,

									count(*)::bigint
										AS item_count

								FROM filtered

								WHERE
									brand IS NOT NULL

									AND trim(brand) <> ''

								GROUP BY
									trim(brand)

								ORDER BY
									item_count DESC,
									brand ASC

								LIMIT 50
							) bf
						),
						'[]'::jsonb
					) AS brands,

					min(
						price_amount
					),

					max(
						price_amount
					),

					count(*)::bigint,

					count(*) FILTER (
						WHERE in_stock
					)::bigint,

					count(*) FILTER (
						WHERE NOT in_stock
					)::bigint

				FROM filtered
			`

	minPrice :=
		searchNullableInt64(
			filters.MinPrice,
		)

	maxPrice :=
		searchNullableInt64(
			filters.MaxPrice,
		)

	inStock :=
		searchNullableBool(
			filters.InStock,
		)

	batch :=
		&pgx.Batch{}

	batch.Queue(
		listQuery,
		normalizedQuery,
		filters.CategorySlug,
		filters.Brand,
		minPrice,
		maxPrice,
		inStock,
		limit,
		offset,
	)

	batch.Queue(
		facetQuery,
		normalizedQuery,
		filters.CategorySlug,
		filters.Brand,
		minPrice,
		maxPrice,
		inStock,
	)

	results :=
		r.db.SendBatch(
			ctx,
			batch,
		)

	defer results.Close()

	rows, err :=
		results.Query()
	if err != nil {
		return BrowsePage{},
			fmt.Errorf(
				"query filterable search: %w",
				err,
			)
	}

	items :=
		make(
			[]ProductResult,
			0,
			limit,
		)

	var total int64

	for rows.Next() {
		var item ProductResult

		var rowTotal int64

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
				&item.MatchedSKU,
				&item.Category.ID,
				&item.Category.Name,
				&item.Category.Slug,
				&item.MatchType,
				&item.SearchTier,
				&rowTotal,
			); err != nil {

			rows.Close()

			return BrowsePage{},
				fmt.Errorf(
					"scan filterable search product: %w",
					err,
				)
		}

		total =
			rowTotal

		items =
			append(
				items,
				item,
			)
	}

	if err :=
		rows.Err(); err != nil {

		rows.Close()

		return BrowsePage{},
			fmt.Errorf(
				"iterate filterable search products: %w",
				err,
			)
	}

	rows.Close()

	var categoryJSON []byte
	var brandJSON []byte

	var minPriceDB pgtype.Int8
	var maxPriceDB pgtype.Int8

	var facetTotal int64

	var inStockCount int64
	var outOfStockCount int64

	if err :=
		results.QueryRow().Scan(
			&categoryJSON,
			&brandJSON,
			&minPriceDB,
			&maxPriceDB,
			&facetTotal,
			&inStockCount,
			&outOfStockCount,
		); err != nil {

		return BrowsePage{},
			fmt.Errorf(
				"query search facets: %w",
				err,
			)
	}

	facets :=
		Facets{
			Categories: make(
				[]CategoryFacet,
				0,
			),

			Brands: make(
				[]BrandFacet,
				0,
			),

			Stock: StockFacet{
				InStock: inStockCount,

				OutOfStock: outOfStockCount,
			},
		}

	if len(categoryJSON) > 0 {
		if err :=
			json.Unmarshal(
				categoryJSON,
				&facets.Categories,
			); err != nil {

			return BrowsePage{},
				fmt.Errorf(
					"decode category facets: %w",
					err,
				)
		}
	}

	if len(brandJSON) > 0 {
		if err :=
			json.Unmarshal(
				brandJSON,
				&facets.Brands,
			); err != nil {

			return BrowsePage{},
				fmt.Errorf(
					"decode brand facets: %w",
					err,
				)
		}
	}

	if minPriceDB.Valid {
		value :=
			minPriceDB.Int64

		facets.Price.MinAmount =
			&value
	}

	if maxPriceDB.Valid {
		value :=
			maxPriceDB.Int64

		facets.Price.MaxAmount =
			&value
	}

	// The facet query provides the authoritative total even when
	// the requested page is beyond the final page.
	total =
		facetTotal

	return BrowsePage{
		Items: items,

		Total: total,

		Facets: facets,
	}, nil
}

func browseOrderBy(
	sortValue string,
) (string, error) {
	switch sortValue {
	case SearchSortRelevance:

		return `
			ORDER BY
				f.search_score DESC,
				f.in_stock DESC,
				f.is_featured DESC,
				f.published_at DESC NULLS LAST,
				f.created_at DESC,
				f.id ASC
		`, nil

	case SearchSortFeatured:

		return `
			ORDER BY
				f.is_featured DESC,
				f.search_score DESC,
				f.in_stock DESC,
				f.published_at DESC NULLS LAST,
				f.created_at DESC,
				f.id ASC
		`, nil

	case SearchSortNewest:

		return `
			ORDER BY
				f.published_at DESC NULLS LAST,
				f.created_at DESC,
				f.search_score DESC,
				f.id ASC
		`, nil

	case SearchSortPriceAsc:

		return `
			ORDER BY
				f.price_amount ASC,
				f.search_score DESC,
				f.id ASC
		`, nil

	case SearchSortPriceDesc:

		return `
			ORDER BY
				f.price_amount DESC,
				f.search_score DESC,
				f.id ASC
		`, nil

	case SearchSortNameAsc:

		return `
			ORDER BY
				lower(f.name) ASC,
				f.id ASC
		`, nil

	case SearchSortNameDesc:

		return `
			ORDER BY
				lower(f.name) DESC,
				f.id ASC
		`, nil

	default:

		return "",
			ErrInvalidSort
	}
}

func searchNullableInt64(
	value *int64,
) any {
	if value == nil {
		return nil
	}

	return *value
}

func searchNullableBool(
	value *bool,
) any {
	if value == nil {
		return nil
	}

	return *value
}
