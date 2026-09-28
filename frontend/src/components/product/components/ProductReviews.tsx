"use client";

import { useMemo, useState } from "react";

import { Icon } from "@/components/ui/Icon";
import type {
  ProductReview,
  ProductReviewSummary,
} from "@/lib/api/contracts/reviews";

import styles from "../css/ProductReviews.module.css";

type Props = {
  reviews: ProductReview[];
  summary: ProductReviewSummary | null;
  fallbackRating: number;
  fallbackCount: number;
};

function Stars({ rating }: { rating: number }) {
  const rounded = Math.round(Math.max(0, Math.min(5, rating)));

  return (
    <span className={styles.stars} aria-label={`${rating.toFixed(1)} out of 5 stars`}>
      {[1, 2, 3, 4, 5].map((star) => (
        <span
          key={star}
          className={star <= rounded ? styles.starFilled : styles.starEmpty}
          aria-hidden="true"
        >
          ★
        </span>
      ))}
    </span>
  );
}

function displayName(review: ProductReview) {
  return review.reviewer_name?.trim() || review.customer_name?.trim() || "Customer";
}

function formatDate(value: string) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "";

  return new Intl.DateTimeFormat("en", {
    year: "numeric",
    month: "short",
    day: "numeric",
  }).format(date);
}

export function ProductReviews({
  reviews,
  summary,
  fallbackRating,
  fallbackCount,
}: Props) {
  const [mobileOpen, setMobileOpen] = useState(false);
  const rating = summary?.average_rating ?? fallbackRating;
  const count = summary?.total_reviews ?? fallbackCount;

  const ratingCounts = useMemo(() => {
    const counts = [0, 0, 0, 0, 0, 0];
    for (const review of reviews) {
      const value = Math.max(1, Math.min(5, Math.round(review.rating)));
      counts[value] += 1;
    }
    return counts;
  }, [reviews]);

  const listIsComplete = Boolean(count > 0 && reviews.length >= count);

  return (
    <section
      id="product-reviews"
      className={`${styles.surface} ${mobileOpen ? styles.mobileOpen : ""}`}
      aria-labelledby="product-reviews-heading"
    >
      <button
        type="button"
        className={styles.mobileToggle}
        aria-expanded={mobileOpen}
        onClick={() => setMobileOpen((current) => !current)}
      >
        <span>
          Reviews{count > 0 ? ` (${count.toLocaleString()})` : ""}
        </span>
        <span className={styles.mobileRating}>
          {count > 0 ? `${rating.toFixed(1)} ★` : "No reviews"}
          <Icon name="chevronRight" size={14} />
        </span>
      </button>

      <div className={styles.body}>
        <div className={styles.summary}>
          <span className={styles.eyebrow}>Customer reviews</span>
          <div className={styles.scoreRow}>
            <strong id="product-reviews-heading">
              {count > 0 ? rating.toFixed(1) : "—"}
            </strong>
            <div>
              {count > 0 ? <Stars rating={rating} /> : null}
              <span>
                {count > 0
                  ? `Based on ${count.toLocaleString()} ${count === 1 ? "review" : "reviews"}`
                  : "No published reviews yet"}
              </span>
            </div>
          </div>

          {summary && summary.verified_purchase_count > 0 ? (
            <small>
              {summary.verified_purchase_count.toLocaleString()} verified-purchase
              {summary.verified_purchase_count === 1 ? " review" : " reviews"}
            </small>
          ) : null}

          {listIsComplete ? (
            <div className={styles.breakdown} aria-label="Rating distribution">
              {[5, 4, 3, 2, 1].map((star) => {
                const value = ratingCounts[star];
                const percentage = count > 0 ? Math.round((value / count) * 100) : 0;

                return (
                  <div key={star}>
                    <span>{star} ★</span>
                    <span className={styles.bar} aria-hidden="true">
                      <span style={{ width: `${percentage}%` }} />
                    </span>
                    <span>{percentage}%</span>
                  </div>
                );
              })}
            </div>
          ) : null}
        </div>

        <div className={styles.reviewList}>
          {reviews.length > 0 ? (
            reviews.slice(0, 4).map((review) => (
              <article key={review.id} className={styles.reviewCard}>
                <div className={styles.reviewHeader}>
                  <div>
                    <strong>{displayName(review)}</strong>
                    {review.verified_purchase ? (
                      <span className={styles.verified}>Verified purchase</span>
                    ) : null}
                  </div>
                  <time dateTime={review.created_at}>{formatDate(review.created_at)}</time>
                </div>

                <Stars rating={review.rating} />
                {review.title ? <h3>{review.title}</h3> : null}
                {review.body ? <p>{review.body}</p> : null}
              </article>
            ))
          ) : (
            <div className={styles.empty}>
              <strong>No reviews yet</strong>
              <span>Verified buyers can review delivered purchases.</span>
            </div>
          )}
        </div>
      </div>
    </section>
  );
}
