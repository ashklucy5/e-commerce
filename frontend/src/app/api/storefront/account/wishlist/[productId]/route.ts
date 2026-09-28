// ENE_WISHLIST_PRODUCT_BFF_V2
// Location: src/app/api/storefront/account/wishlist/[productId]/route.ts

import { accountCommerceFetch } from "@/lib/account/request";
import { accountResultResponse } from "@/lib/account/response";
import { accountRouteErrorResponse } from "@/lib/account/route-error";

type Context = {
  params: Promise<{ productId: string }>;
};

async function wishlistProductRequest(
  context: Context,
  method: "GET" | "PUT" | "DELETE",
  fallback: string,
) {
  try {
    const { productId } = await context.params;
    const normalizedProductId = productId.trim();

    if (!normalizedProductId) {
      throw new Error("Wishlist product ID is required.");
    }

    const result = await accountCommerceFetch<unknown>(
      `/api/v1/wishlist/${encodeURIComponent(normalizedProductId)}`,
      {
        method,
        cache: "no-store",
      },
    );

    return accountResultResponse(result);
  } catch (error) {
    return accountRouteErrorResponse(error, fallback);
  }
}

export async function GET(_request: Request, context: Context) {
  return wishlistProductRequest(
    context,
    "GET",
    "Unable to check wishlist state.",
  );
}

export async function PUT(_request: Request, context: Context) {
  return wishlistProductRequest(
    context,
    "PUT",
    "Unable to save wishlist item.",
  );
}

export async function DELETE(_request: Request, context: Context) {
  return wishlistProductRequest(
    context,
    "DELETE",
    "Unable to remove wishlist item.",
  );
}
