// Location: src/components/account/components/OrderDetailClient.tsx
"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useCallback, useEffect, useMemo, useState } from "react";
import type { SyntheticEvent } from "react";

import { Icon } from "@/components/ui/Icon";
import type {
  AccountOrder,
  AccountOrderEvent,
  AccountOrderResponse,
  AccountOrderTimelineResponse,
  AccountOrderTracking,
  AccountOrderTrackingResponse,
} from "@/lib/api/contracts/order-account";
import { formatMoney } from "@/lib/money/format";
import {
  DELIVERY_STEPS,
  deliveryProgressIndex,
  formatOrderDate,
  formatOrderDateTime,
  humanizeOrderValue,
  isDeliveredOrder,
  normalizeOrderValue,
  orderStatusTone,
} from "@/lib/orders/presentation";

import styles from "../css/Orders.module.css";

type Props = {
  orderId: string;
};

type LoadState = "loading" | "ready" | "error" | "signed-out";

type TrackingUpdate = {
  message: string;
  location?: string;
  at?: string;
};

type Notice = {
  kind: "success" | "warning" | "error";
  message: string;
} | null;

function objectValue(value: unknown): Record<string, unknown> | null {
  return value && typeof value === "object" ? (value as Record<string, unknown>) : null;
}

function stringValue(record: Record<string, unknown> | null, key: string) {
  const value = record?.[key];
  return typeof value === "string" && value.trim() ? value.trim() : "";
}

function booleanValue(record: Record<string, unknown> | null, key: string) {
  const value = record?.[key];
  return typeof value === "boolean" ? value : undefined;
}

function apiMessage(payload: unknown, fallback: string) {
  const root = objectValue(payload);
  if (!root) return fallback;

  const error = root.error;
  if (typeof error === "string" && error.trim()) return error;

  const errorRecord = objectValue(error);
  const errorMessage = stringValue(errorRecord, "message");
  if (errorMessage) return errorMessage;

  return stringValue(root, "message") || fallback;
}

async function readError(response: Response, fallback: string) {
  try {
    return apiMessage(await response.json(), fallback);
  } catch {
    return fallback;
  }
}

function hideBrokenItemImage(event: SyntheticEvent<HTMLImageElement>) {
  event.currentTarget.style.display = "none";
}

function timelineEventsFromPayload(payload: unknown): AccountOrderEvent[] {
  const root = objectValue(payload);
  if (!root) return [];

  const data = root.data;

  // Backward-compatible guard: older/local proxies may expose data as an array.
  if (Array.isArray(data)) {
    return data as AccountOrderEvent[];
  }

  const timeline = objectValue(data);
  const events = timeline?.events;

  return Array.isArray(events) ? (events as AccountOrderEvent[]) : [];
}

function trackingStage(tracking: AccountOrderTracking | null) {
  if (!tracking) return "";
  return tracking.current_stage || tracking.stage || tracking.status || "";
}

function trackingArrays(tracking: AccountOrderTracking | null) {
  if (!tracking) return [] as unknown[][];

  const arrays: unknown[][] = [];
  const root = tracking as Record<string, unknown>;

  for (const key of ["events", "timeline", "updates", "tracking_events"]) {
    const value = root[key];
    if (Array.isArray(value)) arrays.push(value);
  }

  for (const key of ["journey", "delivery", "shipment", "tracking"]) {
    const nested = objectValue(root[key]);
    if (!nested) continue;

    for (const nestedKey of ["events", "timeline", "updates", "tracking_events"]) {
      const value = nested[nestedKey];
      if (Array.isArray(value)) arrays.push(value);
    }
  }

  return arrays;
}

