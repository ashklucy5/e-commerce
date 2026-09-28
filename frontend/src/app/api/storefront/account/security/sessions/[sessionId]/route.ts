// Location: src/app/api/storefront/account/security/sessions/[sessionId]/route.ts

import { accountCommerceFetch } from "@/lib/account/request";
import { accountResultResponse } from "@/lib/account/response";
import { accountRouteErrorResponse } from "@/lib/account/route-error";

type Context = {
  params: Promise<{ sessionId: string }>;
};

export async function DELETE(_request: Request, context: Context) {
  try {
    const { sessionId } = await context.params;
    const result = await accountCommerceFetch<unknown>(
      `/api/v1/customers/me/sessions/${encodeURIComponent(sessionId)}`,
      { method: "DELETE", cache: "no-store" },
    );

    return accountResultResponse(result);
  } catch (error) {
    return accountRouteErrorResponse(error, "Unable to revoke this session.");
  }
}
