export type AdminReviewStatus = "published" | "hidden";

export type AdminReview = {
  id: string;
  customer_id: string;
  customer_name: string;
  customer_phone: string;
  order_id: string;
  order_number: string;
  order_item_id: string;
  product_id: string;
  variant_id: string;
  sku: string;
  product_name: string;
  rating: number;
  title?: string;
  body?: string;
  status: AdminReviewStatus;
  verified_purchase: boolean;
  created_at: string;
  updated_at: string;
};

export type AdminReviewPaginationMeta = {
  page: number;
  limit: number;
  total: number;
  total_pages: number;
  has_next: boolean;
  has_previous: boolean;
};

export type AdminReviewListResponse = {
  data: AdminReview[];
  meta: AdminReviewPaginationMeta;
};

export type AdminReviewResponse = {
  data: AdminReview;
};
