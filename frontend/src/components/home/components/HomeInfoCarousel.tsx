"use client";

import {
  useRef,
  useState,
} from "react";

import {
  Icon,
  type IconName,
} from "@/components/ui/Icon";

import styles from "../css/HomeInfoCarousel.module.css";

type Item = {
  icon: IconName;
  title: string;
  text: string;
};

const items: Item[] = [
  {
    icon: "review",
    title:
      "Verified catalog",
    text:
      "Structured product data",
  },
  {
    icon:
      "secureCheckout",
    title:
      "Secure payments",
    text:
      "Protected checkout",
  },
  {
    icon: "delivery",
    title:
      "Global shipping",
    text:
      "Reliable delivery",
  },
  {
    icon: "orders",
    title:
      "Flexible MOQ",
    text:
      "Order what you need",
  },
  {
    icon: "returns",
    title:
      "Easy returns",
    text:
      "Post-purchase support",
  },
  {
    icon: "support",
    title:
      "Bulk-order support",
    text:
      "Request more stock",
  },
];

export function HomeInfoCarousel() {
  const railRef =
    useRef<HTMLDivElement | null>(
      null,
    );

  const [
    page,
    setPage,
  ] =
    useState(0);

  const maxPage =
    Math.max(
      0,
      Math.ceil(
        items.length /
          2,
      ) - 1,
    );

  function move(
    direction:
      -1 | 1,
  ) {
    const rail =
      railRef.current;

    if (!rail) {
      return;
    }

    const next =
      Math.min(
        maxPage,
        Math.max(
          0,
          page +
            direction,
        ),
      );

    setPage(next);

    const reduced =
      window.matchMedia(
        "(prefers-reduced-motion: reduce)",
      ).matches;

    rail.scrollTo({
      left:
        next *
        rail.clientWidth *
        0.72,

      behavior:
        reduced
          ? "auto"
          : "smooth",
    });
  }

  return (
    <section
      className={
        styles.carousel
      }
      aria-label="Marketplace information"
    >
      <button
        type="button"
        className={
          styles.navButton
        }
        disabled={
          page === 0
        }
        onClick={() =>
          move(-1)
        }
        aria-label="Previous"
      >
        <Icon
          name="chevronLeft"
          size={16}
        />
      </button>

      <div
        ref={railRef}
        className={
          styles.rail
        }
      >
        {items.map(
          (item) => (
            <article
              key={
                item.title
              }
              className={
                styles.item
              }
            >
              <span
                className={
                  styles.icon
                }
              >
                <Icon
                  name={
                    item.icon
                  }
                  size={
                    20
                  }
                />
              </span>

              <span>
                <strong>
                  {
                    item.title
                  }
                </strong>

                <small>
                  {
                    item.text
                  }
                </small>
              </span>
            </article>
          ),
        )}
      </div>

      <button
        type="button"
        className={
          styles.navButton
        }
        disabled={
          page ===
          maxPage
        }
        onClick={() =>
          move(1)
        }
        aria-label="Next"
      >
        <Icon
          name="chevronRight"
          size={16}
        />
      </button>

      <div
        className={
          styles.dots
        }
      >
        {Array.from({
          length:
            maxPage +
            1,
        }).map(
          (
            _,
            index,
          ) => (
            <span
              key={
                index
              }
              className={
                index ===
                page
                  ? styles.activeDot
                  : undefined
              }
            />
          ),
        )}
      </div>
    </section>
  );
}