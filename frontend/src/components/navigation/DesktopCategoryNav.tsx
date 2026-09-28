"use client";

import Link from "next/link";
import { useMemo, useState } from "react";

import { Icon } from "@/components/ui/Icon";
import type { CategoryNode } from "@/lib/api/contracts/catalog";

type Props = {
  categories: CategoryNode[];
};

export function DesktopCategoryNav({ categories }: Props) {
  const [activeSlug, setActiveSlug] = useState(categories[0]?.slug ?? "");

  const activeCategory = useMemo(
    () => categories.find((item) => item.slug === activeSlug) ?? categories[0],
    [activeSlug, categories],
  );

  return (
    <nav className="desktop-category-nav" aria-label="Product categories">
      <div className="site-container desktop-category-nav__inner">
        <div className="desktop-mega-menu">
          <button type="button" className="desktop-category-nav__all">
            <Icon name="categories" size={18} />
            <span>All categories</span>
            <Icon name="chevronRight" size={14} className="desktop-mega-menu__chevron" />
          </button>

          {categories.length > 0 && activeCategory && (
            <div className="desktop-mega-menu__panel">
              <div className="desktop-mega-menu__parents">
                {categories.map((category) => (
                  <Link
                    key={category.id}
                    href={`/category/${category.slug}`}
                    className={category.slug === activeCategory.slug ? "is-active" : ""}
                    onMouseEnter={() => setActiveSlug(category.slug)}
                    onFocus={() => setActiveSlug(category.slug)}
                  >
                    <span>{category.name}</span>
                    <Icon name="chevronRight" size={14} />
                  </Link>
                ))}
              </div>

              <div className="desktop-mega-menu__content">
                <div className="desktop-mega-menu__heading">
                  <div>
                    <span>Explore</span>
                    <strong>{activeCategory.name}</strong>
                  </div>

                  <Link href={`/category/${activeCategory.slug}`}>View all</Link>
                </div>

                <div className="desktop-mega-menu__children">
                  {activeCategory.children.length > 0 ? (
                    activeCategory.children.map((child) => (
                      <Link key={child.id} href={`/category/${child.slug}`}>
                        <span>{child.name}</span>
                        {child.children.length > 0 && (
                          <small>{child.children.length} sections</small>
                        )}
                      </Link>
                    ))
                  ) : (
                    <Link href={`/category/${activeCategory.slug}`}>
                      <span>Browse {activeCategory.name}</span>
                      <small>View available products</small>
                    </Link>
                  )}
                </div>
              </div>

              <Link
                href={`/category/${activeCategory.slug}`}
                className="desktop-mega-menu__feature"
              >
                <div>
                  <span>Wholesale selection</span>
                  <strong>Source {activeCategory.name} faster.</strong>
                  <small>Browse current stock and business pricing.</small>
                </div>
                <span className="desktop-mega-menu__feature-arrow">→</span>
              </Link>
            </div>
          )}
        </div>

        <div className="desktop-category-nav__links">
          <Link href="/search" className="desktop-category-nav__link desktop-category-nav__link--red">
            Wholesale picks
          </Link>

          {categories.slice(0, 7).map((category) => (
            <Link
              key={category.id}
              href={`/category/${category.slug}`}
              className="desktop-category-nav__link"
            >
              {category.name}
            </Link>
          ))}
        </div>
      </div>
    </nav>
  );
}
