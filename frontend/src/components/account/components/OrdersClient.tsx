// Location: src/components/account/components/OrdersClient.tsx
"use client";

import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { type ChangeEvent, useCallback, useEffect, useMemo, useState } from "react";

import { Icon } from "@/components/ui/Icon";
import type {
  AccountOrderHistoryItem,
  AccountOrderHistoryResponse,
  AccountPaginationMeta,
} from "@/lib/api/contracts/account";
import type { AccountOrderResponse } from "@/lib/api/contracts/order-account";
import { formatMoney } from "@/lib/money/format";
import {
  formatOrderDate,
  humanizeOrderValue,
  isDeliveredOrder,
  orderMatchesFilter,
  orderStatusTone,
  type OrderFilter,
} from "@/lib/orders/presentation";

import styles from "../css/Orders.module.css";

type LoadState = "loading" | "ready" | "error" | "signed-out";

type AccountOrderHistoryCard = AccountOrderHistoryItem & {
  /** Enriched by the storefront orders-list BFF from the public catalog. */
  image_url?: string;
  product_slug?: string;
};

type AccountOrderHistoryCardResponse = Omit<AccountOrderHistoryResponse, "data"> & {
  data: AccountOrderHistoryCard[];
};

type Notice = {
  kind: "success" | "warning" | "error";
  message: string;
} | null;

const FILTERS: Array<{ key: OrderFilter; label: string }> = [
  { key: "all", label: "All" },
  { key: "processing", label: "Processing" },
  { key: "transit", label: "In transit" },
  { key: "delivered", label: "Delivered" },
];

function apiMessage(payload: unknown, fallback: string) {
  if (!payload || typeof payload !== "object") return fallback;
  const root = payload as Record<string, unknown>;
  const error = root.error;

  if (typeof error === "string" && error.trim()) return error;
  if (error && typeof error === "object") {
    const message = (error as Record<string, unknown>).message;
    if (typeof message === "string" && message.trim()) return message;
  }

  if (typeof root.message === "string" && root.message.trim()) return root.message;
  return fallback;
}

async function readError(response: Response, fallback: string) {
  try {
    return apiMessage(await response.json(), fallback);
  } catch {
    return fallback;
  }
}

function queryMatches(order: AccountOrderHistoryCard, value: string) {
  const query = value.trim().toLowerCase();
  if (!query) return true;

  const searchable = [
    order.order_number,
    order.first_product_name ?? "",
    humanizeOrderValue(order.status),
    humanizeOrderValue(order.payment_status),
    humanizeOrderValue(order.payment_method),
    humanizeOrderValue(order.delivery_method),
  ]
    .join(" ")
    .toLowerCase();

  return searchable.includes(query);
}

function hideBrokenOrderImage(event: { currentTarget: HTMLImageElement }) {
  event.currentTarget.style.display = "none";
}

function statusClass(status: string) {
  switch (orderStatusTone(status)) {
    case "delivered":
      return styles.statusDelivered;
    case "warning":
      return styles.statusWarning;
    case "danger":
      return styles.statusDanger;
    case "neutral":
      return styles.statusNeutral;
    default:
      return styles.statusActive;
  }
}

async function addOrderToCart(orderId: string) {
  const orderResponse = await fetch(
    `/api/storefront/account/orders/${encodeURIComponent(orderId)}`,
    { cache: "no-store" },
  );

  if (!orderResponse.ok) {
    throw Object.assign(
      new Error(await readError(orderResponse, "Unable to reload this order.")),
      { status: orderResponse.status },
    );
  }

  const payload = (await orderResponse.json()) as AccountOrderResponse;
  let added = 0;
  let failed = 0;

  for (const item of payload.data.items) {
    const response = await fetch("/api/storefront/cart/items", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        variant_id: item.variant_id,
        quantity: item.quantity,
      }),
    });

    if (response.ok) added += 1;
    else failed += 1;
  }

  return { added, failed, total: payload.data.items.length };
}

