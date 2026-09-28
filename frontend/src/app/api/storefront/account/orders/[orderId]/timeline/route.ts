// Location: src/app/api/storefront/account/orders/[orderId]/timeline/route.ts

import { accountCommerceFetch } from "@/lib/account/request";
import { accountResultResponse } from "@/lib/account/response";
import { accountRouteErrorResponse } from "@/lib/account/route-error";
import type { AccountOrderTimelineResponse } from "@/lib/api/contracts/order-account";

type RouteContext = {
  params: Promise<{ orderId: string }>;
};

export async function GET(_request: Request, context: RouteContext) {
  const { orderId } = await context.params;

  try {
    const result = await accountCommerceFetch<AccountOrderTimelineResponse>(
      `/api/v1/orders/${encodeURIComponent(orderId)}/timeline`,
      { cache: "no-store" },
    );

    return accountResultResponse(result);
  } catch (error) {
    return accountRouteErrorResponse(error, "Unable to load the order timeline.");
  }
}
