// Location: src/components/product/components/ProductPurchasePanel.tsx
"use client";

import Link from "next/link";
import { useMemo, useState } from "react";
import { useRouter } from "next/navigation";

import { notifyCartUpdated } from "@/components/commerce/CartBadge";
import { Icon } from "@/components/ui/Icon";
import type { Cart } from "@/lib/api/contracts/commerce";
import type { ProductVariant } from "@/lib/api/contracts/catalog";
import { formatMoney } from "@/lib/money/format";

import styles from "../css/ProductPurchasePanel.module.css";

type Props = {
  productName: string;
  variants: ProductVariant[];
  selectedVariant: ProductVariant;
  quantity: number;
  onSelectVariant: (variant: ProductVariant) => void;
  onQuantityChange: (quantity: number) => void;
};

type ApiErrorPayload = {
  error?: {
    code?: string;
    message?: string;
  };
};

function effectivePrice(variant: ProductVariant, quantity: number) {
  let price = variant.price_amount;

  for (const tier of [...variant.price_tiers].sort(
    (a, b) => a.min_quantity - b.min_quantity,
  )) {
    if (quantity >= tier.min_quantity) price = tier.unit_price_amount;
  }

  return price;
}

function discountPercent(variant: ProductVariant) {
  const compare = variant.compare_at_price_amount;

  if (!compare || compare <= variant.price_amount) return 0;

  return Math.round(((compare - variant.price_amount) / compare) * 100);
}

function normalize(value?: string | null) {
  return value?.trim().toLocaleLowerCase() ?? "";
}

async function readError(response: Response) {
  try {
    const payload = (await response.json()) as ApiErrorPayload;
    return payload.error?.message ?? "Unable to complete this action.";
  } catch {
    return "Unable to complete this action.";
  }
}

