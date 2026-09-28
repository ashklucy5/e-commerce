// Location: src/app/(storefront)/category/[slug]/page.tsx
import type { Metadata } from "next";
import Link from "next/link";
import { notFound } from "next/navigation";

import { ProductCard } from "@/components/commerce/components/ProductCard";
import { CategoryControls } from "@/components/category/components/CategoryControls";
import { CategoryHero } from "@/components/category/components/CategoryHero";
import { CategoryPagination } from "@/components/category/components/CategoryPagination";
import { CategorySubcategoryRail } from "@/components/category/components/CategorySubcategoryRail";
import styles from "@/components/category/css/CategoryPage.module.css";
import {
  getCategory,
  getCategoryTree,
  getProductsByCategory,
  type ProductListSort,
} from "@/lib/api/catalog";
import type {
  CategoryNode,
  ProductCard as CatalogProductCard,
} from "@/lib/api/contracts/catalog";
import { CommerceApiError } from "@/lib/api/error";
import { siteConfig } from "@/lib/config/site";

const PAGE_SIZE = 30;

const VALID_SORTS = new Set<ProductListSort>([
  "recommended",
  "featured",
  "newest",
  "price_asc",
  "price_desc",
  "name_asc",
  "name_desc",
]);

type SearchParamValue =
  | string
  | string[]
  | undefined;

type CategorySearchParams = {
  page?: SearchParamValue;
  sort?: SearchParamValue;
  stock?: SearchParamValue;
  min_price?: SearchParamValue;
  max_price?: SearchParamValue;
  subcategory?: SearchParamValue;
};

type CategoryPageProps = {
  params: Promise<{
    slug: string;
  }>;
  searchParams: Promise<CategorySearchParams>;
};

type CategoryState = {
  page: number;
  sort: ProductListSort;
  stock: "" | "true" | "false";
  minPrice: string;
  maxPrice: string;
  subcategory: string;
};

function firstValue(
  value: SearchParamValue,
) {
  if (Array.isArray(value)) {
    return value[0] ?? "";
  }

  return value ?? "";
}

function parsePage(
  value: SearchParamValue,
) {
  const parsed = Number.parseInt(
    firstValue(value),
    10,
  );

  if (
    !Number.isFinite(parsed) ||
    parsed < 1
  ) {
    return 1;
  }

  return Math.min(
    parsed,
    100000,
  );
}

function normalizeSort(
  value: SearchParamValue,
): ProductListSort {
  const candidate = firstValue(
    value,
  ) as ProductListSort;

  return VALID_SORTS.has(candidate)
    ? candidate
    : "recommended";
}

function normalizeStock(
  value: SearchParamValue,
): "" | "true" | "false" {
  const candidate = firstValue(value);

  if (
    candidate === "true" ||
    candidate === "false"
  ) {
    return candidate;
  }

  return "";
}

function normalizePrice(
  value: SearchParamValue,
) {
  const raw = firstValue(value).trim();

  if (!raw) {
    return "";
  }

  const parsed = Number(raw);

  if (
    !Number.isFinite(parsed) ||
    parsed < 0
  ) {
    return "";
  }

  const rounded =
    Math.round(parsed * 100) /
    100;

  return String(rounded);
}

function resolveState(
  searchParams: CategorySearchParams,
): CategoryState {
  const page = parsePage(
    searchParams.page,
  );

  const sort = normalizeSort(
    searchParams.sort,
  );

  const stock = normalizeStock(
    searchParams.stock,
  );

  let minPrice = normalizePrice(
    searchParams.min_price,
  );

  let maxPrice = normalizePrice(
    searchParams.max_price,
  );

  if (
    minPrice &&
    maxPrice &&
    Number(minPrice) >
      Number(maxPrice)
  ) {
    minPrice = "";
    maxPrice = "";
  }

  return {
    page,
    sort,
    stock,
    minPrice,
    maxPrice,
    subcategory: firstValue(searchParams.subcategory).trim(),
  };
}

function toMinorAmount(
  value: string,
) {
  if (!value) {
    return undefined;
  }

  const number = Number(value);

  if (!Number.isFinite(number)) {
    return undefined;
  }

  return Math.round(
    number * 100,
  );
}

