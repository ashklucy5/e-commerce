import { formatMoney } from "@/lib/money/format";
import type {
  FinanceProfitLoss,
} from "@/lib/admin/commerce-intelligence-types";

import styles from "../css/CommerceIntelligence.module.css";

type Props = {
  finance: FinanceProfitLoss;
  compact?: boolean;
};

function formatPercentBPS(
  value: number | null | undefined,
): string {
  if (typeof value !== "number") {
    return "—";
  }

  return `${(value / 100).toFixed(1)}%`;
}

function compactNumber(
  value: number,
): string {
  return new Intl.NumberFormat("en", {
    notation:
      Math.abs(value) >= 10_000
        ? "compact"
        : "standard",

    maximumFractionDigits: 1,
  }).format(value);
}

function Amount({
  value,
  currency,
  muted = false,
}: {
  value: number;
  currency: string;
  muted?: boolean;
}) {
  return (
    <strong
      className={[
        styles.spineAmount,
        muted ? styles.spineAmountMuted : "",
      ]
        .filter(Boolean)
        .join(" ")}
    >
      {formatMoney(
        value,
        currency,
      )}
    </strong>
  );
}

export default function ProfitSpine({
  finance,
  compact = false,
}: Props) {
  const coverage =
    Math.max(
      0,
      Math.min(
        100,
        finance.cogs_coverage_bps / 100,
      ),
    );

  const profitLabel =
    finance.profit_complete
      ? "Gross profit"
      : "Known gross profit";

  const profitValue =
    finance.profit_complete &&
    finance.gross_profit_amount != null
      ? finance.gross_profit_amount
      : finance.known_gross_profit_amount;

  const finalProfit =
    finance.net_profit_after_recorded_expenses_amount;

  return (
    <section
      className={[
        styles.profitSpine,
        compact ? styles.profitSpineCompact : "",
      ]
        .filter(Boolean)
        .join(" ")}
      aria-label="Profit calculation"
    >
      <div
        className={styles.spineLiquidLight}
        aria-hidden="true"
      />

      <header className={styles.spineHeader}>
        <div>
          <p className={styles.contextText}>
            Profit truth
          </p>

          <h2>
            Follow the money through the business
          </h2>

          <p className={styles.spineIntro}>
            Revenue becomes meaningful only when
            refunds, buying costs and recorded
            operating expenses are accounted for.
          </p>
        </div>

        <div
          className={[
            styles.coverageBadge,
            finance.profit_complete
              ? styles.coverageBadgeComplete
              : styles.coverageBadgePartial,
          ].join(" ")}
        >
          <span>
            {formatPercentBPS(
              finance.cogs_coverage_bps,
            )}
          </span>

          <small>
            COGS coverage
          </small>
        </div>
      </header>

      <div className={styles.spineFlow}>
        <div className={styles.spineResult}>
          <div>
            <span className={styles.spineLabel}>
              Gross collected revenue
            </span>

            <small>
              Merchandise after discounts, plus
              shipping collected
            </small>
          </div>

          <Amount
            value={
              finance.gross_collected_revenue_amount
            }
            currency={finance.currency}
          />
        </div>

        <div className={styles.spineConnector}>
          <span
            className={styles.connectorLine}
            aria-hidden="true"
          />

          <div
            className={styles.connectorDeduction}
          >
            <span>Successful refunds</span>

            <strong>
              −
              {formatMoney(
                finance.successful_refund_amount,
                finance.currency,
              )}
            </strong>
          </div>
        </div>

        <div className={styles.spineResult}>
          <div>
            <span className={styles.spineLabel}>
              Net collected revenue
            </span>

            <small>
              Collected revenue after successful
              refunds
            </small>
          </div>

          <Amount
            value={
              finance.net_collected_revenue_amount
            }
            currency={finance.currency}
          />
        </div>

        <div className={styles.spineConnector}>
          <span
            className={[
              styles.connectorLine,
              !finance.profit_complete
                ? styles.connectorLinePartial
                : "",
            ]
              .filter(Boolean)
              .join(" ")}
            aria-hidden="true"
          />

          <div
            className={styles.connectorDeduction}
          >
            <span>
              {finance.profit_complete
                ? "Net COGS"
                : "Known net COGS"}
            </span>

            <strong>
              −
              {formatMoney(
                finance.net_cogs_amount,
                finance.currency,
              )}
            </strong>
          </div>
        </div>

        {!finance.profit_complete ? (
          <div
            className={styles.coverageInterruption}
          >
            <div
              className={
                styles.coverageInterruptionHead
              }
            >
              <span>
                Cost truth is incomplete
              </span>

              <strong>
                {compactNumber(
                  finance.missing_cost_units,
                )}{" "}
                units unresolved
              </strong>
            </div>

            <div
              className={styles.coverageTrack}
              aria-label={`COGS coverage ${coverage.toFixed(
                1,
              )}%`}
            >
              <span
                style={{
                  width: `${coverage}%`,
                }}
              />
            </div>

            <p>
              Missing buying costs are not treated
              as zero. Final gross and net profit
              stay unavailable until cost coverage
              is complete.
            </p>
          </div>
        ) : null}

        <div
          className={[
            styles.spineResult,
            styles.spineResultProfit,
          ].join(" ")}
        >
          <div>
            <span className={styles.spineLabel}>
              {profitLabel}
            </span>

            <small>
              {finance.profit_complete
                ? `Gross margin ${formatPercentBPS(
                    finance.gross_margin_bps,
                  )}`
                : "Before unresolved buying costs"}
            </small>
          </div>

          <Amount
            value={profitValue}
            currency={finance.currency}
          />
        </div>

        <div className={styles.spineConnector}>
          <span
            className={styles.connectorLine}
            aria-hidden="true"
          />

          <div
            className={styles.connectorDeduction}
          >
            <span>
              Recorded operating expenses
            </span>

            <strong>
              −
              {formatMoney(
                finance.recorded_expenses_amount,
                finance.currency,
              )}
            </strong>
          </div>
        </div>

        <div
          className={[
            styles.spineResult,
            styles.spineFinal,
            finalProfit != null &&
            finalProfit < 0
              ? styles.spineFinalNegative
              : "",
          ]
            .filter(Boolean)
            .join(" ")}
        >
          <div>
            <span className={styles.spineLabel}>
              Profit after recorded expenses
            </span>

            <small>
              {finalProfit != null
                ? "Based on recorded operating expenses"
                : "Waiting for complete COGS coverage"}
            </small>
          </div>

          {finalProfit != null ? (
            <Amount
              value={finalProfit}
              currency={finance.currency}
            />
          ) : (
            <strong
              className={styles.spineUnavailable}
            >
              Not calculated
            </strong>
          )}
        </div>
      </div>

      <footer className={styles.spineFooter}>
        <span>
          {compactNumber(
            finance.collected_orders,
          )}{" "}
          collected orders
        </span>

        <span>
          {compactNumber(
            finance.units_sold,
          )}{" "}
          units sold
        </span>

        <span>
          {compactNumber(
            finance.recorded_expense_entries,
          )}{" "}
          recorded expense entries
        </span>
      </footer>
    </section>
  );
}