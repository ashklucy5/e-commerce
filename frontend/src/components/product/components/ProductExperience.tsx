"use client";

import {
  useMemo,
  useState,
} from "react";

import type {
  ProductCard,
  ProductDetail,
  ProductImage,
  ProductVariant,
} from "@/lib/api/contracts/catalog";

import type {
  ProductReview,
  ProductReviewSummary,
} from "@/lib/api/contracts/reviews";

import { ProductGallery } from "./ProductMediaGallery";
import { ProductInformation } from "./ProductInformation";
import { ProductPurchasePanel } from "./ProductPurchasePanel";
import { ProductRecommendations } from "./ProductRecommendations";
import { ProductReviews } from "./ProductReviews";

import styles from "../css/ProductExperience.module.css";

type Props = {
  product:
    ProductDetail;

  reviews:
    ProductReview[];

  reviewSummary:
    ProductReviewSummary | null;

  recommendations:
    ProductCard[];

  backHref:
    string;
};

function minimumQuantity(
  variant:
    ProductVariant |
    undefined,
) {
  return Math.max(
    1,
    variant
      ?.minimum_order_quantity ??
      1,
  );
}

function preferredImage(
  images:
    ProductImage[],
  variantID?:
    string,
) {
  return (
    images.find(
      (image) =>
        variantID &&
        image.variant_id ===
          variantID,
    ) ??
    images.find(
      (image) =>
        image.is_primary,
    ) ??
    [...images].sort(
      (a, b) =>
        a.sort_order -
        b.sort_order,
    )[0]
  );
}

function isPurchasable(
  variant:
    ProductVariant |
    undefined,
) {
  if (!variant) {
    return false;
  }

  return (
    variant.in_stock &&
    variant.available_quantity >=
      Math.max(
        1,
        variant.minimum_order_quantity,
      )
  );
}

function RatingStars({
  rating,
}: {
  rating:
    number;
}) {
  const rounded =
    Math.round(
      Math.max(
        0,
        Math.min(
          5,
          rating,
        ),
      ),
    );

  return (
    <span
      className={
        styles.stars
      }
      aria-label={`${rating.toFixed(
        1,
      )} out of 5 stars`}
    >
      {[1, 2, 3, 4, 5].map(
        (star) => (
          <span
            key={
              star
            }
            className={
              star <=
              rounded
                ? styles.starFilled
                : styles.starEmpty
            }
            aria-hidden="true"
          >
            ★
          </span>
        ),
      )}
    </span>
  );
}

