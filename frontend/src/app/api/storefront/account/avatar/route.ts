// Location: src/app/api/storefront/account/avatar/route.ts
import { accountCommerceFetch } from "@/lib/account/request";
import { accountResultResponse } from "@/lib/account/response";
import { accountRouteErrorResponse } from "@/lib/account/route-error";

export async function DELETE() {
  try {
    const result = await accountCommerceFetch<unknown>(
      "/api/v1/customers/me/avatar",
      {
        method: "DELETE",
        cache: "no-store",
      },
    );

    return accountResultResponse(result);
  } catch (error) {
    return accountRouteErrorResponse(error, "Unable to remove avatar.");
  }
}
