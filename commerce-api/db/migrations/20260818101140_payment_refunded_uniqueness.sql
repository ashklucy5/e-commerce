-- Drop index "payments_one_succeeded_per_order" from table: "payments"
DROP INDEX "payments_one_succeeded_per_order";
-- Create index "payments_one_succeeded_per_order" to table: "payments"
CREATE UNIQUE INDEX "payments_one_succeeded_per_order" ON "payments" ("order_id") WHERE ((status)::text = ANY ((ARRAY['succeeded'::character varying, 'refunded'::character varying])::text[]));
