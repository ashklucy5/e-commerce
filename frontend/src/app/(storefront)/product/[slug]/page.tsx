import type { Metadata } from "next";
import { notFound } from "next/navigation";

import { ProductExperience } from "@/components/product/components/ProductExperience";
import styles from "@/components/product/css/Product.module.css";

import {
  getProduct,
  getProductsByCategory,
} from "@/lib/api/catalog";

import type { ProductCard } from "@/lib/api/contracts/catalog";

import type {
  ProductReview,
  ProductReviewSummary,
} from "@/lib/api/contracts/reviews";

import { CommerceApiError } from "@/lib/api/error";

import {
  getProductReviews,
  getProductReviewSummary,
} from "@/lib/api/reviews";

import { siteConfig } from "@/lib/config/site";

type ProductPageProps = {
  params: Promise<{
    slug: string;
  }>;
};

export async function generateMetadata({
  params,
}: ProductPageProps): Promise<Metadata> {
  const { slug } = await params;

  try {
    const product =
      await getProduct(slug);

    const primaryImage =
      product.images.find(
        (image) =>
          image.is_primary,
      ) ??
      product.images[0];

    const description =
      product.short_description ??
      product.description ??
      `Shop ${product.name} on Ene Dei.`;

    return {
      title:
        product.name,

      description,

      alternates: {
        canonical:
          `/product/${product.slug}`,
      },

      openGraph: {
        type:
          "website",

        title:
          product.name,

        description,

        url:
          `/product/${product.slug}`,

        images:
          primaryImage
            ? [
                {
                  url:
                    primaryImage.url,

                  alt:
                    primaryImage.alt_text ??
                    product.name,
                },
              ]
            : undefined,
      },
    };
  } catch {
    return {
      title:
        "Product",
    };
  }
}

