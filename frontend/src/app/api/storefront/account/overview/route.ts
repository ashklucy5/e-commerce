// Location: src/app/api/storefront/account/overview/route.ts

import { accountCommerceFetch } from "@/lib/account/request";
import { accountResultResponse } from "@/lib/account/response";
import { accountRouteErrorResponse } from "@/lib/account/route-error";

export async function GET() {
  try {
    const result = await accountCommerceFetch<{ data: Record<string, unknown> }>(
      "/api/v1/customers/me/overview",
      { cache: "no-store" },
    );

    return accountResultResponse(result);
  } catch (error) {
    return accountRouteErrorResponse(error, "Unable to load account overview.");
  }
}
