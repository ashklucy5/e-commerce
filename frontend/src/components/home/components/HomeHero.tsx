"use client";

import Link from "next/link";
import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";

import { CatalogImage } from "@/components/commerce/components/CatalogImage";
import type { HomeProductCard } from "@/lib/api/contracts/home";
import { formatMoney } from "@/lib/money/format";

import styles from "../css/HomeHero.module.css";

const MAX_SLIDES = 4;
const AUTOPLAY_DELAY = 6000;
const MANUAL_RESUME_DELAY = 8000;

type Props = {
  products: HomeProductCard[];
};

const campaigns = [
  {
    eyebrow: "B2B wholesale platform",
    first: "Precision.",
    second: "Built for Business.",
    description:
      "Discover business-ready products with transparent pricing, dependable availability and support when you need to source more.",
  },
  {
    eyebrow: "Popular with businesses",
    first: "Products that Move.",
    second: "Supply that Scales.",
    description:
      "Discover popular products businesses are already buying, with current pricing and live availability.",
  },
  {
    eyebrow: "Business purchasing",
    first: "Buy Smarter.",
    second: "Source with Confidence.",
    description:
      "Find products quickly, compare real availability and request sourcing help when the catalog does not have what you need.",
  },
  {
    eyebrow: "Reliable discovery",
    first: "Built for Demand.",
    second: "Ready for Growth.",
    description:
      "Move from product discovery to purchasing with a storefront designed around real business buying.",
  },
] as const;

