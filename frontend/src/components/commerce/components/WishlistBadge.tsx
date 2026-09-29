"use client";

import {
  useCallback,
  useEffect,
  useState,
} from "react";

import {
  useCustomerSession,
} from "@/lib/account/use-customer-session";

const WISHLIST_UPDATED_EVENT =
  "ene-wishlist-updated";

type Props = {
  className?: string;
};

type WishlistCountResponse = {
  data?: {
    total?: number;
  };
};

function normalizeCount(
  value: unknown,
) {
  const count =
    Number(value);

  if (
    !Number.isFinite(count) ||
    count <= 0
  ) {
    return 0;
  }

  return Math.floor(
    count,
  );
}

export function WishlistBadge({
  className = "",
}: Props) {
  const {
    isAuthenticated,
    isReady,
  } =
    useCustomerSession();

  const [
    count,
    setCount,
  ] =
    useState<number | null>(
      null,
    );

  const refresh =
    useCallback(
      async () => {
        /*
         * Never probe protected wishlist
         * endpoints for anonymous visitors.
         */
        if (
          !isAuthenticated
        ) {
          setCount(0);
          return;
        }

        try {
          const response =
            await fetch(
              "/api/storefront/account/wishlist/count",
              {
                cache:
                  "no-store",
              },
            );

          if (
            response.status ===
            401
          ) {
            setCount(0);

            window.dispatchEvent(
              new Event(
                "customer-auth-changed",
              ),
            );

            return;
          }

          if (
            !response.ok
          ) {
            return;
          }

          const payload =
            (await response.json()) as
              WishlistCountResponse;

          setCount(
            normalizeCount(
              payload.data
                ?.total,
            ),
          );
        } catch {
          /*
           * Header badge is non-critical.
           */
        }
      },
      [
        isAuthenticated,
      ],
    );

  useEffect(() => {
    if (!isReady) {
      return;
    }

    if (
      !isAuthenticated
    ) {
      setCount(0);
      return;
    }

    void refresh();

    function handleWishlistUpdated() {
      void refresh();
    }

    window.addEventListener(
      WISHLIST_UPDATED_EVENT,
      handleWishlistUpdated,
    );

    return () => {
      window.removeEventListener(
        WISHLIST_UPDATED_EVENT,
        handleWishlistUpdated,
      );
    };
  }, [
    isAuthenticated,
    isReady,
    refresh,
  ]);

  if (
    count === null ||
    count === 0
  ) {
    return null;
  }

  return (
    <span
      className={
        className
      }
      aria-label={`${count} ${
        count === 1
          ? "product"
          : "products"
      } in wishlist`}
    >
      {count > 99
        ? "99+"
        : count}
    </span>
  );
}