export type AdminPromotionStatus = "draft" | "active" | "disabled";
export type AdminPromotionEffectiveState =
  | "draft"
  | "disabled"
  | "scheduled"
  | "live"
  | "expired";
export type AdminPromotionMode = "automatic" | "code";
export type AdminPromotionScope = "order" | "product";
export type AdminPromotionCampaignType = "standard" | "flash_sale";
export type AdminPromotionDiscountType = "percentage" | "fixed";

export type AdminPromotionTarget = {
  type: "product" | "variant" | string;
  product_id?: string;
  product_name?: string;
  variant_id?: string;
  sku?: string;
};

export type AdminPromotion = {
  id: string;
  name: string;
  code?: string;
  mode: AdminPromotionMode;
  scope: AdminPromotionScope;
  campaign_type: AdminPromotionCampaignType;
  discount_type: AdminPromotionDiscountType;
  percentage_bps?: number;
  fixed_amount?: number;
  minimum_subtotal_amount: number;
  maximum_discount_amount?: number;
  currency: string;
  status: AdminPromotionStatus;
  effective_state: AdminPromotionEffectiveState;
  starts_at?: string;
  ends_at?: string;
  target_count: number;
  targets?: AdminPromotionTarget[];
  created_at: string;
  updated_at: string;
};

export type AdminPromotionPaginationMeta = {
  page: number;
  limit: number;
  total: number;
  total_pages: number;
  has_next: boolean;
  has_previous: boolean;
};

export type AdminPromotionListResponse = {
  data: AdminPromotion[];
  meta: AdminPromotionPaginationMeta;
};

export type AdminPromotionResponse = {
  data: AdminPromotion;
};

export type AdminPromotionTargetInput = {
  product_id?: string;
  variant_id?: string;
};

export type AdminPromotionConfigRequest = {
  name: string;
  code: string | null;
  scope: AdminPromotionScope;
  campaign_type: AdminPromotionCampaignType;
  discount_type: AdminPromotionDiscountType;
  percentage_bps: number | null;
  fixed_amount: number | null;
  minimum_subtotal_amount: number;
  maximum_discount_amount: number | null;
  currency: string;
  status: AdminPromotionStatus;
  starts_at: string | null;
  ends_at: string | null;
  targets: AdminPromotionTargetInput[];
};

export type AdminProductDiscount = {
  product_id: string;
  product_code: string;
  product_name: string;
  variant_id: string;
  sku: string;
  discount_active: boolean;
  regular_price_amount: number;
  sale_price_amount: number;
  discount_amount: number;
  discount_bps: number;
  currency: string;
  variant_active: boolean;
  product_status: string;
  updated_at: string;
};

export type AdminProductDiscountListResponse = {
  data: AdminProductDiscount[];
  meta: AdminPromotionPaginationMeta;
};

export type AdminProductDiscountResponse = {
  data: AdminProductDiscount;
};

export type AdminProductDiscountConfigRequest = {
  discount_type: AdminPromotionDiscountType;
  percentage_bps: number | null;
  fixed_amount: number | null;
};
