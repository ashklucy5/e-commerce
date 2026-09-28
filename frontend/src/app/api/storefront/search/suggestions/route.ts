import { NextResponse } from "next/server";

import type {
  SearchSuggestionsResponse,
} from "@/lib/api/contracts/search-suggestions";

import {
  commerceFetch,
} from "@/lib/api/server";

import {
  routeErrorResponse,
} from "@/lib/commerce/route-error";

export async function GET(
  request: Request,
) {
  const url = new URL(
    request.url,
  );

  const query = (
    url.searchParams.get("q") ?? ""
  ).trim();

  const requestedLimit = Number(
    url.searchParams.get("limit") ??
      "6",
  );

  const limit =
    Number.isInteger(requestedLimit) &&
    requestedLimit > 0
      ? Math.min(
          requestedLimit,
          10,
        )
      : 6;

  if (query.length < 2) {
    return NextResponse.json({
      data: {
        query,
        products: [],
        categories: [],
        brands: [],
      },
    });
  }

  const parameters =
    new URLSearchParams({
      q: query,
      limit: String(limit),
    });

  try {
    const result =
      await commerceFetch<SearchSuggestionsResponse>(
        `/api/v1/search/suggestions?${parameters.toString()}`,
        {
          cache: "no-store",
        },
      );

    return NextResponse.json(
      result,
    );
  } catch (error) {
    return routeErrorResponse(
      error,
      "Unable to load search suggestions.",
    );
  }
}