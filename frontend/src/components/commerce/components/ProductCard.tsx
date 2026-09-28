import Image from "next/image";
import Link from "next/link";

import { ProductCardCartButton } from "@/components/commerce/components/ProductCardCartButton";
import { ProductWishlistButton } from "@/components/commerce/components/ProductWishlistButton";

import type {
  ProductCard as Product,
} from "@/lib/api/contracts/catalog";

import {
  formatMoney,
} from "@/lib/money/format";

import styles from "../css/ProductCard.module.css";

type MerchandisingProduct =
  Product & {
    compare_at_price_amount?:
      | number
      | null;

    sold_quantity?: number;

    merchandising_group?:
      | "new"
      | "discount"
      | "popular"
      | "catalog";
  };

type Props = {
  product: MerchandisingProduct;
  eager?: boolean;
};

function getDiscountPercent(
  price: number,
  compareAt?: number | null,
) {
  if (
    !compareAt ||
    compareAt <= price ||
    compareAt <= 0
  ) {
    return 0;
  }

  return Math.round(
    ((compareAt - price) /
      compareAt) *
      100,
  );
}

function getBadge(
  product: MerchandisingProduct,
  discount: number,
) {
  if (!product.in_stock) {
    return {
      label: "Sold out",
      className:
        styles.badgeUnavailable,
    };
  }

  if (discount > 0) {
    return {
      label: `-${discount}%`,
      className:
        styles.badgeSale,
    };
  }

  if (
    product.merchandising_group ===
    "new"
  ) {
    return {
      label: "New",
      className:
        styles.badgeNew,
    };
  }

  if (
    product.merchandising_group ===
    "popular"
  ) {
    return {
      label: "Popular",
      className:
        styles.badgePopular,
    };
  }

  if (product.is_featured) {
    return {
      label: "Featured",
      className:
        styles.badgeFeatured,
    };
  }

  return null;
}

export function ProductCard({
  product,
  eager = false,
}: Props) {
  const discount =
    getDiscountPercent(
      product.price_amount,
      product.compare_at_price_amount,
    );

  const hasDiscount =
    discount > 0;

  const badge =
    getBadge(
      product,
      discount,
    );

  const soldQuantity =
    typeof product.sold_quantity ===
      "number" &&
    product.sold_quantity > 0
      ? product.sold_quantity
      : 0;

  const brand =
    product.brand?.trim() ||
    "Ene dei";

  return (
    <article
      className={styles.card}
      data-available={
        product.in_stock
          ? "true"
          : "false"
      }
    >
      <Link
        href={`/product/${product.slug}`}
        className={styles.link}
        aria-label={`View ${product.name}`}
      >
        <div
          className={styles.media}
        >
          {product.primary_image_url ? (
            <Image
              src={
                product.primary_image_url
              }
              alt={product.name}
              fill
              priority={eager}
              sizes="(max-width: 479px) 47vw, (max-width: 767px) 46vw, (max-width: 1023px) 31vw, (max-width: 1439px) 19vw, 220px"
              className={
                styles.image
              }
            />
          ) : (
            <div
              className={
                styles.fallback
              }
              aria-hidden="true"
            >
              <span>
                {product.name
                  .charAt(0)
                  .toUpperCase()}
              </span>
            </div>
          )}

          <div
            className={
              styles.mediaShade
            }
            aria-hidden="true"
          />

          {badge ? (
            <span
              className={`${styles.badge} ${badge.className}`}
            >
              {badge.label}
            </span>
          ) : null}

          <span
            className={
              styles.openIndicator
            }
            aria-hidden="true"
          >
            <svg
              viewBox="0 0 24 24"
            >
              <path
                d="M7 17 17 7M9 7h8v8"
                fill="none"
                stroke="currentColor"
                strokeWidth="1.8"
                strokeLinecap="round"
                strokeLinejoin="round"
              />
            </svg>
          </span>
        </div>

        <div
          className={styles.body}
        >
          <div
            className={
              styles.topMeta
            }
          >
            <span
              className={
                styles.brand
              }
            >
              {brand}
            </span>

            {soldQuantity > 0 ? (
              <span
                className={
                  styles.sold
                }
              >
                {soldQuantity.toLocaleString()}{" "}
                sold
              </span>
            ) : null}
          </div>

          <h3
            className={styles.title}
            title={product.name}
          >
            {product.name}
          </h3>

          <div
            className={
              styles.priceBlock
            }
          >
            <div
              className={
                styles.priceRow
              }
            >
              <strong
                className={
                  styles.price
                }
              >
                {formatMoney(
                  product.price_amount,
                  product.currency,
                )}
              </strong>

              {hasDiscount &&
              product.compare_at_price_amount ? (
                <span
                  className={
                    styles.oldPrice
                  }
                >
                  {formatMoney(
                    product.compare_at_price_amount,
                    product.currency,
                  )}
                </span>
              ) : null}
            </div>

            <div
              className={
                styles.bottomMeta
              }
            >
              <div
                className={
                  styles.statusGroup
                }
              >
                <span
                  className={
                    product.in_stock
                      ? styles.available
                      : styles.unavailable
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

                {hasDiscount ? (
                  <span
                    className={
                      styles.saving
                    }
                  >
                    Save {discount}%
                  </span>
                ) : null}
              </div>

              <span
                className={
                  styles.viewLabel
                }
              >
                View product
              </span>
            </div>
          </div>
        </div>
      </Link>

      <ProductWishlistButton
        productId={product.id}
        productName={product.name}
        placement="catalog"
      />
      <ProductCardCartButton
        productName={product.name}
        productSlug={product.slug}
        inStock={product.in_stock}
        className={styles.cartButton}
      />
    </article>
  );
}
