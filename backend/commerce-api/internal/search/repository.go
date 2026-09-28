package search

import (
	"context"
	"fmt"

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

func (r *Repository) SearchProducts(
	ctx context.Context,
	normalizedQuery string,
	page int,
	limit int,
) (SearchPage, error) {
	offset := (page - 1) * limit

	const query = `
		WITH RECURSIVE
		input_base AS (
			SELECT lower(
				regexp_replace(
					trim($1::text),
					'\s+',
					' ',
					'g'
				)
			) AS q
		),
		input_words AS (
			SELECT
				q,
				lower(
					regexp_replace(
						replace(q, '-', ' '),
						'\s+',
						' ',
						'g'
					)
				) AS q_words
			FROM input_base
		),
		input AS (
			SELECT
				q,
				q_words,
				CASE
					WHEN char_length(q_words) > 3
						AND q_words LIKE '%s'
						AND q_words NOT LIKE '%ss'
					THEN left(q_words, char_length(q_words) - 1)
					ELSE q_words
				END AS q_singular,
				regexp_replace(
					q_words,
					'[^[:alnum:]]+',
					'',
					'g'
				) AS q_compact,
				cardinality(
					regexp_split_to_array(q_words, '\s+')
				) AS word_count,
				(
					q ~ '[0-9]'
					OR q LIKE '%-%'
				) AS identifier_like,
				CASE
					WHEN char_length(
						regexp_replace(
							q_words,
							'[^[:alnum:]]+',
							'',
							'g'
						)
					) <= 5
					THEN 0.33::real
					WHEN cardinality(
						regexp_split_to_array(q_words, '\s+')
					) = 1
					THEN 0.45::real
					ELSE 0.55::real
				END AS fuzzy_name_threshold
			FROM input_words
		),
		category_normalized AS (
			SELECT
				c.id,
				c.parent_id,
				c.name,
				c.slug,
				c.sort_order,
				lower(
					regexp_replace(
						replace(c.name, '-', ' '),
						'\s+',
						' ',
						'g'
					)
				) AS norm_name,
				lower(
					regexp_replace(
						replace(c.slug, '-', ' '),
						'\s+',
						' ',
						'g'
					)
				) AS norm_slug
			FROM categories c
			WHERE c.is_active = true
		),
		category_forms AS (
			SELECT
				cn.*,
				CASE
					WHEN char_length(cn.norm_name) > 3
						AND cn.norm_name LIKE '%s'
						AND cn.norm_name NOT LIKE '%ss'
					THEN left(
						cn.norm_name,
						char_length(cn.norm_name) - 1
					)
					ELSE cn.norm_name
				END AS singular_name,
				regexp_replace(
					cn.norm_name,
					'[^[:alnum:]]+',
					'',
					'g'
				) AS compact_name
			FROM category_normalized cn
		),
		category_candidates AS (
			SELECT
				cf.*,
				s.score
			FROM category_forms cf
			CROSS JOIN input i
			CROSS JOIN LATERAL (
				SELECT CASE
					WHEN cf.norm_name = i.q_words
						OR cf.singular_name = i.q_singular
						OR cf.norm_slug = i.q_words
					THEN 1000::real
					WHEN cf.compact_name = i.q_compact
						AND i.q_compact <> ''
					THEN 950::real
					WHEN i.word_count = 1
						AND char_length(i.q_compact) <= 5
						AND word_similarity(
							i.q_words,
							cf.norm_name
						) >= 0.30
					THEN 700::real
						+ word_similarity(
							i.q_words,
							cf.norm_name
						) * 100
						- char_length(cf.norm_name)
					WHEN similarity(
						i.q_words,
						cf.norm_name
					) >= 0.38
					THEN 650::real
						+ similarity(
							i.q_words,
							cf.norm_name
						) * 100
					WHEN word_similarity(
						i.q_words,
						cf.norm_name
					) >= 0.55
					THEN 600::real
						+ word_similarity(
							i.q_words,
							cf.norm_name
						) * 100
					ELSE 0::real
				END AS score
			) s
			WHERE s.score > 0
		),
		resolved_category AS (
			SELECT cc.*
			FROM category_candidates cc
			ORDER BY
				cc.score DESC,
				char_length(cc.norm_name) ASC,
				cc.sort_order ASC,
				cc.name ASC
			LIMIT 1
		),
		primary_seed AS (
			SELECT rc.id
			FROM resolved_category rc

			UNION

			SELECT cf.id
			FROM category_forms cf
			CROSS JOIN resolved_category rc
			CROSS JOIN input i
			WHERE
				i.word_count = 1
				AND rc.singular_name NOT LIKE '% %'
				AND (
					cf.singular_name = rc.singular_name
					OR cf.singular_name LIKE (
						'% ' || rc.singular_name
					)
				)
		),
		primary_tree(id) AS (
			SELECT ps.id
			FROM primary_seed ps

			UNION

			SELECT c.id
			FROM categories c
			JOIN primary_tree pt
				ON c.parent_id = pt.id
			WHERE c.is_active = true
		),
		primary_categories AS (
			SELECT DISTINCT id
			FROM primary_tree
		),
		related_seed_raw AS (
			SELECT
				r.target_category_id AS id,
				r.weight::integer AS relation_weight
			FROM recommendation_category_relations r
			JOIN primary_categories pc
				ON pc.id = r.source_category_id
			JOIN categories target
				ON target.id = r.target_category_id
				AND target.is_active = true
			WHERE r.relation_type = 'related'

			UNION ALL

			SELECT
				cf.id,
				70 AS relation_weight
			FROM category_forms cf
			CROSS JOIN resolved_category rc
			WHERE
				cf.id <> rc.id
				AND greatest(
					word_similarity(
						rc.singular_name,
						cf.singular_name
					),
					word_similarity(
						cf.singular_name,
						rc.singular_name
					)
				) >= 0.60
		),
		related_seed AS (
			SELECT
				rsr.id,
				max(rsr.relation_weight) AS relation_weight
			FROM related_seed_raw rsr
			WHERE NOT EXISTS (
				SELECT 1
				FROM primary_categories pc
				WHERE pc.id = rsr.id
			)
			GROUP BY rsr.id
		),
		related_tree(id, relation_weight) AS (
			SELECT
				rs.id,
				rs.relation_weight
			FROM related_seed rs

			UNION

			SELECT
				c.id,
				rt.relation_weight
			FROM categories c
			JOIN related_tree rt
				ON c.parent_id = rt.id
			WHERE c.is_active = true
		),
		related_categories AS (
			SELECT
				rt.id,
				max(rt.relation_weight) AS relation_weight
			FROM related_tree rt
			WHERE NOT EXISTS (
				SELECT 1
				FROM primary_categories pc
				WHERE pc.id = rt.id
			)
			GROUP BY rt.id
		),
		complementary_seed AS (
			SELECT
				r.target_category_id AS id,
				max(r.weight::integer) AS relation_weight
			FROM recommendation_category_relations r
			JOIN primary_categories pc
				ON pc.id = r.source_category_id
			JOIN categories target
				ON target.id = r.target_category_id
				AND target.is_active = true
			WHERE
				r.relation_type = 'complementary'
				AND NOT EXISTS (
					SELECT 1
					FROM primary_categories primary_match
					WHERE primary_match.id = r.target_category_id
				)
				AND NOT EXISTS (
					SELECT 1
					FROM related_categories related_match
					WHERE related_match.id = r.target_category_id
				)
			GROUP BY r.target_category_id
		),
		complementary_tree(id, relation_weight) AS (
			SELECT
				cs.id,
				cs.relation_weight
			FROM complementary_seed cs

			UNION

			SELECT
				c.id,
				ct.relation_weight
			FROM categories c
			JOIN complementary_tree ct
				ON c.parent_id = ct.id
			WHERE c.is_active = true
		),
		complementary_categories AS (
			SELECT
				ct.id,
				max(ct.relation_weight) AS relation_weight
			FROM complementary_tree ct
			WHERE
				NOT EXISTS (
					SELECT 1
					FROM primary_categories pc
					WHERE pc.id = ct.id
				)
				AND NOT EXISTS (
					SELECT 1
					FROM related_categories rc
					WHERE rc.id = ct.id
				)
			GROUP BY ct.id
		),
		exact_identifier_candidates AS (
			SELECT
				p.id AS product_id,
				NULL::text AS matched_sku,
				1 AS exact_rank
			FROM products p
			JOIN categories c
				ON c.id = p.category_id
				AND c.is_active = true
			CROSS JOIN input i
			WHERE
				p.status = 'active'
				AND lower(p.product_code) = i.q
				AND EXISTS (
					SELECT 1
					FROM product_variants active_variant
					WHERE
						active_variant.product_id = p.id
						AND active_variant.is_active = true
				)

			UNION ALL

			SELECT
				v.product_id,
				v.sku AS matched_sku,
				0 AS exact_rank
			FROM product_variants v
			JOIN products p
				ON p.id = v.product_id
				AND p.status = 'active'
			JOIN categories c
				ON c.id = p.category_id
				AND c.is_active = true
			CROSS JOIN input i
			WHERE
				v.is_active = true
				AND lower(v.sku) = i.q
		),
		exact_identifier AS (
			SELECT
				eic.product_id,
				eic.matched_sku
			FROM exact_identifier_candidates eic
			ORDER BY eic.exact_rank ASC
			LIMIT 1
		),
		variant_text AS (
			SELECT
				v.product_id,
				string_agg(
					trim(
						concat_ws(
							' ',
							NULLIF(v.color_name, ''),
							NULLIF(v.size, '')
						)
					),
					' '
				) AS value
			FROM product_variants v
			WHERE v.is_active = true
			GROUP BY v.product_id
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
				COALESCE(vt.value, '') AS variant_text,
				primary_image.url AS primary_image_url,
				price.price_amount,
				price.currency,
				EXISTS (
					SELECT 1
					FROM product_variants stock_variant
					LEFT JOIN inventory stock_inventory
						ON stock_inventory.variant_id = stock_variant.id
					WHERE
						stock_variant.product_id = p.id
						AND stock_variant.is_active = true
						AND (
							COALESCE(stock_inventory.quantity_on_hand, 0)
							- COALESCE(stock_inventory.quantity_reserved, 0)
						) > 0
				) AS in_stock
			FROM products p
			JOIN categories c
				ON c.id = p.category_id
				AND c.is_active = true
			LEFT JOIN variant_text vt
				ON vt.product_id = p.id
			JOIN LATERAL (
				SELECT
					v.price_amount,
					v.currency
				FROM product_variants v
				LEFT JOIN inventory i
					ON i.variant_id = v.id
				WHERE
					v.product_id = p.id
					AND v.is_active = true
				ORDER BY
					(
						COALESCE(i.quantity_on_hand, 0)
						- COALESCE(i.quantity_reserved, 0)
					) > 0 DESC,
					v.price_amount ASC,
					v.id ASC
				LIMIT 1
			) price
				ON true
			LEFT JOIN LATERAL (
				SELECT pi.url
				FROM product_images pi
				WHERE
					pi.product_id = p.id
					AND pi.is_primary = true
				ORDER BY
					pi.sort_order ASC,
					pi.id ASC
				LIMIT 1
			) primary_image
				ON true
			WHERE p.status = 'active'
		),
		scored AS (
			SELECT
				pb.*,
				CASE
					WHEN ei.product_id IS NOT NULL
						AND pb.id = ei.product_id
					THEN 10000::real
					WHEN ei.product_id IS NOT NULL
					THEN 0::real
					WHEN i.identifier_like
						AND matched_variant.sku IS NOT NULL
					THEN 900::real
						+ matched_variant.sku_similarity * 100
					WHEN i.identifier_like
						AND similarity(
							lower(pb.product_code),
							i.q
						) >= 0.60
					THEN 850::real
						+ similarity(
							lower(pb.product_code),
							i.q
						) * 100
					WHEN NOT i.identifier_like
						AND lower(pb.name) = i.q
					THEN 980::real
					WHEN NOT i.identifier_like
						AND lower(pb.slug) = replace(i.q_words, ' ', '-')
					THEN 970::real
					WHEN NOT i.identifier_like
						AND to_tsvector(
							'simple',
							concat_ws(
								' ',
								pb.name,
								pb.brand,
								pb.short_description,
								pb.category_name,
								pb.variant_text
							)
						) @@ plainto_tsquery('simple', i.q_words)
					THEN 800::real
						+ ts_rank_cd(
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
							plainto_tsquery('simple', i.q_words)
						) * 100
					WHEN NOT i.identifier_like
						AND word_similarity(
							i.q_words,
							lower(pb.name)
						) >= i.fuzzy_name_threshold
					THEN 650::real
						+ word_similarity(
							i.q_words,
							lower(pb.name)
						) * 100
					ELSE 0::real
				END AS direct_score,
				CASE
					WHEN ei.product_id IS NOT NULL
					THEN ei.matched_sku
					ELSE matched_variant.sku
				END AS matched_sku,
				(pc.id IS NOT NULL) AS in_primary_category,
				rc.relation_weight AS related_weight,
				cc.relation_weight AS complementary_weight,
				ei.product_id AS exact_product_id
			FROM product_base pb
			CROSS JOIN input i
			LEFT JOIN exact_identifier ei
				ON true
			LEFT JOIN primary_categories pc
				ON pc.id = pb.category_id
			LEFT JOIN related_categories rc
				ON rc.id = pb.category_id
			LEFT JOIN complementary_categories cc
				ON cc.id = pb.category_id
			LEFT JOIN LATERAL (
				SELECT
					v.sku,
					similarity(
						lower(v.sku),
						i.q
					) AS sku_similarity
				FROM product_variants v
				WHERE
					v.product_id = pb.id
					AND v.is_active = true
					AND i.identifier_like
					AND similarity(
						lower(v.sku),
						i.q
					) >= 0.60
				ORDER BY
					(lower(v.sku) = i.q) DESC,
					similarity(
						lower(v.sku),
						i.q
					) DESC,
					v.sku ASC
				LIMIT 1
			) matched_variant
				ON true
		),
		classified AS (
			SELECT
				s.*,
				CASE
					WHEN s.exact_product_id IS NOT NULL THEN
						CASE
							WHEN s.id = s.exact_product_id THEN 0
							ELSE NULL
						END
					WHEN s.direct_score > 0
						OR s.in_primary_category
					THEN 1
					WHEN s.related_weight IS NOT NULL
					THEN 2
					WHEN s.complementary_weight IS NOT NULL
					THEN 3
					ELSE NULL
				END AS search_tier,
				CASE
					WHEN s.exact_product_id IS NOT NULL
						AND s.id = s.exact_product_id
					THEN 'exact'
					WHEN s.exact_product_id IS NULL
						AND (
							s.direct_score > 0
							OR s.in_primary_category
						)
					THEN 'primary'
					WHEN s.exact_product_id IS NULL
						AND s.related_weight IS NOT NULL
					THEN 'related'
					WHEN s.exact_product_id IS NULL
						AND s.complementary_weight IS NOT NULL
					THEN 'complementary'
					ELSE ''
				END AS match_type,
				greatest(
					COALESCE(s.related_weight, 0),
					COALESCE(s.complementary_weight, 0)
				) AS relation_weight
			FROM scored s
		),
		ranked AS (
			SELECT
				c.*,
				count(*) OVER () AS total_count,
				count(*) FILTER (
					WHERE c.search_tier <= 1
				) OVER () AS primary_count,
				count(*) FILTER (
					WHERE c.search_tier >= 2
				) OVER () AS similar_count
			FROM classified c
			WHERE c.search_tier IS NOT NULL
		)
		SELECT
			r.id::text,
			r.product_code,
			r.name,
			r.slug,
			r.brand,
			r.short_description,
			r.is_featured,
			r.primary_image_url,
			r.price_amount,
			r.currency,
			r.in_stock,
			r.matched_sku,
			r.category_id::text,
			r.category_name,
			r.category_slug,
			r.match_type,
			r.search_tier,
			r.total_count,
			r.primary_count,
			r.similar_count
		FROM ranked r
		ORDER BY
			r.search_tier ASC,
			r.direct_score DESC,
			r.relation_weight DESC,
			r.in_stock DESC,
			r.is_featured DESC,
			r.published_at DESC NULLS LAST,
			r.created_at DESC,
			r.id ASC
		LIMIT $2
		OFFSET $3
	`

	rows, err := r.db.Query(
		ctx,
		query,
		normalizedQuery,
		limit,
		offset,
	)
	if err != nil {
		return SearchPage{}, fmt.Errorf(
			"search products: %w",
			err,
		)
	}
	defer rows.Close()

	result := SearchPage{
		Items: make(
			[]ProductResult,
			0,
			limit,
		),
	}

	for rows.Next() {
		var item ProductResult
		var total int64
		var primaryTotal int64
		var similarTotal int64

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
			&item.Currency,
			&item.InStock,
			&item.MatchedSKU,
			&item.Category.ID,
			&item.Category.Name,
			&item.Category.Slug,
			&item.MatchType,
			&item.SearchTier,
			&total,
			&primaryTotal,
			&similarTotal,
		); err != nil {
			return SearchPage{}, fmt.Errorf(
				"scan search product: %w",
				err,
			)
		}

		result.Total = total
		result.PrimaryTotal = primaryTotal
		result.SimilarTotal = similarTotal
		result.Items = append(
			result.Items,
			item,
		)
	}

	if err := rows.Err(); err != nil {
		return SearchPage{}, fmt.Errorf(
			"iterate search products: %w",
			err,
		)
	}

	return result, nil
}

