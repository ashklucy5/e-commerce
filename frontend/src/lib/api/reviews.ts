import { commerceFetch } from "./server";

import type {
  ProductReviewListResponse,
  ProductReviewSummaryResponse,
} from "./contracts/reviews";

export async function getProductReviews(productID: string) {
  return commerceFetch<ProductReviewListResponse>(
    `/api/v1/reviews/products/${encodeURIComponent(productID)}`,
    {
      next: {
        revalidate: 60,
        tags: [`product-reviews:${productID}`],
      },
    },
  );
}

export async function getProductReviewSummary(productID: string) {
  const response = await commerceFetch<ProductReviewSummaryResponse>(
    `/api/v1/reviews/products/${encodeURIComponent(productID)}/summary`,
    {
      next: {
        revalidate: 60,
        tags: [`product-review-summary:${productID}`],
      },
    },
  );

  return response.data;
}
