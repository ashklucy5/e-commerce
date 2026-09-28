"use client";

import { useCallback, useEffect, useRef, useState } from "react";

import { Icon } from "@/components/ui/Icon";
import type { HomeProductCard } from "@/lib/api/contracts/home";

import { HomeProductTile } from "./HomeProductTile";
import styles from "../css/HomeProductRail.module.css";

type Props = {
  products: HomeProductCard[];
  label: string;
};

export function HomeProductRail({ products, label }: Props) {
  const railRef = useRef<HTMLDivElement | null>(null);
  const [page, setPage] = useState(0);
  const [pageCount, setPageCount] = useState(1);

  const updateMetrics = useCallback(() => {
    const rail = railRef.current;

    if (!rail) {
      return;
    }

    const maxScroll = Math.max(0, rail.scrollWidth - rail.clientWidth);

    if (maxScroll <= 1) {
      setPage(0);
      setPageCount(1);
      return;
    }

    const pages = Math.max(2, Math.ceil(rail.scrollWidth / rail.clientWidth));
    const nextPage = Math.round((rail.scrollLeft / maxScroll) * (pages - 1));

    setPageCount(pages);
    setPage(Math.max(0, Math.min(pages - 1, nextPage)));
  }, []);

  useEffect(() => {
    const rail = railRef.current;

    if (!rail) {
      return;
    }

    updateMetrics();

    const observer = new ResizeObserver(updateMetrics);
    observer.observe(rail);

    return () => observer.disconnect();
  }, [products.length, updateMetrics]);

  function goTo(requested: number) {
    const rail = railRef.current;

    if (!rail) {
      return;
    }

    const safePage = Math.max(0, Math.min(pageCount - 1, requested));
    const maxScroll = Math.max(0, rail.scrollWidth - rail.clientWidth);
    const left = pageCount <= 1 ? 0 : (maxScroll / (pageCount - 1)) * safePage;
    const reducedMotion = window.matchMedia(
      "(prefers-reduced-motion: reduce)",
    ).matches;

    rail.scrollTo({
      left,
      behavior: reducedMotion ? "auto" : "smooth",
    });
    setPage(safePage);
  }

  return (
    <div className={styles.shell}>
      <button
        type="button"
        className={`${styles.arrow} ${styles.previous}`}
        disabled={page <= 0}
        aria-label={`Previous ${label}`}
        onClick={() => goTo(page - 1)}
      >
        <Icon name="chevronLeft" size={15} />
      </button>

      <div
        ref={railRef}
        className={styles.rail}
        aria-label={label}
        onScroll={updateMetrics}
      >
        {products.map((product) => (
          <HomeProductTile
            key={product.id}
            product={product}
          />
        ))}
      </div>

      <button
        type="button"
        className={`${styles.arrow} ${styles.next}`}
        disabled={page >= pageCount - 1}
        aria-label={`Next ${label}`}
        onClick={() => goTo(page + 1)}
      >
        <Icon name="chevronRight" size={15} />
      </button>

      {pageCount > 1 ? (
        <div className={styles.dots} aria-label={`${label} pages`}>
          {Array.from({ length: pageCount }).map((_, index) => (
            <button
              key={index}
              type="button"
              className={index === page ? styles.activeDot : styles.dot}
              aria-label={`Show ${label} page ${index + 1}`}
              aria-current={index === page ? "true" : undefined}
              onClick={() => goTo(index)}
            />
          ))}
        </div>
      ) : null}
    </div>
  );
}
