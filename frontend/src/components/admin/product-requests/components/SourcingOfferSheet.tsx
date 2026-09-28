"use client";

import { ChangeEvent, useEffect, useMemo, useRef, useState } from "react";
import { createPortal } from "react-dom";

import type {
  AdminProductRequest,
  AdminSourcingOffer,
  AdminSourcingOfferMutationPayload,
} from "@/lib/admin/product-request-types";
import { formatMoney } from "@/lib/money/format";

import {
  MAX_OFFER_ATTACHMENT_JSON_BYTES,
  MAX_SOURCING_ATTACHMENTS,
  attachmentPayloadBytes,
  createLinkAttachment,
  imageToSourcingAttachment,
  parseAttachments,
  serialiseAttachments,
  type SourcingAttachment,
} from "../../../../lib/product-requests/sourcing-attachments";
import SourcingAttachmentView from "./SourcingAttachmentView";
import styles from "../css/SourcingNegotiation.module.css";

type SpecRow = {
  id: string;
  key: string;
  value: string;
};

type OfferForm = {
  productName: string;
  description: string;
  unitPrice: string;
  shippingPrice: string;
  currency: string;
  quotedQuantity: string;
  minimumOrderQuantity: string;
  expiresAt: string;
};

type Props = {
  open: boolean;
  request: AdminProductRequest;
  offer: AdminSourcingOffer | null;
  busy: boolean;
  onClose: () => void;
  onPreview: (attachment: SourcingAttachment) => void;
  onSave: (
    offerId: string | null,
    payload: AdminSourcingOfferMutationPayload,
    sendAfterSave: boolean,
  ) => Promise<void>;
};

function randomID() {
  if (typeof crypto !== "undefined" && typeof crypto.randomUUID === "function") {
    return crypto.randomUUID();
  }
  return `${Date.now()}-${Math.random().toString(36).slice(2)}`;
}

function minorToMajor(value: number): string {
  if (!Number.isFinite(value)) return "0";
  return (value / 100).toFixed(2).replace(/\.00$/, "");
}

function majorToMinor(value: string): number {
  const parsed = Number(value);
  if (!Number.isFinite(parsed) || parsed < 0) return Number.NaN;
  return Math.round(parsed * 100);
}

function toLocalDateTime(value?: string): string {
  if (!value) return "";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "";
  const offset = date.getTimezoneOffset() * 60_000;
  return new Date(date.getTime() - offset).toISOString().slice(0, 16);
}

function specsFromUnknown(value: unknown): SpecRow[] {
  if (!value || typeof value !== "object" || Array.isArray(value)) return [];

  return Object.entries(value as Record<string, unknown>).map(([key, item]) => ({
    id: randomID(),
    key,
    value: item == null ? "" : String(item),
  }));
}

function requestSpecs(request: AdminProductRequest): SpecRow[] {
  return specsFromUnknown(request.customer_requirements);
}

function initialForm(request: AdminProductRequest, offer: AdminSourcingOffer | null): OfferForm {
  if (offer) {
    return {
      productName: offer.product_name,
      description: offer.description,
      unitPrice: minorToMajor(offer.unit_price),
      shippingPrice: minorToMajor(offer.shipping_price),
      currency: offer.currency,
      quotedQuantity: offer.quoted_quantity ? String(offer.quoted_quantity) : "",
      minimumOrderQuantity: offer.minimum_order_quantity
        ? String(offer.minimum_order_quantity)
        : "",
      expiresAt: toLocalDateTime(offer.expires_at),
    };
  }

  return {
    productName: request.requested_product_name,
    description: request.description,
    unitPrice: "",
    shippingPrice: "0",
    currency: "BDT",
    quotedQuantity: request.requested_quantity ? String(request.requested_quantity) : "",
    minimumOrderQuantity: "1",
    expiresAt: "",
  };
}

