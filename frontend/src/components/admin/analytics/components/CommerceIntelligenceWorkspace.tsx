"use client";

import {
  useCallback,
  useEffect,
  useMemo,
  useState,
} from "react";

import AdminPageHeader from "@/components/admin/layout/components/AdminPageHeader";
import { useAdminSession } from "@/components/admin/layout/context/AdminSessionContext";
import CommerceIntelligenceHeader from "./CommerceIntelligenceHeader";

import {
  adminFetch,
  AdminRequestError,
} from "@/lib/admin/api";

import type {
  AdminAnalyticsOverviewResponse,
} from "@/lib/admin/types";

import type {
  BusinessInsight,
  BusinessInsights,
  BusinessInsightsResponse,
  FinanceProfitLoss,
  FinanceProfitLossPoint,
  FinanceProfitLossResponse,
  FinanceProfitLossTrendResponse,
} from "@/lib/admin/commerce-intelligence-types";

import { formatMoney } from "@/lib/money/format";

import ProfitSpine from "./ProfitSpine";

import styles from "../css/CommerceIntelligence.module.css";

type Props = {
  portal: string;
};

type PeriodDays =
  | 7
  | 30
  | 90;

type WorkspaceMode =
  | "overview"
  | "profit";

type IntelligenceState = {
  overview:
    | AdminAnalyticsOverviewResponse["data"]
    | null;

  finance:
    | FinanceProfitLoss
    | null;

  financeTrend:
    FinanceProfitLossPoint[];

  insights:
    | BusinessInsights
    | null;
};

const EMPTY_STATE: IntelligenceState = {
  overview: null,
  finance: null,
  financeTrend: [],
  insights: null,
};

