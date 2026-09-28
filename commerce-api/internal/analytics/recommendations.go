package analytics

import (
	"context"
	"fmt"
	"time"
)

const recommendationAttributionWindowDays = 7

type RecommendationPerformance struct {
	Impressions int64 `json:"impressions"`
	Clicks      int64 `json:"clicks"`
	AddToCarts  int64 `json:"add_to_carts"`

	ClickThroughRateBPS int64 `json:"click_through_rate_bps"`
	AddToCartRateBPS    int64 `json:"add_to_cart_rate_bps"`

	AttributedOrders            int64 `json:"attributed_orders"`
	AttributedUnits             int64 `json:"attributed_units"`
	AttributedMerchandiseAmount int64 `json:"attributed_merchandise_amount"`

	AttributionWindowDays int    `json:"attribution_window_days"`
	Currency              string `json:"currency"`
}

type RecommendationBreakdown struct {
	Key string `json:"key"`

	Impressions int64 `json:"impressions"`
	Clicks      int64 `json:"clicks"`
	AddToCarts  int64 `json:"add_to_carts"`

	ClickThroughRateBPS int64 `json:"click_through_rate_bps"`
	AddToCartRateBPS    int64 `json:"add_to_cart_rate_bps"`

	AttributedOrders            int64 `json:"attributed_orders"`
	AttributedUnits             int64 `json:"attributed_units"`
	AttributedMerchandiseAmount int64 `json:"attributed_merchandise_amount"`

	Currency string `json:"currency"`
}

type RecommendationTrendPoint struct {
	BucketStart string `json:"bucket_start"`

	Impressions int64 `json:"impressions"`
	Clicks      int64 `json:"clicks"`
	AddToCarts  int64 `json:"add_to_carts"`

	AttributedOrders            int64 `json:"attributed_orders"`
	AttributedUnits             int64 `json:"attributed_units"`
	AttributedMerchandiseAmount int64 `json:"attributed_merchandise_amount"`
}

type RecommendationProduct struct {
	ProductID   string `json:"product_id"`
	ProductName string `json:"product_name"`

	Impressions int64 `json:"impressions"`
	Clicks      int64 `json:"clicks"`
	AddToCarts  int64 `json:"add_to_carts"`

	ClickThroughRateBPS int64 `json:"click_through_rate_bps"`

	AttributedOrders            int64 `json:"attributed_orders"`
	AttributedUnits             int64 `json:"attributed_units"`
	AttributedMerchandiseAmount int64 `json:"attributed_merchandise_amount"`

	Currency string `json:"currency"`
}

type RecommendationEngineDiagnostics struct {
	EligibleProducts          int64 `json:"eligible_products"`
	CustomersWithActiveIntent int64 `json:"customers_with_active_intent"`
	ActiveSearchSignals       int64 `json:"active_search_signals"`
	RelatedRelations          int64 `json:"related_relations"`
	ComplementaryRelations    int64 `json:"complementary_relations"`
	BehaviorWindowHours       int   `json:"behavior_window_hours"`
}

type RecommendationCategoryRelation struct {
	SourceCategoryID   string `json:"source_category_id"`
	SourceCategoryName string `json:"source_category_name"`
	TargetCategoryID   string `json:"target_category_id"`
	TargetCategoryName string `json:"target_category_name"`
	RelationType       string `json:"relation_type"`
	Weight             int    `json:"weight"`
}

