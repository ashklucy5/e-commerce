"use client";

import Link from "next/link";

import {
  type FormEvent,
  useEffect,
  useState,
} from "react";

import AdminPageHeader from "@/components/admin/layout/components/AdminPageHeader";

import {
  adminFetch,
  AdminRequestError,
} from "@/lib/admin/api";

import type {
  AdminCatalogPaginationMeta,
  AdminCatalogProductDetail,
  AdminCatalogProductListItem,
  AdminCatalogProductListResponse,
  AdminProductStatus,
} from "@/lib/admin/catalog-types";

import ProductDetailDrawer from "./ProductDetailDrawer";

import styles from "../css/ProductsManagement.module.css";

const PAGE_LIMIT = 50;

type ProductStatusFilter =
  | ""
  | AdminProductStatus;

type ProductsManagementProps = {
  portal: string;
};

function errorMessage(
  value: unknown,
): string {
  if (
    value instanceof
    AdminRequestError
  ) {
    return value.message;
  }

  if (value instanceof Error) {
    return value.message;
  }

  return "Products could not be loaded.";
}

function normalizeListResponse(
  response:
    AdminCatalogProductListResponse,
): {
  items:
    AdminCatalogProductListItem[];
  meta:
    AdminCatalogPaginationMeta | null;
} {
  if (
    Array.isArray(
      response.data,
    )
  ) {
    return {
      items: response.data,
      meta: null,
    };
  }

  const data =
    response.data;

  return {
    items:
      data.items ??
      data.Items ??
      [],

    meta:
      data.meta ??
      data.Meta ??
      null,
  };
}

function metaNumber(
  meta:
    AdminCatalogPaginationMeta | null,
  lowercase: string,
  uppercase: string,
): number | null {
  if (!meta) {
    return null;
  }

  const value =
    meta[lowercase] ??
    meta[uppercase];

  return typeof value ===
    "number"
    ? value
    : null;
}

function statusClass(
  status: string,
): string {
  switch (status) {
    case "active":
      return styles.statusActive;

    case "archived":
      return styles.statusArchived;

    default:
      return styles.statusDraft;
  }
}

