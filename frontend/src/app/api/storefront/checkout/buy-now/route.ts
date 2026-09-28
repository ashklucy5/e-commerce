// Location: src/app/api/storefront/checkout/buy-now/route.ts
import { NextResponse } from "next/server";

import { accountCommerceFetch } from "@/lib/account/request";
import {
  getCustomerAccessToken,
  getCustomerRefreshToken,
  setAccountAuthCookies,
} from "@/lib/account/session";
import { commerceFetch } from "@/lib/api/server";
import type { ApiData, Cart, CheckoutSession } from "@/lib/api/contracts/commerce";
import { createBackendCart } from "@/lib/commerce/server";
import {
  CHECKOUT_COOKIE,
  CHECKOUT_SOURCE_COOKIE,
  checkoutCookieOptions,
} from "@/lib/commerce/session";
import { accountRouteErrorResponse } from "@/lib/account/route-error";

type Body = { variant_id?: string; quantity?: number };

export async function POST(request: Request) {
  let body: Body;

  try {
    body = (await request.json()) as Body;
  } catch {
    return NextResponse.json(
      { error: { code: "INVALID_REQUEST", message: "Invalid Buy Now request." } },
      { status: 400 },
    );
  }

  const variantID = String(body.variant_id ?? "").trim();
  const quantity = Number(body.quantity);

  if (!variantID || !Number.isInteger(quantity) || quantity <= 0) {
    return NextResponse.json(
      { error: { code: "INVALID_REQUEST", message: "Choose a valid variant and quantity." } },
      { status: 400 },
    );
  }

  const [accessToken, refreshToken] = await Promise.all([
    getCustomerAccessToken(),
    getCustomerRefreshToken(),
  ]);

  if (!accessToken && !refreshToken) {
    return NextResponse.json(
      { error: { code: "UNAUTHORIZED", message: "Sign in to continue to checkout." } },
      { status: 401 },
    );
  }

  try {
    const cart = await createBackendCart();

    await commerceFetch<ApiData<Cart>>(
      `/api/v1/carts/${encodeURIComponent(cart.cart_key)}/items`,
      {
        method: "POST",
        cache: "no-store",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ variant_id: variantID, quantity }),
      },
    );

    const checkout = await accountCommerceFetch<ApiData<CheckoutSession>>(
      "/api/v1/checkouts",
      {
        method: "POST",
        cache: "no-store",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ cart_key: cart.cart_key, promotion_code: null }),
      },
    );

    const response = NextResponse.json(checkout.payload);
    if (checkout.rotatedTokens) setAccountAuthCookies(response, checkout.rotatedTokens);

    response.cookies.set(
      CHECKOUT_COOKIE,
      checkout.payload.data.checkout_key,
      checkoutCookieOptions,
    );
    response.cookies.set(CHECKOUT_SOURCE_COOKIE, "buy-now", checkoutCookieOptions);

    return response;
  } catch (error) {
    return accountRouteErrorResponse(error, "Unable to start Buy Now checkout.");
  }
}
