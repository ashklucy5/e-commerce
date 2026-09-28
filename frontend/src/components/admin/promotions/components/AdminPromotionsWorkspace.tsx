"use client";

import {
  type FormEvent,
  useCallback,
  useEffect,
  useMemo,
  useState,
} from "react";
import { createPortal } from "react-dom";

import AdminPageHeader from "@/components/admin/layout/components/AdminPageHeader";
import { useAdminSession } from "@/components/admin/layout/context/AdminSessionContext";
import { AdminRequestError, adminFetch } from "@/lib/admin/api";
import type {
  AdminCatalogProductDetail,
  AdminCatalogProductDetailResponse,
  AdminCatalogProductListItem,
  AdminCatalogProductListResponse,
} from "@/lib/admin/catalog-types";
import type {
  AdminProductDiscount,
  AdminProductDiscountConfigRequest,
  AdminProductDiscountListResponse,
  AdminProductDiscountResponse,
  AdminPromotion,
  AdminPromotionCampaignType,
  AdminPromotionConfigRequest,
  AdminPromotionDiscountType,
  AdminPromotionEffectiveState,
  AdminPromotionListResponse,
  AdminPromotionResponse,
  AdminPromotionScope,
  AdminPromotionStatus,
  AdminPromotionTargetInput,
} from "@/lib/admin/promotion-types";
import { formatMoney } from "@/lib/money/format";

import styles from "../css/AdminPromotions.module.css";

type WorkspaceTab = "campaigns" | "discounts";
type PromotionEditorMode = "create" | "edit";

type PromotionDraft = {
  name: string;
  code: string;
  scope: AdminPromotionScope;
  campaignType: AdminPromotionCampaignType;
  discountType: AdminPromotionDiscountType;
  discountValue: string;
  minimumSubtotal: string;
  maximumDiscount: string;
  currency: string;
  status: AdminPromotionStatus;
  startsAt: string;
  endsAt: string;
  targets: AdminPromotionTargetInput[];
};

type DiscountDraft = {
  variantId: string;
  label: string;
  discountType: AdminPromotionDiscountType;
  discountValue: string;
};

const PAGE_SIZE = 24;

function messageForError(value: unknown): string {
  if (value instanceof AdminRequestError) return value.message;
  if (value instanceof Error) return value.message;
  return "Unable to complete this request.";
}

function toInputDateTime(value?: string): string {
  if (!value) return "";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "";
  const local = new Date(date.getTime() - date.getTimezoneOffset() * 60_000);
  return local.toISOString().slice(0, 16);
}

function toISO(value: string): string | null {
  const trimmed = value.trim();
  if (!trimmed) return null;
  const date = new Date(trimmed);
  return Number.isNaN(date.getTime()) ? null : date.toISOString();
}

function formatDateTime(value?: string): string {
  if (!value) return "No schedule";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "No schedule";
  return new Intl.DateTimeFormat("en", {
    month: "short",
    day: "numeric",
    year: "numeric",
    hour: "numeric",
    minute: "2-digit",
  }).format(date);
}

function formatPercentBPS(value?: number): string {
  if (typeof value !== "number") return "—";
  const percent = value / 100;
  return `${Number.isInteger(percent) ? percent : percent.toFixed(2)}%`;
}

function promotionValue(promotion: AdminPromotion): string {
  if (promotion.discount_type === "percentage") {
    return `${formatPercentBPS(promotion.percentage_bps)} off`;
  }
  return typeof promotion.fixed_amount === "number"
    ? `${formatMoney(promotion.fixed_amount, promotion.currency)} off`
    : "Fixed discount";
}

function stateLabel(value: AdminPromotionEffectiveState): string {
  switch (value) {
    case "live":
      return "Live";
    case "scheduled":
      return "Scheduled";
    case "expired":
      return "Expired";
    case "disabled":
      return "Disabled";
    default:
      return "Draft";
  }
}

function emptyPromotionDraft(): PromotionDraft {
  return {
    name: "",
    code: "",
    scope: "order",
    campaignType: "standard",
    discountType: "percentage",
    discountValue: "10",
    minimumSubtotal: "0",
    maximumDiscount: "",
    currency: "BDT",
    status: "draft",
    startsAt: "",
    endsAt: "",
    targets: [],
  };
}

function draftFromPromotion(promotion: AdminPromotion): PromotionDraft {
  return {
    name: promotion.name,
    code: promotion.code ?? "",
    scope: promotion.scope,
    campaignType: promotion.campaign_type,
    discountType: promotion.discount_type,
    discountValue:
      promotion.discount_type === "percentage"
        ? String((promotion.percentage_bps ?? 0) / 100)
        : String((promotion.fixed_amount ?? 0) / 100),
    minimumSubtotal: String(promotion.minimum_subtotal_amount / 100),
    maximumDiscount:
      typeof promotion.maximum_discount_amount === "number"
        ? String(promotion.maximum_discount_amount / 100)
        : "",
    currency: promotion.currency,
    status: promotion.status,
    startsAt: toInputDateTime(promotion.starts_at),
    endsAt: toInputDateTime(promotion.ends_at),
    targets: (promotion.targets ?? []).map((target) =>
      target.variant_id
        ? { variant_id: target.variant_id }
        : { product_id: target.product_id },
    ),
  };
}

