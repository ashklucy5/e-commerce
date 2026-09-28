-- Create "checkout_sessions" table
CREATE TABLE "checkout_sessions" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "checkout_key" character varying(160) NOT NULL,
  "cart_id" uuid NOT NULL,
  "status" character varying(20) NOT NULL DEFAULT 'active',
  "currency" character varying(3) NOT NULL DEFAULT 'BDT',
  "subtotal_amount" bigint NOT NULL DEFAULT 0,
  "discount_amount" bigint NOT NULL DEFAULT 0,
  "shipping_amount" bigint NOT NULL DEFAULT 0,
  "total_amount" bigint NOT NULL DEFAULT 0,
  "customer_name" character varying(160) NULL,
  "customer_phone" character varying(40) NULL,
  "customer_email" character varying(255) NULL,
  "shipping_address_line1" character varying(255) NULL,
  "shipping_address_line2" character varying(255) NULL,
  "shipping_city" character varying(120) NULL,
  "shipping_area" character varying(120) NULL,
  "shipping_postal_code" character varying(30) NULL,
  "delivery_method" character varying(80) NULL,
  "payment_method" character varying(80) NULL,
  "expires_at" timestamptz NOT NULL,
  "completed_at" timestamptz NULL,
  "cancelled_at" timestamptz NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "checkout_sessions_cart_id_fkey" FOREIGN KEY ("cart_id") REFERENCES "carts" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "checkout_sessions_checkout_key_not_blank" CHECK (length(TRIM(BOTH FROM checkout_key)) > 0),
  CONSTRAINT "checkout_sessions_currency_valid" CHECK ((length((currency)::text) = 3) AND ((currency)::text = upper((currency)::text))),
  CONSTRAINT "checkout_sessions_discount_nonnegative" CHECK (discount_amount >= 0),
  CONSTRAINT "checkout_sessions_discount_not_above_subtotal" CHECK (discount_amount <= subtotal_amount),
  CONSTRAINT "checkout_sessions_shipping_nonnegative" CHECK (shipping_amount >= 0),
  CONSTRAINT "checkout_sessions_status_valid" CHECK ((status)::text = ANY ((ARRAY['active'::character varying, 'completed'::character varying, 'cancelled'::character varying, 'expired'::character varying])::text[])),
  CONSTRAINT "checkout_sessions_subtotal_nonnegative" CHECK (subtotal_amount >= 0),
  CONSTRAINT "checkout_sessions_total_consistent" CHECK (total_amount = ((subtotal_amount - discount_amount) + shipping_amount)),
  CONSTRAINT "checkout_sessions_total_nonnegative" CHECK (total_amount >= 0)
);
-- Create index "checkout_sessions_checkout_key_key" to table: "checkout_sessions"
CREATE UNIQUE INDEX "checkout_sessions_checkout_key_key" ON "checkout_sessions" ("checkout_key");
-- Create index "checkout_sessions_one_active_per_cart" to table: "checkout_sessions"
CREATE UNIQUE INDEX "checkout_sessions_one_active_per_cart" ON "checkout_sessions" ("cart_id") WHERE ((status)::text = 'active'::text);
-- Create index "idx_checkout_sessions_cart" to table: "checkout_sessions"
CREATE INDEX "idx_checkout_sessions_cart" ON "checkout_sessions" ("cart_id");
-- Create index "idx_checkout_sessions_status_expiry" to table: "checkout_sessions"
CREATE INDEX "idx_checkout_sessions_status_expiry" ON "checkout_sessions" ("status", "expires_at");
-- Normalize legacy order increments.
-- Customer quantities are controlled by seller-defined MOQ
-- and available stock, not quantity multiples.
UPDATE "product_variants"
SET "order_increment" = 1
WHERE "order_increment" <> 1;
-- Modify "product_variants" table
ALTER TABLE "product_variants" DROP CONSTRAINT "product_variants_order_increment_positive", ADD CONSTRAINT "product_variants_order_increment_valid" CHECK (order_increment = 1);
-- Create "checkout_items" table
CREATE TABLE "checkout_items" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "checkout_id" uuid NOT NULL,
  "variant_id" uuid NOT NULL,
  "sku" character varying(100) NOT NULL,
  "product_name" character varying(180) NOT NULL,
  "quantity" integer NOT NULL,
  "minimum_order_quantity" integer NOT NULL,
  "unit_price_amount" bigint NOT NULL,
  "line_total_amount" bigint NOT NULL,
  "currency" character varying(3) NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "checkout_items_checkout_id_fkey" FOREIGN KEY ("checkout_id") REFERENCES "checkout_sessions" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "checkout_items_variant_id_fkey" FOREIGN KEY ("variant_id") REFERENCES "product_variants" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "checkout_items_currency_valid" CHECK ((length((currency)::text) = 3) AND ((currency)::text = upper((currency)::text))),
  CONSTRAINT "checkout_items_line_total_consistent" CHECK (line_total_amount = (unit_price_amount * quantity)),
  CONSTRAINT "checkout_items_line_total_nonnegative" CHECK (line_total_amount >= 0),
  CONSTRAINT "checkout_items_minimum_order_quantity_positive" CHECK (minimum_order_quantity > 0),
  CONSTRAINT "checkout_items_product_name_not_blank" CHECK (length(TRIM(BOTH FROM product_name)) > 0),
  CONSTRAINT "checkout_items_quantity_meets_minimum" CHECK (quantity >= minimum_order_quantity),
  CONSTRAINT "checkout_items_quantity_positive" CHECK (quantity > 0),
  CONSTRAINT "checkout_items_sku_not_blank" CHECK (length(TRIM(BOTH FROM sku)) > 0),
  CONSTRAINT "checkout_items_unit_price_nonnegative" CHECK (unit_price_amount >= 0)
);
-- Create index "checkout_items_checkout_variant_key" to table: "checkout_items"
CREATE UNIQUE INDEX "checkout_items_checkout_variant_key" ON "checkout_items" ("checkout_id", "variant_id");
-- Create index "idx_checkout_items_variant" to table: "checkout_items"
CREATE INDEX "idx_checkout_items_variant" ON "checkout_items" ("variant_id");
