-- Create "customer_wishlist_items" table
CREATE TABLE "customer_wishlist_items" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "customer_id" uuid NOT NULL,
  "product_id" uuid NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "customer_wishlist_items_customer_id_fkey" FOREIGN KEY ("customer_id") REFERENCES "customers" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "customer_wishlist_items_product_id_fkey" FOREIGN KEY ("product_id") REFERENCES "products" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- Create index "customer_wishlist_items_customer_product_key" to table: "customer_wishlist_items"
CREATE UNIQUE INDEX "customer_wishlist_items_customer_product_key" ON "customer_wishlist_items" ("customer_id", "product_id");
-- Create index "idx_customer_wishlist_items_customer_created" to table: "customer_wishlist_items"
CREATE INDEX "idx_customer_wishlist_items_customer_created" ON "customer_wishlist_items" ("customer_id", "created_at", "id");
-- Create index "idx_customer_wishlist_items_product" to table: "customer_wishlist_items"
CREATE INDEX "idx_customer_wishlist_items_product" ON "customer_wishlist_items" ("product_id");
