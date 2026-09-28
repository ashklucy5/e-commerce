-- Create "recommendation_events" table
CREATE TABLE "recommendation_events" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "customer_id" uuid NULL,
  "session_id" character varying(128) NULL,
  "product_id" uuid NOT NULL,
  "placement" character varying(40) NOT NULL,
  "event_type" character varying(30) NOT NULL,
  "strategy" character varying(40) NOT NULL,
  "rank_position" integer NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "recommendation_events_customer_id_fkey" FOREIGN KEY ("customer_id") REFERENCES "customers" ("id") ON UPDATE NO ACTION ON DELETE SET NULL,
  CONSTRAINT "recommendation_events_product_id_fkey" FOREIGN KEY ("product_id") REFERENCES "products" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "recommendation_events_event_type_valid" CHECK ((event_type)::text = ANY ((ARRAY['impression'::character varying, 'click'::character varying, 'add_to_cart'::character varying])::text[])),
  CONSTRAINT "recommendation_events_identity_present" CHECK ((customer_id IS NOT NULL) OR ((session_id IS NOT NULL) AND (length(TRIM(BOTH FROM session_id)) > 0))),
  CONSTRAINT "recommendation_events_placement_valid" CHECK ((placement)::text = ANY ((ARRAY['catalog'::character varying, 'product_page'::character varying, 'home'::character varying, 'search'::character varying, 'cart'::character varying])::text[])),
  CONSTRAINT "recommendation_events_rank_position_valid" CHECK ((rank_position IS NULL) OR (rank_position >= 0)),
  CONSTRAINT "recommendation_events_session_id_valid" CHECK ((session_id IS NULL) OR (length(TRIM(BOTH FROM session_id)) > 0)),
  CONSTRAINT "recommendation_events_strategy_valid" CHECK ((strategy)::text = ANY ((ARRAY['personalized'::character varying, 'category_related'::character varying, 'related'::character varying, 'complementary'::character varying, 'fallback'::character varying, 'merchandising'::character varying])::text[]))
);
-- Create index "idx_recommendation_events_created" to table: "recommendation_events"
CREATE INDEX "idx_recommendation_events_created" ON "recommendation_events" ("created_at");
-- Create index "idx_recommendation_events_customer_created" to table: "recommendation_events"
CREATE INDEX "idx_recommendation_events_customer_created" ON "recommendation_events" ("customer_id", "created_at") WHERE (customer_id IS NOT NULL);
-- Create index "idx_recommendation_events_placement_created" to table: "recommendation_events"
CREATE INDEX "idx_recommendation_events_placement_created" ON "recommendation_events" ("placement", "created_at");
-- Create index "idx_recommendation_events_product_created" to table: "recommendation_events"
CREATE INDEX "idx_recommendation_events_product_created" ON "recommendation_events" ("product_id", "created_at");
-- Create index "idx_recommendation_events_session_created" to table: "recommendation_events"
CREATE INDEX "idx_recommendation_events_session_created" ON "recommendation_events" ("session_id", "created_at") WHERE (session_id IS NOT NULL);
-- Create index "idx_recommendation_events_strategy_created" to table: "recommendation_events"
CREATE INDEX "idx_recommendation_events_strategy_created" ON "recommendation_events" ("strategy", "created_at");
-- Create index "idx_recommendation_events_type_created" to table: "recommendation_events"
CREATE INDEX "idx_recommendation_events_type_created" ON "recommendation_events" ("event_type", "created_at");
