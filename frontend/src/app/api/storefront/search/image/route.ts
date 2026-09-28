import {
  NextResponse,
} from "next/server";

import type {
  ImageSearchResponse,
} from "@/lib/api/contracts/image-search";

import {
  commerceFetch,
} from "@/lib/api/server";

import {
  routeErrorResponse,
} from "@/lib/commerce/route-error";

const MAX_IMAGE_BYTES =
  10 * 1024 * 1024;

const MAX_MULTIPART_BYTES =
  MAX_IMAGE_BYTES +
  512 * 1024;

const ACCEPTED_TYPES =
  new Set([
    "image/jpeg",
    "image/png",
    "image/webp",
    "image/gif",
  ]);

function positiveInt(
  value:
    string | null,

  fallback:
    number,

  maximum:
    number,
) {
  const parsed =
    Number.parseInt(
      value ?? "",
      10,
    );

  if (
    !Number.isInteger(
      parsed,
    ) ||
    parsed < 1
  ) {
    return fallback;
  }

  return Math.min(
    parsed,
    maximum,
  );
}

export async function POST(
  request:
    Request,
) {
  const url =
    new URL(
      request.url,
    );

  const page =
    positiveInt(
      url.searchParams.get(
        "page",
      ),
      1,
      100000,
    );

  const limit =
    positiveInt(
      url.searchParams.get(
        "limit",
      ),
      12,
      30,
    );

  /*
   * Reject obviously oversized requests before
   * parsing multipart data into memory.
   */
  const contentLength =
    Number(
      request.headers.get(
        "content-length",
      ) ?? "0",
    );

  if (
    Number.isFinite(
      contentLength,
    ) &&
    contentLength >
      MAX_MULTIPART_BYTES
  ) {
    return NextResponse.json(
      {
        error: {
          code:
            "SEARCH_IMAGE_TOO_LARGE",

          message:
            "Image search accepts files up to 10 MB.",
        },
      },
      {
        status: 413,
      },
    );
  }

  let incoming:
    FormData;

  try {
    incoming =
      await request.formData();
  } catch {
    return NextResponse.json(
      {
        error: {
          code:
            "INVALID_SEARCH_IMAGE",

          message:
            "Choose a valid image file.",
        },
      },
      {
        status: 400,
      },
    );
  }

  const image =
    incoming.get(
      "image",
    );

  if (
    !(
      image instanceof
      File
    ) ||
    image.size <= 0
  ) {
    return NextResponse.json(
      {
        error: {
          code:
            "INVALID_SEARCH_IMAGE",

          message:
            "Choose a non-empty product image.",
        },
      },
      {
        status: 400,
      },
    );
  }

  if (
    image.size >
    MAX_IMAGE_BYTES
  ) {
    return NextResponse.json(
      {
        error: {
          code:
            "SEARCH_IMAGE_TOO_LARGE",

          message:
            "Image search accepts files up to 10 MB.",
        },
      },
      {
        status: 413,
      },
    );
  }

  if (
    !ACCEPTED_TYPES.has(
      image.type,
    )
  ) {
    return NextResponse.json(
      {
        error: {
          code:
            "INVALID_SEARCH_IMAGE_TYPE",

          message:
            "Use a JPEG, PNG, WebP, or GIF image.",
        },
      },
      {
        status: 415,
      },
    );
  }

  const form =
    new FormData();

  form.append(
    "image",
    image,
    image.name ||
      "product-image",
  );

  const query =
    new URLSearchParams({
      page:
        String(page),

      limit:
        String(limit),
    });

  try {
    const result =
      await commerceFetch<ImageSearchResponse>(
        `/api/v1/search/image?${query.toString()}`,
        {
          method:
            "POST",

          cache:
            "no-store",

          body:
            form,
        },
      );

    return NextResponse.json(
      result,
    );
  } catch (error) {
    return routeErrorResponse(
      error,
      "Unable to search by image.",
    );
  }
}