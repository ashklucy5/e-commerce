"use client";

import { createPortal } from "react-dom";
import { useEffect, useState } from "react";

import type { AdminSourcingOffer } from "@/lib/admin/product-request-types";
import { formatMoney } from "@/lib/money/format";

import styles from "../css/SourcingNegotiation.module.css";

type Props = {
  offer: AdminSourcingOffer | null;
  busy: boolean;
  onClose: () => void;
  onFinalize: (offerId: string) => Promise<void>;
};

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

function specificationEntries(value: unknown): Array<[string, string]> {
  if (!value || typeof value !== "object" || Array.isArray(value)) return [];
  return Object.entries(value as Record<string, unknown>)
    .filter(([, item]) => item != null && String(item).trim())
    .map(([key, item]) => [key, String(item)]);
}

export default function SourcingFinalizeDialog({ offer, busy, onClose, onFinalize }: Props) {
  const [mounted, setMounted] = useState(false);
  const [localError, setLocalError] = useState("");

  useEffect(() => setMounted(true), []);
  useEffect(() => {
    if (offer) setLocalError("");
  }, [offer]);

  if (!mounted || !offer) return null;

  const quantity = offer.quoted_quantity ?? 0;
  const total = offer.unit_price * quantity + offer.shipping_price;
  const specs = specificationEntries(offer.offered_specifications);

  return createPortal(
    <div className={styles.modalBackdrop} role="presentation" onMouseDown={onClose}>
      <section
        className={`${styles.modalCard} ${styles.finalizeDialog}`}
        role="dialog"
        aria-modal="true"
        aria-labelledby="finalize-sourcing-title"
        onMouseDown={(event) => event.stopPropagation()}
      >
        <div className={styles.sheetHandle} aria-hidden="true" />
        <header className={styles.modalHeader}>
          <div>
            <span className={styles.overline}>Read-only commercial review</span>
            <h2 id="finalize-sourcing-title">Finalize agreement</h2>
            <p>These are the exact terms the customer accepted. Finalization locks this commercial snapshot.</p>
          </div>
          <button type="button" className={styles.iconButton} onClick={onClose} aria-label="Close final agreement review">×</button>
        </header>

        {localError ? <div className={styles.inlineError}>{localError}</div> : null}

        <div className={styles.finalizeProduct}>
          <span>Accepted product</span>
          <strong>{offer.product_name}</strong>
          <p>{offer.description}</p>
        </div>

        {specs.length > 0 ? (
          <div className={styles.specReview}>
            {specs.map(([key, value]) => (
              <span key={key}>
                <small>{key}</small>
                <strong>{value}</strong>
              </span>
            ))}
          </div>
        ) : null}

        <div className={styles.commercialReviewGrid}>
          <span><small>Quantity</small><strong>{offer.quoted_quantity ?? "—"}</strong></span>
          <span><small>MOQ</small><strong>{offer.minimum_order_quantity ?? "—"}</strong></span>
          <span><small>Unit price</small><strong>{formatMoney(offer.unit_price, offer.currency)}</strong></span>
          <span><small>Shipping</small><strong>{formatMoney(offer.shipping_price, offer.currency)}</strong></span>
          <span className={styles.commercialReviewTotal}><small>Total</small><strong>{formatMoney(total, offer.currency)}</strong></span>
        </div>

        <div className={styles.acceptedAt}>
          <span>Customer accepted</span>
          <strong>{formatDateTime(offer.customer_responded_at)}</strong>
        </div>

        <footer className={styles.modalFooter}>
          <button type="button" className={styles.secondaryButton} onClick={onClose} disabled={busy}>Cancel</button>
          <button
            type="button"
            className={styles.primaryButton}
            disabled={busy}
            onClick={() => {
              setLocalError("");
              void onFinalize(offer.id).catch((value: unknown) => {
                setLocalError(value instanceof Error ? value.message : "Unable to finalize this agreement.");
              });
            }}
          >
            {busy ? "Finalizing…" : "Finalize agreement"}
          </button>
        </footer>
      </section>
    </div>,
    document.body,
  );
}
