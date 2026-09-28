-- Drop index "payments_one_succeeded_per_order" from table: "payments"
DROP INDEX "payments_one_succeeded_per_order";
-- Create index "payments_one_succeeded_per_order" to table: "payments"
CREATE UNIQUE INDEX "payments_one_succeeded_per_order" ON "payments" ("order_id") WHERE ((status)::text = ANY ((ARRAY['succeeded'::character varying, 'refunded'::character varying])::text[]));
-- Create "promotions" table
CREATE TABLE "promotions" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "name" character varying(160) NOT NULL,
  "code" character varying(64) NULL,
  "discount_type" character varying(20) NOT NULL,
  "percentage_bps" integer NULL,
  "fixed_amount" bigint NULL,
  "minimum_subtotal_amount" bigint NOT NULL DEFAULT 0,
  "maximum_discount_amount" bigint NULL,
  "currency" character varying(3) NOT NULL DEFAULT 'BDT',
  "status" character varying(20) NOT NULL DEFAULT 'draft',
  "starts_at" timestamptz NULL,
  "ends_at" timestamptz NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "promotions_code_valid" CHECK ((code IS NULL) OR (((code)::text = TRIM(BOTH FROM code)) AND (length((code)::text) > 0) AND ((code)::text = upper((code)::text)))),
  CONSTRAINT "promotions_currency_valid" CHECK ((length((currency)::text) = 3) AND ((currency)::text = upper((currency)::text))),
  CONSTRAINT "promotions_discount_configuration_valid" CHECK ((((discount_type)::text = 'percentage'::text) AND (percentage_bps IS NOT NULL) AND ((percentage_bps >= 1) AND (percentage_bps <= 10000)) AND (fixed_amount IS NULL)) OR (((discount_type)::text = 'fixed'::text) AND (fixed_amount IS NOT NULL) AND (fixed_amount > 0) AND (percentage_bps IS NULL))),
  CONSTRAINT "promotions_discount_type_valid" CHECK ((discount_type)::text = ANY ((ARRAY['percentage'::character varying, 'fixed'::character varying])::text[])),
  CONSTRAINT "promotions_maximum_discount_positive" CHECK ((maximum_discount_amount IS NULL) OR (maximum_discount_amount > 0)),
  CONSTRAINT "promotions_minimum_subtotal_nonnegative" CHECK (minimum_subtotal_amount >= 0),
  CONSTRAINT "promotions_name_not_blank" CHECK (length(TRIM(BOTH FROM name)) > 0),
  CONSTRAINT "promotions_status_valid" CHECK ((status)::text = ANY ((ARRAY['draft'::character varying, 'active'::character varying, 'disabled'::character varying])::text[])),
  CONSTRAINT "promotions_window_valid" CHECK ((starts_at IS NULL) OR (ends_at IS NULL) OR (ends_at > starts_at))
);
-- Create index "idx_promotions_status_window" to table: "promotions"
CREATE INDEX "idx_promotions_status_window" ON "promotions" ("status", "starts_at", "ends_at");
-- Create index "promotions_code_key" to table: "promotions"
CREATE UNIQUE INDEX "promotions_code_key" ON "promotions" ("code");
-- Modify "checkout_sessions" table
ALTER TABLE "checkout_sessions" ADD CONSTRAINT "checkout_sessions_promotion_code_valid" CHECK ((promotion_code IS NULL) OR ((promotion_id IS NOT NULL) AND ((promotion_code)::text = TRIM(BOTH FROM promotion_code)) AND (length((promotion_code)::text) > 0) AND ((promotion_code)::text = upper((promotion_code)::text)))), ADD COLUMN "promotion_id" uuid NULL, ADD COLUMN "promotion_code" character varying(64) NULL, ADD CONSTRAINT "checkout_sessions_promotion_id_fkey" FOREIGN KEY ("promotion_id") REFERENCES "promotions" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- Create index "idx_checkout_sessions_promotion" to table: "checkout_sessions"
CREATE INDEX "idx_checkout_sessions_promotion" ON "checkout_sessions" ("promotion_id");
-- Modify "orders" table
ALTER TABLE "orders" ADD CONSTRAINT "orders_promotion_code_valid" CHECK ((promotion_code IS NULL) OR ((promotion_id IS NOT NULL) AND ((promotion_code)::text = TRIM(BOTH FROM promotion_code)) AND (length((promotion_code)::text) > 0) AND ((promotion_code)::text = upper((promotion_code)::text)))), ADD COLUMN "promotion_id" uuid NULL, ADD COLUMN "promotion_code" character varying(64) NULL, ADD CONSTRAINT "orders_promotion_id_fkey" FOREIGN KEY ("promotion_id") REFERENCES "promotions" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- Create index "idx_orders_promotion" to table: "orders"
CREATE INDEX "idx_orders_promotion" ON "orders" ("promotion_id");
