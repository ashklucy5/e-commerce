// Location: src/app/api/storefront/auth/ensure/route.ts
import { NextResponse } from "next/server";

import { accountCommerceFetch } from "@/lib/account/request";
import {
  clearAccountAuthCookies,
  setAccountAuthCookies,
} from "@/lib/account/session";
import type {
  AccountCustomer,
  AccountDataResponse,
} from "@/lib/api/contracts/account";

function safeReturnTo(value: string | null) {
  const candidate = value?.trim() ?? "";

  if (
    !candidate.startsWith("/") ||
    candidate.startsWith("//") ||
    candidate.includes("\\")
  ) {
    return "/account";
  }

  return candidate;
}

export async function GET(request: Request) {
  const requestURL = new URL(request.url);
  const returnTo = safeReturnTo(requestURL.searchParams.get("next"));

  try {
    const result = await accountCommerceFetch<
      AccountDataResponse<AccountCustomer>
    >("/api/v1/customers/me", { cache: "no-store" });

    const response = NextResponse.redirect(
      new URL(returnTo, requestURL.origin),
    );

    if (result.rotatedTokens) {
      setAccountAuthCookies(response, result.rotatedTokens);
    }

    return response;
  } catch {
    const signInURL = new URL("/account/sign-in", requestURL.origin);
    signInURL.searchParams.set("next", returnTo);

    const response = NextResponse.redirect(signInURL);
    clearAccountAuthCookies(response);
    return response;
  }
}
