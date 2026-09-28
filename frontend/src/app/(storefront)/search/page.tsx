import type {
  Metadata,
} from "next";

import Link from "next/link";

import {
  ProductCard,
} from "@/components/commerce/components/ProductCard";

import type {
  ProductCard as Product,
} from "@/lib/api/contracts/catalog";

import {
  commerceFetch,
} from "@/lib/api/server";

export const dynamic =
  "force-dynamic";

export const metadata: Metadata = {
  title: "Search",

  description:
    "Search products, brands, categories and product codes.",

  alternates: {
    canonical: "/search",
  },

  robots: {
    index: false,
    follow: true,
  },
};

type SearchPageProps = {
  searchParams: Promise<{
    q?: string | string[];
    page?: string | string[];
  }>;
};

type SearchMeta = {
  query: string;

  page: number;
  limit: number;

  total: number;
  total_pages: number;

  primary_total?: number;
  similar_total?: number;

  no_match?: boolean;
  showing_similar?: boolean;
};

type SearchResponse = {
  data: Product[];
  meta: SearchMeta;
};

const SEARCH_LIMIT = 30;

function firstValue(
  value:
    | string
    | string[]
    | undefined,
) {
  if (
    Array.isArray(
      value,
    )
  ) {
    return (
      value[0] ?? ""
    );
  }

  return value ?? "";
}

function parsePage(
  value: string,
) {
  const parsed =
    Number.parseInt(
      value,
      10,
    );

  if (
    !Number.isInteger(
      parsed,
    ) ||
    parsed < 1
  ) {
    return 1;
  }

  return parsed;
}

function searchHref(
  query: string,
  page: number,
) {
  const params =
    new URLSearchParams();

  params.set(
    "q",
    query,
  );

  if (
    page > 1
  ) {
    params.set(
      "page",
      String(
        page,
      ),
    );
  }

  return `/search?${params.toString()}`;
}

async function runSearch(
  query: string,
  page: number,
) {
  const params =
    new URLSearchParams({
      q: query,

      page:
        String(
          page,
        ),

      limit:
        String(
          SEARCH_LIMIT,
        ),
    });

  return commerceFetch<SearchResponse>(
    `/api/v1/search/products?${params.toString()}`,
    {
      cache: "no-store",
    },
  );
}

export default async function SearchPage({
  searchParams,
}: SearchPageProps) {
  const params =
    await searchParams;

  const query =
    firstValue(
      params.q,
    ).trim();

  const page =
    parsePage(
      firstValue(
        params.page,
      ),
    );

  let result:
    | SearchResponse
    | null = null;

  let searchFailed =
    false;

  if (query) {
    try {
      result =
        await runSearch(
          query,
          page,
        );
    } catch {
      searchFailed =
        true;
    }
  }

  const products =
    result?.data ??
    [];

  const meta =
    result?.meta ??
    null;

  const resultCount =
    meta?.total ??
    0;

  const hasPrevious =
    Boolean(
      meta &&
        meta.page >
          1,
    );

  const hasNext =
    Boolean(
      meta &&
        meta.page <
          meta.total_pages,
    );

  const showingSimilar =
    Boolean(
      meta?.showing_similar,
    );

  return (
    <main className="catalog-page search-page">
      <div className="site-container">
        <header className="catalog-page__header">
          <div>
            <p className="catalog-page__eyebrow">
              Product discovery
            </p>

            <h1>
              {query
                ? `Results for “${query}”`
                : "Search the catalog"}
            </h1>

            <p className="catalog-page__description">
              {query
                ? `${resultCount.toLocaleString()} ${
                    resultCount ===
                    1
                      ? "product"
                      : "products"
                  }`
                : "Use the search field above to find products, brands, categories or SKUs."}
            </p>
          </div>
        </header>

        {searchFailed ? (
          <section
            className="catalog-empty-state"
            aria-live="polite"
          >
            <span className="catalog-empty-state__eyebrow">
              Search unavailable
            </span>

            <h2>
              We could not load
              search results.
            </h2>

            <p>
              Please try your
              search again.
            </p>

            <Link href="/">
              Back to home
            </Link>
          </section>
        ) : null}

        {!searchFailed &&
        !query ? (
          <section className="catalog-empty-state">
            <span className="catalog-empty-state__eyebrow">
              Start typing
            </span>

            <h2>
              Find what you
              need faster.
            </h2>

            <p>
              Use the main
              search field in
              the header.
              Product,
              category and
              brand
              suggestions
              will appear as
              you type.
            </p>
          </section>
        ) : null}

        {!searchFailed &&
        query &&
        showingSimilar ? (
          <div
            className="catalog-search-notice"
            role="status"
          >
            <strong>
              Similar results
            </strong>

            <span>
              We could not
              find enough
              direct matches
              for “{query}”,
              so related
              catalog
              products are
              shown too.
            </span>
          </div>
        ) : null}

        {!searchFailed &&
        query &&
        products.length ===
          0 ? (
          <section className="catalog-empty-state">
            <span className="catalog-empty-state__eyebrow">
              No matches
            </span>

            <h2>
              No products
              found for
              “{query}”.
            </h2>

            <p>
              Try another
              product name,
              brand,
              category, SKU
              or a broader
              description.
            </p>

            <Link href="/">
              Browse all
              products
            </Link>
          </section>
        ) : null}

        {!searchFailed &&
        products.length >
          0 ? (
          <>
            <section
              className="catalog-results"
              aria-label="Search results"
            >
              <div className="product-grid product-grid--dense">
                {products.map(
                  (
                    product,
                    index,
                  ) => (
                    <ProductCard
                      key={
                        product.id
                      }
                      product={
                        product
                      }
                      eager={
                        page ===
                          1 &&
                        index <
                          6
                      }
                    />
                  ),
                )}
              </div>
            </section>

            {hasPrevious ||
            hasNext ? (
              <nav
                className="catalog-pagination"
                aria-label="Search result pages"
              >
                <div>
                  {hasPrevious ? (
                    <Link
                      href={searchHref(
                        query,
                        page -
                          1,
                      )}
                      rel="prev"
                    >
                      <span
                        aria-hidden="true"
                      >
                        ←
                      </span>

                      Previous
                    </Link>
                  ) : null}
                </div>

                <span className="catalog-pagination__status">
                  Page{" "}

                  <strong>
                    {meta?.page ??
                      page}
                  </strong>

                  {meta &&
                  meta.total_pages >
                    0 ? (
                    <>
                      {" "}
                      of{" "}

                      <strong>
                        {
                          meta.total_pages
                        }
                      </strong>
                    </>
                  ) : null}
                </span>

                <div>
                  {hasNext ? (
                    <Link
                      href={searchHref(
                        query,
                        page +
                          1,
                      )}
                      rel="next"
                    >
                      Next

                      <span
                        aria-hidden="true"
                      >
                        →
                      </span>
                    </Link>
                  ) : null}
                </div>
              </nav>
            ) : null}
          </>
        ) : null}
      </div>
    </main>
  );
}