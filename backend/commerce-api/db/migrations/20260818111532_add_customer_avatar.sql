-- Modify "customers" table
ALTER TABLE "customers" ADD CONSTRAINT "customers_avatar_consistent" CHECK (((avatar_storage_key IS NULL) AND (avatar_updated_at IS NULL)) OR ((avatar_storage_key IS NOT NULL) AND (avatar_updated_at IS NOT NULL))), ADD CONSTRAINT "customers_avatar_storage_key_not_blank" CHECK ((avatar_storage_key IS NULL) OR (length(TRIM(BOTH FROM avatar_storage_key)) > 0)), ADD COLUMN "avatar_storage_key" text NULL, ADD COLUMN "avatar_updated_at" timestamptz NULL;
-- Drop index "payments_one_succeeded_per_order" from table: "payments"
DROP INDEX "payments_one_succeeded_per_order";
-- Create index "payments_one_succeeded_per_order" to table: "payments"
CREATE UNIQUE INDEX "payments_one_succeeded_per_order" ON "payments" ("order_id") WHERE ((status)::text = ANY ((ARRAY['succeeded'::character varying, 'refunded'::character varying])::text[]));
