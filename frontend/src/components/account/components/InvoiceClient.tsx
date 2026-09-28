// Location: src/components/account/components/InvoiceClient.tsx
"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useCallback, useEffect, useMemo, useState } from "react";

import type {
  CustomerInvoice,
  CustomerInvoiceResponse,
} from "@/lib/api/contracts/invoice";
import { formatMoney } from "@/lib/money/format";

import styles from "../css/Invoice.module.css";

type Props = {
  orderId: string;
};

type LoadState = "loading" | "ready" | "error" | "signed-out";

function objectValue(value: unknown): Record<string, unknown> | null {
  return value && typeof value === "object" ? (value as Record<string, unknown>) : null;
}

function apiMessage(payload: unknown, fallback: string) {
  const root = objectValue(payload);
  if (!root) return fallback;

  const error = root.error;
  if (typeof error === "string" && error.trim()) return error;

  const errorRecord = objectValue(error);
  const nested = errorRecord?.message;
  if (typeof nested === "string" && nested.trim()) return nested;

  const message = root.message;
  return typeof message === "string" && message.trim() ? message : fallback;
}

async function readError(response: Response, fallback: string) {
  try {
    return apiMessage(await response.json(), fallback);
  } catch {
    return fallback;
  }
}

function humanize(value?: string) {
  if (!value) return "-";
  return value
    .trim()
    .replaceAll("-", "_")
    .replaceAll(" ", "_")
    .replaceAll("_", " ")
    .replace(/\b\w/g, (character) => character.toUpperCase());
}

function dateTime(value?: string) {
  if (!value) return "-";
  if (Number.isNaN(Date.parse(value))) return value;

  return new Intl.DateTimeFormat("en-BD", {
    day: "numeric",
    month: "short",
    year: "numeric",
    hour: "numeric",
    minute: "2-digit",
  }).format(new Date(value));
}

function dateOnly(value?: string) {
  if (!value) return "-";
  if (Number.isNaN(Date.parse(value))) return value;

  return new Intl.DateTimeFormat("en-BD", {
    day: "numeric",
    month: "short",
    year: "numeric",
  }).format(new Date(value));
}

function paymentDetail(invoice: CustomerInvoice) {
  if (invoice.paid_at) return `Paid ${dateTime(invoice.paid_at)}`;
  if (invoice.payment_due_at) return `Due ${dateTime(invoice.payment_due_at)}`;
  if (invoice.payment_method.toLowerCase().includes("cod")) return "Due on delivery";
  return humanize(invoice.payment_status);
}

function DownloadIcon() {
  return (
    <svg className={styles.buttonIcon} aria-hidden="true" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8">
      <path d="M12 3v12" />
      <path d="m7.5 10.5 4.5 4.5 4.5-4.5" />
      <path d="M5 20h14" />
    </svg>
  );
}

function PrintIcon() {
  return (
    <svg className={styles.buttonIcon} aria-hidden="true" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8">
      <path d="M7 8V3h10v5" />
      <path d="M7 17h10v4H7z" />
      <path d="M5 17H3v-7h18v7h-2" />
    </svg>
  );
}

function ArrowIcon() {
  return (
    <svg className={styles.buttonIcon} aria-hidden="true" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8">
      <path d="M19 12H5" />
      <path d="m10 7-5 5 5 5" />
    </svg>
  );
}

function Ambient() {
  return (
    <div className={styles.ambient} aria-hidden="true">
      <span />
      <span />
      <span />
    </div>
  );
}

