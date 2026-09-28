// Location: src/lib/account/response.ts
import { NextResponse } from "next/server";

import type { AccountRequestResult } from "./request";
import { setAccountAuthCookies } from "./session";

export function accountResultResponse<T>(
  result: AccountRequestResult<T>,
  init?: ResponseInit,
) {
  const response = NextResponse.json(result.payload, init);

  if (result.rotatedTokens) {
    setAccountAuthCookies(response, result.rotatedTokens);
  }

  return response;
}