function moneyInputToMinor(value: string): number {
  const parsed = Number(value);
  if (!Number.isFinite(parsed)) return 0;
  return Math.round(parsed * 100);
}

function percentageInputToBPS(value: string): number {
  const parsed = Number(value);
  if (!Number.isFinite(parsed)) return 0;
  return Math.round(parsed * 100);
}

function buildPromotionRequest(draft: PromotionDraft): AdminPromotionConfigRequest {
  const flashSale = draft.campaignType === "flash_sale";
  const productScope = flashSale ? true : draft.scope === "product";
  const code = flashSale ? "" : draft.code.trim().toUpperCase();

  return {
    name: draft.name.trim(),
    code: code || null,
    scope: productScope ? "product" : "order",
    campaign_type: draft.campaignType,
    discount_type: draft.discountType,
    percentage_bps:
      draft.discountType === "percentage"
        ? percentageInputToBPS(draft.discountValue)
        : null,
    fixed_amount:
      draft.discountType === "fixed"
        ? moneyInputToMinor(draft.discountValue)
        : null,
    minimum_subtotal_amount: flashSale
      ? 0
      : moneyInputToMinor(draft.minimumSubtotal),
    maximum_discount_amount:
      flashSale || !draft.maximumDiscount.trim()
        ? null
        : moneyInputToMinor(draft.maximumDiscount),
    currency: draft.currency.trim().toUpperCase(),
    status: draft.status,
    starts_at: toISO(draft.startsAt),
    ends_at: toISO(draft.endsAt),
    targets: productScope ? draft.targets : [],
  };
}

function normalizeProductList(
  response: AdminCatalogProductListResponse,
): AdminCatalogProductListItem[] {
  if (Array.isArray(response.data)) return response.data;
  return response.data.items ?? response.data.Items ?? [];
}

function targetKey(target: AdminPromotionTargetInput): string {
  return target.variant_id
    ? `variant:${target.variant_id}`
    : `product:${target.product_id ?? ""}`;
}

