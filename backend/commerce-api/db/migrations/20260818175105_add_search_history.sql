-- Create "customer_search_history" table
CREATE TABLE "customer_search_history" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "customer_id" uuid NOT NULL,
  "query" character varying(200) NOT NULL,
  "normalized_query" character varying(200) NOT NULL,
  "result_count" bigint NOT NULL DEFAULT 0,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "expires_at" timestamptz NOT NULL DEFAULT (now() + '72:00:00'::interval),
  PRIMARY KEY ("id"),
  CONSTRAINT "customer_search_history_customer_id_fkey" FOREIGN KEY ("customer_id") REFERENCES "customers" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "customer_search_history_expiry_valid" CHECK ((expires_at > created_at) AND (expires_at <= (created_at + '72:00:00'::interval))),
  CONSTRAINT "customer_search_history_normalized_query_not_blank" CHECK (length(TRIM(BOTH FROM normalized_query)) > 0),
  CONSTRAINT "customer_search_history_query_not_blank" CHECK (length(TRIM(BOTH FROM query)) > 0),
  CONSTRAINT "customer_search_history_result_count_nonnegative" CHECK (result_count >= 0)
);
-- Create index "idx_customer_search_history_customer_created" to table: "customer_search_history"
CREATE INDEX "idx_customer_search_history_customer_created" ON "customer_search_history" ("customer_id", "created_at");
-- Create index "idx_customer_search_history_customer_expires" to table: "customer_search_history"
CREATE INDEX "idx_customer_search_history_customer_expires" ON "customer_search_history" ("customer_id", "expires_at");
-- Create index "idx_customer_search_history_expires" to table: "customer_search_history"
CREATE INDEX "idx_customer_search_history_expires" ON "customer_search_history" ("expires_at");
-- Create "customer_search_history_categories" table
CREATE TABLE "customer_search_history_categories" (
  "search_history_id" uuid NOT NULL,
  "category_id" uuid NOT NULL,
  "relevance_weight" smallint NOT NULL DEFAULT 100,
  PRIMARY KEY ("search_history_id", "category_id"),
  CONSTRAINT "customer_search_history_categories_category_id_fkey" FOREIGN KEY ("category_id") REFERENCES "categories" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "customer_search_history_categories_history_id_fkey" FOREIGN KEY ("search_history_id") REFERENCES "customer_search_history" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "customer_search_history_categories_weight_valid" CHECK ((relevance_weight >= 1) AND (relevance_weight <= 100))
);
-- Create index "idx_customer_search_history_categories_category" to table: "customer_search_history_categories"
CREATE INDEX "idx_customer_search_history_categories_category" ON "customer_search_history_categories" ("category_id");
-- Create "recommendation_category_relations" table
CREATE TABLE "recommendation_category_relations" (
  "source_category_id" uuid NOT NULL,
  "target_category_id" uuid NOT NULL,
  "relation_type" character varying(20) NOT NULL,
  "weight" smallint NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("source_category_id", "target_category_id"),
  CONSTRAINT "recommendation_category_relations_source_fkey" FOREIGN KEY ("source_category_id") REFERENCES "categories" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "recommendation_category_relations_target_fkey" FOREIGN KEY ("target_category_id") REFERENCES "categories" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "recommendation_category_relations_not_self" CHECK (source_category_id <> target_category_id),
  CONSTRAINT "recommendation_category_relations_type_valid" CHECK ((relation_type)::text = ANY ((ARRAY['related'::character varying, 'complementary'::character varying])::text[])),
  CONSTRAINT "recommendation_category_relations_weight_valid" CHECK ((weight >= 1) AND (weight <= 100))
);
-- Create index "idx_recommendation_category_relations_target" to table: "recommendation_category_relations"
CREATE INDEX "idx_recommendation_category_relations_target" ON "recommendation_category_relations" ("target_category_id");
