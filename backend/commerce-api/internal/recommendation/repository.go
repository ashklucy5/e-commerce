package recommendation

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
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

func (r *Repository) RankProductIDs(
	ctx context.Context,
	customerID string,
	limit int,
) ([]string, error) {
	if limit <= 0 {
		limit = 200
	}

	if limit > 500 {
		limit = 500
	}

	const query = `
		WITH deleted_expired AS (
			DELETE FROM customer_search_history
			WHERE
				customer_id = $1::uuid
				AND expires_at <= now()
			RETURNING id
		),

		active_signals AS (
			SELECT
				hc.category_id AS source_category_id,
				h.created_at,

				(
					hc.relevance_weight::double precision
					/ 100.0
				)
				*
				CASE
					WHEN h.created_at >=
						now() - interval '24 hours'
					THEN 1.00::double precision

					WHEN h.created_at >=
						now() - interval '48 hours'
					THEN 0.65::double precision

					ELSE 0.35::double precision
				END AS signal_score

			FROM customer_search_history h

			JOIN customer_search_history_categories hc
				ON hc.search_history_id = h.id

			JOIN categories source_category
				ON source_category.id = hc.category_id
				AND source_category.is_active = true

			WHERE
				h.customer_id = $1::uuid
				AND h.expires_at > now()
				AND h.created_at >=
					now() - interval '72 hours'
		),

		intents AS (
			SELECT
				source_category_id,

				MAX(
					signal_score
				) AS intent_score,

				MAX(
					created_at
				) AS latest_at

			FROM active_signals

			GROUP BY source_category_id
		),

		intent_ordered AS (
			SELECT
				i.*,

				row_number() OVER (
					ORDER BY
						i.intent_score DESC,
						i.latest_at DESC,
						i.source_category_id ASC
				) AS intent_order

			FROM intents i
		),

		intent_relations_raw AS (
			-- Direct category:
			-- strongest recommendation candidate source.
			SELECT
				i.source_category_id,
				i.source_category_id AS target_category_id,
				i.intent_score,
				i.latest_at,
				1.00::double precision AS relation_score

			FROM intents i

			UNION ALL

			-- Explicit seller/admin-managed category graph.
			SELECT
				i.source_category_id,
				r.target_category_id,
				i.intent_score,
				i.latest_at,

				(
					r.weight::double precision
					/ 100.0
				)
				*
				CASE r.relation_type
					WHEN 'related'
					THEN 0.80::double precision

					WHEN 'complementary'
					THEN 0.65::double precision

					ELSE 0.00::double precision
				END AS relation_score

			FROM intents i

			JOIN recommendation_category_relations r
				ON r.source_category_id =
					i.source_category_id

			JOIN categories target_category
				ON target_category.id =
					r.target_category_id
				AND target_category.is_active = true

			WHERE r.relation_type IN (
				'related',
				'complementary'
			)

			UNION ALL

			-- Taxonomy fallback.
			--
			-- Sibling categories are a conservative related
			-- discovery source when no explicit relation is needed.
			SELECT
				i.source_category_id,
				sibling.id AS target_category_id,
				i.intent_score,
				i.latest_at,
				0.50::double precision AS relation_score

			FROM intents i

			JOIN categories source_category
				ON source_category.id =
					i.source_category_id
				AND source_category.is_active = true

			JOIN categories sibling
				ON sibling.parent_id =
					source_category.parent_id
				AND sibling.id <>
					source_category.id
				AND sibling.is_active = true

			WHERE source_category.parent_id
				IS NOT NULL
		),

		intent_relations AS (
			SELECT
				source_category_id,
				target_category_id,

				MAX(
					intent_score
				) AS intent_score,

				MAX(
					latest_at
				) AS latest_at,

				MAX(
					relation_score
				) AS relation_score

			FROM intent_relations_raw

			WHERE relation_score > 0

			GROUP BY
				source_category_id,
				target_category_id
		),

		candidates AS (
			SELECT
				p.id AS product_id,
				ir.source_category_id,
				io.intent_order,

				ir.intent_score
				* ir.relation_score
					AS candidate_score,

				ir.relation_score,
				p.is_featured,
				p.published_at,
				p.created_at

			FROM intent_relations ir

			JOIN intent_ordered io
				ON io.source_category_id =
					ir.source_category_id

			JOIN products p
				ON p.category_id =
					ir.target_category_id
				AND p.status = 'active'

			JOIN categories product_category
				ON product_category.id =
					p.category_id
				AND product_category.is_active = true

			WHERE EXISTS (
				SELECT 1

				FROM product_variants v

				JOIN inventory inv
					ON inv.variant_id = v.id

				WHERE
					v.product_id = p.id
					AND v.is_active = true
					AND (
						inv.quantity_on_hand
						- inv.quantity_reserved
					) > 0
			)
		),

		ranked_per_intent AS (
			SELECT
				c.*,

				row_number() OVER (
					PARTITION BY
						c.source_category_id

					ORDER BY
						c.candidate_score DESC,
						c.relation_score DESC,
						c.is_featured DESC,
						c.published_at DESC NULLS LAST,
						c.created_at DESC,
						c.product_id ASC
				) AS candidate_rank

			FROM candidates c
		),

		deduplicated AS (
			SELECT
				rpi.*,

				row_number() OVER (
					PARTITION BY
						rpi.product_id

					ORDER BY
						rpi.candidate_rank ASC,
						rpi.intent_order ASC,
						rpi.candidate_score DESC,
						rpi.product_id ASC
				) AS product_pick

			FROM ranked_per_intent rpi
		)

		SELECT
			product_id::text

		FROM deduplicated

		WHERE product_pick = 1

		-- Important:
		-- candidate_rank comes before intent_order.
		--
		-- This gives us a round-robin shape:
		-- first candidate from each intent,
		-- then second candidate from each intent,
		-- etc.
		ORDER BY
			candidate_rank ASC,
			intent_order ASC,
			candidate_score DESC,
			product_id ASC

		LIMIT $2
	`

	rows, err := r.db.Query(
		ctx,
		query,
		customerID,
		limit,
	)
	if err != nil {
		return nil,
			fmt.Errorf(
				"rank recommendation products: %w",
				err,
			)
	}
	defer rows.Close()

	productIDs := make(
		[]string,
		0,
	)

	for rows.Next() {
		var productID string

		if err := rows.Scan(
			&productID,
		); err != nil {
			return nil,
				fmt.Errorf(
					"scan recommendation product: %w",
					err,
				)
		}

		productIDs = append(
			productIDs,
			productID,
		)
	}

	if err := rows.Err(); err != nil {
		return nil,
			fmt.Errorf(
				"iterate recommendation products: %w",
				err,
			)
	}

	return productIDs, nil
}

