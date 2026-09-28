-- Create "reviews" table
CREATE TABLE "reviews" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "customer_id" uuid NOT NULL,
  "order_item_id" uuid NOT NULL,
  "rating" integer NOT NULL,
  "title" character varying(160) NULL,
  "body" text NULL,
  "status" character varying(20) NOT NULL DEFAULT 'published',
  "deleted_at" timestamptz NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "reviews_customer_id_fkey" FOREIGN KEY ("customer_id") REFERENCES "customers" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "reviews_order_item_id_fkey" FOREIGN KEY ("order_item_id") REFERENCES "order_items" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "reviews_body_valid" CHECK ((body IS NULL) OR ((length(TRIM(BOTH FROM body)) > 0) AND (char_length(body) <= 5000))),
  CONSTRAINT "reviews_rating_valid" CHECK ((rating >= 1) AND (rating <= 5)),
  CONSTRAINT "reviews_status_valid" CHECK ((status)::text = ANY ((ARRAY['published'::character varying, 'hidden'::character varying])::text[])),
  CONSTRAINT "reviews_title_valid" CHECK ((title IS NULL) OR (length(TRIM(BOTH FROM title)) > 0))
);
-- Create index "idx_reviews_customer_created" to table: "reviews"
CREATE INDEX "idx_reviews_customer_created" ON "reviews" ("customer_id", "created_at") WHERE (deleted_at IS NULL);
-- Create index "idx_reviews_status_created" to table: "reviews"
CREATE INDEX "idx_reviews_status_created" ON "reviews" ("status", "created_at") WHERE (deleted_at IS NULL);
-- Create index "reviews_order_item_active_key" to table: "reviews"
CREATE UNIQUE INDEX "reviews_order_item_active_key" ON "reviews" ("order_item_id") WHERE (deleted_at IS NULL);
