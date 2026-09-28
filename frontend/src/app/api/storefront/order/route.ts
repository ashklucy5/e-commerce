// Location: src/app/api/storefront/order/route.ts

import { NextResponse } from "next/server";
import { cookies } from "next/headers";

import { accountCommerceFetch } from "@/lib/account/request";
import { setAccountAuthCookies } from "@/lib/account/session";
import type { PlaceOrderResponse } from "@/lib/api/contracts/commerce";

import {
  CART_COOKIE,
  CHECKOUT_COOKIE,
  CHECKOUT_SOURCE_COOKIE,
  ORDER_ACCESS_COOKIE,
  ORDER_COOKIE,
  orderAccessCookieOptions,
  orderCookieOptions,
} from "@/lib/commerce/session";

import { accountRouteErrorResponse } from "@/lib/account/route-error";

export async function POST() {
  const store =
    await cookies();

  const checkoutKey =
    store.get(
      CHECKOUT_COOKIE,
    )?.value ?? "";

  const source =
    store.get(
      CHECKOUT_SOURCE_COOKIE,
    )?.value ?? "";

  if (!checkoutKey) {
    return NextResponse.json(
      {
        error: {
          code:
            "CHECKOUT_NOT_FOUND",

          message:
            "Checkout not found.",
        },
      },
      {
        status: 404,
      },
    );
  }

  try {
    const result =
      await accountCommerceFetch<PlaceOrderResponse>(
        `/api/v1/checkouts/${encodeURIComponent(
          checkoutKey,
        )}/place-order`,
        {
          method: "POST",
          cache: "no-store",
        },
      );

    const response =
      NextResponse.json(
        result.payload,
        {
          status:
            result.payload.meta
              .created
              ? 201
              : 200,
        },
      );

    if (
      result.rotatedTokens
    ) {
      setAccountAuthCookies(
        response,
        result.rotatedTokens,
      );
    }

    response.cookies.set(
      ORDER_COOKIE,
      result.payload.data.id,
      orderCookieOptions,
    );

    response.cookies.set(
      ORDER_ACCESS_COOKIE,
      checkoutKey,
      orderAccessCookieOptions,
    );

    response.cookies.delete(
      CHECKOUT_COOKIE,
    );

    response.cookies.delete(
      CHECKOUT_SOURCE_COOKIE,
    );

    if (
      source === "cart"
    ) {
      response.cookies.delete(
        CART_COOKIE,
      );
    }

    return response;
  } catch (error) {
    return accountRouteErrorResponse(
      error,
      "Unable to place order.",
    );
  }
}