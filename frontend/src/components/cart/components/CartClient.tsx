// Location: src/components/cart/components/CartClient.tsx
"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useLayoutEffect, useMemo, useState } from "react";

import { notifyCartUpdated } from "@/components/commerce/CartBadge";
import { CatalogImage } from "@/components/commerce/components/CatalogImage";
import { Icon } from "@/components/ui/Icon";
import type { Cart, CartItem } from "@/lib/api/contracts/commerce";
import { formatMoney } from "@/lib/money/format";

import styles from "@/components/cart/css/CartPage.module.css";

type Props = {
  initialCart: Cart | null;
  initialLoadError?: boolean;
};

type ApiErrorPayload = {
  error?: {
    code?: string;
    message?: string;
  };
};

type QuantityDrafts = Record<string, string>;
type ItemErrors = Record<string, string>;

function quantityDrafts(cart: Cart | null): QuantityDrafts {
  if (!cart) {
    return {};
  }

  return Object.fromEntries(
    cart.items.map((item) => [item.id, String(item.quantity)]),
  );
}

function itemBlocked(item: CartItem) {
  const moq = Math.max(1, item.minimum_order_quantity);

  return (
    !item.is_available ||
    item.available_quantity < moq ||
    item.quantity < moq ||
    item.quantity > item.available_quantity
  );
}

function variantParts(item: CartItem) {
  return [
    item.color_name?.trim() || null,
    item.size?.trim() ? `Size ${item.size.trim()}` : null,
    item.sku?.trim() ? `SKU: ${item.sku.trim()}` : null,
  ].filter((part): part is string => Boolean(part));
}

async function readError(response: Response, fallback: string) {
  try {
    const payload = (await response.json()) as ApiErrorPayload;
    return payload.error?.message?.trim() || fallback;
  } catch {
    return fallback;
  }
}

function TrashIcon() {
  return (
    <svg
      viewBox="0 0 24 24"
      aria-hidden="true"
      focusable="false"
      className={styles.trashIcon}
    >
      <path d="M4 7h16" />
      <path d="M9 7V4h6v3" />
      <path d="m6.5 7 .8 13h9.4l.8-13" />
      <path d="M10 11v5M14 11v5" />
    </svg>
  );
}

