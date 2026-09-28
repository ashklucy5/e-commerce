// Location: src/app/api/storefront/account/addresses/[id]/route.ts

import { NextResponse } from "next/server";

import type {
  AccountAddress,
  AccountDataResponse,
  UpdateAccountAddressRequest,
} from "@/lib/api/contracts/account";
import { accountCommerceFetch } from "@/lib/account/request";
import { accountResultResponse } from "@/lib/account/response";
import { accountRouteErrorResponse } from "@/lib/account/route-error";

type Context = {
  params: Promise<{ id: string }>;
};

export async function PATCH(request: Request, context: Context) {
  let body: UpdateAccountAddressRequest;

  try {
    body = (await request.json()) as UpdateAccountAddressRequest;
  } catch {
    return NextResponse.json(
      { error: { code: "INVALID_REQUEST", message: "Invalid address update." } },
      { status: 400 },
    );
  }

  try {
    const { id } = await context.params;
    const result = await accountCommerceFetch<AccountDataResponse<AccountAddress>>(
      `/api/v1/customers/me/addresses/${encodeURIComponent(id)}`,
      {
        method: "PATCH",
        cache: "no-store",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      },
    );

    return accountResultResponse(result);
  } catch (error) {
    return accountRouteErrorResponse(error, "Unable to update address.");
  }
}

export async function DELETE(_request: Request, context: Context) {
  try {
    const { id } = await context.params;
    const result = await accountCommerceFetch<unknown>(
      `/api/v1/customers/me/addresses/${encodeURIComponent(id)}`,
      {
        method: "DELETE",
        cache: "no-store",
      },
    );

    return accountResultResponse(result);
  } catch (error) {
    return accountRouteErrorResponse(error, "Unable to remove address.");
  }
}