func (s *Service) RecommendationOverview(ctx context.Context, query Query) (RecommendationPerformance, error) {
	queryCtx, cancel, err := s.queryContext(ctx)
	if err != nil {
		return RecommendationPerformance{}, err
	}
	defer cancel()

	var result RecommendationPerformance
	err = s.db.QueryRow(queryCtx, `
		WITH event_totals AS (
			SELECT
				COUNT(*) FILTER (WHERE event_type = 'impression')::bigint AS impressions,
				COUNT(*) FILTER (WHERE event_type = 'click')::bigint AS clicks,
				COUNT(*) FILTER (WHERE event_type = 'add_to_cart')::bigint AS add_to_carts
			FROM recommendation_events
			WHERE created_at >= $1 AND created_at < $2
		),
		attributed_items AS (
			SELECT oi.order_id, oi.quantity, oi.line_total_amount
			FROM orders o
			JOIN order_items oi ON oi.order_id = o.id
			JOIN product_variants v ON v.id = oi.variant_id
			WHERE
				o.customer_id IS NOT NULL
				AND o.paid_at >= $1
				AND o.paid_at < $2
				AND o.currency = $3
				AND o.payment_status IN ('paid', 'cod_collected', 'refunded')
				AND EXISTS (
					SELECT 1
					FROM recommendation_events re
					WHERE
						re.customer_id = o.customer_id
						AND re.product_id = v.product_id
						AND re.event_type = 'click'
						AND re.created_at <= o.paid_at
						AND re.created_at >= o.paid_at - interval '7 days'
				)
		)
		SELECT
			et.impressions,
			et.clicks,
			et.add_to_carts,
			COUNT(DISTINCT ai.order_id)::bigint,
			COALESCE(SUM(ai.quantity), 0)::bigint,
			COALESCE(SUM(ai.line_total_amount), 0)::bigint
		FROM event_totals et
		LEFT JOIN attributed_items ai ON true
		GROUP BY et.impressions, et.clicks, et.add_to_carts
	`, query.Start, query.End, query.Currency).Scan(
		&result.Impressions,
		&result.Clicks,
		&result.AddToCarts,
		&result.AttributedOrders,
		&result.AttributedUnits,
		&result.AttributedMerchandiseAmount,
	)
	if err != nil {
		return RecommendationPerformance{}, fmt.Errorf("load recommendation analytics overview: %w", err)
	}

	result.ClickThroughRateBPS = basisPoints(result.Clicks, result.Impressions)
	result.AddToCartRateBPS = basisPoints(result.AddToCarts, result.Clicks)
	result.AttributionWindowDays = recommendationAttributionWindowDays
	result.Currency = query.Currency
	return result, nil
}

func (s *Service) RecommendationBreakdownByPlacement(ctx context.Context, query Query) ([]RecommendationBreakdown, error) {
	return s.recommendationBreakdown(ctx, query, "placement")
}

func (s *Service) RecommendationBreakdownByStrategy(ctx context.Context, query Query) ([]RecommendationBreakdown, error) {
	return s.recommendationBreakdown(ctx, query, "strategy")
}

func (s *Service) recommendationBreakdown(ctx context.Context, query Query, dimension string) ([]RecommendationBreakdown, error) {
	if dimension != "placement" && dimension != "strategy" {
		return nil, fmt.Errorf("unsupported recommendation breakdown dimension")
	}

	queryCtx, cancel, err := s.queryContext(ctx)
	if err != nil {
		return nil, err
	}
	defer cancel()

	querySQL := fmt.Sprintf(`
		WITH event_totals AS (
			SELECT
				%s AS key,
				COUNT(*) FILTER (WHERE event_type = 'impression')::bigint AS impressions,
				COUNT(*) FILTER (WHERE event_type = 'click')::bigint AS clicks,
				COUNT(*) FILTER (WHERE event_type = 'add_to_cart')::bigint AS add_to_carts
			FROM recommendation_events
			WHERE created_at >= $1 AND created_at < $2
			GROUP BY %s
		),
		attributed AS (
			SELECT
				click.%s AS key,
				oi.order_id,
				oi.quantity,
				oi.line_total_amount
			FROM orders o
			JOIN order_items oi ON oi.order_id = o.id
			JOIN product_variants v ON v.id = oi.variant_id
			CROSS JOIN LATERAL (
				SELECT re.placement, re.strategy
				FROM recommendation_events re
				WHERE
					re.customer_id = o.customer_id
					AND re.product_id = v.product_id
					AND re.event_type = 'click'
					AND re.created_at <= o.paid_at
					AND re.created_at >= o.paid_at - interval '7 days'
				ORDER BY re.created_at DESC, re.id DESC
				LIMIT 1
			) click
			WHERE
				o.customer_id IS NOT NULL
				AND o.paid_at >= $1
				AND o.paid_at < $2
				AND o.currency = $3
				AND o.payment_status IN ('paid', 'cod_collected', 'refunded')
		),
		attribution_totals AS (
			SELECT
				key,
				COUNT(DISTINCT order_id)::bigint AS attributed_orders,
				COALESCE(SUM(quantity), 0)::bigint AS attributed_units,
				COALESCE(SUM(line_total_amount), 0)::bigint AS attributed_revenue
			FROM attributed
			GROUP BY key
		)
		SELECT
			et.key,
			et.impressions,
			et.clicks,
			et.add_to_carts,
			COALESCE(at.attributed_orders, 0)::bigint,
			COALESCE(at.attributed_units, 0)::bigint,
			COALESCE(at.attributed_revenue, 0)::bigint
		FROM event_totals et
		LEFT JOIN attribution_totals at ON at.key = et.key
		ORDER BY et.impressions DESC, et.key
	`, dimension, dimension, dimension)

	rows, err := s.db.Query(queryCtx, querySQL, query.Start, query.End, query.Currency)
	if err != nil {
		return nil, fmt.Errorf("load recommendation %s analytics: %w", dimension, err)
	}
	defer rows.Close()

	result := make([]RecommendationBreakdown, 0)
	for rows.Next() {
		var item RecommendationBreakdown
		if err := rows.Scan(
			&item.Key,
			&item.Impressions,
			&item.Clicks,
			&item.AddToCarts,
			&item.AttributedOrders,
			&item.AttributedUnits,
			&item.AttributedMerchandiseAmount,
		); err != nil {
			return nil, fmt.Errorf("scan recommendation %s analytics: %w", dimension, err)
		}
		item.ClickThroughRateBPS = basisPoints(item.Clicks, item.Impressions)
		item.AddToCartRateBPS = basisPoints(item.AddToCarts, item.Clicks)
		item.Currency = query.Currency
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate recommendation %s analytics: %w", dimension, err)
	}
	return result, nil
}

