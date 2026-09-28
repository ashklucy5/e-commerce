"use client";

import Image from "next/image";
import Link from "next/link";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";

import type { ProductListMeta } from "@/lib/api/contracts/catalog";
import type {
  HomeProductCard,
  HomeProductFeedResponse,
  StorefrontPromotion,
} from "@/lib/api/contracts/home";

import { FlashSaleCountdown } from "./FlashSaleCountdown";
import { HomeProductRail } from "./HomeProductRail";
import { HomeProductTile } from "./HomeProductTile";
import styles from "../css/HomeFeed.module.css";

const BATCH_SIZE = 50;
const FEATURE_SIZE = 8;

type Props = {
  initialProducts: HomeProductCard[];
  initialMeta: ProductListMeta;
  promotions: StorefrontPromotion[];
};

function discountPercent(product: HomeProductCard) {
  const compareAt = product.compare_at_price_amount;

  if (!compareAt || compareAt <= product.price_amount) {
    return 0;
  }

  return Math.round(((compareAt - product.price_amount) / compareAt) * 100);
}

function uniqueAppend(current: HomeProductCard[], incoming: HomeProductCard[]) {
  const seen = new Set(current.map((product) => product.id));

  return [
    ...current,
    ...incoming.filter((product) => {
      if (seen.has(product.id)) {
        return false;
      }

      seen.add(product.id);
      return true;
    }),
  ];
}

function buildRows(products: HomeProductCard[]) {
  const used = new Set<string>();

  const flash = products
    .filter((product) => product.merchandising_group === "discount")
    .sort((a, b) => discountPercent(b) - discountPercent(a))
    .slice(0, FEATURE_SIZE);

  flash.forEach((product) => used.add(product.id));

  const arrivals = products
    .filter((product) => product.merchandising_group === "new")
    .sort(
      (a, b) =>
        Date.parse(b.published_at ?? b.created_at) -
        Date.parse(a.published_at ?? a.created_at),
    )
    .slice(0, FEATURE_SIZE);

  arrivals.forEach((product) => used.add(product.id));

  const popular = products
    .filter((product) => product.merchandising_group === "popular")
    .sort((a, b) => {
      const sold = b.sold_quantity - a.sold_quantity;
      return sold !== 0 ? sold : Number(b.is_featured) - Number(a.is_featured);
    })
    .slice(0, FEATURE_SIZE);

  popular.forEach((product) => used.add(product.id));

  return {
    flash,
    arrivals,
    popular,
    featuredIds: used,
  };
}


function isFlashSalePromotion(promotion: StorefrontPromotion) {
  const name = promotion.name.toLowerCase();
  return name.includes("flash") || /\bsale\b/.test(name);
}

function flashPromotionEnd(promotions: StorefrontPromotion[]) {
  const now = Date.now();

  return promotions
    .filter(isFlashSalePromotion)
    .map((promotion) => promotion.ends_at)
    .filter((value): value is string => Boolean(value))
    .map((value) => Date.parse(value))
    .filter((value) => Number.isFinite(value) && value > now)
    .sort((a, b) => a - b)[0];
}

