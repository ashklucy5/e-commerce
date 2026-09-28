-- Create "categories" table
CREATE TABLE "categories" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "parent_id" uuid NULL,
  "name" character varying(120) NOT NULL,
  "slug" character varying(140) NOT NULL,
  "description" text NULL,
  "image_url" text NULL,
  "icon_url" text NULL,
  "sort_order" integer NOT NULL DEFAULT 0,
  "is_active" boolean NOT NULL DEFAULT true,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "categories_parent_id_fkey" FOREIGN KEY ("parent_id") REFERENCES "categories" ("id") ON UPDATE NO ACTION ON DELETE SET NULL,
  CONSTRAINT "categories_name_not_blank" CHECK (length(TRIM(BOTH FROM name)) > 0),
  CONSTRAINT "categories_not_own_parent" CHECK ((parent_id IS NULL) OR (parent_id <> id)),
  CONSTRAINT "categories_slug_not_blank" CHECK (length(TRIM(BOTH FROM slug)) > 0)
);
-- Create index "categories_slug_key" to table: "categories"
CREATE UNIQUE INDEX "categories_slug_key" ON "categories" ("slug");
-- Create index "idx_categories_active_sort" to table: "categories"
CREATE INDEX "idx_categories_active_sort" ON "categories" ("is_active", "sort_order");
-- Create index "idx_categories_parent_id" to table: "categories"
CREATE INDEX "idx_categories_parent_id" ON "categories" ("parent_id");
