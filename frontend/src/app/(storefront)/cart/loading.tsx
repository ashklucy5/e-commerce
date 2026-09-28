// Location: src/app/(storefront)/cart/loading.tsx
import styles from "@/components/cart/css/CartPage.module.css";

export default function CartLoading() {
  return (
    <main className={styles.page} aria-busy="true" aria-label="Loading cart">
      <div className="site-container">
        <div className={styles.loadingHeader}>
          <span />
          <i />
        </div>

        <div className={styles.loadingLayout}>
          <section className={styles.loadingItems}>
            <div className={styles.loadingColumnHeader} />
            {[0, 1, 2].map((item) => (
              <div key={item} className={styles.loadingItem}>
                <span className={styles.loadingImage} />
                <div className={styles.loadingCopy}>
                  <span />
                  <span />
                  <span />
                </div>
                <span className={styles.loadingSmall} />
                <span className={styles.loadingQuantity} />
                <span className={styles.loadingSmall} />
              </div>
            ))}
          </section>

          <aside className={styles.loadingSummary}>
            <span />
            <span />
            <span />
            <span className={styles.loadingButton} />
          </aside>
        </div>
      </div>
    </main>
  );
}