function trackingUpdateFromRecord(value: unknown): TrackingUpdate | null {
  const record = objectValue(value);
  if (!record) return null;

  const publicMessage = stringValue(record, "public_message");
  const customerVisible = booleanValue(record, "customer_visible");
  const customerMessage = customerVisible === true ? stringValue(record, "message") : "";
  const status =
    stringValue(record, "status") ||
    stringValue(record, "event_code") ||
    stringValue(record, "stage");

  const message = publicMessage || customerMessage || (status ? humanizeOrderValue(status) : "");
  if (!message) return null;

  const locationRecord = objectValue(record.location);
  const location =
    stringValue(record, "location_name") ||
    stringValue(locationRecord, "name") ||
    [stringValue(record, "city"), stringValue(record, "country_code")]
      .filter(Boolean)
      .join(", ") ||
    undefined;

  const at =
    stringValue(record, "occurred_at") ||
    stringValue(record, "event_at") ||
    stringValue(record, "created_at") ||
    stringValue(record, "updated_at") ||
    stringValue(record, "timestamp") ||
    undefined;

  return { message, location, at };
}

function latestCustomerUpdate(
  tracking: AccountOrderTracking | null,
  timeline: AccountOrderEvent[],
): TrackingUpdate | null {
  const candidates = trackingArrays(tracking)
    .flatMap((items) => items)
    .map(trackingUpdateFromRecord)
    .filter((item): item is TrackingUpdate => Boolean(item));

  candidates.sort((left, right) => {
    const a = left.at ? Date.parse(left.at) : 0;
    const b = right.at ? Date.parse(right.at) : 0;
    return b - a;
  });

  if (candidates[0]) return candidates[0];

  const safeTimeline = Array.isArray(timeline) ? timeline : [];
  const event = [...safeTimeline]
    .sort((left, right) => Date.parse(right.created_at) - Date.parse(left.created_at))[0];

  if (!event) return null;

  return {
    message: humanizeOrderValue(event.event_type),
    at: event.created_at,
  };
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

async function addItemsToCart(order: AccountOrder) {
  let added = 0;
  let failed = 0;

  for (const item of order.items) {
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

  return { added, failed, total: order.items.length };
}

export function OrderDetailClient({ orderId }: Props) {
  const router = useRouter();
  const [state, setState] = useState<LoadState>("loading");
  const [order, setOrder] = useState<AccountOrder | null>(null);
  const [timeline, setTimeline] = useState<AccountOrderEvent[]>([]);
  const [tracking, setTracking] = useState<AccountOrderTracking | null>(null);
  const [error, setError] = useState("");
  const [buyingAgain, setBuyingAgain] = useState(false);
  const [notice, setNotice] = useState<Notice>(null);

  const load = useCallback(async () => {
    setState("loading");
    setError("");

    try {
      const response = await fetch(
        `/api/storefront/account/orders/${encodeURIComponent(orderId)}`,
        { cache: "no-store" },
      );

      if (response.status === 401) {
        setState("signed-out");
        return;
      }

      if (!response.ok) {
        throw new Error(await readError(response, "Unable to load this order."));
      }

      const orderPayload = (await response.json()) as AccountOrderResponse;
      setOrder(orderPayload.data);

      const [timelineResult, trackingResult] = await Promise.allSettled([
        fetch(
          `/api/storefront/account/orders/${encodeURIComponent(orderId)}/timeline`,
          { cache: "no-store" },
        ),
        fetch(
          `/api/storefront/account/orders/${encodeURIComponent(orderId)}/tracking`,
          { cache: "no-store" },
        ),
      ]);

      if (timelineResult.status === "fulfilled" && timelineResult.value.ok) {
        const payload: AccountOrderTimelineResponse | unknown =
          await timelineResult.value.json();
        setTimeline(timelineEventsFromPayload(payload));
      } else {
        setTimeline([]);
      }

      if (trackingResult.status === "fulfilled" && trackingResult.value.ok) {
        const payload = (await trackingResult.value.json()) as AccountOrderTrackingResponse;
        setTracking(payload.data ?? null);
      } else {
        setTracking(null);
      }

      setState("ready");
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : "Unable to load this order.");
      setState("error");
    }
  }, [orderId]);

  useEffect(() => {
    const timer = window.setTimeout(() => {
      void load();
    }, 0);

    return () => window.clearTimeout(timer);
  }, [load]);

  useEffect(() => {
    if (state !== "signed-out") return;
    const next = encodeURIComponent(`/account/orders/${orderId}`);
    router.replace(`/account/sign-in?next=${next}`);
  }, [orderId, router, state]);

  const currentTrackingStage = useMemo(() => trackingStage(tracking), [tracking]);
  const progress = useMemo(
    () => (order ? deliveryProgressIndex(order.status, currentTrackingStage) : -1),
    [currentTrackingStage, order],
  );
  const latestUpdate = useMemo(
    () => latestCustomerUpdate(tracking, timeline),
    [tracking, timeline],
  );

  async function buyAgain() {
    if (!order || buyingAgain) return;
    setBuyingAgain(true);
    setNotice(null);

    try {
      const result = await addItemsToCart(order);

      if (result.total === 0 || result.added === 0) {
        setNotice({ kind: "error", message: "These order items cannot be added to the cart right now." });
        return;
      }

      if (result.failed > 0) {
        setNotice({
          kind: "warning",
          message: `${result.added} item${result.added === 1 ? "" : "s"} added. ${result.failed} item${result.failed === 1 ? " is" : "s are"} unavailable in the original quantity.`,
        });
        return;
      }

      router.push("/cart");
    } catch (caught) {
      setNotice({
        kind: "error",
        message: caught instanceof Error ? caught.message : "Unable to buy this order again.",
      });
    } finally {
      setBuyingAgain(false);
    }
  }

  if (state === "loading" || state === "signed-out") {
    return (
      <main className={styles.page} aria-busy="true">
        <Ambient />
        <div className={styles.shell}>
          <div className={`${styles.glassPanel} ${styles.detailSkeleton}`} />
          <div className={styles.detailSkeletonGrid}><div /><div /></div>
        </div>
      </main>
    );
  }

  if (state === "error" || !order) {
    return (
      <main className={styles.page}>
        <Ambient />
        <section className={`${styles.glassPanel} ${styles.stateCard}`} role="alert">
          <span className={styles.stateIcon}><Icon name="support" size={24} /></span>
          <h1>We could not open this order.</h1>
          <p>{error || "The order is unavailable."}</p>
          <div className={styles.stateActions}>
            <button type="button" className={styles.primaryButton} onClick={() => void load()}>Try again</button>
            <Link className={styles.secondaryButton} href="/account/orders">Back to orders</Link>
          </div>
        </section>
      </main>
    );
  }

  const delivered = isDeliveredOrder(order.status);
  const shipment = order.shipment ?? tracking?.shipment;
  const cancelled = ["cancelled", "canceled", "payment_expired"].includes(normalizeOrderValue(order.status));

  return (
    <main className={styles.page}>
      <Ambient />
      <div className={styles.shell}>
        <Link className={styles.backLink} href="/account/orders">
          <Icon name="chevronLeft" size={14} /> Orders
        </Link>

        <header className={`${styles.glassPanel} ${styles.detailHero}`}>
          <div className={styles.detailHeroMain}>
            <span className={styles.eyebrow}>Order details</span>
            <div className={styles.detailTitleRow}>
              <h1>Order #{order.order_number}</h1>
              <span className={`${styles.statusChip} ${statusClass(order.status)}`}>
                <i aria-hidden="true" /> {humanizeOrderValue(order.status)}
              </span>
            </div>
            <p>
              Placed {formatOrderDateTime(order.created_at)} · {order.item_count.toLocaleString()} product {order.item_count === 1 ? "line" : "lines"} · {order.quantity_total.toLocaleString()} units
            </p>
          </div>

          <div className={styles.detailHeroActions}>
            <Link
              className={styles.primaryButton}
              href={`/account/orders/${order.id}/invoice`}
            >
              View invoice
            </Link>
            <Link className={styles.secondaryButton} href={`/account/support?order_id=${order.id}`}>
              <Icon name="support" size={16} /> Get support
            </Link>
            {delivered ? (
              <button type="button" className={styles.secondaryButton} disabled={buyingAgain} onClick={() => void buyAgain()}>
                {buyingAgain ? "Adding…" : "Buy again"}
              </button>
            ) : null}
          </div>
        </header>

        {notice ? (
          <div className={`${styles.notice} ${styles[`notice${notice.kind[0].toUpperCase()}${notice.kind.slice(1)}`]}`} role="status">
            <span>{notice.message}</span>
            {notice.kind === "warning" ? <Link href="/cart">Open cart</Link> : null}
            <button type="button" aria-label="Dismiss message" onClick={() => setNotice(null)}>×</button>
          </div>
        ) : null}

        <section className={`${styles.glassPanel} ${styles.journeyPanel}`} aria-labelledby="delivery-journey-heading">
          <div className={styles.sectionHeading}>
            <div>
              <span className={styles.eyebrow}>Delivery journey</span>
              <h2 id="delivery-journey-heading">From order confirmation to your delivery</h2>
              <p>Progress is based on the latest order and customer-safe tracking state from the commerce backend.</p>
            </div>
            <span className={styles.sectionIcon}><Icon name="delivery" size={21} /></span>
          </div>

          {cancelled ? (
            <div className={styles.cancelledJourney}>
              <strong>{humanizeOrderValue(order.status)}</strong>
              <span>{order.cancellation_reason || "This order will not continue through delivery."}</span>
            </div>
          ) : (
            <div className={styles.journeyRail} role="list" aria-label="Delivery progress">
              {DELIVERY_STEPS.map((step, index) => {
                const done = progress > index;
                const current = progress === index;
                return (
                  <div key={step} className={styles.journeyStep} role="listitem">
                    <span
                      className={`${styles.journeyDot} ${done ? styles.journeyDone : ""} ${current ? styles.journeyCurrent : ""}`}
                      aria-hidden="true"
                    >
                      {done ? "✓" : index + 1}
                    </span>
                    {index < DELIVERY_STEPS.length - 1 ? (
                      <span className={`${styles.journeyLine} ${progress > index ? styles.journeyLineDone : ""}`} aria-hidden="true" />
                    ) : null}
                    <strong>{step}</strong>
                    <small>
                      {index === 0 && order.confirmed_at ? formatOrderDate(order.confirmed_at) : null}
                      {index === 1 && order.processing_at ? formatOrderDate(order.processing_at) : null}
                      {index === 2 && order.shipped_at ? formatOrderDate(order.shipped_at) : null}
                      {index === 4 && (order.delivered_at || order.completed_at) ? formatOrderDate(order.delivered_at || order.completed_at) : null}
                    </small>
                  </div>
                );
              })}
            </div>
          )}

          <div className={styles.latestUpdate}>
            <span className={styles.updateGlyph}><Icon name="delivery" size={17} /></span>
            <div>
              <span>Latest update</span>
              <strong>{latestUpdate?.message || humanizeOrderValue(currentTrackingStage || order.status)}</strong>
              <small>
                {[latestUpdate?.location, latestUpdate?.at ? formatOrderDateTime(latestUpdate.at) : ""]
                  .filter(Boolean)
                  .join(" · ") || "Tracking will update as your order moves forward."}
              </small>
            </div>
            {shipment?.tracking_number ? (
              <div className={styles.trackingNumber}>
                <span>Tracking</span>
                <strong>{shipment.tracking_number}</strong>
              </div>
            ) : null}
          </div>
        </section>

        <div className={styles.detailGrid}>
          <section className={`${styles.glassPanel} ${styles.itemsPanel}`} aria-labelledby="ordered-items-heading">
            <div className={styles.sectionHeadingCompact}>
              <div>
                <span className={styles.eyebrow}>Order contents</span>
                <h2 id="ordered-items-heading">Items</h2>
              </div>
              <span>{order.quantity_total.toLocaleString()} units</span>
            </div>

            <div className={styles.itemRows}>
              {order.items.map((item) => (
                <article key={item.id} className={styles.detailItemRow}>
                  {item.product_slug ? (
                    <Link
                      href={`/product/${encodeURIComponent(item.product_slug)}`}
                      className={styles.detailItemThumbLink}
                      aria-label={`View ${item.product_name}`}
                    >
                      <span className={styles.detailItemThumb}>
                        <span className={styles.detailItemFallback} aria-hidden="true">
                          {item.product_name.charAt(0).toUpperCase()}
                        </span>
                        {item.image_url ? (
                          // Catalog images can come from multiple configured remote hosts.
                          // A plain img avoids routing tiny order-history thumbnails through
                          // Next image optimization, while retaining a local fallback.
                          // eslint-disable-next-line @next/next/no-img-element
                          <img
                            src={item.image_url}
                            alt=""
                            loading="lazy"
                            decoding="async"
                            className={styles.detailItemThumbImage}
                            onError={hideBrokenItemImage}
                          />
                        ) : null}
                      </span>
                    </Link>
                  ) : (
                    <span className={styles.detailItemThumb}>
                      <span className={styles.detailItemFallback} aria-hidden="true">
                        {item.product_name.charAt(0).toUpperCase()}
                      </span>
                      {item.image_url ? (
                        // eslint-disable-next-line @next/next/no-img-element
                        <img
                          src={item.image_url}
                          alt=""
                          loading="lazy"
                          decoding="async"
                          className={styles.detailItemThumbImage}
                          onError={hideBrokenItemImage}
                        />
                      ) : null}
                    </span>
                  )}
                  <div className={styles.detailItemMain}>
                    <strong>{item.product_name}</strong>
                    <span>SKU {item.sku}</span>
                    <small>{item.quantity.toLocaleString()} units · {formatMoney(item.unit_price_amount, item.currency)} / unit</small>
                  </div>
                  <strong className={styles.detailItemPrice}>{formatMoney(item.line_total_amount, item.currency)}</strong>
                </article>
              ))}
            </div>

            {delivered ? (
              <div className={styles.reviewCallout} id="review-order">
                <div>
                  <span><Icon name="review" size={17} /></span>
                  <div>
                    <strong>How were these items?</strong>
                    <small>Reviews are available for delivered purchases.</small>
                  </div>
                </div>
                <Link className={styles.secondaryButton} href={`/account/reviews?order_id=${order.id}`}>
                  Review items
                </Link>
              </div>
            ) : null}
          </section>

          <aside className={styles.detailSidebar}>
            <section className={`${styles.glassPanel} ${styles.infoCard}`}>
              <div className={styles.infoCardTitle}>
                <span><Icon name="delivery" size={17} /></span>
                <h2>Shipping</h2>
              </div>
              <address>
                <strong>{order.customer_name}</strong>
                <span>{order.shipping_address_line1}</span>
                {order.shipping_address_line2 ? <span>{order.shipping_address_line2}</span> : null}
                <span>{[order.shipping_area, order.shipping_city, order.shipping_postal_code].filter(Boolean).join(", ")}</span>
                <span>{order.customer_phone}</span>
              </address>
              <dl className={styles.infoList}>
                <div><dt>Method</dt><dd>{humanizeOrderValue(order.delivery_method)}</dd></div>
                {shipment?.courier_name ? <div><dt>Courier</dt><dd>{shipment.courier_name}</dd></div> : null}
              </dl>
            </section>

            <section className={`${styles.glassPanel} ${styles.infoCard}`}>
              <div className={styles.infoCardTitle}>
                <span><Icon name="secureCheckout" size={17} /></span>
                <h2>Payment</h2>
              </div>
              <dl className={styles.infoList}>
                <div><dt>Method</dt><dd>{humanizeOrderValue(order.payment_method)}</dd></div>
                <div><dt>Status</dt><dd>{humanizeOrderValue(order.payment_status)}</dd></div>
                {order.paid_at ? <div><dt>Paid</dt><dd>{formatOrderDateTime(order.paid_at)}</dd></div> : null}
              </dl>
            </section>

            <section className={`${styles.glassPanel} ${styles.summaryCard}`}>
              <span className={styles.eyebrow}>Order summary</span>
              <div className={styles.summaryRows}>
                <div><span>Subtotal</span><strong>{formatMoney(order.subtotal_amount, order.currency)}</strong></div>
                <div><span>Discount</span><strong>− {formatMoney(order.discount_amount, order.currency)}</strong></div>
                <div><span>Shipping</span><strong>{formatMoney(order.shipping_amount, order.currency)}</strong></div>
                <div className={styles.summaryTotal}><span>Total</span><strong>{formatMoney(order.total_amount, order.currency)}</strong></div>
              </div>
              <Link
                className={styles.primaryButton}
                href={`/account/orders/${order.id}/invoice`}
              >
                View invoice
              </Link>
            </section>
          </aside>
        </div>
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
