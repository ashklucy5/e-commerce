package search

import (
	"context"
	"fmt"
)

type EmbeddingDocument struct {
	ProductID string
	Text      string
	ImageURL  string

	StoredHash  string
	StoredModel string
}

func (r *Repository) SearchSemanticProducts(
	ctx context.Context,
	queryVector string,
	model string,
	page int,
	limit int,
	primaryMin float64,
	similarMin float64,
) (SearchPage, error) {
	offset :=
		(page - 1) *
			limit

	const query = `
		WITH candidates AS (
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

				c.id AS category_id,
				c.name AS category_name,
				c.slug AS category_slug,

				primary_image.url
					AS primary_image_url,

				price.price_amount,
				price.currency,

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
				) AS in_stock,

				(
					1.0 -
					(
						pse.embedding::vector(1024)
						<=>
						$1::vector(1024)
					)
				) AS semantic_similarity

			FROM product_search_embeddings pse

			JOIN products p
				ON p.id =
					pse.product_id
				AND p.status =
					'active'

			JOIN categories c
				ON c.id =
					p.category_id
				AND c.is_active =
					true

			JOIN LATERAL (
				SELECT
					v.price_amount,
					v.currency
				FROM product_variants v
				LEFT JOIN inventory i
					ON i.variant_id =
						v.id
				WHERE
					v.product_id =
						p.id
					AND v.is_active =
						true
				ORDER BY
					(
						COALESCE(
							i.quantity_on_hand,
							0
						)
						-
						COALESCE(
							i.quantity_reserved,
							0
						)
					) > 0 DESC,
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

			WHERE
				pse.embedding_model =
					$2
				AND cardinality(
					pse.embedding
				) = 1024
		),

		classified AS (
			SELECT
				c.*,

				CASE
					WHEN c.semantic_similarity >=
						$5::double precision
					THEN 1

					WHEN c.semantic_similarity >=
						$6::double precision
					THEN 2

					ELSE NULL
				END AS search_tier,

				CASE
					WHEN c.semantic_similarity >=
						$5::double precision
					THEN 'primary'

					WHEN c.semantic_similarity >=
						$6::double precision
					THEN 'similar'

					ELSE ''
				END AS match_type

			FROM candidates c
		),

		ranked AS (
			SELECT
				c.*,

				count(*) OVER ()
					AS total_count,

				count(*) FILTER (
					WHERE c.search_tier = 1
				) OVER ()
					AS primary_count,

				count(*) FILTER (
					WHERE c.search_tier = 2
				) OVER ()
					AS similar_count

			FROM classified c

			WHERE
				c.search_tier IS NOT NULL
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

			NULL::text AS matched_sku,

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
			r.semantic_similarity DESC,
			r.in_stock DESC,
			r.is_featured DESC,
			r.published_at DESC NULLS LAST,
			r.created_at DESC,
			r.id ASC

		LIMIT $3
		OFFSET $4
	`

	rows, err :=
		r.db.Query(
			ctx,
			query,
			queryVector,
			model,
			limit,
			offset,
			primaryMin,
			similarMin,
		)
	if err != nil {
		return SearchPage{},
			fmt.Errorf(
				"search semantic products: %w",
				err,
			)
	}

	defer rows.Close()

	result :=
		SearchPage{
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
				&total,
				&primaryTotal,
				&similarTotal,
			); err != nil {
			return SearchPage{},
				fmt.Errorf(
					"scan semantic search product: %w",
					err,
				)
		}

		result.Total =
			total

		result.PrimaryTotal =
			primaryTotal

		result.SimilarTotal =
			similarTotal

		result.Items =
			append(
				result.Items,
				item,
			)
	}

	if err :=
		rows.Err(); err != nil {
		return SearchPage{},
			fmt.Errorf(
				"iterate semantic search products: %w",
				err,
			)
	}

	return result, nil
}

