import type { Metadata } from "next";
import Link from "next/link";

import {
  MobileCategoryBrowser,
} from "@/components/navigation/MobileCategoryBrowser";

import {
  getCategoryTree,
} from "@/lib/api/catalog";

export const dynamic = "force-dynamic";

export const metadata: Metadata = {
  title: "Categories",

  description:
    "Browse Ene dei products by category and subcategory.",
};

export default async function CategoriesPage() {
  const categories =
    await getCategoryTree().catch(
      () => [],
    );

  return (
    <>
      <div className="categories-mobile">
        <MobileCategoryBrowser
          categories={categories}
        />
      </div>

      <div className="categories-desktop site-container">
        <section className="desktop-categories-page">
          <header className="desktop-categories-page__header">
            <p>Explore</p>

            <h1>Shop by category</h1>
          </header>

          <div className="desktop-categories-grid">
            {categories.map(
              (category) => (
                <article
                  key={category.id}
                  className="desktop-category-card"
                >
                  <Link
                    href={`/category/${category.slug}`}
                    className="desktop-category-card__title"
                  >
                    {category.name}
                  </Link>

                  <div className="desktop-category-card__children">
                    {category.children.map(
                      (child) => (
                        <Link
                          key={child.id}
                          href={`/category/${child.slug}`}
                        >
                          {child.name}
                        </Link>
                      ),
                    )}
                  </div>
                </article>
              ),
            )}
          </div>
        </section>
      </div>
    </>
  );
}
