"use client";

import Link from "next/link";
import {
  type FormEvent,
  useEffect,
  useState,
} from "react";
import {
  usePathname,
  useRouter,
} from "next/navigation";

import { Icon } from "@/components/ui/Icon";

import styles from "../css/CategoryPage.module.css";

export type CategorySort =
  | "recommended"
  | "featured"
  | "newest"
  | "price_asc"
  | "price_desc"
  | "name_asc"
  | "name_desc";

type Props = {
  total: number;
  page: number;
  totalPages: number;
  sort: CategorySort;
  stock: "" | "true" | "false";
  minPrice: string;
  maxPrice: string;
  subcategory: string;
};

const SORT_OPTIONS: Array<{
  value: CategorySort;
  label: string;
}> = [
  {
    value: "recommended",
    label: "Recommended",
  },
  {
    value: "featured",
    label: "Featured",
  },
  {
    value: "newest",
    label: "Newest",
  },
  {
    value: "price_asc",
    label: "Price: low to high",
  },
  {
    value: "price_desc",
    label: "Price: high to low",
  },
  {
    value: "name_asc",
    label: "Name: A–Z",
  },
  {
    value: "name_desc",
    label: "Name: Z–A",
  },
];

function validPrice(value: string) {
  if (!value) {
    return true;
  }

  const number = Number(value);

  return (
    Number.isFinite(number) &&
    number >= 0
  );
}

