// Location: src/app/api/storefront/account/orders/[orderId]/tracking/route.ts

import { accountCommerceFetch } from "@/lib/account/request";
import { accountResultResponse } from "@/lib/account/response";
import { accountRouteErrorResponse } from "@/lib/account/route-error";
import type { AccountOrderTrackingResponse } from "@/lib/api/contracts/order-account";

type RouteContext = {
  params: Promise<{ orderId: string }>;
};

export async function GET(_request: Request, context: RouteContext) {
  const { orderId } = await context.params;

  try {
    const result = await accountCommerceFetch<AccountOrderTrackingResponse>(
      `/api/v1/orders/${encodeURIComponent(orderId)}/tracking`,
      { cache: "no-store" },
    );

    return accountResultResponse(result);
  } catch (error) {
    return accountRouteErrorResponse(error, "Unable to load delivery tracking.");
  }
}
