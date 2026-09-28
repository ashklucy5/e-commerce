-- Modify "promotions" table
ALTER TABLE "promotions" ADD CONSTRAINT "promotions_campaign_scope_valid" CHECK (((campaign_type)::text <> 'flash_sale'::text) OR ((scope)::text = 'product'::text)), ADD CONSTRAINT "promotions_campaign_type_valid" CHECK ((campaign_type)::text = ANY ((ARRAY['standard'::character varying, 'flash_sale'::character varying])::text[])), ADD CONSTRAINT "promotions_flash_sale_configuration_valid" CHECK (((campaign_type)::text <> 'flash_sale'::text) OR ((code IS NULL) AND (starts_at IS NOT NULL) AND (ends_at IS NOT NULL) AND (minimum_subtotal_amount = 0) AND (maximum_discount_amount IS NULL))), ADD CONSTRAINT "promotions_scope_valid" CHECK ((scope)::text = ANY ((ARRAY['order'::character varying, 'product'::character varying])::text[])), ADD COLUMN "scope" character varying(20) NOT NULL DEFAULT 'order', ADD COLUMN "campaign_type" character varying(20) NOT NULL DEFAULT 'standard';
-- Create index "idx_promotions_scope_campaign_status_window" to table: "promotions"
CREATE INDEX "idx_promotions_scope_campaign_status_window" ON "promotions" ("scope", "campaign_type", "status", "starts_at", "ends_at");
-- Create "promotion_targets" table
CREATE TABLE "promotion_targets" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "promotion_id" uuid NOT NULL,
  "product_id" uuid NULL,
  "variant_id" uuid NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "promotion_targets_product_id_fkey" FOREIGN KEY ("product_id") REFERENCES "products" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "promotion_targets_promotion_id_fkey" FOREIGN KEY ("promotion_id") REFERENCES "promotions" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "promotion_targets_variant_id_fkey" FOREIGN KEY ("variant_id") REFERENCES "product_variants" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "promotion_targets_exactly_one_target" CHECK (((product_id IS NOT NULL) AND (variant_id IS NULL)) OR ((product_id IS NULL) AND (variant_id IS NOT NULL)))
);
-- Create index "idx_promotion_targets_product" to table: "promotion_targets"
CREATE INDEX "idx_promotion_targets_product" ON "promotion_targets" ("product_id") WHERE (product_id IS NOT NULL);
-- Create index "idx_promotion_targets_variant" to table: "promotion_targets"
CREATE INDEX "idx_promotion_targets_variant" ON "promotion_targets" ("variant_id") WHERE (variant_id IS NOT NULL);
-- Create index "promotion_targets_promotion_product_key" to table: "promotion_targets"
CREATE UNIQUE INDEX "promotion_targets_promotion_product_key" ON "promotion_targets" ("promotion_id", "product_id") WHERE (product_id IS NOT NULL);
-- Create index "promotion_targets_promotion_variant_key" to table: "promotion_targets"
CREATE UNIQUE INDEX "promotion_targets_promotion_variant_key" ON "promotion_targets" ("promotion_id", "variant_id") WHERE (variant_id IS NOT NULL);