func (r *Repository) RecordSearchHistory(
	ctx context.Context,
	customerID string,
	query string,
	normalizedQuery string,
	resultCount int64,
	categories []HistoryCategorySignal,
) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf(
			"begin search history transaction: %w",
			err,
		)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if _, err := tx.Exec(
		ctx,
		`
			DELETE FROM customer_search_history
			WHERE
				customer_id = $1::uuid
				AND expires_at <= now()
		`,
		customerID,
	); err != nil {
		return fmt.Errorf(
			"delete expired search history: %w",
			err,
		)
	}

	var historyID string

	if err := tx.QueryRow(
		ctx,
		`
			INSERT INTO customer_search_history (
				customer_id,
				query,
				normalized_query,
				result_count,
				created_at,
				expires_at
			)
			VALUES (
				$1::uuid,
				$2,
				$3,
				$4,
				now(),
				now() + interval '72 hours'
			)
			RETURNING id::text
		`,
		customerID,
		query,
		normalizedQuery,
		resultCount,
	).Scan(
		&historyID,
	); err != nil {
		return fmt.Errorf(
			"insert search history: %w",
			err,
		)
	}

	for _, category := range categories {
		if _, err := tx.Exec(
			ctx,
			`
				INSERT INTO customer_search_history_categories (
					search_history_id,
					category_id,
					relevance_weight
				)
				VALUES (
					$1::uuid,
					$2::uuid,
					$3
				)
				ON CONFLICT (
					search_history_id,
					category_id
				)
				DO UPDATE SET
					relevance_weight = EXCLUDED.relevance_weight
			`,
			historyID,
			category.CategoryID,
			category.RelevanceWeight,
		); err != nil {
			return fmt.Errorf(
				"insert search history category: %w",
				err,
			)
		}
	}

	if _, err := tx.Exec(
		ctx,
		`
			DELETE FROM customer_search_history
			WHERE id IN (
				SELECT id
				FROM customer_search_history
				WHERE customer_id = $1::uuid
				ORDER BY
					created_at DESC,
					id DESC
				OFFSET 250
			)
		`,
		customerID,
	); err != nil {
		return fmt.Errorf(
			"trim customer search history: %w",
			err,
		)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf(
			"commit search history transaction: %w",
			err,
		)
	}

	return nil
}
