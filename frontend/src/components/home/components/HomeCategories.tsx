"use client";

import Link from "next/link";

import {
  useMemo,
  useState,
} from "react";

import type {
  CategoryNode,
} from "@/lib/api/contracts/catalog";

import styles from "../css/HomeCategories.module.css";

type Props = {
  categories: CategoryNode[];
};

function getInitials(
  value: string,
) {
  return value
    .split(/\s+/)
    .filter(Boolean)
    .map(
      part =>
        part.charAt(0),
    )
    .join("")
    .slice(0, 2)
    .toUpperCase();
}

export function HomeCategories({
  categories,
}: Props) {
  const [
    activeSlug,
    setActiveSlug,
  ] = useState(
    categories[0]?.slug ?? "",
  );

  const activeCategory =
    useMemo(
      () =>
        categories.find(
          category =>
            category.slug ===
            activeSlug,
        ) ??
        categories[0],
      [
        activeSlug,
        categories,
      ],
    );

  if (
    categories.length === 0 ||
    !activeCategory
  ) {
    return null;
  }

  const mobileChildren =
    activeCategory.children.length >
    0
      ? activeCategory.children
      : [activeCategory];

  return (
    <section
      className={
        styles.categories
      }
      aria-labelledby="home-categories-title"
    >
      <div
        className={
          styles.sectionHeading
        }
      >
        <div>
          <span>
            Browse faster
          </span>

          <h2 id="home-categories-title">
            Categories
          </h2>
        </div>

        <Link href="/categories">
          All categories

          <span
            aria-hidden="true"
          >
            →
          </span>
        </Link>
      </div>

      <div
        className={
          styles.desktopCategoryView
        }
      >
        <div
          className={
            styles.categoryGrid
          }
        >
          {categories
            .slice(
              0,
              8,
            )
            .map(
              (
                category,
                index,
              ) => (
                <Link
                  key={
                    category.id
                  }
                  href={`/category/${category.slug}`}
                  className={
                    styles.categoryCard
                  }
                >
                  <span
                    className={
                      styles.categoryNumber
                    }
                    aria-hidden="true"
                  >
                    {String(
                      index + 1,
                    ).padStart(
                      2,
                      "0",
                    )}
                  </span>

                  <span
                    className={
                      styles.categoryMonogram
                    }
                    aria-hidden="true"
                  >
                    {getInitials(
                      category.name,
                    )}
                  </span>

                  <span
                    className={
                      styles.categoryText
                    }
                  >
                    <strong>
                      {category.name}
                    </strong>

                    <small>
                      {category.children
                        .length >
                      0
                        ? `${category.children.length} ${
                            category.children
                              .length ===
                            1
                              ? "section"
                              : "sections"
                          }`
                        : "Browse products"}
                    </small>
                  </span>

                  <span
                    className={
                      styles.categoryArrow
                    }
                    aria-hidden="true"
                  >
                    →
                  </span>
                </Link>
              ),
            )}
        </div>
      </div>

      <div
        className={
          styles.mobileCategoryView
        }
      >
        <div
          className={
            styles.mobileCategoryTabs
          }
          role="tablist"
          aria-label="Product categories"
        >
          {categories.map(
            category => {
              const active =
                category.slug ===
                activeCategory.slug;

              return (
                <button
                  key={
                    category.id
                  }
                  id={`home-category-tab-${category.slug}`}
                  type="button"
                  role="tab"
                  aria-selected={
                    active
                  }
                  aria-controls={`home-category-panel-${category.slug}`}
                  className={
                    active
                      ? styles.mobileCategoryTabActive
                      : styles.mobileCategoryTab
                  }
                  onClick={() =>
                    setActiveSlug(
                      category.slug,
                    )
                  }
                >
                  {category.name}
                </button>
              );
            },
          )}
        </div>

        <div
          id={`home-category-panel-${activeCategory.slug}`}
          role="tabpanel"
          aria-labelledby={`home-category-tab-${activeCategory.slug}`}
          className={
            styles.mobileSubcategoryPanel
          }
        >
          <div
            className={
              styles.mobileSubcategoryHeading
            }
          >
            <strong>
              {
                activeCategory.name
              }
            </strong>

            <Link
              href={`/category/${activeCategory.slug}`}
            >
              View all
            </Link>
          </div>

          <div
            className={
              styles.mobileSubcategoryRail
            }
          >
            {mobileChildren.map(
              child => (
                <Link
                  key={
                    child.id
                  }
                  href={`/category/${child.slug}`}
                  className={
                    styles.mobileSubcategoryCard
                  }
                >
                  <span
                    className={
                      styles.mobileSubcategoryMonogram
                    }
                    aria-hidden="true"
                  >
                    {getInitials(
                      child.name,
                    )}
                  </span>

                  <strong>
                    {child.name}
                  </strong>
                </Link>
              ),
            )}
          </div>
        </div>
      </div>
    </section>
  );
}