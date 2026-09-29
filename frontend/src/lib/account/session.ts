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

import {
  CUSTOMER_SESSION_HINT_COOKIE,
} from "./session-contract";

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

/*
 * Non-sensitive browser-visible session hint.
 *
 * IMPORTANT:
 * This is never authentication authority.
 *
 * It contains no customer ID, access token,
 * refresh token, role, permission or secret.
 *
 * Its only purpose is to stop public storefront
 * components from probing authenticated endpoints
 * just to discover whether somebody is signed in.
 */
function sessionHintCookieOptions(
  expiresAt: string,
) {
  return {
    httpOnly: false,

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

  response.cookies.set(
    CUSTOMER_SESSION_HINT_COOKIE,
    "1",
    sessionHintCookieOptions(
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

  response.cookies.delete(
    CUSTOMER_SESSION_HINT_COOKIE,
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