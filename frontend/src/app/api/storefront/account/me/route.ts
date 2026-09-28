// Location: src/app/api/storefront/account/me/route.ts

import type {
  AccountCustomer,
  AccountDataResponse,
} from "@/lib/api/contracts/account";
import { accountCommerceFetch } from "@/lib/account/request";
import { accountResultResponse } from "@/lib/account/response";
import { accountRouteErrorResponse } from "@/lib/account/route-error";

export async function GET() {
  try {
    const result = await accountCommerceFetch<AccountDataResponse<AccountCustomer>>(
      "/api/v1/customers/me",
      { cache: "no-store" },
    );

    return accountResultResponse(result);
  } catch (error) {
    return accountRouteErrorResponse(error, "Unable to load your account.");
  }
}
