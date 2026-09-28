import {
  getHomeProductFeed,
} from "@/lib/api/home";

function positiveInteger(
  value: string | null,
  fallback: number,
) {
  if (!value) {
    return fallback;
  }

  const parsed =
    Number.parseInt(
      value,
      10,
    );

  if (
    !Number.isFinite(
      parsed,
    ) ||
    parsed < 1
  ) {
    return fallback;
  }

  return parsed;
}

export async function GET(
  request: Request,
) {
  const url =
    new URL(
      request.url,
    );

  const page =
    positiveInteger(
      url.searchParams.get(
        "page",
      ),
      1,
    );

  const requestedLimit =
    positiveInteger(
      url.searchParams.get(
        "limit",
      ),
      50,
    );

  const limit =
    Math.min(
      requestedLimit,
      50,
    );

  try {
    const result =
      await getHomeProductFeed(
        page,
        limit,
      );

    return Response.json(
      result,
      {
        status: 200,

        headers: {
          "Cache-Control":
            "public, max-age=15, stale-while-revalidate=45",
        },
      },
    );
  } catch (error) {
    console.error(
      "home product feed route failed",
      error,
    );

    return Response.json(
      {
        error: {
          code:
            "HOME_FEED_UNAVAILABLE",

          message:
            "Unable to load more products right now.",
        },
      },
      {
        status: 502,
      },
    );
  }
}