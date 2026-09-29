import {
  commerceFetch,
} from "./server";

import type {
  HomeProductFeedResponse,
  StorefrontPromotion,
  StorefrontPromotionsResponse,
} from "./contracts/home";

export async function getHomeProductFeed(
  page = 1,
  limit = 50,
): Promise<HomeProductFeedResponse> {
  const params =
    new URLSearchParams({
      page:
        String(page),

      limit:
        String(limit),
    });

  return commerceFetch<HomeProductFeedResponse>(
    `/api/v1/products/home-feed?${params.toString()}`,
    {
      next: {
        revalidate: 30,

        tags: [
          "home-product-feed",
        ],
      },
    },
  );
}

/*
 * Homepage timed Deals / Flash Sale data.
 *
 * IMPORTANT:
 *
 * /promotions/active contains ordinary
 * order-level promotions.
 *
 * Product flash-sale campaigns use the
 * dedicated storefront endpoint below.
 */
export async function getHomePromotions(): Promise<
  StorefrontPromotion[]
> {
  const response =
    await commerceFetch<StorefrontPromotionsResponse>(
      "/api/v1/promotions/flash-sales/active?currency=BDT",
      {
        next: {
          revalidate: 30,

          tags: [
            "promotions",
            "flash-sales",
          ],
        },
      },
    );

  return response.data;
}