func (s *Service) RecommendationTrend(ctx context.Context, query Query) ([]RecommendationTrendPoint, error) {
	queryCtx, cancel, err := s.queryContext(ctx)
	if err != nil {
		return nil, err
	}
	defer cancel()

	rows, err := s.db.Query(queryCtx, `
		WITH event_points AS (
			SELECT
				to_char(date_trunc($4, created_at AT TIME ZONE 'Asia/Dhaka'), 'YYYY-MM-DD') AS bucket_start,
				COUNT(*) FILTER (WHERE event_type = 'impression')::bigint AS impressions,
				COUNT(*) FILTER (WHERE event_type = 'click')::bigint AS clicks,
				COUNT(*) FILTER (WHERE event_type = 'add_to_cart')::bigint AS add_to_carts,
				0::bigint AS attributed_orders,
				0::bigint AS attributed_units,
				0::bigint AS attributed_revenue
			FROM recommendation_events
			WHERE created_at >= $1 AND created_at < $2
			GROUP BY 1
		),
		attributed_points AS (
			SELECT
				to_char(date_trunc($4, o.paid_at AT TIME ZONE 'Asia/Dhaka'), 'YYYY-MM-DD') AS bucket_start,
				0::bigint AS impressions,
				0::bigint AS clicks,
				0::bigint AS add_to_carts,
				COUNT(DISTINCT o.id)::bigint AS attributed_orders,
				COALESCE(SUM(oi.quantity), 0)::bigint AS attributed_units,
				COALESCE(SUM(oi.line_total_amount), 0)::bigint AS attributed_revenue
			FROM orders o
			JOIN order_items oi ON oi.order_id = o.id
			JOIN product_variants v ON v.id = oi.variant_id
			WHERE
				o.customer_id IS NOT NULL
				AND o.paid_at >= $1 AND o.paid_at < $2
				AND o.currency = $3
				AND o.payment_status IN ('paid', 'cod_collected', 'refunded')
				AND EXISTS (
					SELECT 1 FROM recommendation_events re
					WHERE re.customer_id = o.customer_id
					  AND re.product_id = v.product_id
					  AND re.event_type = 'click'
					  AND re.created_at <= o.paid_at
					  AND re.created_at >= o.paid_at - interval '7 days'
				)
			GROUP BY 1
		)
		SELECT
			bucket_start,
			SUM(impressions)::bigint,
			SUM(clicks)::bigint,
			SUM(add_to_carts)::bigint,
			SUM(attributed_orders)::bigint,
			SUM(attributed_units)::bigint,
			SUM(attributed_revenue)::bigint
		FROM (
			SELECT * FROM event_points
			UNION ALL
			SELECT * FROM attributed_points
		) combined
		GROUP BY bucket_start
		ORDER BY bucket_start
	`, query.Start, query.End, query.Currency, query.Granularity)
	if err != nil {
		return nil, fmt.Errorf("load recommendation trend: %w", err)
	}
	defer rows.Close()

	observed := make(map[string]RecommendationTrendPoint)
	for rows.Next() {
		var point RecommendationTrendPoint
		if err := rows.Scan(
			&point.BucketStart,
			&point.Impressions,
			&point.Clicks,
			&point.AddToCarts,
			&point.AttributedOrders,
			&point.AttributedUnits,
			&point.AttributedMerchandiseAmount,
		); err != nil {
			return nil, fmt.Errorf("scan recommendation trend: %w", err)
		}
		observed[point.BucketStart] = point
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate recommendation trend: %w", err)
	}

	result := make([]RecommendationTrendPoint, 0)
	for bucket := analyticsBucketStart(query.Start, query.Granularity); bucket.Before(query.End); bucket = nextRecommendationBucket(bucket, query.Granularity) {
		key := bucket.Format(time.DateOnly)
		point := observed[key]
		point.BucketStart = key
		result = append(result, point)
	}
	return result, nil
}

