-- Create "finance_expenses" table
CREATE TABLE "finance_expenses" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "category" character varying(40) NOT NULL,
  "amount" bigint NOT NULL,
  "currency" character varying(3) NOT NULL DEFAULT 'BDT',
  "occurred_at" timestamptz NOT NULL DEFAULT now(),
  "description" character varying(500) NOT NULL,
  "order_id" uuid NULL,
  "reference" character varying(160) NULL,
  "status" character varying(20) NOT NULL DEFAULT 'active',
  "idempotency_key_hash" character varying(64) NOT NULL,
  "request_fingerprint" character varying(64) NOT NULL,
  "created_by_staff_id" uuid NOT NULL,
  "voided_by_staff_id" uuid NULL,
  "void_reason" character varying(500) NULL,
  "voided_at" timestamptz NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "finance_expenses_created_by_staff_id_fkey" FOREIGN KEY ("created_by_staff_id") REFERENCES "staff_accounts" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "finance_expenses_order_id_fkey" FOREIGN KEY ("order_id") REFERENCES "orders" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "finance_expenses_voided_by_staff_id_fkey" FOREIGN KEY ("voided_by_staff_id") REFERENCES "staff_accounts" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "finance_expenses_amount_positive" CHECK (amount > 0),
  CONSTRAINT "finance_expenses_category_valid" CHECK ((category)::text = ANY ((ARRAY['delivery'::character varying, 'payment_fee'::character varying, 'china_freight'::character varying, 'customs'::character varying, 'packaging'::character varying, 'marketing'::character varying, 'warehouse'::character varying, 'other'::character varying])::text[])),
  CONSTRAINT "finance_expenses_currency_valid" CHECK ((length((currency)::text) = 3) AND ((currency)::text = upper((currency)::text))),
  CONSTRAINT "finance_expenses_description_not_blank" CHECK (length(TRIM(BOTH FROM description)) > 0),
  CONSTRAINT "finance_expenses_idempotency_hash_valid" CHECK (length((idempotency_key_hash)::text) = 64),
  CONSTRAINT "finance_expenses_reference_not_blank" CHECK ((reference IS NULL) OR (length(TRIM(BOTH FROM reference)) > 0)),
  CONSTRAINT "finance_expenses_request_fingerprint_valid" CHECK (length((request_fingerprint)::text) = 64),
  CONSTRAINT "finance_expenses_status_valid" CHECK ((status)::text = ANY ((ARRAY['active'::character varying, 'voided'::character varying])::text[])),
  CONSTRAINT "finance_expenses_updated_at_valid" CHECK (updated_at >= created_at),
  CONSTRAINT "finance_expenses_void_fields_consistent" CHECK ((((status)::text = 'active'::text) AND (voided_by_staff_id IS NULL) AND (void_reason IS NULL) AND (voided_at IS NULL)) OR (((status)::text = 'voided'::text) AND (voided_by_staff_id IS NOT NULL) AND (void_reason IS NOT NULL) AND (length(TRIM(BOTH FROM void_reason)) > 0) AND (voided_at IS NOT NULL)))
);
-- Create index "finance_expenses_idempotency_key_hash_key" to table: "finance_expenses"
CREATE UNIQUE INDEX "finance_expenses_idempotency_key_hash_key" ON "finance_expenses" ("idempotency_key_hash");
-- Create index "idx_finance_expenses_category_occurred" to table: "finance_expenses"
CREATE INDEX "idx_finance_expenses_category_occurred" ON "finance_expenses" ("category", "occurred_at");
-- Create index "idx_finance_expenses_created_by" to table: "finance_expenses"
CREATE INDEX "idx_finance_expenses_created_by" ON "finance_expenses" ("created_by_staff_id", "created_at");
-- Create index "idx_finance_expenses_currency_occurred" to table: "finance_expenses"
CREATE INDEX "idx_finance_expenses_currency_occurred" ON "finance_expenses" ("currency", "occurred_at");
-- Create index "idx_finance_expenses_order" to table: "finance_expenses"
CREATE INDEX "idx_finance_expenses_order" ON "finance_expenses" ("order_id", "occurred_at");
-- Create index "idx_finance_expenses_status_occurred" to table: "finance_expenses"
CREATE INDEX "idx_finance_expenses_status_occurred" ON "finance_expenses" ("status", "occurred_at");
