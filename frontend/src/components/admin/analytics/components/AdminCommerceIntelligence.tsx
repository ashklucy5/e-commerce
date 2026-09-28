"use client";

import {
  useCallback,
  useEffect,
  useMemo,
  useState,
} from "react";
import { useRouter } from "next/navigation";

import AdminPageHeader from "@/components/admin/layout/components/AdminPageHeader";
import { useAdminSession } from "@/components/admin/layout/context/AdminSessionContext";
import {
  adminFetch,
  AdminRequestError,
} from "@/lib/admin/api";
import type {
  AdminAnalyticsMeta,
  AdminAnalyticsOverview,
  AdminAnalyticsResponse,
  AdminBusinessInsights,
  AdminBusinessInsight,
  AdminCheckoutFunnel,
  AdminInventoryRunRateRisk,
  AdminProfitLoss,
  AdminProfitLossPoint,
  AdminProductProfitability,
  AdminPromotionAttribution,
  AdminRecommendationBreakdown,
  AdminRecommendationEngineResponse,
  AdminRecommendationPerformance,
  AdminRecommendationProduct,
  AdminRecommendationRelationsResponse,
  AdminRecommendationTrendPoint,
  AdminSalesPoint,
  AdminTopProduct,
} from "@/lib/admin/analytics-types";
import { formatMoney } from "@/lib/money/format";

import styles from "../css/AdminCommerceIntelligence.module.css";

type AdminCommerceIntelligenceProps = {
  portal: string;
};

type AnalyticsView =
  | "overview"
  | "sales"
  | "products"
  | "funnel"
  | "finance"
  | "recommendations";

type PeriodDays = 7 | 30 | 90;

type ChartPoint = {
  label: string;
  value: number;
};

const PERIODS: Array<{
  label: string;
  value: PeriodDays;
}> = [
  { label: "7D", value: 7 },
  { label: "30D", value: 30 },
  { label: "90D", value: 90 },
];

const VIEWS: Array<{
  id: AnalyticsView;
  label: string;
  shortLabel: string;
}> = [
  {
    id: "overview",
    label: "Overview",
    shortLabel: "Overview",
  },
  {
    id: "sales",
    label: "Sales",
    shortLabel: "Sales",
  },
  {
    id: "products",
    label: "Products",
    shortLabel: "Products",
  },
  {
    id: "funnel",
    label: "Funnel",
    shortLabel: "Funnel",
  },
  {
    id: "finance",
    label: "Finance",
    shortLabel: "Finance",
  },
  {
    id: "recommendations",
    label: "Recommendations",
    shortLabel: "Recs",
  },
];

function errorMessage(
  value: unknown,
): string {
  if (
    value instanceof
    AdminRequestError
  ) {
    return value.message;
  }

  if (
    value instanceof
    Error
  ) {
    return value.message;
  }

  return "Unable to load commerce intelligence.";
}

function businessDate(
  offsetDays = 0,
): string {
  const parts =
    new Intl.DateTimeFormat(
      "en-CA",
      {
        timeZone:
          "Asia/Dhaka",
        year: "numeric",
        month: "2-digit",
        day: "2-digit",
      },
    ).formatToParts(
      new Date(),
    );

  const year =
    Number(
      parts.find(
        (part) =>
          part.type ===
          "year",
      )?.value,
    );

  const month =
    Number(
      parts.find(
        (part) =>
          part.type ===
          "month",
      )?.value,
    );

  const day =
    Number(
      parts.find(
        (part) =>
          part.type ===
          "day",
      )?.value,
    );

  const date =
    new Date(
      Date.UTC(
        year,
        month - 1,
        day,
      ),
    );

  date.setUTCDate(
    date.getUTCDate() +
      offsetDays,
  );

  return date
    .toISOString()
    .slice(
      0,
      10,
    );
}

function analyticsQuery(
  period: PeriodDays,
): string {
  const params =
    new URLSearchParams(
      {
        from: businessDate(
          -(
            period -
            1
          ),
        ),
        to: businessDate(),
        granularity:
          period === 90
            ? "week"
            : "day",
      },
    );

  return params.toString();
}

function formatCount(
  value: number,
): string {
  return new Intl.NumberFormat(
    "en",
    {
      notation:
        Math.abs(
          value,
        ) >=
        10_000
          ? "compact"
          : "standard",
      maximumFractionDigits: 1,
    },
  ).format(
    value,
  );
}

function formatPercentBPS(
  value:
    | number
    | null
    | undefined,
): string {
  if (
    typeof value !==
    "number"
  ) {
    return "—";
  }

  return `${(
    value /
    100
  ).toFixed(1)}%`;
}

function formatDecimal(
  value: number,
  digits = 1,
): string {
  return new Intl.NumberFormat(
    "en",
    {
      minimumFractionDigits:
        digits,
      maximumFractionDigits:
        digits,
    },
  ).format(
    value,
  );
}

function titleCase(
  value: string,
): string {
  return value
    .replaceAll(
      "_",
      " ",
    )
    .replaceAll(
      "-",
      " ",
    )
    .replace(
      /\b\w/g,
      (
        character,
      ) =>
        character.toUpperCase(),
    );
}

function formatShortDate(
  value: string,
): string {
  const date =
    new Date(
      `${value}T00:00:00Z`,
    );

  if (
    Number.isNaN(
      date.getTime(),
    )
  ) {
    return value;
  }

  return new Intl.DateTimeFormat(
    "en",
    {
      month: "short",
      day: "numeric",
      timeZone: "UTC",
    },
  ).format(
    date,
  );
}

function formatDateTime(
  value?: string,
): string {
  if (!value) {
    return "—";
  }

  const date =
    new Date(
      value,
    );

  if (
    Number.isNaN(
      date.getTime(),
    )
  ) {
    return "—";
  }

  return new Intl.DateTimeFormat(
    "en",
    {
      month: "short",
      day: "numeric",
      hour: "numeric",
      minute: "2-digit",
    },
  ).format(
    date,
  );
}

function chartPath(
  points: ChartPoint[],
): string {
  if (
    points.length ===
    0
  ) {
    return "";
  }

  const values =
    points.map(
      (point) =>
        point.value,
    );

  const maxValue =
    Math.max(
      ...values,
      1,
    );

  const minValue =
    Math.min(
      ...values,
      0,
    );

  const span =
    Math.max(
      maxValue -
        minValue,
      1,
    );

  const width = 720;
  const height = 230;
  const xPadding = 14;
  const yPadding = 18;

  const usableWidth =
    width -
    xPadding * 2;

  const usableHeight =
    height -
    yPadding * 2;

  return points
    .map(
      (
        point,
        index,
      ) => {
        const x =
          xPadding +
          (
            points.length ===
            1
              ? usableWidth /
                2
              : (
                  index /
                  (
                    points.length -
                    1
                  )
                ) *
                usableWidth
          );

        const normalized =
          (
            point.value -
            minValue
          ) /
          span;

        const y =
          yPadding +
          (
            1 -
            normalized
          ) *
            usableHeight;

        return `${
          index === 0
            ? "M"
            : "L"
        }${x.toFixed(
          2,
        )},${y.toFixed(
          2,
        )}`;
      },
    )
    .join(" ");
}

function MetricCard({
  label,
  value,
  detail,
  tone = "neutral",
}: {
  label: string;
  value: string;
  detail?: string;
  tone?:
    | "neutral"
    | "positive"
    | "blue"
    | "violet"
    | "warning";
}) {
  return (
    <article
      className={[
        styles.metricCard,
        tone ===
        "positive"
          ? styles.metricPositive
          : "",
        tone ===
        "blue"
          ? styles.metricBlue
          : "",
        tone ===
        "violet"
          ? styles.metricViolet
          : "",
        tone ===
        "warning"
          ? styles.metricWarning
          : "",
      ]
        .filter(
          Boolean,
        )
        .join(
          " ",
        )}
    >
      <span
        className={
          styles.metricLabel
        }
      >
        {label}
      </span>

      <strong
        className={
          styles.metricValue
        }
      >
        {value}
      </strong>

      {detail ? (
        <span
          className={
            styles.metricDetail
          }
        >
          {detail}
        </span>
      ) : null}
    </article>
  );
}

