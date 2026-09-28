// Location: src/app/api/storefront/order/[orderId]/payment/initiate/route.ts

import { accountCommerceFetch } from "@/lib/account/request";
import { accountResultResponse } from "@/lib/account/response";
import { accountRouteErrorResponse } from "@/lib/account/route-error";

type RouteContext = {
  params: Promise<{
    orderId: string;
  }>;
};

export async function POST(
  _request: Request,
  context: RouteContext,
) {
  const { orderId } = await context.params;

  try {
    const result = await accountCommerceFetch<unknown>(
      `/api/v1/orders/${encodeURIComponent(orderId)}/payment/initiate`,
      {
        method: "POST",
        cache: "no-store",
      },
    );

    return accountResultResponse(result);
  } catch (error) {
    return accountRouteErrorResponse(
      error,
      "Unable to start payment.",
    );
  }
}
