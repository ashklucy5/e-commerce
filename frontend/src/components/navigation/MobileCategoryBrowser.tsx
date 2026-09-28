"use client";

import Link from "next/link";
import { useMemo, useState } from "react";

import { Icon } from "@/components/ui/Icon";

import type {
  CategoryNode,
} from "@/lib/api/contracts/catalog";

type Props = {
  categories: CategoryNode[];
};

export function MobileCategoryBrowser({
  categories,
}: Props) {
  const [activeSlug, setActiveSlug] =
    useState(
      categories[0]?.slug ?? "",
    );

  const activeCategory = useMemo(
    () =>
      categories.find(
        (category) =>
          category.slug === activeSlug,
      ) ?? categories[0],
    [activeSlug, categories],
  );

  if (
    categories.length === 0 ||
    !activeCategory
  ) {
    return (
      <div className="category-empty-state">
        Categories are temporarily
        unavailable.
      </div>
    );
  }

  return (
    <section className="mobile-category-browser">
      <div
        className="mobile-category-tabs"
        role="tablist"
        aria-label="Product categories"
      >
        {categories.map((category) => {
          const active =
            category.slug ===
            activeCategory.slug;

          return (
            <button
              key={category.id}
              type="button"
              role="tab"
              aria-selected={active}
              className={[
                "mobile-category-tab",
                active
                  ? "mobile-category-tab--active"
                  : "",
              ]
                .filter(Boolean)
                .join(" ")}
              onClick={() =>
                setActiveSlug(
                  category.slug,
                )
              }
            >
              {category.name}
            </button>
          );
        })}
      </div>

      <div className="mobile-subcategory-section">
        <div className="mobile-subcategory-heading">
          <div>
            <p className="mobile-subcategory-eyebrow">
              Explore
            </p>

            <h1>
              {activeCategory.name}
            </h1>
          </div>

          <Link
            href={`/category/${activeCategory.slug}`}
            className="mobile-subcategory-all"
          >
            View all
          </Link>
        </div>

        <div className="mobile-subcategory-grid">
          {activeCategory.children.map(
            (subcategory) => (
              <Link
                key={subcategory.id}
                href={`/category/${subcategory.slug}`}
                className="mobile-subcategory-item"
              >
                <div className="mobile-subcategory-icon">
                  <Icon
                    name="categories"
                    size={26}
                  />
                </div>

                <span>
                  {subcategory.name}
                </span>
              </Link>
            ),
          )}
        </div>
      </div>
    </section>
  );
}