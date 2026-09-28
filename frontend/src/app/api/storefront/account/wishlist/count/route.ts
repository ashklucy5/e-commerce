// ENE_WISHLIST_COUNT_BFF_V1
// Location: src/app/api/storefront/account/wishlist/count/route.ts

import { accountCommerceFetch } from "@/lib/account/request";
import { accountResultResponse } from "@/lib/account/response";
import { accountRouteErrorResponse } from "@/lib/account/route-error";

export async function GET() {
  try {
    const result = await accountCommerceFetch<unknown>(
      "/api/v1/wishlist/count",
      {
        cache: "no-store",
      },
    );

    return accountResultResponse(result);
  } catch (error) {
    return accountRouteErrorResponse(error, "Unable to load wishlist count.");
  }
}