function findCategoryPath(
  categories: CategoryNode[],
  slug: string,
  ancestors: CategoryNode[] = [],
): CategoryNode[] | null {
  for (const category of categories) {
    const path = [
      ...ancestors,
      category,
    ];

    if (category.slug === slug) {
      return path;
    }

    const childPath =
      findCategoryPath(
        category.children ?? [],
        slug,
        path,
      );

    if (childPath) {
      return childPath;
    }
  }

  return null;
}

function categoryHref(
  slug: string,
  state: CategoryState,
  page = state.page,
) {
  const params = new URLSearchParams();

  if (page > 1) {
    params.set(
      "page",
      String(page),
    );
  }

  if (state.sort !== "recommended") {
    params.set(
      "sort",
      state.sort,
    );
  }

  if (state.stock) {
    params.set(
      "stock",
      state.stock,
    );
  }

  if (state.minPrice) {
    params.set(
      "min_price",
      state.minPrice,
    );
  }

  if (state.maxPrice) {
    params.set(
      "max_price",
      state.maxPrice,
    );
  }

  if (state.subcategory) {
    params.set(
      "subcategory",
      state.subcategory,
    );
  }

  const query = params.toString();
  const pathname =
    `/category/${encodeURIComponent(
      slug,
    )}`;

  return query
    ? `${pathname}?${query}`
    : pathname;
}

function hasFacetQuery(
  searchParams: CategorySearchParams,
) {
  return Boolean(
    firstValue(searchParams.sort) ||
      firstValue(searchParams.stock) ||
      firstValue(
        searchParams.min_price,
      ) ||
      firstValue(
        searchParams.max_price,
      ) ||
      firstValue(
        searchParams.subcategory,
      ),
  );
}

function pickHeroImage(
  categoryImage: string | null | undefined,
  products: CatalogProductCard[],
) {
  const direct =
    categoryImage?.trim();

  if (direct) {
    return direct;
  }

  return (
    products.find(
      (product) =>
        product.in_stock &&
        product.primary_image_url,
    )?.primary_image_url ??
    products.find(
      (product) =>
        product.primary_image_url,
    )?.primary_image_url
  );
}

export async function generateMetadata({
  params,
  searchParams,
}: CategoryPageProps): Promise<Metadata> {
  const [{ slug }, rawSearchParams] =
    await Promise.all([
      params,
      searchParams,
    ]);

  const state = resolveState(
    rawSearchParams,
  );

  const faceted = hasFacetQuery(
    rawSearchParams,
  );

  try {
    const category = await getCategory(
      slug,
    );

    const basePath =
      `/category/${category.slug}`;

    const canonical = faceted
      ? basePath
      : categoryHref(
          category.slug,
          {
            ...state,
            sort: "recommended",
            stock: "",
            minPrice: "",
            maxPrice: "",
          },
          state.page,
        );

    const description =
      category.description?.trim() ||
      `Browse ${category.name} products for business purchasing, sourcing, and dependable commerce on Ene Dei.`;

    const pageSuffix =
      !faceted && state.page > 1
        ? ` – Page ${state.page}`
        : "";

    const image =
      category.image_url?.trim();

    return {
      title:
        `${category.name}${pageSuffix}`,

      description,

      alternates: {
        canonical,
      },

      robots: faceted
        ? {
            index: false,
            follow: true,
          }
        : {
            index: true,
            follow: true,
          },

      openGraph: {
        type: "website",
        title:
          `${category.name}${pageSuffix} | Ene Dei`,
        description,
        url: canonical,
        images: image
          ? [
              {
                url: image,
                alt: category.name,
              },
            ]
          : undefined,
      },

      twitter: {
        card: image
          ? "summary_large_image"
          : "summary",
        title:
          `${category.name}${pageSuffix} | Ene Dei`,
        description,
        images: image
          ? [image]
          : undefined,
      },
    };
  } catch {
    return {
      title: "Category",
      robots: {
        index: false,
        follow: false,
      },
    };
  }
}