export function ProductExperience({
  product,
  reviews,
  reviewSummary,
  recommendations,
  backHref,
}: Props) {
  const initialVariant =
    useMemo(
      () =>
        product.variants.find(
          (variant) =>
            isPurchasable(
              variant,
            ),
        ) ??
        product.variants.find(
          (variant) =>
            variant.in_stock,
        ) ??
        product.variants[0],
      [
        product.variants,
      ],
    );

  const initialImage =
    useMemo(
      () =>
        preferredImage(
          product.images,
          initialVariant?.id,
        ),
      [
        initialVariant,
        product.images,
      ],
    );

  const [
    selectedVariantID,
    setSelectedVariantID,
  ] =
    useState(
      initialVariant?.id ??
        "",
    );

  const [
    selectedImageID,
    setSelectedImageID,
  ] =
    useState(
      initialImage?.id ??
        "",
    );

  const [
    quantity,
    setQuantity,
  ] =
    useState(
      minimumQuantity(
        initialVariant,
      ),
    );

  const selectedVariant =
    useMemo(
      () =>
        product.variants.find(
          (variant) =>
            variant.id ===
            selectedVariantID,
        ) ??
        initialVariant,
      [
        initialVariant,
        product.variants,
        selectedVariantID,
      ],
    );

  const rating =
    reviewSummary
      ?.average_rating ??
    product.rating_average ??
    0;

  const reviewCount =
    reviewSummary
      ?.total_reviews ??
    product.review_count ??
    0;

  const sold =
    product.sold_quantity ??
    0;

  const selectedInStock =
    isPurchasable(
      selectedVariant,
    );

  function selectVariant(
    variant:
      ProductVariant,
  ) {
    setSelectedVariantID(
      variant.id,
    );

    setQuantity(
      minimumQuantity(
        variant,
      ),
    );

    const image =
      preferredImage(
        product.images,
        variant.id,
      );

    if (image) {
      setSelectedImageID(
        image.id,
      );
    }
  }

  return (
    <div
      className={
        styles.experience
      }
    >
      <div
        className={
          styles.main
        }
      >
        <section
          className={
            styles.mediaColumn
          }
          aria-label="Product photos"
        >
          <ProductGallery
            productName={
              product.name
            }
            images={
              product.images
            }
            selectedImageID={
              selectedImageID
            }
            onSelectImage={
              setSelectedImageID
            }
            backHref={
              backHref
            }
          />
        </section>

        <aside
          className={
            styles.commerce
          }
          aria-label="Product purchase information"
        >
          <div
            className={
              styles.topMeta
            }
          >
            <div
              className={
                styles.badges
              }
            >
              {product.is_featured ? (
                <span
                  className={
                    styles.featured
                  }
                >
                  Most popular
                </span>
              ) : null}

              <span
                className={
                  selectedInStock
                    ? styles.inStock
                    : styles.outOfStock
                }
              >
                <span
                  className={
                    styles.statusDot
                  }
                  aria-hidden="true"
                />

                {selectedInStock
                  ? "In stock"
                  : "Out of stock"}
              </span>
            </div>

            <span
              className={
                styles.sku
              }
            >
              SKU:{" "}
              {selectedVariant
                ?.sku ??
                product.product_code}
            </span>
          </div>

          <h1
            className={
              styles.title
            }
          >
            {product.name}
          </h1>

          {product.brand ? (
            <p
              className={
                styles.brand
              }
            >
              Brand:{" "}

              <strong>
                {
                  product.brand
                }
              </strong>
            </p>
          ) : null}

          <div
            className={
              styles.ratingRow
            }
          >
            {reviewCount >
            0 ? (
              <>
                <RatingStars
                  rating={
                    rating
                  }
                />

                <a href="#product-reviews">
                  {rating.toFixed(
                    1,
                  )}{" "}
                  (
                  {reviewCount.toLocaleString()}
                  )
                </a>
              </>
            ) : (
              <span
                className={
                  styles.newProduct
                }
              >
                No reviews yet
              </span>
            )}

            {sold > 0 ? (
              <>
                <span
                  className={
                    styles.separator
                  }
                  aria-hidden="true"
                />

                <span>
                  {sold.toLocaleString()}{" "}
                  sold
                </span>
              </>
            ) : null}
          </div>

          {selectedVariant ? (
            <ProductPurchasePanel
              productName={
                product.name
              }
              variants={
                product.variants
              }
              selectedVariant={
                selectedVariant
              }
              quantity={
                quantity
              }
              onSelectVariant={
                selectVariant
              }
              onQuantityChange={
                setQuantity
              }
            />
          ) : (
            <div
              className={
                styles.unavailable
              }
            >
              <strong>
                Currently
                unavailable
              </strong>

              <span>
                This product
                does not have
                a purchasable
                variant.
              </span>
            </div>
          )}
        </aside>
      </div>

      <ProductInformation
        product={
          product
        }
        selectedVariant={
          selectedVariant
        }
      />

      <ProductReviews
        reviews={
          reviews
        }
        summary={
          reviewSummary
        }
        fallbackRating={
          product.rating_average ??
          0
        }
        fallbackCount={
          product.review_count ??
          0
        }
      />

      <ProductRecommendations
        products={
          recommendations
        }
      />
    </div>
  );
}