export function ProductPurchasePanel({
  productName,
  variants,
  selectedVariant,
  quantity,
  onSelectVariant,
  onQuantityChange,
}: Props) {
  const router = useRouter();
  const [busy, setBusy] = useState<"cart" | "buy" | null>(null);
  const [message, setMessage] = useState("");
  const [error, setError] = useState("");
  const [showPriceBreaks, setShowPriceBreaks] = useState(false);

  const unitPrice = useMemo(
    () => effectivePrice(selectedVariant, quantity),
    [selectedVariant, quantity],
  );

  const discount = discountPercent(selectedVariant);
  const moq = Math.max(1, selectedVariant.minimum_order_quantity);
  const stock = Math.max(0, selectedVariant.available_quantity);
  const subtotal = unitPrice * Math.max(0, quantity);
  const available = selectedVariant.in_stock && stock >= moq;
  const belowMoq = quantity < moq;
  const aboveStock = quantity > stock;
  const canBuy = available && !belowMoq && !aboveStock;

  const selectedColor = normalize(selectedVariant.color_name);
  const selectedSize = selectedVariant.size?.trim() ?? "";

  const colorOptions = useMemo(() => {
    const names = Array.from(
      new Set(
        variants
          .map((variant) => variant.color_name?.trim())
          .filter((value): value is string => Boolean(value)),
      ),
    );

    return names.map((label) => {
      const key = normalize(label);
      const candidates = variants.filter(
        (variant) => normalize(variant.color_name) === key,
      );

      const variant =
        candidates.find((candidate) => candidate.size?.trim() === selectedSize) ??
        candidates.find(
          (candidate) =>
            candidate.in_stock && candidate.available_quantity > 0,
        ) ??
        candidates[0];

      return {
        key,
        label,
        variant,
      };
    });
  }, [selectedSize, variants]);

  const sizeOptions = useMemo(() => {
    const candidates = selectedColor
      ? variants.filter((variant) => normalize(variant.color_name) === selectedColor)
      : variants;

    const sizes = Array.from(
      new Set(
        candidates
          .map((variant) => variant.size?.trim())
          .filter((value): value is string => Boolean(value)),
      ),
    );

    return sizes.map((label) => {
      const variant =
        candidates.find((candidate) => candidate.size?.trim() === label) ??
        variants.find((candidate) => candidate.size?.trim() === label);

      return { label, variant };
    });
  }, [selectedColor, variants]);

  const hasStructuredOptions = colorOptions.length > 0 || sizeOptions.length > 0;

  const sortedTiers = useMemo(
    () => [...selectedVariant.price_tiers].sort((a, b) => a.min_quantity - b.min_quantity),
    [selectedVariant.price_tiers],
  );

  function setQuantity(value: number) {
    if (!Number.isFinite(value)) return;

    onQuantityChange(Math.max(0, Math.floor(value)));
    setMessage("");
    setError("");
  }

  function decrease() {
    setQuantity(Math.max(moq, quantity - 1));
  }

  function increase() {
    setQuantity(Math.min(stock, quantity + 1));
  }

  async function addToCart() {
    if (!canBuy || busy) return;

    setBusy("cart");
    setMessage("");
    setError("");

    try {
      const response = await fetch("/api/storefront/cart/items", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          variant_id: selectedVariant.id,
          quantity,
        }),
      });

      if (!response.ok) throw new Error(await readError(response));

      const payload = (await response.json()) as { data: Cart };
      notifyCartUpdated(payload.data);
      setMessage(`${productName} added to cart.`);
    } catch (caught) {
      setError(
        caught instanceof Error ? caught.message : "Unable to add to cart.",
      );
    } finally {
      setBusy(null);
    }
  }

  async function buyNow() {
    if (!canBuy || busy) return;

    setBusy("buy");
    setMessage("");
    setError("");

    try {
      const response = await fetch("/api/storefront/checkout/buy-now", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          variant_id: selectedVariant.id,
          quantity,
        }),
      });

      if (response.status === 401) {
        setBusy(null);
        const next = encodeURIComponent(
          `${window.location.pathname}${window.location.search}`,
        );
        router.push(`/account/sign-in?next=${next}`);
        return;
      }

      if (!response.ok) throw new Error(await readError(response));

      router.push("/checkout");
      router.refresh();
    } catch (caught) {
      setError(
        caught instanceof Error ? caught.message : "Unable to start checkout.",
      );
      setBusy(null);
    }
  }

  return (
    <section className={styles.panel} aria-label="Purchase options">
      <div className={styles.priceRow}>
        <strong className={styles.currentPrice}>
          {formatMoney(unitPrice, selectedVariant.currency)}
        </strong>
        <span className={styles.unit}>/ unit</span>

        {selectedVariant.compare_at_price_amount &&
        selectedVariant.compare_at_price_amount > unitPrice ? (
          <del className={styles.comparePrice}>
            {formatMoney(
              selectedVariant.compare_at_price_amount,
              selectedVariant.currency,
            )}
          </del>
        ) : null}

        {discount > 0 ? <span className={styles.discount}>-{discount}%</span> : null}
      </div>

      {colorOptions.length > 0 ? (
        <div className={styles.optionSection}>
          <div className={styles.optionHeading}>
            <strong>Color (variant)</strong>
            <small>Each color is a separate variant.</small>
          </div>

          <div className={styles.variantRail}>
            {colorOptions.map((option) => {
              const selected = normalize(selectedVariant.color_name) === option.key;
              const optionInStock = Boolean(
                option.variant &&
                  option.variant.in_stock &&
                  option.variant.available_quantity >=
                    Math.max(1, option.variant.minimum_order_quantity),
              );

              return (
                <button
                  key={option.key}
                  type="button"
                  className={`${styles.variantCard} ${
                    selected ? styles.variantCardSelected : ""
                  }`}
                  aria-pressed={selected}
                  onClick={() => {
                    if (option.variant) onSelectVariant(option.variant);
                  }}
                >
                  <strong>{option.label}</strong>
                  <span>{option.variant?.sku ?? "Variant"}</span>
                  <small
                    className={
                      optionInStock ? styles.inStock : styles.outOfStock
                    }
                  >
                    {optionInStock ? "In stock" : "Out of stock"}
                  </small>
                </button>
              );
            })}
          </div>
        </div>
      ) : null}

      {sizeOptions.length > 0 ? (
        <div className={styles.optionSection}>
          <div className={styles.optionHeading}>
            <strong>Size</strong>
          </div>

          <div className={styles.sizeRail}>
            {sizeOptions.map((option) => {
              const selected = selectedVariant.size?.trim() === option.label;

              return (
                <button
                  key={option.label}
                  type="button"
                  className={`${styles.sizeButton} ${
                    selected ? styles.sizeButtonSelected : ""
                  }`}
                  aria-pressed={selected}
                  onClick={() => {
                    if (option.variant) onSelectVariant(option.variant);
                  }}
                >
                  {option.label}
                </button>
              );
            })}
          </div>
        </div>
      ) : null}

      {!hasStructuredOptions && variants.length > 1 ? (
        <div className={styles.optionSection}>
          <div className={styles.optionHeading}>
            <strong>Variant</strong>
          </div>

          <div className={styles.variantRail}>
            {variants.map((variant) => {
              const selected = variant.id === selectedVariant.id;
              const optionInStock =
                variant.in_stock &&
                variant.available_quantity >= Math.max(1, variant.minimum_order_quantity);

              return (
                <button
                  key={variant.id}
                  type="button"
                  className={`${styles.variantCard} ${
                    selected ? styles.variantCardSelected : ""
                  }`}
                  aria-pressed={selected}
                  onClick={() => onSelectVariant(variant)}
                >
                  <strong>{variant.sku}</strong>
                  <small
                    className={
                      optionInStock ? styles.inStock : styles.outOfStock
                    }
                  >
                    {optionInStock ? "In stock" : "Out of stock"}
                  </small>
                </button>
              );
            })}
          </div>
        </div>
      ) : null}

      <dl className={styles.facts}>
        <div>
          <dt>MOQ</dt>
          <dd>{moq.toLocaleString()} units</dd>
        </div>
        <div>
          <dt>Available</dt>
          <dd className={available ? styles.stockValue : styles.stockValueEmpty}>
            {stock.toLocaleString()} units
          </dd>
        </div>
        <div>
          <dt>SKU</dt>
          <dd>{selectedVariant.sku}</dd>
        </div>
      </dl>

      <div className={styles.quantitySection}>
        <strong className={styles.quantityLabel}>Quantity</strong>

        <div className={styles.quantityLine}>
          <div className={styles.quantityControl}>
            <button
              type="button"
              aria-label="Decrease quantity"
              disabled={quantity <= moq || !available}
              onClick={decrease}
            >
              <Icon name="minus" size={14} />
            </button>

            <input
              type="number"
              inputMode="numeric"
              min={moq}
              max={stock}
              step={1}
              value={quantity}
              aria-label="Order quantity"
              aria-describedby="product-quantity-help"
              onChange={(event) => setQuantity(Number(event.target.value))}
              onBlur={() => {
                if (quantity < moq) setQuantity(moq);
                if (stock >= moq && quantity > stock) setQuantity(stock);
              }}
            />

            <button
              type="button"
              aria-label="Increase quantity"
              disabled={!available || quantity >= stock}
              onClick={increase}
            >
              <Icon name="plus" size={14} />
            </button>
          </div>

          <small id="product-quantity-help" className={styles.quantityHelp}>
            Min. {moq.toLocaleString()}
            <span aria-hidden="true">·</span>
            Max. {stock.toLocaleString()}
          </small>
        </div>
      </div>

      {sortedTiers.length > 0 ? (
        <div className={styles.priceBreakSection}>
          <button
            type="button"
            className={styles.priceBreakToggle}
            aria-expanded={showPriceBreaks}
            onClick={() => setShowPriceBreaks((current) => !current)}
          >
            <span>Price breaks (per unit)</span>
            <Icon name="chevronRight" size={13} />
          </button>

          <div
            className={`${styles.priceBreakPreview} ${
              showPriceBreaks ? styles.priceBreakPreviewOpen : ""
            }`}
          >
            <div>
              <span>{moq}+ units</span>
              <strong>{formatMoney(selectedVariant.price_amount, selectedVariant.currency)}</strong>
            </div>

            {sortedTiers.slice(0, showPriceBreaks ? undefined : 2).map((tier) => (
              <div key={tier.min_quantity}>
                <span>{tier.min_quantity}+ units</span>
                <strong>
                  {formatMoney(tier.unit_price_amount, selectedVariant.currency)}
                </strong>
              </div>
            ))}
          </div>
        </div>
      ) : null}

      {belowMoq ? (
        <p className={styles.error} role="alert">
          Minimum order is {moq.toLocaleString()} units.
        </p>
      ) : null}

      {aboveStock ? (
        <p className={styles.error} role="alert">
          Only {stock.toLocaleString()} units are available.
        </p>
      ) : null}

      <div className={styles.actions}>
        <button
          type="button"
          className={styles.addToCart}
          disabled={!canBuy || busy !== null}
          onClick={() => void addToCart()}
        >
          <Icon name="cart" size={17} />
          <span>{busy === "cart" ? "Adding…" : "Add to Cart"}</span>
        </button>

        <button
          type="button"
          className={styles.buyNow}
          disabled={!canBuy || busy !== null}
          onClick={() => void buyNow()}
        >
          {busy === "buy" ? "Opening…" : "Buy Now"}
        </button>
      </div>

      {!available ? (
        <Link className={styles.requestLink} href="/account/request">
          Need this variant? Request bulk sourcing
        </Link>
      ) : null}

      <div className={styles.trustStrip} aria-label="Purchase assurances">
        <span><Icon name="secureCheckout" size={14} />Secure payment</span>
        <span><Icon name="delivery" size={14} />Reliable delivery</span>
        <span><Icon name="support" size={14} />Business support</span>
      </div>

      <p className={styles.subtotal}>
        Current subtotal: <strong>{formatMoney(subtotal, selectedVariant.currency)}</strong>
      </p>

      {message ? <p className={styles.success} role="status">{message}</p> : null}
      {error ? <p className={styles.error} role="alert">{error}</p> : null}
    </section>
  );
}
