-- Create "invoices" table
CREATE TABLE "invoices" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "order_id" uuid NOT NULL,
  "customer_id" uuid NULL,
  "invoice_number" character varying(80) NOT NULL,
  "issued_at" timestamptz NOT NULL,
  "currency" character varying(3) NOT NULL,
  "subtotal_amount" bigint NOT NULL,
  "discount_amount" bigint NOT NULL DEFAULT 0,
  "shipping_amount" bigint NOT NULL DEFAULT 0,
  "total_amount" bigint NOT NULL,
  "customer_name" character varying(160) NOT NULL,
  "customer_phone" character varying(40) NOT NULL,
  "customer_email" character varying(255) NULL,
  "shipping_address_line1" character varying(255) NOT NULL,
  "shipping_address_line2" character varying(255) NULL,
  "shipping_city" character varying(120) NOT NULL,
  "shipping_area" character varying(120) NOT NULL,
  "shipping_postal_code" character varying(30) NULL,
  "delivery_method" character varying(80) NOT NULL,
  "payment_method" character varying(80) NOT NULL,
  "payment_status_at_issue" character varying(30) NOT NULL,
  "merchant_snapshot" jsonb NOT NULL DEFAULT '{}',
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "invoices_customer_id_fkey" FOREIGN KEY ("customer_id") REFERENCES "customers" ("id") ON UPDATE NO ACTION ON DELETE SET NULL,
  CONSTRAINT "invoices_order_id_fkey" FOREIGN KEY ("order_id") REFERENCES "orders" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "invoices_currency_valid" CHECK ((length((currency)::text) = 3) AND ((currency)::text = upper((currency)::text))),
  CONSTRAINT "invoices_customer_email_not_blank" CHECK ((customer_email IS NULL) OR (length(TRIM(BOTH FROM customer_email)) > 0)),
  CONSTRAINT "invoices_customer_name_not_blank" CHECK (length(TRIM(BOTH FROM customer_name)) > 0),
  CONSTRAINT "invoices_customer_phone_not_blank" CHECK (length(TRIM(BOTH FROM customer_phone)) > 0),
  CONSTRAINT "invoices_delivery_method_not_blank" CHECK (length(TRIM(BOTH FROM delivery_method)) > 0),
  CONSTRAINT "invoices_discount_nonnegative" CHECK (discount_amount >= 0),
  CONSTRAINT "invoices_discount_not_above_subtotal" CHECK (discount_amount <= subtotal_amount),
  CONSTRAINT "invoices_invoice_number_not_blank" CHECK (length(TRIM(BOTH FROM invoice_number)) > 0),
  CONSTRAINT "invoices_merchant_snapshot_object" CHECK (jsonb_typeof(merchant_snapshot) = 'object'::text),
  CONSTRAINT "invoices_payment_method_not_blank" CHECK (length(TRIM(BOTH FROM payment_method)) > 0),
  CONSTRAINT "invoices_payment_status_valid" CHECK ((payment_status_at_issue)::text = ANY ((ARRAY['pending'::character varying, 'paid'::character varying, 'failed'::character varying, 'expired'::character varying, 'cod_pending'::character varying, 'cod_collected'::character varying, 'refunded'::character varying])::text[])),
  CONSTRAINT "invoices_shipping_address_line1_not_blank" CHECK (length(TRIM(BOTH FROM shipping_address_line1)) > 0),
  CONSTRAINT "invoices_shipping_area_not_blank" CHECK (length(TRIM(BOTH FROM shipping_area)) > 0),
  CONSTRAINT "invoices_shipping_city_not_blank" CHECK (length(TRIM(BOTH FROM shipping_city)) > 0),
  CONSTRAINT "invoices_shipping_nonnegative" CHECK (shipping_amount >= 0),
  CONSTRAINT "invoices_subtotal_nonnegative" CHECK (subtotal_amount >= 0),
  CONSTRAINT "invoices_total_consistent" CHECK (total_amount = ((subtotal_amount - discount_amount) + shipping_amount)),
  CONSTRAINT "invoices_total_nonnegative" CHECK (total_amount >= 0)
);
-- Create index "idx_invoices_customer_issued" to table: "invoices"
CREATE INDEX "idx_invoices_customer_issued" ON "invoices" ("customer_id", "issued_at");
-- Create index "idx_invoices_issued" to table: "invoices"
CREATE INDEX "idx_invoices_issued" ON "invoices" ("issued_at", "id");
-- Create index "invoices_invoice_number_key" to table: "invoices"
CREATE UNIQUE INDEX "invoices_invoice_number_key" ON "invoices" ("invoice_number");
-- Create index "invoices_order_id_key" to table: "invoices"
CREATE UNIQUE INDEX "invoices_order_id_key" ON "invoices" ("order_id");
-- Modify "order_items" table
ALTER TABLE "order_items" ADD CONSTRAINT "order_items_cost_currency_valid" CHECK ((cost_currency IS NULL) OR ((length((cost_currency)::text) = 3) AND ((cost_currency)::text = upper((cost_currency)::text)))), ADD CONSTRAINT "order_items_cost_snapshot_consistent" CHECK (((unit_cost_amount IS NULL) AND (line_cost_amount IS NULL) AND (cost_currency IS NULL)) OR ((unit_cost_amount IS NOT NULL) AND (line_cost_amount IS NOT NULL) AND (cost_currency IS NOT NULL) AND ((line_cost_amount)::numeric = ((unit_cost_amount)::numeric * (quantity)::numeric)))), ADD CONSTRAINT "order_items_line_cost_nonnegative" CHECK ((line_cost_amount IS NULL) OR (line_cost_amount >= 0)), ADD CONSTRAINT "order_items_unit_cost_nonnegative" CHECK ((unit_cost_amount IS NULL) OR (unit_cost_amount >= 0)), ADD COLUMN "unit_cost_amount" bigint NULL, ADD COLUMN "line_cost_amount" bigint NULL, ADD COLUMN "cost_currency" character varying(3) NULL;
-- Create "invoice_items" table
CREATE TABLE "invoice_items" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "invoice_id" uuid NOT NULL,
  "order_item_id" uuid NOT NULL,
  "variant_id" uuid NOT NULL,
  "sku" character varying(100) NOT NULL,
  "product_name" character varying(180) NOT NULL,
  "quantity" integer NOT NULL,
  "unit_price_amount" bigint NOT NULL,
  "line_total_amount" bigint NOT NULL,
  "currency" character varying(3) NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "invoice_items_invoice_id_fkey" FOREIGN KEY ("invoice_id") REFERENCES "invoices" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "invoice_items_order_item_id_fkey" FOREIGN KEY ("order_item_id") REFERENCES "order_items" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "invoice_items_variant_id_fkey" FOREIGN KEY ("variant_id") REFERENCES "product_variants" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "invoice_items_currency_valid" CHECK ((length((currency)::text) = 3) AND ((currency)::text = upper((currency)::text))),
  CONSTRAINT "invoice_items_line_total_consistent" CHECK ((line_total_amount)::numeric = ((unit_price_amount)::numeric * (quantity)::numeric)),
  CONSTRAINT "invoice_items_line_total_nonnegative" CHECK (line_total_amount >= 0),
  CONSTRAINT "invoice_items_product_name_not_blank" CHECK (length(TRIM(BOTH FROM product_name)) > 0),
  CONSTRAINT "invoice_items_quantity_positive" CHECK (quantity > 0),
  CONSTRAINT "invoice_items_sku_not_blank" CHECK (length(TRIM(BOTH FROM sku)) > 0),
  CONSTRAINT "invoice_items_unit_price_nonnegative" CHECK (unit_price_amount >= 0)
);
-- Create index "idx_invoice_items_invoice" to table: "invoice_items"
CREATE INDEX "idx_invoice_items_invoice" ON "invoice_items" ("invoice_id", "created_at");
-- Create index "idx_invoice_items_variant" to table: "invoice_items"
CREATE INDEX "idx_invoice_items_variant" ON "invoice_items" ("variant_id");
-- Create index "invoice_items_order_item_id_key" to table: "invoice_items"
CREATE UNIQUE INDEX "invoice_items_order_item_id_key" ON "invoice_items" ("order_item_id");
