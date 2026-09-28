import { NextResponse } from "next/server";

import { CommerceApiError } from "@/lib/api/error";

export function routeErrorResponse(
  error: unknown,
  fallback = "Unable to process request.",
) {
  if (error instanceof CommerceApiError) {
    return NextResponse.json(
      {
        error: {
          code: error.code ?? "COMMERCE_ERROR",
          message: error.message,
        },
      },
      { status: error.status },
    );
  }

  console.error(error);

  return NextResponse.json(
    {
      error: {
        code: "INTERNAL_ERROR",
        message: fallback,
      },
    },
    { status: 500 },
  );
}
