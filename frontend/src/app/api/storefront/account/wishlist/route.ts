// Location: src/app/api/storefront/account/wishlist/route.ts

import type { AccountWishlistResponse } from "@/lib/api/contracts/account";
import { accountCommerceFetch } from "@/lib/account/request";
import { accountResultResponse } from "@/lib/account/response";
import { accountRouteErrorResponse } from "@/lib/account/route-error";

export async function GET(request: Request) {
  const url = new URL(request.url);
  const page = url.searchParams.get("page")?.trim() || "1";
  const limit = url.searchParams.get("limit")?.trim() || "20";
  const query = new URLSearchParams({ page, limit });

  try {
    const result = await accountCommerceFetch<AccountWishlistResponse>(
      `/api/v1/wishlist?${query.toString()}`,
      { cache: "no-store" },
    );

    return accountResultResponse(result);
  } catch (error) {
    return accountRouteErrorResponse(error, "Unable to load your wishlist.");
  }
}
