// Location:
// src/app/api/storefront/account/orders/route.ts

import { NextResponse } from "next/server";

import { accountCommerceFetch } from "@/lib/account/request";
import { accountResultResponse } from "@/lib/account/response";
import { accountRouteErrorResponse } from "@/lib/account/route-error";

import type {
  AccountOrderHistoryItem,
  AccountOrderHistoryResponse,
} from "@/lib/api/contracts/account";

import {
  catalogMediaForOrderProductName,
  type OrderCatalogMedia,
} from "@/lib/orders/catalog-media";

type EnrichedOrderHistoryItem =
  AccountOrderHistoryItem &
  OrderCatalogMedia;

type EnrichedOrderHistoryResponse =
  Omit<
    AccountOrderHistoryResponse,
    "data"
  > & {
    data:
      EnrichedOrderHistoryItem[];
  };

function boundedInteger(
  value: string | null,
  fallback: number,
  min: number,
  max: number,
) {
  const parsed =
    Number(value);

  if (
    !Number.isInteger(
      parsed,
    )
  ) {
    return fallback;
  }

  return Math.min(
    max,
    Math.max(
      min,
      parsed,
    ),
  );
}

function normalize(
  value:
    | string
    | null
    | undefined,
) {
  return String(
    value ?? "",
  )
    .trim()
    .toLocaleLowerCase();
}

async function enrichOrderHistory(
  orders:
    AccountOrderHistoryItem[],
) {
  if (
    orders.length === 0
  ) {
    return orders as EnrichedOrderHistoryItem[];
  }

  const mediaByName =
    new Map<
      string,
      OrderCatalogMedia
    >();

  const uniqueNames =
    Array.from(
      new Set(
        orders
          .map(
            (order) =>
              order
                .first_product_name
                ?.trim() ??
              "",
          )
          .filter(Boolean)
          .map(normalize),
      ),
    );

  const originalNameByNormalized =
    new Map<
      string,
      string
    >();

  for (
    const order
    of orders
  ) {
    const original =
      order
        .first_product_name
        ?.trim();

    if (original) {
      originalNameByNormalized.set(
        normalize(
          original,
        ),
        original,
      );
    }
  }

  let cursor = 0;

  const workerCount =
    Math.min(
      5,
      uniqueNames.length,
    );

  async function worker() {
    while (
      cursor <
      uniqueNames.length
    ) {
      const index =
        cursor;

      cursor += 1;

      const key =
        uniqueNames[index];

      if (!key) {
        continue;
      }

      const originalName =
        originalNameByNormalized.get(
          key,
        );

      if (
        !originalName
      ) {
        continue;
      }

      mediaByName.set(
        key,
        await catalogMediaForOrderProductName(
          originalName,
        ),
      );
    }
  }

  await Promise.all(
    Array.from(
      {
        length:
          workerCount,
      },

      () =>
        worker(),
    ),
  );

  return orders.map(
    (order) => ({
      ...order,

      ...(order
        .first_product_name
        ? mediaByName.get(
            normalize(
              order.first_product_name,
            ),
          )
        : undefined),
    }),
  );
}

export async function GET(
  request: Request,
) {
  const url =
    new URL(
      request.url,
    );

  const page =
    boundedInteger(
      url.searchParams.get(
        "page",
      ),
      1,
      1,
      100000,
    );

  const limit =
    boundedInteger(
      url.searchParams.get(
        "limit",
      ),
      50,
      1,
      100,
    );

  const status =
    (
      url.searchParams.get(
        "status",
      ) ?? ""
    )
      .trim()
      .slice(
        0,
        40,
      );

  const query =
    new URLSearchParams({
      page:
        String(page),

      limit:
        String(limit),
    });

  if (
    status &&
    status !== "all"
  ) {
    query.set(
      "status",
      status,
    );
  }

  try {
    const result =
      await accountCommerceFetch<AccountOrderHistoryResponse>(
        `/api/v1/orders?${query.toString()}`,
        {
          cache:
            "no-store",
        },
      );

    const orders =
      Array.isArray(
        result.payload.data,
      )
        ? result.payload.data
        : [];

    const enrichedOrders =
      await enrichOrderHistory(
        orders,
      );

    const enrichedPayload:
      EnrichedOrderHistoryResponse =
      {
        ...result.payload,

        data:
          enrichedOrders,
      };

    return accountResultResponse(
      {
        ...result,

        payload:
          enrichedPayload,
      },
    );
  } catch (error) {
    return accountRouteErrorResponse(
      error,
      "Unable to load your orders.",
    );
  }
}

export function OPTIONS() {
  return new NextResponse(
    null,
    {
      status: 204,
    },
  );
}