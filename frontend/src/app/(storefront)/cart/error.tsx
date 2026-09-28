// Location: src/app/(storefront)/cart/error.tsx
"use client";

import Link from "next/link";

import styles from "@/components/cart/css/CartPage.module.css";

export default function CartError({
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  return (
    <main className={styles.page}>
      <div className="site-container">
        <section className={styles.statePanel} role="alert">
          <span className={styles.stateBadge} aria-hidden="true">
            !
          </span>
          <h1>Something interrupted the cart.</h1>
          <p>
            Your cart data has not been intentionally changed. Try loading this
            page again, or continue shopping and return later.
          </p>
          <div className={styles.stateActions}>
            <button type="button" onClick={reset}>
              Try again
            </button>
            <Link href="/">Continue shopping</Link>
          </div>
        </section>
      </div>
    </main>
  );
}
