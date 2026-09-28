// Location: src/lib/account/route-error.ts
import {
  CommerceApiError,
} from "@/lib/api/error";

import {
  routeErrorResponse,
} from "@/lib/commerce/route-error";

import {
  clearAccountAuthCookies,
} from "./session";

export function accountRouteErrorResponse(
  error: unknown,
  fallback: string,
) {
  const response =
    routeErrorResponse(
      error,
      fallback,
    );

  if (
    error instanceof
      CommerceApiError &&
    error.status ===
      401
  ) {
    clearAccountAuthCookies(
      response,
    );
  }

  return response;
}