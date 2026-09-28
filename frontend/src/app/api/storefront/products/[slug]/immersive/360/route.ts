import {
  NextResponse,
} from "next/server";

import {
  commerceFetch,
} from "@/lib/api/server";

import type {
  ProductSpin360FrameResponse,
} from "@/lib/api/contracts/catalog";

import {
  routeErrorResponse,
} from "@/lib/commerce/route-error";

type Context = {
  params: Promise<{
    slug: string;
  }>;
};

export async function GET(
  request: Request,
  context: Context,
) {
  const {
    slug,
  } =
    await context.params;

  const requestURL =
    new URL(
      request.url,
    );

  const variantID =
    requestURL.searchParams
      .get(
        "variant_id",
      )
      ?.trim() ??
    "";

  const query =
    variantID
      ? `?variant_id=${encodeURIComponent(
          variantID,
        )}`
      : "";

  try {
    const result =
      await commerceFetch<
        ProductSpin360FrameResponse
      >(
        `/api/v1/products/${encodeURIComponent(
          slug,
        )}/immersive/360${query}`,
        {
          cache:
            "no-store",
        },
      );

    return NextResponse.json(
      result,
    );
  } catch (
    error
  ) {
    return routeErrorResponse(
      error,
      "Unable to load product 360 media.",
    );
  }
}