func (s *Service) RecommendationTopProducts(ctx context.Context, query Query, limit int) ([]RecommendationProduct, error) {
	if limit <= 0 || limit > 50 {
		return nil, fmt.Errorf("%w: limit must be between 1 and 50", ErrInvalidLimit)
	}
	queryCtx, cancel, err := s.queryContext(ctx)
	if err != nil {
		return nil, err
	}
	defer cancel()

	rows, err := s.db.Query(queryCtx, `
		WITH event_totals AS (
			SELECT
				product_id,
				COUNT(*) FILTER (WHERE event_type = 'impression')::bigint AS impressions,
				COUNT(*) FILTER (WHERE event_type = 'click')::bigint AS clicks,
				COUNT(*) FILTER (WHERE event_type = 'add_to_cart')::bigint AS add_to_carts
			FROM recommendation_events
			WHERE created_at >= $1 AND created_at < $2
			GROUP BY product_id
		),
		attributed AS (
			SELECT
				v.product_id,
				COUNT(DISTINCT o.id)::bigint AS attributed_orders,
				COALESCE(SUM(oi.quantity), 0)::bigint AS attributed_units,
				COALESCE(SUM(oi.line_total_amount), 0)::bigint AS attributed_revenue
			FROM orders o
			JOIN order_items oi ON oi.order_id = o.id
			JOIN product_variants v ON v.id = oi.variant_id
			WHERE
				o.customer_id IS NOT NULL
				AND o.paid_at >= $1 AND o.paid_at < $2
				AND o.currency = $3
				AND o.payment_status IN ('paid', 'cod_collected', 'refunded')
				AND EXISTS (
					SELECT 1 FROM recommendation_events re
					WHERE re.customer_id = o.customer_id
					  AND re.product_id = v.product_id
					  AND re.event_type = 'click'
					  AND re.created_at <= o.paid_at
					  AND re.created_at >= o.paid_at - interval '7 days'
				)
			GROUP BY v.product_id
		)
		SELECT
			p.id::text,
			p.name,
			COALESCE(et.impressions, 0)::bigint,
			COALESCE(et.clicks, 0)::bigint,
			COALESCE(et.add_to_carts, 0)::bigint,
			COALESCE(a.attributed_orders, 0)::bigint,
			COALESCE(a.attributed_units, 0)::bigint,
			COALESCE(a.attributed_revenue, 0)::bigint
		FROM products p
		LEFT JOIN event_totals et ON et.product_id = p.id
		LEFT JOIN attributed a ON a.product_id = p.id
		WHERE et.product_id IS NOT NULL OR a.product_id IS NOT NULL
		ORDER BY COALESCE(a.attributed_revenue, 0) DESC, COALESCE(et.clicks, 0) DESC, p.id
		LIMIT $4
	`, query.Start, query.End, query.Currency, limit)
	if err != nil {
		return nil, fmt.Errorf("load recommendation top products: %w", err)
	}
	defer rows.Close()

	result := make([]RecommendationProduct, 0, limit)
	for rows.Next() {
		var item RecommendationProduct
		if err := rows.Scan(
			&item.ProductID,
			&item.ProductName,
			&item.Impressions,
			&item.Clicks,
			&item.AddToCarts,
			&item.AttributedOrders,
			&item.AttributedUnits,
			&item.AttributedMerchandiseAmount,
		); err != nil {
			return nil, fmt.Errorf("scan recommendation top product: %w", err)
		}
		item.ClickThroughRateBPS = basisPoints(item.Clicks, item.Impressions)
		item.Currency = query.Currency
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate recommendation top products: %w", err)
	}
	return result, nil
}

