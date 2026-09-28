// Location: src/components/account/components/WatchlistClient.tsx
"use client";

import Link from "next/link";
import {
  type ChangeEvent,
  type FormEvent,
  type SyntheticEvent,
  useCallback,
  useEffect,
  useMemo,
  useState,
} from "react";

import { Icon } from "@/components/ui/Icon";
import type {
  AccountOrderHistoryItem,
  AccountOrderHistoryResponse,
} from "@/lib/api/contracts/account";
import { formatMoney } from "@/lib/money/format";

import styles from "../css/Watchlist.module.css";

type WatchlistOrder = AccountOrderHistoryItem & {
  image_url?: string;
  product_slug?: string;
};

type WatchlistResponse = Omit<AccountOrderHistoryResponse, "data"> & {
  data: WatchlistOrder[];
};

type State =
  | { kind: "loading" }
  | { kind: "error"; message: string }
  | { kind: "ready"; orders: WatchlistOrder[] };

type OrderAction = "cancel" | "refund";

type ActionDialog = {
  order: WatchlistOrder;
  action: OrderAction;
} | null;

type Notice = {
  tone: "success" | "error";
  message: string;
} | null;

const REASONS = [
  "Changed my mind",
  "Ordered by mistake",
  "Need to change the order",
  "Payment issue",
  "Other",
] as const;

function normalize(value: string | null | undefined) {
  return String(value ?? "")
    .trim()
    .toLowerCase()
    .replaceAll("-", "_")
    .replaceAll(" ", "_");
}

function humanize(value: string | null | undefined) {
  const normalized = normalize(value);
  if (!normalized) return "—";
  return normalized
    .replaceAll("_", " ")
    .replace(/\b\w/g, (letter) => letter.toUpperCase());
}

function waitingForDelivery(order: AccountOrderHistoryItem) {
  return ["pending_payment", "confirmed", "processing"].includes(
    normalize(order.status),
  );
}

function actionForOrder(order: AccountOrderHistoryItem): OrderAction | null {
  const status = normalize(order.status);
  const paymentStatus = normalize(order.payment_status);
  const paymentMethod = normalize(order.payment_method);

  if (status === "pending_payment" && paymentStatus === "pending") {
    return "cancel";
  }

  if (["confirmed", "processing"].includes(status)) {
    if (paymentMethod === "cod" && paymentStatus === "cod_pending") {
      return "cancel";
    }

    if (paymentMethod !== "cod" && paymentStatus === "paid") {
      return "refund";
    }
  }

  return null;
}

function formatDate(value: string) {
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) return value;
  return new Intl.DateTimeFormat("en-BD", {
    day: "numeric",
    month: "short",
    year: "numeric",
  }).format(parsed);
}

function displayTitle(order: AccountOrderHistoryItem) {
  const base = order.first_product_name?.trim() || `Order #${order.order_number}`;
  const extra = Math.max(0, Number(order.item_count || 0) - 1);
  return extra > 0 ? `${base} + ${extra} more ${extra === 1 ? "product" : "products"}` : base;
}

function paymentLabel(order: AccountOrderHistoryItem) {
  const method = normalize(order.payment_method);
  const status = normalize(order.payment_status);

  if (method === "cod") {
    return status === "cod_collected" ? "COD · collected" : "COD · unpaid";
  }

  return `${humanize(method)} · ${humanize(status)}`;
}

function progressState(order: AccountOrderHistoryItem) {
  const status = normalize(order.status);
  if (status === "pending_payment") {
    return {
      middle: "Payment pending",
      firstDone: false,
      middleCurrent: false,
      copy: "The order is created, but payment has not been completed yet.",
    };
  }

  if (status === "confirmed") {
    return {
      middle: "Confirmed",
      firstDone: true,
      middleCurrent: true,
      copy: "The order is confirmed and remains here until fulfillment starts moving toward shipment.",
    };
  }

  return {
    middle: "Processing",
    firstDone: true,
    middleCurrent: true,
    copy: "The order is being prepared. It leaves Watchlist as soon as shipment or delivery begins.",
  };
}

