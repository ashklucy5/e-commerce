// ENE_PRODUCT_CARD_WISHLIST_V1
// Location: src/components/commerce/components/ProductWishlistButton.tsx

"use client";

import { useRouter } from "next/navigation";
import { useEffect, useState, useSyncExternalStore } from "react";
import {
  useCustomerSession,
} from "@/lib/account/use-customer-session";
import type {
  AccountWishlistItem,
  AccountWishlistResponse,
} from "@/lib/api/contracts/account";

import styles from "../css/ProductWishlistButton.module.css";

export const WISHLIST_UPDATED_EVENT = "ene-wishlist-updated";

type Placement = "catalog" | "home" | "recommendation";
type SharedState = "idle" | "loading" | "ready" | "signed-out" | "error";
type Feedback = "idle" | "error";

type Props = {
  productId: string;
  productName: string;
  placement?: Placement;
};

type WishlistStateResponse = {
  data?: {
    wishlisted?: boolean;
  };
};

let sharedState: SharedState = "idle";
let sharedRevision = 0;
let wishlistProductIds = new Set<string>();
let sharedLoadPromise: Promise<void> | null = null;
const sharedListeners = new Set<() => void>();

function emitSharedChange() {
  sharedRevision += 1;
  for (const listener of sharedListeners) listener();
}

function subscribeShared(listener: () => void) {
  sharedListeners.add(listener);
  return () => sharedListeners.delete(listener);
}

function getSharedRevision() {
  return sharedRevision;
}

function getServerRevision() {
  return 0;
}

function productIdsFromItems(items: AccountWishlistItem[]) {
  return items
    .map((item) => item.product_id?.trim())
    .filter((id): id is string => Boolean(id));
}

async function readError(response: Response, fallback: string) {
  try {
    const payload = (await response.json()) as {
      error?: { message?: string } | string;
      message?: string;
    };

    if (typeof payload.error === "string" && payload.error.trim()) {
      return payload.error.trim();
    }

    if (
      payload.error &&
      typeof payload.error === "object" &&
      payload.error.message?.trim()
    ) {
      return payload.error.message.trim();
    }

    return payload.message?.trim() || fallback;
  } catch {
    return fallback;
  }
}

async function fetchWishlistPage(page: number) {
  const response = await fetch(
    `/api/storefront/account/wishlist?page=${page}&limit=100`,
    { cache: "no-store" },
  );

  if (response.status === 401) {
    return null;
  }

  if (!response.ok) {
    throw new Error(await readError(response, "Unable to load wishlist state."));
  }

  return (await response.json()) as AccountWishlistResponse;
}

async function loadWishlistState() {
  sharedState = "loading";
  emitSharedChange();

  try {
    const firstPage = await fetchWishlistPage(1);

    if (!firstPage) {
      wishlistProductIds = new Set();
      sharedState = "signed-out";
      emitSharedChange();
      return;
    }

    const nextIds = new Set(productIdsFromItems(firstPage.data ?? []));
    const totalPages = Math.max(1, Number(firstPage.meta?.total_pages) || 1);

    // Wishlist is account data, so load all saved IDs once rather than issuing
    // one state request for every product card on the page.
    for (let start = 2; start <= totalPages; start += 4) {
      const pages = Array.from(
        { length: Math.min(4, totalPages - start + 1) },
        (_, index) => start + index,
      );

      const results = await Promise.all(pages.map(fetchWishlistPage));

      for (const result of results) {
        if (!result) {
          wishlistProductIds = new Set();
          sharedState = "signed-out";
          emitSharedChange();
          return;
        }

        for (const id of productIdsFromItems(result.data ?? [])) {
          nextIds.add(id);
        }
      }
    }

    wishlistProductIds = nextIds;
    sharedState = "ready";
    emitSharedChange();
  } catch {
    sharedState = "error";
    emitSharedChange();
  }
}

function ensureWishlistLoaded() {
  if (sharedState === "ready" || sharedState === "signed-out") {
    return Promise.resolve();
  }

  if (!sharedLoadPromise) {
    sharedLoadPromise = loadWishlistState().finally(() => {
      sharedLoadPromise = null;
    });
  }

  return sharedLoadPromise;
}

async function resolveSingleProductState(productId: string) {
  await ensureWishlistLoaded();

  if (sharedState === "signed-out") return null;
  if (sharedState === "ready") return wishlistProductIds.has(productId);

  // If the list request failed, resolve only the clicked product before
  // deciding between PUT and DELETE. This avoids an incorrect toggle.
  const response = await fetch(
    `/api/storefront/account/wishlist/${encodeURIComponent(productId)}`,
    { cache: "no-store" },
  );

  if (response.status === 401) {
    wishlistProductIds = new Set();
    sharedState = "signed-out";
    emitSharedChange();
    return null;
  }

  if (!response.ok) {
    throw new Error(await readError(response, "Unable to check wishlist state."));
  }

  const payload = (await response.json()) as WishlistStateResponse;
  const wishlisted = payload.data?.wishlisted === true;
  const next = new Set(wishlistProductIds);

  if (wishlisted) next.add(productId);
  else next.delete(productId);

  wishlistProductIds = next;
  emitSharedChange();
  return wishlisted;
}

