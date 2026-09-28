-- Create "product_code_namespaces" table
CREATE TABLE "product_code_namespaces" (
  "category_id" uuid NOT NULL,
  "prefix" character varying(7) NOT NULL,
  "next_number" bigint NOT NULL DEFAULT 1,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("category_id"),
  CONSTRAINT "product_code_namespaces_category_id_fkey" FOREIGN KEY ("category_id") REFERENCES "categories" ("id") ON UPDATE NO ACTION ON DELETE RESTRICT,
  CONSTRAINT "product_code_namespaces_next_number_limit" CHECK (next_number <= 1000000),
  CONSTRAINT "product_code_namespaces_next_number_positive" CHECK (next_number > 0),
  CONSTRAINT "product_code_namespaces_prefix_format" CHECK ((prefix)::text ~ '^[A-Z0-9]{3}-[A-Z0-9]{3}$'::text)
);
-- Create index "product_code_namespaces_prefix_key" to table: "product_code_namespaces"
CREATE UNIQUE INDEX "product_code_namespaces_prefix_key" ON "product_code_namespaces" ("prefix");
