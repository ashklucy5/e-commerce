// Location: src/app/api/storefront/account/security/sessions/route.ts

import type { AccountSessionsResponse } from "@/lib/api/contracts/account";
import { accountCommerceFetch } from "@/lib/account/request";
import { accountResultResponse } from "@/lib/account/response";
import { accountRouteErrorResponse } from "@/lib/account/route-error";

export async function GET() {
  try {
    const result = await accountCommerceFetch<AccountSessionsResponse>(
      "/api/v1/customers/me/sessions",
      { cache: "no-store" },
    );

    return accountResultResponse(result);
  } catch (error) {
    return accountRouteErrorResponse(error, "Unable to load account sessions.");
  }
}
