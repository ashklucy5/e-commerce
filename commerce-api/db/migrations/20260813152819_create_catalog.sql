-- Create "products" table
CREATE TABLE "products" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "category_id" uuid NOT NULL,
  "name" character varying(180) NOT NULL,
  "slug" character varying(200) NOT NULL,
  "brand" character varying(120) NULL,
  "short_description" character varying(500) NULL,
  "description" text NULL,
  "status" character varying(20) NOT NULL DEFAULT 'draft',
  "is_featured" boolean NOT NULL DEFAULT false,
  "published_at" timestamptz NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "products_category_id_fkey" FOREIGN KEY ("category_id") REFERENCES "categories" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "products_name_not_blank" CHECK (length(TRIM(BOTH FROM name)) > 0),
  CONSTRAINT "products_slug_not_blank" CHECK (length(TRIM(BOTH FROM slug)) > 0),
  CONSTRAINT "products_status_valid" CHECK ((status)::text = ANY ((ARRAY['draft'::character varying, 'active'::character varying, 'archived'::character varying])::text[]))
);
-- Create index "idx_products_category_status" to table: "products"
CREATE INDEX "idx_products_category_status" ON "products" ("category_id", "status");
-- Create index "idx_products_featured_status" to table: "products"
CREATE INDEX "idx_products_featured_status" ON "products" ("is_featured", "status");
-- Create index "products_slug_key" to table: "products"
CREATE UNIQUE INDEX "products_slug_key" ON "products" ("slug");
-- Create "product_variants" table
CREATE TABLE "product_variants" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "product_id" uuid NOT NULL,
  "sku" character varying(100) NOT NULL,
  "color_name" character varying(80) NULL,
  "color_hex" character varying(7) NULL,
  "size" character varying(40) NULL,
  "price_amount" bigint NOT NULL,
  "compare_at_price_amount" bigint NULL,
  "cost_amount" bigint NULL,
  "currency" character varying(3) NOT NULL DEFAULT 'BDT',
  "barcode" character varying(100) NULL,
  "weight_grams" integer NULL,
  "is_active" boolean NOT NULL DEFAULT true,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "product_variants_product_id_fkey" FOREIGN KEY ("product_id") REFERENCES "products" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "product_variants_color_not_blank" CHECK ((color_name IS NULL) OR (length(TRIM(BOTH FROM color_name)) > 0)),
  CONSTRAINT "product_variants_compare_at_price_valid" CHECK ((compare_at_price_amount IS NULL) OR (compare_at_price_amount >= price_amount)),
  CONSTRAINT "product_variants_cost_nonnegative" CHECK ((cost_amount IS NULL) OR (cost_amount >= 0)),
  CONSTRAINT "product_variants_currency_valid" CHECK ((length((currency)::text) = 3) AND ((currency)::text = upper((currency)::text))),
  CONSTRAINT "product_variants_price_nonnegative" CHECK (price_amount >= 0),
  CONSTRAINT "product_variants_size_not_blank" CHECK ((size IS NULL) OR (length(TRIM(BOTH FROM size)) > 0)),
  CONSTRAINT "product_variants_sku_not_blank" CHECK (length(TRIM(BOTH FROM sku)) > 0),
  CONSTRAINT "product_variants_weight_nonnegative" CHECK ((weight_grams IS NULL) OR (weight_grams >= 0))
);
-- Create index "idx_product_variants_product_active" to table: "product_variants"
CREATE INDEX "idx_product_variants_product_active" ON "product_variants" ("product_id", "is_active");
-- Create index "product_variants_barcode_key" to table: "product_variants"
CREATE UNIQUE INDEX "product_variants_barcode_key" ON "product_variants" ("barcode");
-- Create index "product_variants_sku_key" to table: "product_variants"
CREATE UNIQUE INDEX "product_variants_sku_key" ON "product_variants" ("sku");
-- Create "inventory" table
CREATE TABLE "inventory" (
  "variant_id" uuid NOT NULL,
  "quantity_on_hand" integer NOT NULL DEFAULT 0,
  "quantity_reserved" integer NOT NULL DEFAULT 0,
  "reorder_level" integer NOT NULL DEFAULT 0,
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("variant_id"),
  CONSTRAINT "inventory_variant_id_fkey" FOREIGN KEY ("variant_id") REFERENCES "product_variants" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "inventory_quantity_on_hand_nonnegative" CHECK (quantity_on_hand >= 0),
  CONSTRAINT "inventory_quantity_reserved_nonnegative" CHECK (quantity_reserved >= 0),
  CONSTRAINT "inventory_reorder_level_nonnegative" CHECK (reorder_level >= 0),
  CONSTRAINT "inventory_reserved_not_above_on_hand" CHECK (quantity_reserved <= quantity_on_hand)
);
-- Create "product_images" table
CREATE TABLE "product_images" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "product_id" uuid NOT NULL,
  "url" text NOT NULL,
  "alt_text" character varying(255) NULL,
  "sort_order" integer NOT NULL DEFAULT 0,
  "is_primary" boolean NOT NULL DEFAULT false,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "product_images_product_id_fkey" FOREIGN KEY ("product_id") REFERENCES "products" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "product_images_sort_order_nonnegative" CHECK (sort_order >= 0),
  CONSTRAINT "product_images_url_not_blank" CHECK (length(TRIM(BOTH FROM url)) > 0)
);
-- Create index "idx_product_images_product_sort" to table: "product_images"
CREATE INDEX "idx_product_images_product_sort" ON "product_images" ("product_id", "sort_order");
-- Create index "product_images_one_primary_per_product" to table: "product_images"
CREATE UNIQUE INDEX "product_images_one_primary_per_product" ON "product_images" ("product_id") WHERE (is_primary = true);
