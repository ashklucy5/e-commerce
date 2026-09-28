package analytics

import (
	"context"
	"fmt"
	"math"
	"time"

	"golang.org/x/sync/errgroup"
)

const (
	maxBusinessInsightSignals     = 8
	maxNonInventoryInsightSignals = 6
)

type InsightEvidence struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

type InsightAction struct {
	Type       string `json:"type"`
	ResourceID string `json:"resource_id,omitempty"`
}

type BusinessInsight struct {
	Type     string `json:"type"`
	Severity string `json:"severity"`
	Title    string `json:"title"`
	Summary  string `json:"summary"`

	Evidence []InsightEvidence `json:"evidence"`
	Action   *InsightAction    `json:"action,omitempty"`
}

type InventoryRunRateRisk struct {
	VariantID   string `json:"variant_id"`
	ProductID   string `json:"product_id"`
	SKU         string `json:"sku"`
	ProductName string `json:"product_name"`

	AvailableUnits int64 `json:"available_units"`
	UnitsSold      int64 `json:"units_sold"`

	DailyUnitsRunRate float64 `json:"daily_units_run_rate"`
	DaysOfCover       float64 `json:"days_of_cover"`
}

type BusinessInsightsResult struct {
	Signals        []BusinessInsight      `json:"signals"`
	InventoryRisks []InventoryRunRateRisk `json:"inventory_run_rate_risks"`
}

