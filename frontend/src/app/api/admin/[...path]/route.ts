import { NextRequest, NextResponse } from "next/server";

const API_URL =
  process.env.COMMERCE_API_URL?.replace(/\/+$/, "") ??
  "http://127.0.0.1:8081";

const MUTATING_METHODS = new Set([
  "POST",
  "PUT",
  "PATCH",
  "DELETE",
]);

function buildBackendUrl(
  request: NextRequest,
  path: string[],
): string {
  const url = new URL(
    `${API_URL}/api/v1/admin/${path.join("/")}`,
  );

  request.nextUrl.searchParams.forEach((value, key) => {
    url.searchParams.append(key, value);
  });

  return url.toString();
}

function copySetCookies(
  backendResponse: Response,
  nextResponse: NextResponse,
) {
  const headers = backendResponse.headers as Headers & {
    getSetCookie?: () => string[];
  };

  const setCookies =
    typeof headers.getSetCookie === "function"
      ? headers.getSetCookie()
      : backendResponse.headers.get("set-cookie")
        ? [backendResponse.headers.get("set-cookie")!]
        : [];

  for (const cookie of setCookies) {
    const rewrittenCookie = cookie.replace(
      /Path=\/api\/v1\/admin(?=;|$)/i,
      "Path=/api/admin",
    );

    nextResponse.headers.append(
      "set-cookie",
      rewrittenCookie,
    );
  }
}

async function proxyRequest(
  request: NextRequest,
  context: {
    params: Promise<{
      path: string[];
    }>;
  },
) {
  const { path } = await context.params;
  const method = request.method.toUpperCase();

  const headers = new Headers();

  const contentType = request.headers.get("content-type");
  const accept = request.headers.get("accept");
  const cookie = request.headers.get("cookie");

  if (contentType) {
    headers.set("content-type", contentType);
  }

  if (accept) {
    headers.set("accept", accept);
  }

  if (cookie) {
    headers.set("cookie", cookie);
  }

  const explicitCsrf = request.headers.get("x-csrf-token");

  const csrfCookie =
    request.cookies.get("commerce_admin_csrf")?.value;

  if (explicitCsrf) {
    headers.set("x-csrf-token", explicitCsrf);
  } else if (
    MUTATING_METHODS.has(method) &&
    csrfCookie
  ) {
    headers.set("x-csrf-token", csrfCookie);
  }

  const init: RequestInit = {
    method,
    headers,
    cache: "no-store",
    redirect: "manual",
  };

  if (method !== "GET" && method !== "HEAD") {
    init.body = await request.arrayBuffer();
  }

  let backendResponse: Response;

  try {
    backendResponse = await fetch(
      buildBackendUrl(request, path),
      init,
    );
  } catch {
    return NextResponse.json(
      {
        error: {
          code: "ADMIN_API_UNAVAILABLE",
          message:
            "The commerce API is currently unavailable.",
        },
      },
      {
        status: 503,
      },
    );
  }

  const body = await backendResponse.arrayBuffer();

  const response = new NextResponse(
    body.byteLength > 0 ? body : null,
    {
      status: backendResponse.status,
    },
  );

  const responseContentType =
    backendResponse.headers.get("content-type");

  if (responseContentType) {
    response.headers.set(
      "content-type",
      responseContentType,
    );
  }

  const requestId =
    backendResponse.headers.get("x-request-id");

  if (requestId) {
    response.headers.set(
      "x-request-id",
      requestId,
    );
  }

  copySetCookies(
    backendResponse,
    response,
  );

  return response;
}

export const GET = proxyRequest;
export const POST = proxyRequest;
export const PUT = proxyRequest;
export const PATCH = proxyRequest;
export const DELETE = proxyRequest;