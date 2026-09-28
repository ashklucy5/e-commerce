// Location: src/lib/account/request.ts
import "server-only";

import {
  CommerceApiError,
} from "@/lib/api/error";

import {
  commerceFetch,
} from "@/lib/api/server";

import type {
  AccountAuthResponse,
  AccountAuthTokens,
} from "@/lib/api/contracts/account";

import {
  getCustomerAccessToken,
  getCustomerRefreshToken,
} from "./session";

export type AccountRequestResult<T> = {
  payload: T;

  rotatedTokens?:
    AccountAuthTokens;
};

function withAuthorization(
  token: string,
  init: RequestInit,
) {
  const headers =
    new Headers(
      init.headers,
    );

  headers.set(
    "Authorization",
    `Bearer ${token}`,
  );

  return {
    ...init,

    headers,
  };
}

async function refreshCustomerSession(
  refreshToken: string,
) {
  return commerceFetch<
    AccountAuthResponse
  >(
    "/api/v1/auth/refresh",
    {
      method:
        "POST",

      cache:
        "no-store",

      headers: {
        "Content-Type":
          "application/json",
      },

      body:
        JSON.stringify({
          refresh_token:
            refreshToken,
        }),
    },
  );
}

export async function accountCommerceFetch<T>(
  path: string,
  init: RequestInit = {},
): Promise<
  AccountRequestResult<T>
> {
  const [
    accessToken,
    refreshToken,
  ] =
    await Promise.all([
      getCustomerAccessToken(),

      getCustomerRefreshToken(),
    ]);

  if (
    !accessToken &&
    !refreshToken
  ) {
    throw new CommerceApiError(
      "Authentication required.",
      401,
      "UNAUTHORIZED",
    );
  }

  if (
    accessToken
  ) {
    try {
      const payload =
        await commerceFetch<T>(
          path,
          withAuthorization(
            accessToken,
            init,
          ),
        );

      return {
        payload,
      };
    } catch (
      error
    ) {
      if (
        !(
          error instanceof
            CommerceApiError
        ) ||
        error.status !==
          401
      ) {
        throw error;
      }
    }
  }

  if (
    !refreshToken
  ) {
    throw new CommerceApiError(
      "Your session has expired.",
      401,
      "UNAUTHORIZED",
    );
  }

  const refreshed =
    await refreshCustomerSession(
      refreshToken,
    );

  const payload =
    await commerceFetch<T>(
      path,
      withAuthorization(
        refreshed.data
          .tokens
          .access_token,
        init,
      ),
    );

  return {
    payload,

    rotatedTokens:
      refreshed.data
        .tokens,
  };
}