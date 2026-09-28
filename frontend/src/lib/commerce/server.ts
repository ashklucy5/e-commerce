import "server-only";

import {
  commerceFetch,
} from "@/lib/api/server";

import {
  CommerceApiError,
} from "@/lib/api/error";

import type {
  ApiData,
  Cart,
  CheckoutSession,
  DeliveryOption,
  Order,
  PaymentOption,
} from "@/lib/api/contracts/commerce";

import {
  getCartKey,
  getCheckoutKey,
  getOrderAccessKey,
  getOrderID,
} from "./session";

export async function createBackendCart() {
  const response =
    await commerceFetch<
      ApiData<Cart>
    >(
      "/api/v1/carts",
      {
        method:
          "POST",

        cache:
          "no-store",
      },
    );

  return response.data;
}

export async function getBackendCart(
  cartKey: string,
) {
  const response =
    await commerceFetch<
      ApiData<Cart>
    >(
      `/api/v1/carts/${encodeURIComponent(
        cartKey,
      )}`,
      {
        cache:
          "no-store",
      },
    );

  return response.data;
}

export async function getCurrentCart() {
  const cartKey =
    await getCartKey();

  if (
    !cartKey
  ) {
    return null;
  }

  try {
    return await getBackendCart(
      cartKey,
    );
  } catch (
    error
  ) {
    if (
      error instanceof
        CommerceApiError &&
      [
        "CART_NOT_FOUND",
        "CART_EXPIRED",
        "CART_INACTIVE",
      ].includes(
        error.code ??
          "",
      )
    ) {
      return null;
    }

    throw error;
  }
}

export async function getBackendCheckout(
  checkoutKey: string,
) {
  const response =
    await commerceFetch<
      ApiData<CheckoutSession>
    >(
      `/api/v1/checkouts/${encodeURIComponent(
        checkoutKey,
      )}`,
      {
        cache:
          "no-store",
      },
    );

  return response.data;
}

export async function getCurrentCheckout() {
  const checkoutKey =
    await getCheckoutKey();

  if (
    !checkoutKey
  ) {
    return null;
  }

  try {
    return await getBackendCheckout(
      checkoutKey,
    );
  } catch (
    error
  ) {
    if (
      error instanceof
        CommerceApiError &&
      [
        "CHECKOUT_NOT_FOUND",
        "CHECKOUT_NOT_ACTIVE",
        "EXPIRED",
      ].includes(
        error.code ??
          "",
      )
    ) {
      return null;
    }

    throw error;
  }
}

export async function getCheckoutDeliveryOptions(
  checkoutKey: string,
) {
  const response =
    await commerceFetch<
      ApiData<
        DeliveryOption[]
      >
    >(
      `/api/v1/checkouts/${encodeURIComponent(
        checkoutKey,
      )}/delivery-methods`,
      {
        cache:
          "no-store",
      },
    );

  return response.data;
}

export async function getCheckoutPaymentOptions(
  checkoutKey: string,
) {
  const query =
    new URLSearchParams({
      checkout_key:
        checkoutKey,
    });

  const response =
    await commerceFetch<
      ApiData<
        PaymentOption[]
      >
    >(
      `/api/v1/checkout-options/payment-methods?${query.toString()}`,
      {
        cache:
          "no-store",
      },
    );

  return response.data;
}

/*
 * Load an order from Go.
 *
 * For authenticated customer orders, the normal
 * authentication context remains authoritative.
 *
 * For guest orders, customeraccess.Guard requires
 * X-Checkout-Key as proof of possession.
 */
export async function getBackendOrder(
  orderID: string,
  orderAccessKey = "",
) {
  const response =
    await commerceFetch<
      ApiData<Order>
    >(
      `/api/v1/orders/${encodeURIComponent(
        orderID,
      )}`,
      {
        cache:
          "no-store",

        headers:
          orderAccessKey
            ? {
                "X-Checkout-Key":
                  orderAccessKey,
              }
            : undefined,
      },
    );

  return response.data;
}

export async function getCurrentOrder() {
  const [
    orderID,
    orderAccessKey,
  ] =
    await Promise.all([
      getOrderID(),

      getOrderAccessKey(),
    ]);

  if (
    !orderID
  ) {
    return null;
  }

  try {
    return await getBackendOrder(
      orderID,
      orderAccessKey,
    );
  } catch (
    error
  ) {
    if (
      error instanceof
        CommerceApiError &&
      error.code ===
        "ORDER_NOT_FOUND"
    ) {
      return null;
    }

    throw error;
  }
}