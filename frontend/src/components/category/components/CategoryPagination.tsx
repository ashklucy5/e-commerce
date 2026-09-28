import Link from "next/link";

import styles from "../css/CategoryPage.module.css";

type Props = {
  page: number;
  totalPages: number;
  hrefForPage: (page: number) => string;
};

function visiblePages(page: number, totalPages: number) {
  const pages = new Set<number>([1, totalPages, page - 1, page, page + 1]);

  return [...pages]
    .filter((value) => value >= 1 && value <= totalPages)
    .sort((a, b) => a - b);
}

export function CategoryPagination({
  page,
  totalPages,
  hrefForPage,
}: Props) {
  if (totalPages <= 1) return null;

  const pages = visiblePages(page, totalPages);

  return (
    <nav className={styles.pagination} aria-label="Category pages">
      {page > 1 ? (
        <Link className={styles.pageArrow} href={hrefForPage(page - 1)} rel="prev">
          <span aria-hidden="true">‹</span>
          <span className={styles.srOnly}>Previous page</span>
        </Link>
      ) : (
        <span className={`${styles.pageArrow} ${styles.pageDisabled}`} aria-hidden="true">
          ‹
        </span>
      )}

      {pages.map((pageNumber, index) => {
        const previous = pages[index - 1];
        const showGap = previous !== undefined && pageNumber - previous > 1;

        return (
          <span key={pageNumber} className={styles.pageGroup}>
            {showGap ? <span className={styles.pageGap}>…</span> : null}

            <Link
              href={hrefForPage(pageNumber)}
              className={`${styles.pageLink} ${pageNumber === page ? styles.pageCurrent : ""}`}
              aria-current={pageNumber === page ? "page" : undefined}
            >
              {pageNumber}
            </Link>
          </span>
        );
      })}

      {page < totalPages ? (
        <Link className={styles.pageArrow} href={hrefForPage(page + 1)} rel="next">
          <span aria-hidden="true">›</span>
          <span className={styles.srOnly}>Next page</span>
        </Link>
      ) : (
        <span className={`${styles.pageArrow} ${styles.pageDisabled}`} aria-hidden="true">
          ›
        </span>
      )}
    </nav>
  );
}
