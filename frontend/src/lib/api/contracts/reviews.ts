export type ProductReview = {
  id: string;
  product_id: string;
  order_item_id?: string;
  customer_id?: string;
  rating: number;
  title?: string | null;
  body?: string | null;
  verified_purchase: boolean;
  customer_name?: string | null;
  reviewer_name?: string | null;
  created_at: string;
  updated_at?: string;
};

export type ProductReviewListResponse = {
  data: ProductReview[];
  meta?: {
    page?: number;
    limit?: number;
    total?: number;
    total_pages?: number;
  };
};

export type ProductReviewSummary = {
  product_id?: string;
  total_reviews: number;
  verified_purchase_count: number;
  average_rating: number;
};

export type ProductReviewSummaryResponse = {
  data: ProductReviewSummary;
};