export default function SourcingOfferSheet({
  open,
  request,
  offer,
  busy,
  onClose,
  onPreview,
  onSave,
}: Props) {
  const [mounted, setMounted] = useState(false);
  const [form, setForm] = useState<OfferForm>(() => initialForm(request, offer));
  const [specs, setSpecs] = useState<SpecRow[]>(() =>
    offer ? specsFromUnknown(offer.offered_specifications) : requestSpecs(request),
  );
  const [attachments, setAttachments] = useState<SourcingAttachment[]>(() =>
    parseAttachments(offer?.attachments),
  );
  const [linkDraft, setLinkDraft] = useState("");
  const [localError, setLocalError] = useState("");
  const [attachmentBusy, setAttachmentBusy] = useState(false);
  const fileInputRef = useRef<HTMLInputElement | null>(null);

  useEffect(() => setMounted(true), []);

  const projectedTotal = useMemo(() => {
    const unit = majorToMinor(form.unitPrice);
    const shipping = majorToMinor(form.shippingPrice);
    const quantity = Number.parseInt(form.quotedQuantity, 10);
    if (!Number.isFinite(unit) || !Number.isFinite(shipping) || !Number.isInteger(quantity)) {
      return null;
    }
    return unit * quantity + shipping;
  }, [form.quotedQuantity, form.shippingPrice, form.unitPrice]);

  if (!mounted || !open) return null;

  function setField<K extends keyof OfferForm>(key: K, value: OfferForm[K]) {
    setForm((current) => ({ ...current, [key]: value }));
  }

  function updateSpec(id: string, field: "key" | "value", value: string) {
    setSpecs((current) =>
      current.map((row) => (row.id === id ? { ...row, [field]: value } : row)),
    );
  }

  function removeAttachment(id: string) {
    setAttachments((current) => current.filter((attachment) => attachment.id !== id));
  }

  function addLink() {
    setLocalError("");
    try {
      if (attachments.length >= MAX_SOURCING_ATTACHMENTS) {
        throw new Error(`You can attach up to ${MAX_SOURCING_ATTACHMENTS} references.`);
      }
      const next = [...attachments, createLinkAttachment(linkDraft)];
      if (attachmentPayloadBytes(next) > MAX_OFFER_ATTACHMENT_JSON_BYTES) {
        throw new Error("Offer references are too large. Remove an image or link and try again.");
      }
      setAttachments(next);
      setLinkDraft("");
    } catch (value: unknown) {
      setLocalError(value instanceof Error ? value.message : "Unable to add that link.");
    }
  }

  async function addImage(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0];
    event.target.value = "";
    if (!file) return;

    setLocalError("");
    setAttachmentBusy(true);
    try {
      if (attachments.length >= MAX_SOURCING_ATTACHMENTS) {
        throw new Error(`You can attach up to ${MAX_SOURCING_ATTACHMENTS} references.`);
      }
      const attachment = await imageToSourcingAttachment(file, "offer");
      const next = [...attachments, attachment];
      if (attachmentPayloadBytes(next) > MAX_OFFER_ATTACHMENT_JSON_BYTES) {
        throw new Error("Offer references are too large. Remove an image and try again.");
      }
      setAttachments(next);
    } catch (value: unknown) {
      setLocalError(value instanceof Error ? value.message : "Unable to prepare that image.");
    } finally {
      setAttachmentBusy(false);
    }
  }

  async function submit(sendAfterSave: boolean) {
    setLocalError("");

    const unitPrice = majorToMinor(form.unitPrice);
    const shippingPrice = majorToMinor(form.shippingPrice);
    const quotedQuantity = Number.parseInt(form.quotedQuantity, 10);
    const minimumOrderQuantity = Number.parseInt(form.minimumOrderQuantity, 10);

    if (!form.productName.trim() || !form.description.trim()) {
      setLocalError("Add the product name and description.");
      return;
    }
    if (form.productName.trim().length > 180 || form.description.trim().length > 8000) {
      setLocalError("Product name or description is longer than the sourcing API allows.");
      return;
    }
    if (!/^[A-Za-z]{3}$/.test(form.currency.trim())) {
      setLocalError("Currency must be a three-letter code such as BDT or USD.");
      return;
    }
    if (!Number.isFinite(unitPrice) || !Number.isFinite(shippingPrice)) {
      setLocalError("Enter valid unit and shipping prices.");
      return;
    }
    if (
      !Number.isInteger(quotedQuantity) ||
      quotedQuantity <= 0 ||
      !Number.isInteger(minimumOrderQuantity) ||
      minimumOrderQuantity <= 0 ||
      quotedQuantity < minimumOrderQuantity
    ) {
      setLocalError("Quantity must be a positive number and cannot be below the MOQ.");
      return;
    }

    const specificationObject: Record<string, string> = {};
    for (const row of specs) {
      const key = row.key.trim();
      const value = row.value.trim();
      if (!key && !value) continue;
      if (!key || !value) {
        setLocalError("Each specification row needs both a name and a value.");
        return;
      }
      specificationObject[key] = value;
    }

    if (attachmentPayloadBytes(attachments) > MAX_OFFER_ATTACHMENT_JSON_BYTES) {
      setLocalError("Offer references are too large. Remove an image or link and try again.");
      return;
    }

    if (new TextEncoder().encode(JSON.stringify(specificationObject)).byteLength > 124 * 1024) {
      setLocalError("Offer specifications are too large. Shorten the structured details and try again.");
      return;
    }

    if (form.expiresAt) {
      const expiry = new Date(form.expiresAt);
      if (Number.isNaN(expiry.getTime()) || expiry.getTime() <= Date.now()) {
        setLocalError("Offer expiry must be in the future.");
        return;
      }
    }

    const payload: AdminSourcingOfferMutationPayload = {
      product_name: form.productName.trim(),
      description: form.description.trim(),
      attachments: serialiseAttachments(attachments),
      offered_specifications: specificationObject,
      unit_price: unitPrice,
      shipping_price: shippingPrice,
      currency: form.currency.trim().toUpperCase(),
      quoted_quantity: quotedQuantity,
      minimum_order_quantity: minimumOrderQuantity,
      expires_at: form.expiresAt ? new Date(form.expiresAt).toISOString() : null,
    };

    try {
      await onSave(offer?.id ?? null, payload, sendAfterSave);
    } catch (value: unknown) {
      setLocalError(value instanceof Error ? value.message : "Unable to save this commercial offer.");
    }
  }

  return createPortal(
    <div className={styles.sheetBackdrop} role="presentation" onMouseDown={onClose}>
      <section
        className={styles.offerSheet}
        role="dialog"
        aria-modal="true"
        aria-labelledby="offer-sheet-title"
        onMouseDown={(event) => event.stopPropagation()}
      >
        <div className={styles.sheetHandle} aria-hidden="true" />
        <header className={styles.offerSheetHeader}>
          <div>
            <span className={styles.overline}>{offer ? "Draft offer" : "Commercial sourcing"}</span>
            <h2 id="offer-sheet-title">{offer ? "Edit commercial offer" : "Create commercial offer"}</h2>
            <p>Commercial fields become immutable once this offer is sent to the customer.</p>
          </div>
          <button type="button" className={styles.iconButton} onClick={onClose} aria-label="Close offer editor">×</button>
        </header>

        <form
          className={styles.offerSheetForm}
          onSubmit={(event) => {
            event.preventDefault();
            void submit(false);
          }}
        >
          {localError ? <div className={styles.inlineError}>{localError}</div> : null}

          <div className={styles.offerFormGrid}>
            <label className={styles.fieldWide}>
              <span>Product name</span>
              <input maxLength={180} value={form.productName} onChange={(event) => setField("productName", event.target.value)} />
            </label>

            <label className={styles.fieldWide}>
              <span>Description</span>
              <textarea rows={4} maxLength={8000} value={form.description} onChange={(event) => setField("description", event.target.value)} />
            </label>

            <label>
              <span>Quantity</span>
              <input type="number" min="1" step="1" value={form.quotedQuantity} onChange={(event) => setField("quotedQuantity", event.target.value)} />
            </label>

            <label>
              <span>Minimum order quantity</span>
              <input type="number" min="1" step="1" value={form.minimumOrderQuantity} onChange={(event) => setField("minimumOrderQuantity", event.target.value)} />
            </label>

            <label>
              <span>Unit price</span>
              <input type="number" min="0" step="0.01" value={form.unitPrice} onChange={(event) => setField("unitPrice", event.target.value)} />
            </label>

            <label>
              <span>Shipping price</span>
              <input type="number" min="0" step="0.01" value={form.shippingPrice} onChange={(event) => setField("shippingPrice", event.target.value)} />
            </label>

            <label>
              <span>Currency</span>
              <input maxLength={3} value={form.currency} onChange={(event) => setField("currency", event.target.value.toUpperCase())} />
            </label>

            <label>
              <span>Offer expiry <small>optional</small></span>
              <input type="datetime-local" value={form.expiresAt} onChange={(event) => setField("expiresAt", event.target.value)} />
            </label>
          </div>

          <section className={styles.offerFormSection}>
            <div className={styles.offerFormSectionHeader}>
              <div>
                <span className={styles.overline}>Structured details</span>
                <h3>Specifications</h3>
              </div>
              <button
                type="button"
                className={styles.ghostButton}
                onClick={() => setSpecs((current) => [...current, { id: randomID(), key: "", value: "" }])}
              >
                + Add specification
              </button>
            </div>

            <div className={styles.specRows}>
              {specs.length === 0 ? (
                <div className={styles.emptyMini}>No structured specifications yet.</div>
              ) : (
                specs.map((row) => (
                  <div className={styles.specRow} key={row.id}>
                    <input aria-label="Specification name" placeholder="e.g. Color" value={row.key} onChange={(event) => updateSpec(row.id, "key", event.target.value)} />
                    <input aria-label="Specification value" placeholder="e.g. Black" value={row.value} onChange={(event) => updateSpec(row.id, "value", event.target.value)} />
                    <button type="button" className={styles.iconButtonSmall} onClick={() => setSpecs((current) => current.filter((item) => item.id !== row.id))} aria-label="Remove specification">×</button>
                  </div>
                ))
              )}
            </div>
          </section>

          <section className={styles.offerFormSection}>
            <div className={styles.offerFormSectionHeader}>
              <div>
                <span className={styles.overline}>Images & references</span>
                <h3>Offer references</h3>
              </div>
              <span className={styles.counter}>{attachments.length}/{MAX_SOURCING_ATTACHMENTS}</span>
            </div>

            {attachments.length > 0 ? (
              <div className={styles.editableAttachments}>
                {attachments.map((attachment) => (
                  <div className={styles.editableAttachment} key={attachment.id}>
                    <SourcingAttachmentView value={[attachment]} onPreview={onPreview} compact />
                    <button type="button" className={styles.removeAttachment} onClick={() => removeAttachment(attachment.id)} aria-label={`Remove ${attachment.name}`}>×</button>
                  </div>
                ))}
              </div>
            ) : null}

            <div className={styles.referenceActions}>
              <input ref={fileInputRef} type="file" accept="image/*" hidden onChange={(event) => void addImage(event)} />
              <button type="button" className={styles.secondaryButton} disabled={attachmentBusy || attachments.length >= MAX_SOURCING_ATTACHMENTS} onClick={() => fileInputRef.current?.click()}>
                {attachmentBusy ? "Preparing image…" : "+ Add image"}
              </button>
              <div className={styles.linkAdder}>
                <input value={linkDraft} placeholder="https://supplier.example/item" onChange={(event) => setLinkDraft(event.target.value)} />
                <button type="button" className={styles.ghostButton} disabled={!linkDraft.trim() || attachments.length >= MAX_SOURCING_ATTACHMENTS} onClick={addLink}>Add link</button>
              </div>
            </div>
          </section>

          <section className={styles.offerSummary}>
            <div>
              <span>Commercial total</span>
              <strong>
                {projectedTotal == null
                  ? "—"
                  : formatMoney(projectedTotal, form.currency.trim().toUpperCase() || "BDT")}
              </strong>
            </div>
            <p>Quantity × unit price + shipping. The backend remains the source of truth when the offer is finalized.</p>
          </section>

          <footer className={styles.offerSheetFooter}>
            <button type="button" className={styles.secondaryButton} onClick={onClose} disabled={busy}>Cancel</button>
            <button type="submit" className={styles.secondaryButtonStrong} disabled={busy || attachmentBusy}>
              {busy ? "Saving…" : "Save draft"}
            </button>
            <button
              type="button"
              className={styles.primaryButton}
              disabled={busy || attachmentBusy}
              onClick={() => void submit(true)}
            >
              {busy ? "Working…" : "Send to customer"}
            </button>
          </footer>
        </form>
      </section>
    </div>,
    document.body,
  );
}
