-- Drop index "idx_product_sourcing_offers_request_id" from table: "product_sourcing_offers"
DROP INDEX "idx_product_sourcing_offers_request_id";
-- Modify "product_sourcing_offers" table
ALTER TABLE "product_sourcing_offers" DROP CONSTRAINT "product_sourcing_offers_status_valid", ADD CONSTRAINT "product_sourcing_offers_status_valid" CHECK ((status)::text = ANY ((ARRAY['draft'::character varying, 'sent'::character varying, 'accepted'::character varying, 'rejected'::character varying, 'customer_accepted'::character varying, 'customer_rejected'::character varying, 'superseded'::character varying, 'expired'::character varying, 'finalized'::character varying])::text[])), ADD CONSTRAINT "product_sourcing_offers_currency_valid" CHECK ((length((currency)::text) = 3) AND ((currency)::text = upper((currency)::text))), ADD CONSTRAINT "product_sourcing_offers_expiry_valid" CHECK ((expires_at IS NULL) OR (expires_at > created_at)), ADD CONSTRAINT "product_sourcing_offers_minimum_order_quantity_positive" CHECK ((minimum_order_quantity IS NULL) OR (minimum_order_quantity > 0)), ADD CONSTRAINT "product_sourcing_offers_quantity_meets_minimum" CHECK ((quoted_quantity IS NULL) OR (minimum_order_quantity IS NULL) OR (quoted_quantity >= minimum_order_quantity)), ADD CONSTRAINT "product_sourcing_offers_quoted_quantity_positive" CHECK ((quoted_quantity IS NULL) OR (quoted_quantity > 0)), ADD CONSTRAINT "product_sourcing_offers_shipping_price_nonnegative" CHECK (shipping_price >= 0), ADD CONSTRAINT "product_sourcing_offers_unit_price_nonnegative" CHECK (unit_price >= 0), ADD COLUMN "quoted_quantity" integer NULL, ADD COLUMN "minimum_order_quantity" integer NULL, ADD COLUMN "expires_at" timestamptz NULL, ADD COLUMN "sent_at" timestamptz NULL, ADD COLUMN "customer_responded_at" timestamptz NULL, ADD COLUMN "finalized_at" timestamptz NULL;
-- Create index "idx_product_sourcing_offers_request_id" to table: "product_sourcing_offers"
CREATE INDEX "idx_product_sourcing_offers_request_id" ON "product_sourcing_offers" ("request_id", "created_at");
-- Create index "product_sourcing_offers_one_finalized_per_request" to table: "product_sourcing_offers"
CREATE UNIQUE INDEX "product_sourcing_offers_one_finalized_per_request" ON "product_sourcing_offers" ("request_id") WHERE ((status)::text = 'finalized'::text);
-- Modify "product_sourcing_requests" table
ALTER TABLE "product_sourcing_requests" ADD CONSTRAINT "product_sourcing_requests_review_reason_not_blank" CHECK ((review_reason IS NULL) OR (length(TRIM(BOTH FROM review_reason)) > 0)), ADD CONSTRAINT "product_sourcing_requests_status_valid" CHECK ((status)::text = ANY ((ARRAY['pending_review'::character varying, 'accepted_for_sourcing'::character varying, 'rejected'::character varying, 'negotiating'::character varying, 'agreed'::character varying, 'converted_to_order'::character varying, 'cancelled'::character varying])::text[])), ADD COLUMN "status" character varying(30) NOT NULL DEFAULT 'pending_review', ADD COLUMN "review_reason" text NULL, ADD COLUMN "reviewed_by" uuid NULL, ADD COLUMN "reviewed_at" timestamptz NULL, ADD CONSTRAINT "product_sourcing_requests_reviewed_by_fkey" FOREIGN KEY ("reviewed_by") REFERENCES "support_actors" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;
-- Create index "idx_product_sourcing_requests_reviewed_by" to table: "product_sourcing_requests"
CREATE INDEX "idx_product_sourcing_requests_reviewed_by" ON "product_sourcing_requests" ("reviewed_by");
-- Create index "idx_product_sourcing_requests_status_updated" to table: "product_sourcing_requests"
CREATE INDEX "idx_product_sourcing_requests_status_updated" ON "product_sourcing_requests" ("status", "updated_at");
-- Create "product_sourcing_confirmations" table
CREATE TABLE "product_sourcing_confirmations" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "request_id" uuid NOT NULL,
  "offer_id" uuid NOT NULL,
  "quantity" integer NOT NULL,
  "minimum_order_quantity" integer NOT NULL,
  "accepted_product_name" character varying(180) NOT NULL,
  "accepted_specifications" jsonb NULL,
  "unit_price_snapshot" bigint NOT NULL,
  "shipping_price_snapshot" bigint NOT NULL,
  "currency" character varying(3) NOT NULL,
  "total_amount" bigint NOT NULL,
  "status" character varying(30) NOT NULL DEFAULT 'confirmed',
  "finalized_by" uuid NOT NULL,
  "created_product_id" uuid NULL,
  "created_variant_id" uuid NULL,
  "created_order_id" uuid NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "product_sourcing_confirmations_finalized_by_fkey" FOREIGN KEY ("finalized_by") REFERENCES "support_actors" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "product_sourcing_confirmations_offer_id_fkey" FOREIGN KEY ("offer_id") REFERENCES "product_sourcing_offers" ("id") ON UPDATE NO ACTION ON DELETE RESTRICT,
  CONSTRAINT "product_sourcing_confirmations_order_id_fkey" FOREIGN KEY ("created_order_id") REFERENCES "orders" ("id") ON UPDATE NO ACTION ON DELETE SET NULL,
  CONSTRAINT "product_sourcing_confirmations_product_id_fkey" FOREIGN KEY ("created_product_id") REFERENCES "products" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "product_sourcing_confirmations_request_id_fkey" FOREIGN KEY ("request_id") REFERENCES "product_sourcing_requests" ("id") ON UPDATE NO ACTION ON DELETE RESTRICT,
  CONSTRAINT "product_sourcing_confirmations_variant_id_fkey" FOREIGN KEY ("created_variant_id") REFERENCES "product_variants" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "product_sourcing_confirmations_currency_valid" CHECK ((length((currency)::text) = 3) AND ((currency)::text = upper((currency)::text))),
  CONSTRAINT "product_sourcing_confirmations_minimum_order_quantity_positive" CHECK (minimum_order_quantity > 0),
  CONSTRAINT "product_sourcing_confirmations_product_name_not_blank" CHECK (length(TRIM(BOTH FROM accepted_product_name)) > 0),
  CONSTRAINT "product_sourcing_confirmations_quantity_meets_minimum" CHECK (quantity >= minimum_order_quantity),
  CONSTRAINT "product_sourcing_confirmations_quantity_positive" CHECK (quantity > 0),
  CONSTRAINT "product_sourcing_confirmations_shipping_price_nonnegative" CHECK (shipping_price_snapshot >= 0),
  CONSTRAINT "product_sourcing_confirmations_specifications_is_object" CHECK ((accepted_specifications IS NULL) OR (jsonb_typeof(accepted_specifications) = 'object'::text)),
  CONSTRAINT "product_sourcing_confirmations_status_valid" CHECK ((status)::text = ANY ((ARRAY['confirmed'::character varying, 'order_created'::character varying, 'cancelled'::character varying])::text[])),
  CONSTRAINT "product_sourcing_confirmations_total_consistent" CHECK (total_amount = ((unit_price_snapshot * quantity) + shipping_price_snapshot)),
  CONSTRAINT "product_sourcing_confirmations_total_nonnegative" CHECK (total_amount >= 0),
  CONSTRAINT "product_sourcing_confirmations_unit_price_nonnegative" CHECK (unit_price_snapshot >= 0)
);
-- Create index "idx_product_sourcing_confirmations_product_id" to table: "product_sourcing_confirmations"
CREATE INDEX "idx_product_sourcing_confirmations_product_id" ON "product_sourcing_confirmations" ("created_product_id");
-- Create index "idx_product_sourcing_confirmations_variant_id" to table: "product_sourcing_confirmations"
CREATE INDEX "idx_product_sourcing_confirmations_variant_id" ON "product_sourcing_confirmations" ("created_variant_id");
-- Create index "product_sourcing_confirmations_offer_key" to table: "product_sourcing_confirmations"
CREATE UNIQUE INDEX "product_sourcing_confirmations_offer_key" ON "product_sourcing_confirmations" ("offer_id");
-- Create index "product_sourcing_confirmations_order_key" to table: "product_sourcing_confirmations"
CREATE UNIQUE INDEX "product_sourcing_confirmations_order_key" ON "product_sourcing_confirmations" ("created_order_id") WHERE (created_order_id IS NOT NULL);
-- Create index "product_sourcing_confirmations_request_key" to table: "product_sourcing_confirmations"
CREATE UNIQUE INDEX "product_sourcing_confirmations_request_key" ON "product_sourcing_confirmations" ("request_id");
