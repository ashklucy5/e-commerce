import {
  NextResponse,
} from "next/server";

import {
  commerceFetch,
} from "@/lib/api/server";

import {
  clearAccountAuthCookies,
  getCustomerAccessToken,
} from "@/lib/account/session";

export async function POST() {
  const accessToken =
    await getCustomerAccessToken();

  if (
    accessToken
  ) {
    try {
      await commerceFetch(
        "/api/v1/auth/logout",
        {
          method:
            "POST",

          cache:
            "no-store",

          headers: {
            Authorization:
              `Bearer ${accessToken}`,
          },
        },
      );
    } catch {
      /*
       * Local logout still succeeds if
       * the backend session is already gone.
       */
    }
  }

  const response =
    new NextResponse(
      null,
      {
        status: 204,
      },
    );

  clearAccountAuthCookies(
    response,
  );

  return response;
}