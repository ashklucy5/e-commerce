"use client";

import Link from "next/link";
import {
  useEffect,
  useMemo,
  useState,
} from "react";

import AdminPageHeader from "@/components/admin/layout/components/AdminPageHeader";
import {
  adminFetch,
  AdminRequestError,
} from "@/lib/admin/api";
import type {
  AdminCatalogProductListItem,
  AdminCatalogProductListResponse,
} from "@/lib/admin/catalog-types";
import type {
  AdminAnalyticsOverviewResponse,
  AdminDashboardResponse,
  AdminDashboardSummary,
  AdminFinanceProfitLossResponse,
  AdminFinanceProfitLoss,
  AdminProductRequest,
  AdminProductRequestListResponse,
  AdminSalesPoint,
  AdminSalesTrendResponse,
} from "@/lib/admin/types";
import { formatMoney } from "@/lib/money/format";

import styles from "../css/AdminDashboard.module.css";

type AdminDashboardProps = {
  portal: string;
};

type PeriodDays = 7 | 30 | 90;

type BusinessInsightTone =
  | "positive"
  | "opportunity"
  | "attention"
  | "watch";

type BusinessInsight = {
  id: string;
  tone: BusinessInsightTone;
  kicker: string;
  title: string;
  summary: string;
  evidence: Array<{
    label: string;
    value: string;
  }>;
};

type AttentionItem = {
  id: string;
  label: string;
  value: number;
  detail: string;
  tone: "neutral" | "warning" | "critical";
};

type DashboardData = {
  summary: AdminDashboardSummary | null;
  overview: AdminAnalyticsOverviewResponse["data"] | null;
  sales: AdminSalesPoint[];
  finance: AdminFinanceProfitLoss | null;
  requests: AdminProductRequest[];
  products: AdminCatalogProductListItem[];
};

const EMPTY_DATA: DashboardData = {
  summary: null,
  overview: null,
  sales: [],
  finance: null,
  requests: [],
  products: [],
};

const PERIOD_OPTIONS: Array<{
  label: string;
  value: PeriodDays;
}> = [
  { label: "7D", value: 7 },
  { label: "30D", value: 30 },
  { label: "90D", value: 90 },
];

function businessDate(offsetDays = 0): string {
  const parts = new Intl.DateTimeFormat(
    "en-CA",
    {
      timeZone: "Asia/Dhaka",
      year: "numeric",
      month: "2-digit",
      day: "2-digit",
    },
  ).formatToParts(new Date());

  const year = Number(
    parts.find((part) => part.type === "year")?.value,
  );
  const month = Number(
    parts.find((part) => part.type === "month")?.value,
  );
  const day = Number(
    parts.find((part) => part.type === "day")?.value,
  );

  const date = new Date(
    Date.UTC(year, month - 1, day),
  );

  date.setUTCDate(
    date.getUTCDate() + offsetDays,
  );

  return date.toISOString().slice(0, 10);
}

function compactNumber(value: number): string {
  return new Intl.NumberFormat(
    "en",
    {
      notation: value >= 10_000 ? "compact" : "standard",
      maximumFractionDigits: 1,
    },
  ).format(value);
}

function formatPercentBPS(
  value: number | null | undefined,
): string {
  if (typeof value !== "number") {
    return "—";
  }

  return `${(value / 100).toFixed(1)}%`;
}

function friendlyStatus(value: string): string {
  return value
    .replaceAll("_", " ")
    .replace(/\b\w/g, (character) => character.toUpperCase());
}



function percentChange(
  previous: number,
  current: number,
): number | null {
  if (previous <= 0) {
    return null;
  }

  return ((current - previous) / previous) * 100;
}

function signedPercent(value: number): string {
  const rounded = Math.abs(value).toFixed(1);
  return `${value >= 0 ? "+" : "−"}${rounded}%`;
}

function insightToneClass(
  tone: BusinessInsightTone,
): string {
  switch (tone) {
    case "positive":
      return styles.insightPositive;
    case "opportunity":
      return styles.insightOpportunity;
    case "attention":
      return styles.insightAttention;
    case "watch":
      return styles.insightWatch;
  }
}

function attentionToneClass(
  tone: AttentionItem["tone"],
): string {
  switch (tone) {
    case "critical":
      return styles.attentionCritical;
    case "warning":
      return styles.attentionWarning;
    default:
      return styles.attentionNeutral;
  }
}

