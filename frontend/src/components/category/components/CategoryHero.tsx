import { CatalogImage } from "@/components/commerce/components/CatalogImage";

import styles from "../css/CategoryPage.module.css";

type Props = {
  name: string;
  description?: string;
  total: number;
  imageURL?: string;
};

export function CategoryHero({
  name,
  description,
  total,
  imageURL,
}: Props) {
  return (
    <section
      className={styles.hero}
      aria-labelledby="category-title"
    >
      {imageURL ? (
        <div
          className={styles.heroBackdrop}
          aria-hidden="true"
        >
          <CatalogImage
            src={imageURL}
            alt=""
            fill
            sizes="(max-width: 48rem) 42vw, 34vw"
            className={
              styles.heroBackdropImage
            }
            aria-hidden="true"
          />
        </div>
      ) : null}

      <div
        className={styles.heroWash}
        aria-hidden="true"
      />

      <div className={styles.heroCopy}>
        <p className={styles.eyebrow}>
          Category
        </p>

        <h1 id="category-title">
          {name}
        </h1>

        <p
          className={
            styles.heroDescription
          }
        >
          {description?.trim() ||
            `Browse ${name} products selected for dependable business purchasing and everyday commerce.`}
        </p>

        <span
          className={styles.productCount}
        >
          {total.toLocaleString()} {" "}
          {total === 1
            ? "product"
            : "products"}
        </span>
      </div>

      <div className={styles.heroMedia}>
        {imageURL ? (
          <div
            className={
              styles.heroProductImage
            }
          >
            <CatalogImage
              src={imageURL}
              alt={`${name} category`}
              fill
              priority
              sizes="(max-width: 48rem) 42vw, 34vw"
              className={
                styles.heroSharpImage
              }
            />
          </div>
        ) : (
          <div
            className={styles.heroFallback}
            aria-hidden="true"
          >
            <span>
              {name
                .charAt(0)
                .toUpperCase()}
            </span>
          </div>
        )}
      </div>
    </section>
  );
}
