export type AdminAnalyticsGranularity = "day" | "week";

export type AdminAnalyticsMeta = {
  from: string;
  to: string;
  timezone: string;
  granularity?: AdminAnalyticsGranularity;
  currency?: string;
};

export type AdminAnalyticsResponse<T> = {
  data: T;
  meta: AdminAnalyticsMeta;
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

export type AdminSalesPoint = {
  bucket_start: string;
  orders_created: number;
  collected_orders: number;
  new_customers: number;
  gross_collected_revenue_amount: number;
  successful_refund_amount: number;
  net_collected_revenue_amount: number;
};

export type AdminCheckoutFunnel = {
  carts_created: number;
  checkout_started_carts: number;
  ordered_carts: number;
  collected_carts: number;
  cart_to_checkout_bps: number;
  checkout_to_order_bps: number;
  order_to_collected_bps: number;
  cart_to_collected_bps: number;
};

export type AdminTopProduct = {
  variant_id: string;
  product_id: string;
  sku: string;
  product_name: string;
  collected_orders: number;
  units_sold: number;
  gross_collected_merchandise_amount: number;
  currency: string;
};

export type AdminPromotionAttribution = {
  promotion_id: string;
  name: string;
  code?: string;
  collected_orders: number;
  units_sold: number;
  gross_collected_revenue_amount: number;
  discount_amount: number;
  average_gross_order_value_amount: number;
  currency: string;
};

export type AdminFinanceExpenseCategory = {
  category: string;
  entries: number;
  amount: number;
};

export type AdminProfitLoss = {
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

export type AdminProfitLossPoint = {
  bucket_start: string;

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

  restocked_units: number;
  restock_costed_units: number;
  restock_missing_cost_units: number;

  gross_cogs_amount: number;
  restock_cogs_recovery_amount: number;
  net_cogs_amount: number;
  cogs_coverage_bps: number;

  known_gross_profit_amount: number;
  gross_profit_amount: number | null;
  gross_margin_bps: number | null;
  profit_complete: boolean;

  recorded_expenses_amount: number;

  net_profit_after_recorded_expenses_amount: number | null;
};

export type AdminProductProfitability = {
  variant_id: string;
  product_id: string;

  sku: string;
  product_name: string;

  collected_orders: number;
  units_sold: number;

  gross_merchandise_revenue_amount: number;
  net_merchandise_revenue_amount: number;

  costed_units: number;
  missing_cost_units: number;
  cogs_coverage_bps: number;
  known_cogs_amount: number;

  gross_profit_before_refunds_amount: number | null;
  gross_margin_before_refunds_bps: number | null;

  profit_complete: boolean;

  currency: string;
};

export type AdminInsightEvidence = {
  label: string;
  value: string;
};

export type AdminInsightAction = {
  type: string;
  resource_id?: string;
};

export type AdminBusinessInsight = {
  type: string;
  severity: string;
  title: string;
  summary: string;
  evidence: AdminInsightEvidence[];
  action?: AdminInsightAction;
};

export type AdminInventoryRunRateRisk = {
  variant_id: string;
  product_id: string;
  sku: string;
  product_name: string;

  available_units: number;
  units_sold: number;

  daily_units_run_rate: number;
  days_of_cover: number;
};

export type AdminBusinessInsights = {
  signals: AdminBusinessInsight[];
  inventory_run_rate_risks: AdminInventoryRunRateRisk[];
};

export type AdminRecommendationPerformance = {
  impressions: number;
  clicks: number;
  add_to_carts: number;

  click_through_rate_bps: number;
  add_to_cart_rate_bps: number;

  attributed_orders: number;
  attributed_units: number;
  attributed_merchandise_amount: number;

  attribution_window_days: number;

  currency: string;
};

export type AdminRecommendationBreakdown = {
  key: string;

  impressions: number;
  clicks: number;
  add_to_carts: number;

  click_through_rate_bps: number;
  add_to_cart_rate_bps: number;

  attributed_orders: number;
  attributed_units: number;
  attributed_merchandise_amount: number;

  currency: string;
};

export type AdminRecommendationTrendPoint = {
  bucket_start: string;

  impressions: number;
  clicks: number;
  add_to_carts: number;

  attributed_orders: number;
  attributed_units: number;
  attributed_merchandise_amount: number;
};

export type AdminRecommendationProduct = {
  product_id: string;
  product_name: string;

  impressions: number;
  clicks: number;
  add_to_carts: number;

  click_through_rate_bps: number;

  attributed_orders: number;
  attributed_units: number;
  attributed_merchandise_amount: number;

  currency: string;
};

export type AdminRecommendationEngineDiagnostics = {
  eligible_products: number;
  customers_with_active_intent: number;
  active_search_signals: number;

  related_relations: number;
  complementary_relations: number;

  behavior_window_hours: number;
};

export type AdminRecommendationCategoryRelation = {
  source_category_id: string;
  source_category_name: string;

  target_category_id: string;
  target_category_name: string;

  relation_type: string;
  weight: number;
};

export type AdminRecommendationEngineResponse = {
  data: AdminRecommendationEngineDiagnostics;
};

export type AdminRecommendationRelationsResponse = {
  data: AdminRecommendationCategoryRelation[];
};