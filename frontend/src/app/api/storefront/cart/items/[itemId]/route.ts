import { NextResponse } from "next/server";
import { cookies } from "next/headers";

import { commerceFetch } from "@/lib/api/server";
import type { ApiData, Cart } from "@/lib/api/contracts/commerce";
import { CART_COOKIE } from "@/lib/commerce/session";
import { routeErrorResponse } from "@/lib/commerce/route-error";

type Context = {
  params: Promise<{ itemId: string }>;
};

export async function PATCH(request: Request, context: Context) {
  const store = await cookies();
  const cartKey = store.get(CART_COOKIE)?.value ?? "";

  if (!cartKey) {
    return NextResponse.json(
      { error: { code: "CART_NOT_FOUND", message: "Cart not found." } },
      { status: 404 },
    );
  }

  let payload: { quantity?: number };

  try {
    payload = await request.json();
  } catch {
    return NextResponse.json(
      { error: { code: "INVALID_REQUEST", message: "Invalid quantity." } },
      { status: 400 },
    );
  }

  const quantity = Number(payload.quantity);

  if (!Number.isInteger(quantity) || quantity <= 0) {
    return NextResponse.json(
      { error: { code: "INVALID_REQUEST", message: "Invalid quantity." } },
      { status: 400 },
    );
  }

  try {
    const { itemId } = await context.params;
    const response = await commerceFetch<ApiData<Cart>>(
      `/api/v1/carts/${encodeURIComponent(cartKey)}/items/${encodeURIComponent(itemId)}`,
      {
        method: "PATCH",
        cache: "no-store",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ quantity }),
      },
    );

    return NextResponse.json(response);
  } catch (error) {
    return routeErrorResponse(error, "Unable to update cart item.");
  }
}

export async function DELETE(_request: Request, context: Context) {
  const store = await cookies();
  const cartKey = store.get(CART_COOKIE)?.value ?? "";

  if (!cartKey) {
    return NextResponse.json(
      { error: { code: "CART_NOT_FOUND", message: "Cart not found." } },
      { status: 404 },
    );
  }

  try {
    const { itemId } = await context.params;
    const response = await commerceFetch<ApiData<Cart>>(
      `/api/v1/carts/${encodeURIComponent(cartKey)}/items/${encodeURIComponent(itemId)}`,
      {
        method: "DELETE",
        cache: "no-store",
      },
    );

    return NextResponse.json(response);
  } catch (error) {
    return routeErrorResponse(error, "Unable to remove cart item.");
  }
}