async function readError(response: Response, fallback: string) {
  try {
    const payload = (await response.json()) as {
      error?: string | { message?: string };
      message?: string;
    };

    if (typeof payload.error === "string" && payload.error.trim()) {
      return payload.error.trim();
    }
    if (payload.error && typeof payload.error === "object") {
      const message = payload.error.message?.trim();
      if (message) return message;
    }
    return payload.message?.trim() || fallback;
  } catch {
    return fallback;
  }
}

function hideBrokenImage(event: SyntheticEvent<HTMLImageElement>) {
  event.currentTarget.style.display = "none";
}

export function WatchlistClient() {
  const [state, setState] = useState<State>({ kind: "loading" });
  const [dialog, setDialog] = useState<ActionDialog>(null);
  const [reason, setReason] = useState<string>(REASONS[0]);
  const [busyOrderID, setBusyOrderID] = useState("");
  const [notice, setNotice] = useState<Notice>(null);

  const load = useCallback(async (showLoading = true) => {
    if (showLoading) setState({ kind: "loading" });

    try {
      const response = await fetch(
        "/api/storefront/account/orders?page=1&limit=100",
        { cache: "no-store" },
      );

      if (response.status === 401) {
        location.replace("/account/sign-in?next=%2Faccount%2Fwatchlist");
        return;
      }

      if (!response.ok) {
        throw new Error(await readError(response, "Unable to load your watchlist."));
      }

      const payload = (await response.json()) as WatchlistResponse;
      const orders = Array.isArray(payload.data)
        ? payload.data.filter(waitingForDelivery)
        : [];

      setState({ kind: "ready", orders });
    } catch (caught) {
      setState({
        kind: "error",
        message:
          caught instanceof Error ? caught.message : "Unable to load your watchlist.",
      });
    }
  }, []);

  useEffect(() => {
    const timer = window.setTimeout(() => {
      void load();
    }, 0);

    return () => window.clearTimeout(timer);
  }, [load]);

  useEffect(() => {
    if (!dialog) return;

    function onKeyDown(event: KeyboardEvent) {
      if (event.key === "Escape" && !busyOrderID) {
        setDialog(null);
      }
    }

    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [dialog, busyOrderID]);

  const orders = useMemo(
    () => (state.kind === "ready" ? state.orders : []),
    [state],
  );

  function openAction(order: WatchlistOrder, action: OrderAction) {
    if (busyOrderID) return;
    setReason(REASONS[0]);
    setNotice(null);
    setDialog({ order, action });
  }

  async function submitAction(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!dialog || busyOrderID) return;

    const { order, action } = dialog;
    setBusyOrderID(order.id);
    setNotice(null);

    try {
      const response = await fetch(
        `/api/storefront/account/orders/${encodeURIComponent(order.id)}/cancel`,
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ reason }),
        },
      );

      if (response.status === 401) {
        location.replace("/account/sign-in?next=%2Faccount%2Fwatchlist");
        return;
      }

      if (!response.ok) {
        throw new Error(
          await readError(
            response,
            action === "refund"
              ? "Unable to start the cancellation and refund request."
              : "Unable to cancel this order.",
          ),
        );
      }

      setDialog(null);
      setNotice({
        tone: "success",
        message:
          action === "refund"
            ? "Order cancelled. The refund lifecycle has been started for the collected payment."
            : "Order cancelled successfully.",
      });
      await load(false);
    } catch (caught) {
      setNotice({
        tone: "error",
        message:
          caught instanceof Error
            ? caught.message
            : "Unable to update this order.",
      });
      await load(false);
    } finally {
      setBusyOrderID("");
    }
  }

  return (
    <main className={styles.page}>
      <div className={styles.ambient} aria-hidden="true">
        <span className={styles.orbOne} />
        <span className={styles.orbTwo} />
      </div>

      <div className={styles.shell}>
        <Link className={styles.backLink} href="/account">
          <Icon name="chevronLeft" size={15} /> Account
        </Link>

        <header className={styles.header}>
          <div>
            <span className={styles.eyebrow}>Orders you are watching</span>
            <h1>Watchlist</h1>
            <p>
              Orders stay here while payment, confirmation and preparation happen before
              shipping or delivery begins.
            </p>
          </div>
          <span className={styles.headerIcon} aria-hidden="true">
            <Icon name="bell" size={21} />
          </span>
        </header>

        <section className={styles.infoPanel} aria-labelledby="watchlist-rule-heading">
          <span className={styles.infoIcon} aria-hidden="true">
            <Icon name="delivery" size={20} />
          </span>
          <div>
            <span className={styles.eyebrow}>Automatic watchlist</span>
            <h2 id="watchlist-rule-heading">Only orders that have not started delivery</h2>
            <p>
              Pending-payment, confirmed and processing orders can appear here. Once an
              order enters shipment or delivery, it automatically leaves Watchlist and
              continues under Orders and tracking.
            </p>
          </div>
        </section>

        {notice ? (
          <div
            className={`${styles.notice} ${notice.tone === "error" ? styles.noticeError : styles.noticeSuccess}`}
            role="status"
          >
            {notice.message}
          </div>
        ) : null}

        {state.kind === "loading" ? (
          <section className={styles.loadingList} aria-busy="true" aria-label="Loading watchlist">
            <div className={styles.loadingCard} />
            <div className={styles.loadingCard} />
          </section>
        ) : null}

        {state.kind === "error" ? (
          <section className={styles.stateCard} role="alert">
            <span className={styles.stateIcon}><Icon name="support" size={22} /></span>
            <h2>Watchlist is unavailable.</h2>
            <p>{state.message}</p>
            <button className={styles.primaryButton} type="button" onClick={() => void load()}>
              Try again
            </button>
          </section>
        ) : null}

        {state.kind === "ready" && orders.length === 0 ? (
          <section className={styles.stateCard}>
            <span className={styles.stateIcon}><Icon name="delivery" size={23} /></span>
            <h2>Nothing is waiting here.</h2>
            <p>
              Watchlist fills automatically when you place an eligible order and clears
              it once shipping or delivery begins.
            </p>
            <div className={styles.emptyActions}>
              <Link className={styles.primaryButton} href="/account/orders">View all orders</Link>
              <Link className={styles.secondaryButton} href="/">Continue shopping</Link>
            </div>
          </section>
        ) : null}

        {state.kind === "ready" && orders.length > 0 ? (
          <section className={styles.orderList} aria-label="Orders waiting before delivery">
            {orders.map((order) => {
              const action = actionForOrder(order);
              const progress = progressState(order);
              const busy = busyOrderID === order.id;

              return (
                <article className={styles.orderCard} key={order.id}>
                  <div className={styles.orderBody}>
                    <div className={styles.thumbnail}>
                      <span className={styles.thumbnailFallback} aria-hidden="true">
                        {(order.first_product_name || order.order_number).trim().charAt(0).toUpperCase() || "E"}
                      </span>
                      {order.image_url ? (
                        // Catalog media is enrichment only; the fallback remains underneath.
                        // eslint-disable-next-line @next/next/no-img-element
                        <img
                          src={order.image_url}
                          alt=""
                          loading="lazy"
                          onError={hideBrokenImage}
                        />
                      ) : null}
                    </div>

                    <div className={styles.orderMain}>
                      <h2>{displayTitle(order)}</h2>
                      <p className={styles.orderNumber}>Order #{order.order_number}</p>
                      <div className={styles.metaRow}>
                        <span>{order.quantity_total.toLocaleString()} units</span>
                        <span>{order.item_count.toLocaleString()} product lines</span>
                        <span>Placed {formatDate(order.created_at)}</span>
                      </div>
                      <div className={styles.badgeRow}>
                        <span className={styles.statusBadge}>
                          <span className={styles.statusDot} aria-hidden="true" />
                          {humanize(order.status)}
                        </span>
                        <span className={styles.paymentBadge}>{paymentLabel(order)}</span>
                      </div>
                    </div>

                    <div className={styles.orderSide}>
                      <span className={styles.totalLabel}>Order total</span>
                      <strong className={styles.total}>{formatMoney(order.total_amount, order.currency)}</strong>
                      <div className={styles.actions}>
                        {action ? (
                          <button
                            className={styles.conditionalButton}
                            type="button"
                            disabled={busy}
                            onClick={() => openAction(order, action)}
                          >
                            {action === "refund" ? "Request refund" : "Cancel order"}
                          </button>
                        ) : null}
                        <Link
                          className={styles.secondaryButton}
                          href={`/account/orders/${encodeURIComponent(order.id)}/invoice`}
                        >
                          Invoice
                        </Link>
                        <Link
                          className={styles.primaryButton}
                          href={`/account/orders/${encodeURIComponent(order.id)}`}
                        >
                          View order <Icon name="chevronRight" size={13} />
                        </Link>
                      </div>
                    </div>
                  </div>

                  <div className={styles.progressPanel}>
                    <div className={styles.progressTop}>
                      <strong>Before delivery</strong>
                      <span>Leaves Watchlist when shipment starts</span>
                    </div>
                    <div className={styles.progressTrack} aria-hidden="true">
                      <span className={`${styles.progressStep} ${progress.firstDone ? styles.progressDone : styles.progressCurrent}`}>{progress.firstDone ? "✓" : "1"}</span>
                      <span className={`${styles.progressLine} ${progress.firstDone ? styles.progressLineDone : ""}`} />
                      <span className={`${styles.progressStep} ${progress.middleCurrent ? styles.progressCurrent : ""}`}>2</span>
                      <span className={styles.progressLine} />
                      <span className={styles.progressStep}>3</span>
                    </div>
                    <div className={styles.progressLabels}>
                      <span>Order placed</span>
                      <span>{progress.middle}</span>
                      <span>Delivery starts</span>
                    </div>
                    <p className={styles.progressCopy}>{progress.copy}</p>
                  </div>
                </article>
              );
            })}
          </section>
        ) : null}
      </div>

      {dialog ? (
        <div className={styles.dialogBackdrop} role="presentation">
          <section
            className={styles.dialog}
            role="dialog"
            aria-modal="true"
            aria-labelledby="watchlist-action-title"
          >
            <span className={styles.dialogIcon} aria-hidden="true">
              <Icon name={dialog.action === "refund" ? "returns" : "orders"} size={20} />
            </span>
            <span className={styles.eyebrow}>
              {dialog.action === "refund" ? "Refund request" : "Cancel order"}
            </span>
            <h2 id="watchlist-action-title">
              {dialog.action === "refund" ? "Cancel and request a refund?" : "Cancel this order?"}
            </h2>
            <p>
              {dialog.action === "refund"
                ? "This payment has already been collected. The order cancellation is processed first, and the backend starts the refund lifecycle in the same pre-shipment flow."
                : "No collected payment needs to be refunded. Cancellation eligibility is checked again before the order is changed."}
            </p>
            <p className={styles.dialogOrder}>Order #{dialog.order.order_number}</p>

            <form onSubmit={submitAction}>
              <label className={styles.reasonField}>
                <span>Reason</span>
                <select
                  value={reason}
                  onChange={(event: ChangeEvent<HTMLSelectElement>) => setReason(event.target.value)}
                  disabled={Boolean(busyOrderID)}
                  autoFocus
                >
                  {REASONS.map((item) => <option key={item} value={item}>{item}</option>)}
                </select>
              </label>

              <div className={styles.dialogActions}>
                <button
                  className={styles.secondaryButton}
                  type="button"
                  disabled={Boolean(busyOrderID)}
                  onClick={() => setDialog(null)}
                >
                  Keep order
                </button>
                <button
                  className={styles.primaryButton}
                  type="submit"
                  disabled={Boolean(busyOrderID)}
                >
                  {busyOrderID
                    ? "Checking eligibility…"
                    : dialog.action === "refund"
                      ? "Continue refund request"
                      : "Cancel order"}
                </button>
              </div>
            </form>
          </section>
        </div>
      ) : null}
    </main>
  );
}