export default function ProductsManagement({
  portal,
}: ProductsManagementProps) {
  const [
    products,
    setProducts,
  ] = useState<
    AdminCatalogProductListItem[]
  >([]);

  const [
    meta,
    setMeta,
  ] =
    useState<AdminCatalogPaginationMeta | null>(
      null,
    );

  const [
    query,
    setQuery,
  ] = useState("");

  const [
    status,
    setStatus,
  ] =
    useState<ProductStatusFilter>(
      "",
    );

  const [
    appliedQuery,
    setAppliedQuery,
  ] = useState("");

  const [
    appliedStatus,
    setAppliedStatus,
  ] =
    useState<ProductStatusFilter>(
      "",
    );

  const [
    offset,
    setOffset,
  ] = useState(0);

  const [
    loading,
    setLoading,
  ] = useState(true);

  const [
    error,
    setError,
  ] = useState("");

  const [
    selectedProductId,
    setSelectedProductId,
  ] = useState<
    string | null
  >(null);

  useEffect(() => {
    let cancelled = false;

    const params =
      new URLSearchParams({
        limit:
          String(
            PAGE_LIMIT,
          ),

        offset: "0",
      });

    adminFetch<AdminCatalogProductListResponse>(
      `/products?${params.toString()}`,
    )
      .then(
        (response) => {
          if (cancelled) {
            return;
          }

          const result =
            normalizeListResponse(
              response,
            );

          setProducts(
            result.items,
          );

          setMeta(
            result.meta,
          );

          setLoading(false);
        },
      )
      .catch(
        (value: unknown) => {
          if (cancelled) {
            return;
          }

          setError(
            errorMessage(
              value,
            ),
          );

          setLoading(false);
        },
      );

    return () => {
      cancelled = true;
    };
  }, []);

  async function loadProducts(
    nextQuery: string,
    nextStatus:
      ProductStatusFilter,
    nextOffset: number,
  ) {
    const normalizedQuery =
      nextQuery.trim();

    setLoading(true);
    setError("");

    const params =
      new URLSearchParams({
        limit:
          String(
            PAGE_LIMIT,
          ),

        offset:
          String(
            nextOffset,
          ),
      });

    if (normalizedQuery) {
      params.set(
        "q",
        normalizedQuery,
      );
    }

    if (nextStatus) {
      params.set(
        "status",
        nextStatus,
      );
    }

    try {
      const response =
        await adminFetch<AdminCatalogProductListResponse>(
          `/products?${params.toString()}`,
        );

      const result =
        normalizeListResponse(
          response,
        );

      setProducts(
        result.items,
      );

      setMeta(
        result.meta,
      );

      setAppliedQuery(
        normalizedQuery,
      );

      setAppliedStatus(
        nextStatus,
      );

      setOffset(
        nextOffset,
      );
    } catch (value) {
      setError(
        errorMessage(
          value,
        ),
      );
    } finally {
      setLoading(false);
    }
  }

  function handleSearch(
    event:
      FormEvent<HTMLFormElement>,
  ) {
    event.preventDefault();

    void loadProducts(
      query,
      status,
      0,
    );
  }

  function handleStatusChange(
    nextStatus:
      ProductStatusFilter,
  ) {
    setStatus(
      nextStatus,
    );

    void loadProducts(
      query,
      nextStatus,
      0,
    );
  }

  function resetFilters() {
    setQuery("");
    setStatus("");

    void loadProducts(
      "",
      "",
      0,
    );
  }

  function refresh() {
    void loadProducts(
      appliedQuery,
      appliedStatus,
      offset,
    );
  }

  function handleProductUpdated(
    updated:
      AdminCatalogProductDetail,
  ) {
    setProducts(
      (current) =>
        current.map(
          (product) =>
            product.id ===
            updated.id
              ? {
                  ...product,

                  category_id:
                    updated.category_id,

                  category_name:
                    updated.category_name,

                  name:
                    updated.name,

                  slug:
                    updated.slug,

                  brand:
                    updated.brand,

                  status:
                    updated.status,

                  is_featured:
                    updated.is_featured,

                  variant_count:
                    updated.variant_count,

                  active_variant_count:
                    updated.active_variant_count,

                  available_stock:
                    updated.available_stock,

                  updated_at:
                    updated.updated_at,
                }
              : product,
        ),
    );
  }

  const total =
    metaNumber(
      meta,
      "total",
      "Total",
    );

  const firstVisible =
    products.length > 0
      ? offset + 1
      : 0;

  const lastVisible =
    offset +
    products.length;

  const canPrevious =
    offset > 0 &&
    !loading;

  const canNext =
    !loading &&
    (
      total !== null
        ? lastVisible <
          total
        : products.length ===
          PAGE_LIMIT
    );

  return (
    <>
      <div
        className={
          styles.page
        }
      >
        <div
          className={
            styles.pageHeader
          }
        >
          <AdminPageHeader
            eyebrow="Catalog operations"
            title="Products"
            description="Stage a simple or advanced XLSX workbook, inspect every planned create/update/skip action, then explicitly apply only a clean validated batch."
          />

          <div
            className={
              styles.headerActions
            }
          >
            <Link
  href={`/${portal}/products/import`}
  className={
    styles.importButton
  }
>
  Import Excel

  <span>
    Stage & preview
  </span>
</Link>



            <Link
              href={`/${portal}/products/new`}
              className={
                styles.primaryButton
              }
            >
              <span
                aria-hidden="true"
              >
                +
              </span>

              Add product
            </Link>
          </div>
        </div>

        <section
          className={
            styles.controlPanel
          }
        >
          <form
            className={
              styles.searchForm
            }
            onSubmit={
              handleSearch
            }
          >
            <label
              className={
                styles.searchField
              }
            >
              <span>
                Search catalog
              </span>

              <div
                className={
                  styles.searchInputWrap
                }
              >
                <span
                  aria-hidden="true"
                  className={
                    styles.searchIcon
                  }
                >
                  ⌕
                </span>

                <input
                  type="search"
                  value={
                    query
                  }
                  onChange={(
                    event,
                  ) =>
                    setQuery(
                      event
                        .target
                        .value,
                    )
                  }
                  placeholder="Product name, product code, SKU…"
                />

                {query && (
                  <button
                    type="button"
                    className={
                      styles.clearSearch
                    }
                    aria-label="Clear search"
                    onClick={() =>
                      setQuery(
                        "",
                      )
                    }
                  >
                    ×
                  </button>
                )}
              </div>
            </label>

            <label
              className={
                styles.statusField
              }
            >
              <span>
                Status
              </span>

              <select
                value={
                  status
                }
                onChange={(
                  event,
                ) =>
                  handleStatusChange(
                    event
                      .target
                      .value as ProductStatusFilter,
                  )
                }
              >
                <option value="">
                  All statuses
                </option>

                <option value="active">
                  Active
                </option>

                <option value="draft">
                  Draft
                </option>

                <option value="archived">
                  Archived
                </option>
              </select>
            </label>

            <button
              type="submit"
              className={
                styles.searchButton
              }
              disabled={
                loading
              }
            >
              Search
            </button>

            {(query ||
              status ||
              appliedQuery ||
              appliedStatus) && (
              <button
                type="button"
                className={
                  styles.resetButton
                }
                onClick={
                  resetFilters
                }
                disabled={
                  loading
                }
              >
                Reset
              </button>
            )}
          </form>

          <div
            className={
              styles.resultMeta
            }
          >
            <span
              className={
                loading
                  ? styles.liveDotLoading
                  : styles.liveDot
              }
            />

            {loading ? (
              <span>
                Loading catalog…
              </span>
            ) : total !== null ? (
              <span>
                Showing{" "}
                <strong>
                  {firstVisible}
                  –
                  {lastVisible}
                </strong>{" "}
                of{" "}
                <strong>
                  {total}
                </strong>{" "}
                products
              </span>
            ) : (
              <span>
                <strong>
                  {
                    products.length
                  }
                </strong>{" "}
                products on this page
              </span>
            )}
          </div>
        </section>

        {error && (
          <div
            className={
              styles.errorPanel
            }
            role="alert"
          >
            <div>
              <strong>
                Catalog request failed
              </strong>

              <span>
                {error}
              </span>
            </div>

            <button
              type="button"
              onClick={
                refresh
              }
            >
              Retry
            </button>
          </div>
        )}

        <section
          className={
            styles.catalogPanel
          }
          aria-busy={
            loading
          }
        >
          <div
            className={
              styles.tableHeader
            }
          >
            <span>
              Product
            </span>

            <span>
              Category
            </span>

            <span>
              Status
            </span>

            <span>
              Variants
            </span>

            <span>
              Available
            </span>

            <span>
              Action
            </span>
          </div>

          {loading &&
          products.length === 0 ? (
            <div
              className={
                styles.loadingState
              }
            >
              <span
                className={
                  styles.loader
                }
              />

              <strong>
                Loading products
              </strong>

              <p>
                Reading the
                catalog from the
                commerce API.
              </p>
            </div>
          ) : products.length ===
            0 ? (
            <div
              className={
                styles.emptyState
              }
            >
              <span
                className={
                  styles.emptyIcon
                }
              >
                ◇
              </span>

              <strong>
                No products found
              </strong>

              <p>
                Try another search
                or remove the
                current filters.
              </p>

              {(appliedQuery ||
                appliedStatus) && (
                <button
                  type="button"
                  onClick={
                    resetFilters
                  }
                >
                  Clear filters
                </button>
              )}
            </div>
          ) : (
            <div
              className={
                styles.productRows
              }
            >
              {products.map(
                (product) => (
                  <button
                    key={
                      product.id
                    }
                    type="button"
                    className={
                      styles.productRow
                    }
                    onClick={() =>
                      setSelectedProductId(
                        product.id,
                      )
                    }
                    aria-label={`Open ${product.name}`}
                  >
                    <span
                      className={
                        styles.productIdentity
                      }
                    >
                      <span
                        className={
                          styles.productInitial
                        }
                        aria-hidden="true"
                      >
                        {product.name
                          .trim()
                          .charAt(
                            0,
                          )
                          .toUpperCase() ||
                          "P"}
                      </span>

                      <span
                        className={
                          styles.productText
                        }
                      >
                        <strong>
                          {
                            product.name
                          }
                        </strong>

                        <small>
                          {
                            product.product_code
                          }

                          {product.brand
                            ? ` · ${product.brand}`
                            : ""}
                        </small>
                      </span>

                      {product.is_featured && (
                        <span
                          className={
                            styles.featuredMark
                          }
                        >
                          Featured
                        </span>
                      )}
                    </span>

                    <span
                      className={
                        styles.cell
                      }
                      data-label="Category"
                    >
                      {
                        product.category_name
                      }
                    </span>

                    <span
                      className={
                        styles.cell
                      }
                      data-label="Status"
                    >
                      <span
                        className={`${styles.statusBadge} ${statusClass(
                          product.status,
                        )}`}
                      >
                        {
                          product.status
                        }
                      </span>
                    </span>

                    <span
                      className={
                        styles.cell
                      }
                      data-label="Variants"
                    >
                      <strong>
                        {
                          product.active_variant_count
                        }
                      </strong>

                      <small>
                        {" "}
                        /{" "}
                        {
                          product.variant_count
                        }{" "}
                        active
                      </small>
                    </span>

                    <span
                      className={
                        styles.stockCell
                      }
                      data-label="Available"
                    >
                      <strong>
                        {
                          product.available_stock
                        }
                      </strong>

                      <small>
                        units
                      </small>
                    </span>

                    <span
                      className={
                        styles.openCell
                      }
                    >
                      Inspect
                      <span
                        aria-hidden="true"
                      >
                        →
                      </span>
                    </span>
                  </button>
                ),
              )}
            </div>
          )}
        </section>

        <div
          className={
            styles.pagination
          }
        >
          <button
            type="button"
            disabled={
              !canPrevious
            }
            onClick={() =>
              void loadProducts(
                appliedQuery,
                appliedStatus,
                Math.max(
                  0,
                  offset -
                    PAGE_LIMIT,
                ),
              )
            }
          >
            ← Previous
          </button>

          <span>
            {products.length >
            0
              ? `${firstVisible}–${lastVisible}`
              : "0"}

            {total !== null
              ? ` of ${total}`
              : ""}
          </span>

          <button
            type="button"
            disabled={
              !canNext
            }
            onClick={() =>
              void loadProducts(
                appliedQuery,
                appliedStatus,
                offset +
                  PAGE_LIMIT,
              )
            }
          >
            Next →
          </button>
        </div>
      </div>

      {selectedProductId && (
        <ProductDetailDrawer
          key={
            selectedProductId
          }
          productId={
            selectedProductId
          }
          onClose={() =>
            setSelectedProductId(
              null,
            )
          }
          onProductUpdated={
            handleProductUpdated
          }
        />
      )}
    </>
  );
}