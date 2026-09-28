// Location: src/app/api/storefront/account/security/sessions/revoke-others/route.ts

import { accountCommerceFetch } from "@/lib/account/request";
import { accountResultResponse } from "@/lib/account/response";
import { accountRouteErrorResponse } from "@/lib/account/route-error";

export async function POST() {
  try {
    const result = await accountCommerceFetch<unknown>(
      "/api/v1/customers/me/sessions/revoke-others",
      { method: "POST", cache: "no-store" },
    );

    return accountResultResponse(result);
  } catch (error) {
    return accountRouteErrorResponse(error, "Unable to sign out other devices.");
  }
}