func (r *Repository) RecordEvent(
	ctx context.Context,
	input EventInput,
) (Event, error) {
	const query = `
		INSERT INTO recommendation_events (
			customer_id,
			session_id,
			product_id,
			placement,
			event_type,
			strategy,
			rank_position
		)
		SELECT
			NULLIF($1, '')::uuid,
			NULLIF($2, ''),
			p.id,
			$4,
			$5,
			$6,
			$7
		FROM products p
		WHERE p.id = $3::uuid
		  AND p.status = 'active'
		RETURNING
			id::text,
			COALESCE(customer_id::text, ''),
			COALESCE(session_id, ''),
			product_id::text,
			placement,
			event_type,
			strategy,
			rank_position,
			created_at
	`

	var event Event
	err := r.db.QueryRow(
		ctx,
		query,
		input.CustomerID,
		input.SessionID,
		input.ProductID,
		input.Placement,
		input.EventType,
		input.Strategy,
		input.RankPosition,
	).Scan(
		&event.ID,
		&event.CustomerID,
		&event.SessionID,
		&event.ProductID,
		&event.Placement,
		&event.EventType,
		&event.Strategy,
		&event.RankPosition,
		&event.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return Event{}, ErrProductNotFound
		}
		return Event{}, fmt.Errorf("record recommendation event: %w", err)
	}

	return event, nil
}