export function InvoiceClient({ orderId }: Props) {
  const router = useRouter();
  const [state, setState] = useState<LoadState>("loading");
  const [invoice, setInvoice] = useState<CustomerInvoice | null>(null);
  const [error, setError] = useState("");

  const load = useCallback(async () => {
    setState("loading");
    setError("");

    try {
      const response = await fetch(
        `/api/storefront/order/${encodeURIComponent(orderId)}/invoice?format=json`,
        { cache: "no-store" },
      );

      if (response.status === 401) {
        setState("signed-out");
        return;
      }

      if (!response.ok) {
        throw new Error(await readError(response, "Unable to load this invoice."));
      }

      const payload = (await response.json()) as CustomerInvoiceResponse;
      if (!payload?.data?.invoice_number) {
        throw new Error("The invoice response is incomplete.");
      }

      setInvoice(payload.data);
      setState("ready");
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : "Unable to load this invoice.");
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
    const next = encodeURIComponent(`/account/orders/${orderId}/invoice`);
    router.replace(`/account/sign-in?next=${next}`);
  }, [orderId, router, state]);

  const totals = useMemo(() => {
    if (!invoice) return { productLines: 0, units: 0 };
    return {
      productLines: invoice.items.length,
      units: invoice.items.reduce((sum, item) => sum + item.quantity, 0),
    };
  }, [invoice]);

  if (state === "loading" || state === "signed-out") {
    return (
      <main className={styles.page} aria-busy="true">
        <Ambient />
        <div className={styles.shell}>
          <div className={`${styles.glassPanel} ${styles.heroSkeleton}`} />
          <div className={styles.loadingGrid}>
            <div className={`${styles.glassPanel} ${styles.documentSkeleton}`} />
            <div className={styles.sideSkeletons}><div /><div /><div /></div>
          </div>
        </div>
      </main>
    );
  }

  if (state === "error" || !invoice) {
    return (
      <main className={styles.page}>
        <Ambient />
        <section className={`${styles.glassPanel} ${styles.stateCard}`} role="alert">
          <span className={styles.stateMark}>!</span>
          <span className={styles.eyebrow}>Invoice unavailable</span>
          <h1>We could not open this invoice.</h1>
          <p>{error || "The invoice is unavailable."}</p>
          <div className={styles.stateActions}>
            <button type="button" className={styles.redGlassButton} onClick={() => void load()}>Try again</button>
            <Link className={styles.whiteGlassButton} href={`/account/orders/${encodeURIComponent(orderId)}`}>Back to order</Link>
          </div>
        </section>
      </main>
    );
  }

  const printableURL = `/api/storefront/order/${encodeURIComponent(invoice.order_id)}/invoice?print=1`;
  const downloadURL = `/api/storefront/order/${encodeURIComponent(invoice.order_id)}/invoice?download=1`;
  const orderURL = `/account/orders/${encodeURIComponent(invoice.order_id)}`;
  const supportURL = `/account/support?order_id=${encodeURIComponent(invoice.order_id)}`;
  const trackingURL = `${orderURL}#delivery-journey-heading`;
  const addressLines = [
    invoice.shipping_address_line1,
    invoice.shipping_address_line2,
    invoice.shipping_area,
    invoice.shipping_city,
    invoice.shipping_postal_code,
  ].filter(Boolean);

  return (
    <main className={styles.page}>
      <Ambient />
      <div className={styles.shell}>
        <nav className={styles.breadcrumb} aria-label="Breadcrumb">
          <Link href="/account">Account</Link><span aria-hidden="true">/</span>
          <Link href="/account/orders">Orders</Link><span aria-hidden="true">/</span>
          <span aria-current="page">Invoice</span>
        </nav>

        <header className={styles.pageHero}>
          <div className={styles.heroCopy}>
            <span className={styles.eyebrow}>Invoice</span>
            <div className={styles.invoiceTitleRow}>
              <h1>Invoice #{invoice.invoice_number}</h1>
              <span className={styles.paymentChip}><i aria-hidden="true" /> {paymentDetail(invoice)}</span>
            </div>
            <p>Order #{invoice.order_number} · Issued {dateOnly(invoice.issued_at)}</p>
          </div>

          <div className={styles.heroActions}>
            <Link className={styles.whiteGlassButton} href={orderURL}><ArrowIcon /> View order</Link>
            <a className={styles.whiteGlassButton} href={printableURL} target="_blank" rel="noreferrer"><PrintIcon /> Print / Save PDF</a>
            <a className={styles.redGlassButton} href={downloadURL}><DownloadIcon /> Download invoice</a>
          </div>
        </header>

        <section className={`${styles.glassPanel} ${styles.metricPanel}`} aria-label="Invoice summary">
          <div className={styles.glassMetric}>
            <span>Invoice total</span>
            <strong>{formatMoney(invoice.total_amount, invoice.currency)}</strong>
          </div>
          <div className={styles.glassMetric}>
            <span>Payment</span>
            <strong>{humanize(invoice.payment_method)}</strong>
            <small>{paymentDetail(invoice)}</small>
          </div>
          <div className={styles.glassMetric}>
            <span>Order status</span>
            <strong>{humanize(invoice.order_status)}</strong>
            <small>{totals.units.toLocaleString()} units · {totals.productLines.toLocaleString()} product {totals.productLines === 1 ? "line" : "lines"}</small>
          </div>
        </section>

        <div className={styles.mainGrid}>
          <section className={styles.invoiceStage} aria-label="Invoice document preview">
            <div className={styles.documentGlow} aria-hidden="true" />
            <article className={styles.invoicePaper}>
              <div className={styles.invoiceRedRule} />
              <div className={styles.invoiceInner}>
                <header className={styles.invoiceTop}>
                  <div>
                    <div className={styles.brand}>ENE DEI</div>
                    <div className={styles.brandSub}>Precision Commerce</div>
                    <p className={styles.businessAddress}>Business commerce platform<br />Dhaka, Bangladesh</p>
                  </div>
                  <div className={styles.invoiceHeading}>
                    <h2>INVOICE</h2>
                    <strong>#{invoice.invoice_number}</strong>
                    <span>Order #{invoice.order_number}</span>
                    <span>Issued {dateOnly(invoice.issued_at)}</span>
                  </div>
                </header>

                <section className={styles.partyGrid}>
                  <div>
                    <span className={styles.documentLabel}>Bill to</span>
                    <strong>{invoice.customer_name}</strong>
                    <p>{invoice.customer_phone}{invoice.customer_email ? <><br />{invoice.customer_email}</> : null}</p>
                  </div>
                  <div>
                    <span className={styles.documentLabel}>Ship to</span>
                    <strong>{invoice.customer_name}</strong>
                    <p>{addressLines.map((line, index) => <span key={`${line}-${index}`}>{line}{index < addressLines.length - 1 ? <br /> : null}</span>)}</p>
                  </div>
                </section>

                <section className={styles.invoiceMetaGrid}>
                  <div><span className={styles.documentLabel}>Payment method</span><strong>{humanize(invoice.payment_method)}</strong></div>
                  <div><span className={styles.documentLabel}>Payment status</span><strong>{humanize(invoice.payment_status)}</strong></div>
                  <div><span className={styles.documentLabel}>Currency</span><strong>{invoice.currency}</strong></div>
                </section>

                <section className={styles.itemsSection} aria-label="Invoice items">
                  <div className={styles.invoiceTableHead} aria-hidden="true">
                    <span>Item</span><span>Qty</span><span>Unit price</span><span>Amount</span>
                  </div>
                  <div className={styles.invoiceItems}>
                    {invoice.items.map((item) => (
                      <div className={styles.invoiceItemRow} key={`${item.variant_id}-${item.sku}`}>
                        <div className={styles.invoiceItemName}>
                          <strong>{item.product_name}</strong>
                          <small>SKU {item.sku}</small>
                        </div>
                        <div className={styles.invoiceItemValue} data-label="Qty">{item.quantity.toLocaleString()}</div>
                        <div className={styles.invoiceItemValue} data-label="Unit price">{formatMoney(item.unit_price_amount, invoice.currency)}</div>
                        <div className={`${styles.invoiceItemValue} ${styles.invoiceItemAmount}`} data-label="Amount">{formatMoney(item.line_total_amount, invoice.currency)}</div>
                      </div>
                    ))}
                  </div>
                </section>

                <section className={styles.invoiceBottom}>
                  <div className={styles.invoiceRecord}>
                    <span className={styles.documentLabel}>Invoice record</span>
                    <p>This customer invoice reflects the persistent commercial snapshot recorded for this order.</p>
                  </div>
                  <div className={styles.invoiceTotals}>
                    <div><span>Subtotal</span><strong>{formatMoney(invoice.subtotal_amount, invoice.currency)}</strong></div>
                    <div><span>Discount</span><strong>- {formatMoney(invoice.discount_amount, invoice.currency)}</strong></div>
                    <div><span>Delivery</span><strong>{formatMoney(invoice.shipping_amount, invoice.currency)}</strong></div>
                    <div className={styles.invoiceTotal}><span>Total</span><strong>{formatMoney(invoice.total_amount, invoice.currency)}</strong></div>
                  </div>
                </section>

                <footer className={styles.invoiceFooter}>
                  <span>Invoice {invoice.invoice_number}</span>
                  <span>ENE DEI · Customer copy</span>
                </footer>
              </div>
            </article>
          </section>

          <aside className={styles.sideColumn}>
            <section className={`${styles.glassPanel} ${styles.sideCard}`}>
              <span className={styles.eyebrow}>Invoice file</span>
              <h2>Keep a customer copy</h2>
              <p>Download the persistent invoice for your records or accounting workflow.</p>
              <a className={`${styles.redGlassButton} ${styles.fullButton}`} href={downloadURL}><DownloadIcon /> Download invoice</a>
            </section>

            <section className={`${styles.glassPanel} ${styles.sideCard}`}>
              <span className={styles.eyebrow}>Related order</span>
              <h2>Order #{invoice.order_number}</h2>
              <p>{humanize(invoice.delivery_method)} · {humanize(invoice.order_status)}</p>
              <div className={styles.sideLinks}>
                <Link className={styles.glassLink} href={orderURL}><span>View order details</span><b aria-hidden="true">›</b></Link>
                <Link className={styles.glassLink} href={trackingURL}><span>Track delivery</span><b aria-hidden="true">›</b></Link>
              </div>
            </section>

            <section className={`${styles.glassPanel} ${styles.sideCard}`}>
              <span className={styles.eyebrow}>Payment</span>
              <h2>{humanize(invoice.payment_method)}</h2>
              <p>{paymentDetail(invoice)}. Payment status: {humanize(invoice.payment_status)}.</p>
              <span className={styles.paymentAmount}>{formatMoney(invoice.total_amount, invoice.currency)}</span>
            </section>

            <section className={`${styles.glassPanel} ${styles.sideCard}`}>
              <span className={styles.eyebrow}>Need help?</span>
              <h2>Invoice support</h2>
              <p>Questions about this invoice, payment, or related order?</p>
              <Link className={`${styles.whiteGlassButton} ${styles.fullButton}`} href={supportURL}>Get support</Link>
            </section>
          </aside>
        </div>

        <div className={styles.mobileActionBar} aria-label="Invoice actions">
          <Link className={styles.whiteGlassButton} href={orderURL}>View order</Link>
          <a className={styles.whiteGlassButton} href={printableURL} target="_blank" rel="noreferrer"><PrintIcon /> Print</a>
          <a className={`${styles.redGlassButton} ${styles.mobileDownload}`} href={downloadURL}><DownloadIcon /> Download invoice</a>
        </div>
      </div>
    </main>
  );
}
