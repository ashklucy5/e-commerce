// Location: src/app/api/storefront/account/orders/[orderId]/route.ts

import { accountCommerceFetch } from "@/lib/account/request";
import { accountResultResponse } from "@/lib/account/response";
import { accountRouteErrorResponse } from "@/lib/account/route-error";
import { commerceFetch } from "@/lib/api/server";
import type { AccountOrderItem, AccountOrderResponse } from "@/lib/api/contracts/order-account";

type RouteContext = {
  params: Promise<{ orderId: string }>;
};

type CatalogSearchItem = {
  id: string;
  name: string;
  slug: string;
  primary_image_url?: string | null;
  matched_sku?: string | null;
};

type CatalogSearchResponse = {
  data: CatalogSearchItem[];
};

type CatalogMedia = {
  image_url?: string;
  product_slug?: string;
};

function normalize(value: string | null | undefined) {
  return String(value ?? "").trim().toLocaleLowerCase();
}

async function catalogMediaForItem(item: AccountOrderItem): Promise<CatalogMedia> {
  const sku = item.sku.trim();
  if (!sku) return {};

  try {
    const query = new URLSearchParams({
      q: sku,
      limit: "3",
    });

    const response = await commerceFetch<CatalogSearchResponse>(
      `/api/v1/search/products?${query.toString()}`,
      {
        next: {
          revalidate: 300,
          tags: [`order-item-media:${sku}`],
        },
      },
    );

    const exactSku = response.data.find(
      (product) => normalize(product.matched_sku) === normalize(sku),
    );

    const exactName = response.data.find(
      (product) => normalize(product.name) === normalize(item.product_name),
    );

    const product = exactSku ?? exactName;
    if (!product) return {};

    return {
      image_url: product.primary_image_url?.trim() || undefined,
      product_slug: product.slug?.trim() || undefined,
    };
  } catch {
    // Product imagery is enrichment only. A catalog/network failure must never
    // prevent the customer from opening their immutable order snapshot.
    return {};
  }
}

async function enrichOrderItems(items: AccountOrderItem[]) {
  if (items.length === 0) return items;

  const mediaBySku = new Map<string, CatalogMedia>();
  const uniqueItems = Array.from(
    new Map(
      items
        .filter((item) => item.sku.trim())
        .map((item) => [normalize(item.sku), item] as const),
    ).values(),
  );

  let cursor = 0;
  const workerCount = Math.min(5, uniqueItems.length);

  async function worker() {
    while (cursor < uniqueItems.length) {
      const index = cursor;
      cursor += 1;

      const item = uniqueItems[index];
      if (!item) continue;

      const media = await catalogMediaForItem(item);
      mediaBySku.set(normalize(item.sku), media);
    }
  }

  await Promise.all(Array.from({ length: workerCount }, () => worker()));

  return items.map((item) => ({
    ...item,
    ...mediaBySku.get(normalize(item.sku)),
  }));
}

export async function GET(_request: Request, context: RouteContext) {
  const { orderId } = await context.params;

  try {
    const result = await accountCommerceFetch<AccountOrderResponse>(
      `/api/v1/orders/${encodeURIComponent(orderId)}`,
      { cache: "no-store" },
    );

    const items = Array.isArray(result.payload.data.items)
      ? result.payload.data.items
      : [];

    const enrichedItems = await enrichOrderItems(items);

    return accountResultResponse({
      ...result,
      payload: {
        ...result.payload,
        data: {
          ...result.payload.data,
          items: enrichedItems,
        },
      },
    });
  } catch (error) {
    return accountRouteErrorResponse(error, "Unable to load this order.");
  }
}
