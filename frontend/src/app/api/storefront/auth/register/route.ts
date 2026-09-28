// Location: src/app/api/storefront/auth/register/route.ts
import { NextResponse } from "next/server";

import { setAccountAuthCookies } from "@/lib/account/session";
import { commerceFetch } from "@/lib/api/server";
import type { AccountAuthResponse } from "@/lib/api/contracts/account";
import { routeErrorResponse } from "@/lib/commerce/route-error";

type RegisterBody = {
  full_name?: string;
  phone?: string;
  email?: string;
  password?: string;
};

export async function POST(request: Request) {
  let body: RegisterBody;

  try {
    body = (await request.json()) as RegisterBody;
  } catch {
    return NextResponse.json(
      { error: { code: "INVALID_REQUEST", message: "Invalid registration request." } },
      { status: 400 },
    );
  }

  try {
    const result = await commerceFetch<AccountAuthResponse>("/api/v1/auth/register", {
      method: "POST",
      cache: "no-store",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        full_name: body.full_name?.trim() ?? "",
        phone: body.phone?.trim() ?? "",
        email: body.email?.trim() ?? "",
        password: body.password ?? "",
      }),
    });

    const response = NextResponse.json({ data: result.data.customer }, { status: 201 });
    setAccountAuthCookies(response, result.data.tokens);
    return response;
  } catch (error) {
    return routeErrorResponse(error, "Unable to create account.");
  }
}