function ProductVariantPicker({
  mode,
  selected,
  onSelectTarget,
  onSelectVariant,
  onClose,
}: {
  mode: "target" | "variant";
  selected: AdminPromotionTargetInput[];
  onSelectTarget?: (target: AdminPromotionTargetInput, label: string) => void;
  onSelectVariant?: (variantId: string, label: string) => void;
  onClose: () => void;
}) {
  const [query, setQuery] = useState("");
  const [products, setProducts] = useState<AdminCatalogProductListItem[]>([]);
  const [details, setDetails] = useState<Record<string, AdminCatalogProductDetail>>({});
  const [expanded, setExpanded] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  const selectedKeys = useMemo(
    () => new Set(selected.map(targetKey)),
    [selected],
  );

  const loadProducts = useCallback(async () => {
    setLoading(true);
    setError("");
    const params = new URLSearchParams({ page: "1", limit: "24" });
    if (query.trim()) params.set("q", query.trim());

    try {
      const response = await adminFetch<AdminCatalogProductListResponse>(
        `/products?${params.toString()}`,
      );
      setProducts(normalizeProductList(response));
    } catch (value: unknown) {
      setProducts([]);
      setError(messageForError(value));
    } finally {
      setLoading(false);
    }
  }, [query]);

  useEffect(() => {
    const timer = window.setTimeout(() => void loadProducts(), 220);
    return () => window.clearTimeout(timer);
  }, [loadProducts]);

  async function toggleProduct(productId: string) {
    if (expanded === productId) {
      setExpanded(null);
      return;
    }
    setExpanded(productId);
    if (details[productId]) return;

    try {
      const response = await adminFetch<AdminCatalogProductDetailResponse>(
        `/products/${productId}`,
      );
      setDetails((current) => ({ ...current, [productId]: response.data }));
    } catch (value: unknown) {
      setError(messageForError(value));
    }
  }

  return createPortal(
    <div className={styles.modalBackdrop} role="presentation" onMouseDown={onClose}>
      <section
        className={styles.picker}
        role="dialog"
        aria-modal="true"
        aria-label={mode === "target" ? "Choose promotion targets" : "Choose product variant"}
        onMouseDown={(event) => event.stopPropagation()}
      >
        <header className={styles.modalHeader}>
          <div>
            <p className={styles.kicker}>Catalog</p>
            <h2>{mode === "target" ? "Choose targets" : "Choose a variant"}</h2>
          </div>
          <button type="button" className={styles.iconButton} onClick={onClose} aria-label="Close">
            ×
          </button>
        </header>

        <div className={styles.pickerSearch}>
          <input
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            placeholder="Search product code, slug, SKU or ID"
            autoFocus
          />
        </div>

        {error ? <p className={styles.inlineError}>{error}</p> : null}

        <div className={styles.pickerList}>
          {loading ? <p className={styles.emptyText}>Loading catalog…</p> : null}
          {!loading && products.length === 0 ? (
            <p className={styles.emptyText}>No matching products.</p>
          ) : null}

          {products.map((product) => {
            const productSelected = selectedKeys.has(`product:${product.id}`);
            const detail = details[product.id];
            const isExpanded = expanded === product.id;

            return (
              <article key={product.id} className={styles.pickerProduct}>
                <div className={styles.pickerProductRow}>
                  <button
                    type="button"
                    className={styles.expandButton}
                    onClick={() => void toggleProduct(product.id)}
                  >
                    <span>
                      <strong>{product.name}</strong>
                      <small>{product.product_code} · {product.variant_count} variants</small>
                    </span>
                    <span aria-hidden="true">{isExpanded ? "−" : "+"}</span>
                  </button>

                  {mode === "target" ? (
                    <button
                      type="button"
                      className={styles.secondaryButton}
                      disabled={productSelected}
                      onClick={() => {
                        onSelectTarget?.({ product_id: product.id }, product.name);
                      }}
                    >
                      {productSelected ? "Added" : "Whole product"}
                    </button>
                  ) : null}
                </div>

                {isExpanded ? (
                  <div className={styles.variantList}>
                    {!detail ? <p className={styles.emptyText}>Loading variants…</p> : null}
                    {detail?.variants.map((variant) => {
                      const variantSelected = selectedKeys.has(`variant:${variant.id}`);
                      const label = `${product.name} · ${variant.sku}`;
                      return (
                        <button
                          key={variant.id}
                          type="button"
                          className={styles.variantRow}
                          disabled={variantSelected}
                          onClick={() => {
                            if (mode === "target") {
                              onSelectTarget?.({ variant_id: variant.id }, label);
                            } else {
                              onSelectVariant?.(variant.id, label);
                            }
                          }}
                        >
                          <span>
                            <strong>{variant.sku}</strong>
                            <small>
                              {typeof variant.price_amount === "number"
                                ? formatMoney(variant.price_amount, variant.currency ?? "BDT")
                                : "Price unavailable"}
                              {variant.is_active === false ? " · inactive" : ""}
                            </small>
                          </span>
                          <span>{variantSelected ? "Added" : mode === "target" ? "Add variant" : "Choose"}</span>
                        </button>
                      );
                    })}
                  </div>
                ) : null}
              </article>
            );
          })}
        </div>
      </section>
    </div>,
    document.body,
  );
}

