import { NextResponse } from "next/server";

import type { ProductDetailResponse } from "@/lib/api/contracts/catalog";
import { commerceFetch } from "@/lib/api/server";
import { routeErrorResponse } from "@/lib/commerce/route-error";

type RouteContext = {
  params: Promise<{
    slug: string;
  }>;
};

export async function GET(
  _request: Request,
  { params }: RouteContext,
) {
  try {
    const { slug } = await params;

    const response = await commerceFetch<ProductDetailResponse>(
      `/api/v1/products/${encodeURIComponent(slug)}`,
      {
        cache: "no-store",
      },
    );

    const product = response.data;

    return NextResponse.json(
      {
        data: {
          slug: product.slug,
          variants: product.variants.map((variant) => ({
            id: variant.id,
            sku: variant.sku,
            color_name: variant.color_name ?? null,
            size: variant.size ?? null,
            minimum_order_quantity: Math.max(
              1,
              variant.minimum_order_quantity,
            ),
            available_quantity: variant.available_quantity,
            in_stock: variant.in_stock,
          })),
        },
      },
      {
        headers: {
          "Cache-Control": "no-store",
        },
      },
    );
  } catch (error) {
    return routeErrorResponse(
      error,
      "Unable to load product purchase options.",
    );
  }
}
