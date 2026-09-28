export type AdminProductStatus =
  | "draft"
  | "active"
  | "archived";

export type AdminCatalogProductListItem = {
  id: string;

  category_id: string;
  category_name: string;

  product_code: string;

  name: string;
  slug: string;
  brand?: string;

  status: AdminProductStatus;
  is_featured: boolean;

  variant_count: number;
  active_variant_count: number;

  available_stock: number;

  created_at?: string;
  updated_at?: string;
};

export type AdminCatalogPaginationMeta = {
  limit?: number;
  offset?: number;
  total?: number;

  Limit?: number;
  Offset?: number;
  Total?: number;

  [key: string]: unknown;
};

export type AdminCatalogProductListResult = {
  items?: AdminCatalogProductListItem[];
  meta?: AdminCatalogPaginationMeta;

  /*
   * The current Go result struct
   * may serialize these using the
   * exported field names.
   */
  Items?: AdminCatalogProductListItem[];
  Meta?: AdminCatalogPaginationMeta;
};

export type AdminCatalogProductListResponse = {
  data:
    | AdminCatalogProductListItem[]
    | AdminCatalogProductListResult;
};

export type AdminCatalogPriceTier = {
  min_quantity: number;
  unit_price_amount: number;
};

export type AdminCatalogVariant = {
  id: string;
  sku: string;

  color_name?: string;
  color_hex?: string;
  size?: string;

  minimum_order_quantity?: number;
  order_increment?: number;

  price_amount?: number;
  compare_at_price_amount?:
    | number
    | null;

  currency?: string;

  is_active?: boolean;

  /*
   * The Admin detail query may
   * expose inventory using one
   * of these fields depending on
   * the backend projection.
   */
  stock?: number;
  available_stock?: number;
  available_quantity?: number;

  quantity_on_hand?: number;
  quantity_reserved?: number;

  reorder_level?: number;

  price_tiers?:
    AdminCatalogPriceTier[];
};

export type AdminCatalogImage = {
  id: string;

  product_id?: string;

  variant_id?:
    | string
    | null;

  url: string;

  alt_text?:
    | string
    | null;

  sort_order: number;
  is_primary: boolean;
};

export type AdminCatalogProductDetail =
  AdminCatalogProductListItem & {
    short_description?: string;
    description?: string;

    variants:
      AdminCatalogVariant[];

    images:
      AdminCatalogImage[];
  };

export type AdminCatalogProductDetailResponse = {
  data: AdminCatalogProductDetail;
};

export type AdminUpdateCatalogProductRequest = {
  category_id?: string;

  name?: string;
  slug?: string;

  brand?: string;

  short_description?: string;
  description?: string;

  status?: AdminProductStatus;
  is_featured?: boolean;
};