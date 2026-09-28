"use client";

import {
  type FormEvent,
  useCallback,
  useEffect,
  useMemo,
  useState,
} from "react";

import AdminPageHeader from "@/components/admin/layout/components/AdminPageHeader";
import { useAdminSession } from "@/components/admin/layout/context/AdminSessionContext";
import {
  adminFetch,
  AdminRequestError,
} from "@/lib/admin/api";
import type {
  AdminOrderListItem,
  AdminOrdersResponse,
  AdminPaginationMeta,
} from "@/lib/admin/order-types";
import { formatMoney } from "@/lib/money/format";

import AdminOrderDrawer from "./AdminOrderDrawer";
import styles from "../css/AdminOrders.module.css";

const PAGE_LIMIT = 25;

const ORDER_STATUS_OPTIONS = [
  ["", "All orders"],
  ["pending_payment", "Pending payment"],
  ["confirmed", "Confirmed"],
  ["processing", "Processing"],
  ["shipped", "Shipped"],
  ["delivered", "Delivered"],
  ["completed", "Completed"],
  ["payment_expired", "Payment expired"],
  ["cancelled", "Cancelled"],
] as const;

const FULFILLMENT_STATUS_OPTIONS = [
  ["confirmed", "Ready to process"],
  ["processing", "Processing"],
  ["shipped", "Shipped"],
  ["delivered", "Delivered"],
] as const;

const PAYMENT_STATUS_OPTIONS = [
  ["", "All payments"],
  ["paid", "Paid"],
  ["pending", "Pending"],
  ["cod_pending", "COD pending"],
  ["cod_collected", "COD collected"],
  ["refunded", "Refunded"],
  ["failed", "Failed"],
  ["expired", "Expired"],
] as const;

type WorkspaceMode = "orders" | "fulfillment";

type AdminOrdersWorkspaceProps = {
  portal: string;
  mode: WorkspaceMode;
};

function titleCase(value: string): string {
  return value
    .replace(/_/g, " ")
    .replace(/\b\w/g, (character) => character.toUpperCase());
}

function formatDateTime(value: string): string {
  const date = new Date(value);

  if (Number.isNaN(date.getTime())) {
    return "—";
  }

  return new Intl.DateTimeFormat("en", {
    month: "short",
    day: "numeric",
    year: "numeric",
    hour: "numeric",
    minute: "2-digit",
  }).format(date);
}

function statusTone(status: string): string {
  switch (status) {
    case "completed":
    case "delivered":
      return styles.tonePositive;

    case "confirmed":
    case "processing":
    case "shipped":
      return styles.toneInfo;

    case "awaiting_procurement":
    case "pending_payment":
    case "cod_pending":
      return styles.toneAttention;

    case "cancelled":
    case "payment_expired":
    case "failed":
    case "expired":
      return styles.toneDanger;

    default:
      return styles.toneNeutral;
  }
}

function paymentTone(status: string): string {
  switch (status) {
    case "paid":
    case "cod_collected":
      return styles.tonePositive;

    case "pending":
    case "cod_pending":
      return styles.toneAttention;

    case "refunded":
      return styles.toneViolet;

    case "failed":
    case "expired":
      return styles.toneDanger;

    default:
      return styles.toneNeutral;
  }
}

function errorMessage(value: unknown): string {
  if (value instanceof AdminRequestError) {
    return value.message;
  }

  if (value instanceof Error) {
    return value.message;
  }

  return "Orders could not be loaded.";
}

function loadingRows(count = 7) {
  return Array.from({ length: count }, (_, index) => (
    <div
      key={index}
      className={styles.loadingRow}
      aria-hidden="true"
    >
      <span />
      <span />
      <span />
      <span />
      <span />
    </div>
  ));
}

