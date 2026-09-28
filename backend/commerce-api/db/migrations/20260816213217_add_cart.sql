-- Create "carts" table
CREATE TABLE "carts" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "cart_key" character varying(160) NOT NULL,
  "status" character varying(20) NOT NULL DEFAULT 'active',
  "currency" character varying(3) NOT NULL DEFAULT 'BDT',
  "expires_at" timestamptz NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "carts_cart_key_not_blank" CHECK (length(TRIM(BOTH FROM cart_key)) > 0),
  CONSTRAINT "carts_currency_valid" CHECK ((length((currency)::text) = 3) AND ((currency)::text = upper((currency)::text))),
  CONSTRAINT "carts_status_valid" CHECK ((status)::text = ANY ((ARRAY['active'::character varying, 'converted'::character varying, 'abandoned'::character varying, 'expired'::character varying])::text[]))
);
-- Create index "carts_cart_key_key" to table: "carts"
CREATE UNIQUE INDEX "carts_cart_key_key" ON "carts" ("cart_key");
-- Create index "idx_carts_active_expiry" to table: "carts"
CREATE INDEX "idx_carts_active_expiry" ON "carts" ("status", "expires_at") WHERE ((status)::text = 'active'::text);
-- Create index "idx_carts_status_updated" to table: "carts"
CREATE INDEX "idx_carts_status_updated" ON "carts" ("status", "updated_at");
-- Create "cart_items" table
CREATE TABLE "cart_items" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "cart_id" uuid NOT NULL,
  "variant_id" uuid NOT NULL,
  "quantity" integer NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "cart_items_cart_id_fkey" FOREIGN KEY ("cart_id") REFERENCES "carts" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "cart_items_variant_id_fkey" FOREIGN KEY ("variant_id") REFERENCES "product_variants" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "cart_items_quantity_positive" CHECK (quantity > 0)
);
-- Create index "cart_items_cart_variant_key" to table: "cart_items"
CREATE UNIQUE INDEX "cart_items_cart_variant_key" ON "cart_items" ("cart_id", "variant_id");
-- Create index "idx_cart_items_cart" to table: "cart_items"
CREATE INDEX "idx_cart_items_cart" ON "cart_items" ("cart_id");
-- Create index "idx_cart_items_variant" to table: "cart_items"
CREATE INDEX "idx_cart_items_variant" ON "cart_items" ("variant_id");