function PromotionEditor({
  mode,
  promotion,
  onClose,
  onSaved,
}: {
  mode: PromotionEditorMode;
  promotion: AdminPromotion | null;
  onClose: () => void;
  onSaved: (promotion: AdminPromotion) => void;
}) {
  const [draft, setDraft] = useState<PromotionDraft>(
    promotion ? draftFromPromotion(promotion) : emptyPromotionDraft(),
  );
  const [targetLabels, setTargetLabels] = useState<Record<string, string>>(() => {
    const labels: Record<string, string> = {};
    for (const target of promotion?.targets ?? []) {
      const input = target.variant_id
        ? { variant_id: target.variant_id }
        : { product_id: target.product_id };
      labels[targetKey(input)] = target.variant_id
        ? `${target.product_name || "Variant"} · ${target.sku || target.variant_id}`
        : target.product_name || target.product_id || "Product";
    }
    return labels;
  });
  const [pickerOpen, setPickerOpen] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");

  const flashSale = draft.campaignType === "flash_sale";
  const productScope = flashSale || draft.scope === "product";

  function update<K extends keyof PromotionDraft>(key: K, value: PromotionDraft[K]) {
    setDraft((current) => ({ ...current, [key]: value }));
  }

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setSaving(true);
    setError("");

    const payload = buildPromotionRequest(draft);
    if (!payload.name) {
      setError("Promotion name is required.");
      setSaving(false);
      return;
    }
    if (payload.discount_type === "percentage" && (!payload.percentage_bps || payload.percentage_bps < 1 || payload.percentage_bps > 10000)) {
      setError("Percentage must be greater than 0 and no more than 100%.");
      setSaving(false);
      return;
    }
    if (payload.discount_type === "fixed" && (!payload.fixed_amount || payload.fixed_amount <= 0)) {
      setError("Fixed discount must be greater than zero.");
      setSaving(false);
      return;
    }
    if (payload.campaign_type === "flash_sale" && (!payload.starts_at || !payload.ends_at)) {
      setError("Flash sales require a start and end time.");
      setSaving(false);
      return;
    }
    if (payload.scope === "product" && payload.status === "active" && payload.targets.length === 0) {
      setError("Active product promotions require at least one product or variant target.");
      setSaving(false);
      return;
    }

    try {
      const response = await adminFetch<AdminPromotionResponse>(
        mode === "create" ? "/promotions" : `/promotions/${promotion?.id}`,
        {
          method: mode === "create" ? "POST" : "PUT",
          body: JSON.stringify(payload),
        },
      );
      onSaved(response.data);
    } catch (value: unknown) {
      setError(messageForError(value));
    } finally {
      setSaving(false);
    }
  }

  return createPortal(
    <div className={styles.modalBackdrop} role="presentation" onMouseDown={onClose}>
      <section
        className={styles.editor}
        role="dialog"
        aria-modal="true"
        aria-label={mode === "create" ? "Create promotion" : "Edit promotion"}
        onMouseDown={(event) => event.stopPropagation()}
      >
        <header className={styles.modalHeader}>
          <div>
            <p className={styles.kicker}>{mode === "create" ? "New promotion" : "Promotion settings"}</p>
            <h2>{mode === "create" ? "Create campaign" : promotion?.name}</h2>
          </div>
          <button type="button" className={styles.iconButton} onClick={onClose} aria-label="Close">×</button>
        </header>

        <form className={styles.editorForm} onSubmit={submit}>
          <div className={styles.formGrid}>
            <label className={styles.fieldWide}>
              <span>Name</span>
              <input value={draft.name} onChange={(event) => update("name", event.target.value)} maxLength={160} required />
            </label>

            <label>
              <span>Campaign type</span>
              <select
                value={draft.campaignType}
                onChange={(event) => {
                  const value = event.target.value as AdminPromotionCampaignType;
                  setDraft((current) => ({
                    ...current,
                    campaignType: value,
                    scope: value === "flash_sale" ? "product" : current.scope,
                    code: value === "flash_sale" ? "" : current.code,
                    minimumSubtotal: value === "flash_sale" ? "0" : current.minimumSubtotal,
                    maximumDiscount: value === "flash_sale" ? "" : current.maximumDiscount,
                  }));
                }}
              >
                <option value="standard">Standard promotion</option>
                <option value="flash_sale">Flash sale</option>
              </select>
            </label>

            <label>
              <span>Status</span>
              <select value={draft.status} onChange={(event) => update("status", event.target.value as AdminPromotionStatus)}>
                <option value="draft">Draft</option>
                <option value="active">Active</option>
                <option value="disabled">Disabled</option>
              </select>
            </label>

            <label>
              <span>Scope</span>
              <select
                value={flashSale ? "product" : draft.scope}
                disabled={flashSale}
                onChange={(event) => update("scope", event.target.value as AdminPromotionScope)}
              >
                <option value="order">Whole order</option>
                <option value="product">Products / variants</option>
              </select>
            </label>

            <label>
              <span>Code</span>
              <input
                value={flashSale ? "" : draft.code}
                disabled={flashSale}
                onChange={(event) => update("code", event.target.value.toUpperCase())}
                placeholder={flashSale ? "Automatic for flash sales" : "Leave blank for automatic"}
                maxLength={64}
              />
            </label>

            <label>
              <span>Discount type</span>
              <select value={draft.discountType} onChange={(event) => update("discountType", event.target.value as AdminPromotionDiscountType)}>
                <option value="percentage">Percentage</option>
                <option value="fixed">Fixed amount</option>
              </select>
            </label>

            <label>
              <span>{draft.discountType === "percentage" ? "Discount (%)" : `Discount (${draft.currency || "currency"})`}</span>
              <input
                type="number"
                min="0"
                step={draft.discountType === "percentage" ? "0.01" : "0.01"}
                value={draft.discountValue}
                onChange={(event) => update("discountValue", event.target.value)}
                required
              />
            </label>

            <label>
              <span>Currency</span>
              <input value={draft.currency} onChange={(event) => update("currency", event.target.value.toUpperCase())} maxLength={3} required />
            </label>

            <label>
              <span>Minimum subtotal</span>
              <input
                type="number"
                min="0"
                step="0.01"
                disabled={flashSale}
                value={flashSale ? "0" : draft.minimumSubtotal}
                onChange={(event) => update("minimumSubtotal", event.target.value)}
              />
            </label>

            <label>
              <span>Maximum discount</span>
              <input
                type="number"
                min="0"
                step="0.01"
                disabled={flashSale}
                value={flashSale ? "" : draft.maximumDiscount}
                onChange={(event) => update("maximumDiscount", event.target.value)}
                placeholder="Optional"
              />
            </label>

            <label>
              <span>Starts</span>
              <input type="datetime-local" value={draft.startsAt} onChange={(event) => update("startsAt", event.target.value)} required={flashSale} />
            </label>

            <label>
              <span>Ends</span>
              <input type="datetime-local" value={draft.endsAt} onChange={(event) => update("endsAt", event.target.value)} required={flashSale} />
            </label>
          </div>

          {productScope ? (
            <section className={styles.targetSection}>
              <div className={styles.targetHeader}>
                <div>
                  <strong>Product targets</strong>
                  <span>Choose a whole product or individual variants.</span>
                </div>
                <button type="button" className={styles.secondaryButton} onClick={() => setPickerOpen(true)}>Add targets</button>
              </div>

              <div className={styles.targetChips}>
                {draft.targets.length === 0 ? <p className={styles.emptyText}>No targets selected.</p> : null}
                {draft.targets.map((target) => {
                  const key = targetKey(target);
                  return (
                    <span key={key} className={styles.targetChip}>
                      <span>{targetLabels[key] ?? (target.variant_id ? `Variant ${target.variant_id}` : `Product ${target.product_id}`)}</span>
                      <button
                        type="button"
                        aria-label="Remove target"
                        onClick={() => update("targets", draft.targets.filter((item) => targetKey(item) !== key))}
                      >
                        ×
                      </button>
                    </span>
                  );
                })}
              </div>
            </section>
          ) : null}

          {flashSale ? (
            <p className={styles.formHint}>Flash sales are automatic product promotions. They require a time window and cannot use a promo code, minimum subtotal, or maximum-discount cap.</p>
          ) : null}

          {error ? <p className={styles.formError}>{error}</p> : null}

          <footer className={styles.modalFooter}>
            <button type="button" className={styles.secondaryButton} onClick={onClose}>Cancel</button>
            <button type="submit" className={styles.primaryButton} disabled={saving}>{saving ? "Saving…" : mode === "create" ? "Create promotion" : "Save changes"}</button>
          </footer>
        </form>

        {pickerOpen ? (
          <ProductVariantPicker
            mode="target"
            selected={draft.targets}
            onClose={() => setPickerOpen(false)}
            onSelectTarget={(target, label) => {
              const key = targetKey(target);
              if (draft.targets.some((item) => targetKey(item) === key)) return;
              update("targets", [...draft.targets, target]);
              setTargetLabels((current) => ({ ...current, [key]: label }));
            }}
          />
        ) : null}
      </section>
    </div>,
    document.body,
  );
}

