// ENE_WISHLIST_CLIENT_V1
// Location: src/components/account/components/WishlistClient.tsx

"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import {
  type ChangeEvent,
  type KeyboardEvent,
  type MouseEvent,
  type SyntheticEvent,
  useCallback,
  useEffect,
  useMemo,
  useState,
} from "react";

import { notifyCartUpdated } from "@/components/commerce/CartBadge";
import type {
  AccountWishlistItem,
  AccountWishlistResponse,
} from "@/lib/api/contracts/account";
import type { Cart } from "@/lib/api/contracts/commerce";
import { formatMoney } from "@/lib/money/format";

import styles from "../css/Wishlist.module.css";

type LoadState = "loading" | "ready" | "error" | "signed-out";
type SortMode = "recent" | "price-low" | "price-high";
type NoticeTone = "success" | "error";

type Notice = {
  tone: NoticeTone;
  text: string;
} | null;

type PurchaseVariant = {
  id: string;
  sku: string;
  color_name: string | null;
  size: string | null;
  minimum_order_quantity: number;
  available_quantity: number;
  in_stock: boolean;
};

type PurchaseOptionsResponse = {
  data: {
    slug: string;
    variants: PurchaseVariant[];
  };
};

type ProductOptionsState = {
  status: "loading" | "ready" | "error";
  variants: PurchaseVariant[];
};

type ProductOptionsMap = Record<string, ProductOptionsState>;
type SelectedVariantMap = Record<string, string>;

function readProductSlug(item: AccountWishlistItem) {
  return (item.product_slug ?? item.slug ?? "").trim();
}

function readProductImage(item: AccountWishlistItem) {
  return (item.primary_image_url ?? item.image_url ?? "").trim();
}

function productHref(item: AccountWishlistItem) {
  const slug = readProductSlug(item);
  return slug ? `/product/${encodeURIComponent(slug)}` : "";
}

function variantIsPurchasable(variant: PurchaseVariant) {
  const moq = Math.max(1, Number(variant.minimum_order_quantity) || 1);
  return variant.in_stock && variant.available_quantity >= moq;
}

function variantLabel(variant: PurchaseVariant) {
  const color = variant.color_name?.trim() ?? "";
  const size = variant.size?.trim() ?? "";
  const parts = [color, size ? `Size ${size}` : ""].filter(Boolean);
  return parts.length > 0 ? parts.join(" - ") : variant.sku;
}

function savedLabel(value?: string) {
  if (!value || Number.isNaN(Date.parse(value))) return "Saved";

  const then = new Date(value);
  const now = new Date();
  const sameDay =
    then.getFullYear() === now.getFullYear() &&
    then.getMonth() === now.getMonth() &&
    then.getDate() === now.getDate();

  if (sameDay) return "Saved today";

  return `Saved ${new Intl.DateTimeFormat("en", {
    month: "short",
    day: "numeric",
  }).format(then)}`;
}

function apiMessage(payload: unknown, fallback: string) {
  if (!payload || typeof payload !== "object") return fallback;

  const root = payload as Record<string, unknown>;
  const error = root.error;

  if (typeof error === "string" && error.trim()) return error;

  if (error && typeof error === "object") {
    const message = (error as Record<string, unknown>).message;
    if (typeof message === "string" && message.trim()) return message;
  }

  if (typeof root.message === "string" && root.message.trim()) {
    return root.message;
  }

  return fallback;
}

async function readError(response: Response, fallback: string) {
  try {
    return apiMessage(await response.json(), fallback);
  } catch {
    return fallback;
  }
}

async function runWithConcurrency<T>(
  values: T[],
  limit: number,
  task: (value: T) => Promise<void>,
) {
  if (values.length === 0) return;

  let cursor = 0;
  const workerCount = Math.min(Math.max(1, limit), values.length);

  async function worker() {
    while (cursor < values.length) {
      const index = cursor;
      cursor += 1;
      const value = values[index];
      if (value === undefined) return;
      await task(value);
    }
  }

  await Promise.all(Array.from({ length: workerCount }, () => worker()));
}

function SearchIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" aria-hidden="true">
      <circle cx="11" cy="11" r="7" />
      <path d="m20 20-3.8-3.8" />
    </svg>
  );
}

