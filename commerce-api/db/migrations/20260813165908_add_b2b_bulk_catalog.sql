-- Modify "products" table
ALTER TABLE "products" ADD CONSTRAINT "products_product_code_not_blank" CHECK (length(TRIM(BOTH FROM product_code)) > 0), ADD COLUMN "product_code" character varying(100) NOT NULL;
-- Create index "products_product_code_key" to table: "products"
CREATE UNIQUE INDEX "products_product_code_key" ON "products" ("product_code");
-- Create "catalog_import_batches" table
CREATE TABLE "catalog_import_batches" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "source_filename" character varying(255) NOT NULL,
  "file_storage_key" text NOT NULL,
  "image_archive_storage_key" text NULL,
  "file_checksum_sha256" character varying(64) NULL,
  "status" character varying(30) NOT NULL DEFAULT 'uploaded',
  "total_rows" integer NOT NULL DEFAULT 0,
  "valid_rows" integer NOT NULL DEFAULT 0,
  "failed_rows" integer NOT NULL DEFAULT 0,
  "created_products" integer NOT NULL DEFAULT 0,
  "updated_products" integer NOT NULL DEFAULT 0,
  "created_variants" integer NOT NULL DEFAULT 0,
  "updated_variants" integer NOT NULL DEFAULT 0,
  "created_categories" integer NOT NULL DEFAULT 0,
  "last_error" text NULL,
  "started_at" timestamptz NULL,
  "completed_at" timestamptz NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "catalog_import_batches_counts_nonnegative" CHECK ((total_rows >= 0) AND (valid_rows >= 0) AND (failed_rows >= 0) AND (created_products >= 0) AND (updated_products >= 0) AND (created_variants >= 0) AND (updated_variants >= 0) AND (created_categories >= 0)),
  CONSTRAINT "catalog_import_batches_filename_not_blank" CHECK (length(TRIM(BOTH FROM source_filename)) > 0),
  CONSTRAINT "catalog_import_batches_status_valid" CHECK ((status)::text = ANY ((ARRAY['uploaded'::character varying, 'parsing'::character varying, 'validating'::character varying, 'ready'::character varying, 'applying'::character varying, 'completed'::character varying, 'failed'::character varying])::text[])),
  CONSTRAINT "catalog_import_batches_storage_key_not_blank" CHECK (length(TRIM(BOTH FROM file_storage_key)) > 0)
);
-- Create index "idx_catalog_import_batches_status_created" to table: "catalog_import_batches"
CREATE INDEX "idx_catalog_import_batches_status_created" ON "catalog_import_batches" ("status", "created_at");
-- Create "catalog_import_rows" table
CREATE TABLE "catalog_import_rows" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "batch_id" uuid NOT NULL,
  "sheet_name" character varying(100) NOT NULL,
  "row_number" integer NOT NULL,
  "product_code" character varying(100) NULL,
  "sku" character varying(100) NULL,
  "action" character varying(20) NULL,
  "status" character varying(20) NOT NULL DEFAULT 'pending',
  "raw_data" jsonb NOT NULL,
  "error_details" jsonb NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "catalog_import_rows_batch_id_fkey" FOREIGN KEY ("batch_id") REFERENCES "catalog_import_batches" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "catalog_import_rows_action_valid" CHECK ((action IS NULL) OR ((action)::text = ANY ((ARRAY['create'::character varying, 'update'::character varying, 'skip'::character varying])::text[]))),
  CONSTRAINT "catalog_import_rows_row_number_positive" CHECK (row_number > 0),
  CONSTRAINT "catalog_import_rows_sheet_name_not_blank" CHECK (length(TRIM(BOTH FROM sheet_name)) > 0),
  CONSTRAINT "catalog_import_rows_status_valid" CHECK ((status)::text = ANY ((ARRAY['pending'::character varying, 'valid'::character varying, 'invalid'::character varying, 'applied'::character varying, 'skipped'::character varying, 'failed'::character varying])::text[]))
);
-- Create index "catalog_import_rows_batch_sheet_row_key" to table: "catalog_import_rows"
CREATE UNIQUE INDEX "catalog_import_rows_batch_sheet_row_key" ON "catalog_import_rows" ("batch_id", "sheet_name", "row_number");
-- Create index "idx_catalog_import_rows_batch_status" to table: "catalog_import_rows"
CREATE INDEX "idx_catalog_import_rows_batch_status" ON "catalog_import_rows" ("batch_id", "status");
-- Modify "product_variants" table
ALTER TABLE "product_variants" ADD CONSTRAINT "product_variants_minimum_order_quantity_positive" CHECK (minimum_order_quantity > 0), ADD CONSTRAINT "product_variants_order_increment_positive" CHECK (order_increment > 0), ADD COLUMN "minimum_order_quantity" integer NOT NULL DEFAULT 1, ADD COLUMN "order_increment" integer NOT NULL DEFAULT 1;
-- Create index "product_variants_id_product_id_key" to table: "product_variants"
CREATE UNIQUE INDEX "product_variants_id_product_id_key" ON "product_variants" ("id", "product_id");
-- Modify "product_images" table
ALTER TABLE "product_images" ADD COLUMN "variant_id" uuid NULL, ADD CONSTRAINT "product_images_variant_product_fkey" FOREIGN KEY ("variant_id", "product_id") REFERENCES "product_variants" ("id", "product_id") ON UPDATE NO ACTION ON DELETE CASCADE;
-- Create index "idx_product_images_variant" to table: "product_images"
CREATE INDEX "idx_product_images_variant" ON "product_images" ("variant_id");
-- Create "product_variant_price_tiers" table
CREATE TABLE "product_variant_price_tiers" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "variant_id" uuid NOT NULL,
  "min_quantity" integer NOT NULL,
  "unit_price_amount" bigint NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "product_variant_price_tiers_variant_id_fkey" FOREIGN KEY ("variant_id") REFERENCES "product_variants" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "product_variant_price_tiers_min_quantity_positive" CHECK (min_quantity > 0),
  CONSTRAINT "product_variant_price_tiers_price_nonnegative" CHECK (unit_price_amount >= 0)
);
-- Create index "product_variant_price_tiers_variant_quantity_key" to table: "product_variant_price_tiers"
CREATE UNIQUE INDEX "product_variant_price_tiers_variant_quantity_key" ON "product_variant_price_tiers" ("variant_id", "min_quantity");