func (s *Service) BusinessInsights(
	ctx context.Context,
	query Query,
) (
	BusinessInsightsResult,
	error,
) {
	previous :=
		previousAnalyticsQuery(
			query,
		)

	var currentOverview Overview
	var previousOverview Overview

	var currentFunnel Funnel
	var previousFunnel Funnel

	var currentRecommendations RecommendationPerformance
	var previousRecommendations RecommendationPerformance

	var currentProfitLoss ProfitLoss
	var previousProfitLoss ProfitLoss

	var inventoryRisks []InventoryRunRateRisk

	group, groupCtx :=
		errgroup.WithContext(
			ctx,
		)

	group.Go(
		func() error {
			var err error

			currentOverview, err =
				s.Overview(
					groupCtx,
					query,
				)

			return err
		},
	)

	group.Go(
		func() error {
			var err error

			previousOverview, err =
				s.Overview(
					groupCtx,
					previous,
				)

			return err
		},
	)

	group.Go(
		func() error {
			var err error

			currentFunnel, err =
				s.Funnel(
					groupCtx,
					query,
				)

			return err
		},
	)

	group.Go(
		func() error {
			var err error

			previousFunnel, err =
				s.Funnel(
					groupCtx,
					previous,
				)

			return err
		},
	)

	group.Go(
		func() error {
			var err error

			currentRecommendations, err =
				s.RecommendationOverview(
					groupCtx,
					query,
				)

			return err
		},
	)

	group.Go(
		func() error {
			var err error

			previousRecommendations, err =
				s.RecommendationOverview(
					groupCtx,
					previous,
				)

			return err
		},
	)

	group.Go(
		func() error {
			var err error

			currentProfitLoss, err =
				s.ProfitLoss(
					groupCtx,
					query,
				)

			return err
		},
	)

	group.Go(
		func() error {
			var err error

			previousProfitLoss, err =
				s.ProfitLoss(
					groupCtx,
					previous,
				)

			return err
		},
	)

	group.Go(
		func() error {
			var err error

			inventoryRisks, err =
				s.InventoryRunRateRisks(
					groupCtx,
					query,
					10,
				)

			return err
		},
	)

	if err :=
		group.Wait(); err != nil {

		return BusinessInsightsResult{},
			fmt.Errorf(
				"load business insights: %w",
				err,
			)
	}

	signals :=
		make(
			[]BusinessInsight,
			0,
			maxBusinessInsightSignals,
		)

	// ---------------------------------------------------------
	// Revenue movement
	// ---------------------------------------------------------

	if previousOverview.NetCollectedRevenueAmount > 0 &&
		len(signals) <
			maxNonInventoryInsightSignals {

		change :=
			signedChangeBPS(
				currentOverview.NetCollectedRevenueAmount,
				previousOverview.NetCollectedRevenueAmount,
			)

		if abs64(
			change,
		) >= 500 {

			severity :=
				"positive"

			title :=
				"Revenue momentum improving"

			summary :=
				"Net collected revenue is above the previous comparable period."

			if change < 0 {
				severity =
					"warning"

				title =
					"Revenue has softened"

				summary =
					"Net collected revenue is below the previous comparable period."
			}

			signals =
				append(
					signals,
					BusinessInsight{
						Type: "revenue_movement",

						Severity: severity,

						Title: title,

						Summary: summary,

						Evidence: []InsightEvidence{
							{
								Label: "Current net revenue",

								Value: formatMoneyAmount(
									currentOverview.NetCollectedRevenueAmount,
									query.Currency,
								),
							},
							{
								Label: "Previous net revenue",

								Value: formatMoneyAmount(
									previousOverview.NetCollectedRevenueAmount,
									query.Currency,
								),
							},
							{
								Label: "Change",

								Value: formatSignedPercentBPS(
									change,
								),
							},
						},

						Action: &InsightAction{
							Type: "open_sales",
						},
					},
				)
		}
	}

	// ---------------------------------------------------------
	// Profitability completeness / COGS coverage
	// ---------------------------------------------------------

	if !currentProfitLoss.ProfitComplete &&
		len(signals) <
			maxNonInventoryInsightSignals {

		severity :=
			"warning"

		if currentProfitLoss.COGSCoverageBPS <
			8000 {

			severity =
				"high"
		}

		signals =
			append(
				signals,
				BusinessInsight{
					Type: "profitability_coverage",

					Severity: severity,

					Title: "Profitability coverage is incomplete",

					Summary: "Some cost-sensitive units do not have a usable order-item cost snapshot or historical SKU buying cost, so gross and net profit remain partial.",

					Evidence: []InsightEvidence{
						{
							Label: "COGS coverage",

							Value: formatPercentBPS(
								currentProfitLoss.COGSCoverageBPS,
							),
						},
						{
							Label: "Sales units missing cost",

							Value: fmt.Sprintf(
								"%d",
								currentProfitLoss.MissingCostUnits,
							),
						},
						{
							Label: "Restocked units missing cost",

							Value: fmt.Sprintf(
								"%d",
								currentProfitLoss.RestockMissingCostUnits,
							),
						},
						{
							Label: "Known gross profit",

							Value: formatMoneyAmount(
								currentProfitLoss.KnownGrossProfitAmount,
								query.Currency,
							),
						},
					},

					Action: &InsightAction{
						Type: "open_finance",
					},
				},
			)
	}

	// ---------------------------------------------------------
	// Negative profit after recorded expenses
	// ---------------------------------------------------------

	if currentProfitLoss.ProfitComplete &&
		currentProfitLoss.NetProfitAfterRecordedExpensesAmount != nil &&
		*currentProfitLoss.NetProfitAfterRecordedExpensesAmount < 0 &&
		len(signals) <
			maxNonInventoryInsightSignals {

		grossProfit :=
			int64(
				0,
			)

		if currentProfitLoss.GrossProfitAmount != nil {
			grossProfit =
				*currentProfitLoss.GrossProfitAmount
		}

		signals =
			append(
				signals,
				BusinessInsight{
					Type: "recorded_net_loss",

					Severity: "high",

					Title: "Recorded costs exceed collected revenue",

					Summary: "The period is negative after known COGS and the active operating expenses recorded in the finance ledger. This reflects recorded expenses only, not unrecorded real-world costs.",

					Evidence: []InsightEvidence{
						{
							Label: "Net collected revenue",

							Value: formatMoneyAmount(
								currentProfitLoss.NetCollectedRevenueAmount,
								query.Currency,
							),
						},
						{
							Label: "Gross profit",

							Value: formatMoneyAmount(
								grossProfit,
								query.Currency,
							),
						},
						{
							Label: "Recorded expenses",

							Value: formatMoneyAmount(
								currentProfitLoss.RecordedExpensesAmount,
								query.Currency,
							),
						},
						{
							Label: "Profit after recorded expenses",

							Value: formatMoneyAmount(
								*currentProfitLoss.NetProfitAfterRecordedExpensesAmount,
								query.Currency,
							),
						},
					},

					Action: &InsightAction{
						Type: "open_finance",
					},
				},
			)
	}

	// ---------------------------------------------------------
	// Gross-margin movement
	// ---------------------------------------------------------

	if currentProfitLoss.ProfitComplete &&
		previousProfitLoss.ProfitComplete &&
		currentProfitLoss.GrossMarginBPS != nil &&
		previousProfitLoss.GrossMarginBPS != nil &&
		len(signals) <
			maxNonInventoryInsightSignals {

		marginDelta :=
			*currentProfitLoss.GrossMarginBPS -
				*previousProfitLoss.GrossMarginBPS

		if abs64(
			marginDelta,
		) >= 300 {

			severity :=
				"positive"

			title :=
				"Gross margin improved"

			summary :=
				"Gross margin is higher than in the previous comparable period."

			if marginDelta < 0 {
				severity =
					"warning"

				title =
					"Gross margin compressed"

				summary =
					"Gross margin is lower than in the previous comparable period."
			}

			signals =
				append(
					signals,
					BusinessInsight{
						Type: "gross_margin_movement",

						Severity: severity,

						Title: title,

						Summary: summary,

						Evidence: []InsightEvidence{
							{
								Label: "Current gross margin",

								Value: formatPercentBPS(
									*currentProfitLoss.GrossMarginBPS,
								),
							},
							{
								Label: "Previous gross margin",

								Value: formatPercentBPS(
									*previousProfitLoss.GrossMarginBPS,
								),
							},
							{
								Label: "Change",

								Value: formatSignedPercentagePointBPS(
									marginDelta,
								),
							},
							{
								Label: "Current gross profit",

								Value: formatMoneyAmount(
									*currentProfitLoss.GrossProfitAmount,
									query.Currency,
								),
							},
						},

						Action: &InsightAction{
							Type: "open_finance",
						},
					},
				)
		}
	}

	// ---------------------------------------------------------
	// Recorded expense pressure
	// ---------------------------------------------------------

	if currentProfitLoss.NetCollectedRevenueAmount > 0 &&
		currentProfitLoss.RecordedExpensesAmount > 0 &&
		len(signals) <
			maxNonInventoryInsightSignals {

		currentExpenseBPS :=
			basisPoints(
				currentProfitLoss.RecordedExpensesAmount,
				currentProfitLoss.NetCollectedRevenueAmount,
			)

		previousExpenseBPS :=
			int64(
				0,
			)

		if previousProfitLoss.NetCollectedRevenueAmount > 0 {
			previousExpenseBPS =
				basisPoints(
					previousProfitLoss.RecordedExpensesAmount,
					previousProfitLoss.NetCollectedRevenueAmount,
				)
		}

		expenseDelta :=
			currentExpenseBPS -
				previousExpenseBPS

		if currentExpenseBPS >= 2500 &&
			(previousProfitLoss.NetCollectedRevenueAmount <= 0 ||
				expenseDelta >= 500) {

			signals =
				append(
					signals,
					BusinessInsight{
						Type: "recorded_expense_pressure",

						Severity: "warning",

						Title: "Recorded expense pressure is elevated",

						Summary: "Active expenses recorded in the finance ledger consume a relatively large share of net collected revenue. This ratio includes recorded expenses only.",

						Evidence: []InsightEvidence{
							{
								Label: "Recorded expenses",

								Value: formatMoneyAmount(
									currentProfitLoss.RecordedExpensesAmount,
									query.Currency,
								),
							},
							{
								Label: "Net collected revenue",

								Value: formatMoneyAmount(
									currentProfitLoss.NetCollectedRevenueAmount,
									query.Currency,
								),
							},
							{
								Label: "Recorded expense ratio",

								Value: formatPercentBPS(
									currentExpenseBPS,
								),
							},
							{
								Label: "Previous ratio",

								Value: formatPercentBPS(
									previousExpenseBPS,
								),
							},
						},

						Action: &InsightAction{
							Type: "open_finance",
						},
					},
				)
		}
	}

	// ---------------------------------------------------------
	// Foreign-currency expense exclusion
	// ---------------------------------------------------------

	if currentProfitLoss.ForeignCurrencyExpenseEntriesExcluded > 0 &&
		len(signals) <
			maxNonInventoryInsightSignals {

		signals =
			append(
				signals,
				BusinessInsight{
					Type: "finance_currency_exclusion",

					Severity: "warning",

					Title: "Some expenses are excluded from this currency view",

					Summary: "Active expense entries recorded in other currencies are excluded because Commerce Intelligence does not perform implicit FX conversion.",

					Evidence: []InsightEvidence{
						{
							Label: "Reporting currency",

							Value: query.Currency,
						},
						{
							Label: "Excluded expense entries",

							Value: fmt.Sprintf(
								"%d",
								currentProfitLoss.ForeignCurrencyExpenseEntriesExcluded,
							),
						},
					},

					Action: &InsightAction{
						Type: "open_finance",
					},
				},
			)
	}

	// ---------------------------------------------------------
	// Funnel movement
	// ---------------------------------------------------------

	funnelDelta :=
		currentFunnel.CartToCheckoutBPS -
			previousFunnel.CartToCheckoutBPS

	if abs64(
		funnelDelta,
	) >= 300 &&
		previousFunnel.CartsCreated > 0 &&
		len(signals) <
			maxNonInventoryInsightSignals {

		severity :=
			"positive"

		title :=
			"Cart-to-checkout conversion improved"

		summary :=
			"A larger share of carts are reaching checkout than in the previous period."

		if funnelDelta < 0 {
			severity =
				"warning"

			title =
				"Cart-to-checkout conversion declined"

			summary =
				"More carts are dropping before checkout than in the previous period."
		}

		signals =
			append(
				signals,
				BusinessInsight{
					Type: "funnel_change",

					Severity: severity,

					Title: title,

					Summary: summary,

					Evidence: []InsightEvidence{
						{
							Label: "Current conversion",

							Value: formatPercentBPS(
								currentFunnel.CartToCheckoutBPS,
							),
						},
						{
							Label: "Previous conversion",

							Value: formatPercentBPS(
								previousFunnel.CartToCheckoutBPS,
							),
						},
						{
							Label: "Change",

							Value: formatSignedPercentagePointBPS(
								funnelDelta,
							),
						},
					},

					Action: &InsightAction{
						Type: "open_funnel",
					},
				},
			)
	}

	// ---------------------------------------------------------
	// Refund pressure
	// ---------------------------------------------------------

	currentRefundBPS :=
		basisPoints(
			currentOverview.SuccessfulRefundAmount,
			currentOverview.GrossCollectedRevenueAmount,
		)

	previousRefundBPS :=
		basisPoints(
			previousOverview.SuccessfulRefundAmount,
			previousOverview.GrossCollectedRevenueAmount,
		)

	if currentRefundBPS >= 500 &&
		currentRefundBPS-
			previousRefundBPS >= 150 &&
		len(signals) <
			maxNonInventoryInsightSignals {

		signals =
			append(
				signals,
				BusinessInsight{
					Type: "refund_pressure",

					Severity: "warning",

					Title: "Refund pressure is elevated",

					Summary: "Refunds are consuming a larger share of collected revenue than in the previous period.",

					Evidence: []InsightEvidence{
						{
							Label: "Current refund rate",

							Value: formatPercentBPS(
								currentRefundBPS,
							),
						},
						{
							Label: "Previous refund rate",

							Value: formatPercentBPS(
								previousRefundBPS,
							),
						},
						{
							Label: "Refund amount",

							Value: formatMoneyAmount(
								currentOverview.SuccessfulRefundAmount,
								query.Currency,
							),
						},
					},

					Action: &InsightAction{
						Type: "open_finance",
					},
				},
			)
	}

	// ---------------------------------------------------------
	// Recommendation engagement
	// ---------------------------------------------------------

	if currentRecommendations.Impressions >= 100 &&
		len(signals) <
			maxNonInventoryInsightSignals {

		delta :=
			currentRecommendations.ClickThroughRateBPS -
				previousRecommendations.ClickThroughRateBPS

		if currentRecommendations.ClickThroughRateBPS <
			300 {

			signals =
				append(
					signals,
					BusinessInsight{
						Type: "recommendation_engagement",

						Severity: "warning",

						Title: "Recommendation engagement is low",

						Summary: "Recommendation impressions are receiving relatively few clicks.",

						Evidence: []InsightEvidence{
							{
								Label: "Impressions",

								Value: fmt.Sprintf(
									"%d",
									currentRecommendations.Impressions,
								),
							},
							{
								Label: "CTR",

								Value: formatPercentBPS(
									currentRecommendations.ClickThroughRateBPS,
								),
							},
							{
								Label: "Previous CTR",

								Value: formatPercentBPS(
									previousRecommendations.ClickThroughRateBPS,
								),
							},
						},

						Action: &InsightAction{
							Type: "open_recommendations",
						},
					},
				)
		} else if delta >= 150 &&
			previousRecommendations.Impressions >= 100 {

			signals =
				append(
					signals,
					BusinessInsight{
						Type: "recommendation_engagement",

						Severity: "positive",

						Title: "Recommendation engagement improved",

						Summary: "Recommendation click-through rate is higher than in the previous period.",

						Evidence: []InsightEvidence{
							{
								Label: "Current CTR",

								Value: formatPercentBPS(
									currentRecommendations.ClickThroughRateBPS,
								),
							},
							{
								Label: "Previous CTR",

								Value: formatPercentBPS(
									previousRecommendations.ClickThroughRateBPS,
								),
							},
							{
								Label: "Attributed merchandise",

								Value: formatMoneyAmount(
									currentRecommendations.AttributedMerchandiseAmount,
									query.Currency,
								),
							},
						},

						Action: &InsightAction{
							Type: "open_recommendations",
						},
					},
				)
		}
	}

	// ---------------------------------------------------------
	// Inventory run-rate pressure
	//
	// Reserve room for inventory signals even if many business
	// metrics fired above. This remains explicitly a run-rate
	// heuristic, not a demand forecast.
	// ---------------------------------------------------------

	for _, risk := range inventoryRisks {

		if risk.DaysOfCover > 14 {
			continue
		}

		severity :=
			"warning"

		if risk.DaysOfCover <= 7 {
			severity =
				"high"
		}

		signals =
			append(
				signals,
				BusinessInsight{
					Type: "inventory_run_rate",

					Severity: severity,

					Title: fmt.Sprintf(
						"Inventory pressure: %s",
						risk.ProductName,
					),

					Summary: "Available stock is low relative to the recent paid-unit sales run rate. This is a run-rate indicator, not a demand forecast.",

					Evidence: []InsightEvidence{
						{
							Label: "Available units",

							Value: fmt.Sprintf(
								"%d",
								risk.AvailableUnits,
							),
						},
						{
							Label: "Recent units sold",

							Value: fmt.Sprintf(
								"%d",
								risk.UnitsSold,
							),
						},
						{
							Label: "Days of cover",

							Value: fmt.Sprintf(
								"%.1f",
								risk.DaysOfCover,
							),
						},
					},

					Action: &InsightAction{
						Type:       "open_inventory",
						ResourceID: risk.VariantID,
					},
				},
			)

		if len(
			signals,
		) >= maxBusinessInsightSignals {

			break
		}
	}

	return BusinessInsightsResult{
			Signals: signals,

			InventoryRisks: inventoryRisks,
		},
		nil
}