function HeartIcon() {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true">
      <path d="M12 21s-7.2-4.35-9.45-8.24C.66 9.49 1.61 5.5 5.25 4.3 7.54 3.55 10 4.44 12 6.56c2-2.12 4.46-3.01 6.75-2.26 3.64 1.2 4.59 5.19 2.7 8.46C19.2 16.65 12 21 12 21Z" />
    </svg>
  );
}

function CartIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.9" aria-hidden="true">
      <path d="M3 4h2l2.2 10.2a2 2 0 0 0 2 1.6h7.9a2 2 0 0 0 1.9-1.4L21 8H7" />
      <circle cx="10" cy="20" r="1.4" />
      <circle cx="18" cy="20" r="1.4" />
      <path d="M14 5v6m-3-3h6" />
    </svg>
  );
}

function TrashIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" aria-hidden="true">
      <path d="M4 7h16M9 7V4h6v3m-8 0 1 13h8l1-13M10 11v5m4-5v5" />
    </svg>
  );
}

function ProductImage({ item }: { item: AccountWishlistItem }) {
  const image = readProductImage(item);
  const initial = item.name.trim().charAt(0).toUpperCase() || "P";

  return (
    <div className={styles.media}>
      <span className={styles.imageFallback} aria-hidden="true">
        {initial}
      </span>

      {image ? (
        // Dynamic catalog media can come from backend-configured remote hosts.
        // eslint-disable-next-line @next/next/no-img-element
        <img
          className={styles.productImage}
          src={image}
          alt={item.name}
          loading="lazy"
          decoding="async"
          onError={(event: SyntheticEvent<HTMLImageElement>) => {
            event.currentTarget.style.display = "none";
          }}
        />
      ) : null}

      <span className={styles.savedHeart} aria-label="Saved to wishlist">
        <HeartIcon />
      </span>
    </div>
  );
}

