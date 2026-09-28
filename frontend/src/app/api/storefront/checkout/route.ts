// Location: src/app/api/storefront/checkout/route.ts
import { NextResponse } from "next/server";
import { cookies } from "next/headers";

import { accountCommerceFetch } from "@/lib/account/request";
import { setAccountAuthCookies } from "@/lib/account/session";
import { CommerceApiError } from "@/lib/api/error";
import type { ApiData, CheckoutSession } from "@/lib/api/contracts/commerce";
import {
  CART_COOKIE,
  CHECKOUT_COOKIE,
  CHECKOUT_SOURCE_COOKIE,
  checkoutCookieOptions,
} from "@/lib/commerce/session";
import { accountRouteErrorResponse } from "@/lib/account/route-error";

export async function POST() {
  const store = await cookies();
  const cartKey = store.get(CART_COOKIE)?.value ?? "";

  if (!cartKey) {
    return NextResponse.json(
      { error: { code: "CART_NOT_FOUND", message: "Your cart is empty." } },
      { status: 404 },
    );
  }

  try {
    const result = await accountCommerceFetch<ApiData<CheckoutSession>>(
      "/api/v1/checkouts",
      {
        method: "POST",
        cache: "no-store",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ cart_key: cartKey, promotion_code: null }),
      },
    );

    const response = NextResponse.json(result.payload);

    if (result.rotatedTokens) {
      setAccountAuthCookies(response, result.rotatedTokens);
    }

    response.cookies.set(
      CHECKOUT_COOKIE,
      result.payload.data.checkout_key,
      checkoutCookieOptions,
    );
    response.cookies.set(CHECKOUT_SOURCE_COOKIE, "cart", checkoutCookieOptions);

    return response;
  } catch (error) {
    return accountRouteErrorResponse(error, "Unable to start checkout.");
  }
}

export async function PATCH(request: Request) {
  const store = await cookies();
  const checkoutKey = store.get(CHECKOUT_COOKIE)?.value ?? "";

  if (!checkoutKey) {
    return NextResponse.json(
      { error: { code: "CHECKOUT_NOT_FOUND", message: "Checkout not found." } },
      { status: 404 },
    );
  }

  let payload: Record<string, unknown>;

  try {
    payload = await request.json();
  } catch {
    return NextResponse.json(
      { error: { code: "INVALID_REQUEST", message: "Invalid checkout update." } },
      { status: 400 },
    );
  }

  try {
    const result = await accountCommerceFetch<ApiData<CheckoutSession>>(
      `/api/v1/checkouts/${encodeURIComponent(checkoutKey)}`,
      {
        method: "PATCH",
        cache: "no-store",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      },
    );

    const response = NextResponse.json(result.payload);
    if (result.rotatedTokens) setAccountAuthCookies(response, result.rotatedTokens);
    return response;
  } catch (error) {
    if (error instanceof CommerceApiError && error.status === 401) {
      return accountRouteErrorResponse(error, "Sign in again to continue checkout.");
    }
    return accountRouteErrorResponse(error, "Unable to update checkout.");
  }
}
