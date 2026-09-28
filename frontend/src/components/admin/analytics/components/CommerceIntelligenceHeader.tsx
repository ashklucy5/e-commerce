"use client";

import AdminPageHeader from "@/components/admin/layout/components/AdminPageHeader";

import styles from "../css/CommerceIntelligence.module.css";

type PeriodDays = 7 | 30 | 90;

type Props = {
  periodDays: PeriodDays;
  refreshing: boolean;
  onPeriodChange: (period: PeriodDays) => void;
  onRefresh: () => void;
};

const PERIODS: Array<{
  value: PeriodDays;
  label: string;
}> = [
  {
    value: 7,
    label: "7 days",
  },
  {
    value: 30,
    label: "30 days",
  },
  {
    value: 90,
    label: "90 days",
  },
];

function Controls({
  periodDays,
  refreshing,
  onPeriodChange,
  onRefresh,
}: Props) {
  return (
    <div className={styles.headerActions}>
      <div
        className={styles.periodControl}
        aria-label="Intelligence period"
      >
        {PERIODS.map((period) => (
          <button
            key={period.value}
            type="button"
            className={[
              styles.periodButton,
              periodDays === period.value
                ? styles.periodButtonActive
                : "",
            ]
              .filter(Boolean)
              .join(" ")}
            aria-pressed={periodDays === period.value}
            onClick={() => onPeriodChange(period.value)}
          >
            {period.label}
          </button>
        ))}
      </div>

      <button
        type="button"
        className={styles.refreshButton}
        disabled={refreshing}
        onClick={onRefresh}
      >
        <span
          className={
            refreshing
              ? styles.refreshGlyphActive
              : styles.refreshGlyph
          }
          aria-hidden="true"
        >
          ↻
        </span>

        {refreshing ? "Refreshing" : "Refresh"}
      </button>
    </div>
  );
}

export default function CommerceIntelligenceHeader({
  periodDays,
  refreshing,
  onPeriodChange,
  onRefresh,
}: Props) {
  return (
    <>
      <div className={styles.mobileCommerceHeader}>
        <AdminPageHeader
          eyebrow="Commerce intelligence"
          title="Commerce Intelligence"
          description="Revenue, profitability, cost truth and decision signals in one evidence-led workspace."
          actions={
            <Controls
              periodDays={periodDays}
              refreshing={refreshing}
              onPeriodChange={onPeriodChange}
              onRefresh={onRefresh}
            />
          }
        />
      </div>

      <header className={styles.desktopCommerceHeader}>
        <div className={styles.desktopCommerceCopy}>
          <h1>Commerce Intelligence</h1>

          <p>
            Revenue, profitability, cost truth and decision signals
            in one evidence-led workspace.
          </p>
        </div>

        <Controls
          periodDays={periodDays}
          refreshing={refreshing}
          onPeriodChange={onPeriodChange}
          onRefresh={onRefresh}
        />
      </header>
    </>
  );
}