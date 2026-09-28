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

export async function getHomePromotions(): Promise<
  StorefrontPromotion[]
> {
  const response =
    await commerceFetch<StorefrontPromotionsResponse>(
      "/api/v1/promotions/active?currency=BDT",
      {
        next: {
          revalidate: 30,

          tags: [
            "promotions",
          ],
        },
      },
    );

  return response.data;
}