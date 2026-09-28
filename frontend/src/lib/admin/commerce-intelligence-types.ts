export type AdminDataResponse<T> = {
  data: T;
};

export type FinanceExpenseCategoryAmount = {
  category: string;
  entries: number;
  amount: number;
};

export type FinanceProfitLoss = {
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

  expense_categories: FinanceExpenseCategoryAmount[];

  foreign_currency_expense_entries_excluded: number;

  net_profit_after_recorded_expenses_amount:
    | number
    | null;

  expense_basis: string;
  currency: string;

  warnings: string[];
};

export type FinanceProfitLossPoint = {
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

  net_profit_after_recorded_expenses_amount:
    | number
    | null;
};

export type ProductProfitability = {
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

  gross_profit_before_refunds_amount:
    | number
    | null;

  gross_margin_before_refunds_bps:
    | number
    | null;

  profit_complete: boolean;

  currency: string;
};

export type InsightEvidence = {
  label: string;
  value: string;
};

export type InsightAction = {
  type: string;
  resource_id?: string;
};

export type BusinessInsight = {
  type: string;
  severity: string;
  title: string;
  summary: string;

  evidence: InsightEvidence[];

  action?: InsightAction;
};

export type InventoryRunRateRisk = {
  variant_id: string;
  product_id: string;
  sku: string;
  product_name: string;

  available_units: number;
  units_sold: number;

  daily_units_run_rate: number;
  days_of_cover: number;
};

export type BusinessInsights = {
  signals: BusinessInsight[];

  inventory_run_rate_risks:
    InventoryRunRateRisk[];
};

export type FinanceProfitLossResponse =
  AdminDataResponse<FinanceProfitLoss>;

export type FinanceProfitLossTrendResponse =
  AdminDataResponse<FinanceProfitLossPoint[]>;

export type ProductProfitabilityResponse =
  AdminDataResponse<ProductProfitability[]>;

export type BusinessInsightsResponse =
  AdminDataResponse<BusinessInsights>;