function DiscountEditor({
  onClose,
  onSaved,
}: {
  onClose: () => void;
  onSaved: (discount: AdminProductDiscount) => void;
}) {
  const [draft, setDraft] = useState<DiscountDraft>({
    variantId: "",
    label: "",
    discountType: "percentage",
    discountValue: "10",
  });
  const [pickerOpen, setPickerOpen] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");
    if (!draft.variantId) {
      setError("Choose a product variant first.");
      return;
    }

    const payload: AdminProductDiscountConfigRequest = {
      discount_type: draft.discountType,
      percentage_bps:
        draft.discountType === "percentage"
          ? percentageInputToBPS(draft.discountValue)
          : null,
      fixed_amount:
        draft.discountType === "fixed"
          ? moneyInputToMinor(draft.discountValue)
          : null,
    };

    if (
      payload.discount_type === "percentage" &&
      (!payload.percentage_bps ||
        payload.percentage_bps < 1 ||
        payload.percentage_bps > 10000)
    ) {
      setError("Percentage must be greater than 0 and no more than 100%.");
      return;
    }

    if (
      payload.discount_type === "fixed" &&
      (!payload.fixed_amount || payload.fixed_amount <= 0)
    ) {
      setError("Fixed discount must be greater than zero.");
      return;
    }

    setSaving(true);
    try {
      const response = await adminFetch<AdminProductDiscountResponse>(
        `/promotions/product-discounts/${draft.variantId}`,
        { method: "PUT", body: JSON.stringify(payload) },
      );
      onSaved(response.data);
    } catch (value: unknown) {
      setError(messageForError(value));
    } finally {
      setSaving(false);
    }
  }

  return createPortal(
    <div className={styles.modalBackdrop} role="presentation" onMouseDown={onClose}>
      <section className={styles.smallEditor} role="dialog" aria-modal="true" aria-label="Apply product discount" onMouseDown={(event) => event.stopPropagation()}>
        <header className={styles.modalHeader}>
          <div>
            <p className={styles.kicker}>Catalog pricing</p>
            <h2>Apply product discount</h2>
          </div>
          <button type="button" className={styles.iconButton} onClick={onClose} aria-label="Close">×</button>
        </header>

        <form className={styles.editorForm} onSubmit={submit}>
          <button type="button" className={styles.variantChooser} onClick={() => setPickerOpen(true)}>
            <span>{draft.variantId ? draft.label : "Choose product variant"}</span>
            <strong>{draft.variantId ? "Change" : "Browse catalog"}</strong>
          </button>

          <div className={styles.formGrid}>
            <label>
              <span>Discount type</span>
              <select value={draft.discountType} onChange={(event) => setDraft((current) => ({ ...current, discountType: event.target.value as AdminPromotionDiscountType }))}>
                <option value="percentage">Percentage</option>
                <option value="fixed">Fixed amount</option>
              </select>
            </label>
            <label>
              <span>{draft.discountType === "percentage" ? "Discount (%)" : "Discount amount"}</span>
              <input type="number" min="0.01" step="0.01" value={draft.discountValue} onChange={(event) => setDraft((current) => ({ ...current, discountValue: event.target.value }))} required />
            </label>
          </div>

          <p className={styles.formHint}>A product discount changes the variant sale price and preserves its regular price as the compare-at price. It is separate from checkout promotion campaigns.</p>
          {error ? <p className={styles.formError}>{error}</p> : null}

          <footer className={styles.modalFooter}>
            <button type="button" className={styles.secondaryButton} onClick={onClose}>Cancel</button>
            <button type="submit" className={styles.primaryButton} disabled={saving}>{saving ? "Applying…" : "Apply discount"}</button>
          </footer>
        </form>

        {pickerOpen ? (
          <ProductVariantPicker
            mode="variant"
            selected={draft.variantId ? [{ variant_id: draft.variantId }] : []}
            onClose={() => setPickerOpen(false)}
            onSelectVariant={(variantId, label) => {
              setDraft((current) => ({ ...current, variantId, label }));
              setPickerOpen(false);
            }}
          />
        ) : null}
      </section>
    </div>,
    document.body,
  );
}

