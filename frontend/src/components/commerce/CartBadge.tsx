"use client";

import {
  useEffect,
  useSyncExternalStore,
} from "react";

import type {
  Cart,
} from "@/lib/api/contracts/commerce";

export const CART_UPDATED_EVENT =
  "ene-cart-updated";

/*
 * Shared cart-count store.
 *
 * CartBadge only reads this external store.
 * It no longer mirrors the store into local
 * React state from inside an effect.
 */

let cachedCount:
  number | null = null;

let pendingRequest:
  Promise<number | null> |
  null = null;

const listeners =
  new Set<() => void>();

function emitChange() {
  listeners.forEach(
    (listener) => {
      listener();
    },
  );
}

function setCachedCount(
  nextCount: number,
) {
  if (
    cachedCount ===
    nextCount
  ) {
    return;
  }

  cachedCount =
    nextCount;

  emitChange();
}

async function fetchCartCount() {
  if (pendingRequest) {
    return pendingRequest;
  }

  pendingRequest =
    (async () => {
      try {
        const response =
          await fetch(
            "/api/storefront/cart",
            {
              cache:
                "no-store",
            },
          );

        if (
          !response.ok
        ) {
          return null;
        }

        const payload =
          (await response.json()) as {
            data:
              | Cart
              | null;
          };

        const nextCount =
          payload.data
            ?.totals
            .item_count ??
          0;

        setCachedCount(
          nextCount,
        );

        return nextCount;
      } catch {
        return null;
      } finally {
        pendingRequest =
          null;
      }
    })();

  return pendingRequest;
}

function getSnapshot() {
  return cachedCount;
}

/*
 * Server rendering must always begin from a
 * deterministic value so hydration remains
 * stable.
 */
function getServerSnapshot() {
  return null;
}

function subscribe(
  listener: () => void,
) {
  listeners.add(
    listener,
  );

  if (
    typeof window ===
    "undefined"
  ) {
    return () => {
      listeners.delete(
        listener,
      );
    };
  }

  const handleCartUpdate = (
    event: Event,
  ) => {
    const customEvent =
      event as CustomEvent<
        Cart | null
      >;

    if (
      customEvent.detail
    ) {
      setCachedCount(
        customEvent.detail
          .totals
          .item_count,
      );

      return;
    }

    /*
     * No cart payload means another storefront
     * action changed the cart and consumers
     * should refresh from the authoritative API.
     */
    void fetchCartCount();
  };

  window.addEventListener(
    CART_UPDATED_EVENT,
    handleCartUpdate,
  );

  return () => {
    listeners.delete(
      listener,
    );

    window.removeEventListener(
      CART_UPDATED_EVENT,
      handleCartUpdate,
    );
  };
}

export function notifyCartUpdated(
  cart?:
    | Cart
    | null,
) {
  if (
    typeof window ===
    "undefined"
  ) {
    return;
  }

  /*
   * Update the shared store immediately when
   * the caller already has the authoritative
   * cart response.
   */
  if (cart) {
    setCachedCount(
      cart.totals
        .item_count,
    );
  }

  /*
   * Keep the existing event contract so other
   * storefront components remain compatible.
   */
  window.dispatchEvent(
    new CustomEvent(
      CART_UPDATED_EVENT,
      {
        detail:
          cart ?? null,
      },
    ),
  );
}

type Props = {
  className?: string;
};

export function CartBadge({
  className = "",
}: Props) {
  const count =
    useSyncExternalStore(
      subscribe,
      getSnapshot,
      getServerSnapshot,
    );

  /*
   * Fetch once after the badge enters the
   * browser if no shared cart count exists yet.
   *
   * This effect synchronizes React with an
   * external API/store. There is no direct or
   * indirect React setState call here.
   */
  useEffect(() => {
    if (
      cachedCount ===
      null
    ) {
      void fetchCartCount();
    }
  }, []);

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
          ? "item"
          : "items"
      } in cart`}
    >
      {count > 99
        ? "99+"
        : count}
    </span>
  );
}