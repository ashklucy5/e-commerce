// ENE_WISHLIST_HEADER_BADGE_V1
// Location: src/components/commerce/components/WishlistBadge.tsx

"use client";

import { useCallback, useEffect, useState } from "react";

const WISHLIST_UPDATED_EVENT = "ene-wishlist-updated";

type Props = {
  className?: string;
};

type WishlistCountResponse = {
  data?: {
    total?: number;
  };
};

function normalizeCount(value: unknown) {
  const count = Number(value);

  if (!Number.isFinite(count) || count <= 0) {
    return 0;
  }

  return Math.floor(count);
}

export function WishlistBadge({ className = "" }: Props) {
  const [count, setCount] = useState<number | null>(null);

  const refresh = useCallback(async () => {
    try {
      const response = await fetch("/api/storefront/account/wishlist/count", {
        cache: "no-store",
      });

      if (response.status === 401) {
        setCount(0);
        return;
      }

      if (!response.ok) {
        return;
      }

      const payload = (await response.json()) as WishlistCountResponse;
      setCount(normalizeCount(payload.data?.total));
    } catch {
      // The badge is non-critical. Wishlist actions remain usable if this fails.
    }
  }, []);

  useEffect(() => {
    // Defer the initial state-producing request so it does not synchronously
    // cascade from the effect body under the project's React lint rules.
    const initialTimer = window.setTimeout(() => {
      void refresh();
    }, 0);

    const handleWishlistUpdated = () => {
      void refresh();
    };

    window.addEventListener(WISHLIST_UPDATED_EVENT, handleWishlistUpdated);
    window.addEventListener("focus", handleWishlistUpdated);

    return () => {
      window.clearTimeout(initialTimer);
      window.removeEventListener(WISHLIST_UPDATED_EVENT, handleWishlistUpdated);
      window.removeEventListener("focus", handleWishlistUpdated);
    };
  }, [refresh]);

  if (count === null || count === 0) {
    return null;
  }

  return (
    <span
      className={className}
      aria-label={`${count} ${count === 1 ? "product" : "products"} in wishlist`}
    >
      {count > 99 ? "99+" : count}
    </span>
  );
}
