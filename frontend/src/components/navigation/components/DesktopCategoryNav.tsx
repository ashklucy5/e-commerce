"use client";

import Link from "next/link";

import {
  useMemo,
  useState,
} from "react";

import {
  Icon,
} from "@/components/ui/Icon";

import type {
  CategoryNode,
} from "@/lib/api/contracts/catalog";

import styles from "../css/DesktopCategoryNav.module.css";

type Props = {
  categories: CategoryNode[];
};

export function DesktopCategoryNav({
  categories,
}: Props) {
  const [
    activeSlug,
    setActiveSlug,
  ] = useState(
    categories[0]
      ?.slug ?? "",
  );

  const activeCategory =
    useMemo(
      () =>
        categories.find(
          (item) =>
            item.slug ===
            activeSlug,
        ) ??
        categories[0],
      [
        activeSlug,
        categories,
      ],
    );

  return (
    <nav
      className={
        styles.nav
      }
      aria-label="Product categories"
    >
      <div
        className={`site-container ${styles.inner}`}
      >
        <div
          className={
            styles.megaMenu
          }
        >
          <button
            type="button"
            className={
              styles.allButton
            }
          >
            <Icon
              name="categories"
              size={16}
            />

            All categories

            <Icon
              name="chevronRight"
              size={12}
              className={
                styles.chevron
              }
            />
          </button>

          {activeCategory ? (
            <div
              className={
                styles.panel
              }
            >
              <div
                className={
                  styles.parents
                }
              >
                {categories.map(
                  (
                    category,
                  ) => (
                    <Link
                      key={
                        category.id
                      }
                      href={`/category/${category.slug}`}
                      className={
                        category.slug ===
                        activeCategory.slug
                          ? styles.activeParent
                          : undefined
                      }
                      onMouseEnter={() =>
                        setActiveSlug(
                          category.slug,
                        )
                      }
                      onFocus={() =>
                        setActiveSlug(
                          category.slug,
                        )
                      }
                    >
                      <span>
                        {
                          category.name
                        }
                      </span>

                      <Icon
                        name="chevronRight"
                        size={
                          13
                        }
                      />
                    </Link>
                  ),
                )}
              </div>

              <div
                className={
                  styles.content
                }
              >
                <div
                  className={
                    styles.heading
                  }
                >
                  <div>
                    <span>
                      Explore
                    </span>

                    <strong>
                      {
                        activeCategory.name
                      }
                    </strong>
                  </div>

                  <Link
                    href={`/category/${activeCategory.slug}`}
                  >
                    View all
                  </Link>
                </div>

                <div
                  className={
                    styles.children
                  }
                >
                  {activeCategory
                    .children
                    .length >
                  0
                    ? activeCategory.children.map(
                        (
                          child,
                        ) => (
                          <Link
                            key={
                              child.id
                            }
                            href={`/category/${child.slug}`}
                          >
                            <span>
                              {
                                child.name
                              }
                            </span>

                            <small>
                              Browse
                              products
                            </small>
                          </Link>
                        ),
                      )
                    : (
                      <Link
                        href={`/category/${activeCategory.slug}`}
                      >
                        <span>
                          Browse{" "}
                          {
                            activeCategory.name
                          }
                        </span>

                        <small>
                          View
                          available
                          products
                        </small>
                      </Link>
                    )}
                </div>
              </div>
            </div>
          ) : null}
        </div>

        <div
          className={
            styles.links
          }
        >
          <Link
            href="/search?sort=featured"
            className={
              styles.hotLink
            }
          >
            Flash deals
          </Link>

          <Link href="/search">
            Trending
          </Link>

          <Link href="/search?sort=newest">
            New arrivals
          </Link>

          {categories
            .slice(0, 5)
            .map(
              (
                category,
              ) => (
                <Link
                  key={
                    category.id
                  }
                  href={`/category/${category.slug}`}
                >
                  {
                    category.name
                  }
                </Link>
              ),
            )}

          <Link
            href="/categories"
            className={
              styles.moreLink
            }
          >
            More
          </Link>
        </div>
      </div>
    </nav>
  );
}