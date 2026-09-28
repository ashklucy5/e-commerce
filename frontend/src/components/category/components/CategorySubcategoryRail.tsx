"use client";

import Link from "next/link";

import type { CategoryNode } from "@/lib/api/contracts/catalog";

import styles from "../css/CategoryPage.module.css";

type Props = {
  categoryName: string;
  categorySlug: string;
  subcategories: CategoryNode[];
  selectedSubcategorySlug: string;
  sort: string;
  stock: "" | "true" | "false";
  minPrice: string;
  maxPrice: string;
};

type Level = {
  parentName: string;
  parentSlug: string;
  children: CategoryNode[];
};

function findCategoryPath(
  categories: CategoryNode[],
  slug: string,
  ancestors: CategoryNode[] = [],
): CategoryNode[] | null {
  for (const category of categories) {
    const path = [...ancestors, category];

    if (category.slug === slug) {
      return path;
    }

    const childPath = findCategoryPath(
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

export function CategorySubcategoryRail({
  categoryName,
  categorySlug,
  subcategories,
  selectedSubcategorySlug,
  sort,
  stock,
  minPrice,
  maxPrice,
}: Props) {
  const selectedPath = selectedSubcategorySlug
    ? findCategoryPath(subcategories, selectedSubcategorySlug) ?? []
    : [];

  const selectedPathSlugs = new Set(
    selectedPath.map((category) => category.slug),
  );

  const levels: Level[] = [
    {
      parentName: categoryName,
      parentSlug: "",
      children: subcategories,
    },
  ];

  for (const selectedCategory of selectedPath) {
    if ((selectedCategory.children ?? []).length === 0) {
      continue;
    }

    levels.push({
      parentName: selectedCategory.name,
      parentSlug: selectedCategory.slug,
      children: selectedCategory.children,
    });
  }

  if (subcategories.length === 0) {
    return null;
  }

  function hrefForCategory(slug = "") {
    const params = new URLSearchParams();

    if (slug) {
      params.set("subcategory", slug);
    }

    if (sort && sort !== "recommended") {
      params.set("sort", sort);
    }

    if (stock) {
      params.set("stock", stock);
    }

    if (minPrice) {
      params.set("min_price", minPrice);
    }

    if (maxPrice) {
      params.set("max_price", maxPrice);
    }

    const query = params.toString();
    const pathname = `/category/${encodeURIComponent(categorySlug)}`;

    return query ? `${pathname}?${query}` : pathname;
  }

  return (
    <section
      className={styles.categoryLevels}
      aria-label={`${categoryName} category filters`}
    >
      {levels.map((level, levelIndex) => {
        const allActive =
          level.parentSlug === ""
            ? !selectedSubcategorySlug
            : selectedSubcategorySlug === level.parentSlug;

        const allHref = hrefForCategory(level.parentSlug);

        return (
          <div
            key={`${level.parentSlug || categorySlug}-${levelIndex}`}
            className={styles.subcategoryLevel}
          >
            {levelIndex > 0 ? (
              <div className={styles.subcategoryLevelHeader}>
                <span>{level.parentName}</span>
                <small>
                  {level.children.length.toLocaleString()} direct{" "}
                  {level.children.length === 1
                    ? "subcategory"
                    : "subcategories"}
                </small>
              </div>
            ) : null}

            <nav
              className={styles.subcategoryRail}
              aria-label={`${level.parentName} subcategories`}
            >
              <Link
                href={allHref}
                className={`${styles.subcategoryPill} ${
                  allActive ? styles.subcategoryActive : ""
                }`}
                aria-current={allActive ? "page" : undefined}
              >
                <span>All {level.parentName}</span>
              </Link>

              {level.children.map((child) => {
                const active = child.slug === selectedSubcategorySlug;
                const ancestor =
                  !active && selectedPathSlugs.has(child.slug);
                const childCount = (child.children ?? []).length;
                const href = hrefForCategory(child.slug);

                return (
                  <Link
                    key={child.id}
                    href={href}
                    className={[
                      styles.subcategoryPill,
                      active ? styles.subcategoryActive : "",
                      ancestor ? styles.subcategoryAncestor : "",
                    ]
                      .filter(Boolean)
                      .join(" ")}
                    aria-current={active ? "page" : undefined}
                    aria-label={
                      childCount > 0
                        ? `${child.name}, ${childCount} ${
                            childCount === 1
                              ? "subcategory"
                              : "subcategories"
                          }`
                        : child.name
                    }
                  >
                    <span>{child.name}</span>
                  </Link>
                );
              })}
            </nav>
          </div>
        );
      })}
    </section>
  );
}