function requestStatusClass(status: string): string {
  switch (status) {
    case "pending_review":
      return styles.requestStatus_pending_review;
    case "negotiating":
      return styles.requestStatus_negotiating;
    case "accepted":
      return styles.requestStatus_accepted;
    case "agreed":
      return styles.requestStatus_agreed;
    case "converted_to_order":
      return styles.requestStatus_converted_to_order;
    case "cancelled":
      return styles.requestStatus_cancelled;
    default:
      return "";
  }
}

function relativeTime(value: string): string {
  const timestamp = new Date(value).getTime();

  if (!Number.isFinite(timestamp)) {
    return "Recently";
  }

  const seconds = Math.max(
    0,
    Math.round((Date.now() - timestamp) / 1000),
  );

  if (seconds < 60) {
    return "Just now";
  }

  const minutes = Math.floor(seconds / 60);

  if (minutes < 60) {
    return `${minutes}m ago`;
  }

  const hours = Math.floor(minutes / 60);

  if (hours < 24) {
    return `${hours}h ago`;
  }

  const days = Math.floor(hours / 24);

  if (days < 7) {
    return `${days}d ago`;
  }

  return new Intl.DateTimeFormat(
    "en",
    {
      month: "short",
      day: "numeric",
    },
  ).format(new Date(value));
}

function normalizeProducts(
  response: AdminCatalogProductListResponse,
): AdminCatalogProductListItem[] {
  if (Array.isArray(response.data)) {
    return response.data;
  }

  return (
    response.data.items ??
    response.data.Items ??
    []
  );
}

async function optionalAdminFetch<T>(
  path: string,
): Promise<T | null> {
  try {
    return await adminFetch<T>(path);
  } catch (error) {
    if (
      error instanceof AdminRequestError &&
      (error.status === 403 || error.status === 404)
    ) {
      return null;
    }

    throw error;
  }
}

function chartPath(
  points: AdminSalesPoint[],
): string {
  if (points.length === 0) {
    return "";
  }

  const values = points.map(
    (point) => point.net_collected_revenue_amount,
  );
  const maxValue = Math.max(...values, 1);
  const minValue = Math.min(...values, 0);
  const span = Math.max(maxValue - minValue, 1);

  const width = 720;
  const height = 240;
  const horizontalPadding = 14;
  const verticalPadding = 18;
  const usableWidth = width - horizontalPadding * 2;
  const usableHeight = height - verticalPadding * 2;

  return points
    .map((point, index) => {
      const x =
        horizontalPadding +
        (points.length === 1
          ? usableWidth / 2
          : (index / (points.length - 1)) * usableWidth);

      const normalized =
        (point.net_collected_revenue_amount - minValue) / span;

      const y =
        verticalPadding +
        (1 - normalized) * usableHeight;

      return `${index === 0 ? "M" : "L"}${x.toFixed(2)},${y.toFixed(2)}`;
    })
    .join(" ");
}

function RevenueTrend({
  points,
  currency,
}: {
  points: AdminSalesPoint[];
  currency: string;
}) {
  const path = useMemo(
    () => chartPath(points),
    [points],
  );

  const total = points.reduce(
    (sum, point) =>
      sum + point.net_collected_revenue_amount,
    0,
  );

  const peak = points.reduce<AdminSalesPoint | null>(
    (best, point) => {
      if (
        !best ||
        point.net_collected_revenue_amount >
          best.net_collected_revenue_amount
      ) {
        return point;
      }

      return best;
    },
    null,
  );

  if (points.length === 0) {
    return (
      <div className={styles.chartEmpty}>
        <span className={styles.chartEmptyMark}>↗</span>
        <p>No revenue activity in this period yet.</p>
      </div>
    );
  }

  return (
    <div className={styles.chartWrap}>
      <div className={styles.chartSummary}>
        <div>
          <span className={styles.chartLabel}>
            Period revenue
          </span>
          <strong className={styles.chartValue}>
            {formatMoney(total, currency)}
          </strong>
        </div>

        <div className={styles.chartPeak}>
          <span>Peak day</span>
          <strong>
            {peak
              ? formatMoney(
                  peak.net_collected_revenue_amount,
                  currency,
                )
              : "—"}
          </strong>
        </div>
      </div>

      <div className={styles.chartCanvas}>
        <svg
          viewBox="0 0 720 240"
          role="img"
          aria-label="Net collected revenue trend"
          preserveAspectRatio="none"
        >
          <defs>
            <linearGradient
              id="dashboardRevenueArea"
              x1="0"
              y1="0"
              x2="0"
              y2="1"
            >
              <stop
                offset="0%"
                stopColor="rgba(80, 214, 164, 0.30)"
              />
              <stop
                offset="100%"
                stopColor="rgba(102, 184, 255, 0)"
              />
            </linearGradient>
          </defs>

          <path
            className={styles.chartGridLine}
            d="M14 62 H706"
          />
          <path
            className={styles.chartGridLine}
            d="M14 120 H706"
          />
          <path
            className={styles.chartGridLine}
            d="M14 178 H706"
          />

          {path ? (
            <>
              <path
                className={styles.chartArea}
                d={`${path} L706,222 L14,222 Z`}
              />
              <path
                className={styles.chartLine}
                d={path}
              />
            </>
          ) : null}
        </svg>
      </div>

      <div className={styles.chartAxis}>
        <span>
          {new Intl.DateTimeFormat("en", {
            month: "short",
            day: "numeric",
          }).format(new Date(`${points[0].bucket_start}T00:00:00`))}
        </span>
        <span>
          {new Intl.DateTimeFormat("en", {
            month: "short",
            day: "numeric",
          }).format(
            new Date(
              `${points[points.length - 1].bucket_start}T00:00:00`,
            ),
          )}
        </span>
      </div>
    </div>
  );
}

