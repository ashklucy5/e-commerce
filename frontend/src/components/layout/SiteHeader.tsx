import Image from "next/image";
import Link from "next/link";

import { CartBadge } from "@/components/commerce/CartBadge";
import { DesktopCategoryNav } from "@/components/navigation/DesktopCategoryNav";
import { SearchAutocomplete } from "@/components/search/components/SearchAutocomplete";
import { Icon } from "@/components/ui/Icon";

import { getCategoryTree } from "@/lib/api/catalog";
import { siteConfig } from "@/lib/config/site";

import { PromoBar } from "./PromoBar";

export async function SiteHeader() {
  const categories = await getCategoryTree().catch(() => []);

  return (
    <>
      <PromoBar />

      <header className="site-header">
        <div className="site-container site-header__desktop">
          <Link
            href="/"
            className="site-header__brand"
            aria-label={`${siteConfig.name} home`}
          >
            <Image
              src={siteConfig.assets.logo}
              alt=""
              width={270}
              height={72}
              priority
              className="site-header__logo"
            />
          </Link>

          <SearchAutocomplete />

          <nav
            className="site-header__actions"
            aria-label="Customer actions"
          >
            <Link
              href="/account"
              className="header-action"
            >
              <Icon
                name="account"
                size={21}
              />

              <span>
                Account
              </span>
            </Link>

            <Link
              href="/cart"
              className="header-action header-action--cart"
            >
              <span className="header-action__icon-wrap">
                <Icon
                  name="cart"
                  size={23}
                />

                <CartBadge className="header-action__badge" />
              </span>

              <span>
                Cart
              </span>
            </Link>
          </nav>
        </div>

        <div className="site-container site-header__tablet">
          <div className="tablet-header__top">
            <Link
              href="/"
              className="tablet-header__brand"
              aria-label={`${siteConfig.name} home`}
            >
              <Image
                src={siteConfig.assets.logo}
                alt=""
                width={270}
                height={72}
                priority
                className="tablet-header__logo"
              />
            </Link>

            <div className="tablet-header__actions">
              <Link
                href="/account"
                className="icon-button"
                aria-label="Account"
              >
                <Icon
                  name="account"
                  size={20}
                />
              </Link>

              <Link
                href="/cart"
                className="icon-button icon-button--cart"
                aria-label="Cart"
              >
                <Icon
                  name="cart"
                  size={22}
                />

                <CartBadge className="mobile-cart-badge" />
              </Link>
            </div>
          </div>

          <div className="tablet-header__search">
            <SearchAutocomplete />
          </div>
        </div>

        <div
          className="tablet-category-strip site-container"
          aria-label="Categories"
        >
          <Link
            href="/categories"
            className="tablet-category-strip__all"
          >
            <Icon
              name="categories"
              size={16}
            />

            <span>
              Categories
            </span>
          </Link>

          {categories
            .slice(0, 7)
            .map((category) => (
              <Link
                key={category.id}
                href={`/category/${category.slug}`}
              >
                {category.name}
              </Link>
            ))}
        </div>

        <DesktopCategoryNav categories={categories} />
      </header>
    </>
  );
}