export function HomeHero({
  products,
}: Props) {
  const railRef =
    useRef<HTMLDivElement | null>(
      null,
    );

  const interactionUntilRef =
    useRef(0);

  const dragRef =
    useRef({
      active: false,
      pointerId: -1,
      startX: 0,
      startScrollLeft: 0,
    });

  const [
    activeIndex,
    setActiveIndex,
  ] = useState(0);

  const slides = useMemo(
    () =>
      products.length > 0
        ? products.slice(
            0,
            MAX_SLIDES,
          )
        : [undefined],
    [products],
  );

  const prefersReducedMotion =
    useCallback(() => {
      if (
        typeof window ===
        "undefined"
      ) {
        return false;
      }

      return window.matchMedia(
        "(prefers-reduced-motion: reduce)",
      ).matches;
    }, []);

  const markManualInteraction =
    useCallback(() => {
      interactionUntilRef.current =
        Date.now() +
        MANUAL_RESUME_DELAY;
    }, []);

  const goTo =
    useCallback(
      (
        requestedIndex: number,
        behavior:
          ScrollBehavior =
          "smooth",
      ) => {
        const rail =
          railRef.current;

        if (
          !rail ||
          slides.length === 0
        ) {
          return;
        }

        const index =
          (
            requestedIndex +
            slides.length
          ) %
          slides.length;

        const target =
          rail.children[
            index
          ] as
            | HTMLElement
            | undefined;

        if (!target) {
          return;
        }

        rail.scrollTo({
          left:
            target.offsetLeft,

          behavior:
            prefersReducedMotion()
              ? "auto"
              : behavior,
        });
      },
      [
        prefersReducedMotion,
        slides.length,
      ],
    );

  /*
   * Automatic banner rotation.
   *
   * No Play/Pause control.
   *
   * It only pauses temporarily when the
   * shopper manually interacts with the
   * carousel, then resumes automatically.
   *
   * Reduced-motion is the accessibility
   * exception.
   */
  useEffect(() => {
    if (
      slides.length <= 1
    ) {
      return;
    }

    const timer =
      window.setInterval(
        () => {
          if (
            document.hidden ||
            dragRef.current
              .active ||
            Date.now() <
              interactionUntilRef.current ||
            prefersReducedMotion()
          ) {
            return;
          }

          const rail =
            railRef.current;

          if (
            !rail ||
            rail.clientWidth <=
              0
          ) {
            return;
          }

          const currentIndex =
            Math.round(
              rail.scrollLeft /
                rail.clientWidth,
            );

          goTo(
            currentIndex +
              1,
          );
        },
        AUTOPLAY_DELAY,
      );

    return () => {
      window.clearInterval(
        timer,
      );
    };
  }, [
    goTo,
    prefersReducedMotion,
    slides.length,
  ]);

  /*
   * Keep the current slide aligned when the
   * viewport, zoom level or container width
   * changes.
   */
  useEffect(() => {
    const rail =
      railRef.current;

    if (!rail) {
      return;
    }

    const observer =
      new ResizeObserver(
        () => {
          const target =
            rail.children[
              activeIndex
            ] as
              | HTMLElement
              | undefined;

          if (!target) {
            return;
          }

          rail.scrollTo({
            left:
              target.offsetLeft,

            behavior: "auto",
          });
        },
      );

    observer.observe(
      rail,
    );

    return () => {
      observer.disconnect();
    };
  }, [activeIndex]);

  function handleScroll() {
    const rail =
      railRef.current;

    if (
      !rail ||
      rail.clientWidth <= 0
    ) {
      return;
    }

    const index =
      Math.round(
        rail.scrollLeft /
          rail.clientWidth,
      );

    setActiveIndex(
      Math.max(
        0,
        Math.min(
          slides.length - 1,
          index,
        ),
      ),
    );
  }

  function handlePointerDown(
    event:
      React.PointerEvent<HTMLDivElement>,
  ) {
    markManualInteraction();

    /*
     * Phones/tablets use the browser's native
     * touch scrolling.
     */
    if (
      event.pointerType !==
        "mouse" ||
      event.button !== 0
    ) {
      return;
    }

    const target =
      event.target as HTMLElement;

    /*
     * Do not turn normal button/link clicks
     * into drag gestures.
     */
    if (
      target.closest(
        "a, button",
      )
    ) {
      return;
    }

    const rail =
      railRef.current;

    if (!rail) {
      return;
    }

    dragRef.current = {
      active: true,

      pointerId:
        event.pointerId,

      startX:
        event.clientX,

      startScrollLeft:
        rail.scrollLeft,
    };

    rail.setPointerCapture(
      event.pointerId,
    );

    rail.dataset.dragging =
      "true";
  }

  function handlePointerMove(
    event:
      React.PointerEvent<HTMLDivElement>,
  ) {
    const rail =
      railRef.current;

    const drag =
      dragRef.current;

    if (
      !rail ||
      !drag.active ||
      drag.pointerId !==
        event.pointerId
    ) {
      return;
    }

    const distance =
      event.clientX -
      drag.startX;

    rail.scrollLeft =
      drag.startScrollLeft -
      distance;
  }

  function finishPointerDrag(
    event:
      React.PointerEvent<HTMLDivElement>,
  ) {
    const rail =
      railRef.current;

    const drag =
      dragRef.current;

    if (
      !rail ||
      !drag.active ||
      drag.pointerId !==
        event.pointerId
    ) {
      return;
    }

    dragRef.current.active =
      false;

    delete rail.dataset
      .dragging;

    if (
      rail.hasPointerCapture(
        event.pointerId,
      )
    ) {
      rail.releasePointerCapture(
        event.pointerId,
      );
    }

    if (
      rail.clientWidth <=
      0
    ) {
      return;
    }

    const index =
      Math.round(
        rail.scrollLeft /
          rail.clientWidth,
      );

    goTo(
      Math.max(
        0,
        Math.min(
          slides.length - 1,
          index,
        ),
      ),
    );
  }

  function handleDot(
    index: number,
  ) {
    markManualInteraction();

    goTo(index);
  }

  return (
    <section
      className={
        styles.hero
      }
      aria-label="Popular business products"
      aria-roledescription="carousel"
    >
      <div
        ref={railRef}
        className={
          styles.rail
        }
        onScroll={
          handleScroll
        }
        onPointerDown={
          handlePointerDown
        }
        onPointerMove={
          handlePointerMove
        }
        onPointerUp={
          finishPointerDrag
        }
        onPointerCancel={
          finishPointerDrag
        }
        onWheel={
          markManualInteraction
        }
        onFocusCapture={
          markManualInteraction
        }
      >
        {slides.map(
          (
            product,
            index,
          ) => {
            const campaign =
              campaigns[
                index %
                  campaigns.length
              ];

            return (
              <article
                className={
                  styles.slide
                }
                key={
                  product?.id ??
                  `fallback-${index}`
                }
                aria-label={`Featured product ${
                  index + 1
                } of ${
                  slides.length
                }`}
              >
                <div
                  className={
                    styles.copy
                  }
                >
                  <span
                    className={
                      styles.eyebrow
                    }
                  >
                    {
                      campaign.eyebrow
                    }
                  </span>

                  {index ===
                  0 ? (
                    <h1>
                      {
                        campaign.first
                      }

                      <span>
                        {
                          campaign.second
                        }
                      </span>
                    </h1>
                  ) : (
                    <h2>
                      {
                        campaign.first
                      }

                      <span>
                        {
                          campaign.second
                        }
                      </span>
                    </h2>
                  )}

                  <p>
                    {
                      campaign.description
                    }
                  </p>

                  <div
                    className={
                      styles.actions
                    }
                  >
                    <Link
                      href="#home-catalog"
                      className={
                        styles.primaryAction
                      }
                    >
                      Explore products
                    </Link>

                    <Link
                      href="/account/request"
                      className={
                        styles.secondaryAction
                      }
                    >
                      Request a product
                    </Link>
                  </div>
                </div>

                <div
                  className={
                    styles.visual
                  }
                >
                  <span
                    className={
                      styles.visualHalo
                    }
                    aria-hidden="true"
                  />

                  {product?.primary_image_url ? (
                    <>
                      <div
                        className={
                          styles.productStage
                        }
                      >
                        <CatalogImage
                          src={
                            product.primary_image_url
                          }
                          alt={
                            product.name
                          }
                          fill
                          priority={
                            index === 0
                          }
                          sizes="(max-width: 48rem) 48vw, (max-width: 64rem) 45vw, 43vw"
                          className={
                            styles.productImage
                          }
                        />
                      </div>

                      <Link
                        href={`/product/${product.slug}`}
                        className={
                          styles.productInfo
                        }
                        aria-label={`View ${product.name}`}
                      >
                        <span
                          className={
                            styles.productInfoLabel
                          }
                        >
                          Popular product
                        </span>

                        <strong
                          className={
                            styles.productName
                          }
                        >
                          {
                            product.name
                          }
                        </strong>

                        <span
                          className={
                            styles.productInfoBottom
                          }
                        >
                          <span
                            className={
                              product.in_stock
                                ? styles.inStock
                                : styles.outOfStock
                            }
                          >
                            <span
                              className={
                                styles.stockDot
                              }
                              aria-hidden="true"
                            />

                            {product.in_stock
                              ? "In stock"
                              : "Out of stock"}
                          </span>

                          <strong>
                            {formatMoney(
                              product.price_amount,
                              product.currency,
                            )}
                          </strong>
                        </span>
                      </Link>
                    </>
                  ) : (
                    <div
                      className={
                        styles.fallback
                      }
                      aria-hidden="true"
                    >
                      ED
                    </div>
                  )}
                </div>
              </article>
            );
          },
        )}
      </div>

      {slides.length >
      1 ? (
        <div
          className={
            styles.dots
          }
          aria-label="Featured product slides"
        >
          {slides.map(
            (
              product,
              index,
            ) => (
              <button
                key={
                  product?.id ??
                  `dot-${index}`
                }
                type="button"
                className={
                  index ===
                  activeIndex
                    ? styles.activeDot
                    : undefined
                }
                aria-label={`Show featured slide ${
                  index + 1
                }`}
                aria-current={
                  index ===
                  activeIndex
                    ? "true"
                    : undefined
                }
                onClick={() =>
                  handleDot(
                    index,
                  )
                }
              />
            ),
          )}
        </div>
      ) : null}
    </section>
  );
}