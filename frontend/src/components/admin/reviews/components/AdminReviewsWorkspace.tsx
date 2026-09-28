"use client";

import {
  type FormEvent,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { createPortal } from "react-dom";

import { useAdminSession } from "@/components/admin/layout/context/AdminSessionContext";
import { AdminRequestError, adminFetch } from "@/lib/admin/api";
import type {
  AdminReview,
  AdminReviewListResponse,
  AdminReviewPaginationMeta,
  AdminReviewResponse,
  AdminReviewStatus,
} from "@/lib/admin/review-types";

import styles from "../css/AdminReviews.module.css";

type StatusFilter = "" | AdminReviewStatus;
type RatingFilter = "" | "1" | "2" | "3" | "4" | "5";

type ReviewCounts = {
  all: number;
  published: number;
  hidden: number;
};

const PAGE_SIZE = 24;

function errorMessage(value: unknown): string {
  if (value instanceof AdminRequestError) return value.message;
  if (value instanceof Error) return value.message;
  return "Unable to complete the review request.";
}

function formatDateTime(value?: string): string {
  if (!value) return "—";

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "—";

  return new Intl.DateTimeFormat("en", {
    month: "short",
    day: "numeric",
    year: "numeric",
    hour: "numeric",
    minute: "2-digit",
  }).format(date);
}

function reviewExcerpt(review: AdminReview): string {
  const body = review.body?.trim();
  if (body) return body;

  const title = review.title?.trim();
  if (title) return title;

  return "No written review was provided.";
}

function statusLabel(status: AdminReviewStatus): string {
  return status === "published" ? "Published" : "Hidden";
}

function ReviewStars({ rating }: { rating: number }) {
  return (
    <span className={styles.stars} aria-label={`${rating} out of 5 stars`}>
      {Array.from({ length: 5 }, (_, index) => (
        <span
          key={index}
          className={index < rating ? styles.starFilled : styles.starEmpty}
          aria-hidden="true"
        >
          ★
        </span>
      ))}
    </span>
  );
}

export default function AdminReviewsWorkspace() {
  const principal = useAdminSession();
  const isSuperAdmin = principal.staff.roles.includes("admin_superuser");
  const permissions = principal.staff.permissions;

  const hasPermission = useCallback(
    (permission: string) => isSuperAdmin || permissions.includes(permission),
    [isSuperAdmin, permissions],
  );

  const canRead = hasPermission("admin.review.read");
  const canModerate = hasPermission("admin.review.moderate");

  const [items, setItems] = useState<AdminReview[]>([]);
  const [meta, setMeta] = useState<AdminReviewPaginationMeta | null>(null);
  const [counts, setCounts] = useState<ReviewCounts>({
    all: 0,
    published: 0,
    hidden: 0,
  });

  const [status, setStatus] = useState<StatusFilter>("");
  const [rating, setRating] = useState<RatingFilter>("");
  const [queryInput, setQueryInput] = useState("");
  const [query, setQuery] = useState("");
  const [page, setPage] = useState(1);

  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [detail, setDetail] = useState<AdminReview | null>(null);

  const [loading, setLoading] = useState(true);
  const [detailLoading, setDetailLoading] = useState(false);
  const [moderating, setModerating] = useState(false);
  const [error, setError] = useState("");
  const [detailError, setDetailError] = useState("");

  const listAbortRef = useRef<AbortController | null>(null);

  const loadCounts = useCallback(async () => {
    if (!canRead) return;

    try {
      const [all, published, hidden] = await Promise.all([
        adminFetch<AdminReviewListResponse>("/reviews?page=1&limit=1"),
        adminFetch<AdminReviewListResponse>("/reviews?page=1&limit=1&status=published"),
        adminFetch<AdminReviewListResponse>("/reviews?page=1&limit=1&status=hidden"),
      ]);

      setCounts({
        all: all.meta.total,
        published: published.meta.total,
        hidden: hidden.meta.total,
      });
    } catch {
      // The moderation list remains usable even if summary counts fail.
    }
  }, [canRead]);

  const loadReviews = useCallback(async () => {
    if (!canRead) {
      setLoading(false);
      return;
    }

    listAbortRef.current?.abort();
    const controller = new AbortController();
    listAbortRef.current = controller;

    setLoading(true);
    setError("");

    const params = new URLSearchParams({
      page: String(page),
      limit: String(PAGE_SIZE),
    });

    if (status) params.set("status", status);
    if (rating) params.set("rating", rating);
    if (query.trim()) params.set("q", query.trim());

    try {
      const response = await adminFetch<AdminReviewListResponse>(
        `/reviews?${params.toString()}`,
        { signal: controller.signal },
      );

      if (controller.signal.aborted) return;

      setItems(response.data ?? []);
      setMeta(response.meta);
    } catch (value: unknown) {
      if (controller.signal.aborted) return;

      setItems([]);
      setMeta(null);
      setError(errorMessage(value));
    } finally {
      if (!controller.signal.aborted) setLoading(false);
    }
  }, [canRead, page, query, rating, status]);

  const loadDetail = useCallback(
    async (reviewId: string) => {
      if (!canRead) return;

      setDetailLoading(true);
      setDetailError("");

      try {
        const response = await adminFetch<AdminReviewResponse>(
          `/reviews/${reviewId}`,
        );
        setDetail(response.data);
      } catch (value: unknown) {
        setDetail(null);
        setDetailError(errorMessage(value));
      } finally {
        setDetailLoading(false);
      }
    },
    [canRead],
  );

  useEffect(() => {
    void loadCounts();
  }, [loadCounts]);

  useEffect(() => {
    void loadReviews();
    return () => listAbortRef.current?.abort();
  }, [loadReviews]);

  useEffect(() => {
    if (!selectedId) {
      setDetail(null);
      setDetailError("");
      return;
    }

    void loadDetail(selectedId);
  }, [loadDetail, selectedId]);

  useEffect(() => {
    const timer = window.setTimeout(() => {
      setQuery(queryInput.trim());
      setPage(1);
    }, 300);

    return () => window.clearTimeout(timer);
  }, [queryInput]);

  useEffect(() => {
    if (!selectedId) return;

    const previousOverflow = document.body.style.overflow;
    document.body.style.overflow = "hidden";

    return () => {
      document.body.style.overflow = previousOverflow;
    };
  }, [selectedId]);

  const selectedFromList = useMemo(
    () => items.find((review) => review.id === selectedId) ?? null,
    [items, selectedId],
  );

  const activeReview = detail ?? selectedFromList;

  function submitSearch(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setQuery(queryInput.trim());
    setPage(1);
  }

  function selectStatus(nextStatus: StatusFilter) {
    setStatus(nextStatus);
    setPage(1);
  }

  async function updateStatus(nextStatus: AdminReviewStatus) {
    if (!activeReview || !canModerate || moderating) return;

    const reviewId = activeReview.id;
    setModerating(true);
    setDetailError("");

    try {
      const response = await adminFetch<AdminReviewResponse>(
        `/reviews/${reviewId}/status`,
        {
          method: "PATCH",
          body: JSON.stringify({ status: nextStatus }),
        },
      );

      setDetail(response.data);
      setItems((current) =>
        current.map((item) =>
          item.id === reviewId ? response.data : item,
        ),
      );

      await Promise.all([loadCounts(), loadReviews()]);
    } catch (value: unknown) {
      setDetailError(errorMessage(value));
    } finally {
      setModerating(false);
    }
  }

  if (!canRead) {
    return (
      <main className={styles.page}>
        <section className={styles.permissionDenied}>
          <span>Restricted area</span>
          <h1>Reviews</h1>
          <p>Your current role does not include review visibility.</p>
        </section>
      </main>
    );
  }

  return (
    <main className={styles.page}>
      <header className={styles.pageHeader}>
        <div>
          <p className={styles.eyebrow}>Storefront moderation</p>
          <h1>Reviews</h1>
          <p>
            Control which verified-purchase reviews are visible on the storefront.
            Customer review content is never edited here.
          </p>
        </div>

        <button
          type="button"
          className={styles.refreshButton}
          onClick={() => void Promise.all([loadReviews(), loadCounts()])}
          disabled={loading}
        >
          <span aria-hidden="true">↻</span>
          <span className={styles.refreshLabel}>Refresh</span>
        </button>
      </header>

      <section className={styles.summaryBar} aria-label="Review moderation summary">
        <button
          type="button"
          className={status === "" ? styles.summaryActive : ""}
          onClick={() => selectStatus("")}
        >
          <span>All reviews</span>
          <strong>{counts.all}</strong>
        </button>

        <button
          type="button"
          className={status === "published" ? styles.summaryActive : ""}
          onClick={() => selectStatus("published")}
        >
          <span>Published</span>
          <strong>{counts.published}</strong>
        </button>

        <button
          type="button"
          className={status === "hidden" ? styles.summaryActive : ""}
          onClick={() => selectStatus("hidden")}
        >
          <span>Hidden</span>
          <strong>{counts.hidden}</strong>
        </button>
      </section>

      <section className={styles.workspace}>
        <div className={styles.toolbar}>
          <form className={styles.searchForm} onSubmit={submitSearch}>
            <span className={styles.searchIcon} aria-hidden="true">⌕</span>
            <input
              value={queryInput}
              onChange={(event) => setQueryInput(event.target.value)}
              placeholder="Order number, SKU, phone or UUID…"
              aria-label="Search reviews"
              maxLength={100}
            />
            {queryInput ? (
              <button
                type="button"
                className={styles.clearSearch}
                onClick={() => {
                  setQueryInput("");
                  setQuery("");
                  setPage(1);
                }}
                aria-label="Clear review search"
              >
                ×
              </button>
            ) : null}
          </form>

          <label className={styles.ratingFilter}>
            <span>Rating</span>
            <select
              value={rating}
              onChange={(event) => {
                setRating(event.target.value as RatingFilter);
                setPage(1);
              }}
            >
              <option value="">All ratings</option>
              <option value="5">5 stars</option>
              <option value="4">4 stars</option>
              <option value="3">3 stars</option>
              <option value="2">2 stars</option>
              <option value="1">1 star</option>
            </select>
          </label>
        </div>

        {error ? <div className={styles.errorBanner}>{error}</div> : null}

        <div className={styles.listHeader} aria-hidden="true">
          <span>Review</span>
          <span>Customer / order</span>
          <span>Status</span>
          <span>Date</span>
        </div>

        <div className={styles.reviewList}>
          {loading ? (
            Array.from({ length: 7 }, (_, index) => (
              <div className={styles.skeletonRow} key={index}>
                <span />
                <span />
                <span />
              </div>
            ))
          ) : items.length === 0 ? (
            <div className={styles.emptyState}>
              <span aria-hidden="true">✦</span>
              <strong>No reviews found</strong>
              <p>Try changing the status, rating or exact identifier search.</p>
            </div>
          ) : (
            items.map((review) => (
              <button
                type="button"
                className={styles.reviewRow}
                key={review.id}
                onClick={() => setSelectedId(review.id)}
              >
                <div className={styles.reviewPrimary}>
                  <div className={styles.reviewTopline}>
                    <ReviewStars rating={review.rating} />
                    <span className={styles.verifiedBadge}>Verified purchase</span>
                  </div>

                  <strong className={styles.productName}>{review.product_name}</strong>

                  {review.title ? (
                    <span className={styles.reviewTitle}>{review.title}</span>
                  ) : null}

                  <p className={styles.reviewExcerpt}>{reviewExcerpt(review)}</p>
                </div>

                <div className={styles.reviewIdentity}>
                  <strong>{review.customer_name || "Customer"}</strong>
                  <span>{review.order_number}</span>
                  <small>{review.sku}</small>
                </div>

                <span className={styles.statusBadge} data-status={review.status}>
                  {statusLabel(review.status)}
                </span>

                <span className={styles.reviewDate}>{formatDateTime(review.created_at)}</span>

                <span className={styles.openChevron} aria-hidden="true">›</span>
              </button>
            ))
          )}
        </div>

        {meta && meta.total_pages > 1 ? (
          <footer className={styles.pagination}>
            <span>
              Page {meta.page} of {meta.total_pages} · {meta.total} review{meta.total === 1 ? "" : "s"}
            </span>

            <div>
              <button
                type="button"
                onClick={() => setPage((current) => Math.max(1, current - 1))}
                disabled={!meta.has_previous || loading}
              >
                Previous
              </button>
              <button
                type="button"
                onClick={() => setPage((current) => current + 1)}
                disabled={!meta.has_next || loading}
              >
                Next
              </button>
            </div>
          </footer>
        ) : null}
      </section>

      {selectedId && typeof document !== "undefined"
        ? createPortal(
            <div
              className={styles.sheetBackdrop}
              role="presentation"
              onMouseDown={(event) => {
                if (event.target === event.currentTarget) setSelectedId(null);
              }}
            >
              <aside
                className={styles.detailSheet}
                role="dialog"
                aria-modal="true"
                aria-label="Review details"
              >
                <div className={styles.sheetHandle} aria-hidden="true" />

                <header className={styles.detailHeader}>
                  <div>
                    <p className={styles.eyebrow}>Review details</p>
                    <h2>{activeReview?.product_name ?? "Review"}</h2>
                  </div>

                  <button
                    type="button"
                    className={styles.closeButton}
                    onClick={() => setSelectedId(null)}
                    aria-label="Close review details"
                  >
                    ×
                  </button>
                </header>

                <div className={styles.detailScroll}>
                  {detailError ? (
                    <div className={styles.errorBanner}>{detailError}</div>
                  ) : null}

                  {detailLoading && !activeReview ? (
                    <div className={styles.detailLoading}>Loading review…</div>
                  ) : activeReview ? (
                    <>
                      <section className={styles.reviewCard}>
                        <div className={styles.reviewCardTop}>
                          <ReviewStars rating={activeReview.rating} />
                          <span className={styles.statusBadge} data-status={activeReview.status}>
                            {statusLabel(activeReview.status)}
                          </span>
                        </div>

                        {activeReview.title ? <h3>{activeReview.title}</h3> : null}
                        <p>{activeReview.body?.trim() || "No written review was provided."}</p>

                        <div className={styles.verifiedLine}>
                          <span aria-hidden="true">✓</span>
                          Verified purchase
                        </div>
                      </section>

                      <section className={styles.detailSection}>
                        <div className={styles.sectionHeading}>
                          <div>
                            <span>Commerce context</span>
                            <h3>Purchase details</h3>
                          </div>
                        </div>

                        <div className={styles.factGrid}>
                          <div>
                            <span>Product</span>
                            <strong>{activeReview.product_name}</strong>
                          </div>
                          <div>
                            <span>SKU</span>
                            <strong>{activeReview.sku}</strong>
                          </div>
                          <div>
                            <span>Order</span>
                            <strong>{activeReview.order_number}</strong>
                          </div>
                          <div>
                            <span>Customer</span>
                            <strong>{activeReview.customer_name || "Customer"}</strong>
                          </div>
                          <div>
                            <span>Customer phone</span>
                            <strong>{activeReview.customer_phone || "—"}</strong>
                          </div>
                          <div>
                            <span>Created</span>
                            <strong>{formatDateTime(activeReview.created_at)}</strong>
                          </div>
                        </div>
                      </section>

                      <section className={styles.detailSection}>
                        <div className={styles.sectionHeading}>
                          <div>
                            <span>Moderation</span>
                            <h3>Storefront visibility</h3>
                          </div>
                        </div>

                        <div className={styles.moderationState} data-status={activeReview.status}>
                          <div>
                            <span>Current state</span>
                            <strong>{statusLabel(activeReview.status)}</strong>
                          </div>
                          <p>
                            {activeReview.status === "published"
                              ? "This review is currently visible wherever the storefront displays it."
                              : "This review is retained in the system but hidden from the storefront."}
                          </p>
                        </div>

                        {canModerate ? (
                          <button
                            type="button"
                            className={
                              activeReview.status === "published"
                                ? styles.hideButton
                                : styles.publishButton
                            }
                            onClick={() =>
                              void updateStatus(
                                activeReview.status === "published" ? "hidden" : "published",
                              )
                            }
                            disabled={moderating}
                          >
                            {moderating
                              ? "Updating…"
                              : activeReview.status === "published"
                                ? "Hide review from storefront"
                                : "Publish review to storefront"}
                          </button>
                        ) : (
                          <p className={styles.readOnlyNote}>
                            Your role can view reviews but does not include moderation permission.
                          </p>
                        )}
                      </section>

                      <details className={styles.technicalDetails}>
                        <summary>Technical identifiers</summary>
                        <dl>
                          <div><dt>Review ID</dt><dd>{activeReview.id}</dd></div>
                          <div><dt>Customer ID</dt><dd>{activeReview.customer_id}</dd></div>
                          <div><dt>Order ID</dt><dd>{activeReview.order_id}</dd></div>
                          <div><dt>Order item ID</dt><dd>{activeReview.order_item_id}</dd></div>
                          <div><dt>Product ID</dt><dd>{activeReview.product_id}</dd></div>
                          <div><dt>Variant ID</dt><dd>{activeReview.variant_id}</dd></div>
                        </dl>
                      </details>
                    </>
                  ) : (
                    <div className={styles.detailLoading}>Review unavailable.</div>
                  )}
                </div>
              </aside>
            </div>,
            document.body,
          )
        : null}
    </main>
  );
}