export default async function ProductPage({
  params,
}: ProductPageProps) {
  const { slug } = await params;

  let product;

  try {
    product =
      await getProduct(slug);
  } catch (error) {
    if (
      error instanceof
        CommerceApiError &&
      error.status === 404
    ) {
      notFound();
    }

    throw error;
  }

  const [
    reviewListResult,
    reviewSummaryResult,
    recommendationsResult,
  ] =
    await Promise.allSettled([
      getProductReviews(
        product.id,
      ),

      getProductReviewSummary(
        product.id,
      ),

      /*
       * getProductsByCategory accepts only:
       * categorySlug + optional limit.
       */
      getProductsByCategory(
        product.category.slug,
        30,
      ),
    ]);

  const reviews:
    ProductReview[] =
      reviewListResult.status ===
      "fulfilled"
        ? reviewListResult.value
            .data
        : [];

  const reviewSummary:
    ProductReviewSummary | null =
      reviewSummaryResult.status ===
      "fulfilled"
        ? reviewSummaryResult.value
        : product.review_count &&
            product.rating_average
          ? {
              total_reviews:
                product.review_count,

              verified_purchase_count:
                0,

              average_rating:
                product.rating_average,
            }
          : null;

  const recommendations:
  ProductCard[] =
    recommendationsResult.status ===
    "fulfilled"
      ? recommendationsResult.value
          .data
          .filter(
            (item) =>
              item.id !==
              product.id,
          )
      : [];

  const productURL =
    `${siteConfig.url}/product/${product.slug}`;

  const categoryURL =
    `${siteConfig.url}/category/${product.category.slug}`;

  const primaryVariant =
    product.variants.find(
      (variant) =>
        variant.in_stock &&
        variant.available_quantity >
          0,
    ) ??
    product.variants[0];

  const aggregateRating =
    reviewSummary &&
    reviewSummary.total_reviews >
      0
      ? {
          "@type":
            "AggregateRating",

          ratingValue:
            reviewSummary.average_rating,

          reviewCount:
            reviewSummary.total_reviews,

          bestRating:
            5,

          worstRating:
            1,
        }
      : undefined;

  const variantProducts =
    product.variants.map(
      (variant) => ({
        "@type":
          "Product",

        "@id":
          `${productURL}#variant-${variant.id}`,

        name:
          product.name,

        sku:
          variant.sku,

        color:
          variant.color_name ??
          undefined,

        size:
          variant.size ??
          undefined,

        image:
          product.images
            .filter(
              (image) =>
                !image.variant_id ||
                image.variant_id ===
                  variant.id,
            )
            .map(
              (image) =>
                image.url,
            ),

        brand:
          product.brand
            ? {
                "@type":
                  "Brand",

                name:
                  product.brand,
              }
            : undefined,

        offers: {
          "@type":
            "Offer",

          url:
            productURL,

          priceCurrency:
            variant.currency,

          price:
            (
              variant.price_amount /
              100
            ).toFixed(
              2,
            ),

          availability:
            variant.in_stock &&
            variant.available_quantity >
              0
              ? "https://schema.org/InStock"
              : "https://schema.org/OutOfStock",

          itemCondition:
            "https://schema.org/NewCondition",
        },
      }),
    );

  const schemaProduct =
    product.variants.length >
    1
      ? {
          "@type":
            "ProductGroup",

          "@id":
            `${productURL}#product-group`,

          name:
            product.name,

          description:
            product.description ??
            product.short_description,

          productGroupID:
            product.product_code,

          url:
            productURL,

          image:
            product.images.map(
              (image) =>
                image.url,
            ),

          category:
            product.category.name,

          brand:
            product.brand
              ? {
                  "@type":
                    "Brand",

                  name:
                    product.brand,
                }
              : undefined,

          variesBy: [
            ...(product.variants.some(
              (variant) =>
                variant.color_name,
            )
              ? [
                  "https://schema.org/color",
                ]
              : []),

            ...(product.variants.some(
              (variant) =>
                variant.size,
            )
              ? [
                  "https://schema.org/size",
                ]
              : []),
          ],

          aggregateRating,

          hasVariant:
            variantProducts,
        }
      : {
          "@type":
            "Product",

          "@id":
            `${productURL}#product`,

          name:
            product.name,

          description:
            product.description ??
            product.short_description,

          sku:
            primaryVariant?.sku ??
            product.product_code,

          url:
            productURL,

          image:
            product.images.map(
              (image) =>
                image.url,
            ),

          category:
            product.category.name,

          brand:
            product.brand
              ? {
                  "@type":
                    "Brand",

                  name:
                    product.brand,
                }
              : undefined,

          aggregateRating,

          offers:
            primaryVariant
              ? {
                  "@type":
                    "Offer",

                  url:
                    productURL,

                  priceCurrency:
                    primaryVariant.currency,

                  price:
                    (
                      primaryVariant.price_amount /
                      100
                    ).toFixed(
                      2,
                    ),

                  availability:
                    primaryVariant.in_stock &&
                    primaryVariant.available_quantity >
                      0
                      ? "https://schema.org/InStock"
                      : "https://schema.org/OutOfStock",

                  itemCondition:
                    "https://schema.org/NewCondition",
                }
              : undefined,
        };

  /*
   * Visible breadcrumbs are intentionally not
   * shown because they waste valuable commerce
   * space.
   *
   * Keep BreadcrumbList structured data for SEO.
   */
  const structuredData = {
    "@context":
      "https://schema.org",

    "@graph": [
      schemaProduct,

      {
        "@type":
          "BreadcrumbList",

        itemListElement: [
          {
            "@type":
              "ListItem",

            position:
              1,

            name:
              "Home",

            item:
              siteConfig.url,
          },

          {
            "@type":
              "ListItem",

            position:
              2,

            name:
              product.category
                .name,

            item:
              categoryURL,
          },

          {
            "@type":
              "ListItem",

            position:
              3,

            name:
              product.name,

            item:
              productURL,
          },
        ],
      },
    ],
  };

  const jsonLD =
    JSON.stringify(
      structuredData,
    ).replace(
      /</g,
      "\\u003c",
    );

  return (
    <div
      className={`product-route ${styles.page}`}
    >
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html:
            jsonLD,
        }}
      />

      <div className="site-container">
        <ProductExperience
          product={
            product
          }
          reviews={
            reviews
          }
          reviewSummary={
            reviewSummary
          }
          recommendations={
            recommendations
          }
          backHref={`/category/${product.category.slug}`}
        />
      </div>
    </div>
  );
}