export function CategoryControls({
  total,
  page,
  totalPages,
  sort,
  stock,
  minPrice,
  maxPrice,
  subcategory,
}: Props) {
  const pathname = usePathname();
  const router = useRouter();

  const [filtersOpen, setFiltersOpen] =
    useState(false);

  const [filterError, setFilterError] =
    useState("");

  const filterCount =
    (stock ? 1 : 0) +
    (minPrice ? 1 : 0) +
    (maxPrice ? 1 : 0);

  useEffect(() => {
    if (!filtersOpen) {
      return;
    }

    function onKeyDown(event: KeyboardEvent) {
      if (event.key === "Escape") {
        setFiltersOpen(false);
      }
    }

    window.addEventListener(
      "keydown",
      onKeyDown,
    );

    return () => {
      window.removeEventListener(
        "keydown",
        onKeyDown,
      );
    };
  }, [filtersOpen]);

  function baseParams() {
    const params = new URLSearchParams();

    if (stock) {
      params.set("stock", stock);
    }

    if (minPrice) {
      params.set(
        "min_price",
        minPrice,
      );
    }

    if (maxPrice) {
      params.set(
        "max_price",
        maxPrice,
      );
    }

    if (sort !== "recommended") {
      params.set("sort", sort);
    }

    if (subcategory) {
      params.set(
        "subcategory",
        subcategory,
      );
    }

    return params;
  }

  function navigate(
    params: URLSearchParams,
  ) {
    const query = params.toString();

    router.push(
      query
        ? `${pathname}?${query}`
        : pathname,
    );
  }

  function submitFilters(
    event: FormEvent<HTMLFormElement>,
  ) {
    event.preventDefault();

    const formData = new FormData(
      event.currentTarget,
    );

    const nextStock = String(
      formData.get("stock") ?? "",
    ).trim();

    const nextMin = String(
      formData.get("min_price") ?? "",
    ).trim();

    const nextMax = String(
      formData.get("max_price") ?? "",
    ).trim();

    if (
      !validPrice(nextMin) ||
      !validPrice(nextMax)
    ) {
      setFilterError(
        "Enter a valid non-negative price.",
      );

      return;
    }

    if (
      nextMin &&
      nextMax &&
      Number(nextMin) > Number(nextMax)
    ) {
      setFilterError(
        "Minimum price cannot be higher than maximum price.",
      );

      return;
    }

    const params = new URLSearchParams();

    if (
      nextStock === "true" ||
      nextStock === "false"
    ) {
      params.set(
        "stock",
        nextStock,
      );
    }

    if (nextMin) {
      params.set(
        "min_price",
        nextMin,
      );
    }

    if (nextMax) {
      params.set(
        "max_price",
        nextMax,
      );
    }

    if (sort !== "recommended") {
      params.set("sort", sort);
    }

    if (subcategory) {
      params.set(
        "subcategory",
        subcategory,
      );
    }

    setFilterError("");
    setFiltersOpen(false);
    navigate(params);
  }

  function changeSort(
    nextSort: CategorySort,
  ) {
    const params = baseParams();

    params.delete("page");

    if (
      nextSort === "recommended"
    ) {
      params.delete("sort");
    } else {
      params.set(
        "sort",
        nextSort,
      );
    }

    navigate(params);
  }

  return (
    <>
      <div className={styles.controlsRow}>
      <div
        className={styles.resultsSummary}
        aria-live="polite"
      >
        <strong>
          {total.toLocaleString()}
        </strong>

        <span>
          {total === 1
            ? "product"
            : "products"}
        </span>

        {totalPages > 1 ? (
          <small>
            Page {page.toLocaleString()} of{" "}
            {totalPages.toLocaleString()}
          </small>
        ) : null}
      </div>

      <div className={styles.controlActions}>
        <button
          type="button"
          className={`${styles.controlButton} ${
            filterCount > 0
              ? styles.controlButtonActive
              : ""
          }`}
          aria-expanded={filtersOpen}
          aria-controls="category-filter-panel"
          onClick={() => {
            setFilterError("");
            setFiltersOpen(
              (current) => !current,
            );
          }}
        >
          <Icon
            name="filter"
            size={15}
          />

          <span>Filters</span>

          {filterCount > 0 ? (
            <b>{filterCount}</b>
          ) : null}
        </button>

        <label className={styles.sortControl}>
          <Icon
            name="sort"
            size={15}
          />

          <span className={styles.srOnly}>
            Sort products
          </span>

          <select
            value={sort}
            aria-label="Sort products"
            onChange={(event) =>
              changeSort(
                event.target
                  .value as CategorySort,
              )
            }
          >
            {SORT_OPTIONS.map(
              (option) => (
                <option
                  key={option.value}
                  value={option.value}
                >
                  {option.label}
                </option>
              ),
            )}
          </select>
        </label>
      </div>
    </div>

      {filtersOpen ? (
        <div
          className={styles.filterOverlay}
        >
          <button
            type="button"
            className={styles.filterBackdrop}
            aria-label="Close filters"
            onClick={() =>
              setFiltersOpen(false)
            }
          />

          <div
            id="category-filter-panel"
            className={styles.filterPanel}
            role="dialog"
            aria-modal="true"
            aria-labelledby="category-filter-title"
          >
            <div
              className={
                styles.filterPanelHeader
              }
            >
              <div>
                <span>Refine catalog</span>

                <h2 id="category-filter-title">
                  Filters
                </h2>
              </div>

              <button
                type="button"
                className={styles.closeButton}
                aria-label="Close filters"
                autoFocus
                onClick={() =>
                  setFiltersOpen(false)
                }
              >
                <Icon
                  name="close"
                  size={16}
                />
              </button>
            </div>

            <form
              className={styles.filterForm}
              onSubmit={submitFilters}
            >
              <fieldset>
                <legend>Availability</legend>

                <label>
                  <input
                    type="radio"
                    name="stock"
                    value=""
                    defaultChecked={!stock}
                  />

                  <span>All products</span>
                </label>

                <label>
                  <input
                    type="radio"
                    name="stock"
                    value="true"
                    defaultChecked={
                      stock === "true"
                    }
                  />

                  <span>In stock</span>
                </label>

                <label>
                  <input
                    type="radio"
                    name="stock"
                    value="false"
                    defaultChecked={
                      stock === "false"
                    }
                  />

                  <span>Out of stock</span>
                </label>
              </fieldset>

              <fieldset>
                <legend>Price range</legend>

                <div className={styles.priceInputs}>
                  <label>
                    <span>Minimum</span>

                    <input
                      type="number"
                      name="min_price"
                      inputMode="decimal"
                      min="0"
                      step="0.01"
                      defaultValue={minPrice}
                      placeholder="0"
                    />
                  </label>

                  <span aria-hidden="true">
                    —
                  </span>

                  <label>
                    <span>Maximum</span>

                    <input
                      type="number"
                      name="max_price"
                      inputMode="decimal"
                      min="0"
                      step="0.01"
                      defaultValue={maxPrice}
                      placeholder="Any"
                    />
                  </label>
                </div>

                <small>
                  Enter prices in the storefront
                  currency.
                </small>
              </fieldset>

              {filterError ? (
                <p
                  className={styles.filterError}
                  role="alert"
                >
                  {filterError}
                </p>
              ) : null}

              <div className={styles.filterFooter}>
                <Link
                  href={
                    subcategory
                      ? `${pathname}?subcategory=${encodeURIComponent(
                          subcategory,
                        )}`
                      : pathname
                  }
                  className={styles.clearFilters}
                  onClick={() =>
                    setFiltersOpen(false)
                  }
                >
                  Clear all
                </Link>

                <button
                  type="submit"
                  className={styles.applyFilters}
                >
                  Show products
                </button>
              </div>
            </form>
          </div>
        </div>
      ) : null}
    </>
  );
}