func (s *Service) InventoryRunRateRisks(
	ctx context.Context,
	query Query,
	limit int,
) (
	[]InventoryRunRateRisk,
	error,
) {
	if limit <= 0 ||
		limit > 50 {

		return nil,
			fmt.Errorf(
				"%w: limit must be between 1 and 50",
				ErrInvalidLimit,
			)
	}

	queryCtx, cancel, err :=
		s.queryContext(
			ctx,
		)
	if err != nil {
		return nil, err
	}

	defer cancel()

	windowDays :=
		query.End.Sub(
			query.Start,
		).Hours() / 24

	if windowDays < 1 {
		windowDays =
			1
	}

	rows, err :=
		s.db.Query(
			queryCtx,
			`
				WITH sales AS (
					SELECT
						oi.variant_id,

						SUM(
							oi.quantity
						)::bigint
							AS units_sold

					FROM orders o

					JOIN order_items oi
						ON oi.order_id = o.id

					WHERE
						o.paid_at >= $1
						AND o.paid_at < $2
						AND o.currency = $3
						AND o.payment_status IN (
							'paid',
							'cod_collected',
							'refunded'
						)

					GROUP BY
						oi.variant_id
				)

				SELECT
					v.id::text,
					p.id::text,
					v.sku,
					p.name,

					GREATEST(
						i.quantity_on_hand -
						i.quantity_reserved,
						0
					)::bigint
						AS available_units,

					s.units_sold

				FROM sales s

				JOIN product_variants v
					ON v.id = s.variant_id

				JOIN products p
					ON p.id = v.product_id

				JOIN inventory i
					ON i.variant_id = v.id

				WHERE
					p.status = 'active'
					AND v.is_active = true
					AND s.units_sold > 0

				ORDER BY
					(
						GREATEST(
							i.quantity_on_hand -
							i.quantity_reserved,
							0
						)::numeric /
						NULLIF(
							s.units_sold,
							0
						)
					) ASC,

					s.units_sold DESC,

					v.id

				LIMIT $4
			`,
			query.Start,
			query.End,
			query.Currency,
			limit,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"load inventory run-rate risks: %w",
				err,
			)
	}

	defer rows.Close()

	result :=
		make(
			[]InventoryRunRateRisk,
			0,
			limit,
		)

	for rows.Next() {
		var item InventoryRunRateRisk

		if err :=
			rows.Scan(
				&item.VariantID,
				&item.ProductID,
				&item.SKU,
				&item.ProductName,
				&item.AvailableUnits,
				&item.UnitsSold,
			); err != nil {

			return nil,
				fmt.Errorf(
					"scan inventory run-rate risk: %w",
					err,
				)
		}

		item.DailyUnitsRunRate =
			float64(
				item.UnitsSold,
			) /
				windowDays

		if item.DailyUnitsRunRate > 0 {
			item.DaysOfCover =
				float64(
					item.AvailableUnits,
				) /
					item.DailyUnitsRunRate
		}

		item.DailyUnitsRunRate =
			math.Round(
				item.DailyUnitsRunRate*
					100,
			) /
				100

		item.DaysOfCover =
			math.Round(
				item.DaysOfCover*
					10,
			) /
				10

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
				"iterate inventory run-rate risks: %w",
				err,
			)
	}

	return result,
		nil
}

