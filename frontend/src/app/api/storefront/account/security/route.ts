// Location: src/app/api/storefront/account/security/route.ts

import type {
  AccountDataResponse,
  AccountSecurityOverview,
} from "@/lib/api/contracts/account";
import { accountCommerceFetch } from "@/lib/account/request";
import { accountResultResponse } from "@/lib/account/response";
import { accountRouteErrorResponse } from "@/lib/account/route-error";

export async function GET() {
  try {
    const result = await accountCommerceFetch<
      AccountDataResponse<AccountSecurityOverview>
    >("/api/v1/customers/me/security", { cache: "no-store" });

    return accountResultResponse(result);
  } catch (error) {
    return accountRouteErrorResponse(error, "Unable to load account security.");
  }
}
