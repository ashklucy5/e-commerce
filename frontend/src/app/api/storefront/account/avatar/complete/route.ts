// Location: src/app/api/storefront/account/avatar/complete/route.ts
import { NextResponse } from "next/server";

import { accountCommerceFetch } from "@/lib/account/request";
import { accountResultResponse } from "@/lib/account/response";
import { accountRouteErrorResponse } from "@/lib/account/route-error";

type AvatarCompleteBody = {
  key?: string;
  storage_key?: string;
};

export async function POST(request: Request) {
  let body: AvatarCompleteBody;

  try {
    body = (await request.json()) as AvatarCompleteBody;
  } catch {
    return NextResponse.json(
      {
        error: {
          code: "INVALID_REQUEST",
          message: "Invalid avatar completion request.",
        },
      },
      { status: 400 },
    );
  }

  const key = (body.key ?? body.storage_key ?? "").trim();

  if (!key) {
    return NextResponse.json(
      {
        error: {
          code: "INVALID_AVATAR_KEY",
          message: "The uploaded avatar could not be identified.",
        },
      },
      { status: 400 },
    );
  }

  try {
    const result = await accountCommerceFetch<unknown>(
      "/api/v1/customers/me/avatar/complete",
      {
        method: "POST",
        cache: "no-store",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ key }),
      },
    );

    return accountResultResponse(result);
  } catch (error) {
    return accountRouteErrorResponse(error, "Unable to save avatar.");
  }
}