func previousAnalyticsQuery(
	query Query,
) Query {
	duration :=
		query.End.Sub(
			query.Start,
		)

	previousEnd :=
		query.Start

	previousStart :=
		previousEnd.Add(
			-duration,
		)

	previousTo :=
		previousEnd.
			In(
				businessLocation,
			).
			Add(
				-time.Nanosecond,
			)

	previousFrom :=
		previousStart.In(
			businessLocation,
		)

	return Query{
		Start: previousStart,

		End: previousEnd,

		FromDate: previousFrom.Format(
			time.DateOnly,
		),

		ToDate: previousTo.Format(
			time.DateOnly,
		),

		Granularity: query.Granularity,

		Currency: query.Currency,
	}
}

func signedChangeBPS(
	current int64,
	previous int64,
) int64 {
	if previous == 0 {
		return 0
	}

	return ((current - previous) * 10000) / previous
}

func abs64(
	value int64,
) int64 {
	if value < 0 {
		return -value
	}

	return value
}

func formatPercentBPS(
	value int64,
) string {
	return fmt.Sprintf(
		"%.1f%%",
		float64(
			value,
		)/100,
	)
}

func formatSignedPercentBPS(
	value int64,
) string {
	return fmt.Sprintf(
		"%+.1f%%",
		float64(
			value,
		)/100,
	)
}

func formatSignedPercentagePointBPS(
	value int64,
) string {
	return fmt.Sprintf(
		"%+.1f pp",
		float64(
			value,
		)/100,
	)
}

func formatMoneyAmount(
	amount int64,
	currency string,
) string {
	return fmt.Sprintf(
		"%d %s",
		amount,
		currency,
	)
}