export default function AdminPromotionsWorkspace() {
  const principal = useAdminSession();
  const isSuperAdmin = principal.staff.roles.includes("admin_superuser");
  const permissions = principal.staff.permissions;
  const canRead = isSuperAdmin || permissions.includes("admin.promotion.read");
  const canManage = isSuperAdmin || permissions.includes("admin.promotion.manage");

  const [tab, setTab] = useState<WorkspaceTab>("campaigns");
  const [queryInput, setQueryInput] = useState("");
  const [query, setQuery] = useState("");
  const [page, setPage] = useState(1);
  const [status, setStatus] = useState("");
  const [campaignType, setCampaignType] = useState("");
  const [scope, setScope] = useState("");
  const [campaigns, setCampaigns] = useState<AdminPromotion[]>([]);
  const [campaignMeta, setCampaignMeta] = useState<AdminPromotionListResponse["meta"] | null>(null);
  const [discounts, setDiscounts] = useState<AdminProductDiscount[]>([]);
  const [discountMeta, setDiscountMeta] = useState<AdminProductDiscountListResponse["meta"] | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  const [editorMode, setEditorMode] = useState<PromotionEditorMode | null>(null);
  const [selectedPromotion, setSelectedPromotion] = useState<AdminPromotion | null>(null);
  const [discountEditorOpen, setDiscountEditorOpen] = useState(false);
  const [clearingVariant, setClearingVariant] = useState("");

  useEffect(() => {
    const timer = window.setTimeout(() => {
      setQuery(queryInput.trim());
      setPage(1);
    }, 260);
    return () => window.clearTimeout(timer);
  }, [queryInput]);

  const loadCampaigns = useCallback(async () => {
    if (!canRead) {
      setLoading(false);
      return;
    }
    setLoading(true);
    setError("");
    const params = new URLSearchParams({ page: String(page), limit: String(PAGE_SIZE) });
    if (query) params.set("q", query);
    if (status) params.set("status", status);
    if (campaignType) params.set("campaign_type", campaignType);
    if (scope) params.set("scope", scope);

    try {
      const response = await adminFetch<AdminPromotionListResponse>(`/promotions?${params.toString()}`);
      setCampaigns(response.data ?? []);
      setCampaignMeta(response.meta);
    } catch (value: unknown) {
      setCampaigns([]);
      setCampaignMeta(null);
      setError(messageForError(value));
    } finally {
      setLoading(false);
    }
  }, [campaignType, canRead, page, query, scope, status]);

  const loadDiscounts = useCallback(async () => {
    if (!canRead) {
      setLoading(false);
      return;
    }
    setLoading(true);
    setError("");
    const params = new URLSearchParams({ page: String(page), limit: String(PAGE_SIZE) });
    if (query) params.set("q", query);

    try {
      const response = await adminFetch<AdminProductDiscountListResponse>(`/promotions/product-discounts?${params.toString()}`);
      setDiscounts(response.data ?? []);
      setDiscountMeta(response.meta);
    } catch (value: unknown) {
      setDiscounts([]);
      setDiscountMeta(null);
      setError(messageForError(value));
    } finally {
      setLoading(false);
    }
  }, [canRead, page, query]);

  useEffect(() => {
    if (tab === "campaigns") void loadCampaigns();
    else void loadDiscounts();
  }, [loadCampaigns, loadDiscounts, tab]);

  useEffect(() => {
    setPage(1);
    setError("");
  }, [tab]);

  async function openPromotion(promotion: AdminPromotion) {
    if (!canManage) return;
    setError("");
    try {
      const response = await adminFetch<AdminPromotionResponse>(`/promotions/${promotion.id}`);
      setSelectedPromotion(response.data);
      setEditorMode("edit");
    } catch (value: unknown) {
      setError(messageForError(value));
    }
  }

  async function clearDiscount(discount: AdminProductDiscount) {
    if (!canManage) return;
    if (!window.confirm(`Remove the product discount from ${discount.product_name} · ${discount.sku}?`)) return;

    setClearingVariant(discount.variant_id);
    setError("");
    try {
      await adminFetch<AdminProductDiscountResponse>(
        `/promotions/product-discounts/${discount.variant_id}`,
        { method: "DELETE" },
      );
      await loadDiscounts();
    } catch (value: unknown) {
      setError(messageForError(value));
    } finally {
      setClearingVariant("");
    }
  }

  const meta = tab === "campaigns" ? campaignMeta : discountMeta;
  const liveCount = useMemo(() => campaigns.filter((item) => item.effective_state === "live").length, [campaigns]);
  const flashCount = useMemo(() => campaigns.filter((item) => item.campaign_type === "flash_sale").length, [campaigns]);

  if (!canRead) {
    return (
      <div className={styles.workspace}>
        <AdminPageHeader
          eyebrow="Commerce"
          title="Promotions & discounts"
          description="Your staff account does not have permission to view promotion controls."
        />
        <div className={styles.permissionState}>Promotion access requires <code>admin.promotion.read</code>.</div>
      </div>
    );
  }

  return (
    <div className={styles.workspace}>
      <AdminPageHeader
        eyebrow="Commerce"
        title="Promotions & discounts"
        description="Manage checkout campaigns, time-boxed flash sales and direct catalog price markdowns without changing the underlying product catalog structure."
        actions={
          canManage ? (
            <button
              type="button"
              className={styles.primaryButton}
              onClick={() => {
                if (tab === "campaigns") {
                  setSelectedPromotion(null);
                  setEditorMode("create");
                } else {
                  setDiscountEditorOpen(true);
                }
              }}
            >
              {tab === "campaigns" ? "New promotion" : "New product discount"}
            </button>
          ) : null
        }
      />

      <div className={styles.tabBar} role="tablist" aria-label="Promotion controls">
        <button type="button" role="tab" aria-selected={tab === "campaigns"} className={tab === "campaigns" ? styles.tabActive : styles.tab} onClick={() => setTab("campaigns")}>Campaigns</button>
        <button type="button" role="tab" aria-selected={tab === "discounts"} className={tab === "discounts" ? styles.tabActive : styles.tab} onClick={() => setTab("discounts")}>Product discounts</button>
      </div>

      {tab === "campaigns" ? (
        <div className={styles.summaryGrid}>
          <div className={styles.summaryCard}><span>Visible page</span><strong>{campaigns.length}</strong></div>
          <div className={styles.summaryCard}><span>Live now</span><strong>{liveCount}</strong></div>
          <div className={styles.summaryCard}><span>Flash sales</span><strong>{flashCount}</strong></div>
          <div className={styles.summaryCard}><span>Total campaigns</span><strong>{campaignMeta?.total ?? "—"}</strong></div>
        </div>
      ) : (
        <div className={styles.summaryGrid}>
          <div className={styles.summaryCard}><span>Active markdowns</span><strong>{discountMeta?.total ?? discounts.length}</strong></div>
          <div className={styles.summaryCard}><span>Visible page</span><strong>{discounts.length}</strong></div>
          <div className={styles.summaryCardWide}><span>Pricing model</span><strong>Variant sale price + compare-at regular price</strong></div>
        </div>
      )}

      <section className={styles.controlBar}>
        <label className={styles.searchBox}>
          <span aria-hidden="true">⌕</span>
          <input value={queryInput} onChange={(event) => setQueryInput(event.target.value)} placeholder={tab === "campaigns" ? "Search code or promotion ID" : "Search product, code, SKU or ID"} />
        </label>

        {tab === "campaigns" ? (
          <div className={styles.filters}>
            <select value={status} onChange={(event) => { setStatus(event.target.value); setPage(1); }} aria-label="Promotion status">
              <option value="">All statuses</option>
              <option value="draft">Draft</option>
              <option value="active">Active</option>
              <option value="disabled">Disabled</option>
            </select>
            <select value={campaignType} onChange={(event) => { setCampaignType(event.target.value); setPage(1); }} aria-label="Campaign type">
              <option value="">All campaign types</option>
              <option value="standard">Standard</option>
              <option value="flash_sale">Flash sale</option>
            </select>
            <select value={scope} onChange={(event) => { setScope(event.target.value); setPage(1); }} aria-label="Promotion scope">
              <option value="">All scopes</option>
              <option value="order">Order</option>
              <option value="product">Product</option>
            </select>
          </div>
        ) : null}
      </section>

      {error ? <div className={styles.errorBanner}>{error}</div> : null}

      {loading ? (
        <div className={styles.loadingState}>Loading promotion controls…</div>
      ) : tab === "campaigns" ? (
        campaigns.length ? (
          <div className={styles.campaignList}>
            {campaigns.map((promotion) => (
              <article key={promotion.id} className={styles.campaignCard}>
                <div className={styles.campaignMain}>
                  <div className={styles.campaignTitleRow}>
                    <span className={`${styles.stateBadge} ${styles[`state_${promotion.effective_state}`]}`}>{stateLabel(promotion.effective_state)}</span>
                    {promotion.campaign_type === "flash_sale" ? <span className={styles.flashBadge}>Flash sale</span> : null}
                    <span className={styles.modeBadge}>{promotion.mode === "code" ? promotion.code : "Automatic"}</span>
                  </div>
                  <h2>{promotion.name}</h2>
                  <p>{promotionValue(promotion)} · {promotion.scope === "product" ? `${promotion.target_count} target${promotion.target_count === 1 ? "" : "s"}` : "whole order"}</p>
                </div>

                <div className={styles.campaignSchedule}>
                  <span>Starts</span><strong>{formatDateTime(promotion.starts_at)}</strong>
                  <span>Ends</span><strong>{formatDateTime(promotion.ends_at)}</strong>
                </div>

                <div className={styles.campaignActions}>
                  <span>{promotion.status}</span>
                  {canManage ? <button type="button" className={styles.secondaryButton} onClick={() => void openPromotion(promotion)}>Edit</button> : null}
                </div>
              </article>
            ))}
          </div>
        ) : (
          <div className={styles.emptyState}><strong>No promotions found.</strong><span>Create a campaign or adjust the filters.</span></div>
        )
      ) : discounts.length ? (
        <div className={styles.discountTableWrap}>
          <table className={styles.discountTable}>
            <thead><tr><th>Product</th><th>Regular</th><th>Sale price</th><th>Discount</th><th>Status</th><th /></tr></thead>
            <tbody>
              {discounts.map((discount) => (
                <tr key={discount.variant_id}>
                  <td data-label="Product"><strong>{discount.product_name}</strong><span>{discount.product_code} · {discount.sku}</span></td>
                  <td data-label="Regular">{formatMoney(discount.regular_price_amount, discount.currency)}</td>
                  <td data-label="Sale price"><strong>{formatMoney(discount.sale_price_amount, discount.currency)}</strong></td>
                  <td data-label="Discount">{formatPercentBPS(discount.discount_bps)} · {formatMoney(discount.discount_amount, discount.currency)}</td>
                  <td data-label="Status"><span className={styles.liveDot} />{discount.variant_active && discount.product_status === "active" ? "Live catalog" : "Not storefront-active"}</td>
                  <td className={styles.tableAction}>{canManage ? <button type="button" className={styles.dangerLink} disabled={clearingVariant === discount.variant_id} onClick={() => void clearDiscount(discount)}>{clearingVariant === discount.variant_id ? "Removing…" : "Remove"}</button> : null}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : (
        <div className={styles.emptyState}><strong>No active product discounts.</strong><span>Apply a markdown to a product variant when you are ready.</span></div>
      )}

      {meta && meta.total_pages > 1 ? (
        <footer className={styles.pagination}>
          <button type="button" className={styles.secondaryButton} disabled={!meta.has_previous} onClick={() => setPage((current) => Math.max(1, current - 1))}>Previous</button>
          <span>Page {meta.page} of {meta.total_pages}</span>
          <button type="button" className={styles.secondaryButton} disabled={!meta.has_next} onClick={() => setPage((current) => current + 1)}>Next</button>
        </footer>
      ) : null}

      {editorMode ? (
        <PromotionEditor
          mode={editorMode}
          promotion={selectedPromotion}
          onClose={() => { setEditorMode(null); setSelectedPromotion(null); }}
          onSaved={() => { setEditorMode(null); setSelectedPromotion(null); void loadCampaigns(); }}
        />
      ) : null}

      {discountEditorOpen ? (
        <DiscountEditor
          onClose={() => setDiscountEditorOpen(false)}
          onSaved={() => { setDiscountEditorOpen(false); void loadDiscounts(); }}
        />
      ) : null}
    </div>
  );
}
