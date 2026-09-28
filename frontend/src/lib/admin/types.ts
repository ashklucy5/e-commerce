export type AdminApiErrorResponse = {
  error?:
    | string
    | {
        code?: string;
        message?: string;
      };

  code?: string;
  message?: string;
};

export type AdminStaff = {
  id: string;
  staff_code: string;
  full_name: string;
  email: string;
  status: string;
  roles: string[];
  permissions: string[];
};

export type AdminPrincipal = {
  session_id: string;
  staff: AdminStaff;
  authenticated_at: string;
  mfa_verified_at: string;
  access_expires_at: string;
};

export type AdminLoginChallenge = {
  challenge_token: string;
  challenge_expires_at: string;
  mfa_enrollment_required: boolean;
};

export type AdminLoginResponse = {
  data: AdminLoginChallenge;
};

export type AdminMfaEnrollment = {
  credential_id: string;
  label: string;
  secret: string;
  enrollment_uri: string;
  algorithm: string;
  digits: number;
  period_seconds: number;
};

export type AdminMfaEnrollmentResponse = {
  data: {
    enrollment: AdminMfaEnrollment;
  };
};

export type AdminSessionResponse = {
  data: {
    principal: AdminPrincipal;
    csrf_token: string;
    recovery_codes?: string[];
  };
};

export type AdminMeResponse = {
  data: {
    principal: AdminPrincipal;
  };
};

export type AdminCreatedVariant = {
  id: string;
  sku: string;
};

export type AdminCreateProductResponse = {
  data: {
    id: string;
    product_code: string;
    slug: string;
    variants: AdminCreatedVariant[];
  };
};

export type AdminCreateUploadRequest = {
  purpose: "product_image";
  filename: string;
  content_type: string;
  content_length: number;
};

export type AdminCreatedUpload = {
  provider: string;
  key: string;
  method: string;
  url: string;
  headers?: Record<string, string>;
  expires_at: string;
};

export type AdminCreateUploadResponse = {
  filename: string;
  purpose: "product_image";
  upload: AdminCreatedUpload;
};

export type AdminAttachProductImageRequest = {
  storage_key: string;
  variant_id: string;
  alt_text: string;
  sort_order: number;
  is_primary: boolean;
};


export type AdminCategory = {
  id: string;
  parent_id?: string;
  name: string;
  slug: string;
  description?: string;
  sort_order: number;
  is_active: boolean;
  product_code_prefix?: string;
  product_code_ready: boolean;
  children?: AdminCategory[];
};

export type AdminCategoryTreeResponse = {
  data: AdminCategory[];
};

export type AdminCreateCategoryResponse = {
  data: AdminCategory;
};
export type AdminDashboardSummary = {
  generated_at: string;
  active_products: number;
  active_variants: number;
  available_units: number;
  active_customers: number;
  orders_today: number;
  revenue_today_amount: number;
  open_crm_cases: number;
  awaiting_confirmation_shipments: number;
  pending_fulfillments: number;
  published_reviews: number;
};

export type AdminDashboardResponse = {
  data: AdminDashboardSummary;
};

export type AdminAnalyticsOverview = {
  generated_at: string;
  orders_created: number;
  collected_orders: number;
  units_sold: number;
  new_customers: number;
  gross_collected_revenue_amount: number;
  successful_refund_amount: number;
  net_collected_revenue_amount: number;
  discount_amount: number;
  average_order_value_amount: number;
  currency: string;
};

export type AdminAnalyticsMeta = {
  from: string;
  to: string;
  timezone: string;
  granularity?: string;
  currency?: string;
};

export type AdminAnalyticsOverviewResponse = {
  data: AdminAnalyticsOverview;
  meta: AdminAnalyticsMeta;
};

export type AdminSalesPoint = {
  bucket_start: string;
  orders_created: number;
  collected_orders: number;
  new_customers: number;
  gross_collected_revenue_amount: number;
  successful_refund_amount: number;
  net_collected_revenue_amount: number;
};

export type AdminSalesTrendResponse = {
  data: AdminSalesPoint[];
  meta: AdminAnalyticsMeta;
};

export type AdminFinanceExpenseCategory = {
  category: string;
  entries: number;
  amount: number;
};

export type AdminFinanceProfitLoss = {
  generated_at: string;
  collected_orders: number;
  units_sold: number;
  gross_merchandise_revenue_amount: number;
  discount_amount: number;
  net_merchandise_revenue_amount: number;
  shipping_revenue_amount: number;
  gross_collected_revenue_amount: number;
  successful_refund_amount: number;
  net_collected_revenue_amount: number;
  costed_units: number;
  missing_cost_units: number;
  sales_cogs_coverage_bps: number;
  gross_cogs_amount: number;
  restocked_units: number;
  restock_costed_units: number;
  restock_missing_cost_units: number;
  restock_cogs_coverage_bps: number;
  restock_cogs_recovery_amount: number;
  net_cogs_amount: number;
  cogs_coverage_bps: number;
  return_received_units: number;
  non_restocked_return_units: number;
  known_return_inventory_loss_amount: number;
  return_loss_missing_cost_units: number;
  known_gross_profit_amount: number;
  gross_profit_amount: number | null;
  gross_margin_bps: number | null;
  profit_complete: boolean;
  recorded_expense_entries: number;
  recorded_expenses_amount: number;
  expense_categories: AdminFinanceExpenseCategory[];
  foreign_currency_expense_entries_excluded: number;
  net_profit_after_recorded_expenses_amount: number | null;
  expense_basis: string;
  currency: string;
  warnings: string[];
};

export type AdminFinanceProfitLossResponse = {
  data: AdminFinanceProfitLoss;
  meta: AdminAnalyticsMeta;
};

export type AdminProductRequestCustomer = {
  id: string;
  full_name: string;
  phone: string;
  email?: string;
};

export type AdminProductRequest = {
  id: string;
  request_number: string;
  case_id: string;
  case_number: string;
  requested_product_name: string;
  description: string;
  requested_quantity: number;
  status: string;
  status_reason?: string;
  crm_status: string;
  crm_priority: string;
  customer: AdminProductRequestCustomer;
  last_message_at: string;
  last_customer_message_at?: string | null;
  last_support_message_at?: string | null;
  reviewed_at?: string | null;
  created_at: string;
  updated_at: string;
};

export type AdminProductRequestListResponse = {
  data: AdminProductRequest[];
  meta: {
    limit: number;
    offset: number;
  };
};
