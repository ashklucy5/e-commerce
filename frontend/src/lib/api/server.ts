import "server-only";

import {
  CommerceApiError,
  type CommerceApiErrorPayload,
} from "./error";

type CommerceFetchInit = RequestInit & {
  next?: {
    revalidate?: number | false;
    tags?: string[];
  };
};

function getCommerceApiUrl(): string {
  const value =
    process.env.COMMERCE_API_URL?.trim();

  if (!value) {
    throw new Error(
      "COMMERCE_API_URL is not configured.",
    );
  }

  return value.replace(/\/+$/, "");
}

export async function commerceFetch<T>(
  path: string,
  init: CommerceFetchInit = {},
): Promise<T> {
  const normalizedPath = path.startsWith("/")
    ? path
    : `/${path}`;

  const headers = new Headers(init.headers);

  if (!headers.has("Accept")) {
    headers.set("Accept", "application/json");
  }

  const response = await fetch(
    `${getCommerceApiUrl()}${normalizedPath}`,
    {
      ...init,
      headers,
    },
  );

  const contentType =
    response.headers.get("content-type") ?? "";

  let payload: unknown = null;

  if (contentType.includes("application/json")) {
    payload = await response.json();
  }

  if (!response.ok) {
    const errorPayload =
      payload as CommerceApiErrorPayload | null;

    throw new CommerceApiError(
      errorPayload?.error?.message ??
        `Commerce API request failed with HTTP ${response.status}.`,
      response.status,
      errorPayload?.error?.code ?? null,
    );
  }

  return payload as T;
}