export default function AdminDashboard({
  portal,
}: AdminDashboardProps) {
  const [periodDays, setPeriodDays] =
    useState<PeriodDays>(30);
  const [data, setData] =
    useState<DashboardData>(EMPTY_DATA);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [error, setError] = useState("");
  const [refreshToken, setRefreshToken] = useState(0);

  useEffect(() => {
    let cancelled = false;

    async function loadDashboard() {
      setRefreshing(true);
      setError("");

      const to = businessDate();
      const from = businessDate(-(periodDays - 1));
      const query = new URLSearchParams({
        from,
        to,
        granularity: "day",
        currency: "BDT",
      }).toString();

      try {
        const [
          summaryResponse,
          overviewResponse,
          salesResponse,
          financeResponse,
          requestsResponse,
          productsResponse,
        ] = await Promise.all([
          adminFetch<AdminDashboardResponse>("/dashboard"),
          optionalAdminFetch<AdminAnalyticsOverviewResponse>(
            `/analytics/overview?${query}`,
          ),
          optionalAdminFetch<AdminSalesTrendResponse>(
            `/analytics/sales?${query}`,
          ),
          optionalAdminFetch<AdminFinanceProfitLossResponse>(
            `/finance/profit-loss?${query}`,
          ),
          optionalAdminFetch<AdminProductRequestListResponse>(
            "/product-requests?limit=5&offset=0",
          ),
          optionalAdminFetch<AdminCatalogProductListResponse>(
            "/products?page=1&limit=5",
          ),
        ]);

        if (cancelled) {
          return;
        }

        setData({
          summary: summaryResponse.data,
          overview: overviewResponse?.data ?? null,
          sales: salesResponse?.data ?? [],
          finance: financeResponse?.data ?? null,
          requests: requestsResponse?.data ?? [],
          products: productsResponse
            ? normalizeProducts(productsResponse)
            : [],
        });
      } catch (value) {
        if (cancelled) {
          return;
        }

        setError(
          value instanceof Error
            ? value.message
            : "Dashboard data could not be loaded.",
        );
      } finally {
        if (!cancelled) {
          setLoading(false);
          setRefreshing(false);
        }
      }
    }

    void loadDashboard();

    return () => {
      cancelled = true;
    };
  }, [periodDays, refreshToken]);

  const currency =
    data.overview?.currency ??
    data.finance?.currency ??
    "BDT";

  const financeValue =
    data.finance?.net_profit_after_recorded_expenses_amount ??
    data.finance?.gross_profit_amount ??
    null;

  const financeLabel =
    data.finance?.net_profit_after_recorded_expenses_amount != null
      ? "Net profit"
      : "Gross profit";

  const generatedAt =
    data.summary?.generated_at ??
    data.overview?.generated_at ??
    null;

  const attentionRequests = data.requests.filter(
    (request) =>
      request.status === "pending_review" ||
      request.status === "negotiating" ||
      request.status === "on_hold",
  ).length;

  const lowStockProducts = data.products.filter(
    (product) => product.available_stock <= 5,
  );

  const businessInsights = useMemo<BusinessInsight[]>(() => {
    const insights: BusinessInsight[] = [];

    if (data.sales.length >= 4) {
      const midpoint = Math.floor(data.sales.length / 2);
      const earlierRevenue = data.sales
        .slice(0, midpoint)
        .reduce(
          (sum, point) => sum + point.net_collected_revenue_amount,
          0,
        );
      const recentRevenue = data.sales
        .slice(midpoint)
        .reduce(
          (sum, point) => sum + point.net_collected_revenue_amount,
          0,
        );
      const change = percentChange(earlierRevenue, recentRevenue);

      if (change != null) {
        insights.push({
          id: "revenue-momentum",
          tone:
            change >= 8
              ? "positive"
              : change <= -8
                ? "watch"
                : "opportunity",
          kicker:
            change >= 8
              ? "Positive trend"
              : change <= -8
                ? "Watch"
                : "Steady",
          title:
            change >= 8
              ? "Revenue momentum is improving"
              : change <= -8
                ? "Revenue momentum has softened"
                : "Revenue is holding a steady pace",
          summary:
            change >= 8
              ? "The recent half of the selected period collected more revenue than the earlier half. Keep an eye on the products and channels driving that momentum."
              : change <= -8
                ? "Recent collected revenue is below the earlier half of the selected period. Review order flow, product availability and sourcing demand before acting."
                : "Collected revenue is moving within a relatively stable range across the selected period.",
          evidence: [
            {
              label: "Recent vs earlier",
              value: signedPercent(change),
            },
            {
              label: "Recent revenue",
              value: formatMoney(recentRevenue, currency),
            },
          ],
        });
      }
    }

    if (attentionRequests > 0) {
      insights.push({
        id: "sourcing-demand",
        tone: attentionRequests >= 4 ? "attention" : "opportunity",
        kicker: attentionRequests >= 4 ? "Action" : "Opportunity",
        title: "Product request demand needs a response",
        summary:
          "Open sourcing conversations can turn into revenue when review, negotiation and confirmation steps move quickly.",
        evidence: [
          {
            label: "Requests needing attention",
            value: compactNumber(attentionRequests),
          },
          {
            label: "Visible requests",
            value: compactNumber(data.requests.length),
          },
        ],
      });
    }

    if (lowStockProducts.length > 0) {
      const lowest = lowStockProducts.reduce(
        (best, product) =>
          product.available_stock < best.available_stock ? product : best,
        lowStockProducts[0],
      );

      insights.push({
        id: "inventory-pressure",
        tone: lowest.available_stock <= 0 ? "attention" : "watch",
        kicker: lowest.available_stock <= 0 ? "Action" : "Watch",
        title:
          lowest.available_stock <= 0
            ? "Some catalog items are out of stock"
            : "Low-stock products are approaching a constraint",
        summary:
          "Review replenishment or sourcing before availability starts limiting otherwise healthy commerce activity.",
        evidence: [
          {
            label: "Low-stock products",
            value: compactNumber(lowStockProducts.length),
          },
          {
            label: "Lowest visible stock",
            value: compactNumber(lowest.available_stock),
          },
        ],
      });
    }

    const grossRevenue =
      data.overview?.gross_collected_revenue_amount ?? 0;
    const refundAmount =
      data.overview?.successful_refund_amount ?? 0;

    if (grossRevenue > 0) {
      const refundRate = (refundAmount / grossRevenue) * 100;

      insights.push({
        id: "refund-pressure",
        tone: refundRate <= 3 ? "positive" : refundRate >= 7 ? "watch" : "opportunity",
        kicker:
          refundRate <= 3
            ? "Healthy signal"
            : refundRate >= 7
              ? "Watch"
              : "Monitor",
        title:
          refundRate <= 3
            ? "Refund pressure is currently contained"
            : refundRate >= 7
              ? "Refund pressure is elevated"
              : "Refund activity is worth monitoring",
        summary:
          refundRate <= 3
            ? "Successful refunds remain a small share of collected gross revenue for the selected period."
            : "Compare the affected orders, products and return reasons before deciding whether an operational change is needed.",
        evidence: [
          {
            label: "Refund share",
            value: `${refundRate.toFixed(1)}%`,
          },
          {
            label: "Refund amount",
            value: formatMoney(refundAmount, currency),
          },
        ],
      });
    }

    return insights.slice(0, 4);
  }, [
    attentionRequests,
    currency,
    data.overview,
    data.requests.length,
    data.sales,
    lowStockProducts,
  ]);

  const attentionItems = useMemo<AttentionItem[]>(() => {
    const items: AttentionItem[] = [
      {
        id: "fulfillment",
        label: "Pending fulfillment",
        value: data.summary?.pending_fulfillments ?? 0,
        detail: "Warehouse work still waiting to complete",
        tone:
          (data.summary?.pending_fulfillments ?? 0) >= 10
            ? "critical"
            : "warning",
      },
      {
        id: "shipments",
        label: "Awaiting shipment confirmation",
        value: data.summary?.awaiting_confirmation_shipments ?? 0,
        detail: "Shipments waiting for confirmation",
        tone:
          (data.summary?.awaiting_confirmation_shipments ?? 0) >= 10
            ? "critical"
            : "warning",
      },
      {
        id: "sourcing",
        label: "Product requests",
        value: attentionRequests,
        detail: "Requests in review, negotiation or on hold",
        tone: attentionRequests >= 5 ? "critical" : "warning",
      },
      {
        id: "support",
        label: "Open support cases",
        value: data.summary?.open_crm_cases ?? 0,
        detail: "Customer cases still open",
        tone:
          (data.summary?.open_crm_cases ?? 0) >= 10
            ? "critical"
            : "warning",
      },
      {
        id: "stock",
        label: "Low-stock products",
        value: lowStockProducts.length,
        detail: "Visible catalog items with five units or fewer",
        tone: lowStockProducts.some((product) => product.available_stock <= 0)
          ? "critical"
          : "warning",
      },
    ];

    return items
      .filter((item) => item.value > 0)
      .sort((left, right) => right.value - left.value)
      .slice(0, 5);
  }, [attentionRequests, data.summary, lowStockProducts]);

  return (
    <div className={styles.dashboard}>
      <AdminPageHeader
        eyebrow="Commerce command center"
        title="Dashboard"
        description="Revenue, orders, sourcing, catalog and inventory — one live view of the business."
        actions={
          <div className={styles.headerActions}>
            <div
              className={styles.periodControl}
              aria-label="Dashboard period"
            >
              {PERIOD_OPTIONS.map((option) => (
                <button
                  key={option.value}
                  type="button"
                  className={[
                    styles.periodButton,
                    periodDays === option.value
                      ? styles.periodButtonActive
                      : "",
                  ]
                    .filter(Boolean)
                    .join(" ")}
                  onClick={() => setPeriodDays(option.value)}
                >
                  {option.label}
                </button>
              ))}
            </div>

            <button
              type="button"
              className={styles.refreshButton}
              onClick={() =>
                setRefreshToken((value) => value + 1)
              }
              disabled={loading || refreshing}
            >
              <span
                className={refreshing ? styles.spinning : ""}
                aria-hidden="true"
              >
                ↻
              </span>
              Refresh
            </button>
          </div>
        }
      />

      {error ? (
        <div className={styles.errorBanner} role="alert">
          <div>
            <strong>Dashboard data is temporarily unavailable.</strong>
            <span>{error}</span>
          </div>
          <button
            type="button"
            onClick={() => setRefreshToken((value) => value + 1)}
          >
            Try again
          </button>
        </div>
      ) : null}

      <section className={styles.section}>
        <div className={styles.sectionHeading}>
          <div>
            <p className={styles.sectionEyebrow}>Business health</p>
            <h2>Finance & revenue</h2>
          </div>
          <p className={styles.sectionMeta}>
            {generatedAt
              ? `Updated ${relativeTime(generatedAt)}`
              : "Live business metrics"}
          </p>
        </div>

        <div className={styles.kpiGrid}>
          <article className={`${styles.kpiCard} ${styles.kpiCardPrimary}`}>
            <span className={styles.kpiLabel}>Net revenue</span>
            <strong className={styles.kpiValue}>
              {loading
                ? "—"
                : formatMoney(
                    data.overview?.net_collected_revenue_amount ??
                      data.summary?.revenue_today_amount ??
                      0,
                    currency,
                  )}
            </strong>
            <span className={styles.kpiDetail}>
              {data.overview
                ? `${periodDays}-day collected revenue after refunds`
                : "Revenue collected today"}
            </span>
          </article>

          <article className={styles.kpiCard}>
            <span className={styles.kpiLabel}>Gross revenue</span>
            <strong className={styles.kpiValue}>
              {loading
                ? "—"
                : data.overview
                  ? formatMoney(
                      data.overview.gross_collected_revenue_amount,
                      currency,
                    )
                  : "—"}
            </strong>
            <span className={styles.kpiDetail}>
              Refunds {data.overview
                ? formatMoney(
                    data.overview.successful_refund_amount,
                    currency,
                  )
                : "—"}
            </span>
          </article>

          <article className={styles.kpiCard}>
            <span className={styles.kpiLabel}>{financeLabel}</span>
            <strong className={styles.kpiValue}>
              {loading
                ? "—"
                : financeValue != null
                  ? formatMoney(financeValue, currency)
                  : "Restricted"}
            </strong>
            <span className={styles.kpiDetail}>
              Margin {formatPercentBPS(data.finance?.gross_margin_bps)}
            </span>
          </article>

          <article className={styles.kpiCard}>
            <span className={styles.kpiLabel}>Collected orders</span>
            <strong className={styles.kpiValue}>
              {loading
                ? "—"
                : compactNumber(
                    data.overview?.collected_orders ??
                      data.summary?.orders_today ??
                      0,
                  )}
            </strong>
            <span className={styles.kpiDetail}>
              {data.overview
                ? `${compactNumber(data.overview.orders_created)} created in period`
                : `${compactNumber(data.summary?.orders_today ?? 0)} orders today`}
            </span>
          </article>

          <article className={styles.kpiCard}>
            <span className={styles.kpiLabel}>Average order value</span>
            <strong className={styles.kpiValue}>
              {loading
                ? "—"
                : data.overview
                  ? formatMoney(
                      data.overview.average_order_value_amount,
                      currency,
                    )
                  : "—"}
            </strong>
            <span className={styles.kpiDetail}>
              {data.overview
                ? `${compactNumber(data.overview.new_customers)} new customers`
                : `${compactNumber(data.summary?.active_customers ?? 0)} active customers`}
            </span>
          </article>
        </div>

        <div className={styles.analyticsGrid}>
          <article className={styles.panel}>
            <div className={styles.panelHeader}>
              <div>
                <span className={styles.panelEyebrow}>Analytics</span>
                <h3>Revenue performance</h3>
              </div>
              <span className={styles.panelTag}>{periodDays} days</span>
            </div>

            <RevenueTrend
              points={data.sales}
              currency={currency}
            />
          </article>

          <article className={`${styles.panel} ${styles.financePanel}`}>
            <div className={styles.panelHeader}>
              <div>
                <span className={styles.panelEyebrow}>Finance</span>
                <h3>Profit snapshot</h3>
              </div>
              <span className={styles.panelTag}>BDT</span>
            </div>

            {data.finance ? (
              <div className={styles.financeList}>
                <div className={styles.financeRow}>
                  <span>Collected revenue</span>
                  <strong>
                    {formatMoney(
                      data.finance.net_collected_revenue_amount,
                      data.finance.currency,
                    )}
                  </strong>
                </div>
                <div className={styles.financeRow}>
                  <span>Net COGS</span>
                  <strong>
                    {formatMoney(
                      data.finance.net_cogs_amount,
                      data.finance.currency,
                    )}
                  </strong>
                </div>
                <div className={styles.financeRow}>
                  <span>Recorded expenses</span>
                  <strong>
                    {formatMoney(
                      data.finance.recorded_expenses_amount,
                      data.finance.currency,
                    )}
                  </strong>
                </div>
                <div className={styles.financeRow}>
                  <span>Refunds</span>
                  <strong>
                    {formatMoney(
                      data.finance.successful_refund_amount,
                      data.finance.currency,
                    )}
                  </strong>
                </div>
                <div className={styles.financeDivider} />
                <div className={`${styles.financeRow} ${styles.financeRowEmphasis}`}>
                  <span>{financeLabel}</span>
                  <strong>
                    {financeValue != null
                      ? formatMoney(
                          financeValue,
                          data.finance.currency,
                        )
                      : "Incomplete cost coverage"}
                  </strong>
                </div>
                <div className={styles.coverageBar}>
                  <span
                    style={{
                      width: `${Math.min(100, data.finance.cogs_coverage_bps / 100)}%`,
                    }}
                  />
                </div>
                <p className={styles.coverageCopy}>
                  COGS coverage {formatPercentBPS(data.finance.cogs_coverage_bps)}
                </p>
              </div>
            ) : (
              <div className={styles.permissionEmpty}>
                Finance analytics are hidden for this role.
              </div>
            )}
          </article>
        </div>
      </section>

      <section className={styles.section}>
        <div className={styles.sectionHeading}>
          <div>
            <p className={styles.sectionEyebrow}>Decision support</p>
            <h2>Business recommendations</h2>
          </div>
          <p className={styles.sectionMeta}>
            Derived from live finance, sales, sourcing and inventory metrics
          </p>
        </div>

        {businessInsights.length > 0 ? (
          <div className={styles.insightGrid}>
            {businessInsights.map((insight) => (
              <article
                key={insight.id}
                className={[styles.insightCard, insightToneClass(insight.tone)]
                  .filter(Boolean)
                  .join(" ")}
              >
                <div className={styles.insightTopline}>
                  <span>{insight.kicker}</span>
                  <i aria-hidden="true" />
                </div>
                <h3>{insight.title}</h3>
                <p>{insight.summary}</p>
                <div className={styles.insightEvidence}>
                  {insight.evidence.map((item) => (
                    <div key={`${insight.id}-${item.label}`}>
                      <span>{item.label}</span>
                      <strong>{item.value}</strong>
                    </div>
                  ))}
                </div>
              </article>
            ))}
          </div>
        ) : (
          <div className={styles.recommendationEmpty}>
            <strong>No recommendation signal yet</strong>
            <span>More live activity is needed before the dashboard can derive useful business guidance.</span>
          </div>
        )}

        <article className={styles.attentionCenter}>
          <div className={styles.attentionHeader}>
            <div>
              <span>Operations</span>
              <h3>Attention center</h3>
            </div>
            <strong>
              {attentionItems.length > 0
                ? `${attentionItems.length} active signal${attentionItems.length === 1 ? "" : "s"}`
                : "Clear"}
            </strong>
          </div>

          {attentionItems.length > 0 ? (
            <div className={styles.attentionGrid}>
              {attentionItems.map((item) => (
                <div
                  key={item.id}
                  className={[styles.attentionItem, attentionToneClass(item.tone)]
                    .filter(Boolean)
                    .join(" ")}
                >
                  <span className={styles.attentionDot} aria-hidden="true" />
                  <div>
                    <strong>{item.label}</strong>
                    <span>{item.detail}</span>
                  </div>
                  <em>{compactNumber(item.value)}</em>
                </div>
              ))}
            </div>
          ) : (
            <div className={styles.attentionClear}>
              <span aria-hidden="true">✓</span>
              <div>
                <strong>No urgent operational exceptions</strong>
                <p>Current dashboard signals do not show outstanding fulfillment, sourcing, support or low-stock pressure.</p>
              </div>
            </div>
          )}
        </article>
      </section>

      <section className={styles.section}>
        <div className={styles.sectionHeading}>
          <div>
            <p className={styles.sectionEyebrow}>Commerce</p>
            <h2>Orders & sales</h2>
          </div>
          <p className={styles.sectionMeta}>Today + selected period</p>
        </div>

        <div className={styles.commerceGrid}>
          <article className={styles.commerceMetric}>
            <span>Orders today</span>
            <strong>{compactNumber(data.summary?.orders_today ?? 0)}</strong>
            <small>New orders since business-day start</small>
          </article>
          <article className={styles.commerceMetric}>
            <span>Units sold</span>
            <strong>{compactNumber(data.overview?.units_sold ?? 0)}</strong>
            <small>{periodDays}-day collected units</small>
          </article>
          <article className={styles.commerceMetric}>
            <span>Pending fulfillment</span>
            <strong>{compactNumber(data.summary?.pending_fulfillments ?? 0)}</strong>
            <small>Warehouse work still in progress</small>
          </article>
          <article className={styles.commerceMetric}>
            <span>Awaiting shipment</span>
            <strong>{compactNumber(data.summary?.awaiting_confirmation_shipments ?? 0)}</strong>
            <small>Shipments waiting for confirmation</small>
          </article>
        </div>
      </section>

      <section className={styles.section}>
        <div className={styles.sectionHeading}>
          <div>
            <p className={styles.sectionEyebrow}>Sourcing</p>
            <h2>Product requests</h2>
          </div>
          <p className={styles.sectionMeta}>
            {data.requests.length > 0
              ? `${attentionRequests} need attention`
              : "Live request queue"}
          </p>
        </div>

        <article className={styles.listPanel}>
          {data.requests.length > 0 ? (
            <div className={styles.requestList}>
              {data.requests.map((request) => (
                <div key={request.id} className={styles.requestRow}>
                  <div className={styles.requestIdentity}>
                    <span className={styles.requestNumber}>
                      {request.request_number}
                    </span>
                    <strong>{request.requested_product_name}</strong>
                    <span>{request.customer.full_name}</span>
                  </div>

                  <div className={styles.requestQuantity}>
                    <span>Qty</span>
                    <strong>{compactNumber(request.requested_quantity)}</strong>
                  </div>

                  <span
                    className={[styles.statusBadge, requestStatusClass(request.status)]
                      .filter(Boolean)
                      .join(" ")}
                  >
                    {friendlyStatus(request.status)}
                  </span>

                  <span className={styles.rowTime}>
                    {relativeTime(request.last_message_at || request.updated_at)}
                  </span>
                </div>
              ))}
            </div>
          ) : (
            <div className={styles.permissionEmpty}>
              No product requests are visible for this role, or the queue is empty.
            </div>
          )}
        </article>
      </section>

      <section className={styles.bottomGrid}>
        <div className={styles.sectionCompact}>
          <div className={styles.sectionHeading}>
            <div>
              <p className={styles.sectionEyebrow}>Catalog</p>
              <h2>Products</h2>
            </div>
            <Link
              href={`/${portal}/products`}
              className={styles.sectionLink}
            >
              Open catalog →
            </Link>
          </div>

          <article className={styles.listPanel}>
            <div className={styles.summaryStrip}>
              <div>
                <span>Active</span>
                <strong>{compactNumber(data.summary?.active_products ?? 0)}</strong>
              </div>
              <div>
                <span>Variants</span>
                <strong>{compactNumber(data.summary?.active_variants ?? 0)}</strong>
              </div>
              <div>
                <span>Published reviews</span>
                <strong>{compactNumber(data.summary?.published_reviews ?? 0)}</strong>
              </div>
            </div>

            {data.products.length > 0 ? (
              <div className={styles.productList}>
                {data.products.map((product) => (
                  <div key={product.id} className={styles.productRow}>
                    <div className={styles.productCopy}>
                      <strong>{product.name}</strong>
                      <span>
                        {product.product_code} · {product.category_name}
                      </span>
                    </div>
                    <span className={styles.productStatus}>
                      {friendlyStatus(product.status)}
                    </span>
                    <strong className={styles.stockNumber}>
                      {compactNumber(product.available_stock)}
                    </strong>
                  </div>
                ))}
              </div>
            ) : (
              <div className={styles.permissionEmpty}>
                Product catalog summary is unavailable for this role.
              </div>
            )}
          </article>
        </div>

        <div className={styles.sectionCompact}>
          <div className={styles.sectionHeading}>
            <div>
              <p className={styles.sectionEyebrow}>Operations</p>
              <h2>Inventory</h2>
            </div>
            <p className={styles.sectionMeta}>Stock health</p>
          </div>

          <article className={`${styles.listPanel} ${styles.inventoryPanel}`}>
            <div className={styles.inventoryHero}>
              <span>Available units</span>
              <strong>{compactNumber(data.summary?.available_units ?? 0)}</strong>
              <p>Across active inventory records</p>
            </div>

            <div className={styles.inventoryStats}>
              <div>
                <span>Active variants</span>
                <strong>{compactNumber(data.summary?.active_variants ?? 0)}</strong>
              </div>
              <div>
                <span>Recent low stock</span>
                <strong>{compactNumber(lowStockProducts.length)}</strong>
              </div>
              <div>
                <span>Open support</span>
                <strong>{compactNumber(data.summary?.open_crm_cases ?? 0)}</strong>
              </div>
            </div>

            {lowStockProducts.length > 0 ? (
              <div className={styles.stockAlerts}>
                {lowStockProducts.slice(0, 3).map((product) => (
                  <div key={product.id}>
                    <span>{product.product_code}</span>
                    <strong>{product.name}</strong>
                    <em>{product.available_stock} available</em>
                  </div>
                ))}
              </div>
            ) : (
              <p className={styles.inventoryFootnote}>
                No low-stock items appear in the latest catalog activity.
              </p>
            )}
          </article>
        </div>
      </section>
    </div>
  );
}
