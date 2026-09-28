// Location: src/lib/account/session.ts
import "server-only";

import {
  cookies,
} from "next/headers";

import type {
  NextResponse,
} from "next/server";

import type {
  AccountAuthTokens,
} from "@/lib/api/contracts/account";

export const CUSTOMER_ACCESS_COOKIE =
  "ene_dei_customer_access";

export const CUSTOMER_REFRESH_COOKIE =
  "ene_dei_customer_refresh";

const secure =
  process.env.NODE_ENV ===
  "production";

function authCookieOptions(
  expiresAt: string,
) {
  return {
    httpOnly: true,

    secure,

    sameSite:
      "lax" as const,

    path: "/",

    expires:
      new Date(
        expiresAt,
      ),
  };
}

export function setAccountAuthCookies(
  response: NextResponse,
  tokens: AccountAuthTokens,
) {
  response.cookies.set(
    CUSTOMER_ACCESS_COOKIE,
    tokens.access_token,
    authCookieOptions(
      tokens.access_expires_at,
    ),
  );

  response.cookies.set(
    CUSTOMER_REFRESH_COOKIE,
    tokens.refresh_token,
    authCookieOptions(
      tokens.refresh_expires_at,
    ),
  );
}

export function clearAccountAuthCookies(
  response: NextResponse,
) {
  response.cookies.delete(
    CUSTOMER_ACCESS_COOKIE,
  );

  response.cookies.delete(
    CUSTOMER_REFRESH_COOKIE,
  );
}

export async function getCustomerAccessToken() {
  const store =
    await cookies();

  return (
    store.get(
      CUSTOMER_ACCESS_COOKIE,
    )?.value ??
    ""
  );
}

export async function getCustomerRefreshToken() {
  const store =
    await cookies();

  return (
    store.get(
      CUSTOMER_REFRESH_COOKIE,
    )?.value ??
    ""
  );
}