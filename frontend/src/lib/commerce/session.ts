import "server-only";

import {
  cookies,
} from "next/headers";

export const CART_COOKIE =
  "ene_dei_cart";

export const CHECKOUT_COOKIE =
  "ene_dei_checkout";

export const CHECKOUT_SOURCE_COOKIE =
  "ene_dei_checkout_source";

export const ORDER_COOKIE =
  "ene_dei_order";

/*
 * Secure possession credential for guest-order access.
 *
 * The Go customer-access guard requires the original
 * checkout key in X-Checkout-Key when a guest reads
 * their order.
 *
 * Keep this separate from CHECKOUT_COOKIE because
 * checkout itself is finished after order creation.
 */
export const ORDER_ACCESS_COOKIE =
  "ene_dei_order_access";

const secure =
  process.env.NODE_ENV ===
  "production";

export const cartCookieOptions = {
  httpOnly: true,

  secure,

  sameSite:
    "lax" as const,

  path: "/",

  maxAge:
    30 *
    24 *
    60 *
    60,
};

export const checkoutCookieOptions = {
  httpOnly: true,

  secure,

  sameSite:
    "lax" as const,

  path: "/",

  maxAge:
    24 *
    60 *
    60,
};

export const orderCookieOptions = {
  httpOnly: true,

  secure,

  sameSite:
    "lax" as const,

  path: "/",

  maxAge:
    24 *
    60 *
    60,
};

export const orderAccessCookieOptions = {
  httpOnly: true,

  secure,

  sameSite:
    "lax" as const,

  path: "/",

  maxAge:
    24 *
    60 *
    60,
};

export async function getCartKey() {
  const store =
    await cookies();

  return (
    store.get(
      CART_COOKIE,
    )?.value ??
    ""
  );
}

export async function getCheckoutKey() {
  const store =
    await cookies();

  return (
    store.get(
      CHECKOUT_COOKIE,
    )?.value ??
    ""
  );
}

export async function getOrderID() {
  const store =
    await cookies();

  return (
    store.get(
      ORDER_COOKIE,
    )?.value ??
    ""
  );
}

export async function getOrderAccessKey() {
  const store =
    await cookies();

  return (
    store.get(
      ORDER_ACCESS_COOKIE,
    )?.value ??
    ""
  );
}