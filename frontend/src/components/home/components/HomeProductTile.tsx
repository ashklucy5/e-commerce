import Link from "next/link";

import { ProductCardCartButton } from "@/components/commerce/components/ProductCardCartButton";
import { ProductWishlistButton } from "@/components/commerce/components/ProductWishlistButton";
import { CatalogImage } from "@/components/commerce/components/CatalogImage";
import type { HomeProductCard } from "@/lib/api/contracts/home";
import { formatMoney } from "@/lib/money/format";

import styles from "../css/HomeProductTile.module.css";

type Props = {
  product: HomeProductCard;
  variant?: "rail" | "catalog";
};

function discountPercent(product: HomeProductCard) {
  const compareAt = product.compare_at_price_amount;

  if (!compareAt || compareAt <= product.price_amount) {
    return 0;
  }

  return Math.round(((compareAt - product.price_amount) / compareAt) * 100);
}

export function HomeProductTile({ product, variant = "rail" }: Props) {
  const discount = discountPercent(product);
  const discounted = discount > 0;

  return (
    <article className={styles.card}>
      <Link href={`/product/${product.slug}`} className={styles.link}>
        <div className={styles.media}>
          {product.primary_image_url ? (
            <CatalogImage
              src={product.primary_image_url}
              alt={product.name}
              fill
              sizes={
                variant === "catalog"
                  ? "(max-width: 48rem) 50vw, (max-width: 64rem) 25vw, 15vw"
                  : "(max-width: 48rem) 47vw, (max-width: 64rem) 30vw, 18vw"
              }
              className={styles.image}
            />
          ) : (
            <span className={styles.fallback} aria-hidden="true">
              {product.name.charAt(0).toUpperCase()}
            </span>
          )}

          {discounted ? (
            <span className={styles.discount}>-{discount}%</span>
          ) : null}
        </div>

        <div className={styles.body}>
          <h3>{product.name}</h3>

          <div className={styles.meta}>
            <span>{product.in_stock ? "In stock" : "Unavailable"}</span>

            {product.sold_quantity > 0 ? (
              <span>{product.sold_quantity.toLocaleString()} sold</span>
            ) : null}
          </div>

          <div className={styles.price}>
            <strong>{formatMoney(product.price_amount, product.currency)}</strong>

            {discounted && product.compare_at_price_amount ? (
              <span>
                {formatMoney(product.compare_at_price_amount, product.currency)}
              </span>
            ) : null}
          </div>
        </div>
      </Link>

      <ProductWishlistButton
        productId={product.id}
        productName={product.name}
        placement="home"
      />
      <ProductCardCartButton
        productName={product.name}
        productSlug={product.slug}
        inStock={product.in_stock}
        className={styles.cartButton}
      />
    </article>
  );
}

