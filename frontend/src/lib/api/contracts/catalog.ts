export type CategorySummary = {
  id: string;
  name: string;
  slug: string;
};

export type CategoryNode = CategorySummary & {
  parent_id?: string;
  description?: string;
  image_url?: string | null;
  icon_url?: string | null;
  sort_order: number;
  children: CategoryNode[];
};

export type CategoryDetail = CategorySummary & {
  parent_id?: string;
  description?: string;
  image_url?: string | null;
  icon_url?: string | null;
  sort_order: number;
  is_active: boolean;
  created_at: string;
  updated_at: string;
};

export type ProductCard = {
  id: string;
  product_code: string;
  name: string;
  slug: string;
  brand?: string;
  short_description?: string;
  is_featured: boolean;
  primary_image_url?: string;
  price_amount: number;
  currency: string;
  in_stock: boolean;

  category?: CategorySummary;
  match_type?: string;
};

export type ProductListMeta = {
  page: number;
  limit: number;
  total: number;
  total_pages: number;
  has_next: boolean;
  has_previous: boolean;
  sort: string;
};

export type ProductListResponse = {
  data: ProductCard[];
  meta: ProductListMeta;
};

export type CategoryTreeResponse = {
  data: CategoryNode[];
};

export type CategoryDetailResponse = {
  data: CategoryDetail;
};

export type ProductImage = {
  id: string;
  variant_id?: string;
  url: string;
  alt_text?: string;
  sort_order: number;
  is_primary: boolean;
};

export type ProductPriceTier = {
  min_quantity: number;
  unit_price_amount: number;
};

export type ProductVariant = {
  id: string;
  sku: string;

  color_name?: string | null;
  color_hex?: string | null;
  size?: string | null;

  minimum_order_quantity: number;
  order_increment: number;

  price_amount: number;
  compare_at_price_amount?: number | null;
  currency: string;

  weight_grams?: number | null;

  available_quantity: number;
  in_stock: boolean;

  price_tiers: ProductPriceTier[];
};

/* =========================================================
   360° MEDIA
========================================================= */

export type ProductSpin360Frame = {
  url: string;
  frame: number;
};

/*
 * Resolved viewer data.
 *
 * This is NOT returned in the normal product-detail
 * response. Frames are loaded only after the customer
 * activates the 360° viewer.
 */
export type ProductSpin360Media = {
  poster_url?: string | null;

  frames: ProductSpin360Frame[];
};

/*
 * Variant-specific lightweight 360 metadata returned
 * with the product detail response.
 */
export type ProductSpin360VariantSummary = {
  variant_id: string;

  frame_count: number;

  poster_url: string;
};

/*
 * Lightweight product-detail metadata.
 *
 * This lets the storefront know whether the 360° mode
 * should be shown without downloading every frame.
 */
export type ProductSpin360Summary = {
  available: boolean;

  frame_count?: number;

  poster_url?: string | null;

  variants?: ProductSpin360VariantSummary[];
};

/*
 * Response from:
 *
 * GET /api/v1/products/:slug/immersive/360
 */
export type ProductSpin360FrameSet = {
  variant_id?: string | null;

  frames: ProductSpin360Frame[];
};

export type ProductSpin360FrameResponse = {
  data: ProductSpin360FrameSet;
};

/* =========================================================
   3D MEDIA
========================================================= */

/*
 * Final model passed into the 3D viewer.
 */
export type ProductModel3DMedia = {
  glb_url: string;

  poster_url?: string | null;
};

/*
 * Optional variant-specific model metadata.
 */
export type ProductModel3DVariantSummary = {
  variant_id: string;

  glb_url: string;

  poster_url?: string | null;
};

/*
 * Lightweight metadata returned by the product-detail API.
 *
 * Unlike a 360 frame set, we only need the GLB URL here.
 * The browser will not download the GLB itself until the
 * 3D viewer is activated.
 */
export type ProductModel3DSummary = {
  available: boolean;

  glb_url?: string | null;

  poster_url?: string | null;

  variants?: ProductModel3DVariantSummary[];
};

/* =========================================================
   TRY-ON
   Reserved for the later AI phase.
========================================================= */

export type ProductTryOnMedia = {
  enabled: boolean;

  category?:
    | "upper_body"
    | "lower_body"
    | "full_body"
    | "accessory"
    | string;
};

/* =========================================================
   IMMERSIVE PRODUCT MEDIA
========================================================= */

export type ProductImmersiveMedia = {
  spin_360?:
    | ProductSpin360Summary
    | null;

  model_3d?:
    | ProductModel3DSummary
    | null;

  try_on?:
    | ProductTryOnMedia
    | null;
};

/* =========================================================
   PRODUCT DETAIL
========================================================= */

export type ProductDetail = {
  id: string;

  category_id: string;

  product_code: string;

  name: string;

  slug: string;

  brand?: string;

  short_description?: string;

  description?: string;

  is_featured: boolean;

  published_at?: string;

  created_at: string;

  updated_at: string;

  category: CategorySummary;

  images: ProductImage[];

  variants: ProductVariant[];

  /*
   * Optional so ordinary products remain lightweight.
   */
  immersive_media?:
    | ProductImmersiveMedia
    | null;

  /*
   * Optional PDP enrichment.
   */
  rating_average?:
    | number
    | null;

  review_count?:
    | number
    | null;

  sold_quantity?:
    | number
    | null;
};

export type ProductDetailResponse = {
  data: ProductDetail;
};

/* =========================================================
   SEARCH
========================================================= */

export type SearchCategoryFacet = {
  id: string;
  name: string;
  slug: string;
  count: number;
};

export type SearchBrandFacet = {
  name: string;
  count: number;
};

export type SearchFacets = {
  categories: SearchCategoryFacet[];

  brands: SearchBrandFacet[];

  price: {
    min_amount: number;
    max_amount: number;
  };

  stock: {
    in_stock: number;
    out_of_stock: number;
  };
};

export type SearchResponse = {
  data: ProductCard[];

  facets: SearchFacets;

  meta: {
    query: string;
    page: number;
    limit: number;
    total: number;
    total_pages: number;
    sort: string;
    no_match: boolean;
  };
};

export type ActivePromotionsResponse = {
  data: unknown[];
};