function SectionHeading({
  eyebrow,
  title,
  description,
}: {
  eyebrow: string;
  title: string;
  description?: string;
}) {
  return (
    <div
      className={
        styles.sectionHeading
      }
    >
      <div>
        <p
          className={
            styles.sectionEyebrow
          }
        >
          {eyebrow}
        </p>

        <h2>
          {title}
        </h2>

        {description ? (
          <p
            className={
              styles.sectionDescription
            }
          >
            {description}
          </p>
        ) : null}
      </div>
    </div>
  );
}

function TrendChart({
  points,
  valueLabel,
  emptyLabel,
  formatter,
}: {
  points: ChartPoint[];
  valueLabel: string;
  emptyLabel: string;
  formatter: (
    value: number,
  ) => string;
}) {
  const path =
    useMemo(
      () =>
        chartPath(
          points,
        ),
      [
        points,
      ],
    );

  const values =
    points.map(
      (point) =>
        point.value,
    );

  const total =
    values.reduce(
      (
        sum,
        value,
      ) =>
        sum +
        value,
      0,
    );

  const peak =
    values.length >
    0
      ? Math.max(
          ...values,
        )
      : 0;

  const peakIndex =
    values.indexOf(
      peak,
    );

  if (
    points.length ===
    0
  ) {
    return (
      <div
        className={
          styles.emptyState
        }
      >
        {emptyLabel}
      </div>
    );
  }

  return (
    <div
      className={
        styles.trendChart
      }
    >
      <div
        className={
          styles.chartSummary
        }
      >
        <div>
          <span>
            Total{" "}
            {
              valueLabel
            }
          </span>

          <strong>
            {formatter(
              total,
            )}
          </strong>
        </div>

        <div>
          <span>
            Peak bucket
          </span>

          <strong>
            {formatter(
              peak,
            )}
          </strong>

          <small>
            {peakIndex >=
            0
              ? points[
                  peakIndex
                ]?.label
              : "—"}
          </small>
        </div>
      </div>

      <div
        className={
          styles.chartFrame
        }
      >
        <svg
          className={
            styles.chartSvg
          }
          viewBox="0 0 720 230"
          preserveAspectRatio="none"
          role="img"
          aria-label={`${valueLabel} trend`}
        >
          <line
            x1="14"
            y1="18"
            x2="706"
            y2="18"
          />

          <line
            x1="14"
            y1="115"
            x2="706"
            y2="115"
          />

          <line
            x1="14"
            y1="212"
            x2="706"
            y2="212"
          />

          <path
            d={
              path
            }
          />
        </svg>
      </div>

      <div
        className={
          styles.chartAxis
        }
      >
        <span>
          {
            points[
              0
            ]?.label
          }
        </span>

        <span>
          {
            points[
              Math.floor(
                (
                  points.length -
                  1
                ) /
                  2,
              )
            ]?.label
          }
        </span>

        <span>
          {
            points.at(
              -1,
            )?.label
          }
        </span>
      </div>
    </div>
  );
}

function PermissionState() {
  return (
    <section
      className={
        styles.permissionState
      }
    >
      <div
        className={
          styles.permissionIcon
        }
        aria-hidden="true"
      >
        ◇
      </div>

      <div>
        <h2>
          Analytics access
          is restricted
        </h2>

        <p>
          Your staff
          account does not
          include the{" "}
          <code>
            admin.analytics.read
          </code>{" "}
          permission.
        </p>
      </div>
    </section>
  );
}

function LoadingState() {
  return (
    <div
      className={
        styles.loadingGrid
      }
      aria-label="Loading analytics"
    >
      {Array.from({
        length: 6,
      }).map(
        (
          _,
          index,
        ) => (
          <div
            key={
              index
            }
            className={
              styles.loadingCard
            }
          />
        ),
      )}
    </div>
  );
}

function EmptyRow({
  children,
}: {
  children: string;
}) {
  return (
    <div
      className={
        styles.emptyState
      }
    >
      {children}
    </div>
  );
}

function insightTone(
  severity: string,
): string {
  switch (
    severity
  ) {
    case "positive":
      return styles.insightPositive;

    case "high":
      return styles.insightHigh;

    case "warning":
      return styles.insightWarning;

    default:
      return styles.insightNeutral;
  }
}

function relationTone(
  type: string,
): string {
  return type ===
    "complementary"
    ? styles.relationComplementary
    : styles.relationRelated;
}