export function OrdersClient() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const initialFilter = searchParams.get("status") as OrderFilter | null;

  const [state, setState] = useState<LoadState>("loading");
  const [orders, setOrders] = useState<AccountOrderHistoryCard[]>([]);
  const [meta, setMeta] = useState<AccountPaginationMeta | null>(null);
  const [error, setError] = useState("");
  const [query, setQuery] = useState("");
  const [filter, setFilter] = useState<OrderFilter>(
    initialFilter && FILTERS.some((item) => item.key === initialFilter)
      ? initialFilter
      : "all",
  );
  const [loadingMore, setLoadingMore] = useState(false);
  const [busyOrderId, setBusyOrderId] = useState("");
  const [notice, setNotice] = useState<Notice>(null);

  const loadPage = useCallback(async (page: number, append: boolean) => {
    const response = await fetch(
      `/api/storefront/account/orders?page=${page}&limit=100`,
      { cache: "no-store" },
    );

    if (response.status === 401) {
      setState("signed-out");
      return;
    }

    if (!response.ok) {
      throw new Error(await readError(response, "Unable to load your orders."));
    }

    const payload = (await response.json()) as AccountOrderHistoryCardResponse;
    setOrders((current) => (append ? [...current, ...payload.data] : payload.data));
    setMeta(payload.meta);
    setState("ready");
  }, []);

  const loadInitial = useCallback(async () => {
    setState("loading");
    setError("");

    try {
      await loadPage(1, false);
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : "Unable to load your orders.");
      setState("error");
    }
  }, [loadPage]);

  useEffect(() => {
    const timer = window.setTimeout(() => {
      void loadInitial();
    }, 0);

    return () => window.clearTimeout(timer);
  }, [loadInitial]);

  useEffect(() => {
    if (state !== "signed-out") return;
    router.replace("/account/sign-in?next=%2Faccount%2Forders");
  }, [router, state]);

  const counts = useMemo(() => {
    const result: Record<OrderFilter, number> = {
      all: orders.length,
      processing: 0,
      transit: 0,
      delivered: 0,
    };

    for (const order of orders) {
      if (orderMatchesFilter(order.status, "processing")) result.processing += 1;
      if (orderMatchesFilter(order.status, "transit")) result.transit += 1;
      if (orderMatchesFilter(order.status, "delivered")) result.delivered += 1;
    }

    return result;
  }, [orders]);

  const visibleOrders = useMemo(
    () =>
      orders.filter(
        (order) => orderMatchesFilter(order.status, filter) && queryMatches(order, query),
      ),
    [filter, orders, query],
  );

  const hasMore = Boolean(meta && meta.page < meta.total_pages);

  async function loadMore() {
    if (!meta || loadingMore || !hasMore) return;
    setLoadingMore(true);

    try {
      await loadPage(meta.page + 1, true);
    } catch (caught) {
      setNotice({
        kind: "error",
        message: caught instanceof Error ? caught.message : "Unable to load more orders.",
      });
    } finally {
      setLoadingMore(false);
    }
  }

  async function buyAgain(orderId: string) {
    if (busyOrderId) return;
    setBusyOrderId(orderId);
    setNotice(null);

    try {
      const result = await addOrderToCart(orderId);

      if (result.total === 0 || result.added === 0) {
        setNotice({
          kind: "error",
          message: "These order items cannot be added to the cart right now.",
        });
        return;
      }

      if (result.failed > 0) {
        setNotice({
          kind: "warning",
          message: `${result.added} item${result.added === 1 ? "" : "s"} added. ${result.failed} item${result.failed === 1 ? " is" : "s are"} no longer available in the original quantity.`,
        });
        return;
      }

      router.push("/cart");
    } catch (caught) {
      const errorWithStatus = caught as Error & { status?: number };
      if (errorWithStatus.status === 401) {
        router.replace("/account/sign-in?next=%2Faccount%2Forders");
        return;
      }

      setNotice({
        kind: "error",
        message: caught instanceof Error ? caught.message : "Unable to buy this order again.",
      });
    } finally {
      setBusyOrderId("");
    }
  }

  if (state === "loading" || state === "signed-out") {
    return (
      <main className={styles.page} aria-busy="true">
        <Ambient />
        <div className={styles.shell}>
          <div className={`${styles.glassPanel} ${styles.heroSkeleton}`} />
          <div className={styles.listSkeleton}>
            <div /><div /><div />
          </div>
        </div>
      </main>
    );
  }

  if (state === "error") {
    return (
      <main className={styles.page}>
        <Ambient />
        <section className={`${styles.glassPanel} ${styles.stateCard}`} role="alert">
          <span className={styles.stateIcon}><Icon name="support" size={24} /></span>
          <h1>Orders are temporarily unavailable.</h1>
          <p>{error}</p>
          <button type="button" className={styles.primaryButton} onClick={() => void loadInitial()}>
            Try again
          </button>
        </section>
      </main>
    );
  }

  return (
    <main className={styles.page}>
      <Ambient />
      <div className={styles.shell}>
        <Link className={styles.backLink} href="/account">
          <Icon name="chevronLeft" size={14} /> Account overview
        </Link>

        <header className={`${styles.glassPanel} ${styles.hero}`}>
          <div className={styles.heroCopy}>
            <span className={styles.eyebrow}>Purchase history</span>
            <h1>Your orders</h1>
            <p>Find an order, follow its status, open the invoice or place the same items back in your cart.</p>
          </div>
          <div className={styles.heroActions}>
            <span className={styles.heroIcon}><Icon name="orders" size={23} /></span>
            <Link className={styles.secondaryButton} href="/account/support">
              <Icon name="support" size={16} /> Order help
            </Link>
          </div>
        </header>

        <section className={`${styles.glassPanel} ${styles.toolbar}`} aria-label="Order search and filters">
          <label className={styles.searchBox}>
            <Icon name="search" size={17} />
            <span className={styles.srOnly}>Search orders</span>
            <input
              type="search"
              value={query}
              placeholder="Search order number, product or status"
              onChange={(event: ChangeEvent<HTMLInputElement>) => setQuery(event.target.value)}
            />
          </label>

          <div className={styles.filters} role="group" aria-label="Filter orders">
            {FILTERS.map((item) => (
              <button
                key={item.key}
                type="button"
                className={filter === item.key ? styles.filterActive : styles.filterButton}
                aria-pressed={filter === item.key}
                onClick={() => setFilter(item.key)}
              >
                {item.label}
                <span>{counts[item.key]}</span>
              </button>
            ))}
          </div>
        </section>

        <div className={styles.resultSummary} aria-live="polite">
          <span>
            {visibleOrders.length.toLocaleString()} matching order{visibleOrders.length === 1 ? "" : "s"}
            {meta ? ` · ${orders.length.toLocaleString()} of ${meta.total.toLocaleString()} loaded` : ""}
          </span>
          {query ? (
            <button type="button" onClick={() => setQuery("")}>Clear search</button>
          ) : null}
        </div>

        {notice ? (
          <div className={`${styles.notice} ${styles[`notice${notice.kind[0].toUpperCase()}${notice.kind.slice(1)}`]}`} role="status">
            <span>{notice.message}</span>
            {notice.kind === "warning" ? <Link href="/cart">Open cart</Link> : null}
            <button type="button" aria-label="Dismiss message" onClick={() => setNotice(null)}>×</button>
          </div>
        ) : null}

        {visibleOrders.length ? (
          <section className={styles.orderList} aria-label="Orders">
            {visibleOrders.map((order) => (
              <article key={order.id} className={`${styles.glassPanel} ${styles.orderCard}`}>
                <div className={styles.cardTop}>
                  <div className={styles.orderIdentity}>
                    <span className={styles.orderGlyph}><Icon name="orders" size={18} /></span>
                    <div>
                      <span>Order</span>
                      <strong>#{order.order_number}</strong>
                    </div>
                    <span className={`${styles.statusChip} ${statusClass(order.status)}`}>
                      <i aria-hidden="true" /> {humanizeOrderValue(order.status)}
                    </span>
                  </div>
                  <div className={styles.totalBlock}>
                    <span>Total</span>
                    <strong>{formatMoney(order.total_amount, order.currency)}</strong>
                  </div>
                </div>

                <div className={styles.cardBody}>
                  <div className={styles.itemSummary}>
                    {order.product_slug ? (
                      <Link
                        href={`/product/${encodeURIComponent(order.product_slug)}`}
                        className={styles.itemThumbLink}
                        aria-label={`View ${order.first_product_name || "ordered product"}`}
                      >
                        <span className={styles.itemThumb}>
                          <span className={styles.itemThumbFallback} aria-hidden="true">
                            {(order.first_product_name || "O").charAt(0).toUpperCase()}
                          </span>
                          {order.image_url ? (
                            // Catalog images can come from configured remote hosts.
                            // Keep a local fallback underneath if the remote image fails.
                            // eslint-disable-next-line @next/next/no-img-element
                            <img
                              src={order.image_url}
                              alt=""
                              loading="lazy"
                              decoding="async"
                              className={styles.itemThumbImage}
                              onError={hideBrokenOrderImage}
                            />
                          ) : null}
                        </span>
                      </Link>
                    ) : (
                      <span className={styles.itemThumb}>
                        <span className={styles.itemThumbFallback} aria-hidden="true">
                          {(order.first_product_name || "O").charAt(0).toUpperCase()}
                        </span>
                        {order.image_url ? (
                          // eslint-disable-next-line @next/next/no-img-element
                          <img
                            src={order.image_url}
                            alt=""
                            loading="lazy"
                            decoding="async"
                            className={styles.itemThumbImage}
                            onError={hideBrokenOrderImage}
                          />
                        ) : null}
                      </span>
                    )}
                    <div>
                      <strong>{order.first_product_name || `${order.item_count} product lines`}</strong>
                      <span>
                        {order.item_count.toLocaleString()} product {order.item_count === 1 ? "line" : "lines"}
                        {order.item_count > 1 ? ` · +${order.item_count - 1} more` : ""}
                      </span>
                      <small>{order.quantity_total.toLocaleString()} units</small>
                    </div>
                  </div>

                  <dl className={styles.orderFacts}>
                    <div><dt>Placed</dt><dd>{formatOrderDate(order.created_at)}</dd></div>
                    <div><dt>Delivery</dt><dd>{humanizeOrderValue(order.delivery_method)}</dd></div>
                    <div><dt>Payment</dt><dd>{humanizeOrderValue(order.payment_status)}</dd></div>
                  </dl>
                </div>

                <footer className={styles.cardActions}>
                  <Link className={styles.primaryButton} href={`/account/orders/${order.id}`}>
                    View order <Icon name="chevronRight" size={14} />
                  </Link>
                  <Link
                    className={styles.secondaryButton}
                    href={`/account/orders/${order.id}/invoice`}
                  >
                    Invoice
                  </Link>
                  {isDeliveredOrder(order.status) ? (
                    <>
                      <button
                        type="button"
                        className={styles.secondaryButton}
                        disabled={busyOrderId === order.id}
                        onClick={() => void buyAgain(order.id)}
                      >
                        {busyOrderId === order.id ? "Adding…" : "Buy again"}
                      </button>
                      <Link className={styles.textAction} href={`/account/reviews?order_id=${order.id}`}>
                        Review
                      </Link>
                    </>
                  ) : null}
                </footer>
              </article>
            ))}
          </section>
        ) : (
          <section className={`${styles.glassPanel} ${styles.emptyState}`}>
            <span className={styles.stateIcon}><Icon name="orders" size={24} /></span>
            <h2>{orders.length ? "No orders match these filters." : "No orders yet."}</h2>
            <p>
              {orders.length
                ? "Try another status or clear the search to see more of your history."
                : "When you place an order, its status, invoice and delivery progress will appear here."}
            </p>
            {orders.length ? (
              <button type="button" className={styles.secondaryButton} onClick={() => { setFilter("all"); setQuery(""); }}>
                Show all orders
              </button>
            ) : (
              <Link className={styles.primaryButton} href="/">Continue shopping</Link>
            )}
          </section>
        )}

        {hasMore ? (
          <div className={styles.loadMoreWrap}>
            <button type="button" className={styles.secondaryButton} disabled={loadingMore} onClick={() => void loadMore()}>
              {loadingMore ? "Loading older orders…" : "Load older orders"}
            </button>
          </div>
        ) : null}
      </div>
    </main>
  );
}

function Ambient() {
  return (
    <div className={styles.ambient} aria-hidden="true">
      <span className={styles.orbOne} />
      <span className={styles.orbTwo} />
      <span className={styles.orbThree} />
    </div>
  );
}
