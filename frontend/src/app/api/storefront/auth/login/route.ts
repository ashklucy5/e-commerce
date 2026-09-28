// Location: src/app/api/storefront/auth/login/route.ts
import { NextResponse } from "next/server";

import { setAccountAuthCookies } from "@/lib/account/session";
import { commerceFetch } from "@/lib/api/server";
import type { AccountAuthResponse } from "@/lib/api/contracts/account";
import { routeErrorResponse } from "@/lib/commerce/route-error";

type LoginBody = {
  phone?: string;
  password?: string;
};

export async function POST(request: Request) {
  let body: LoginBody;

  try {
    body = (await request.json()) as LoginBody;
  } catch {
    return NextResponse.json(
      { error: { code: "INVALID_REQUEST", message: "Invalid sign-in request." } },
      { status: 400 },
    );
  }

  try {
    const result = await commerceFetch<AccountAuthResponse>("/api/v1/auth/login", {
      method: "POST",
      cache: "no-store",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        phone: body.phone?.trim() ?? "",
        password: body.password ?? "",
      }),
    });

    const response = NextResponse.json({ data: result.data.customer });
    setAccountAuthCookies(response, result.data.tokens);
    return response;
  } catch (error) {
    return routeErrorResponse(error, "Unable to sign in.");
  }
}