export default function AdminCommerceIntelligence({
  portal,
}: AdminCommerceIntelligenceProps) {
  const router =
    useRouter();

  const principal =
    useAdminSession();

  const isSuperAdmin =
    principal.staff.roles.includes(
      "admin_superuser",
    );

  const canRead =
    isSuperAdmin ||
    principal.staff.permissions.includes(
      "admin.analytics.read",
    );

  const [
    activeView,
    setActiveView,
  ] =
    useState<AnalyticsView>(
      "overview",
    );

  const [
    period,
    setPeriod,
  ] =
    useState<PeriodDays>(
      30,
    );

  const [
    refreshNonce,
    setRefreshNonce,
  ] =
    useState(
      0,
    );

  const [
    loading,
    setLoading,
  ] =
    useState(
      false,
    );

  const [
    error,
    setError,
  ] =
    useState(
      "",
    );

  const [
    meta,
    setMeta,
  ] =
    useState<AdminAnalyticsMeta | null>(
      null,
    );

  const [
    overview,
    setOverview,
  ] =
    useState<AdminAnalyticsOverview | null>(
      null,
    );

  const [
    insights,
    setInsights,
  ] =
    useState<AdminBusinessInsights | null>(
      null,
    );

  const [
    sales,
    setSales,
  ] =
    useState<AdminSalesPoint[]>(
      [],
    );

  const [
    funnel,
    setFunnel,
  ] =
    useState<AdminCheckoutFunnel | null>(
      null,
    );

  const [
    topProducts,
    setTopProducts,
  ] =
    useState<AdminTopProduct[]>(
      [],
    );

  const [
    promotions,
    setPromotions,
  ] =
    useState<AdminPromotionAttribution[]>(
      [],
    );

  const [
    profitLoss,
    setProfitLoss,
  ] =
    useState<AdminProfitLoss | null>(
      null,
    );

  const [
    profitTrend,
    setProfitTrend,
  ] =
    useState<AdminProfitLossPoint[]>(
      [],
    );

  const [
    productProfitability,
    setProductProfitability,
  ] =
    useState<AdminProductProfitability[]>(
      [],
    );

  const [
    recommendationOverview,
    setRecommendationOverview,
  ] =
    useState<AdminRecommendationPerformance | null>(
      null,
    );

  const [
    recommendationTrend,
    setRecommendationTrend,
  ] =
    useState<AdminRecommendationTrendPoint[]>(
      [],
    );

  const [
    recommendationPlacements,
    setRecommendationPlacements,
  ] =
    useState<AdminRecommendationBreakdown[]>(
      [],
    );

  const [
    recommendationStrategies,
    setRecommendationStrategies,
  ] =
    useState<AdminRecommendationBreakdown[]>(
      [],
    );

  const [
    recommendationProducts,
    setRecommendationProducts,
  ] =
    useState<AdminRecommendationProduct[]>(
      [],
    );

  const [
    recommendationEngine,
    setRecommendationEngine,
  ] =
    useState<AdminRecommendationEngineResponse["data"] | null>(
      null,
    );

  const [
    recommendationRelations,
    setRecommendationRelations,
  ] =
    useState<AdminRecommendationRelationsResponse["data"]>(
      [],
    );

  useEffect(
    () => {
      if (
        !canRead
      ) {
        return;
      }

      const controller =
        new AbortController();

      const query =
        analyticsQuery(
          period,
        );

      const signal =
        controller.signal;

      async function loadActiveView() {
        setLoading(
          true,
        );

        setError(
          "",
        );

        try {
          if (
            activeView ===
            "overview"
          ) {
            const [
              overviewResponse,
              insightResponse,
              financeResponse,
            ] =
              await Promise.all(
                [
                  adminFetch<
                    AdminAnalyticsResponse<AdminAnalyticsOverview>
                  >(
                    `/analytics/overview?${query}`,
                    {
                      signal,
                    },
                  ),

                  adminFetch<
                    AdminAnalyticsResponse<AdminBusinessInsights>
                  >(
                    `/analytics/insights?${query}`,
                    {
                      signal,
                    },
                  ),

                  adminFetch<
                    AdminAnalyticsResponse<AdminProfitLoss>
                  >(
                    `/finance/profit-loss?${query}`,
                    {
                      signal,
                    },
                  ),
                ],
              );

            setOverview(
              overviewResponse.data,
            );

            setInsights(
              insightResponse.data,
            );

            setProfitLoss(
              financeResponse.data,
            );

            setMeta(
              overviewResponse.meta,
            );
          }

          if (
            activeView ===
            "sales"
          ) {
            const [
              salesResponse,
              promotionResponse,
            ] =
              await Promise.all(
                [
                  adminFetch<
                    AdminAnalyticsResponse<AdminSalesPoint[]>
                  >(
                    `/analytics/sales?${query}`,
                    {
                      signal,
                    },
                  ),

                  adminFetch<
                    AdminAnalyticsResponse<AdminPromotionAttribution[]>
                  >(
                    `/analytics/promotions?${query}&limit=10`,
                    {
                      signal,
                    },
                  ),
                ],
              );

            setSales(
              salesResponse.data,
            );

            setPromotions(
              promotionResponse.data,
            );

            setMeta(
              salesResponse.meta,
            );
          }

          if (
            activeView ===
            "products"
          ) {
            const [
              productsResponse,
              profitabilityResponse,
            ] =
              await Promise.all(
                [
                  adminFetch<
                    AdminAnalyticsResponse<AdminTopProduct[]>
                  >(
                    `/analytics/products/top?${query}&limit=12`,
                    {
                      signal,
                    },
                  ),

                  adminFetch<
                    AdminAnalyticsResponse<AdminProductProfitability[]>
                  >(
                    `/finance/products/profitability?${query}&limit=12`,
                    {
                      signal,
                    },
                  ),
                ],
              );

            setTopProducts(
              productsResponse.data,
            );

            setProductProfitability(
              profitabilityResponse.data,
            );

            setMeta(
              productsResponse.meta,
            );
          }

          if (
            activeView ===
            "funnel"
          ) {
            const response =
              await adminFetch<
                AdminAnalyticsResponse<AdminCheckoutFunnel>
              >(
                `/analytics/funnel?${query}`,
                {
                  signal,
                },
              );

            setFunnel(
              response.data,
            );

            setMeta(
              response.meta,
            );
          }

          if (
            activeView ===
            "finance"
          ) {
            const [
              financeResponse,
              trendResponse,
            ] =
              await Promise.all(
                [
                  adminFetch<
                    AdminAnalyticsResponse<AdminProfitLoss>
                  >(
                    `/finance/profit-loss?${query}`,
                    {
                      signal,
                    },
                  ),

                  adminFetch<
                    AdminAnalyticsResponse<AdminProfitLossPoint[]>
                  >(
                    `/finance/profit-loss/trend?${query}`,
                    {
                      signal,
                    },
                  ),
                ],
              );

            setProfitLoss(
              financeResponse.data,
            );

            setProfitTrend(
              trendResponse.data,
            );

            setMeta(
              financeResponse.meta,
            );
          }

          if (
            activeView ===
            "recommendations"
          ) {
            const [
              overviewResponse,
              trendResponse,
              placementResponse,
              strategyResponse,
              productResponse,
              engineResponse,
              relationResponse,
            ] =
              await Promise.all(
                [
                  adminFetch<
                    AdminAnalyticsResponse<AdminRecommendationPerformance>
                  >(
                    `/analytics/recommendations/overview?${query}`,
                    {
                      signal,
                    },
                  ),

                  adminFetch<
                    AdminAnalyticsResponse<AdminRecommendationTrendPoint[]>
                  >(
                    `/analytics/recommendations/trend?${query}`,
                    {
                      signal,
                    },
                  ),

                  adminFetch<
                    AdminAnalyticsResponse<AdminRecommendationBreakdown[]>
                  >(
                    `/analytics/recommendations/placements?${query}`,
                    {
                      signal,
                    },
                  ),

                  adminFetch<
                    AdminAnalyticsResponse<AdminRecommendationBreakdown[]>
                  >(
                    `/analytics/recommendations/strategies?${query}`,
                    {
                      signal,
                    },
                  ),

                  adminFetch<
                    AdminAnalyticsResponse<AdminRecommendationProduct[]>
                  >(
                    `/analytics/recommendations/products?${query}&limit=12`,
                    {
                      signal,
                    },
                  ),

                  adminFetch<AdminRecommendationEngineResponse>(
                    "/analytics/recommendations/engine",
                    {
                      signal,
                    },
                  ),

                  adminFetch<AdminRecommendationRelationsResponse>(
                    "/analytics/recommendations/relations?limit=20",
                    {
                      signal,
                    },
                  ),
                ],
              );

            setRecommendationOverview(
              overviewResponse.data,
            );

            setRecommendationTrend(
              trendResponse.data,
            );

            setRecommendationPlacements(
              placementResponse.data,
            );

            setRecommendationStrategies(
              strategyResponse.data,
            );

            setRecommendationProducts(
              productResponse.data,
            );

            setRecommendationEngine(
              engineResponse.data,
            );

            setRecommendationRelations(
              relationResponse.data,
            );

            setMeta(
              overviewResponse.meta,
            );
          }
        } catch (
          value
        ) {
          if (
            controller.signal.aborted ||
            (
              value instanceof
                Error &&
              value.name ===
                "AbortError"
            )
          ) {
            return;
          }

          setError(
            errorMessage(
              value,
            ),
          );
        } finally {
          if (
            !controller.signal.aborted
          ) {
            setLoading(
              false,
            );
          }
        }
      }

      void loadActiveView();

      return () =>
        controller.abort();
    },
    [
      activeView,
      canRead,
      period,
      refreshNonce,
    ],
  );

  const rangeLabel =
    meta
      ? `${meta.from} → ${meta.to}`
      : `${businessDate(
          -(
            period -
            1
          ),
        )} → ${businessDate()}`;

  const currentCurrency =
    meta?.currency ??
    overview?.currency ??
    profitLoss?.currency ??
    recommendationOverview?.currency ??
    "BDT";

  const salesChartPoints =
    useMemo<ChartPoint[]>(
      () =>
        sales.map(
          (
            point,
          ) => ({
            label:
              formatShortDate(
                point.bucket_start,
              ),

            value:
              point.net_collected_revenue_amount,
          }),
        ),
      [
        sales,
      ],
    );

  const financeChartPoints =
    useMemo<ChartPoint[]>(
      () =>
        profitTrend.map(
          (
            point,
          ) => ({
            label:
              formatShortDate(
                point.bucket_start,
              ),

            value:
              point.net_profit_after_recorded_expenses_amount ??
              point.known_gross_profit_amount,
          }),
        ),
      [
        profitTrend,
      ],
    );

  const recommendationChartPoints =
    useMemo<ChartPoint[]>(
      () =>
        recommendationTrend.map(
          (
            point,
          ) => ({
            label:
              formatShortDate(
                point.bucket_start,
              ),

            value:
              point.attributed_merchandise_amount,
          }),
        ),
      [
        recommendationTrend,
      ],
    );

  const handleInsightAction =
    useCallback(
      (
        insight:
          AdminBusinessInsight,
      ) => {
        switch (
          insight.action
            ?.type
        ) {
          case "open_sales":
            setActiveView(
              "sales",
            );
            return;

          case "open_funnel":
            setActiveView(
              "funnel",
            );
            return;

          case "open_finance":
            setActiveView(
              "finance",
            );
            return;

          case "open_recommendations":
            setActiveView(
              "recommendations",
            );
            return;

          case "open_inventory": {
            const resourceID =
              insight.action
                .resource_id;

            const suffix =
              resourceID
                ? `?variant_id=${encodeURIComponent(
                    resourceID,
                  )}`
                : "";

            router.push(
              `/${portal}/inventory${suffix}`,
            );

            return;
          }
        }
      },
      [
        portal,
        router,
      ],
    );

  function renderOverview() {
    if (
      !overview ||
      !profitLoss ||
      !insights
    ) {
      return loading ? (
        <LoadingState />
      ) : null;
    }

    return (
      <div
        className={
          styles.viewStack
        }
      >
        <section
          className={
            styles.section
          }
        >
          <SectionHeading
            eyebrow="Business pulse"
            title="What the commerce engine is doing now"
            description="Collected revenue, order throughput, customer growth, refunds, and profitability for the selected business period."
          />

          <div
            className={
              styles.metricGrid
            }
          >
            <MetricCard
              label="Net collected revenue"
              value={formatMoney(
                overview.net_collected_revenue_amount,
                overview.currency,
              )}
              detail={`${formatCount(
                overview.collected_orders,
              )} collected orders`}
              tone="positive"
            />

            <MetricCard
              label="Average order value"
              value={formatMoney(
                overview.average_order_value_amount,
                overview.currency,
              )}
              detail={`${formatCount(
                overview.units_sold,
              )} units sold`}
              tone="blue"
            />

            <MetricCard
              label="Orders created"
              value={formatCount(
                overview.orders_created,
              )}
              detail={`${formatCount(
                overview.new_customers,
              )} new customers`}
            />

            <MetricCard
              label="Successful refunds"
              value={formatMoney(
                overview.successful_refund_amount,
                overview.currency,
              )}
              detail={`${formatMoney(
                overview.discount_amount,
                overview.currency,
              )} discounts`}
              tone={
                overview.successful_refund_amount >
                0
                  ? "warning"
                  : "neutral"
              }
            />

            <MetricCard
              label={
                profitLoss.profit_complete
                  ? "Gross profit"
                  : "Known gross profit"
              }
              value={formatMoney(
                profitLoss.gross_profit_amount ??
                  profitLoss.known_gross_profit_amount,
                profitLoss.currency,
              )}
              detail={
                profitLoss.profit_complete
                  ? `Margin ${formatPercentBPS(
                      profitLoss.gross_margin_bps,
                    )}`
                  : `COGS coverage ${formatPercentBPS(
                      profitLoss.cogs_coverage_bps,
                    )}`
              }
              tone="violet"
            />

            <MetricCard
              label="Recorded expenses"
              value={formatMoney(
                profitLoss.recorded_expenses_amount,
                profitLoss.currency,
              )}
              detail={`${formatCount(
                profitLoss.recorded_expense_entries,
              )} entries`}
            />
          </div>
        </section>

        <section
          className={
            styles.section
          }
        >
          <SectionHeading
            eyebrow="Deterministic insights"
            title="Signals worth attention"
            description="These are rule-based business signals derived from the current and previous comparable periods; they are not ML predictions."
          />

          {insights.signals
            .length >
          0 ? (
            <div
              className={
                styles.insightGrid
              }
            >
              {insights.signals.map(
                (
                  insight,
                  index,
                ) => (
                  <article
                    key={`${insight.type}-${index}`}
                    className={`${styles.insightCard} ${insightTone(
                      insight.severity,
                    )}`}
                  >
                    <div
                      className={
                        styles.insightTopline
                      }
                    >
                      <span>
                        {titleCase(
                          insight.type,
                        )}
                      </span>

                      <span>
                        {titleCase(
                          insight.severity,
                        )}
                      </span>
                    </div>

                    <h3>
                      {
                        insight.title
                      }
                    </h3>

                    <p>
                      {
                        insight.summary
                      }
                    </p>

                    <dl
                      className={
                        styles.evidenceGrid
                      }
                    >
                      {insight.evidence.map(
                        (
                          evidence,
                        ) => (
                          <div
                            key={`${evidence.label}-${evidence.value}`}
                          >
                            <dt>
                              {
                                evidence.label
                              }
                            </dt>

                            <dd>
                              {
                                evidence.value
                              }
                            </dd>
                          </div>
                        ),
                      )}
                    </dl>

                    {insight.action ? (
                      <button
                        type="button"
                        className={
                          styles.textAction
                        }
                        onClick={() =>
                          handleInsightAction(
                            insight,
                          )
                        }
                      >
                        Open
                        relevant
                        workspace

                        <span
                          aria-hidden="true"
                        >
                          →
                        </span>
                      </button>
                    ) : null}
                  </article>
                ),
              )}
            </div>
          ) : (
            <EmptyRow>
              No threshold-based business signals were triggered for this period.
            </EmptyRow>
          )}
        </section>

        <section
          className={
            styles.section
          }
        >
          <SectionHeading
            eyebrow="Inventory run rate"
            title="Stock pressure from recent paid-unit velocity"
            description="Days of cover is a run-rate indicator using recent paid-unit sales and available stock. It is not a demand forecast."
          />

          {insights
            .inventory_run_rate_risks
            .length >
          0 ? (
            <div
              className={
                styles.riskList
              }
            >
              {insights.inventory_run_rate_risks.map(
                (
                  risk,
                ) => (
                  <InventoryRiskRow
                    key={
                      risk.variant_id
                    }
                    risk={
                      risk
                    }
                  />
                ),
              )}
            </div>
          ) : (
            <EmptyRow>
              No inventory run-rate risks were returned for this period.
            </EmptyRow>
          )}
        </section>
      </div>
    );
  }

  function renderSales() {
    return (
      <div
        className={
          styles.viewStack
        }
      >
        <section
          className={
            styles.section
          }
        >
          <SectionHeading
            eyebrow="Sales movement"
            title="Collected revenue over time"
            description="Net collected revenue is gross collected revenue less successful refunds in the selected period."
          />

          <div
            className={
              styles.chartPanel
            }
          >
            <TrendChart
              points={
                salesChartPoints
              }
              valueLabel="net collected revenue"
              emptyLabel="No sales trend points were returned for this period."
              formatter={(
                value,
              ) =>
                formatMoney(
                  value,
                  currentCurrency,
                )
              }
            />
          </div>

          {sales.length >
          0 ? (
            <div
              className={
                styles.metricGridCompact
              }
            >
              <MetricCard
                label="Created orders"
                value={formatCount(
                  sales.reduce(
                    (
                      sum,
                      point,
                    ) =>
                      sum +
                      point.orders_created,
                    0,
                  ),
                )}
              />

              <MetricCard
                label="Collected orders"
                value={formatCount(
                  sales.reduce(
                    (
                      sum,
                      point,
                    ) =>
                      sum +
                      point.collected_orders,
                    0,
                  ),
                )}
                tone="positive"
              />

              <MetricCard
                label="New customers"
                value={formatCount(
                  sales.reduce(
                    (
                      sum,
                      point,
                    ) =>
                      sum +
                      point.new_customers,
                    0,
                  ),
                )}
                tone="blue"
              />

              <MetricCard
                label="Refunded"
                value={formatMoney(
                  sales.reduce(
                    (
                      sum,
                      point,
                    ) =>
                      sum +
                      point.successful_refund_amount,
                    0,
                  ),
                  currentCurrency,
                )}
                tone="warning"
              />
            </div>
          ) : null}
        </section>

        <section
          className={
            styles.section
          }
        >
          <SectionHeading
            eyebrow="Promotion attribution"
            title="Promotions driving collected orders"
            description="Ordered by gross collected revenue attributed to orders carrying a promotion."
          />

          {promotions.length >
          0 ? (
            <div
              className={
                styles.dataGrid
              }
            >
              <div
                className={`${styles.dataRow} ${styles.dataHeader}`}
              >
                <span>
                  Promotion
                </span>

                <span>
                  Orders
                </span>

                <span>
                  Units
                </span>

                <span>
                  Revenue
                </span>

                <span>
                  Discount
                </span>

                <span>
                  Avg order
                </span>
              </div>

              {promotions.map(
                (
                  promotion,
                ) => (
                  <div
                    key={
                      promotion.promotion_id
                    }
                    className={
                      styles.dataRow
                    }
                  >
                    <span
                      data-label="Promotion"
                      className={
                        styles.primaryCell
                      }
                    >
                      <strong>
                        {
                          promotion.name
                        }
                      </strong>

                      <small>
                        {promotion.code ||
                          "No code"}
                      </small>
                    </span>

                    <span
                      data-label="Orders"
                    >
                      {formatCount(
                        promotion.collected_orders,
                      )}
                    </span>

                    <span
                      data-label="Units"
                    >
                      {formatCount(
                        promotion.units_sold,
                      )}
                    </span>

                    <span
                      data-label="Revenue"
                    >
                      {formatMoney(
                        promotion.gross_collected_revenue_amount,
                        promotion.currency,
                      )}
                    </span>

                    <span
                      data-label="Discount"
                    >
                      {formatMoney(
                        promotion.discount_amount,
                        promotion.currency,
                      )}
                    </span>

                    <span
                      data-label="Avg order"
                    >
                      {formatMoney(
                        promotion.average_gross_order_value_amount,
                        promotion.currency,
                      )}
                    </span>
                  </div>
                ),
              )}
            </div>
          ) : (
            <EmptyRow>
              No promotion attribution was recorded in this period.
            </EmptyRow>
          )}
        </section>
      </div>
    );
  }

  function renderProducts() {
    return (
      <div
        className={
          styles.viewStack
        }
      >
        <section
          className={
            styles.section
          }
        >
          <SectionHeading
            eyebrow="Merchandise performance"
            title="Top collected products"
            description="Products are ranked by collected merchandise value from paid or collected orders."
          />

          {topProducts.length >
          0 ? (
            <div
              className={
                styles.dataGrid
              }
            >
              <div
                className={`${styles.dataRow} ${styles.dataHeader} ${styles.productGrid}`}
              >
                <span>
                  Product
                </span>

                <span>
                  SKU
                </span>

                <span>
                  Orders
                </span>

                <span>
                  Units
                </span>

                <span>
                  Merchandise
                </span>
              </div>

              {topProducts.map(
                (
                  product,
                ) => (
                  <div
                    key={
                      product.variant_id
                    }
                    className={`${styles.dataRow} ${styles.productGrid}`}
                  >
                    <span
                      data-label="Product"
                      className={
                        styles.primaryCell
                      }
                    >
                      <strong>
                        {
                          product.product_name
                        }
                      </strong>

                      <small>
                        {
                          product.product_id
                        }
                      </small>
                    </span>

                    <span
                      data-label="SKU"
                    >
                      {
                        product.sku
                      }
                    </span>

                    <span
                      data-label="Orders"
                    >
                      {formatCount(
                        product.collected_orders,
                      )}
                    </span>

                    <span
                      data-label="Units"
                    >
                      {formatCount(
                        product.units_sold,
                      )}
                    </span>

                    <span
                      data-label="Merchandise"
                    >
                      {formatMoney(
                        product.gross_collected_merchandise_amount,
                        product.currency,
                      )}
                    </span>
                  </div>
                ),
              )}
            </div>
          ) : (
            <EmptyRow>
              No collected product sales were returned for this period.
            </EmptyRow>
          )}
        </section>

        <section
          className={
            styles.section
          }
        >
          <SectionHeading
            eyebrow="Profitability coverage"
            title="Product margin visibility"
            description="Profit values are complete only when every sold unit in the row has cost coverage."
          />

          {productProfitability.length >
          0 ? (
            <div
              className={
                styles.dataGrid
              }
            >
              <div
                className={`${styles.dataRow} ${styles.dataHeader} ${styles.profitabilityGrid}`}
              >
                <span>
                  Product
                </span>

                <span>
                  Net merch.
                </span>

                <span>
                  COGS cover
                </span>

                <span>
                  Gross profit
                </span>

                <span>
                  Margin
                </span>

                <span>
                  State
                </span>
              </div>

              {productProfitability.map(
                (
                  product,
                ) => (
                  <div
                    key={
                      product.variant_id
                    }
                    className={`${styles.dataRow} ${styles.profitabilityGrid}`}
                  >
                    <span
                      data-label="Product"
                      className={
                        styles.primaryCell
                      }
                    >
                      <strong>
                        {
                          product.product_name
                        }
                      </strong>

                      <small>
                        {
                          product.sku
                        }
                      </small>
                    </span>

                    <span
                      data-label="Net merch."
                    >
                      {formatMoney(
                        product.net_merchandise_revenue_amount,
                        product.currency,
                      )}
                    </span>

                    <span
                      data-label="COGS cover"
                    >
                      {formatPercentBPS(
                        product.cogs_coverage_bps,
                      )}
                    </span>

                    <span
                      data-label="Gross profit"
                    >
                      {product.gross_profit_before_refunds_amount ===
                      null
                        ? "—"
                        : formatMoney(
                            product.gross_profit_before_refunds_amount,
                            product.currency,
                          )}
                    </span>

                    <span
                      data-label="Margin"
                    >
                      {formatPercentBPS(
                        product.gross_margin_before_refunds_bps,
                      )}
                    </span>

                    <span
                      data-label="State"
                    >
                      <span
                        className={
                          product.profit_complete
                            ? styles.stateComplete
                            : styles.statePartial
                        }
                      >
                        {product.profit_complete
                          ? "Complete"
                          : "Partial"}
                      </span>
                    </span>
                  </div>
                ),
              )}
            </div>
          ) : (
            <EmptyRow>
              No product profitability rows were returned.
            </EmptyRow>
          )}
        </section>
      </div>
    );
  }

  function renderFunnel() {
    if (
      !funnel
    ) {
      return loading ? (
        <LoadingState />
      ) : null;
    }

    const stages = [
      {
        label:
          "Carts created",
        value:
          funnel.carts_created,
        rate: 10000,
      },
      {
        label:
          "Checkout started",
        value:
          funnel.checkout_started_carts,
        rate:
          funnel.cart_to_checkout_bps,
      },
      {
        label:
          "Orders placed",
        value:
          funnel.ordered_carts,
        rate:
          funnel.carts_created >
          0
            ? Math.round(
                (
                  funnel.ordered_carts /
                  funnel.carts_created
                ) *
                  10000,
              )
            : 0,
      },
      {
        label:
          "Payment collected",
        value:
          funnel.collected_carts,
        rate:
          funnel.cart_to_collected_bps,
      },
    ];

    return (
      <div
        className={
          styles.viewStack
        }
      >
        <section
          className={
            styles.section
          }
        >
          <SectionHeading
            eyebrow="Checkout journey"
            title="From cart creation to collected payment"
            description="The funnel follows carts created in the selected period through checkout, order placement, and collected payment."
          />

          <div
            className={
              styles.funnelPanel
            }
          >
            {stages.map(
              (
                stage,
                index,
              ) => (
                <div
                  key={
                    stage.label
                  }
                  className={
                    styles.funnelStage
                  }
                >
                  <div
                    className={
                      styles.funnelCopy
                    }
                  >
                    <span>
                      {
                        stage.label
                      }
                    </span>

                    <strong>
                      {formatCount(
                        stage.value,
                      )}
                    </strong>

                    <small>
                      {index ===
                      0
                        ? "Cohort baseline"
                        : `${formatPercentBPS(
                            stage.rate,
                          )} of carts`}
                    </small>
                  </div>

                  <div
                    className={
                      styles.funnelTrack
                    }
                    aria-hidden="true"
                  >
                    <span
                      style={{
                        width: `${Math.max(
                          2,
                          Math.min(
                            100,
                            stage.rate /
                              100,
                          ),
                        )}%`,
                      }}
                    />
                  </div>
                </div>
              ),
            )}
          </div>
        </section>

        <section
          className={
            styles.section
          }
        >
          <SectionHeading
            eyebrow="Stage conversion"
            title="Where progression changes"
          />

          <div
            className={
              styles.metricGridCompact
            }
          >
            <MetricCard
              label="Cart → checkout"
              value={formatPercentBPS(
                funnel.cart_to_checkout_bps,
              )}
              tone="blue"
            />

            <MetricCard
              label="Checkout → order"
              value={formatPercentBPS(
                funnel.checkout_to_order_bps,
              )}
              tone="violet"
            />

            <MetricCard
              label="Order → collected"
              value={formatPercentBPS(
                funnel.order_to_collected_bps,
              )}
              tone="positive"
            />

            <MetricCard
              label="Cart → collected"
              value={formatPercentBPS(
                funnel.cart_to_collected_bps,
              )}
            />
          </div>
        </section>
      </div>
    );
  }

  function renderFinance() {
    if (
      !profitLoss
    ) {
      return loading ? (
        <LoadingState />
      ) : null;
    }

    return (
      <div
        className={
          styles.viewStack
        }
      >
        <section
          className={
            styles.section
          }
        >
          <SectionHeading
            eyebrow="Profit & loss"
            title="Commercial profitability with coverage context"
            description="Profit is only shown as complete when cost coverage is complete. Known profit remains visible when some unit costs are missing."
          />

          <div
            className={
              styles.metricGrid
            }
          >
            <MetricCard
              label="Net collected revenue"
              value={formatMoney(
                profitLoss.net_collected_revenue_amount,
                profitLoss.currency,
              )}
              detail={`${formatCount(
                profitLoss.collected_orders,
              )} collected orders`}
              tone="positive"
            />

            <MetricCard
              label="Net merchandise revenue"
              value={formatMoney(
                profitLoss.net_merchandise_revenue_amount,
                profitLoss.currency,
              )}
              detail={`${formatMoney(
                profitLoss.shipping_revenue_amount,
                profitLoss.currency,
              )} shipping revenue`}
              tone="blue"
            />

            <MetricCard
              label="Net COGS"
              value={formatMoney(
                profitLoss.net_cogs_amount,
                profitLoss.currency,
              )}
              detail={`${formatPercentBPS(
                profitLoss.cogs_coverage_bps,
              )} cost coverage`}
            />

            <MetricCard
              label={
                profitLoss.profit_complete
                  ? "Gross profit"
                  : "Known gross profit"
              }
              value={formatMoney(
                profitLoss.gross_profit_amount ??
                  profitLoss.known_gross_profit_amount,
                profitLoss.currency,
              )}
              detail={
                profitLoss.profit_complete
                  ? `Gross margin ${formatPercentBPS(
                      profitLoss.gross_margin_bps,
                    )}`
                  : `${formatCount(
                      profitLoss.missing_cost_units,
                    )} sold units missing cost`
              }
              tone="violet"
            />

            <MetricCard
              label="Recorded expenses"
              value={formatMoney(
                profitLoss.recorded_expenses_amount,
                profitLoss.currency,
              )}
              detail={`${formatCount(
                profitLoss.recorded_expense_entries,
              )} entries · ${profitLoss.expense_basis}`}
            />

            <MetricCard
              label="Net after recorded expenses"
              value={
                profitLoss.net_profit_after_recorded_expenses_amount ===
                null
                  ? "Incomplete"
                  : formatMoney(
                      profitLoss.net_profit_after_recorded_expenses_amount,
                      profitLoss.currency,
                    )
              }
              detail={
                profitLoss.foreign_currency_expense_entries_excluded >
                0
                  ? `${formatCount(
                      profitLoss.foreign_currency_expense_entries_excluded,
                    )} foreign-currency expense entries excluded`
                  : "Selected-currency expense basis"
              }
              tone={
                profitLoss.net_profit_after_recorded_expenses_amount !==
                null
                  ? "positive"
                  : "warning"
              }
            />
          </div>
        </section>

        <section
          className={
            styles.section
          }
        >
          <SectionHeading
            eyebrow="Profit trend"
            title="Profit signal over the selected period"
            description="When full cost coverage is unavailable, the chart falls back to known gross profit for that bucket."
          />

          <div
            className={
              styles.chartPanel
            }
          >
            <TrendChart
              points={
                financeChartPoints
              }
              valueLabel="profit signal"
              emptyLabel="No finance trend points were returned for this period."
              formatter={(
                value,
              ) =>
                formatMoney(
                  value,
                  profitLoss.currency,
                )
              }
            />
          </div>
        </section>

        <div
          className={
            styles.twoColumnGrid
          }
        >
          <section
            className={
              styles.panel
            }
          >
            <SectionHeading
              eyebrow="Expenses"
              title="Recorded expense categories"
            />

            {profitLoss
              .expense_categories
              .length >
            0 ? (
              <div
                className={
                  styles.simpleList
                }
              >
                {profitLoss.expense_categories.map(
                  (
                    category,
                  ) => (
                    <div
                      key={
                        category.category
                      }
                      className={
                        styles.simpleRow
                      }
                    >
                      <div>
                        <strong>
                          {titleCase(
                            category.category,
                          )}
                        </strong>

                        <span>
                          {formatCount(
                            category.entries,
                          )}{" "}
                          entries
                        </span>
                      </div>

                      <strong>
                        {formatMoney(
                          category.amount,
                          profitLoss.currency,
                        )}
                      </strong>
                    </div>
                  ),
                )}
              </div>
            ) : (
              <EmptyRow>
                No recorded expenses in this period.
              </EmptyRow>
            )}
          </section>

          <section
            className={
              styles.panel
            }
          >
            <SectionHeading
              eyebrow="Coverage & returns"
              title="Profit completeness"
            />

            <div
              className={
                styles.coverageList
              }
            >
              <div>
                <span>
                  Sales COGS
                  coverage
                </span>

                <strong>
                  {formatPercentBPS(
                    profitLoss.sales_cogs_coverage_bps,
                  )}
                </strong>
              </div>

              <div>
                <span>
                  Restock COGS
                  coverage
                </span>

                <strong>
                  {formatPercentBPS(
                    profitLoss.restock_cogs_coverage_bps,
                  )}
                </strong>
              </div>

              <div>
                <span>
                  Known return
                  inventory loss
                </span>

                <strong>
                  {formatMoney(
                    profitLoss.known_return_inventory_loss_amount,
                    profitLoss.currency,
                  )}
                </strong>
              </div>

              <div>
                <span>
                  Non-restocked
                  return units
                </span>

                <strong>
                  {formatCount(
                    profitLoss.non_restocked_return_units,
                  )}
                </strong>
              </div>
            </div>
          </section>
        </div>

        {profitLoss.warnings
          .length >
        0 ? (
          <section
            className={
              styles.warningPanel
            }
          >
            <strong>
              Finance
              coverage notes
            </strong>

            <ul>
              {profitLoss.warnings.map(
                (
                  warning,
                ) => (
                  <li
                    key={
                      warning
                    }
                  >
                    {
                      warning
                    }
                  </li>
                ),
              )}
            </ul>
          </section>
        ) : null}
      </div>
    );
  }

  function renderRecommendations() {
    if (
      !recommendationOverview ||
      !recommendationEngine
    ) {
      return loading ? (
        <LoadingState />
      ) : null;
    }

    return (
      <div
        className={
          styles.viewStack
        }
      >
        <section
          className={
            styles.section
          }
        >
          <SectionHeading
            eyebrow="Recommendation performance"
            title="Engagement and attributed merchandise"
            description={`Purchase attribution requires a qualifying customer recommendation click within the prior ${recommendationOverview.attribution_window_days} days. The amount shown is attributed merchandise, not total order revenue.`}
          />

          <div
            className={
              styles.metricGrid
            }
          >
            <MetricCard
              label="Impressions"
              value={formatCount(
                recommendationOverview.impressions,
              )}
            />

            <MetricCard
              label="Clicks"
              value={formatCount(
                recommendationOverview.clicks,
              )}
              detail={`${formatPercentBPS(
                recommendationOverview.click_through_rate_bps,
              )} CTR`}
              tone="blue"
            />

            <MetricCard
              label="Add to carts"
              value={formatCount(
                recommendationOverview.add_to_carts,
              )}
              detail={`${formatPercentBPS(
                recommendationOverview.add_to_cart_rate_bps,
              )} impression → cart`}
              tone="violet"
            />

            <MetricCard
              label="Attributed orders"
              value={formatCount(
                recommendationOverview.attributed_orders,
              )}
              detail={`${formatCount(
                recommendationOverview.attributed_units,
              )} units`}
              tone="positive"
            />

            <MetricCard
              label="Attributed merchandise"
              value={formatMoney(
                recommendationOverview.attributed_merchandise_amount,
                recommendationOverview.currency,
              )}
              detail={`${recommendationOverview.attribution_window_days}-day click attribution`}
              tone="positive"
            />
          </div>

          <div
            className={
              styles.chartPanel
            }
          >
            <TrendChart
              points={
                recommendationChartPoints
              }
              valueLabel="attributed merchandise"
              emptyLabel="No recommendation trend points were returned for this period."
              formatter={(
                value,
              ) =>
                formatMoney(
                  value,
                  recommendationOverview.currency,
                )
              }
            />
          </div>
        </section>

        <section
          className={
            styles.section
          }
        >
          <SectionHeading
            eyebrow="Engine health"
            title="Recommendation inputs and coverage"
            description={`Behavior intent uses the backend's ${recommendationEngine.behavior_window_hours}-hour active signal window.`}
          />

          <div
            className={
              styles.metricGridCompact
            }
          >
            <MetricCard
              label="Eligible products"
              value={formatCount(
                recommendationEngine.eligible_products,
              )}
              tone="positive"
            />

            <MetricCard
              label="Customers with intent"
              value={formatCount(
                recommendationEngine.customers_with_active_intent,
              )}
              tone="blue"
            />

            <MetricCard
              label="Active search signals"
              value={formatCount(
                recommendationEngine.active_search_signals,
              )}
            />

            <MetricCard
              label="Related links"
              value={formatCount(
                recommendationEngine.related_relations,
              )}
              tone="violet"
            />

            <MetricCard
              label="Complementary links"
              value={formatCount(
                recommendationEngine.complementary_relations,
              )}
            />
          </div>
        </section>

        <div
          className={
            styles.twoColumnGrid
          }
        >
          <RecommendationBreakdownPanel
            title="Placement performance"
            eyebrow="Where recommendations appear"
            items={
              recommendationPlacements
            }
          />

          <RecommendationBreakdownPanel
            title="Strategy performance"
            eyebrow="How recommendations were selected"
            items={
              recommendationStrategies
            }
          />
        </div>

        <section
          className={
            styles.section
          }
        >
          <SectionHeading
            eyebrow="Product attribution"
            title="Products influenced by recommendations"
          />

          {recommendationProducts.length >
          0 ? (
            <div
              className={
                styles.dataGrid
              }
            >
              <div
                className={`${styles.dataRow} ${styles.dataHeader} ${styles.recommendationProductGrid}`}
              >
                <span>
                  Product
                </span>

                <span>
                  Impressions
                </span>

                <span>
                  CTR
                </span>

                <span>
                  Add to carts
                </span>

                <span>
                  Orders
                </span>

                <span>
                  Merchandise
                </span>
              </div>

              {recommendationProducts.map(
                (
                  product,
                ) => (
                  <div
                    key={
                      product.product_id
                    }
                    className={`${styles.dataRow} ${styles.recommendationProductGrid}`}
                  >
                    <span
                      data-label="Product"
                      className={
                        styles.primaryCell
                      }
                    >
                      <strong>
                        {
                          product.product_name
                        }
                      </strong>

                      <small>
                        {
                          product.product_id
                        }
                      </small>
                    </span>

                    <span
                      data-label="Impressions"
                    >
                      {formatCount(
                        product.impressions,
                      )}
                    </span>

                    <span
                      data-label="CTR"
                    >
                      {formatPercentBPS(
                        product.click_through_rate_bps,
                      )}
                    </span>

                    <span
                      data-label="Add to carts"
                    >
                      {formatCount(
                        product.add_to_carts,
                      )}
                    </span>

                    <span
                      data-label="Orders"
                    >
                      {formatCount(
                        product.attributed_orders,
                      )}
                    </span>

                    <span
                      data-label="Merchandise"
                    >
                      {formatMoney(
                        product.attributed_merchandise_amount,
                        product.currency,
                      )}
                    </span>
                  </div>
                ),
              )}
            </div>
          ) : (
            <EmptyRow>
              No recommendation product activity has been recorded yet.
            </EmptyRow>
          )}
        </section>

        <section
          className={
            styles.section
          }
        >
          <SectionHeading
            eyebrow="Category graph"
            title="Related and complementary category links"
            description="Higher weights are returned first by the recommendation engine diagnostics endpoint."
          />

          {recommendationRelations.length >
          0 ? (
            <div
              className={
                styles.relationGrid
              }
            >
              {recommendationRelations.map(
                (
                  relation,
                ) => (
                  <article
                    key={`${relation.source_category_id}-${relation.target_category_id}-${relation.relation_type}`}
                    className={`${styles.relationCard} ${relationTone(
                      relation.relation_type,
                    )}`}
                  >
                    <span
                      className={
                        styles.relationType
                      }
                    >
                      {titleCase(
                        relation.relation_type,
                      )}
                    </span>

                    <div
                      className={
                        styles.relationRoute
                      }
                    >
                      <strong>
                        {
                          relation.source_category_name
                        }
                      </strong>

                      <span
                        aria-hidden="true"
                      >
                        →
                      </span>

                      <strong>
                        {
                          relation.target_category_name
                        }
                      </strong>
                    </div>

                    <span
                      className={
                        styles.relationWeight
                      }
                    >
                      Weight{" "}
                      {
                        relation.weight
                      }
                    </span>
                  </article>
                ),
              )}
            </div>
          ) : (
            <EmptyRow>
              No category relations are configured.
            </EmptyRow>
          )}
        </section>
      </div>
    );
  }

  function renderActiveView() {
    switch (
      activeView
    ) {
      case "overview":
        return renderOverview();

      case "sales":
        return renderSales();

      case "products":
        return renderProducts();

      case "funnel":
        return renderFunnel();

      case "finance":
        return renderFinance();

      case "recommendations":
        return renderRecommendations();
    }
  }

  return (
    <div
      className={
        styles.page
      }
    >
      <AdminPageHeader
        eyebrow="Commerce intelligence"
        title="Analytics"
        description="Sales, funnel, profitability, inventory signals, and recommendation performance from the Go commerce backend."
        actions={
          <div
            className={
              styles.headerControls
            }
          >
            <div
              className={
                styles.periodControl
              }
              aria-label="Analytics period"
            >
              {PERIODS.map(
                (
                  option,
                ) => (
                  <button
                    key={
                      option.value
                    }
                    type="button"
                    className={
                      period ===
                      option.value
                        ? `${styles.periodButton} ${styles.periodButtonActive}`
                        : styles.periodButton
                    }
                    onClick={() =>
                      setPeriod(
                        option.value,
                      )
                    }
                    disabled={
                      loading
                    }
                  >
                    {
                      option.label
                    }
                  </button>
                ),
              )}
            </div>

            <button
              type="button"
              className={
                styles.refreshButton
              }
              onClick={() =>
                setRefreshNonce(
                  (
                    value,
                  ) =>
                    value +
                    1,
                )
              }
              disabled={
                loading ||
                !canRead
              }
            >
              <span
                className={
                  loading
                    ? styles.spinning
                    : ""
                }
                aria-hidden="true"
              >
                ↻
              </span>

              Refresh
            </button>
          </div>
        }
      />

      {!canRead ? (
        <PermissionState />
      ) : (
        <>
          <div
            className={
              styles.controlBar
            }
          >
            <div
              className={
                styles.viewTabs
              }
              role="tablist"
              aria-label="Analytics views"
            >
              {VIEWS.map(
                (
                  view,
                ) => (
                  <button
                    key={
                      view.id
                    }
                    type="button"
                    role="tab"
                    aria-selected={
                      activeView ===
                      view.id
                    }
                    className={
                      activeView ===
                      view.id
                        ? `${styles.viewTab} ${styles.viewTabActive}`
                        : styles.viewTab
                    }
                    onClick={() =>
                      setActiveView(
                        view.id,
                      )
                    }
                  >
                    <span
                      className={
                        styles.fullTabLabel
                      }
                    >
                      {
                        view.label
                      }
                    </span>

                    <span
                      className={
                        styles.shortTabLabel
                      }
                    >
                      {
                        view.shortLabel
                      }
                    </span>
                  </button>
                ),
              )}
            </div>

            <div
              className={
                styles.rangeMeta
              }
            >
              <span
                className={
                  styles.liveDot
                }
                aria-hidden="true"
              />

              <span>
                {
                  rangeLabel
                }
              </span>

              <span>
                ·
              </span>

              <span>
                {meta?.timezone ??
                  "Asia/Dhaka"}
              </span>
            </div>
          </div>

          {error ? (
            <div
              className={
                styles.errorBanner
              }
              role="alert"
            >
              <div>
                <strong>
                  Unable to
                  load this
                  analytics
                  view
                </strong>

                <span>
                  {
                    error
                  }
                </span>
              </div>

              <button
                type="button"
                onClick={() =>
                  setRefreshNonce(
                    (
                      value,
                    ) =>
                      value +
                      1,
                  )
                }
              >
                Try again
              </button>
            </div>
          ) : null}

          {loading &&
          !error ? (
            <div
              className={
                styles.loadingLine
              }
              aria-hidden="true"
            >
              <span />
            </div>
          ) : null}

          {renderActiveView()}

          <footer
            className={
              styles.dataFooter
            }
          >
            <span>
              {meta?.granularity
                ? `${titleCase(
                    meta.granularity,
                  )} buckets`
                : "Live engine diagnostics"}
            </span>

            <span>
              Currency{" "}
              {
                currentCurrency
              }
            </span>

            {overview?.generated_at &&
            activeView ===
              "overview" ? (
              <span>
                Generated{" "}
                {formatDateTime(
                  overview.generated_at,
                )}
              </span>
            ) : null}
          </footer>
        </>
      )}
    </div>
  );
}