export function WishlistClient() {
  const router = useRouter();

  const [state, setState] = useState<LoadState>("loading");
  const [items, setItems] = useState<AccountWishlistItem[]>([]);
  const [query, setQuery] = useState("");
  const [sort, setSort] = useState<SortMode>("recent");
  const [options, setOptions] = useState<ProductOptionsMap>({});
  const [selectedVariants, setSelectedVariants] = useState<SelectedVariantMap>({});
  const [removingId, setRemovingId] = useState("");
  const [cartBusyId, setCartBusyId] = useState("");
  const [error, setError] = useState("");
  const [notice, setNotice] = useState<Notice>(null);

  useEffect(() => {
    if (!notice) return;
    const timer = window.setTimeout(() => setNotice(null), 2800);
    return () => window.clearTimeout(timer);
  }, [notice]);

  const hydratePurchaseOptions = useCallback(async (nextItems: AccountWishlistItem[]) => {
    const nextLoadingState: ProductOptionsMap = {};

    for (const item of nextItems) {
      nextLoadingState[item.product_id] = {
        status: "loading",
        variants: [],
      };
    }

    setOptions(nextLoadingState);

    await runWithConcurrency(nextItems, 5, async (item) => {
      const slug = readProductSlug(item);

      if (!slug) {
        setOptions((current) => ({
          ...current,
          [item.product_id]: { status: "error", variants: [] },
        }));
        return;
      }

      try {
        const response = await fetch(
          `/api/storefront/products/${encodeURIComponent(slug)}/purchase-options`,
          { cache: "no-store" },
        );

        if (!response.ok) {
          throw new Error("Unable to load purchase options.");
        }

        const payload = (await response.json()) as PurchaseOptionsResponse;
        const variants = Array.isArray(payload.data?.variants)
          ? payload.data.variants
          : [];
        const firstPurchasable = variants.find(variantIsPurchasable);

        setOptions((current) => ({
          ...current,
          [item.product_id]: { status: "ready", variants },
        }));

        if (firstPurchasable) {
          setSelectedVariants((current) => ({
            ...current,
            [item.product_id]: current[item.product_id] || firstPurchasable.id,
          }));
        }
      } catch {
        setOptions((current) => ({
          ...current,
          [item.product_id]: { status: "error", variants: [] },
        }));
      }
    });
  }, []);

  const load = useCallback(async () => {
    setState("loading");
    setError("");

    try {
      const response = await fetch(
        "/api/storefront/account/wishlist?page=1&limit=100",
        { cache: "no-store" },
      );

      if (response.status === 401) {
        setState("signed-out");
        return;
      }

      if (!response.ok) {
        throw new Error(await readError(response, "Unable to load your wishlist."));
      }

      const payload = (await response.json()) as AccountWishlistResponse;
      const nextItems = Array.isArray(payload.data) ? payload.data : [];

      setItems(nextItems);
      setState("ready");
      void hydratePurchaseOptions(nextItems);
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : "Unable to load your wishlist.");
      setState("error");
    }
  }, [hydratePurchaseOptions]);

  useEffect(() => {
    const timer = window.setTimeout(() => {
      void load();
    }, 0);

    return () => window.clearTimeout(timer);
  }, [load]);

  useEffect(() => {
    if (state !== "signed-out") return;
    router.replace("/account/sign-in?next=%2Faccount%2Fwishlist");
  }, [router, state]);

  const visibleItems = useMemo(() => {
    const normalizedQuery = query.trim().toLocaleLowerCase();
    const filtered = normalizedQuery
      ? items.filter((item) => item.name.toLocaleLowerCase().includes(normalizedQuery))
      : [...items];

    return filtered.sort((left, right) => {
      if (sort === "price-low") return left.price_amount - right.price_amount;
      if (sort === "price-high") return right.price_amount - left.price_amount;

      const leftTime = left.created_at ? Date.parse(left.created_at) : 0;
      const rightTime = right.created_at ? Date.parse(right.created_at) : 0;
      return rightTime - leftTime;
    });
  }, [items, query, sort]);

  async function removeItem(item: AccountWishlistItem) {
    if (removingId || cartBusyId) return;

    setRemovingId(item.product_id);
    setNotice(null);

    try {
      const response = await fetch(
        `/api/storefront/account/wishlist/${encodeURIComponent(item.product_id)}`,
        { method: "DELETE" },
      );

      if (response.status === 401) {
        setState("signed-out");
        return;
      }

      if (!response.ok) {
        throw new Error(await readError(response, "Unable to remove this saved product."));
      }

      setItems((current) => current.filter((entry) => entry.product_id !== item.product_id));
      setOptions((current) => {
        const next = { ...current };
        delete next[item.product_id];
        return next;
      });
      setSelectedVariants((current) => {
        const next = { ...current };
        delete next[item.product_id];
        return next;
      });
      setNotice({ tone: "success", text: `${item.name} removed from wishlist.` });
    } catch (caught) {
      setNotice({
        tone: "error",
        text: caught instanceof Error ? caught.message : "Unable to remove this saved product.",
      });
    } finally {
      setRemovingId("");
    }
  }

  async function addToCart(item: AccountWishlistItem, variant: PurchaseVariant | undefined) {
    if (!variant || !variantIsPurchasable(variant) || cartBusyId || removingId) return;

    const quantity = Math.max(1, variant.minimum_order_quantity);
    setCartBusyId(item.product_id);
    setNotice(null);

    try {
      const response = await fetch("/api/storefront/cart/items", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          variant_id: variant.id,
          quantity,
        }),
      });

      if (!response.ok) {
        throw new Error(await readError(response, "Unable to add this product to cart."));
      }

      const payload = (await response.json()) as { data: Cart };
      notifyCartUpdated(payload.data);
      setNotice({
        tone: "success",
        text: `${quantity} x ${variantLabel(variant)} added to cart. The product stays saved.`,
      });
    } catch (caught) {
      setNotice({
        tone: "error",
        text: caught instanceof Error ? caught.message : "Unable to add this product to cart.",
      });
    } finally {
      setCartBusyId("");
    }
  }

  function openProduct(
    item: AccountWishlistItem,
    event: MouseEvent<HTMLElement> | KeyboardEvent<HTMLElement>,
  ) {
    const target = event.target as HTMLElement;
    if (target.closest("button, select, input, a, label")) return;

    const href = productHref(item);
    if (href) router.push(href);
  }

  function onCardKeyDown(item: AccountWishlistItem, event: KeyboardEvent<HTMLElement>) {
    if (event.key !== "Enter" && event.key !== " ") return;
    const target = event.target as HTMLElement;
    if (target.closest("button, select, input, a, label")) return;

    event.preventDefault();
    const href = productHref(item);
    if (href) router.push(href);
  }

  if (state === "signed-out") {
    return (
      <main className={styles.page}>
        <section className={styles.centerState}>
          <span className={styles.stateHeart} aria-hidden="true">
            <HeartIcon />
          </span>
          <p className={styles.eyebrow}>Wishlist</p>
          <h1>Sign in to open your saved products.</h1>
          <p>Your wishlist is attached to your customer account.</p>
          <Link className={styles.primaryLink} href="/account/sign-in?next=%2Faccount%2Fwishlist">
            Sign in
          </Link>
        </section>
      </main>
    );
  }

  return (
    <main className={styles.page}>
      <div className={styles.shell}>
        <header className={styles.pageHeader}>
          <div>
            <p className={styles.eyebrow}>Saved products</p>
            <h1>Wishlist</h1>
            <p className={styles.description}>
              Choose a variant, add its minimum quantity to your cart, or keep it saved for later.
              Open the product by selecting anywhere else on its card.
            </p>
          </div>

          <div className={styles.savedCount} aria-label={`${items.length} saved products`}>
            <span aria-hidden="true"><HeartIcon /></span>
            <strong>{items.length}</strong>
            <span>saved</span>
          </div>
        </header>

        {state === "loading" ? (
          <section className={styles.loadingGrid} aria-label="Loading wishlist">
            {Array.from({ length: 6 }, (_, index) => (
              <div className={styles.skeletonCard} key={index} aria-hidden="true">
                <span className={styles.skeletonMedia} />
                <span className={styles.skeletonLineWide} />
                <span className={styles.skeletonLine} />
                <span className={styles.skeletonControl} />
              </div>
            ))}
          </section>
        ) : null}

        {state === "error" ? (
          <section className={styles.centerState}>
            <p className={styles.eyebrow}>Wishlist unavailable</p>
            <h2>We could not load your saved products.</h2>
            <p>{error || "Please try again."}</p>
            <button className={styles.primaryButton} type="button" onClick={() => void load()}>
              Try again
            </button>
          </section>
        ) : null}

        {state === "ready" && items.length === 0 ? (
          <section className={styles.centerState}>
            <span className={styles.stateHeart} aria-hidden="true">
              <HeartIcon />
            </span>
            <p className={styles.eyebrow}>Nothing saved yet</p>
            <h2>Your wishlist is ready for products you want to revisit.</h2>
            <p>Use the wishlist action on product cards and product pages to save products here.</p>
            <Link className={styles.primaryLink} href="/">
              Browse products
            </Link>
          </section>
        ) : null}

        {state === "ready" && items.length > 0 ? (
          <>
            <section className={styles.toolbar} aria-label="Wishlist controls">
              <label className={styles.searchControl}>
                <span className="sr-only">Search wishlist</span>
                <span className={styles.searchIcon}><SearchIcon /></span>
                <input
                  type="search"
                  value={query}
                  placeholder="Search your wishlist"
                  onChange={(event: ChangeEvent<HTMLInputElement>) => setQuery(event.target.value)}
                />
              </label>

              <label className={styles.sortControl}>
                <span className="sr-only">Sort wishlist</span>
                <select
                  value={sort}
                  onChange={(event: ChangeEvent<HTMLSelectElement>) =>
                    setSort(event.target.value as SortMode)
                  }
                >
                  <option value="recent">Recently saved</option>
                  <option value="price-low">Price: low to high</option>
                  <option value="price-high">Price: high to low</option>
                </select>
              </label>
            </section>

            {visibleItems.length === 0 ? (
              <section className={styles.searchEmpty}>
                <p className={styles.eyebrow}>No matches</p>
                <h2>No saved products match &ldquo;{query.trim()}&rdquo;.</h2>
                <button className={styles.secondaryButton} type="button" onClick={() => setQuery("")}>
                  Clear search
                </button>
              </section>
            ) : (
              <section className={styles.grid} aria-label="Saved products">
                {visibleItems.map((item) => {
                  const optionState = options[item.product_id];
                  const variants = optionState?.variants ?? [];
                  const selectedId = selectedVariants[item.product_id] ?? "";
                  const selectedVariant =
                    variants.find((variant) => variant.id === selectedId) ??
                    variants.find(variantIsPurchasable);
                  const purchasableVariants = variants.filter(variantIsPurchasable);
                  const moq = selectedVariant
                    ? Math.max(1, selectedVariant.minimum_order_quantity)
                    : null;
                  const canAdd = Boolean(
                    selectedVariant &&
                    variantIsPurchasable(selectedVariant) &&
                    item.is_available &&
                    item.in_stock,
                  );
                  const href = productHref(item);
                  const removing = removingId === item.product_id;
                  const adding = cartBusyId === item.product_id;

                  return (
                    <article
                      className={styles.flowShell}
                      key={item.product_id}
                      role={href ? "link" : undefined}
                      tabIndex={href ? 0 : undefined}
                      aria-label={href ? `Open ${item.name}` : undefined}
                      onClick={(event: MouseEvent<HTMLElement>) => openProduct(item, event)}
                      onKeyDown={(event: KeyboardEvent<HTMLElement>) => onCardKeyDown(item, event)}
                    >
                      <div className={styles.card}>
                        <ProductImage item={item} />

                        <div className={styles.cardBody}>
                          <div className={styles.cardTopline}>
                            <span>Saved product</span>
                            <span>{savedLabel(item.created_at)}</span>
                          </div>

                          <h2 className={styles.productName}>{item.name}</h2>

                          <div className={styles.priceRow}>
                            <strong>{formatMoney(item.price_amount, item.currency)}</strong>
                            <span
                              className={item.in_stock && item.is_available ? styles.inStock : styles.outOfStock}
                            >
                              <i aria-hidden="true" />
                              {item.in_stock && item.is_available ? "In stock" : "Out of stock"}
                            </span>
                          </div>

                          <div className={styles.variantBlock}>
                            <div className={styles.variantHeading}>
                              <label htmlFor={`wishlist-variant-${item.product_id}`}>Variant</label>
                              <span>{moq ? `MOQ ${moq}` : "MOQ -"}</span>
                            </div>

                            <select
                              id={`wishlist-variant-${item.product_id}`}
                              className={styles.variantSelect}
                              value={selectedVariant?.id ?? ""}
                              disabled={optionState?.status !== "ready" || purchasableVariants.length === 0}
                              onChange={(event: ChangeEvent<HTMLSelectElement>) => {
                                setSelectedVariants((current) => ({
                                  ...current,
                                  [item.product_id]: event.target.value,
                                }));
                              }}
                              onClick={(event: MouseEvent<HTMLSelectElement>) => event.stopPropagation()}
                            >
                              {optionState?.status === "loading" || !optionState ? (
                                <option value="">Loading variants...</option>
                              ) : null}

                              {optionState?.status === "error" ? (
                                <option value="">Purchase options unavailable</option>
                              ) : null}

                              {optionState?.status === "ready" && variants.length === 0 ? (
                                <option value="">No variants available</option>
                              ) : null}

                              {optionState?.status === "ready"
                                ? variants.map((variant) => {
                                    const available = variantIsPurchasable(variant);
                                    return (
                                      <option
                                        key={variant.id}
                                        value={variant.id}
                                        disabled={!available}
                                      >
                                        {variantLabel(variant)}
                                        {available ? "" : " - unavailable"}
                                      </option>
                                    );
                                  })
                                : null}
                            </select>
                          </div>

                          <div className={styles.cardActions}>
                            <button
                              type="button"
                              className={styles.cartButton}
                              disabled={!canAdd || adding || Boolean(removingId)}
                              onClick={(event: MouseEvent<HTMLButtonElement>) => {
                                event.stopPropagation();
                                void addToCart(item, selectedVariant);
                              }}
                              aria-label={
                                canAdd
                                  ? `Add ${item.name} to cart at minimum order quantity ${moq ?? 1}`
                                  : `${item.name} is not currently purchasable`
                              }
                            >
                              <CartIcon />
                              <span>{adding ? "Adding..." : canAdd ? "Add to cart" : "Unavailable"}</span>
                              {canAdd && moq ? <small>MOQ {moq}</small> : null}
                            </button>

                            <button
                              type="button"
                              className={styles.deleteButton}
                              disabled={removing || Boolean(cartBusyId)}
                              onClick={(event: MouseEvent<HTMLButtonElement>) => {
                                event.stopPropagation();
                                void removeItem(item);
                              }}
                              aria-label={removing ? `Removing ${item.name}` : `Remove ${item.name} from wishlist`}
                              title="Remove from wishlist"
                            >
                              <TrashIcon />
                            </button>
                          </div>

                          <p className={styles.cardHint}>
                            {href ? "Open the card for full product details." : "Product details are temporarily unavailable."}
                          </p>
                        </div>
                      </div>
                    </article>
                  );
                })}
              </section>
            )}
          </>
        ) : null}
      </div>

      {notice ? (
        <div
          className={`${styles.notice} ${notice.tone === "error" ? styles.noticeError : styles.noticeSuccess}`}
          role="status"
          aria-live="polite"
        >
          {notice.text}
        </div>
      ) : null}
    </main>
  );
}