export default async function CategoryPage({
  params,
  searchParams,
}: CategoryPageProps) {
  const [{ slug }, rawSearchParams] =
    await Promise.all([
      params,
      searchParams,
    ]);

  let category;

  try {
    category = await getCategory(slug);
  } catch (error) {
    if (
      error instanceof CommerceApiError &&
      error.status === 404
    ) {
      notFound();
    }

    throw error;
  }

  const state = resolveState(
    rawSearchParams,
  );

  const inStock =
    state.stock === "true"
      ? true
      : state.stock === "false"
        ? false
        : undefined;

  const categoryTree =
    await getCategoryTree().catch(
      () => [],
    );

  const categoryPath =
    findCategoryPath(
      categoryTree,
      category.slug,
    ) ?? [];

  const categoryNode =
    categoryPath.at(-1);

  const subcategories =
    categoryNode?.children ?? [];

  const selectedSubcategoryPath =
    state.subcategory
      ? findCategoryPath(
          subcategories,
          state.subcategory,
        ) ?? []
      : [];

  const selectedSubcategory =
    selectedSubcategoryPath.at(-1);

  const effectiveState: CategoryState = {
    ...state,
    subcategory:
      selectedSubcategory?.slug ?? "",
  };

  const listingCategorySlug =
    selectedSubcategory?.slug ??
    category.slug;

  const listingPromise =
    getProductsByCategory(
      listingCategorySlug,
      PAGE_SIZE,
      {
        page: effectiveState.page,
        sort: effectiveState.sort,
        inStock,
        minPriceAmount:
          toMinorAmount(
            effectiveState.minPrice,
          ),
        maxPriceAmount:
          toMinorAmount(
            effectiveState.maxPrice,
          ),
      },
    );

  /*
   * Keep the hero based on the parent category so
   * selecting a subcategory filters the catalog
   * without changing the parent landing context.
   */
  const heroPromise =
    getProductsByCategory(
      category.slug,
      6,
      {
        page: 1,
        sort: "featured",
      },
    );

  const [
    listingResult,
    heroResult,
  ] = await Promise.allSettled([
    listingPromise,
    heroPromise,
  ]);

  if (
    listingResult.status ===
    "rejected"
  ) {
    throw listingResult.reason;
  }

  const productsResponse =
    listingResult.value;

  const heroResponse =
    heroResult.status ===
    "fulfilled"
      ? heroResult.value
      : productsResponse;

  const products =
    productsResponse.data;

  const meta =
    productsResponse.meta;

  if (
    effectiveState.page > 1 &&
    meta.total > 0 &&
    effectiveState.page > meta.total_pages
  ) {
    notFound();
  }

  const heroImage = pickHeroImage(
    category.image_url ??
      categoryNode?.image_url,
    heroResponse.data,
  );

  const categoryTotal =
    heroResponse.meta.total;

  const faceted = hasFacetQuery(
    rawSearchParams,
  );

  const canonicalPath = faceted
    ? `/category/${category.slug}`
    : categoryHref(
        category.slug,
        {
          ...effectiveState,
          sort: "recommended",
          stock: "",
          minPrice: "",
          maxPrice: "",
        },
        effectiveState.page,
      );

  const canonicalURL =
    `${siteConfig.url}${canonicalPath}`;

  const actualPath = categoryHref(
    category.slug,
    effectiveState,
  );

  const actualURL =
    `${siteConfig.url}${actualPath}`;

  const breadcrumbNodes =
    categoryPath.length > 0
      ? categoryPath
      : [
          {
            id: category.id,
            name: category.name,
            slug: category.slug,
            description:
              category.description,
            image_url:
              category.image_url,
            icon_url:
              category.icon_url,
            sort_order:
              category.sort_order,
            children: [],
          } satisfies CategoryNode,
        ];

  const breadcrumbItems = [
    {
      "@type": "ListItem",
      position: 1,
      name: "Home",
      item: siteConfig.url,
    },
    ...breadcrumbNodes.map(
      (node, index) => ({
        "@type": "ListItem",
        position: index + 2,
        name: node.name,
        item:
          `${siteConfig.url}/category/${node.slug}`,
      }),
    ),
  ];

  const itemList = {
    "@type": "ItemList",
    "@id":
      `${actualURL}#products`,
    numberOfItems:
      meta.total,
    itemListElement:
      products.map(
        (product, index) => ({
          "@type": "ListItem",
          position:
            (effectiveState.page - 1) *
              PAGE_SIZE +
            index +
            1,
          url:
            `${siteConfig.url}/product/${product.slug}`,
          name: product.name,
        }),
      ),
  };

  const structuredData = {
    "@context": "https://schema.org",
    "@graph": [
      {
        "@type": "CollectionPage",
        "@id":
          `${canonicalURL}#collection`,
        url: canonicalURL,
        name: category.name,
        description:
          category.description?.trim() ||
          `Browse ${category.name} products on Ene Dei.`,
        ...(faceted
          ? {}
          : {
              mainEntity: {
                "@id":
                  `${actualURL}#products`,
              },
            }),
      },
      ...(faceted
        ? []
        : [itemList]),
      {
        "@type": "BreadcrumbList",
        itemListElement:
          breadcrumbItems,
      },
    ],
  };

  const jsonLD = JSON.stringify(
    structuredData,
  ).replace(/</g, "\\u003c");

  const hrefForPage = (
    page: number,
  ) =>
    categoryHref(
      category.slug,
      effectiveState,
      page,
    );

  return (
    <main className={styles.page}>
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: jsonLD,
        }}
      />

      <div className="site-container">
        <nav
          className={styles.breadcrumbs}
          aria-label="Breadcrumb"
        >
          <Link href="/">Home</Link>

          {breadcrumbNodes.map(
            (node, index) => {
              const current =
                index ===
                breadcrumbNodes.length - 1;

              return (
                <span
                  key={node.id}
                  className={styles.pageGroup}
                >
                  <span aria-hidden="true">
                    /
                  </span>

                  {current ? (
                    <span
                      className={
                        styles.breadcrumbCurrent
                      }
                      aria-current="page"
                    >
                      {node.name}
                    </span>
                  ) : (
                    <Link
                      href={`/category/${node.slug}`}
                    >
                      {node.name}
                    </Link>
                  )}
                </span>
              );
            },
          )}
        </nav>

        <CategoryHero
          name={category.name}
          description={
            category.description
          }
          total={categoryTotal}
          imageURL={heroImage}
        />

        <CategorySubcategoryRail
          categoryName={category.name}
          categorySlug={category.slug}
          subcategories={subcategories}
          selectedSubcategorySlug={effectiveState.subcategory}
          sort={effectiveState.sort}
          stock={effectiveState.stock}
          minPrice={effectiveState.minPrice}
          maxPrice={effectiveState.maxPrice}
        />

        <CategoryControls
          total={meta.total}
          page={effectiveState.page}
          totalPages={meta.total_pages}
          sort={effectiveState.sort}
          stock={effectiveState.stock}
          minPrice={effectiveState.minPrice}
          maxPrice={effectiveState.maxPrice}
          subcategory={effectiveState.subcategory}
        />

        {products.length > 0 ? (
          <section
            className={styles.results}
            aria-label={`${category.name} products`}
          >
            <div className={styles.productGrid}>
              {products.map(
                (product) => (
                  <ProductCard
                    key={product.id}
                    product={product}
                  />
                ),
              )}
            </div>

            <CategoryPagination
              page={effectiveState.page}
              totalPages={meta.total_pages}
              hrefForPage={hrefForPage}
            />
          </section>
        ) : (
          <section
            className={styles.emptyState}
            aria-live="polite"
          >
            <h2>No matching products</h2>

            <p>
              Try clearing the current filters,
              or continue through the available
              subcategories.
            </p>

            <Link
              href={categoryHref(
                category.slug,
                {
                  ...effectiveState,
                  page: 1,
                  sort: "recommended",
                  stock: "",
                  minPrice: "",
                  maxPrice: "",
                },
                1,
              )}
            >
              Clear filters
            </Link>
          </section>
        )}
      </div>
    </main>
  );
}