function InventoryRiskRow({
  risk,
}: {
  risk: AdminInventoryRunRateRisk;
}) {
  const critical =
    risk.days_of_cover <=
    7;

  return (
    <article
      className={`${styles.riskRow} ${
        critical
          ? styles.riskCritical
          : styles.riskWarning
      }`}
    >
      <div
        className={
          styles.riskProduct
        }
      >
        <strong>
          {
            risk.product_name
          }
        </strong>

        <span>
          {
            risk.sku
          }
        </span>
      </div>

      <div
        className={
          styles.riskMetric
        }
      >
        <span>
          Available
        </span>

        <strong>
          {formatCount(
            risk.available_units,
          )}
        </strong>
      </div>

      <div
        className={
          styles.riskMetric
        }
      >
        <span>
          Units sold
        </span>

        <strong>
          {formatCount(
            risk.units_sold,
          )}
        </strong>
      </div>

      <div
        className={
          styles.riskMetric
        }
      >
        <span>
          Daily run rate
        </span>

        <strong>
          {formatDecimal(
            risk.daily_units_run_rate,
            2,
          )}
        </strong>
      </div>

      <div
        className={
          styles.riskMetric
        }
      >
        <span>
          Days cover
        </span>

        <strong>
          {formatDecimal(
            risk.days_of_cover,
          )}
        </strong>
      </div>
    </article>
  );
}