export default function AdminOrdersWorkspace({
  portal,
  mode,
}: AdminOrdersWorkspaceProps) {
  const principal = useAdminSession();

  const canReadOrders = principal.staff.permissions.includes(
    "admin.order.read",
  );

  const defaultStatus = mode === "fulfillment" ? "confirmed" : "";

  const [orders, setOrders] = useState<AdminOrderListItem[]>([]);
  const [meta, setMeta] = useState<AdminPaginationMeta | null>(null);
  const [query, setQuery] = useState("");
  const [appliedQuery, setAppliedQuery] = useState("");
  const [status, setStatus] = useState(defaultStatus);
  const [paymentStatus, setPaymentStatus] = useState("");
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [selectedOrderId, setSelectedOrderId] = useState<string | null>(null);
  const [refreshKey, setRefreshKey] = useState(0);

  const statusOptions =
    mode === "fulfillment"
      ? FULFILLMENT_STATUS_OPTIONS
      : ORDER_STATUS_OPTIONS;

  const loadOrders = useCallback(
    async (
      nextPage: number,
      nextStatus: string,
      nextPaymentStatus: string,
      nextQuery: string,
    ) => {
      if (!canReadOrders) {
        setLoading(false);
        return;
      }

      setLoading(true);
      setError("");

      const params = new URLSearchParams({
        page: String(nextPage),
        limit: String(PAGE_LIMIT),
      });

      const normalizedQuery = nextQuery.trim();

      if (nextStatus) {
        params.set("status", nextStatus);
      }

      if (nextPaymentStatus) {
        params.set("payment_status", nextPaymentStatus);
      }

      if (normalizedQuery) {
        params.set("q", normalizedQuery);
      }

      try {
        const response = await adminFetch<AdminOrdersResponse>(
          `/orders?${params.toString()}`,
        );

        setOrders(response.data ?? []);
        setMeta(response.meta ?? null);
        setPage(nextPage);
        setAppliedQuery(normalizedQuery);
      } catch (value: unknown) {
        setError(errorMessage(value));
      } finally {
        setLoading(false);
      }
    },
    [canReadOrders],
  );

  useEffect(() => {
    void loadOrders(1, defaultStatus, "", "");
  }, [defaultStatus, loadOrders, refreshKey]);

  const pagePaidTotal = useMemo(
    () =>
      orders
        .filter((order) =>
          order.payment_status === "paid" ||
          order.payment_status === "cod_collected",
        )
        .reduce((sum, order) => sum + order.total_amount, 0),
    [orders],
  );

  const pageAttentionCount = useMemo(
    () =>
      orders.filter((order) =>
        [
          "pending_payment",
          "confirmed",
          "processing",
          "awaiting_procurement",
        ].includes(order.status),
      ).length,
    [orders],
  );

  const pageCurrency = orders[0]?.currency ?? "BDT";

  function applyStatus(nextStatus: string) {
    setStatus(nextStatus);
    void loadOrders(1, nextStatus, paymentStatus, appliedQuery);
  }

  function applyPaymentStatus(nextStatus: string) {
    setPaymentStatus(nextStatus);
    void loadOrders(1, status, nextStatus, appliedQuery);
  }

  function submitSearch(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    void loadOrders(1, status, paymentStatus, query);
  }

  function clearSearch() {
    setQuery("");
    void loadOrders(1, status, paymentStatus, "");
  }

  function changePage(nextPage: number) {
    void loadOrders(nextPage, status, paymentStatus, appliedQuery);
  }

  function refreshOrders() {
    setRefreshKey((value) => value + 1);
  }

  if (!canReadOrders) {
    return (
      <div className={styles.page}>
        <AdminPageHeader
          eyebrow="Commerce operations"
          title={mode === "fulfillment" ? "Fulfillment" : "Orders"}
          description="Your staff role does not include permission to view customer orders."
        />

        <section className={styles.permissionPanel}>
          <div className={styles.permissionIcon}>i</div>
          <div>
            <strong>Order access is restricted</strong>
            <p>
              Ask an authorized Administrator to grant the order-read permission if this workspace is required for your role.
            </p>
          </div>
        </section>
      </div>
    );
  }

  return (
    <div className={styles.page}>
      <AdminPageHeader
        eyebrow={mode === "fulfillment" ? "Warehouse & delivery" : "Commerce operations"}
        title={mode === "fulfillment" ? "Fulfillment" : "Orders"}
        description={
          mode === "fulfillment"
            ? "Move confirmed orders through processing, warehouse preparation, shipment and receipt confirmation."
            : "Search, review and operate every standard and sourcing order from one live workspace."
        }
        actions={
          <button
            type="button"
            className={styles.refreshButton}
            onClick={refreshOrders}
            disabled={loading}
          >
            <span className={styles.refreshGlyph} aria-hidden="true">↻</span>
            Refresh
          </button>
        }
      />

      <section className={styles.summaryGrid} aria-label="Current order view summary">
        <article className={`${styles.summaryCard} ${styles.summaryBlue}`}>
          <span className={styles.summaryLabel}>Matching orders</span>
          <strong>{meta?.total ?? (loading ? "—" : orders.length)}</strong>
          <span className={styles.summaryMeta}>
            {status ? titleCase(status) : "Across all order states"}
          </span>
        </article>

        <article className={`${styles.summaryCard} ${styles.summaryEmerald}`}>
          <span className={styles.summaryLabel}>Collected on page</span>
          <strong>{formatMoney(pagePaidTotal, pageCurrency)}</strong>
          <span className={styles.summaryMeta}>Paid + collected COD</span>
        </article>

        <article className={`${styles.summaryCard} ${styles.summaryGold}`}>
          <span className={styles.summaryLabel}>Needs attention</span>
          <strong>{loading ? "—" : pageAttentionCount}</strong>
          <span className={styles.summaryMeta}>Visible operational orders</span>
        </article>
      </section>

      <section className={styles.controlPanel}>
        <div className={styles.statusScroller} role="tablist" aria-label="Order status">
          {statusOptions.map(([value, label]) => (
            <button
              key={value || "all"}
              type="button"
              role="tab"
              aria-selected={status === value}
              className={`${styles.statusTab} ${status === value ? styles.statusTabActive : ""}`}
              onClick={() => applyStatus(value)}
              disabled={loading && status !== value}
            >
              {label}
            </button>
          ))}
        </div>

        <div className={styles.filtersRow}>
          <form className={styles.searchForm} onSubmit={submitSearch}>
            <label className={styles.searchField}>
              <span className={styles.fieldLabel}>Find an order</span>
              <span className={styles.searchInputWrap}>
                <span className={styles.searchIcon} aria-hidden="true">⌕</span>
                <input
                  value={query}
                  onChange={(event) => setQuery(event.target.value)}
                  placeholder="Order number or customer phone"
                  maxLength={100}
                  autoComplete="off"
                />

                {query ? (
                  <button
                    type="button"
                    className={styles.clearSearch}
                    onClick={clearSearch}
                    aria-label="Clear order search"
                  >
                    ×
                  </button>
                ) : null}
              </span>
            </label>

            <button
              type="submit"
              className={styles.searchButton}
              disabled={loading}
            >
              Search
            </button>
          </form>

          {mode === "orders" ? (
            <label className={styles.paymentField}>
              <span className={styles.fieldLabel}>Payment</span>
              <select
                value={paymentStatus}
                onChange={(event) => applyPaymentStatus(event.target.value)}
                disabled={loading}
              >
                {PAYMENT_STATUS_OPTIONS.map(([value, label]) => (
                  <option key={value || "all"} value={value}>
                    {label}
                  </option>
                ))}
              </select>
            </label>
          ) : null}

          <div className={styles.liveMeta}>
            <span className={loading ? styles.liveDotLoading : styles.liveDot} />
            <span>
              {loading
                ? "Updating…"
                : `${orders.length} shown · page ${meta?.page ?? page}`}
            </span>
          </div>
        </div>
      </section>

      {error ? (
        <section className={styles.errorPanel} role="alert">
          <div>
            <strong>Orders could not be updated</strong>
            <span>{error}</span>
          </div>
          <button
            type="button"
            onClick={() => void loadOrders(page, status, paymentStatus, appliedQuery)}
          >
            Retry
          </button>
        </section>
      ) : null}

      <section className={styles.ordersPanel} aria-busy={loading}>
        <div className={styles.panelHeading}>
          <div>
            <span className={styles.panelEyebrow}>
              {mode === "fulfillment" ? "Operational queue" : "Live order ledger"}
            </span>
            <h2>
              {status
                ? titleCase(status)
                : "Recent orders"}
            </h2>
          </div>

          {appliedQuery ? (
            <span className={styles.queryChip}>
              Search: {appliedQuery}
            </span>
          ) : null}
        </div>

        {loading && orders.length === 0 ? (
          <div className={styles.loadingTable}>
            {loadingRows()}
          </div>
        ) : orders.length === 0 ? (
          <div className={styles.emptyState}>
            <div className={styles.emptyOrb} aria-hidden="true" />
            <strong>No matching orders</strong>
            <p>
              Try another status, payment filter or search term. New orders will appear here automatically when refreshed.
            </p>
          </div>
        ) : (
          <>
            <div className={styles.desktopTableWrap}>
              <table className={styles.ordersTable}>
                <thead>
                  <tr>
                    <th>Order</th>
                    <th>Customer</th>
                    <th>Status</th>
                    <th>Payment</th>
                    <th>Total</th>
                    <th>Placed</th>
                    <th aria-label="Open order" />
                  </tr>
                </thead>
                <tbody>
                  {orders.map((order) => (
                    <tr
                      key={order.id}
                      className={styles.orderRow}
                      onClick={() => setSelectedOrderId(order.id)}
                    >
                      <td>
                        <button
                          type="button"
                          className={styles.orderNumberButton}
                          onClick={(event) => {
                            event.stopPropagation();
                            setSelectedOrderId(order.id);
                          }}
                        >
                          {order.order_number}
                        </button>
                        <span className={styles.orderSubline}>
                          {titleCase(order.delivery_method || "delivery")}
                        </span>
                      </td>
                      <td>
                        <strong className={styles.customerName}>
                          {order.customer_name || "Guest customer"}
                        </strong>
                        <span className={styles.orderSubline}>{order.customer_phone || "—"}</span>
                      </td>
                      <td>
                        <span className={`${styles.statusPill} ${statusTone(order.status)}`}>
                          {titleCase(order.status)}
                        </span>
                      </td>
                      <td>
                        <span className={`${styles.statusPill} ${paymentTone(order.payment_status)}`}>
                          {titleCase(order.payment_status)}
                        </span>
                        <span className={styles.orderSubline}>{titleCase(order.payment_method)}</span>
                      </td>
                      <td className={styles.moneyCell}>
                        {formatMoney(order.total_amount, order.currency)}
                      </td>
                      <td>
                        <span className={styles.dateCell}>{formatDateTime(order.created_at)}</span>
                      </td>
                      <td>
                        <button
                          type="button"
                          className={styles.openButton}
                          aria-label={`Open ${order.order_number}`}
                          onClick={(event) => {
                            event.stopPropagation();
                            setSelectedOrderId(order.id);
                          }}
                        >
                          →
                        </button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>

            <div className={styles.mobileList}>
              {orders.map((order) => (
                <button
                  key={order.id}
                  type="button"
                  className={styles.mobileOrderCard}
                  onClick={() => setSelectedOrderId(order.id)}
                >
                  <span className={styles.mobileOrderTop}>
                    <span>
                      <strong>{order.order_number}</strong>
                      <small>{formatDateTime(order.created_at)}</small>
                    </span>
                    <strong className={styles.mobileMoney}>
                      {formatMoney(order.total_amount, order.currency)}
                    </strong>
                  </span>

                  <span className={styles.mobileCustomer}>
                    <strong>{order.customer_name || "Guest customer"}</strong>
                    <small>{order.customer_phone || "No phone"}</small>
                  </span>

                  <span className={styles.mobilePills}>
                    <span className={`${styles.statusPill} ${statusTone(order.status)}`}>
                      {titleCase(order.status)}
                    </span>
                    <span className={`${styles.statusPill} ${paymentTone(order.payment_status)}`}>
                      {titleCase(order.payment_status)}
                    </span>
                  </span>
                </button>
              ))}
            </div>
          </>
        )}

        {meta && meta.total_pages > 1 ? (
          <footer className={styles.pagination}>
            <button
              type="button"
              onClick={() => changePage(Math.max(1, meta.page - 1))}
              disabled={!meta.has_previous || loading}
            >
              ← Previous
            </button>

            <span>
              Page <strong>{meta.page}</strong> of <strong>{meta.total_pages}</strong>
            </span>

            <button
              type="button"
              onClick={() => changePage(Math.min(meta.total_pages, meta.page + 1))}
              disabled={!meta.has_next || loading}
            >
              Next →
            </button>
          </footer>
        ) : null}
      </section>

      <AdminOrderDrawer
        orderId={selectedOrderId}
        portal={portal}
        mode={mode}
        onClose={() => setSelectedOrderId(null)}
        onUpdated={refreshOrders}
      />
    </div>
  );
}
