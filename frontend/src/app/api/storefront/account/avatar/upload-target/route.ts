// Location: src/app/api/storefront/account/avatar/upload-target/route.ts
import { accountCommerceFetch } from "@/lib/account/request";
import { accountResultResponse } from "@/lib/account/response";
import { accountRouteErrorResponse } from "@/lib/account/route-error";

export async function POST(request: Request) {
  try {
    const body = await request.text();

    const result = await accountCommerceFetch<unknown>(
      "/api/v1/customers/me/avatar/upload-target",
      {
        method: "POST",
        cache: "no-store",
        headers: { "Content-Type": "application/json" },
        body,
      },
    );

    return accountResultResponse(result);
  } catch (error) {
    return accountRouteErrorResponse(error, "Unable to prepare avatar upload.");
  }
}
