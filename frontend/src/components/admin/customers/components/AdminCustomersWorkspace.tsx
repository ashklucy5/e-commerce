"use client";

import {
  type FormEvent,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";

import { useAdminSession } from "@/components/admin/layout/context/AdminSessionContext";
import {
  AdminRequestError,
  adminFetch,
} from "@/lib/admin/api";
import type {
  AdminCustomerDetail,
  AdminCustomerListItem,
  AdminCustomerResponse,
  AdminCustomersResponse,
  AdminCustomerStatus,
  AdminPaginationMeta,
} from "@/lib/admin/customer-types";
import { formatMoney } from "@/lib/money/format";

import styles from "../css/AdminCustomers.module.css";

type Props = {
  portal: string;
};

type SummaryState = {
  total: number;
  active: number;
  disabled: number;
};

const PAGE_SIZE = 30;

function errorMessage(value: unknown): string {
  if (value instanceof AdminRequestError) return value.message;
  if (value instanceof Error) return value.message;
  return "Unable to complete the request.";
}

function formatDateTime(value?: string): string {
  if (!value) return "—";

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "—";

  return new Intl.DateTimeFormat("en", {
    month: "short",
    day: "numeric",
    year: "numeric",
    hour: "numeric",
    minute: "2-digit",
  }).format(date);
}

function titleCase(value: string): string {
  return value
    .replace(/_/g, " ")
    .replace(/\b\w/g, (character) => character.toUpperCase());
}

function customerInitials(name: string): string {
  const initials = name
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((part) => part.charAt(0))
    .join("")
    .toUpperCase();

  return initials || "CU";
}

function isSearchLikelyComplete(value: string): boolean {
  const query = value.trim();
  if (!query) return true;

  if (query.includes("@")) return query.includes(".");
  if (/^[0-9+\-()\s]+$/.test(query)) return query.replace(/\D/g, "").length >= 7;
  return query.length >= 30;
}

export default function AdminCustomersWorkspace({ portal }: Props) {
  const principal = useAdminSession();
  const isSuperAdmin = principal.staff.roles.includes("admin_superuser");
  const permissions = principal.staff.permissions;

  const hasPermission = useCallback(
    (permission: string) => isSuperAdmin || permissions.includes(permission),
    [isSuperAdmin, permissions],
  );

  const canRead = hasPermission("admin.customer.read");
  const canManage = hasPermission("admin.customer.manage");

  const [items, setItems] = useState<AdminCustomerListItem[]>([]);
  const [meta, setMeta] = useState<AdminPaginationMeta | null>(null);
  const [summary, setSummary] = useState<SummaryState>({
    total: 0,
    active: 0,
    disabled: 0,
  });

  const [queryInput, setQueryInput] = useState("");
  const [query, setQuery] = useState("");
  const [status, setStatus] = useState<"" | AdminCustomerStatus>("");
  const [page, setPage] = useState(1);

  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [detail, setDetail] = useState<AdminCustomerDetail | null>(null);

  const [loading, setLoading] = useState(true);
  const [detailLoading, setDetailLoading] = useState(false);
  const [mutating, setMutating] = useState(false);
  const [error, setError] = useState("");
  const [detailError, setDetailError] = useState("");

  const listAbortRef = useRef<AbortController | null>(null);

  const loadSummary = useCallback(async () => {
    if (!canRead) return;

    try {
      const [all, active, disabled] = await Promise.all([
        adminFetch<AdminCustomersResponse>("/customers?page=1&limit=1"),
        adminFetch<AdminCustomersResponse>("/customers?page=1&limit=1&status=active"),
        adminFetch<AdminCustomersResponse>("/customers?page=1&limit=1&status=disabled"),
      ]);

      setSummary({
        total: all.meta.total,
        active: active.meta.total,
        disabled: disabled.meta.total,
      });
    } catch {
      // Summary is helpful, but the main customer queue remains usable without it.
    }
  }, [canRead]);

  const loadCustomers = useCallback(async () => {
    if (!canRead) {
      setLoading(false);
      return;
    }

    listAbortRef.current?.abort();
    const controller = new AbortController();
    listAbortRef.current = controller;

    setLoading(true);
    setError("");

    const params = new URLSearchParams({
      page: String(page),
      limit: String(PAGE_SIZE),
    });

    if (status) params.set("status", status);
    if (query.trim()) params.set("q", query.trim());

    try {
      const response = await adminFetch<AdminCustomersResponse>(
        `/customers?${params.toString()}`,
        { signal: controller.signal },
      );

      if (controller.signal.aborted) return;
      setItems(response.data ?? []);
      setMeta(response.meta);
    } catch (value: unknown) {
      if (controller.signal.aborted) return;
      setError(errorMessage(value));
      setItems([]);
      setMeta(null);
    } finally {
      if (!controller.signal.aborted) setLoading(false);
    }
  }, [canRead, page, query, status]);

  const loadDetail = useCallback(async (customerId: string) => {
    if (!canRead) return;

    setDetailLoading(true);
    setDetailError("");

    try {
      const response = await adminFetch<AdminCustomerResponse>(
        `/customers/${customerId}`,
      );
      setDetail(response.data);
    } catch (value: unknown) {
      setDetailError(errorMessage(value));
      setDetail(null);
    } finally {
      setDetailLoading(false);
    }
  }, [canRead]);

  useEffect(() => {
    void loadCustomers();

    return () => {
      listAbortRef.current?.abort();
    };
  }, [loadCustomers]);

  useEffect(() => {
    void loadSummary();
  }, [loadSummary]);

  useEffect(() => {
    if (!selectedId) {
      setDetail(null);
      setDetailError("");
      return;
    }

    void loadDetail(selectedId);
  }, [loadDetail, selectedId]);

  useEffect(() => {
    const value = queryInput.trim();

    if (!value) {
      const timer = window.setTimeout(() => {
        setQuery("");
        setPage(1);
      }, 180);
      return () => window.clearTimeout(timer);
    }

    if (!isSearchLikelyComplete(value)) return;

    const timer = window.setTimeout(() => {
      setQuery(value);
      setPage(1);
    }, 300);

    return () => window.clearTimeout(timer);
  }, [queryInput]);

  const displayedSummary = useMemo(() => [
    {
      label: "Customers",
      value: summary.total,
      copy: "Registered customer accounts",
      tone: "blue",
    },
    {
      label: "Active",
      value: summary.active,
      copy: "Can currently access their account",
      tone: "green",
    },
    {
      label: "Disabled",
      value: summary.disabled,
      copy: "Access disabled by operations",
      tone: "gold",
    },
  ], [summary]);

  function submitSearch(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setQuery(queryInput.trim());
    setPage(1);
  }

  function openCustomer(customerId: string) {
    setSelectedId(customerId);
  }

  function closeDrawer() {
    if (mutating) return;
    setSelectedId(null);
  }

  async function changeCustomerStatus(nextStatus: AdminCustomerStatus) {
    if (!detail || !canManage || detail.status === nextStatus) return;

    if (
      nextStatus === "disabled" &&
      !window.confirm(
        "Disable this customer account? Active customer sessions will be revoked immediately.",
      )
    ) {
      return;
    }

    setMutating(true);
    setDetailError("");

    try {
      const response = await adminFetch<AdminCustomerResponse>(
        `/customers/${detail.id}/status`,
        {
          method: "PATCH",
          body: JSON.stringify({ status: nextStatus }),
        },
      );

      setDetail(response.data);
      setItems((current) =>
        current.map((item) =>
          item.id === response.data.id ? response.data : item,
        ),
      );

      await loadSummary();
    } catch (value: unknown) {
      setDetailError(errorMessage(value));
    } finally {
      setMutating(false);
    }
  }

  if (!canRead) {
    return (
      <main className={styles.page}>
        <section className={styles.permissionCard}>
          <span>Customers</span>
          <h1>Customer access is restricted</h1>
          <p>
            Your staff account does not have the admin.customer.read permission.
          </p>
        </section>
      </main>
    );
  }

  return (
    <main className={styles.page}>
      <header className={styles.hero}>
        <div>
          <p className={styles.eyebrow}>Customer operations</p>
          <h1>Customers</h1>
          <p className={styles.heroCopy}>
            Find customer accounts, inspect order activity and saved addresses,
            and manage account access from one operational view.
          </p>
        </div>

        <button
          type="button"
          className={styles.refreshButton}
          onClick={() => {
            void loadCustomers();
            void loadSummary();
            if (selectedId) void loadDetail(selectedId);
          }}
          disabled={loading || detailLoading}
        >
          Refresh
        </button>
      </header>

      <section className={styles.summaryGrid} aria-label="Customer summary">
        {displayedSummary.map((card) => (
          <article
            key={card.label}
            className={`${styles.summaryCard} ${styles[card.tone]}`}
          >
            <span>{card.label}</span>
            <strong>{card.value.toLocaleString()}</strong>
            <small>{card.copy}</small>
          </article>
        ))}
      </section>

      <section className={styles.queueCard}>
        <div className={styles.queueHeader}>
          <div>
            <p className={styles.sectionEyebrow}>Customer directory</p>
            <h2>{meta ? `${meta.total.toLocaleString()} matching customers` : "Customers"}</h2>
          </div>
        </div>

        <form className={styles.filters} onSubmit={submitSearch}>
          <label className={styles.searchField}>
            <span className={styles.srOnly}>Search customers</span>
            <input
              value={queryInput}
              onChange={(event) => setQueryInput(event.target.value)}
              placeholder="Phone, email or customer ID..."
              autoComplete="off"
            />
          </label>

          <label className={styles.selectField}>
            <span className={styles.srOnly}>Customer status</span>
            <select
              value={status}
              onChange={(event) => {
                setStatus(event.target.value as "" | AdminCustomerStatus);
                setPage(1);
              }}
            >
              <option value="">All statuses</option>
              <option value="active">Active</option>
              <option value="disabled">Disabled</option>
            </select>
          </label>

          <button type="submit" className={styles.searchButton}>
            Search
          </button>
        </form>

        {queryInput.trim() && !isSearchLikelyComplete(queryInput) ? (
          <p className={styles.searchHint}>
            Customer lookup uses an exact phone, email or customer ID.
          </p>
        ) : null}

        {error ? <div className={styles.errorBanner}>{error}</div> : null}

        <div className={styles.tableWrap}>
          <table className={styles.table}>
            <thead>
              <tr>
                <th>Customer</th>
                <th>Status</th>
                <th>Orders</th>
                <th>Sessions</th>
                <th>Last order</th>
                <th>Joined</th>
              </tr>
            </thead>
            <tbody>
              {items.map((customer) => (
                <tr
                  key={customer.id}
                  onClick={() => openCustomer(customer.id)}
                  tabIndex={0}
                  onKeyDown={(event) => {
                    if (event.key === "Enter" || event.key === " ") {
                      event.preventDefault();
                      openCustomer(customer.id);
                    }
                  }}
                >
                  <td>
                    <div className={styles.customerCell}>
                      <span className={styles.avatar}>{customerInitials(customer.full_name)}</span>
                      <span>
                        <strong>{customer.full_name || "Unnamed customer"}</strong>
                        <small>{customer.phone}</small>
                        {customer.email ? <small>{customer.email}</small> : null}
                      </span>
                    </div>
                  </td>
                  <td>
                    <span className={`${styles.statusPill} ${customer.status === "active" ? styles.active : styles.disabled}`}>
                      {titleCase(customer.status)}
                    </span>
                  </td>
                  <td>
                    <strong className={styles.numberValue}>{customer.order_count}</strong>
                    <small className={styles.cellMeta}>{customer.review_count} reviews</small>
                  </td>
                  <td>
                    <strong className={styles.numberValue}>{customer.active_session_count}</strong>
                    <small className={styles.cellMeta}>active</small>
                  </td>
                  <td>{formatDateTime(customer.last_order_at)}</td>
                  <td>{formatDateTime(customer.created_at)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        <div className={styles.mobileList}>
          {items.map((customer) => (
            <button
              type="button"
              key={customer.id}
              className={styles.mobileCard}
              onClick={() => openCustomer(customer.id)}
            >
              <div className={styles.mobileCardTop}>
                <div className={styles.customerCell}>
                  <span className={styles.avatar}>{customerInitials(customer.full_name)}</span>
                  <span>
                    <strong>{customer.full_name || "Unnamed customer"}</strong>
                    <small>{customer.phone}</small>
                  </span>
                </div>
                <span className={`${styles.statusPill} ${customer.status === "active" ? styles.active : styles.disabled}`}>
                  {titleCase(customer.status)}
                </span>
              </div>

              <div className={styles.mobileStats}>
                <span><b>{customer.order_count}</b> Orders</span>
                <span><b>{customer.active_session_count}</b> Sessions</span>
                <span><b>{customer.review_count}</b> Reviews</span>
              </div>

              <div className={styles.mobileFooter}>
                <span>Last order</span>
                <strong>{formatDateTime(customer.last_order_at)}</strong>
              </div>
            </button>
          ))}
        </div>

        {!loading && items.length === 0 ? (
          <div className={styles.emptyState}>
            <strong>No customers found</strong>
            <span>Try a different status or exact customer lookup.</span>
          </div>
        ) : null}

        {loading ? (
          <div className={styles.loadingState}>Loading customer directory…</div>
        ) : null}

        {meta && meta.total_pages > 1 ? (
          <footer className={styles.pagination}>
            <span>
              Page {meta.page.toLocaleString()} of {meta.total_pages.toLocaleString()}
            </span>
            <div>
              <button
                type="button"
                disabled={!meta.has_previous || loading}
                onClick={() => setPage((current) => Math.max(1, current - 1))}
              >
                Previous
              </button>
              <button
                type="button"
                disabled={!meta.has_next || loading}
                onClick={() => setPage((current) => current + 1)}
              >
                Next
              </button>
            </div>
          </footer>
        ) : null}
      </section>

      {selectedId ? (
        <div className={styles.drawerLayer} role="presentation" onMouseDown={(event) => {
          if (event.target === event.currentTarget) closeDrawer();
        }}>
          <aside className={styles.drawer} aria-label="Customer details">
            <header className={styles.drawerHeader}>
              <div>
                <p className={styles.sectionEyebrow}>Customer profile</p>
                <h2>{detail?.full_name ?? "Customer"}</h2>
              </div>
              <button type="button" onClick={closeDrawer} aria-label="Close customer details">×</button>
            </header>

            {detailError ? <div className={styles.errorBanner}>{detailError}</div> : null}

            {detailLoading && !detail ? (
              <div className={styles.drawerLoading}>Loading customer…</div>
            ) : null}

            {detail ? (
              <div className={styles.drawerBody}>
                <section className={styles.profileHero}>
                  <div className={styles.profileIdentity}>
                    <span className={styles.largeAvatar}>{customerInitials(detail.full_name)}</span>
                    <div>
                      <div className={styles.profileNameLine}>
                        <h3>{detail.full_name || "Unnamed customer"}</h3>
                        <span className={`${styles.statusPill} ${detail.status === "active" ? styles.active : styles.disabled}`}>
                          {titleCase(detail.status)}
                        </span>
                      </div>
                      <p>{detail.phone}</p>
                      <p>{detail.email || "No email on file"}</p>
                    </div>
                  </div>

                  {canManage ? (
                    <div className={styles.statusActions}>
                      {detail.status === "active" ? (
                        <button
                          type="button"
                          className={styles.dangerButton}
                          disabled={mutating}
                          onClick={() => void changeCustomerStatus("disabled")}
                        >
                          Disable account
                        </button>
                      ) : (
                        <button
                          type="button"
                          className={styles.primaryButton}
                          disabled={mutating}
                          onClick={() => void changeCustomerStatus("active")}
                        >
                          Enable account
                        </button>
                      )}
                    </div>
                  ) : null}
                </section>

                <section className={styles.metricGrid}>
                  <article>
                    <span>Orders</span>
                    <strong>{detail.order_count.toLocaleString()}</strong>
                  </article>
                  <article>
                    <span>Delivered</span>
                    <strong>{detail.delivered_or_completed_orders.toLocaleString()}</strong>
                  </article>
                  <article>
                    <span>Reviews</span>
                    <strong>{detail.review_count.toLocaleString()}</strong>
                  </article>
                  <article>
                    <span>Active sessions</span>
                    <strong>{detail.active_session_count.toLocaleString()}</strong>
                  </article>
                </section>

                <section className={styles.panel}>
                  <div className={styles.panelHeader}>
                    <div>
                      <p className={styles.sectionEyebrow}>Recent commerce</p>
                      <h3>Recent orders</h3>
                    </div>
                    <a href={`/${portal}/orders`} className={styles.textLink}>Open orders →</a>
                  </div>

                  <div className={styles.orderList}>
                    {detail.recent_orders.length ? detail.recent_orders.map((order) => (
                      <article key={order.id} className={styles.orderRow}>
                        <div>
                          <strong>{order.order_number}</strong>
                          <span>{formatDateTime(order.created_at)}</span>
                        </div>
                        <div className={styles.orderStatus}>
                          <span>{titleCase(order.status)}</span>
                          <small>{titleCase(order.payment_status)}</small>
                        </div>
                        <strong>{formatMoney(order.total_amount, order.currency)}</strong>
                      </article>
                    )) : (
                      <div className={styles.inlineEmpty}>No orders yet.</div>
                    )}
                  </div>
                </section>

                <section className={styles.panel}>
                  <div className={styles.panelHeader}>
                    <div>
                      <p className={styles.sectionEyebrow}>Delivery profile</p>
                      <h3>Saved addresses</h3>
                    </div>
                  </div>

                  <div className={styles.addressGrid}>
                    {detail.addresses.length ? detail.addresses.map((address) => (
                      <article key={address.id} className={styles.addressCard}>
                        <div className={styles.addressTitle}>
                          <strong>{address.label || "Address"}</strong>
                          {address.is_default ? <span>Default</span> : null}
                        </div>
                        <p>{address.recipient_name}</p>
                        <p>{address.phone}</p>
                        <p>{address.address_line1}</p>
                        {address.address_line2 ? <p>{address.address_line2}</p> : null}
                        <p>{[address.area, address.city, address.postal_code].filter(Boolean).join(", ")}</p>
                      </article>
                    )) : (
                      <div className={styles.inlineEmpty}>No saved addresses.</div>
                    )}
                  </div>
                </section>

                <section className={styles.accountMeta}>
                  <div>
                    <span>Customer ID</span>
                    <strong>{detail.id}</strong>
                  </div>
                  <div>
                    <span>Joined</span>
                    <strong>{formatDateTime(detail.created_at)}</strong>
                  </div>
                  <div>
                    <span>Last updated</span>
                    <strong>{formatDateTime(detail.updated_at)}</strong>
                  </div>
                </section>
              </div>
            ) : null}
          </aside>
        </div>
      ) : null}
    </main>
  );
}
