-- Create "product_sourcing_offers" table
CREATE TABLE "product_sourcing_offers" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "request_id" uuid NOT NULL,
  "status" character varying(30) NOT NULL DEFAULT 'draft',
  "product_name" character varying(180) NOT NULL,
  "description" text NOT NULL,
  "attachments" jsonb NULL,
  "variants" jsonb NULL,
  "unit_price" bigint NOT NULL,
  "shipping_price" bigint NOT NULL DEFAULT 0,
  "currency" character varying(10) NOT NULL,
  "created_by" uuid NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "product_sourcing_offers_created_by_fkey" FOREIGN KEY ("created_by") REFERENCES "support_actors" ("id") ON UPDATE NO ACTION ON DELETE RESTRICT,
  CONSTRAINT "product_sourcing_offers_request_id_fkey" FOREIGN KEY ("request_id") REFERENCES "product_sourcing_requests" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "product_sourcing_offers_attachments_is_array" CHECK ((attachments IS NULL) OR (jsonb_typeof(attachments) = 'array'::text)),
  CONSTRAINT "product_sourcing_offers_description_not_blank" CHECK (length(TRIM(BOTH FROM description)) > 0),
  CONSTRAINT "product_sourcing_offers_product_name_not_blank" CHECK (length(TRIM(BOTH FROM product_name)) > 0),
  CONSTRAINT "product_sourcing_offers_status_valid" CHECK ((status)::text = ANY ((ARRAY['draft'::character varying, 'sent'::character varying, 'accepted'::character varying, 'rejected'::character varying, 'expired'::character varying])::text[])),
  CONSTRAINT "product_sourcing_offers_variants_is_object" CHECK ((variants IS NULL) OR (jsonb_typeof(variants) = 'object'::text))
);
-- Create index "idx_product_sourcing_offers_request_id" to table: "product_sourcing_offers"
CREATE INDEX "idx_product_sourcing_offers_request_id" ON "product_sourcing_offers" ("request_id");
-- Create index "idx_product_sourcing_offers_status" to table: "product_sourcing_offers"
CREATE INDEX "idx_product_sourcing_offers_status" ON "product_sourcing_offers" ("status");
