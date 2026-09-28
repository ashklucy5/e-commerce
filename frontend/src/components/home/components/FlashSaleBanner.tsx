"use client";

import Image from "next/image";
import Link from "next/link";
import { CatalogImage } from "@/components/commerce/components/CatalogImage";

import {
  type PointerEvent,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";

import type {
  HomeProductCard,
  StorefrontPromotion,
} from "@/lib/api/contracts/home";

import {
  formatMoney,
} from "@/lib/money/format";

import styles from "../css/HomeFeed.module.css";

type Props = {
  promotions: StorefrontPromotion[];
  products: HomeProductCard[];
};

type CampaignSlide = {
  id: string;

  eyebrow: string;
  title: string;
  description: string;

  primaryLabel: string;
  primaryHref: string;

  secondaryLabel?: string;
  secondaryHref?: string;

  product?: HomeProductCard;

  endsAt?: string;

  tone:
    | "sale"
    | "new"
    | "popular";
};

const AUTO_ROTATE_MS = 6500;
const SWIPE_THRESHOLD = 48;

function formatPercentage(
  basisPoints: number,
) {
  const value =
    basisPoints / 100;

  return Number.isInteger(value)
    ? `${value}%`
    : `${value
        .toFixed(2)
        .replace(
          /\.00$/,
          "",
        )}%`;
}

function promotionValue(
  promotion: StorefrontPromotion,
) {
  if (
    promotion.discount_type ===
      "percentage" &&
    typeof promotion.percentage_bps ===
      "number"
  ) {
    return `${formatPercentage(
      promotion.percentage_bps,
    )} off`;
  }

  if (
    promotion.discount_type ===
      "fixed" &&
    typeof promotion.fixed_amount ===
      "number"
  ) {
    return `${formatMoney(
      promotion.fixed_amount,
      promotion.currency,
    )} off`;
  }

  return "Special prices";
}

function formatEndsAt(
  value?: string,
) {
  if (!value) {
    return "";
  }

  const date =
    new Date(value);

  if (
    Number.isNaN(
      date.getTime(),
    )
  ) {
    return "";
  }

  return new Intl.DateTimeFormat(
    "en",
    {
      month: "short",
      day: "numeric",
      hour: "numeric",
      minute: "2-digit",
    },
  ).format(date);
}

function pickPromotion(
  promotions:
    StorefrontPromotion[],
) {
  if (
    promotions.length === 0
  ) {
    return null;
  }

  return (
    promotions.find(
      (promotion) => {
        const name =
          promotion.name.toLowerCase();

        return (
          name.includes("flash") ||
          name.includes("sale")
        );
      },
    ) ??
    promotions[0]
  );
}

function hasDiscount(
  product: HomeProductCard,
) {
  const compareAt =
    product.compare_at_price_amount;

  return (
    typeof compareAt ===
      "number" &&
    compareAt >
      product.price_amount
  );
}

function discountPercent(
  product: HomeProductCard,
) {
  const compareAt =
    product.compare_at_price_amount;

  if (
    typeof compareAt !==
      "number" ||
    compareAt <=
      product.price_amount ||
    compareAt <= 0
  ) {
    return 0;
  }

  return Math.round(
    ((compareAt -
      product.price_amount) /
      compareAt) *
      100,
  );
}

function imageScore(
  product: HomeProductCard,
) {
  let score = 0;

  if (
    product.primary_image_url
  ) {
    score += 100;
  }

  if (
    product.primary_image_url &&
    !product.product_code
      .toUpperCase()
      .startsWith("DEMO")
  ) {
    score += 45;
  }

  if (
    product.is_featured
  ) {
    score += 20;
  }

  if (
    product.sold_quantity >
    0
  ) {
    score += Math.min(
      product.sold_quantity,
      40,
    );
  }

  return score;
}

function bestProduct(
  products: HomeProductCard[],
  usedIds: Set<string>,
  predicate: (
    product: HomeProductCard,
  ) => boolean,
  extraScore?: (
    product: HomeProductCard,
  ) => number,
) {
  return products
    .filter(
      (product) =>
        !usedIds.has(
          product.id,
        ) &&
        predicate(product),
    )
    .sort(
      (
        left,
        right,
      ) => {
        const rightScore =
          imageScore(right) +
          (extraScore?.(
            right,
          ) ??
            0);

        const leftScore =
          imageScore(left) +
          (extraScore?.(
            left,
          ) ??
            0);

        return (
          rightScore -
          leftScore
        );
      },
    )[0];
}

function fallbackProduct(
  products: HomeProductCard[],
  usedIds: Set<string>,
) {
  return products
    .filter(
      (product) =>
        !usedIds.has(
          product.id,
        ),
    )
    .sort(
      (
        left,
        right,
      ) =>
        imageScore(right) -
        imageScore(left),
    )[0];
}

function buildSlides(
  promotions:
    StorefrontPromotion[],
  products:
    HomeProductCard[],
): CampaignSlide[] {
  const usedIds =
    new Set<string>();

  const promotion =
    pickPromotion(
      promotions,
    );

  const dealProduct =
    bestProduct(
      products,
      usedIds,
      hasDiscount,
      (product) =>
        discountPercent(
          product,
        ),
    ) ??
    fallbackProduct(
      products,
      usedIds,
    );

  if (dealProduct) {
    usedIds.add(
      dealProduct.id,
    );
  }

  const newProduct =
    bestProduct(
      products,
      usedIds,
      (product) =>
        product.merchandising_group ===
        "new",
      (product) => {
        const timestamp =
          Date.parse(
            product.published_at ??
              product.created_at,
          );

        return Number.isFinite(
          timestamp,
        )
          ? timestamp /
              1_000_000_000
          : 0;
      },
    ) ??
    fallbackProduct(
      products,
      usedIds,
    );

  if (newProduct) {
    usedIds.add(
      newProduct.id,
    );
  }

  const popularProduct =
    bestProduct(
      products,
      usedIds,
      (product) =>
        product.sold_quantity >
        0,
      (product) =>
        product.sold_quantity *
        4,
    ) ??
    bestProduct(
      products,
      usedIds,
      (product) =>
        product.is_featured,
    ) ??
    fallbackProduct(
      products,
      usedIds,
    );

  const slides:
    CampaignSlide[] = [
      {
        id: "deals",

        eyebrow:
          promotion
            ? "Flash sale"
            : "Price drops",

        title:
          promotion
            ? promotion.name
            : "Better prices. Less searching.",

        description:
          promotion
            ? `${promotionValue(
                promotion,
              )}. Shop eligible offers, then continue through the complete catalog.`
            : "Start with today's strongest price reductions, then keep scrolling through every active product.",

        primaryLabel:
          "Shop deals",

        primaryHref:
          "#home-feed-discount",

        secondaryLabel:
          "Browse all",

        secondaryHref:
          "#home-catalog",

        product:
          dealProduct,

        endsAt:
          promotion?.ends_at,

        tone: "sale",
      },

      {
        id: "new",

        eyebrow:
          "Just landed",

        title:
          "Fresh products, ready to discover.",

        description:
          "New arrivals stay easy to find while the complete catalog remains only a scroll away.",

        primaryLabel:
          "See new arrivals",

        primaryHref:
          "#home-feed-new",

        secondaryLabel:
          "Browse all",

        secondaryHref:
          "#home-catalog",

        product:
          newProduct,

        tone: "new",
      },

      {
        id: "popular",

        eyebrow:
          popularProduct &&
          popularProduct.sold_quantity >
            0
            ? "Customer favorite"
            : "Wholesale picks",

        title:
          popularProduct &&
          popularProduct.sold_quantity >
            0
            ? "Popular for a reason."
            : "Useful picks, without the clutter.",

        description:
          popularProduct &&
          popularProduct.sold_quantity >
            0
            ? "Real purchase activity helps surface products customers are already choosing."
            : "A focused product worth noticing before continuing through the full catalog.",

        primaryLabel:
          popularProduct
            ? "View product"
            : "Browse products",

        primaryHref:
          popularProduct
            ? `/product/${popularProduct.slug}`
            : "#home-catalog",

        secondaryLabel:
          "Keep shopping",

        secondaryHref:
          "#home-catalog",

        product:
          popularProduct,

        tone:
          "popular",
      },
    ];

  return slides;
}

function ProductHeroVisual({
  product,
  eager,
}: {
  product?: HomeProductCard;
  eager: boolean;
}) {
  if (
    !product ||
    !product.primary_image_url
  ) {
    return (
      <div
        className={
          styles.campaignFallback
        }
        aria-hidden="true"
      >
        <span>
          E
        </span>
      </div>
    );
  }

  return (
    <>
      <CatalogImage
        src={
          product.primary_image_url
        }
        alt={product.name}
        fill
        priority={eager}
        sizes="(max-width: 767px) 100vw, 48vw"
        className={
          styles.campaignBackdrop
        }
        aria-hidden="true"
      />

      <div
        className={
          styles.campaignProductFrame
        }
      >
        <Image
          src={
            product.primary_image_url
          }
          alt={
            product.name
          }
          fill
          priority={eager}
          sizes="(max-width: 767px) 88vw, 42vw"
          className={
            styles.campaignProductImage
          }
        />
      </div>

      <Link
        href={`/product/${product.slug}`}
        className={
          styles.campaignProductInfo
        }
      >
        <span
          className={
            styles.campaignProductBrand
          }
        >
          {product.brand?.trim() ||
            "Ene dei"}
        </span>

        <strong>
          {product.name}
        </strong>

        <span
          className={
            styles.campaignProductPrice
          }
        >
          {formatMoney(
            product.price_amount,
            product.currency,
          )}

          {hasDiscount(
            product,
          ) &&
          product.compare_at_price_amount ? (
            <del>
              {formatMoney(
                product.compare_at_price_amount,
                product.currency,
              )}
            </del>
          ) : null}
        </span>
      </Link>
    </>
  );
}

export function FlashSaleBanner({
  promotions,
  products,
}: Props) {
  const slides =
    useMemo(
      () =>
        buildSlides(
          promotions,
          products,
        ),
      [
        promotions,
        products,
      ],
    );

  const [
    activeIndex,
    setActiveIndex,
  ] =
    useState(0);

  const [
    interacting,
    setInteracting,
  ] =
    useState(false);

  const [
    reducedMotion,
    setReducedMotion,
  ] =
    useState(false);

  const pointerStartX =
    useRef<
      number | null
    >(null);

  useEffect(
    () => {
      const media =
        window.matchMedia(
          "(prefers-reduced-motion: reduce)",
        );

      const update = () =>
        setReducedMotion(
          media.matches,
        );

      update();

      media.addEventListener(
        "change",
        update,
      );

      return () =>
        media.removeEventListener(
          "change",
          update,
        );
    },
    [],
  );

  useEffect(
    () => {
      if (
        slides.length <= 1 ||
        reducedMotion ||
        interacting
      ) {
        return;
      }

      const timer =
        window.setTimeout(
          () => {
            setActiveIndex(
              (current) =>
                (current + 1) %
                slides.length,
            );
          },
          AUTO_ROTATE_MS,
        );

      return () =>
        window.clearTimeout(
          timer,
        );
    },
    [
      activeIndex,
      interacting,
      reducedMotion,
      slides.length,
    ],
  );

  if (
    slides.length === 0
  ) {
    return null;
  }

  function showSlide(
    index: number,
  ) {
    setActiveIndex(
      (index +
        slides.length) %
        slides.length,
    );
  }

  function handlePointerDown(
    event:
      PointerEvent<HTMLElement>,
  ) {
    if (
      event.pointerType ===
        "mouse"
    ) {
      return;
    }

    pointerStartX.current =
      event.clientX;
  }

  function handlePointerUp(
    event:
      PointerEvent<HTMLElement>,
  ) {
    const start =
      pointerStartX.current;

    pointerStartX.current =
      null;

    if (
      start === null
    ) {
      return;
    }

    const distance =
      event.clientX -
      start;

    if (
      Math.abs(distance) <
      SWIPE_THRESHOLD
    ) {
      return;
    }

    showSlide(
      distance > 0
        ? activeIndex - 1
        : activeIndex + 1,
    );
  }

  return (
    <section
      className={
        styles.campaignCarousel
      }
      aria-roledescription="carousel"
      aria-label="Featured promotions"
      onMouseEnter={() =>
        setInteracting(true)
      }
      onMouseLeave={() =>
        setInteracting(false)
      }
      onFocusCapture={() =>
        setInteracting(true)
      }
      onBlurCapture={(
        event,
      ) => {
        if (
          !event.currentTarget.contains(
            event.relatedTarget,
          )
        ) {
          setInteracting(
            false,
          );
        }
      }}
      onPointerDown={
        handlePointerDown
      }
      onPointerUp={
        handlePointerUp
      }
      onPointerCancel={() => {
        pointerStartX.current =
          null;
      }}
    >
      <div
        className={
          styles.campaignViewport
        }
      >
        {slides.map(
          (
            slide,
            index,
          ) => {
            const active =
              index ===
              activeIndex;

            const endsAt =
              formatEndsAt(
                slide.endsAt,
              );

            return (
              <article
                key={
                  slide.id
                }
                className={`${styles.campaignSlide} ${
                  active
                    ? styles.campaignSlideActive
                    : ""
                }`}
                aria-hidden={
                  !active
                }
              >
                <div
                  className={
                    styles.campaignCopy
                  }
                >
                  <span
                    className={
                      styles.campaignEyebrow
                    }
                  >
                    {
                      slide.eyebrow
                    }

                    {endsAt ? (
                      <small>
                        Ends{" "}
                        {endsAt}
                      </small>
                    ) : null}
                  </span>

                  <h1>
                    {
                      slide.title
                    }
                  </h1>

                  <p>
                    {
                      slide.description
                    }
                  </p>

                  <div
                    className={
                      styles.campaignActions
                    }
                  >
                    <Link
                      href={
                        slide.primaryHref
                      }
                      className={
                        styles.campaignPrimary
                      }
                      tabIndex={
                        active
                          ? 0
                          : -1
                      }
                    >
                      {
                        slide.primaryLabel
                      }

                      <span
                        aria-hidden="true"
                      >
                        →
                      </span>
                    </Link>

                    {slide.secondaryLabel &&
                    slide.secondaryHref ? (
                      <Link
                        href={
                          slide.secondaryHref
                        }
                        className={
                          styles.campaignSecondary
                        }
                        tabIndex={
                          active
                            ? 0
                            : -1
                        }
                      >
                        {
                          slide.secondaryLabel
                        }
                      </Link>
                    ) : null}
                  </div>
                </div>

                <div
                  className={
                    styles.campaignMedia
                  }
                >
                  <ProductHeroVisual
                    product={
                      slide.product
                    }
                    eager={
                      index ===
                      0
                    }
                  />
                </div>
              </article>
            );
          },
        )}
      </div>

      {slides.length > 1 ? (
        <>
          <button
            type="button"
            className={`${styles.campaignArrow} ${styles.campaignArrowPrevious}`}
            aria-label="Previous promotion"
            onClick={() =>
              showSlide(
                activeIndex -
                  1,
              )
            }
          >
            <svg
              viewBox="0 0 24 24"
              aria-hidden="true"
            >
              <path
                d="m15 5-7 7 7 7"
                fill="none"
                stroke="currentColor"
                strokeWidth="1.8"
                strokeLinecap="round"
                strokeLinejoin="round"
              />
            </svg>
          </button>

          <button
            type="button"
            className={`${styles.campaignArrow} ${styles.campaignArrowNext}`}
            aria-label="Next promotion"
            onClick={() =>
              showSlide(
                activeIndex +
                  1,
              )
            }
          >
            <svg
              viewBox="0 0 24 24"
              aria-hidden="true"
            >
              <path
                d="m9 5 7 7-7 7"
                fill="none"
                stroke="currentColor"
                strokeWidth="1.8"
                strokeLinecap="round"
                strokeLinejoin="round"
              />
            </svg>
          </button>

          <div
            className={
              styles.campaignDots
            }
            aria-label="Promotion slides"
          >
            {slides.map(
              (
                slide,
                index,
              ) => (
                <button
                  key={
                    slide.id
                  }
                  type="button"
                  className={`${styles.campaignDot} ${
                    index ===
                    activeIndex
                      ? styles.campaignDotActive
                      : ""
                  }`}
                  aria-label={`Show promotion ${index + 1}`}
                  aria-current={
                    index ===
                    activeIndex
                      ? "true"
                      : undefined
                  }
                  onClick={() =>
                    showSlide(
                      index,
                    )
                  }
                />
              ),
            )}
          </div>
        </>
      ) : null}
    </section>
  );
}