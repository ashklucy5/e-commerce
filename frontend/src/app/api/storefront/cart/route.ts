// Location: src/app/api/storefront/cart/items/route.ts

import { NextResponse } from "next/server";
import { cookies } from "next/headers";

import { commerceFetch } from "@/lib/api/server";
import { CommerceApiError } from "@/lib/api/error";
import type { ApiData, Cart } from "@/lib/api/contracts/commerce";
import {
  createBackendCart,
  getBackendCart,
  getCurrentCart,
} from "@/lib/commerce/server";
import {
  CART_COOKIE,
  cartCookieOptions,
} from "@/lib/commerce/session";
import { routeErrorResponse } from "@/lib/commerce/route-error";

async function addItem(
  cartKey: string,
  variantID: string,
  quantity: number,
) {
  const response = await commerceFetch<ApiData<Cart>>(
    `/api/v1/carts/${encodeURIComponent(cartKey)}/items`,
    {
      method: "POST",
      cache: "no-store",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        variant_id: variantID,
        quantity,
      }),
    },
  );

  return response.data;
}

async function updateItemQuantity(
  cartKey: string,
  itemID: string,
  quantity: number,
) {
  const response = await commerceFetch<ApiData<Cart>>(
    `/api/v1/carts/${encodeURIComponent(cartKey)}/items/${encodeURIComponent(itemID)}`,
    {
      method: "PATCH",
      cache: "no-store",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ quantity }),
    },
  );

  return response.data;
}

async function addOrMergeItem(
  cartKey: string,
  variantID: string,
  quantityToAdd: number,
) {
  const currentCart = await getBackendCart(cartKey);
  const existingItem = currentCart.items.find(
    (item) => item.variant_id === variantID,
  );

  if (!existingItem) {
    return addItem(cartKey, variantID, quantityToAdd);
  }

  const nextQuantity = existingItem.quantity + quantityToAdd;

  if (!Number.isSafeInteger(nextQuantity) || nextQuantity <= 0) {
    throw new Error("Invalid merged cart quantity.");
  }

  return updateItemQuantity(
    cartKey,
    existingItem.id,
    nextQuantity,
  );
}

export async function GET() {
  try {
    const cart =
      await getCurrentCart();

    return NextResponse.json({
      data: cart,
    });
  } catch (error) {
    return routeErrorResponse(
      error,
      "Unable to load cart.",
    );
  }
}

export async function POST(request: Request) {
  let payload: { variant_id?: string; quantity?: number };

  try {
    payload = await request.json();
  } catch {
    return NextResponse.json(
      {
        error: {
          code: "INVALID_REQUEST",
          message: "Invalid cart item request.",
        },
      },
      { status: 400 },
    );
  }

  const variantID = String(payload.variant_id ?? "").trim();
  const quantity = Number(payload.quantity);

  if (!variantID || !Number.isInteger(quantity) || quantity <= 0) {
    return NextResponse.json(
      {
        error: {
          code: "INVALID_REQUEST",
          message: "Choose a valid variant and quantity.",
        },
      },
      { status: 400 },
    );
  }

  try {
    const store = await cookies();
    let cartKey = store.get(CART_COOKIE)?.value ?? "";

    if (!cartKey) {
      const cart = await createBackendCart();
      cartKey = cart.cart_key;
    }

    let cart: Cart;

    try {
      cart = await addOrMergeItem(cartKey, variantID, quantity);
    } catch (error) {
      const stale =
        error instanceof CommerceApiError &&
        ["CART_NOT_FOUND", "CART_EXPIRED", "CART_INACTIVE"].includes(
          error.code ?? "",
        );

      if (!stale) {
        throw error;
      }

      const replacement = await createBackendCart();
      cartKey = replacement.cart_key;
      cart = await addItem(cartKey, variantID, quantity);
    }

    const response = NextResponse.json({ data: cart });
    response.cookies.set(CART_COOKIE, cartKey, cartCookieOptions);
    return response;
  } catch (error) {
    return routeErrorResponse(error, "Unable to add product to cart.");
  }
}
