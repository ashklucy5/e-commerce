import type {
  AdminApiErrorResponse,
} from "./types";

export class AdminRequestError extends Error {
  readonly status: number;
  readonly code: string;

  constructor(
    message: string,
    status: number,
    code = "ADMIN_REQUEST_FAILED",
  ) {
    super(message);

    this.name = "AdminRequestError";
    this.status = status;
    this.code = code;
  }
}

export type AdminFetchOptions = {
  acceptedStatuses?: readonly number[];
};

export type AdminFetchResult<T> = {
  payload: T;
  status: number;
};

function isFormData(
  body: BodyInit | null | undefined,
): body is FormData {
  return (
    typeof FormData !== "undefined" &&
    body instanceof FormData
  );
}

function buildHeaders(
  init?: RequestInit,
): Headers {
  const headers = new Headers(
    init?.headers,
  );

  if (!headers.has("Accept")) {
    headers.set(
      "Accept",
      "application/json",
    );
  }

  /*
   * Do NOT manually set Content-Type for FormData.
   *
   * The browser needs to generate the multipart boundary.
   */
  if (
    init?.body &&
    !isFormData(init.body) &&
    !headers.has("Content-Type")
  ) {
    headers.set(
      "Content-Type",
      "application/json",
    );
  }

  return headers;
}

function errorDetails(
  payload: unknown,
  status: number,
): {
  message: string;
  code: string;
} {
  const fallback = {
    message:
      `Admin request failed with status ${status}.`,
    code: "ADMIN_REQUEST_FAILED",
  };

  if (
    !payload ||
    typeof payload !== "object"
  ) {
    return fallback;
  }

  const candidate =
    payload as AdminApiErrorResponse;

  /*
   * Some backend endpoints return:
   *
   * {
   *   "error": "message"
   * }
   */
  if (
    typeof candidate.error ===
    "string"
  ) {
    return {
      ...fallback,
      message: candidate.error,
    };
  }

  /*
   * Other admin endpoints return:
   *
   * {
   *   "error": {
   *     "code": "...",
   *     "message": "..."
   *   }
   * }
   */
  if (
    candidate.error &&
    typeof candidate.error ===
      "object"
  ) {
    return {
      message:
        candidate.error.message ??
        fallback.message,

      code:
        candidate.error.code ??
        fallback.code,
    };
  }

  /*
   * And tolerate:
   *
   * {
   *   "code": "...",
   *   "message": "..."
   * }
   */
  if (
    typeof candidate.message ===
    "string"
  ) {
    return {
      message:
        candidate.message,

      code:
        candidate.code ??
        fallback.code,
    };
  }

  return fallback;
}

async function requestAdmin<T>(
  path: string,
  init?: RequestInit,
  options?: AdminFetchOptions,
): Promise<AdminFetchResult<T>> {
  const response = await fetch(
    `/api/admin${path}`,
    {
      ...init,

      credentials:
        "same-origin",

      cache:
        "no-store",

      headers:
        buildHeaders(init),
    },
  );

  const contentType =
    response.headers.get(
      "content-type",
    ) ?? "";

  let payload: unknown = null;

  if (
    contentType.includes(
      "application/json",
    )
  ) {
    payload =
      await response.json();
  } else if (
    response.status !== 204
  ) {
    const text =
      await response.text();

    payload =
      text || null;
  }

  /*
   * Catalog import intentionally uses HTTP 422
   * for a workbook which was staged successfully
   * but contains validation failures.
   *
   * We therefore need selected non-2xx statuses
   * to remain readable by the caller.
   */
  const accepted =
    options?.acceptedStatuses?.includes(
      response.status,
    ) ?? false;

  if (
    !response.ok &&
    !accepted
  ) {
    const details =
      errorDetails(
        payload,
        response.status,
      );

    throw new AdminRequestError(
      details.message,
      response.status,
      details.code,
    );
  }

  return {
    payload:
      payload as T,

    status:
      response.status,
  };
}

export async function adminFetch<T>(
  path: string,
  init?: RequestInit,
): Promise<T> {
  const result =
    await requestAdmin<T>(
      path,
      init,
    );

  return result.payload;
}

export async function adminFetchWithStatus<T>(
  path: string,
  init?: RequestInit,
  options?: AdminFetchOptions,
): Promise<AdminFetchResult<T>> {
  return requestAdmin<T>(
    path,
    init,
    options,
  );
}