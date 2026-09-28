// Location: src/app/api/storefront/account/requests/route.ts

import { NextResponse } from "next/server";

import { accountCommerceFetch } from "@/lib/account/request";
import { accountResultResponse } from "@/lib/account/response";
import { accountRouteErrorResponse } from "@/lib/account/route-error";

import type {
  ProductRequest,
  ProductRequestDataResponse,
  ProductRequestListResponse,
} from "@/lib/api/contracts/product-request";

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

  return Math.min(
    max,
    Math.max(
      min,
      parsed,
    ),
  );
}

export async function GET(
  request: Request,
) {
  const url =
    new URL(
      request.url,
    );

  const limit =
    boundedInteger(
      url.searchParams.get(
        "limit",
      ),
      20,
      1,
      100,
    );

  const offset =
    boundedInteger(
      url.searchParams.get(
        "offset",
      ),
      0,
      0,
      1_000_000,
    );

  const query =
    new URLSearchParams({
      limit:
        String(
          limit,
        ),

      offset:
        String(
          offset,
        ),
    });

  try {
    const result =
      await accountCommerceFetch<
        ProductRequestListResponse
      >(
        `/api/v1/product-requests?${query.toString()}`,
        {
          cache:
            "no-store",
        },
      );

    return accountResultResponse(
      result,
    );
  } catch (
    error
  ) {
    return accountRouteErrorResponse(
      error,
      "Unable to load your product requests.",
    );
  }
}

export async function POST(
  request: Request,
) {
  try {
    const body =
      await request.text();

    const result =
      await accountCommerceFetch<
        ProductRequestDataResponse<ProductRequest>
      >(
        "/api/v1/product-requests",
        {
          method:
            "POST",

          cache:
            "no-store",

          headers: {
            "Content-Type":
              "application/json",
          },

          body,
        },
      );

    return accountResultResponse(
      result,
      {
        status:
          201,
      },
    );
  } catch (
    error
  ) {
    return accountRouteErrorResponse(
      error,
      "Unable to create your product request.",
    );
  }
}

export function OPTIONS() {
  return new NextResponse(
    null,
    {
      status:
        204,
    },
  );
}