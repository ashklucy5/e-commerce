import { NextResponse } from "next/server";

import { accountCommerceFetch } from "@/lib/account/request";
import { accountResultResponse } from "@/lib/account/response";
import { accountRouteErrorResponse } from "@/lib/account/route-error";

type RouteContext = {
  params: Promise<{
    path: string[];
  }>;
};

type SupportedMethod = "GET" | "PATCH";

type ResolvedRoute = {
  backendPath: string;
  allowedMethods: SupportedMethod[];
};

function jsonError(
  status: number,
  code: string,
  message: string,
) {
  return NextResponse.json(
    {
      error: {
        code,
        message,
      },
    },
    { status },
  );
}

function validSegment(value: string) {
  const trimmed = value.trim();

  return Boolean(
    trimmed &&
      trimmed.length <= 200 &&
      !trimmed.includes("/") &&
      !trimmed.includes("\\") &&
      trimmed !== "." &&
      trimmed !== "..",
  );
}

function resolveRoute(segments: string[]): ResolvedRoute | null {
  if (
    segments.length !== 1 ||
    !segments.every(validSegment)
  ) {
    return null;
  }

  switch (segments[0]) {
    case "profile":
      return {
        backendPath: "/api/v1/customers/me",
        allowedMethods: ["GET", "PATCH"],
      };

    case "preferences":
      return {
        backendPath: "/api/v1/customers/me/preferences",
        allowedMethods: ["GET", "PATCH"],
      };

    case "notifications":
      return {
        backendPath: "/api/v1/customers/me/notification-preferences",
        allowedMethods: ["GET", "PATCH"],
      };

    case "privacy":
      return {
        backendPath: "/api/v1/customers/me/privacy-settings",
        allowedMethods: ["GET", "PATCH"],
      };

    default:
      return null;
  }
}

async function proxyRequest(
  method: SupportedMethod,
  request: Request,
  context: RouteContext,
) {
  const { path } = await context.params;
  const segments = Array.isArray(path) ? path : [];
  const route = resolveRoute(segments);

  if (!route || !route.allowedMethods.includes(method)) {
    return jsonError(
      404,
      "ACCOUNT_SETTINGS_ROUTE_NOT_FOUND",
      "Account settings action was not found.",
    );
  }

  try {
    const hasBody = method === "PATCH";
    const body = hasBody ? await request.text() : "";

    const result = await accountCommerceFetch<unknown>(
      route.backendPath,
      {
        method,
        cache: "no-store",
        ...(hasBody
          ? {
              headers: {
                "Content-Type": "application/json",
              },
              ...(body ? { body } : {}),
            }
          : {}),
      },
    );

    return accountResultResponse(result);
  } catch (error) {
    return accountRouteErrorResponse(
      error,
      "Unable to update account settings.",
    );
  }
}

export async function GET(
  request: Request,
  context: RouteContext,
) {
  return proxyRequest("GET", request, context);
}

export async function PATCH(
  request: Request,
  context: RouteContext,
) {
  return proxyRequest("PATCH", request, context);
}
