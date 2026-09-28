import Image from "next/image";
import Link from "next/link";

import { CartBadge } from "@/components/commerce/CartBadge";
import { CategoryStrip } from "@/components/navigation/components/CategoryStrip";
import { SearchAutocomplete } from "@/components/search/components/SearchAutocomplete";
import { Icon } from "@/components/ui/Icon";
import { WishlistBadge } from "@/components/commerce/components/WishlistBadge";
import { getCategoryTree } from "@/lib/api/catalog";
import { siteConfig } from "@/lib/config/site";
import { CustomerAccountAction } from "./CustomerAccountAction";

import styles from "../css/SiteHeader.module.css";

export async function SiteHeader() {
  const categories = await getCategoryTree().catch(() => []);

  return (
    <header className={styles.header}>
      <div className={`site-container ${styles.mainRow}`}>
        <Link
          href="/"
          className={styles.brand}
          aria-label={`${siteConfig.name} home`}
        >
          <Image
            src={siteConfig.assets.logo}
            alt=""
            width={1100}
            height={439}
            priority
            sizes="(max-width: 40rem) 7rem, (max-width: 64rem) 8rem, 9.5rem"
            className={styles.logo}
          />
        </Link>

        <div className={styles.searchSlot}>
          <SearchAutocomplete />
        </div>

        <nav
          className={styles.actions}
          aria-label="Customer actions"
        >

          <CustomerAccountAction />

          <Link href="/cart" className={styles.action} aria-label="Cart">
            <span className={styles.cartIcon}>
              <Icon name="cart" size={21} />
              <CartBadge className={styles.cartBadge} />
            </span>
            <span className={styles.actionLabel}>Cart</span>
          </Link>

          <Link
  href="/account/wishlist"
  className={styles.action}
  aria-label="Wishlist"
>
  <span className={styles.cartIcon}>
    <Icon name="heart" size={21} />
    <WishlistBadge className={styles.cartBadge} />
  </span>

  <span className={styles.actionLabel}>Wishlist</span>
</Link>
        </nav>
      </div>

      <CategoryStrip categories={categories} />
    </header>
  );
}
