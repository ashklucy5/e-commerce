export type AdminCatalogImportBatchStatus =
  | "uploaded"
  | "parsing"
  | "validating"
  | "ready"
  | "applying"
  | "completed"
  | "failed";

export type AdminCatalogImportRowStatus =
  | "pending"
  | "valid"
  | "invalid"
  | "applied"
  | "skipped"
  | "failed";

export type AdminCatalogImportAction =
  | "create"
  | "update"
  | "skip"
  | "";

export type AdminCatalogImportValidationError = {
  sheet: string;
  row: number;
  field?: string;
  code: string;
  message: string;
};

export type AdminCatalogImportBatchSummary = {
  id: string;
  status:
    AdminCatalogImportBatchStatus;

  total_rows: number;
  valid_rows: number;
  failed_rows: number;

  created_products: number;
  updated_products: number;

  created_variants: number;
  updated_variants: number;

  created_categories: number;
};

export type AdminCatalogImportStageResult = {
  batch:
    AdminCatalogImportBatchSummary;

  errors:
    AdminCatalogImportValidationError[];
};

export type AdminCatalogImportBatchListItem =
  AdminCatalogImportBatchSummary & {
    source_filename: string;
    storage_key: string;

    checksum_sha256?: string;
    last_error?: string;

    created_at: string;
    updated_at: string;
  };

export type AdminCatalogImportBatchListResult = {
  items:
    AdminCatalogImportBatchListItem[];

  limit: number;
  offset: number;
};

export type AdminCatalogImportRowPreview = {
  id: string;

  sheet_name: string;
  row_number: number;

  product_code?: string;
  sku?: string;

  action:
    AdminCatalogImportAction;

  status:
    AdminCatalogImportRowStatus;

  raw_data:
    | Record<string, unknown>
    | unknown[]
    | null;

  errors?:
    AdminCatalogImportValidationError[];
};

export type AdminCatalogImportPreviewResult = {
  batch:
    AdminCatalogImportBatchSummary;

  rows:
    AdminCatalogImportRowPreview[];
};

export type AdminCatalogImportApplyResult = {
  batch_id: string;

  status:
    | "completed"
    | "applying";

  applied_rows: number;
  skipped_rows: number;
};