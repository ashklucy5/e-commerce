"use client";

import {
  type FormEvent,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";

import AdminShipmentControlDrawer from "@/components/admin/shipments/components/AdminShipmentControlDrawer";
import AdminPageHeader from "@/components/admin/layout/components/AdminPageHeader";
import { useAdminSession } from "@/components/admin/layout/context/AdminSessionContext";
import { adminFetch, AdminRequestError } from "@/lib/admin/api";
import type {
  AdminShipmentQueueItem,
  AdminShipmentQueueResponse,
  AdminShipmentSummary,
  AdminShipmentSummaryResponse,
} from "@/lib/admin/shipment-types";
import { formatMoney } from "@/lib/money/format";

import styles from "../css/AdminShipments.module.css";

const PAGE_LIMIT = 40;
const SEARCH_DELAY_MS = 280;
const LIVE_SYNC_MS = 5_000;

const STATUS_OPTIONS = [
  ["", "All shipment work"],
  ["not_prepared", "Needs shipment"],
  ["pending", "Prepared / handoff"],
  ["shipped", "In transit"],
  ["awaiting_confirmation", "Awaiting confirmation"],
  ["delivered", "Delivered"],
  ["cancelled", "Cancelled"],
] as const;

const MODE_OPTIONS = [
  ["", "All modes"],
  ["courier", "Courier"],
  ["community_rider", "Community rider"],
  ["self_pickup", "Self pickup"],
] as const;

type Props = {
  portal: string;
  initialQuery?: string;
};

function titleCase(value: string): string {
  return value
    .replace(/_/g, " ")
    .replace(/\b\w/g, (character) => character.toUpperCase());
}

function formatDateTime(value?: string): string {
  if (!value) return "—";

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "—";

  return new Intl.DateTimeFormat("en", {
    month: "short",
    day: "numeric",
    hour: "numeric",
    minute: "2-digit",
  }).format(date);
}

function errorMessage(value: unknown): string {
  if (value instanceof AdminRequestError) return value.message;
  if (value instanceof Error && value.name !== "AbortError") return value.message;
  return "Shipments could not be loaded.";
}

function statusTone(status: string): string {
  switch (status) {
    case "delivered":
      return styles.positive;
    case "shipped":
      return styles.info;
    case "awaiting_confirmation":
    case "pending":
    case "not_prepared":
      return styles.attention;
    case "cancelled":
      return styles.danger;
    default:
      return styles.neutral;
  }
}


function shipmentStageLabel(item: AdminShipmentQueueItem): string {
  switch (item.shipment_status) {
    case "not_prepared":
      return "Not prepared";
    case "pending":
      return "Prepared";
    case "shipped":
      return "Shipped";
    case "awaiting_confirmation":
      return "Awaiting confirmation";
    case "delivered":
      return "Delivered";
    case "cancelled":
      return "Cancelled";
    default:
      return titleCase(item.shipment_status);
  }
}

function shipmentActionLabel(item: AdminShipmentQueueItem): string {
  switch (item.shipment_status) {
    case "not_prepared":
      return "Prepare";
    case "pending":
      return "Continue";
    case "shipped":
      return "Update";
    case "awaiting_confirmation":
      return "Confirm";
    case "delivered":
    case "cancelled":
      return "View";
    default:
      return "Open";
  }
}

function rowCarrier(item: AdminShipmentQueueItem): string {
  if (item.courier_name) return item.courier_name;
  if (item.provider_code) return item.provider_code;
  if (item.delivery_mode) return titleCase(item.delivery_mode);
  return "Not prepared";
}

function rowReference(item: AdminShipmentQueueItem): string {
  return (
    item.tracking_number ||
    item.courier_reference ||
    item.rider_reference ||
    item.provider_shipment_id ||
    "—"
  );
}

export default function AdminShipmentsWorkspace({
  portal,
  initialQuery = "",
}: Props) {
  const principal = useAdminSession();
  const canRead = principal.staff.permissions.includes("admin.delivery.read");

  const [items, setItems] = useState<AdminShipmentQueueItem[]>([]);
  const [summary, setSummary] = useState<AdminShipmentSummary | null>(null);
  const [meta, setMeta] = useState<AdminShipmentQueueResponse["meta"] | null>(null);
  const [page, setPage] = useState(1);
  const [query, setQuery] = useState(initialQuery);
  const [appliedQuery, setAppliedQuery] = useState(initialQuery);
  const [status, setStatus] = useState("");
  const [deliveryMode, setDeliveryMode] = useState("");
  const [selectedOrderId, setSelectedOrderId] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [refreshKey, setRefreshKey] = useState(0);
  const [lastSyncedAt, setLastSyncedAt] = useState<Date | null>(null);
  const requestSequence = useRef(0);

  const loadSummary = useCallback(async () => {
    if (!canRead) return;

    try {
      const response = await adminFetch<AdminShipmentSummaryResponse>(
        "/delivery/shipments/summary",
      );
      setSummary(response.data);
    } catch {
      // Queue remains usable even if the summary call is temporarily unavailable.
    }
  }, [canRead]);

  const loadShipments = useCallback(
    async (
      nextPage: number,
      nextQuery: string,
      nextStatus: string,
      nextMode: string,
      signal?: AbortSignal,
      options?: { silent?: boolean },
    ) => {
      if (!canRead) {
        setLoading(false);
        return;
      }

      const sequence = ++requestSequence.current;
      const params = new URLSearchParams({
        page: String(nextPage),
        limit: String(PAGE_LIMIT),
      });

      const normalizedQuery = nextQuery.trim();
      if (normalizedQuery) params.set("q", normalizedQuery);
      if (nextStatus) params.set("status", nextStatus);
      if (nextMode) params.set("delivery_mode", nextMode);

      if (!options?.silent) {
        setLoading(true);
      }
      setError("");

      try {
        const response = await adminFetch<AdminShipmentQueueResponse>(
          `/delivery/shipments?${params.toString()}`,
          { signal },
        );

        if (sequence !== requestSequence.current) return;

        setItems(response.data ?? []);
        setMeta(response.meta ?? null);
        setPage(nextPage);
        setAppliedQuery(normalizedQuery);
        setLastSyncedAt(new Date());
      } catch (value: unknown) {
        if (value instanceof DOMException && value.name === "AbortError") return;
        if (sequence === requestSequence.current) setError(errorMessage(value));
      } finally {
        if (sequence === requestSequence.current && !options?.silent) {
          setLoading(false);
        }
      }
    },
    [canRead],
  );

  useEffect(() => {
    void loadSummary();
  }, [loadSummary, refreshKey]);

  useEffect(() => {
    if (!canRead) return;

    const controller = new AbortController();
    const timer = window.setTimeout(() => {
      void loadShipments(1, query, status, deliveryMode, controller.signal);
    }, SEARCH_DELAY_MS);

    return () => {
      window.clearTimeout(timer);
      controller.abort();
    };
  }, [canRead, deliveryMode, loadShipments, query, refreshKey, status]);

  useEffect(() => {
    if (!canRead) return;

    let disposed = false;

    const sync = () => {
      if (disposed || document.visibilityState !== "visible") return;

      void loadSummary();
      void loadShipments(
        page,
        appliedQuery,
        status,
        deliveryMode,
        undefined,
        { silent: true },
      );
    };

    const intervalId = window.setInterval(sync, LIVE_SYNC_MS);
    const handleVisibility = () => {
      if (document.visibilityState === "visible") sync();
    };

    document.addEventListener("visibilitychange", handleVisibility);

    return () => {
      disposed = true;
      window.clearInterval(intervalId);
      document.removeEventListener("visibilitychange", handleVisibility);
    };
  }, [
    appliedQuery,
    canRead,
    deliveryMode,
    loadShipments,
    loadSummary,
    page,
    status,
  ]);

  const activeCount = useMemo(
    () =>
      (summary?.needs_shipment ?? 0) +
      (summary?.pending ?? 0) +
      (summary?.shipped ?? 0) +
      (summary?.awaiting_confirmation ?? 0),
    [summary],
  );

  function submitSearch(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    void loadShipments(1, query, status, deliveryMode);
  }

  function refresh() {
    setRefreshKey((value) => value + 1);
  }

  function handleUpdated() {
    refresh();
  }

  if (!canRead) {
    return (
      <div className={styles.page}>
        <AdminPageHeader
          eyebrow="Delivery operations"
          title="Shipments"
          description="Your staff role does not include permission to view shipment operations."
        />
        <section className={styles.permissionPanel}>
          <strong>Shipment access is restricted</strong>
          <p>This workspace requires the admin.delivery.read permission.</p>
        </section>
      </div>
    );
  }

  return (
    <div className={styles.page}>
      <AdminPageHeader
        eyebrow="Delivery operations"
        title="Shipments"
        description="Search orders and shipments, then control dispatch, tracking, provider delivery and receipt confirmation from one queue."
        actions={
          <button type="button" className={styles.refreshButton} onClick={refresh} disabled={loading}>
            <span aria-hidden="true">↻</span>
            Refresh
          </button>
        }
      />

      <section className={styles.summaryGrid} aria-label="Shipment summary">
        <button type="button" className={`${styles.summaryCard} ${styles.summaryBlue}`} onClick={() => setStatus("")}>
          <span>Active delivery work</span>
          <strong>{summary ? activeCount : "—"}</strong>
          <small>Needs shipment through receipt confirmation</small>
        </button>
        <button type="button" className={`${styles.summaryCard} ${styles.summaryGold}`} onClick={() => setStatus("not_prepared")}>
          <span>Needs shipment</span>
          <strong>{summary?.needs_shipment ?? "—"}</strong>
          <small>Processing orders without a shipment</small>
        </button>
        <button type="button" className={`${styles.summaryCard} ${styles.summaryViolet}`} onClick={() => setStatus("pending")}>
          <span>Prepared</span>
          <strong>{summary?.pending ?? "—"}</strong>
          <small>Prepared and waiting for handoff / dispatch</small>
        </button>
        <button type="button" className={`${styles.summaryCard} ${styles.summarySky}`} onClick={() => setStatus("shipped")}>
          <span>Shipped</span>
          <strong>{summary?.shipped ?? "—"}</strong>
          <small>Dispatched and in delivery</small>
        </button>
        <button type="button" className={`${styles.summaryCard} ${styles.summaryViolet}`} onClick={() => setStatus("awaiting_confirmation")}>
          <span>Awaiting confirmation</span>
          <strong>{summary?.awaiting_confirmation ?? "—"}</strong>
          <small>Provider delivered, receipt pending</small>
        </button>
        <button type="button" className={`${styles.summaryCard} ${styles.summaryEmerald}`} onClick={() => setStatus("delivered")}>
          <span>Delivered</span>
          <strong>{summary?.delivered ?? "—"}</strong>
          <small>Receipt confirmed</small>
        </button>
      </section>

      <section className={styles.queuePanel}>
        <div className={styles.queueHeader}>
          <div>
            <span className={styles.eyebrow}>Shipment queue</span>
            <h2>{meta?.total ?? (loading ? "—" : items.length)} matching records</h2>
          </div>
          <div className={styles.queueMeta}>
            <span className={styles.liveMeta}>
              <span className={styles.liveDot} aria-hidden="true" />
              Live{lastSyncedAt ? ` · ${formatDateTime(lastSyncedAt.toISOString())}` : ""}
            </span>
            {appliedQuery ? <span className={styles.queryBadge}>Search: {appliedQuery}</span> : null}
          </div>
        </div>

        <form className={styles.filters} onSubmit={submitSearch}>
          <label className={styles.searchField}>
            <span aria-hidden="true">⌕</span>
            <input
              value={query}
              onChange={(event) => setQuery(event.target.value)}
              placeholder="Order, customer, phone, tracking, courier reference…"
              maxLength={100}
              autoComplete="off"
            />
            {query ? (
              <button type="button" onClick={() => setQuery("")} aria-label="Clear shipment search">×</button>
            ) : null}
          </label>

          <select value={status} onChange={(event) => setStatus(event.target.value)} aria-label="Shipment status">
            {STATUS_OPTIONS.map(([value, label]) => <option key={value} value={value}>{label}</option>)}
          </select>

          <select value={deliveryMode} onChange={(event) => setDeliveryMode(event.target.value)} aria-label="Delivery mode">
            {MODE_OPTIONS.map(([value, label]) => <option key={value} value={value}>{label}</option>)}
          </select>

          <button type="submit" className={styles.searchButton}>Search</button>
        </form>

        {error ? <div className={styles.errorBanner} role="alert">{error}</div> : null}

        <div className={styles.tableWrap}>
          <div className={styles.tableHeader} aria-hidden="true">
            <span>Order / customer</span>
            <span>Shipment</span>
            <span>Current stage</span>
            <span>Destination</span>
            <span>Last update</span>
            <span>Total</span>
            <span>Action</span>
          </div>

          <div className={styles.rows}>
            {loading && items.length === 0
              ? Array.from({ length: 7 }, (_, index) => <div key={index} className={styles.loadingRow}><span /><span /><span /><span /><span /></div>)
              : items.map((item) => (
                  <article
                    key={`${item.order_id}:${item.shipment_id ?? "unprepared"}`}
                    className={styles.row}
                    role="button"
                    tabIndex={0}
                    onClick={() => setSelectedOrderId(item.order_id)}
                    onKeyDown={(event) => {
                      if (event.key === "Enter" || event.key === " ") {
                        event.preventDefault();
                        setSelectedOrderId(item.order_id);
                      }
                    }}
                  >
                    <span className={styles.orderCell}>
                      <strong>{item.order_number}</strong>
                      <small>{item.customer_name || "Customer"}</small>
                      <small>{item.customer_phone}</small>
                    </span>

                    <span className={styles.shipmentCell}>
                      <strong>{rowCarrier(item)}</strong>
                      <small>{rowReference(item)}</small>
                      {item.delivery_mode ? <small>{titleCase(item.delivery_mode)}</small> : null}
                    </span>

                    <span className={styles.stageCell}>
                      <span className={`${styles.statusPill} ${statusTone(item.shipment_status)}`}>
                        {shipmentStageLabel(item)}
                      </span>
                      {item.latest_event_code ? (
                        <small>{titleCase(item.latest_event_code)}</small>
                      ) : (
                        <small>{titleCase(item.order_status)}</small>
                      )}
                    </span>

                    <span className={styles.destinationCell}>
                      <strong>{item.shipping_area || item.shipping_city || "—"}</strong>
                      <small>{item.shipping_city}</small>
                      {item.warehouse_name ? <small>From {item.warehouse_name}</small> : null}
                    </span>

                    <span className={styles.updateCell}>
                      <strong>{formatDateTime(item.latest_event_at || item.updated_at)}</strong>
                      <small>{item.latest_event_message || item.provider_status || "No delivery event yet"}</small>
                    </span>

                    <span className={styles.moneyCell}>
                      <strong>{formatMoney(item.total_amount, item.currency)}</strong>
                      <small>{titleCase(item.payment_status)}</small>
                    </span>

                    <span className={styles.actionCell}>
                      <button
                        type="button"
                        className={`${styles.rowAction} ${
                          item.shipment_status === "delivered" ||
                          item.shipment_status === "cancelled"
                            ? styles.rowActionQuiet
                            : ""
                        }`}
                        onClick={(event) => {
                          event.stopPropagation();
                          setSelectedOrderId(item.order_id);
                        }}
                      >
                        {shipmentActionLabel(item)} →
                      </button>
                    </span>
                  </article>
                ))}

            {!loading && items.length === 0 ? (
              <div className={styles.emptyState}>
                <strong>No shipment work matches this view</strong>
                <p>Try another status, delivery mode, order number, phone number or tracking reference.</p>
              </div>
            ) : null}
          </div>
        </div>

        {meta && meta.total_pages > 1 ? (
          <div className={styles.pagination}>
            <button type="button" disabled={!meta.has_previous || loading} onClick={() => void loadShipments(page - 1, appliedQuery, status, deliveryMode)}>Previous</button>
            <span>Page {meta.page} of {meta.total_pages}</span>
            <button type="button" disabled={!meta.has_next || loading} onClick={() => void loadShipments(page + 1, appliedQuery, status, deliveryMode)}>Next</button>
          </div>
        ) : null}
      </section>

      <AdminShipmentControlDrawer
        orderId={selectedOrderId}
        onClose={() => setSelectedOrderId(null)}
        onUpdated={handleUpdated}
      />
    </div>
  );
}