func (r *Repository) ListEmbeddingDocuments(
	ctx context.Context,
) ([]EmbeddingDocument, error) {
	const query = `
		SELECT
			p.id::text,

			concat_ws(
				E'\n',

				'Product name: ' ||
					p.name,

				'Category: ' ||
					c.name,

				'Category slug: ' ||
					c.slug,

				'Brand: ' ||
					COALESCE(
						p.brand,
						''
					),

				'Product code: ' ||
					p.product_code,

				'Short description: ' ||
					COALESCE(
						p.short_description,
						''
					),

				'Description: ' ||
					COALESCE(
						p.description,
						''
					),

				'Variants: ' ||
					COALESCE(
						variant_text.value,
						''
					),

				'Primary image alt text: ' ||
					COALESCE(
						primary_image.alt_text,
						''
					)
			) AS document_text,

			COALESCE(
				primary_image.url,
				''
			) AS primary_image_url,

			COALESCE(
				pse.document_hash,
				''
			),

			COALESCE(
				pse.embedding_model,
				''
			)

		FROM products p

		JOIN categories c
			ON c.id =
				p.category_id
			AND c.is_active =
				true

		LEFT JOIN product_search_embeddings pse
			ON pse.product_id =
				p.id

		LEFT JOIN LATERAL (
			SELECT
				string_agg(
					concat_ws(
						' ',
						v.sku,
						NULLIF(
							v.color_name,
							''
						),
						NULLIF(
							v.size,
							''
						)
					),
					', '
					ORDER BY v.sku
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
				pi.url,
				pi.alt_text

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

		WHERE
			p.status =
				'active'

		ORDER BY
			p.id ASC
	`

	rows, err :=
		r.db.Query(
			ctx,
			query,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"list embedding documents: %w",
				err,
			)
	}

	defer rows.Close()

	result :=
		make(
			[]EmbeddingDocument,
			0,
		)

	for rows.Next() {
		var item EmbeddingDocument

		if err :=
			rows.Scan(
				&item.ProductID,
				&item.Text,
				&item.ImageURL,
				&item.StoredHash,
				&item.StoredModel,
			); err != nil {
			return nil,
				fmt.Errorf(
					"scan embedding document: %w",
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
				"iterate embedding documents: %w",
				err,
			)
	}

	return result, nil
}

func (r *Repository) UpsertProductEmbedding(
	ctx context.Context,
	productID string,
	documentText string,
	documentHash string,
	embedding []float32,
	model string,
) error {
	if err :=
		validateEmbedding(
			embedding,
		); err != nil {
		return err
	}

	_, err :=
		r.db.Exec(
			ctx,
			`
				INSERT INTO product_search_embeddings (
					product_id,
					document_text,
					document_hash,
					embedding,
					embedding_model,
					created_at,
					updated_at
				)
				VALUES (
					$1::uuid,
					$2,
					$3,
					$4::real[],
					$5,
					now(),
					now()
				)

				ON CONFLICT (
					product_id
				)
				DO UPDATE SET
					document_text =
						EXCLUDED.document_text,

					document_hash =
						EXCLUDED.document_hash,

					embedding =
						EXCLUDED.embedding,

					embedding_model =
						EXCLUDED.embedding_model,

					updated_at =
						now()
			`,
			productID,
			documentText,
			documentHash,
			embedding,
			model,
		)
	if err != nil {
		return fmt.Errorf(
			"upsert product embedding: %w",
			err,
		)
	}

	return nil
}

func (r *Repository) DeleteInactiveProductEmbeddings(
	ctx context.Context,
) (int64, error) {
	tag, err :=
		r.db.Exec(
			ctx,
			`
				DELETE FROM product_search_embeddings pse

				WHERE NOT EXISTS (
					SELECT 1

					FROM products p

					JOIN categories c
						ON c.id =
							p.category_id
						AND c.is_active =
							true

					WHERE
						p.id =
							pse.product_id
						AND p.status =
							'active'
				)
			`,
		)
	if err != nil {
		return 0,
			fmt.Errorf(
				"delete inactive product embeddings: %w",
				err,
			)
	}

	return tag.RowsAffected(),
		nil
}