export function CartClient({
  initialCart,
  initialLoadError = false,
}: Props) {
  const router = useRouter();

  const [cart, setCart] = useState<Cart | null>(initialCart);
  const [drafts, setDrafts] = useState<QuantityDrafts>(() =>
    quantityDrafts(initialCart),
  );
  const [busyItemID, setBusyItemID] = useState<string | null>(null);
  const [checkoutBusy, setCheckoutBusy] = useState(false);
  const [reloadBusy, setReloadBusy] = useState(false);
  const [loadFailed, setLoadFailed] = useState(initialLoadError);
  const [globalError, setGlobalError] = useState("");
  const [itemErrors, setItemErrors] = useState<ItemErrors>({});
  const [statusMessage, setStatusMessage] = useState("");

  /*
   * Cart is already the current destination, so its shortcut is hidden from
   * the shared header on this route. The class is removed when the page unmounts.
   */
  useLayoutEffect(() => {
    document.documentElement.classList.add("ene-cart-route");

    return () => {
      document.documentElement.classList.remove("ene-cart-route");
    };
  }, []);

  const blockingItems = useMemo(
    () => cart?.items.filter(itemBlocked) ?? [],
    [cart],
  );

  const hasPendingDraft = useMemo(
    () =>
      cart?.items.some(
        (item) => (drafts[item.id] ?? String(item.quantity)) !== String(item.quantity),
      ) ?? false,
    [cart, drafts],
  );

  const mutationBusy = busyItemID !== null;
  const checkoutBlocked =
    !cart ||
    cart.items.length === 0 ||
    blockingItems.length > 0 ||
    hasPendingDraft ||
    mutationBusy ||
    checkoutBusy;

  function clearMessages() {
    setGlobalError("");
    setStatusMessage("");
  }

  function syncCart(nextCart: Cart | null) {
    setCart(nextCart);
    setDrafts(quantityDrafts(nextCart));
    setItemErrors({});
    notifyCartUpdated(nextCart);
  }

  function setItemError(itemID: string, message: string) {
    setItemErrors((current) => ({
      ...current,
      [itemID]: message,
    }));
  }

  function clearItemError(itemID: string) {
    setItemErrors((current) => {
      if (!current[itemID]) {
        return current;
      }

      const next = { ...current };
      delete next[itemID];
      return next;
    });
  }

  async function reloadCart() {
    if (reloadBusy) {
      return;
    }

    setReloadBusy(true);
    clearMessages();

    try {
      const response = await fetch("/api/storefront/cart", {
        cache: "no-store",
      });

      if (!response.ok) {
        throw new Error(await readError(response, "Unable to load your cart."));
      }

      const payload = (await response.json()) as { data: Cart | null };
      syncCart(payload.data);
      setLoadFailed(false);
    } catch (caught) {
      setLoadFailed(true);
      setGlobalError(
        caught instanceof Error ? caught.message : "Unable to load your cart.",
      );
    } finally {
      setReloadBusy(false);
    }
  }

  async function updateQuantity(item: CartItem, quantity: number) {
    if (!cart || mutationBusy || checkoutBusy) {
      return;
    }

    const moq = Math.max(1, item.minimum_order_quantity);

    if (!Number.isInteger(quantity)) {
      setItemError(item.id, "Enter a whole-number quantity.");
      setDrafts((current) => ({
        ...current,
        [item.id]: String(item.quantity),
      }));
      return;
    }

    if (quantity < moq) {
      setItemError(
        item.id,
        `Minimum order is ${moq.toLocaleString()} units.`,
      );
      setDrafts((current) => ({
        ...current,
        [item.id]: String(item.quantity),
      }));
      return;
    }

    if (quantity > item.available_quantity) {
      setItemError(
        item.id,
        `Only ${item.available_quantity.toLocaleString()} units are currently available.`,
      );
      setDrafts((current) => ({
        ...current,
        [item.id]: String(item.quantity),
      }));
      return;
    }

    if (quantity === item.quantity) {
      clearItemError(item.id);
      setDrafts((current) => ({
        ...current,
        [item.id]: String(item.quantity),
      }));
      return;
    }

    setBusyItemID(item.id);
    clearMessages();
    clearItemError(item.id);

    try {
      const response = await fetch(
        `/api/storefront/cart/items/${encodeURIComponent(item.id)}`,
        {
          method: "PATCH",
          headers: {
            "Content-Type": "application/json",
          },
          body: JSON.stringify({ quantity }),
        },
      );

      if (!response.ok) {
        throw new Error(
          await readError(response, "Unable to update this cart item."),
        );
      }

      const payload = (await response.json()) as { data: Cart };
      syncCart(payload.data);
      setStatusMessage(`${item.product_name} quantity updated.`);
    } catch (caught) {
      setItemError(
        item.id,
        caught instanceof Error
          ? caught.message
          : "Unable to update this cart item.",
      );
      setDrafts((current) => ({
        ...current,
        [item.id]: String(item.quantity),
      }));
    } finally {
      setBusyItemID(null);
    }
  }

  function commitDraft(item: CartItem) {
    const raw = drafts[item.id]?.trim() ?? "";
    const quantity = Number(raw);

    if (!raw || !Number.isInteger(quantity)) {
      setItemError(item.id, "Enter a whole-number quantity.");
      setDrafts((current) => ({
        ...current,
        [item.id]: String(item.quantity),
      }));
      return;
    }

    void updateQuantity(item, quantity);
  }

  function decrease(item: CartItem) {
    const moq = Math.max(1, item.minimum_order_quantity);
    const target =
      item.quantity > item.available_quantity && item.available_quantity >= moq
        ? item.available_quantity
        : Math.max(moq, item.quantity - 1);

    void updateQuantity(item, target);
  }

  function increase(item: CartItem) {
    void updateQuantity(
      item,
      Math.min(item.available_quantity, item.quantity + 1),
    );
  }

  async function removeItem(item: CartItem) {
    if (!cart || mutationBusy || checkoutBusy) {
      return;
    }

    setBusyItemID(item.id);
    clearMessages();
    clearItemError(item.id);

    try {
      const response = await fetch(
        `/api/storefront/cart/items/${encodeURIComponent(item.id)}`,
        {
          method: "DELETE",
        },
      );

      if (!response.ok) {
        throw new Error(
          await readError(response, "Unable to remove this cart item."),
        );
      }

      const payload = (await response.json()) as { data: Cart };
      syncCart(payload.data);
      setStatusMessage(`${item.product_name} removed from cart.`);
    } catch (caught) {
      setItemError(
        item.id,
        caught instanceof Error
          ? caught.message
          : "Unable to remove this cart item.",
      );
    } finally {
      setBusyItemID(null);
    }
  }

  async function startCheckout() {
    if (checkoutBlocked || !cart) {
      return;
    }

    setCheckoutBusy(true);
    clearMessages();

    try {
      const response = await fetch("/api/storefront/checkout", {
        method: "POST",
      });

      if (response.status === 401) {
        setCheckoutBusy(false);
        router.push("/account/sign-in?next=%2Fcheckout");
        return;
      }

      if (!response.ok) {
        throw new Error(await readError(response, "Unable to start checkout."));
      }

      router.push("/checkout");
      router.refresh();
    } catch (caught) {
      setGlobalError(
        caught instanceof Error ? caught.message : "Unable to start checkout.",
      );
      setCheckoutBusy(false);
    }
  }

  if (loadFailed) {
    return (
      <div className={styles.page}>
        <div className="site-container">
          <section className={styles.statePanel} role="alert">
            <span className={styles.stateIcon} aria-hidden="true">
              <Icon name="cart" size={28} />
            </span>
            <h1>We couldn&apos;t load your cart.</h1>
            <p>
              Your cart has not been changed. Check the connection and try again.
            </p>
            {globalError ? (
              <p className={styles.stateError}>{globalError}</p>
            ) : null}
            <div className={styles.stateActions}>
              <button
                type="button"
                disabled={reloadBusy}
                onClick={() => void reloadCart()}
              >
                {reloadBusy ? "Loading..." : "Try again"}
              </button>
              <Link href="/">Continue shopping</Link>
            </div>
          </section>
        </div>
      </div>
    );
  }

  if (!cart || cart.items.length === 0) {
    return (
      <div className={styles.page}>
        <div className="site-container">
          <section className={styles.statePanel}>
            <span className={styles.stateIcon} aria-hidden="true">
              <Icon name="cart" size={30} />
            </span>
            <h1>Your cart is empty.</h1>
            <p>
              Add products to start your order. Variant, MOQ and stock details
              will appear here before checkout.
            </p>
            <div className={styles.stateActions}>
              <Link className={styles.primaryStateLink} href="/">
                Browse products
              </Link>
            </div>
            {statusMessage ? (
              <p className={styles.visuallyHidden} role="status">
                {statusMessage}
              </p>
            ) : null}
          </section>
        </div>
      </div>
    );
  }

  return (
    <div className={styles.page}>
      <div className="site-container">
        <header className={styles.pageHeader}>
          <div>
            <h1>Your cart</h1>
            <p>Review your items and proceed to checkout.</p>
          </div>

          <Link className={styles.continueLink} href="/">
            <Icon name="chevronLeft" size={14} />
            <span>Continue shopping</span>
          </Link>
        </header>

        {globalError ? (
          <div className={styles.alert} role="alert">
            <strong>Cart action failed</strong>
            <span>{globalError}</span>
          </div>
        ) : null}

        {blockingItems.length > 0 ? (
          <div className={styles.warning} role="status">
            <strong>
              Review {blockingItems.length === 1 ? "1 cart item" : `${blockingItems.length} cart items`}
            </strong>
            <span>
              Update unavailable or invalid quantities before continuing to checkout.
            </span>
          </div>
        ) : null}

        <div className={styles.layout}>
          <div className={styles.itemsColumn}>
            <section
              className={styles.itemsCard}
              aria-labelledby="cart-items-heading"
            >
              <h2 id="cart-items-heading" className={styles.visuallyHidden}>
                Cart items
              </h2>

              <div className={styles.columnHeader} aria-hidden="true">
                <span>Product</span>
                <span>Price</span>
                <span>Quantity</span>
                <span>Total</span>
              </div>

              <div className={styles.itemList}>
                {cart.items.map((item) => {
                  const moq = Math.max(1, item.minimum_order_quantity);
                  const busy = busyItemID === item.id;
                  const blocked = itemBlocked(item);
                  const adjustable = item.available_quantity >= moq;
                  const currentError = itemErrors[item.id] ?? "";
                  const parts = variantParts(item);
                  const stockOK =
                    item.is_available && item.available_quantity >= moq;

                  return (
                    <article
                      key={item.id}
                      className={`${styles.itemRow} ${blocked ? styles.itemBlocked : ""}`}
                      aria-busy={busy}
                    >
                      <div className={styles.productCell}>
                        <Link
                          href={`/product/${item.product_slug}`}
                          className={styles.itemMedia}
                          aria-label={`View ${item.product_name}`}
                        >
                          {item.image_url ? (
                            <CatalogImage
                              src={item.image_url}
                              alt={item.product_name}
                              fill
                              sizes="(max-width: 48rem) 5.2rem, 8rem"
                              className={styles.itemImage}
                            />
                          ) : (
                            <span className={styles.imageFallback} aria-hidden="true">
                              {item.product_name.charAt(0).toUpperCase()}
                            </span>
                          )}
                        </Link>

                        <div className={styles.productInfo}>
                          <Link
                            href={`/product/${item.product_slug}`}
                            className={styles.productName}
                          >
                            {item.product_name}
                          </Link>

                          {parts.length > 0 ? (
                            <div className={styles.variantMeta}>
                              {parts.map((part) => (
                                <span key={part}>{part}</span>
                              ))}
                            </div>
                          ) : null}

                          <div className={styles.itemFacts}>
                            <span className={styles.moqBadge}>
                              MOQ {moq.toLocaleString()}
                            </span>
                            <span
                              className={stockOK ? styles.stockIn : styles.stockOut}
                            >
                              <i aria-hidden="true" />
                              {stockOK
                                ? `In stock (${item.available_quantity.toLocaleString()} available)`
                                : "Out of stock"}
                            </span>
                          </div>
                        </div>
                      </div>

                      <div className={styles.unitPriceCell}>
                        <strong>
                          {formatMoney(item.unit_price_amount, item.currency)}
                        </strong>
                        <span>/ unit</span>
                      </div>

                      <div className={styles.quantityCell}>
                        <div className={styles.quantityControl}>
                          <button
                            type="button"
                            aria-label={`Decrease ${item.product_name} quantity`}
                            disabled={
                              mutationBusy ||
                              checkoutBusy ||
                              !adjustable ||
                              item.quantity <= moq
                            }
                            onClick={() => decrease(item)}
                          >
                            <Icon name="minus" size={13} />
                          </button>

                          <input
                            type="number"
                            inputMode="numeric"
                            min={moq}
                            max={item.available_quantity}
                            step={1}
                            value={drafts[item.id] ?? String(item.quantity)}
                            aria-label={`${item.product_name} quantity`}
                            aria-invalid={Boolean(currentError)}
                            aria-describedby={
                              currentError ? `cart-item-error-${item.id}` : undefined
                            }
                            disabled={mutationBusy || checkoutBusy || !adjustable}
                            onChange={(event) => {
                              setDrafts((current) => ({
                                ...current,
                                [item.id]: event.target.value,
                              }));
                              clearItemError(item.id);
                              clearMessages();
                            }}
                            onBlur={() => commitDraft(item)}
                            onKeyDown={(event) => {
                              if (event.key === "Enter") {
                                event.currentTarget.blur();
                              }

                              if (event.key === "Escape") {
                                setDrafts((current) => ({
                                  ...current,
                                  [item.id]: String(item.quantity),
                                }));
                                clearItemError(item.id);
                                event.currentTarget.blur();
                              }
                            }}
                          />

                          <button
                            type="button"
                            aria-label={`Increase ${item.product_name} quantity`}
                            disabled={
                              mutationBusy ||
                              checkoutBusy ||
                              !adjustable ||
                              item.quantity >= item.available_quantity
                            }
                            onClick={() => increase(item)}
                          >
                            <Icon name="plus" size={13} />
                          </button>
                        </div>

                        <span className={styles.quantityLimits}>
                          Min. {moq.toLocaleString()}
                          <i aria-hidden="true">•</i>
                          Max. {item.available_quantity.toLocaleString()}
                        </span>

                        {currentError ? (
                          <p
                            id={`cart-item-error-${item.id}`}
                            className={styles.itemError}
                            role="alert"
                          >
                            {currentError}
                          </p>
                        ) : null}
                      </div>

                      <div className={styles.totalCell}>
                        <strong>
                          {formatMoney(item.line_total_amount, item.currency)}
                        </strong>

                        <button
                          type="button"
                          className={styles.removeButton}
                          aria-label={`Remove ${item.product_name} from cart`}
                          title={`Remove ${item.product_name}`}
                          disabled={mutationBusy || checkoutBusy}
                          onClick={() => void removeItem(item)}
                        >
                          <TrashIcon />
                        </button>
                      </div>
                    </article>
                  );
                })}
              </div>
            </section>

            <section className={styles.trustStrip} aria-label="Shopping benefits">
              <div>
                <span className={styles.trustIcon} aria-hidden="true">
                  <Icon name="secureCheckout" size={20} />
                </span>
                <span>
                  <strong>Secure checkout</strong>
                  <small>Order data is protected</small>
                </span>
              </div>

              <div>
                <span className={styles.trustIcon} aria-hidden="true">
                  <Icon name="orders" size={20} />
                </span>
                <span>
                  <strong>Business ready</strong>
                  <small>MOQ and stock stay visible</small>
                </span>
              </div>

              <div>
                <span className={styles.trustIcon} aria-hidden="true">
                  <Icon name="delivery" size={20} />
                </span>
                <span>
                  <strong>Delivery options</strong>
                  <small>Confirmed at checkout</small>
                </span>
              </div>

              <div>
                <span className={styles.trustIcon} aria-hidden="true">
                  <Icon name="support" size={20} />
                </span>
                <span>
                  <strong>Expert support</strong>
                  <small>Help when you need it</small>
                </span>
              </div>
            </section>
          </div>

          <aside className={styles.summary} aria-labelledby="cart-summary-heading">
            <h2 id="cart-summary-heading">Order summary</h2>

            <div className={styles.summaryRows}>
              <div>
                <span>Product lines</span>
                <strong>{cart.totals.item_count.toLocaleString()}</strong>
              </div>
              <div>
                <span>Total units</span>
                <strong>{cart.totals.quantity_total.toLocaleString()}</strong>
              </div>
            </div>

            <div className={styles.subtotalRow}>
              <span>Subtotal</span>
              <strong>
                {formatMoney(cart.totals.subtotal_amount, cart.totals.currency)}
              </strong>
            </div>

            <p className={styles.summaryNote}>
              <Icon name="delivery" size={16} />
              <span>Shipping, taxes and discounts are calculated at checkout.</span>
            </p>

            <button
              type="button"
              className={styles.checkoutButton}
              disabled={checkoutBlocked}
              onClick={() => void startCheckout()}
            >
              <span>{checkoutBusy ? "Opening checkout..." : "Continue to checkout"}</span>
              <Icon name="chevronRight" size={17} />
            </button>

            {blockingItems.length > 0 ? (
              <p className={styles.checkoutHelp}>
                Resolve the highlighted cart {blockingItems.length === 1 ? "item" : "items"} first.
              </p>
            ) : hasPendingDraft ? (
              <p className={styles.checkoutHelp}>
                Finish the quantity edit before continuing.
              </p>
            ) : null}

            <div className={styles.summarySecurity}>
              <Icon name="secureCheckout" size={15} />
              <span>Price, MOQ and stock are revalidated as checkout begins.</span>
            </div>
          </aside>
        </div>

        <section className={styles.supportStrip} aria-label="Cart support">
          <div>
            <strong>Need help?</strong>
            <span>Contact customer support about your order.</span>
            <Link href="/account/support">Open support</Link>
          </div>

          <div>
            <strong>Have a large request?</strong>
            <span>Ask us to help source more stock or another product.</span>
            <Link href="/account/request">Request sourcing</Link>
          </div>

          <div>
            <strong>Checkout clarity</strong>
            <span>Delivery, discounts and payment choices are confirmed before order placement.</span>
          </div>
        </section>
      </div>

      <div className={styles.mobileDock} aria-label="Cart total and checkout">
        <div>
          <span>
            Subtotal ({cart.totals.item_count.toLocaleString()} {cart.totals.item_count === 1 ? "item" : "items"})
          </span>
          <small>{cart.totals.quantity_total.toLocaleString()} units</small>
        </div>

        <strong>
          {formatMoney(cart.totals.subtotal_amount, cart.totals.currency)}
        </strong>

        <button
          type="button"
          disabled={checkoutBlocked}
          onClick={() => void startCheckout()}
        >
          <span>{checkoutBusy ? "Opening checkout..." : "Continue to checkout"}</span>
          <Icon name="chevronRight" size={17} />
        </button>
      </div>

      <p className={styles.visuallyHidden} role="status" aria-live="polite">
        {statusMessage}
      </p>
    </div>
  );
}
