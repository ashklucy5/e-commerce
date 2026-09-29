"use client";

import {
  useCallback,
  useEffect,
  useRef,
  useState,
} from "react";

import { Icon } from "@/components/ui/Icon";
import type { HomeProductCard } from "@/lib/api/contracts/home";

import { HomeProductTile } from "./HomeProductTile";
import styles from "../css/HomeProductRail.module.css";

type Props = {
  products: HomeProductCard[];
  label: string;
};

const SCROLL_TOLERANCE = 4;

export function HomeProductRail({
  products,
  label,
}: Props) {
  const railRef =
    useRef<HTMLDivElement | null>(
      null,
    );

  const [canScrollPrevious, setCanScrollPrevious] =
    useState(false);

  const [canScrollNext, setCanScrollNext] =
    useState(false);

  const updateScrollState =
    useCallback(() => {
      const rail =
        railRef.current;

      if (!rail) {
        return;
      }

      const maxScroll =
        Math.max(
          0,
          rail.scrollWidth -
            rail.clientWidth,
        );

      setCanScrollPrevious(
        rail.scrollLeft >
          SCROLL_TOLERANCE,
      );

      setCanScrollNext(
        rail.scrollLeft <
          maxScroll -
            SCROLL_TOLERANCE,
      );
    }, []);

  useEffect(() => {
    const rail =
      railRef.current;

    if (!rail) {
      return;
    }

    const frame =
      window.requestAnimationFrame(
        updateScrollState,
      );

    const observer =
      new ResizeObserver(
        updateScrollState,
      );

    observer.observe(
      rail,
    );

    for (
      const child
      of Array.from(
        rail.children,
      )
    ) {
      observer.observe(
        child,
      );
    }

    window.addEventListener(
      "resize",
      updateScrollState,
    );

    return () => {
      window.cancelAnimationFrame(
        frame,
      );

      observer.disconnect();

      window.removeEventListener(
        "resize",
        updateScrollState,
      );
    };
  }, [
    products.length,
    updateScrollState,
  ]);

  const scrollRail =
    useCallback(
      (
        direction:
          | "previous"
          | "next",
      ) => {
        const rail =
          railRef.current;

        if (!rail) {
          return;
        }

        const reducedMotion =
          window.matchMedia(
            "(prefers-reduced-motion: reduce)",
          ).matches;

        /*
         * Move almost one viewport at a
         * time while keeping part of the
         * previous set visible so the
         * movement remains understandable.
         */
        const amount =
          Math.max(
            1,
            rail.clientWidth *
              0.9,
          );

        rail.scrollBy({
          left:
            direction ===
            "next"
              ? amount
              : -amount,

          behavior:
            reducedMotion
              ? "auto"
              : "smooth",
        });
      },
      [],
    );

  return (
    <div
      className={
        styles.shell
      }
    >
      <button
        type="button"
        className={`${styles.arrow} ${styles.previous}`}
        disabled={
          !canScrollPrevious
        }
        aria-label={`Previous ${label}`}
        onClick={() =>
          scrollRail(
            "previous",
          )
        }
      >
        <Icon
          name="chevronLeft"
          size={15}
        />
      </button>

      <div
        ref={railRef}
        className={
          styles.rail
        }
        aria-label={label}
        onScroll={
          updateScrollState
        }
      >
        {products.map(
          product => (
            <HomeProductTile
              key={
                product.id
              }
              product={
                product
              }
            />
          ),
        )}
      </div>

      <button
        type="button"
        className={`${styles.arrow} ${styles.next}`}
        disabled={
          !canScrollNext
        }
        aria-label={`Next ${label}`}
        onClick={() =>
          scrollRail(
            "next",
          )
        }
      >
        <Icon
          name="chevronRight"
          size={15}
        />
      </button>
    </div>
  );
}