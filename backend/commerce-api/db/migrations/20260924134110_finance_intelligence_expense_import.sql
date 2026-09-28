-- Drop index "idx_finance_expenses_order" from table: "finance_expenses"
DROP INDEX "idx_finance_expenses_order";
-- Modify "finance_expenses" table
ALTER TABLE "finance_expenses" DROP CONSTRAINT "finance_expenses_category_valid", ADD CONSTRAINT "finance_expenses_category_valid" CHECK ((category)::text = ANY ((ARRAY['delivery'::character varying, 'payment_fee'::character varying, 'china_freight'::character varying, 'customs'::character varying, 'packaging'::character varying, 'marketing'::character varying, 'warehouse'::character varying, 'salary'::character varying, 'staff_benefit'::character varying, 'rent'::character varying, 'utilities'::character varying, 'software'::character varying, 'professional_service'::character varying, 'bank_fee'::character varying, 'insurance'::character varying, 'tax_fee'::character varying, 'office'::character varying, 'travel'::character varying, 'other'::character varying])::text[])), ADD CONSTRAINT "finance_expenses_single_scope_target" CHECK (num_nonnulls(order_id, product_id, variant_id, staff_account_id, warehouse_id) <= 1), ADD COLUMN "product_id" uuid NULL, ADD COLUMN "variant_id" uuid NULL, ADD COLUMN "staff_account_id" uuid NULL, ADD COLUMN "warehouse_id" uuid NULL, ADD CONSTRAINT "finance_expenses_product_id_fkey" FOREIGN KEY ("product_id") REFERENCES "products" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "finance_expenses_staff_account_id_fkey" FOREIGN KEY ("staff_account_id") REFERENCES "staff_accounts" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "finance_expenses_variant_id_fkey" FOREIGN KEY ("variant_id") REFERENCES "product_variants" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "finance_expenses_warehouse_id_fkey" FOREIGN KEY ("warehouse_id") REFERENCES "warehouses" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- Create index "idx_finance_expenses_order" to table: "finance_expenses"
CREATE INDEX "idx_finance_expenses_order" ON "finance_expenses" ("order_id", "occurred_at") WHERE (order_id IS NOT NULL);
-- Create index "idx_finance_expenses_product" to table: "finance_expenses"
CREATE INDEX "idx_finance_expenses_product" ON "finance_expenses" ("product_id", "occurred_at") WHERE (product_id IS NOT NULL);
-- Create index "idx_finance_expenses_staff" to table: "finance_expenses"
CREATE INDEX "idx_finance_expenses_staff" ON "finance_expenses" ("staff_account_id", "occurred_at") WHERE (staff_account_id IS NOT NULL);
-- Create index "idx_finance_expenses_variant" to table: "finance_expenses"
CREATE INDEX "idx_finance_expenses_variant" ON "finance_expenses" ("variant_id", "occurred_at") WHERE (variant_id IS NOT NULL);
-- Create index "idx_finance_expenses_warehouse" to table: "finance_expenses"
CREATE INDEX "idx_finance_expenses_warehouse" ON "finance_expenses" ("warehouse_id", "occurred_at") WHERE (warehouse_id IS NOT NULL);
-- Create "finance_import_batches" table
CREATE TABLE "finance_import_batches" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "source_filename" character varying(255) NOT NULL,
  "file_checksum_sha256" character varying(64) NOT NULL,
  "template_version" integer NOT NULL DEFAULT 1,
  "status" character varying(30) NOT NULL DEFAULT 'validating',
  "total_rows" integer NOT NULL DEFAULT 0,
  "valid_rows" integer NOT NULL DEFAULT 0,
  "invalid_rows" integer NOT NULL DEFAULT 0,
  "applied_expenses" integer NOT NULL DEFAULT 0,
  "applied_cost_updates" integer NOT NULL DEFAULT 0,
  "skipped_rows" integer NOT NULL DEFAULT 0,
  "created_by_staff_id" uuid NOT NULL,
  "last_error" text NULL,
  "applied_at" timestamptz NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "finance_import_batches_created_by_staff_id_fkey" FOREIGN KEY ("created_by_staff_id") REFERENCES "staff_accounts" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "finance_import_batches_checksum_valid" CHECK (length((file_checksum_sha256)::text) = 64),
  CONSTRAINT "finance_import_batches_counts_nonnegative" CHECK ((total_rows >= 0) AND (valid_rows >= 0) AND (invalid_rows >= 0) AND (applied_expenses >= 0) AND (applied_cost_updates >= 0) AND (skipped_rows >= 0)),
  CONSTRAINT "finance_import_batches_filename_not_blank" CHECK (length(TRIM(BOTH FROM source_filename)) > 0),
  CONSTRAINT "finance_import_batches_status_valid" CHECK ((status)::text = ANY ((ARRAY['validating'::character varying, 'ready'::character varying, 'applying'::character varying, 'completed'::character varying, 'failed'::character varying])::text[])),
  CONSTRAINT "finance_import_batches_template_version_positive" CHECK (template_version > 0),
  CONSTRAINT "finance_import_batches_updated_at_valid" CHECK (updated_at >= created_at)
);
-- Create index "finance_import_batches_file_checksum_sha256_key" to table: "finance_import_batches"
CREATE UNIQUE INDEX "finance_import_batches_file_checksum_sha256_key" ON "finance_import_batches" ("file_checksum_sha256");
-- Create index "idx_finance_import_batches_created_by" to table: "finance_import_batches"
CREATE INDEX "idx_finance_import_batches_created_by" ON "finance_import_batches" ("created_by_staff_id", "created_at");
-- Create index "idx_finance_import_batches_status_created" to table: "finance_import_batches"
CREATE INDEX "idx_finance_import_batches_status_created" ON "finance_import_batches" ("status", "created_at");
-- Create "finance_variant_cost_history" table
CREATE TABLE "finance_variant_cost_history" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "variant_id" uuid NOT NULL,
  "unit_cost_amount" bigint NOT NULL,
  "previous_unit_cost_amount" bigint NULL,
  "currency" character varying(3) NOT NULL,
  "effective_at" timestamptz NOT NULL,
  "source" character varying(20) NOT NULL,
  "description" character varying(500) NULL,
  "reference" character varying(160) NULL,
  "created_by_staff_id" uuid NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "finance_variant_cost_history_created_by_staff_id_fkey" FOREIGN KEY ("created_by_staff_id") REFERENCES "staff_accounts" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "finance_variant_cost_history_variant_id_fkey" FOREIGN KEY ("variant_id") REFERENCES "product_variants" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "finance_variant_cost_history_cost_nonnegative" CHECK (unit_cost_amount >= 0),
  CONSTRAINT "finance_variant_cost_history_currency_valid" CHECK ((length((currency)::text) = 3) AND ((currency)::text = upper((currency)::text))),
  CONSTRAINT "finance_variant_cost_history_description_not_blank" CHECK ((description IS NULL) OR (length(TRIM(BOTH FROM description)) > 0)),
  CONSTRAINT "finance_variant_cost_history_previous_cost_nonnegative" CHECK ((previous_unit_cost_amount IS NULL) OR (previous_unit_cost_amount >= 0)),
  CONSTRAINT "finance_variant_cost_history_reference_not_blank" CHECK ((reference IS NULL) OR (length(TRIM(BOTH FROM reference)) > 0)),
  CONSTRAINT "finance_variant_cost_history_source_valid" CHECK ((source)::text = ANY ((ARRAY['manual'::character varying, 'import'::character varying])::text[]))
);
-- Create index "idx_finance_variant_cost_history_created_by" to table: "finance_variant_cost_history"
CREATE INDEX "idx_finance_variant_cost_history_created_by" ON "finance_variant_cost_history" ("created_by_staff_id", "created_at");
-- Create index "idx_finance_variant_cost_history_effective" to table: "finance_variant_cost_history"
CREATE INDEX "idx_finance_variant_cost_history_effective" ON "finance_variant_cost_history" ("effective_at");
-- Create index "idx_finance_variant_cost_history_variant_effective" to table: "finance_variant_cost_history"
CREATE INDEX "idx_finance_variant_cost_history_variant_effective" ON "finance_variant_cost_history" ("variant_id", "effective_at");
-- Create "finance_import_rows" table
CREATE TABLE "finance_import_rows" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "batch_id" uuid NOT NULL,
  "row_number" integer NOT NULL,
  "record_type" character varying(30) NOT NULL,
  "status" character varying(20) NOT NULL,
  "raw_data" jsonb NOT NULL,
  "normalized_data" jsonb NULL,
  "error_details" jsonb NULL,
  "applied_expense_id" uuid NULL,
  "applied_cost_history_id" uuid NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "finance_import_rows_applied_cost_history_id_fkey" FOREIGN KEY ("applied_cost_history_id") REFERENCES "finance_variant_cost_history" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "finance_import_rows_applied_expense_id_fkey" FOREIGN KEY ("applied_expense_id") REFERENCES "finance_expenses" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "finance_import_rows_batch_id_fkey" FOREIGN KEY ("batch_id") REFERENCES "finance_import_batches" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "finance_import_rows_record_type_valid" CHECK ((record_type)::text = ANY ((ARRAY['expense'::character varying, 'variant_cost'::character varying])::text[])),
  CONSTRAINT "finance_import_rows_row_number_positive" CHECK (row_number > 0),
  CONSTRAINT "finance_import_rows_single_applied_target" CHECK (num_nonnulls(applied_expense_id, applied_cost_history_id) <= 1),
  CONSTRAINT "finance_import_rows_status_valid" CHECK ((status)::text = ANY ((ARRAY['valid'::character varying, 'invalid'::character varying, 'applied'::character varying, 'skipped'::character varying, 'failed'::character varying])::text[])),
  CONSTRAINT "finance_import_rows_updated_at_valid" CHECK (updated_at >= created_at)
);
-- Create index "finance_import_rows_batch_row_key" to table: "finance_import_rows"
CREATE UNIQUE INDEX "finance_import_rows_batch_row_key" ON "finance_import_rows" ("batch_id", "row_number");
-- Create index "idx_finance_import_rows_applied_cost_history" to table: "finance_import_rows"
CREATE INDEX "idx_finance_import_rows_applied_cost_history" ON "finance_import_rows" ("applied_cost_history_id") WHERE (applied_cost_history_id IS NOT NULL);
-- Create index "idx_finance_import_rows_applied_expense" to table: "finance_import_rows"
CREATE INDEX "idx_finance_import_rows_applied_expense" ON "finance_import_rows" ("applied_expense_id") WHERE (applied_expense_id IS NOT NULL);
-- Create index "idx_finance_import_rows_batch_status" to table: "finance_import_rows"
CREATE INDEX "idx_finance_import_rows_batch_status" ON "finance_import_rows" ("batch_id", "status");
