"use client";

import Link from "next/link";
import {
  type MouseEvent,
  type PointerEvent as ReactPointerEvent,
  useRef,
} from "react";

import { CatalogImage } from "@/components/commerce/components/CatalogImage";
import { ProductCardCartButton } from "@/components/commerce/components/ProductCardCartButton";
import { ProductWishlistButton } from "@/components/commerce/components/ProductWishlistButton";
import type { ProductCard } from "@/lib/api/contracts/catalog";
import { formatMoney } from "@/lib/money/format";

import styles from "../css/ProductRecommendations.module.css";

type Props = {
  products: ProductCard[];
  title?: string;
  seeAllHref?: string;
};

export function ProductRecommendations({
  products,
  title = "You might also like",
  seeAllHref,
}: Props) {
  const railRef = useRef<HTMLDivElement | null>(null);
  const dragRef = useRef({
    active: false,
    pointerId: -1,
    startX: 0,
    startScrollLeft: 0,
    moved: false,
  });

  if (products.length === 0) {
    return null;
  }

  function handlePointerDown(event: ReactPointerEvent<HTMLDivElement>) {
    if (event.pointerType !== "mouse" || event.button !== 0) {
      return;
    }

    const rail = railRef.current;
    if (!rail) return;

    dragRef.current = {
      active: true,
      pointerId: event.pointerId,
      startX: event.clientX,
      startScrollLeft: rail.scrollLeft,
      moved: false,
    };
  }

  function handlePointerMove(event: ReactPointerEvent<HTMLDivElement>) {
    const rail = railRef.current;
    const drag = dragRef.current;

    if (!rail || !drag.active || drag.pointerId !== event.pointerId) {
      return;
    }

    const distance = event.clientX - drag.startX;

    if (Math.abs(distance) > 5) {
      drag.moved = true;
      rail.dataset.dragging = "true";
      rail.setPointerCapture(event.pointerId);
      rail.scrollLeft = drag.startScrollLeft - distance;
    }
  }

  function finishDrag(event: ReactPointerEvent<HTMLDivElement>) {
    const rail = railRef.current;
    const drag = dragRef.current;

    if (!rail || !drag.active || drag.pointerId !== event.pointerId) {
      return;
    }

    drag.active = false;
    delete rail.dataset.dragging;

    if (rail.hasPointerCapture(event.pointerId)) {
      rail.releasePointerCapture(event.pointerId);
    }
  }

  function handleClickCapture(event: MouseEvent<HTMLDivElement>) {
    if (dragRef.current.moved) {
      event.preventDefault();
      event.stopPropagation();
      dragRef.current.moved = false;
    }
  }

  const headingId = `recommendation-${title
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-|-$/g, "")}`;

  return (
    <section className={styles.section} aria-labelledby={headingId}>
      <div className={styles.heading}>
        <h2 id={headingId}>{title}</h2>

        {seeAllHref ? (
          <Link href={seeAllHref} className={styles.seeAll}>
            See all
          </Link>
        ) : null}
      </div>

      <div
        ref={railRef}
        className={styles.rail}
        onPointerDown={handlePointerDown}
        onPointerMove={handlePointerMove}
        onPointerUp={finishDrag}
        onPointerCancel={finishDrag}
        onClickCapture={handleClickCapture}
      >
        {products.map((product) => (
          <article key={product.id} className={styles.card}>
            <Link
              href={`/product/${product.slug}`}
              className={styles.productLink}
            >
              <div className={styles.media}>
                {product.primary_image_url ? (
                  <CatalogImage
                    src={product.primary_image_url}
                    alt={product.name}
                    fill
                    sizes="(max-width: 30rem) 34vw, (max-width: 48rem) 28vw, (max-width: 72rem) 18vw, 180px"
                    className={styles.image}
                  />
                ) : (
                  <span className={styles.fallback} aria-hidden="true">
                    {product.name.charAt(0).toUpperCase()}
                  </span>
                )}
              </div>

              <strong className={styles.productName}>{product.name}</strong>

              <span className={styles.price}>
                {formatMoney(product.price_amount, product.currency)}
                <small>/ unit</small>
              </span>

              <span
                className={
                  product.in_stock ? styles.inStock : styles.outOfStock
                }
              >
                <span className={styles.stockDot} aria-hidden="true" />
                {product.in_stock ? "In stock" : "Out of stock"}
              </span>
            </Link>

            <ProductWishlistButton
              productId={product.id}
              productName={product.name}
              placement="recommendation"
            />
            <ProductCardCartButton
              productName={product.name}
              productSlug={product.slug}
              inStock={product.in_stock}
              className={styles.cartButton}
            />
          </article>
        ))}
      </div>
    </section>
  );
}

