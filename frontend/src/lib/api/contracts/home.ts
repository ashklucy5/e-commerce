import type {
  ProductCard,
  ProductListMeta,
} from "./catalog";

export type HomeMerchandisingGroup =
  | "new"
  | "discount"
  | "popular"
  | "catalog";

export type HomeProductCard =
  ProductCard & {
    compare_at_price_amount?:
      | number
      | null;

    published_at?: string;

    created_at: string;

    sold_quantity: number;

    merchandising_group:
      HomeMerchandisingGroup;
  };

export type HomeProductFeedResponse = {
  data: HomeProductCard[];

  meta: ProductListMeta;
};

export type StorefrontPromotion = {
  id: string;

  name: string;

  discount_type:
    | "percentage"
    | "fixed";

  percentage_bps?: number;

  fixed_amount?: number;

  minimum_subtotal_amount: number;

  maximum_discount_amount?: number;

  currency: string;

  starts_at?: string;

  ends_at?: string;
};

export type StorefrontPromotionsResponse = {
  data: StorefrontPromotion[];
};