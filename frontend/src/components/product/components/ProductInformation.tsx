"use client";

import { useState } from "react";

import { Icon } from "@/components/ui/Icon";
import type {
  ProductDetail,
  ProductVariant,
} from "@/lib/api/contracts/catalog";

import styles from "../css/ProductInformation.module.css";

type Props = {
  product: ProductDetail;
  selectedVariant?: ProductVariant;
};

type PanelKey = "description" | "specifications" | "shipping";

export function ProductInformation({ product, selectedVariant }: Props) {
  const [openMobile, setOpenMobile] = useState<PanelKey | null>(null);

  function toggle(panel: PanelKey) {
    setOpenMobile((current) => (current === panel ? null : panel));
  }

  const description =
    product.description ??
    product.short_description ??
    "Product description is being prepared.";

  return (
    <section className={styles.surface} aria-label="Product information">
      <article
        className={`${styles.panel} ${
          openMobile === "description" ? styles.mobileOpen : ""
        }`}
      >
        <button
          type="button"
          className={styles.mobileToggle}
          aria-expanded={openMobile === "description"}
          onClick={() => toggle("description")}
        >
          <span>Description</span>
          <Icon name="chevronRight" size={14} />
        </button>

        <div className={styles.content}>
          <h2>Description</h2>
          <p className={styles.description}>{description}</p>
        </div>
      </article>

      <article
        className={`${styles.panel} ${
          openMobile === "specifications" ? styles.mobileOpen : ""
        }`}
      >
        <button
          type="button"
          className={styles.mobileToggle}
          aria-expanded={openMobile === "specifications"}
          onClick={() => toggle("specifications")}
        >
          <span>Specifications</span>
          <Icon name="chevronRight" size={14} />
        </button>

        <div className={styles.content}>
          <h2>Specifications</h2>
          <dl className={styles.specs}>
            <div><dt>Product code</dt><dd>{product.product_code}</dd></div>
            {product.brand ? <div><dt>Brand</dt><dd>{product.brand}</dd></div> : null}
            <div><dt>Category</dt><dd>{product.category.name}</dd></div>
            {selectedVariant ? (
              <>
                <div><dt>SKU</dt><dd>{selectedVariant.sku}</dd></div>
                {selectedVariant.color_name ? (
                  <div><dt>Color</dt><dd>{selectedVariant.color_name}</dd></div>
                ) : null}
                {selectedVariant.size ? (
                  <div><dt>Size</dt><dd>{selectedVariant.size}</dd></div>
                ) : null}
                {selectedVariant.weight_grams ? (
                  <div>
                    <dt>Weight</dt>
                    <dd>{selectedVariant.weight_grams.toLocaleString()} g</dd>
                  </div>
                ) : null}
              </>
            ) : null}
          </dl>
        </div>
      </article>

      <article
        className={`${styles.panel} ${
          openMobile === "shipping" ? styles.mobileOpen : ""
        }`}
      >
        <button
          type="button"
          className={styles.mobileToggle}
          aria-expanded={openMobile === "shipping"}
          onClick={() => toggle("shipping")}
        >
          <span>Shipping &amp; Returns</span>
          <Icon name="chevronRight" size={14} />
        </button>

        <div className={styles.content}>
          <h2>Shipping &amp; Returns</h2>
          <p>
            Delivery methods, charges and timing are calculated during checkout
            from the destination and order contents.
          </p>
          <p>
            Eligible returns and refunds are managed from the customer order
            flow after purchase.
          </p>
        </div>
      </article>
    </section>
  );
}
