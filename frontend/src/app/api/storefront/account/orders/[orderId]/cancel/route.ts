// Location: src/app/api/storefront/account/orders/[orderId]/cancel/route.ts

import { NextResponse } from "next/server";

import { accountCommerceFetch } from "@/lib/account/request";
import { accountResultResponse } from "@/lib/account/response";
import { accountRouteErrorResponse } from "@/lib/account/route-error";

type RouteContext = {
  params: Promise<{ orderId: string }>;
};

type CancelBody = {
  reason?: unknown;
};

export async function POST(request: Request, context: RouteContext) {
  const { orderId } = await context.params;

  let body: CancelBody;
  try {
    body = (await request.json()) as CancelBody;
  } catch {
    return NextResponse.json(
      { error: { code: "INVALID_CANCELLATION", message: "Choose a cancellation reason." } },
      { status: 400 },
    );
  }

  const reason = typeof body.reason === "string" ? body.reason.trim() : "";
  if (!reason || reason.length > 500) {
    return NextResponse.json(
      {
        error: {
          code: "INVALID_CANCELLATION",
          message: "Choose a cancellation reason of 500 characters or fewer.",
        },
      },
      { status: 400 },
    );
  }

  try {
    const result = await accountCommerceFetch<unknown>(
      `/api/v1/orders/${encodeURIComponent(orderId)}/cancel`,
      {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ reason }),
        cache: "no-store",
      },
    );

    return accountResultResponse(result);
  } catch (error) {
    return accountRouteErrorResponse(
      error,
      "This order can no longer be cancelled before delivery.",
    );
  }
}

export function OPTIONS() {
  return new NextResponse(null, { status: 204 });
}
