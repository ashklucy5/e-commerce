import { NextResponse } from "next/server";

import { accountCommerceFetch } from "@/lib/account/request";
import { accountResultResponse } from "@/lib/account/response";
import { accountRouteErrorResponse } from "@/lib/account/route-error";

import type {
  AccountSupportCaseResponse,
  AccountSupportCasesResponse,
} from "@/lib/api/contracts/account";

function boundedInteger(
  value: string | null,
  fallback: number,
  min: number,
  max: number,
) {
  const parsed = Number(value);

  if (!Number.isInteger(parsed)) {
    return fallback;
  }

  return Math.min(max, Math.max(min, parsed));
}

export async function GET(request: Request) {
  const url = new URL(request.url);

  const limit = boundedInteger(
    url.searchParams.get("limit"),
    50,
    1,
    100,
  );

  const offset = boundedInteger(
    url.searchParams.get("offset"),
    0,
    0,
    1_000_000,
  );

  const query = new URLSearchParams({
    limit: String(limit),
    offset: String(offset),
  });

  try {
    const result =
      await accountCommerceFetch<AccountSupportCasesResponse>(
        `/api/v1/crm/cases?${query.toString()}`,
        {
          cache: "no-store",
        },
      );

    return accountResultResponse(result);
  } catch (error) {
    return accountRouteErrorResponse(
      error,
      "Unable to load support cases.",
    );
  }
}

export async function POST(request: Request) {
  let body: unknown;

  try {
    body = await request.json();
  } catch {
    return NextResponse.json(
      {
        error: {
          code: "INVALID_REQUEST",
          message: "Invalid support request.",
        },
      },
      {
        status: 400,
      },
    );
  }

  try {
    const result =
      await accountCommerceFetch<AccountSupportCaseResponse>(
        "/api/v1/crm/cases",
        {
          method: "POST",
          cache: "no-store",
          headers: {
            "Content-Type": "application/json",
          },
          body: JSON.stringify(body),
        },
      );

    return accountResultResponse(result, {
      status: 201,
    });
  } catch (error) {
    return accountRouteErrorResponse(
      error,
      "Unable to create support case.",
    );
  }
}