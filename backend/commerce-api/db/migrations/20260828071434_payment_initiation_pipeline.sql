-- Drop index "payments_one_succeeded_per_order" from table: "payments"
DROP INDEX "payments_one_succeeded_per_order";
-- Modify "payments" table
ALTER TABLE "payments" DROP CONSTRAINT "payments_provider_payment_id_not_blank", ADD CONSTRAINT "payments_provider_payment_id_not_blank" CHECK ((provider_payment_id IS NULL) OR (length(TRIM(BOTH FROM provider_payment_id)) > 0)), ADD CONSTRAINT "payments_attempt_key_not_blank" CHECK ((attempt_key IS NULL) OR (length(TRIM(BOTH FROM attempt_key)) > 0)), ADD CONSTRAINT "payments_handoff_url_not_blank" CHECK ((handoff_url IS NULL) OR (length(TRIM(BOTH FROM handoff_url)) > 0)), ADD CONSTRAINT "payments_provider_expiry_valid" CHECK ((provider_expires_at IS NULL) OR (provider_expires_at > created_at)), ADD CONSTRAINT "payments_provider_transaction_id_not_blank" CHECK ((provider_transaction_id IS NULL) OR (length(TRIM(BOTH FROM provider_transaction_id)) > 0)), ALTER COLUMN "provider_payment_id" DROP NOT NULL, ADD COLUMN "attempt_key" character varying(100) NULL, ADD COLUMN "handoff_url" text NULL, ADD COLUMN "provider_expires_at" timestamptz NULL;
-- Create index "payments_one_succeeded_per_order" to table: "payments"
CREATE UNIQUE INDEX "payments_one_succeeded_per_order" ON "payments" ("order_id") WHERE ((status)::text = 'succeeded'::text);
-- Create index "payments_attempt_key_key" to table: "payments"
CREATE UNIQUE INDEX "payments_attempt_key_key" ON "payments" ("attempt_key") WHERE (attempt_key IS NOT NULL);
-- Create index "payments_one_pending_per_order" to table: "payments"
CREATE UNIQUE INDEX "payments_one_pending_per_order" ON "payments" ("order_id") WHERE ((status)::text = 'pending'::text);
