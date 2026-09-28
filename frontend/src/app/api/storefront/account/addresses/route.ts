// Location: src/app/api/storefront/account/addresses/route.ts

import { NextResponse } from "next/server";

import type {
  AccountAddress,
  AccountDataResponse,
  CreateAccountAddressRequest,
} from "@/lib/api/contracts/account";
import { accountCommerceFetch } from "@/lib/account/request";
import { accountResultResponse } from "@/lib/account/response";
import { accountRouteErrorResponse } from "@/lib/account/route-error";

export async function GET() {
  try {
    const result = await accountCommerceFetch<AccountDataResponse<AccountAddress[]>>(
      "/api/v1/customers/me/addresses",
      { cache: "no-store" },
    );

    return accountResultResponse(result);
  } catch (error) {
    return accountRouteErrorResponse(error, "Unable to load saved addresses.");
  }
}

export async function POST(request: Request) {
  let body: CreateAccountAddressRequest;

  try {
    body = (await request.json()) as CreateAccountAddressRequest;
  } catch {
    return NextResponse.json(
      { error: { code: "INVALID_REQUEST", message: "Invalid address request." } },
      { status: 400 },
    );
  }

  try {
    const result = await accountCommerceFetch<AccountDataResponse<AccountAddress>>(
      "/api/v1/customers/me/addresses",
      {
        method: "POST",
        cache: "no-store",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      },
    );

    return accountResultResponse(result, { status: 201 });
  } catch (error) {
    return accountRouteErrorResponse(error, "Unable to save address.");
  }
}
