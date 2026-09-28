import { NextResponse } from "next/server";

import { accountCommerceFetch } from "@/lib/account/request";
import { accountResultResponse } from "@/lib/account/response";
import { accountRouteErrorResponse } from "@/lib/account/route-error";

type RouteContext = {
  params: Promise<{
    path: string[];
  }>;
};

type SupportedMethod = "GET" | "POST";

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
    {
      status,
    },
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

function routeAllowed(
  method: SupportedMethod,
  segments: string[],
) {
  if (
    segments.length < 1 ||
    !segments.every(validSegment)
  ) {
    return false;
  }

  if (method === "GET") {
    return (
      segments.length === 1 ||
      (segments.length === 2 &&
        segments[1] === "messages")
    );
  }

  return (
    segments.length === 2 &&
    segments[1] === "messages"
  );
}

async function proxyRequest(
  method: SupportedMethod,
  request: Request,
  context: RouteContext,
) {
  const { path } = await context.params;

  const segments = Array.isArray(path)
    ? path
    : [];

  if (!routeAllowed(method, segments)) {
    return jsonError(
      404,
      "SUPPORT_ROUTE_NOT_FOUND",
      "Support action was not found.",
    );
  }

  const encodedPath = segments
    .map((segment) => encodeURIComponent(segment))
    .join("/");

  const url = new URL(request.url);

  const backendPath =
    `/api/v1/crm/cases/${encodedPath}${url.search}`;

  try {
    const body =
      method === "POST"
        ? await request.text()
        : "";

    const result =
      await accountCommerceFetch<unknown>(
        backendPath,
        {
          method,
          cache: "no-store",

          ...(method === "POST"
            ? {
                headers: {
                  "Content-Type": "application/json",
                },

                ...(body
                  ? {
                      body,
                    }
                  : {}),
              }
            : {}),
        },
      );

    return accountResultResponse(
      result,
      method === "POST"
        ? {
            status: 201,
          }
        : undefined,
    );
  } catch (error) {
    return accountRouteErrorResponse(
      error,
      "Unable to process this support case.",
    );
  }
}

export async function GET(
  request: Request,
  context: RouteContext,
) {
  return proxyRequest(
    "GET",
    request,
    context,
  );
}

export async function POST(
  request: Request,
  context: RouteContext,
) {
  return proxyRequest(
    "POST",
    request,
    context,
  );
}