function businessDate(
  offsetDays = 0,
): string {
  const parts =
    new Intl.DateTimeFormat(
      "en-CA",
      {
        timeZone:
          "Asia/Dhaka",

        year:
          "numeric",

        month:
          "2-digit",

        day:
          "2-digit",
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

  const value =
    new Date(
      Date.UTC(
        year,
        month - 1,
        day,
      ),
    );

  value.setUTCDate(
    value.getUTCDate() +
      offsetDays,
  );

  return value
    .toISOString()
    .slice(
      0,
      10,
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
    value / 100
  ).toFixed(1)}%`;
}

function compactNumber(
  value: number,
): string {
  return new Intl.NumberFormat(
    "en",
    {
      notation:
        Math.abs(value) >=
        10_000
          ? "compact"
          : "standard",

      maximumFractionDigits:
        1,
    },
  ).format(
    value,
  );
}

function relativeTime(
  value: string,
): string {
  const time =
    new Date(
      value,
    ).getTime();

  if (
    Number.isNaN(
      time,
    )
  ) {
    return "recently";
  }

  const seconds =
    Math.max(
      0,
      Math.floor(
        (
          Date.now() -
          time
        ) / 1000,
      ),
    );

  if (
    seconds <
    60
  ) {
    return "just now";
  }

  const minutes =
    Math.floor(
      seconds / 60,
    );

  if (
    minutes <
    60
  ) {
    return `${minutes}m ago`;
  }

  const hours =
    Math.floor(
      minutes / 60,
    );

  if (
    hours <
    24
  ) {
    return `${hours}h ago`;
  }

  const days =
    Math.floor(
      hours / 24,
    );

  return `${days}d ago`;
}

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

  return "Commerce intelligence could not be loaded.";
}

function signalTone(
  signal: BusinessInsight,
): string {
  switch (
    signal.severity
  ) {
    case "positive":
      return styles.signalPositive;

    case "high":
      return styles.signalHigh;

    case "warning":
      return styles.signalWarning;

    default:
      return styles.signalNeutral;
  }
}

function signalLabel(
  severity: string,
): string {
  switch (
    severity
  ) {
    case "positive":
      return "Positive";

    case "high":
      return "Needs attention";

    case "warning":
      return "Watch";

    default:
      return "Signal";
  }
}

function buildTrendPath(
  values: number[],
  width: number,
  height: number,
): string {
  if (
    values.length ===
    0
  ) {
    return "";
  }

  const minimum =
    Math.min(
      ...values,
      0,
    );

  const maximum =
    Math.max(
      ...values,
      1,
    );

  const span =
    Math.max(
      maximum -
        minimum,
      1,
    );

  const padX =
    16;

  const padY =
    18;

  const usableWidth =
    width -
    padX * 2;

  const usableHeight =
    height -
    padY * 2;

  return values
    .map(
      (
        value,
        index,
      ) => {
        const x =
          padX +
          (
            values.length ===
            1
              ? usableWidth /
                2
              : (
                  index /
                  (
                    values.length -
                    1
                  )
                ) *
                usableWidth
          );

        const ratio =
          (
            value -
            minimum
          ) /
          span;

        const y =
          padY +
          (
            1 -
            ratio
          ) *
          usableHeight;

        return `${
          index ===
          0
            ? "M"
            : "L"
        }${x.toFixed(
          2,
        )},${y.toFixed(
          2,
        )}`;
      },
    )
    .join(
      " ",
    );
}

function FinanceTrend({
  points,
  currency,
}: {
  points:
    FinanceProfitLossPoint[];

  currency:
    string;
}) {
  const width =
    920;

  const height =
    300;

  const revenuePath =
    useMemo(
      () =>
        buildTrendPath(
          points.map(
            (point) =>
              point.net_collected_revenue_amount,
          ),
          width,
          height,
        ),
      [
        points,
      ],
    );

  const profitPoints =
    useMemo(
      () =>
        points.filter(
          (
            point,
          ) =>
            point.gross_profit_amount !=
            null,
        ),
      [
        points,
      ],
    );

  const profitPath =
    useMemo(
      () =>
        buildTrendPath(
          profitPoints.map(
            (point) =>
              point.gross_profit_amount ??
              0,
          ),
          width,
          height,
        ),
      [
        profitPoints,
      ],
    );

  const revenue =
    points.reduce(
      (
        total,
        point,
      ) =>
        total +
        point.net_collected_revenue_amount,
      0,
    );

  const completeProfit =
    points.length >
      0 &&
    points.every(
      (point) =>
        point.profit_complete &&
        point.gross_profit_amount !=
          null,
    );

  const profit =
    completeProfit
      ? points.reduce(
          (
            total,
            point,
          ) =>
            total +
            (
              point.gross_profit_amount ??
              0
            ),
          0,
        )
      : null;

  if (
    points.length ===
    0
  ) {
    return (
      <div
        className={
          styles.emptyTrend
        }
      >
        <strong>
          No finance movement yet
        </strong>

        <p>
          Collected revenue and
          profit movement will
          appear when the selected
          period contains paid
          commerce activity.
        </p>
      </div>
    );
  }

  return (
    <div
      className={
        styles.trendContent
      }
    >
      <div
        className={
          styles.trendSummary
        }
      >
        <div>
          <span>
            Net collected
          </span>

          <strong>
            {formatMoney(
              revenue,
              currency,
            )}
          </strong>
        </div>

        <div>
          <span>
            Gross profit
          </span>

          <strong>
            {profit !=
            null
              ? formatMoney(
                  profit,
                  currency,
                )
              : "Partial"}
          </strong>
        </div>
      </div>

      <div
        className={
          styles.chartFrame
        }
      >
        <div
          className={
            styles.chartGlow
          }
          aria-hidden="true"
        />

        <svg
          className={
            styles.trendChart
          }
          viewBox={`0 0 ${width} ${height}`}
          role="img"
          aria-label="Net collected revenue and gross profit movement"
          preserveAspectRatio="none"
        >
          <defs>
            <linearGradient
              id="commerce-intelligence-revenue"
              x1="0"
              y1="0"
              x2="1"
              y2="0"
            >
              <stop
                offset="0%"
                stopColor="rgba(118, 215, 208, 0.66)"
              />

              <stop
                offset="100%"
                stopColor="rgba(131, 175, 255, 0.96)"
              />
            </linearGradient>
          </defs>

          <path
            d={
              revenuePath
            }
            className={
              styles.revenuePath
            }
          />

          {profitPath ? (
            <path
              d={
                profitPath
              }
              className={
                styles.profitPath
              }
            />
          ) : null}
        </svg>

        <div
          className={
            styles.chartLegend
          }
        >
          <span>
            <i
              className={
                styles.legendRevenue
              }
            />

            Net collected
          </span>

          <span>
            <i
              className={
                styles.legendProfit
              }
            />

            Complete gross profit
          </span>
        </div>
      </div>

      <div
        className={
          styles.chartAxis
        }
      >
        <span>
          {new Intl.DateTimeFormat(
            "en",
            {
              month:
                "short",

              day:
                "numeric",
            },
          ).format(
            new Date(
              `${points[0].bucket_start}T00:00:00`,
            ),
          )}
        </span>

        <span>
          {new Intl.DateTimeFormat(
            "en",
            {
              month:
                "short",

              day:
                "numeric",
            },
          ).format(
            new Date(
              `${
                points[
                  points.length -
                    1
                ].bucket_start
              }T00:00:00`,
            ),
          )}
        </span>
      </div>
    </div>
  );
}

export default function CommerceIntelligenceWorkspace({
  portal,
}: Props) {
  const principal =
    useAdminSession();

  const permissions =
    principal.staff
      .permissions;

  const superAdmin =
    principal.staff.roles.includes(
      "admin_superuser",
    );

  const hasPermission =
    useCallback(
      (
        permission:
          string,
      ) =>
        superAdmin ||
        permissions.includes(
          permission,
        ),
      [
        permissions,
        superAdmin,
      ],
    );

  const canAnalyticsRead =
    hasPermission(
      "admin.analytics.read",
    );

  const canFinanceRead =
    hasPermission(
      "admin.finance.read",
    );

  const [
    mode,
    setMode,
  ] =
    useState<WorkspaceMode>(
      canFinanceRead
        ? "overview"
        : "overview",
    );

  const [
    periodDays,
    setPeriodDays,
  ] =
    useState<PeriodDays>(
      30,
    );

  const [
    data,
    setData,
  ] =
    useState<IntelligenceState>(
      EMPTY_STATE,
    );

  const [
    loading,
    setLoading,
  ] =
    useState(
      true,
    );

  const [
    refreshing,
    setRefreshing,
  ] =
    useState(
      false,
    );

  const [
    analyticsError,
    setAnalyticsError,
  ] =
    useState(
      "",
    );

  const [
    financeError,
    setFinanceError,
  ] =
    useState(
      "",
    );

  const [
    refreshKey,
    setRefreshKey,
  ] =
    useState(
      0,
    );

  useEffect(
    () => {
      let cancelled =
        false;

      const controller =
        new AbortController();

      async function load() {
        setRefreshing(
          true,
        );

        setAnalyticsError(
          "",
        );

        setFinanceError(
          "",
        );

        const to =
          businessDate();

        const from =
          businessDate(
            -(
              periodDays -
              1
            ),
          );

        const query =
          new URLSearchParams(
            {
              from,
              to,

              granularity:
                "day",

              currency:
                "BDT",
            },
          ).toString();

        const analyticsTask =
          async () => {
            if (
              !canAnalyticsRead
            ) {
              return;
            }

            try {
              const [
                overviewResponse,
                insightsResponse,
              ] =
                await Promise.all(
                  [
                    adminFetch<AdminAnalyticsOverviewResponse>(
                      `/analytics/overview?${query}`,
                      {
                        signal:
                          controller.signal,
                      },
                    ),

                    adminFetch<BusinessInsightsResponse>(
                      `/analytics/insights?${query}`,
                      {
                        signal:
                          controller.signal,
                      },
                    ),
                  ],
                );

              if (
                cancelled
              ) {
                return;
              }

              setData(
                (
                  current,
                ) => ({
                  ...current,

                  overview:
                    overviewResponse.data,

                  insights:
                    insightsResponse.data,
                }),
              );
            } catch (
              value
            ) {
              if (
                cancelled
              ) {
                return;
              }

              setAnalyticsError(
                errorMessage(
                  value,
                ),
              );
            }
          };

        const financeTask =
          async () => {
            if (
              !canFinanceRead
            ) {
              return;
            }

            try {
              const [
                financeResponse,
                trendResponse,
              ] =
                await Promise.all(
                  [
                    adminFetch<FinanceProfitLossResponse>(
                      `/finance/profit-loss?${query}`,
                      {
                        signal:
                          controller.signal,
                      },
                    ),

                    adminFetch<FinanceProfitLossTrendResponse>(
                      `/finance/profit-loss/trend?${query}`,
                      {
                        signal:
                          controller.signal,
                      },
                    ),
                  ],
                );

              if (
                cancelled
              ) {
                return;
              }

              setData(
                (
                  current,
                ) => ({
                  ...current,

                  finance:
                    financeResponse.data,

                  financeTrend:
                    trendResponse.data ??
                    [],
                }),
              );
            } catch (
              value
            ) {
              if (
                cancelled
              ) {
                return;
              }

              setFinanceError(
                errorMessage(
                  value,
                ),
              );
            }
          };

        await Promise.all([
          analyticsTask(),
          financeTask(),
        ]);

        if (
          !cancelled
        ) {
          setLoading(
            false,
          );

          setRefreshing(
            false,
          );
        }
      }

      void load();

      return () => {
        cancelled =
          true;

        controller.abort();
      };
    },
    [
      canAnalyticsRead,
      canFinanceRead,
      periodDays,
      refreshKey,
    ],
  );

  const currency =
    data.finance
      ?.currency ??
    data.overview
      ?.currency ??
    "BDT";

  const generatedAt =
    data.finance
      ?.generated_at ??
    data.overview
      ?.generated_at ??
    null;

  const signals =
    data.insights
      ?.signals ??
    [];

  const visibleSignals =
    signals.slice(
      0,
      5,
    );

  const expenseCategories =
    useMemo(
      () =>
        [
          ...(
            data.finance
              ?.expense_categories ??
            []
          ),
        ].sort(
          (
            left,
            right,
          ) =>
            right.amount -
            left.amount,
        ),
      [
        data.finance,
      ],
    );

  if (
    !canAnalyticsRead &&
    !canFinanceRead
  ) {
    return (
      <main
        className={
          styles.workspace
        }
      >
        <AdminPageHeader
          eyebrow="Commerce intelligence"
          title="Commerce Intelligence"
          description="Your staff role does not include access to analytics or finance intelligence."
        />

        <section
          className={
            styles.permissionGlass
          }
        >
          <div
            className={
              styles.permissionGlow
            }
            aria-hidden="true"
          />

          <strong>
            Intelligence access is restricted
          </strong>

          <p>
            This workspace requires
            admin.analytics.read,
            admin.finance.read, or
            Super Admin access.
          </p>
        </section>
      </main>
    );
  }

  return (
    <main
      className={
        styles.workspace
      }
    >
      <CommerceIntelligenceHeader
        periodDays={periodDays}
        refreshing={refreshing}
        onPeriodChange={setPeriodDays}
        onRefresh={() =>
          setRefreshKey((value) => value + 1)
        }
      />

      <section
        className={
          styles.intelligenceGlass
        }
      >
        <div
          className={
            styles.intelligenceAmbient
          }
          aria-hidden="true"
        />

        <div
          className={
            styles.workspaceMeta
          }
        >
          <div>
            <span>
              Reporting currency
            </span>

            <strong>
              {currency}
            </strong>
          </div>

          <div>
            <span>
              Period
            </span>

            <strong>
              {periodDays} days
            </strong>
          </div>

          <div>
            <span>
              Updated
            </span>

            <strong>
              {generatedAt
                ? relativeTime(
                    generatedAt,
                  )
                : loading
                  ? "Loading"
                  : "Unavailable"}
            </strong>
          </div>

          {data.finance ? (
            <div>
              <span>
                Cost coverage
              </span>

              <strong
                className={
                  data.finance
                    .profit_complete
                    ? styles.metaKnown
                    : styles.metaPartial
                }
              >
                {formatPercentBPS(
                  data.finance
                    .cogs_coverage_bps,
                )}
              </strong>
            </div>
          ) : null}
        </div>

        <nav
          className={
            styles.modeRail
          }
          aria-label="Commerce intelligence views"
        >
          <button
            type="button"
            className={[
              styles.modeButton,
              mode ===
              "overview"
                ? styles.modeButtonActive
                : "",
            ]
              .filter(
                Boolean,
              )
              .join(
                " ",
              )}
            aria-current={
              mode ===
              "overview"
                ? "page"
                : undefined
            }
            onClick={() =>
              setMode(
                "overview",
              )
            }
          >
            Overview
          </button>

          {canFinanceRead ? (
            <button
              type="button"
              className={[
                styles.modeButton,
                mode ===
                "profit"
                  ? styles.modeButtonActive
                  : "",
              ]
                .filter(
                  Boolean,
                )
                .join(
                  " ",
                )}
              aria-current={
                mode ===
                "profit"
                  ? "page"
                  : undefined
              }
              onClick={() =>
                setMode(
                  "profit",
                )
              }
            >
              Profit
            </button>
          ) : null}
        </nav>
      </section>

      {analyticsError ||
      financeError ? (
        <section
          className={
            styles.errorGlass
          }
          role="alert"
        >
          <div>
            <strong>
              Some intelligence could not be loaded
            </strong>

            {analyticsError ? (
              <p>
                Analytics:{" "}
                {
                  analyticsError
                }
              </p>
            ) : null}

            {financeError ? (
              <p>
                Finance:{" "}
                {
                  financeError
                }
              </p>
            ) : null}
          </div>

          <button
            type="button"
            onClick={() =>
              setRefreshKey(
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
        </section>
      ) : null}

      {mode ===
      "overview" ? (
        <>
          <section
            className={
              styles.heroGrid
            }
          >
            {canFinanceRead &&
            data.finance ? (
              <ProfitSpine
                finance={
                  data.finance
                }
              />
            ) : (
              <section
                className={
                  styles.restrictedGlass
                }
              >
                <div
                  className={
                    styles.restrictedOrb
                  }
                  aria-hidden="true"
                />

                <span>
                  Profitability
                </span>

                <h2>
                  Finance data is
                  restricted for
                  this role.
                </h2>

                <p>
                  Commerce analytics
                  remain available,
                  but profitability
                  requires
                  admin.finance.read.
                </p>
              </section>
            )}

            <aside
              className={
                styles.signalRail
              }
            >
              <header
                className={
                  styles.railHeader
                }
              >
                <div>
                  <span>
                    Evidence
                  </span>

                  <h2>
                    What needs
                    attention
                  </h2>
                </div>

                <strong>
                  {
                    visibleSignals.length
                  }
                </strong>
              </header>

              {!canAnalyticsRead ? (
                <div
                  className={
                    styles.signalEmpty
                  }
                >
                  <strong>
                    Analytics signals are restricted
                  </strong>

                  <p>
                    This role can see
                    finance reporting,
                    but not business
                    insight signals.
                  </p>
                </div>
              ) : loading ? (
                <div
                  className={
                    styles.signalLoading
                  }
                  aria-busy="true"
                >
                  <span />
                  <span />
                  <span />
                </div>
              ) : visibleSignals.length >
                0 ? (
                <div
                  className={
                    styles.signalList
                  }
                >
                  {visibleSignals.map(
                    (
                      signal,
                      index,
                    ) => (
                      <article
                        key={`${signal.type}-${index}`}
                        className={[
                          styles.signalItem,
                          signalTone(
                            signal,
                          ),
                        ]
                          .filter(
                            Boolean,
                          )
                          .join(
                            " ",
                          )}
                      >
                        <div
                          className={
                            styles.signalTop
                          }
                        >
                          <span>
                            {signalLabel(
                              signal.severity,
                            )}
                          </span>

                          <i
                            aria-hidden="true"
                          />
                        </div>

                        <h3>
                          {
                            signal.title
                          }
                        </h3>

                        <p>
                          {
                            signal.summary
                          }
                        </p>

                        {signal.evidence
                          .length >
                        0 ? (
                          <dl
                            className={
                              styles.signalEvidence
                            }
                          >
                            {signal.evidence
                              .slice(
                                0,
                                3,
                              )
                              .map(
                                (
                                  item,
                                  evidenceIndex,
                                ) => (
                                  <div
                                    key={`${item.label}-${evidenceIndex}`}
                                  >
                                    <dt>
                                      {
                                        item.label
                                      }
                                    </dt>

                                    <dd>
                                      {
                                        item.value
                                      }
                                    </dd>
                                  </div>
                                ),
                              )}
                          </dl>
                        ) : null}

                        {signal.action
                          ?.type ===
                        "open_finance" ? (
                          <button
                            type="button"
                            className={
                              styles.signalAction
                            }
                            onClick={() =>
                              setMode(
                                "profit",
                              )
                            }
                          >
                            Review profit
                          </button>
                        ) : signal.action
                            ?.type ===
                          "open_inventory" ? (
                          <a
                            className={
                              styles.signalAction
                            }
                            href={`/${portal}/inventory`}
                          >
                            Open inventory
                          </a>
                        ) : null}
                      </article>
                    ),
                  )}
                </div>
              ) : (
                <div
                  className={
                    styles.signalEmpty
                  }
                >
                  <strong>
                    No material signals
                    in this period
                  </strong>

                  <p>
                    The deterministic
                    insight rules did
                    not surface a
                    business condition
                    that crosses their
                    current thresholds.
                  </p>
                </div>
              )}
            </aside>
          </section>

          <section
            className={
              styles.performanceGlass
            }
          >
            <header
              className={
                styles.sectionHeader
              }
            >
              <div>
                <span>
                  Movement
                </span>

                <h2>
                  Revenue and profit
                  through time
                </h2>

                <p>
                  Profit is drawn only
                  for periods with
                  complete cost
                  coverage.
                </p>
              </div>

              {data.finance ? (
                <div
                  className={
                    styles.sectionMetric
                  }
                >
                  <span>
                    Margin
                  </span>

                  <strong>
                    {formatPercentBPS(
                      data.finance
                        .gross_margin_bps,
                    )}
                  </strong>
                </div>
              ) : null}
            </header>

            {canFinanceRead ? (
              <FinanceTrend
                points={
                  data.financeTrend
                }
                currency={
                  currency
                }
              />
            ) : (
              <div
                className={
                  styles.emptyTrend
                }
              >
                <strong>
                  Finance trend is
                  restricted
                </strong>

                <p>
                  A finance-enabled
                  role can see revenue,
                  cost and profit move
                  together here.
                </p>
              </div>
            )}
          </section>

          <section
            className={
              styles.pulseStrip
            }
            aria-label="Commerce period summary"
          >
            <div>
              <span>
                Net collected
                revenue
              </span>

              <strong>
                {data.finance
                  ? formatMoney(
                      data.finance
                        .net_collected_revenue_amount,
                      currency,
                    )
                  : data.overview
                    ? formatMoney(
                        data.overview
                          .net_collected_revenue_amount,
                        currency,
                      )
                    : "—"}
              </strong>

              <small>
                After successful
                refunds
              </small>
            </div>

            <div>
              <span>
                Collected orders
              </span>

              <strong>
                {compactNumber(
                  data.finance
                    ?.collected_orders ??
                    data.overview
                      ?.collected_orders ??
                    0,
                )}
              </strong>

              <small>
                Paid commercial
                orders
              </small>
            </div>

            <div>
              <span>
                Refunds
              </span>

              <strong>
                {formatMoney(
                  data.finance
                    ?.successful_refund_amount ??
                    data.overview
                      ?.successful_refund_amount ??
                    0,
                  currency,
                )}
              </strong>

              <small>
                Successful refunds
                in period
              </small>
            </div>

            <div>
              <span>
                Average order
                value
              </span>

              <strong>
                {data.overview
                  ? formatMoney(
                      data.overview
                        .average_order_value_amount,
                      currency,
                    )
                  : "—"}
              </strong>

              <small>
                Analytics order
                average
              </small>
            </div>
          </section>
        </>
      ) : null}

      {mode ===
      "profit" ? (
        <section
          className={
            styles.profitWorkspace
          }
        >
          {data.finance ? (
            <>
              <ProfitSpine
                finance={
                  data.finance
                }
                compact
              />

              <div
                className={
                  styles.profitDetailGrid
                }
              >
                <section
                  className={
                    styles.ledgerGlass
                  }
                >
                  <header
                    className={
                      styles.sectionHeader
                    }
                  >
                    <div>
                      <span>
                        Revenue
                      </span>

                      <h2>
                        Commercial
                        waterfall
                      </h2>

                      <p>
                        The exact
                        reported
                        amounts behind
                        the profit
                        calculation.
                      </p>
                    </div>
                  </header>

                  <div
                    className={
                      styles.ledger
                    }
                  >
                    <div>
                      <span>
                        Gross
                        merchandise
                      </span>

                      <strong>
                        {formatMoney(
                          data.finance
                            .gross_merchandise_revenue_amount,
                          currency,
                        )}
                      </strong>
                    </div>

                    <div
                      className={
                        styles.ledgerDeduction
                      }
                    >
                      <span>
                        Discounts
                      </span>

                      <strong>
                        −
                        {formatMoney(
                          data.finance
                            .discount_amount,
                          currency,
                        )}
                      </strong>
                    </div>

                    <div>
                      <span>
                        Net merchandise
                      </span>

                      <strong>
                        {formatMoney(
                          data.finance
                            .net_merchandise_revenue_amount,
                          currency,
                        )}
                      </strong>
                    </div>

                    <div>
                      <span>
                        Shipping
                        collected
                      </span>

                      <strong>
                        +
                        {formatMoney(
                          data.finance
                            .shipping_revenue_amount,
                          currency,
                        )}
                      </strong>
                    </div>

                    <div>
                      <span>
                        Gross
                        collected
                      </span>

                      <strong>
                        {formatMoney(
                          data.finance
                            .gross_collected_revenue_amount,
                          currency,
                        )}
                      </strong>
                    </div>

                    <div
                      className={
                        styles.ledgerDeduction
                      }
                    >
                      <span>
                        Successful
                        refunds
                      </span>

                      <strong>
                        −
                        {formatMoney(
                          data.finance
                            .successful_refund_amount,
                          currency,
                        )}
                      </strong>
                    </div>

                    <div
                      className={
                        styles.ledgerTotal
                      }
                    >
                      <span>
                        Net collected
                        revenue
                      </span>

                      <strong>
                        {formatMoney(
                          data.finance
                            .net_collected_revenue_amount,
                          currency,
                        )}
                      </strong>
                    </div>
                  </div>
                </section>

                <section
                  className={
                    styles.coverageGlass
                  }
                >
                  <header
                    className={
                      styles.sectionHeader
                    }
                  >
                    <div>
                      <span>
                        Cost truth
                      </span>

                      <h2>
                        Buying-cost
                        coverage
                      </h2>

                      <p>
                        Missing costs
                        remain unknown;
                        they never
                        become zero.
                      </p>
                    </div>
                  </header>

                  <div
                    className={
                      styles.coverageNumber
                    }
                  >
                    <strong>
                      {formatPercentBPS(
                        data.finance
                          .cogs_coverage_bps,
                      )}
                    </strong>

                    <span>
                      of cost-sensitive
                      units covered
                    </span>
                  </div>

                  <div
                    className={
                      styles.largeCoverageTrack
                    }
                  >
                    <span
                      style={{
                        width: `${Math.max(
                          0,
                          Math.min(
                            100,
                            data.finance
                              .cogs_coverage_bps /
                              100,
                          ),
                        )}%`,
                      }}
                    />
                  </div>

                  <div
                    className={
                      styles.coverageStats
                    }
                  >
                    <div>
                      <span>
                        Costed units
                      </span>

                      <strong>
                        {compactNumber(
                          data.finance
                            .costed_units,
                        )}
                      </strong>
                    </div>

                    <div>
                      <span>
                        Missing costs
                      </span>

                      <strong>
                        {compactNumber(
                          data.finance
                            .missing_cost_units,
                        )}
                      </strong>
                    </div>

                    <div>
                      <span>
                        Return cost
                        gaps
                      </span>

                      <strong>
                        {compactNumber(
                          data.finance
                            .return_loss_missing_cost_units,
                        )}
                      </strong>
                    </div>
                  </div>

                  {data.finance
                    .warnings
                    .length >
                  0 ? (
                    <div
                      className={
                        styles.warningStack
                      }
                    >
                      {data.finance
                        .warnings
                        .map(
                          (
                            warning,
                            index,
                          ) => (
                            <p
                              key={`${warning}-${index}`}
                            >
                              {
                                warning
                              }
                            </p>
                          ),
                        )}
                    </div>
                  ) : null}
                </section>

                <section
                  className={
                    styles.expenseGlass
                  }
                >
                  <header
                    className={
                      styles.sectionHeader
                    }
                  >
                    <div>
                      <span>
                        Recorded
                        expenses
                      </span>

                      <h2>
                        Operating
                        expense mix
                      </h2>

                      <p>
                        Only entered
                        expenses are
                        represented
                        here.
                      </p>
                    </div>

                    <div
                      className={
                        styles.sectionMetric
                      }
                    >
                      <span>
                        Total
                      </span>

                      <strong>
                        {formatMoney(
                          data.finance
                            .recorded_expenses_amount,
                          currency,
                        )}
                      </strong>
                    </div>
                  </header>

                  {expenseCategories.length >
                  0 ? (
                    <div
                      className={
                        styles.expenseList
                      }
                    >
                      {expenseCategories.map(
                        (
                          item,
                        ) => {
                          const share =
                            data.finance &&
                            data.finance
                              .recorded_expenses_amount >
                              0
                              ? (
                                  item.amount /
                                  data.finance
                                    .recorded_expenses_amount
                                ) *
                                100
                              : 0;

                          return (
                            <div
                              key={
                                item.category
                              }
                              className={
                                styles.expenseRow
                              }
                            >
                              <div
                                className={
                                  styles.expenseRowHead
                                }
                              >
                                <div>
                                  <strong>
                                    {item.category
                                      .replace(
                                        /_/g,
                                        " ",
                                      )
                                      .replace(
                                        /\b\w/g,
                                        (
                                          value,
                                        ) =>
                                          value.toUpperCase(),
                                      )}
                                  </strong>

                                  <span>
                                    {
                                      item.entries
                                    }{" "}
                                    entries
                                  </span>
                                </div>

                                <strong>
                                  {formatMoney(
                                    item.amount,
                                    currency,
                                  )}
                                </strong>
                              </div>

                              <div
                                className={
                                  styles.expenseTrack
                                }
                              >
                                <span
                                  style={{
                                    width: `${Math.min(
                                      100,
                                      share,
                                    )}%`,
                                  }}
                                />
                              </div>
                            </div>
                          );
                        },
                      )}
                    </div>
                  ) : (
                    <div
                      className={
                        styles.quietState
                      }
                    >
                      <strong>
                        No recorded
                        operating
                        expenses
                      </strong>

                      <p>
                        Profitability
                        currently
                        reflects no
                        active expense
                        entries for
                        this period.
                      </p>
                    </div>
                  )}

                  {data.finance
                    .foreign_currency_expense_entries_excluded >
                  0 ? (
                    <div
                      className={
                        styles.currencyNotice
                      }
                    >
                      <strong>
                        {
                          data.finance
                            .foreign_currency_expense_entries_excluded
                        }{" "}
                        foreign-currency
                        expense entries
                        excluded
                      </strong>

                      <p>
                        They are not
                        converted or
                        mixed into the
                        {` ${currency} `}
                        result.
                      </p>
                    </div>
                  ) : null}
                </section>
              </div>
            </>
          ) : (
            <section
              className={
                styles.restrictedGlass
              }
            >
              <h2>
                Profit intelligence
                is unavailable.
              </h2>

              <p>
                Refresh the workspace
                or confirm that this
                role has
                admin.finance.read.
              </p>
            </section>
          )}
        </section>
      ) : null}
    </main>
  );
}