export function HomeCatalogFeed({
  initialProducts,
  initialMeta,
  promotions,
}: Props) {
  const [products, setProducts] = useState(initialProducts);
  const [meta, setMeta] = useState(initialMeta);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [loadedBeyondInitial, setLoadedBeyondInitial] = useState(false);
  const sentinel = useRef<HTMLDivElement | null>(null);
  const loadingRef = useRef(false);

  const rows = useMemo(() => buildRows(initialProducts), [initialProducts]);
  const hasFlashSale = useMemo(
    () => promotions.some(isFlashSalePromotion),
    [promotions],
  );
  const flashEndsAt = useMemo(() => flashPromotionEnd(promotions), [promotions]);
  const discountLabel = hasFlashSale ? "Flash Sale" : "Deals";

  const remaining = useMemo(
    () => products.filter((product) => !rows.featuredIds.has(product.id)),
    [products, rows.featuredIds],
  );

  const loadNext = useCallback(async () => {
    if (loadingRef.current || !meta.has_next) {
      return;
    }

    loadingRef.current = true;
    setLoading(true);
    setError("");

    try {
      const params = new URLSearchParams({
        page: String(meta.page + 1),
        limit: String(BATCH_SIZE),
      });

      const response = await fetch(`/api/storefront/home-feed?${params.toString()}`, {
        headers: {
          Accept: "application/json",
        },
      });

      if (!response.ok) {
        throw new Error("Unable to load more products.");
      }

      const payload = (await response.json()) as HomeProductFeedResponse;
      setProducts((current) => uniqueAppend(current, payload.data ?? []));
      setMeta(payload.meta);
      setLoadedBeyondInitial(true);
    } catch (reason) {
      setError(
        reason instanceof Error ? reason.message : "Unable to load more products.",
      );
    } finally {
      loadingRef.current = false;
      setLoading(false);
    }
  }, [meta.has_next, meta.page]);

  useEffect(() => {
    const element = sentinel.current;

    if (!element || !meta.has_next) {
      return;
    }

    const observer = new IntersectionObserver(
      (entries) => {
        if (entries.some((entry) => entry.isIntersecting)) {
          void loadNext();
        }
      },
      {
        rootMargin: "1800px 0px",
        threshold: 0.01,
      },
    );

    observer.observe(element);
    return () => observer.disconnect();
  }, [loadNext, meta.has_next]);

  if (products.length === 0) {
    return (
      <section id="home-catalog" className={styles.empty}>
        <strong>Products are temporarily unavailable.</strong>
        <span>Please try again shortly.</span>
      </section>
    );
  }

  return (
    <div id="home-catalog" className={styles.feed}>
      {rows.flash.length > 0 ? (
        <section
          id="home-feed-discount"
          className={`${styles.section} ${styles.flashSection}`}
        >
          <header className={styles.sectionHeader}>
            <div className={styles.titleRow}>
              <Image
                src="/icons/commerce/flash.svg"
                alt=""
                width={20}
                height={20}
                unoptimized
                aria-hidden="true"
                className={styles.flashIcon}
              />
              <h2>{discountLabel}</h2>
              <FlashSaleCountdown target={flashEndsAt} />
            </div>

          </header>

          <HomeProductRail products={rows.flash} label={discountLabel} />
        </section>
      ) : null}

      {rows.arrivals.length > 0 ? (
        <section id="home-feed-new" className={styles.section}>
          <header className={styles.sectionHeader}>
            <div className={styles.titleRow}>
              <h2>New Arrivals</h2>
            </div>

          </header>

          <HomeProductRail products={rows.arrivals} label="New Arrivals" />
        </section>
      ) : null}

      {rows.popular.length > 0 ? (
        <section id="home-feed-popular" className={styles.section}>
          <header className={styles.sectionHeader}>
            <div className={styles.titleRow}>
              <h2>Popular Products</h2>
            </div>

          </header>

          <HomeProductRail products={rows.popular} label="Popular Products" />
        </section>
      ) : null}

      <section
        id="all-products"
        className={`${styles.section} ${styles.catalogSection}`}
      >
        <header className={styles.sectionHeader}>
          <div className={styles.titleRow}>
            <h2>All Products</h2>
          </div>

        </header>

        <div className={styles.catalogGrid}>
          {remaining.map((product) => (
            <HomeProductTile
              key={product.id}
              product={product}
              variant="catalog"
            />
          ))}
        </div>
      </section>

      <div ref={sentinel} className={styles.sentinel} aria-hidden="true" />

      {loading ? (
        <div className={styles.status} role="status">
          <span className={styles.spinner} />
          Loading more products…
        </div>
      ) : null}

      {error ? (
        <div className={styles.status} role="alert">
          <span>{error}</span>
          <button type="button" onClick={() => void loadNext()}>
            Retry
          </button>
        </div>
      ) : null}

      {!meta.has_next ? (
        <div className={styles.complete}>
          <strong>All active products loaded.</strong>
          <span>Use the category strip or search to narrow the catalog.</span>
        </div>
      ) : null}

      {!loadedBeyondInitial && initialMeta.total_pages > 1 ? (
        <nav className={styles.pagination} aria-label="Catalog pages">
          {initialMeta.has_previous ? (
            <Link
              href={initialMeta.page <= 2 ? "/" : `/?page=${initialMeta.page - 1}`}
              rel="prev"
            >
              Previous page
            </Link>
          ) : null}

          <span aria-current="page">
            Page {initialMeta.page} of {initialMeta.total_pages}
          </span>

          {initialMeta.has_next ? (
            <Link href={`/?page=${initialMeta.page + 1}`} rel="next">
              Next page
            </Link>
          ) : null}
        </nav>
      ) : null}
    </div>
  );
}