function setSharedProductState(productId: string, wishlisted: boolean) {
  const next = new Set(wishlistProductIds);

  if (wishlisted) next.add(productId);
  else next.delete(productId);

  wishlistProductIds = next;
  sharedState = "ready";
  emitSharedChange();

  if (typeof window !== "undefined") {
    window.dispatchEvent(
      new CustomEvent(WISHLIST_UPDATED_EVENT, {
        detail: { productId, wishlisted },
      }),
    );
  }
}

function currentPageForSignIn() {
  if (typeof window === "undefined") return "/";
  return `${window.location.pathname}${window.location.search}${window.location.hash}`;
}

export function ProductWishlistButton({
  productId,
  productName,
  placement = "catalog",
}: Props) {
  const router = useRouter();
  const {
  isAuthenticated,
  isReady,
} = useCustomerSession();
  const [busy, setBusy] = useState(false);
  const [feedback, setFeedback] = useState<Feedback>("idle");

  // The shared external store means 50+ visible cards still make one wishlist
  // list request, and the same product updates everywhere on the current page.
  useSyncExternalStore(subscribeShared, getSharedRevision, getServerRevision);

  useEffect(() => {
  if (!isReady) {
    return;
  }

  /*
   * Guest product cards do not touch
   * authenticated wishlist APIs.
   */
  if (
    !isAuthenticated
  ) {
    if (
      sharedState !==
        "signed-out" ||
      wishlistProductIds.size >
        0
    ) {
      wishlistProductIds =
        new Set();

      sharedState =
        "signed-out";

      emitSharedChange();
    }

    return;
  }

  /*
   * Customer has since signed in.
   */
  if (
    sharedState ===
    "signed-out"
  ) {
    sharedState =
      "idle";

    emitSharedChange();
  }

  /*
   * For an authenticated customer this
   * single shared request is needed to
   * show saved-heart state accurately.
   *
   * It is shared across every product card,
   * not one request per product.
   */
  void ensureWishlistLoaded();
}, [
  isAuthenticated,
  isReady,
]);

  const wishlisted = wishlistProductIds.has(productId);
  const state = busy ? "busy" : feedback === "error" ? "error" : wishlisted ? "saved" : "idle";
  const actionLabel = wishlisted
    ? `Remove ${productName} from wishlist`
    : `Add ${productName} to wishlist`;

  function goToSignIn() {
    const next = encodeURIComponent(currentPageForSignIn());
    router.push(`/account/sign-in?next=${next}`);
  }

  async function toggleWishlist() {
    if (
  busy ||
  !isReady
) {
  return;
}

if (
  !isAuthenticated
) {
  goToSignIn();
  return;
}

    setBusy(true);
    setFeedback("idle");

    try {
      const current = await resolveSingleProductState(productId);

      if (current === null) {
        goToSignIn();
        return;
      }

      const nextWishlisted = !current;
      const response = await fetch(
        `/api/storefront/account/wishlist/${encodeURIComponent(productId)}`,
        {
          method: nextWishlisted ? "PUT" : "DELETE",
          cache: "no-store",
        },
      );

      if (response.status === 401) {
        wishlistProductIds = new Set();
        sharedState = "signed-out";
        emitSharedChange();
        goToSignIn();
        return;
      }

      if (!response.ok) {
        throw new Error(
          await readError(
            response,
            nextWishlisted
              ? "Unable to save this product."
              : "Unable to remove this saved product.",
          ),
        );
      }

      setSharedProductState(productId, nextWishlisted);
    } catch {
      setFeedback("error");
      window.setTimeout(() => setFeedback("idle"), 1800);
    } finally {
      setBusy(false);
    }
  }

  return (
    <button
      type="button"
      className={`${styles.button} ${styles[placement]}`}
      data-state={state}
      data-saved={wishlisted ? "true" : "false"}
      aria-label={actionLabel}
      aria-pressed={wishlisted}
      title={actionLabel}
      disabled={busy}
      onClick={() => void toggleWishlist()}
    >
      <span className={styles.visual} aria-hidden="true">
        <svg className={styles.heart} viewBox="0 0 24 24">
          <path d="M12 20.4S4.1 15.75 2.45 10.5C1.28 6.8 3.42 3.7 6.8 3.7c2.12 0 3.82 1.12 5.2 2.82 1.38-1.7 3.08-2.82 5.2-2.82 3.38 0 5.52 3.1 4.35 6.8C19.9 15.75 12 20.4 12 20.4Z" />
        </svg>
      </span>
      <span className={styles.live} aria-live="polite">
        {feedback === "error"
          ? "Wishlist update failed"
          : wishlisted
            ? "Saved"
            : ""}
      </span>
    </button>
  );
}