function RecommendationBreakdownPanel({
  eyebrow,
  title,
  items,
}: {
  eyebrow: string;
  title: string;
  items: AdminRecommendationBreakdown[];
}) {
  const maxAttributed =
    Math.max(
      ...items.map(
        (
          item,
        ) =>
          item.attributed_merchandise_amount,
      ),
      1,
    );

  return (
    <section
      className={
        styles.panel
      }
    >
      <SectionHeading
        eyebrow={
          eyebrow
        }
        title={
          title
        }
      />

      {items.length >
      0 ? (
        <div
          className={
            styles.breakdownList
          }
        >
          {items.map(
            (
              item,
            ) => (
              <div
                key={
                  item.key
                }
                className={
                  styles.breakdownRow
                }
              >
                <div
                  className={
                    styles.breakdownTopline
                  }
                >
                  <div>
                    <strong>
                      {titleCase(
                        item.key,
                      )}
                    </strong>

                    <span>
                      {formatCount(
                        item.impressions,
                      )}{" "}
                      impressions ·{" "}
                      {formatPercentBPS(
                        item.click_through_rate_bps,
                      )}{" "}
                      CTR
                    </span>
                  </div>

                  <strong>
                    {formatMoney(
                      item.attributed_merchandise_amount,
                      item.currency,
                    )}
                  </strong>
                </div>

                <div
                  className={
                    styles.breakdownTrack
                  }
                  aria-hidden="true"
                >
                  <span
                    style={{
                      width: `${Math.max(
                        2,
                        (
                          item.attributed_merchandise_amount /
                          maxAttributed
                        ) *
                          100,
                      )}%`,
                    }}
                  />
                </div>

                <div
                  className={
                    styles.breakdownMeta
                  }
                >
                  <span>
                    {formatCount(
                      item.clicks,
                    )}{" "}
                    clicks
                  </span>

                  <span>
                    {formatCount(
                      item.add_to_carts,
                    )}{" "}
                    add to carts
                  </span>

                  <span>
                    {formatCount(
                      item.attributed_orders,
                    )}{" "}
                    orders
                  </span>
                </div>
              </div>
            ),
          )}
        </div>
      ) : (
        <EmptyRow>
          No recommendation activity has been recorded yet.
        </EmptyRow>
      )}
    </section>
  );
}