func (s *Service) RecommendationEngineDiagnostics(ctx context.Context) (RecommendationEngineDiagnostics, error) {
	queryCtx, cancel, err := s.queryContext(ctx)
	if err != nil {
		return RecommendationEngineDiagnostics{}, err
	}
	defer cancel()

	var result RecommendationEngineDiagnostics
	err = s.db.QueryRow(queryCtx, `
		SELECT
			(
				SELECT COUNT(*)::bigint
				FROM products p
				WHERE p.status = 'active'
				  AND EXISTS (
					SELECT 1
					FROM product_variants v
					JOIN inventory i ON i.variant_id = v.id
					WHERE v.product_id = p.id
					  AND v.is_active = true
					  AND (i.quantity_on_hand - i.quantity_reserved) > 0
				  )
			),
			(
				SELECT COUNT(DISTINCT customer_id)::bigint
				FROM customer_search_history
				WHERE expires_at > now()
				  AND created_at >= now() - interval '72 hours'
			),
			(
				SELECT COUNT(*)::bigint
				FROM customer_search_history
				WHERE expires_at > now()
				  AND created_at >= now() - interval '72 hours'
			),
			(
				SELECT COUNT(*)::bigint
				FROM recommendation_category_relations
				WHERE relation_type = 'related'
			),
			(
				SELECT COUNT(*)::bigint
				FROM recommendation_category_relations
				WHERE relation_type = 'complementary'
			)
	`).Scan(
		&result.EligibleProducts,
		&result.CustomersWithActiveIntent,
		&result.ActiveSearchSignals,
		&result.RelatedRelations,
		&result.ComplementaryRelations,
	)
	if err != nil {
		return RecommendationEngineDiagnostics{}, fmt.Errorf("load recommendation engine diagnostics: %w", err)
	}
	result.BehaviorWindowHours = 72
	return result, nil
}

func (s *Service) RecommendationCategoryRelations(ctx context.Context, limit int) ([]RecommendationCategoryRelation, error) {
	if limit <= 0 || limit > 50 {
		return nil, fmt.Errorf("%w: limit must be between 1 and 50", ErrInvalidLimit)
	}
	queryCtx, cancel, err := s.queryContext(ctx)
	if err != nil {
		return nil, err
	}
	defer cancel()

	rows, err := s.db.Query(queryCtx, `
		SELECT
			r.source_category_id::text,
			s.name,
			r.target_category_id::text,
			t.name,
			r.relation_type,
			r.weight
		FROM recommendation_category_relations r
		JOIN categories s ON s.id = r.source_category_id
		JOIN categories t ON t.id = r.target_category_id
		ORDER BY r.weight DESC, s.name, t.name
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("load recommendation category relations: %w", err)
	}
	defer rows.Close()

	result := make([]RecommendationCategoryRelation, 0, limit)
	for rows.Next() {
		var item RecommendationCategoryRelation
		if err := rows.Scan(
			&item.SourceCategoryID,
			&item.SourceCategoryName,
			&item.TargetCategoryID,
			&item.TargetCategoryName,
			&item.RelationType,
			&item.Weight,
		); err != nil {
			return nil, fmt.Errorf("scan recommendation category relation: %w", err)
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate recommendation category relations: %w", err)
	}
	return result, nil
}

func nextRecommendationBucket(current time.Time, granularity string) time.Time {
	if granularity == GranularityWeek {
		return current.AddDate(0, 0, 7)
	}
	return current.AddDate(0, 0, 1)
}
