-- Create "payments" table
CREATE TABLE "payments" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "order_id" uuid NOT NULL,
  "provider" character varying(30) NOT NULL,
  "status" character varying(30) NOT NULL DEFAULT 'pending',
  "amount" bigint NOT NULL,
  "currency" character varying(3) NOT NULL,
  "provider_payment_id" character varying(160) NOT NULL,
  "provider_transaction_id" character varying(160) NULL,
  "paid_at" timestamptz NULL,
  "failed_at" timestamptz NULL,
  "failure_code" character varying(100) NULL,
  "failure_message" character varying(500) NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "payments_order_id_fkey" FOREIGN KEY ("order_id") REFERENCES "orders" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "payments_amount_nonnegative" CHECK (amount >= 0),
  CONSTRAINT "payments_currency_valid" CHECK ((length((currency)::text) = 3) AND ((currency)::text = upper((currency)::text))),
  CONSTRAINT "payments_provider_payment_id_not_blank" CHECK (length(TRIM(BOTH FROM provider_payment_id)) > 0),
  CONSTRAINT "payments_provider_valid" CHECK ((provider)::text = ANY ((ARRAY['bkash'::character varying, 'nagad'::character varying])::text[])),
  CONSTRAINT "payments_status_valid" CHECK ((status)::text = ANY ((ARRAY['pending'::character varying, 'succeeded'::character varying, 'failed'::character varying, 'expired'::character varying, 'refunded'::character varying])::text[]))
);
-- Create index "idx_payments_order_created" to table: "payments"
CREATE INDEX "idx_payments_order_created" ON "payments" ("order_id", "created_at");
-- Create index "idx_payments_status_created" to table: "payments"
CREATE INDEX "idx_payments_status_created" ON "payments" ("status", "created_at");
-- Create index "payments_one_succeeded_per_order" to table: "payments"
CREATE UNIQUE INDEX "payments_one_succeeded_per_order" ON "payments" ("order_id") WHERE ((status)::text = 'succeeded'::text);
-- Create index "payments_provider_payment_id_key" to table: "payments"
CREATE UNIQUE INDEX "payments_provider_payment_id_key" ON "payments" ("provider", "provider_payment_id");
-- Create index "payments_provider_transaction_id_key" to table: "payments"
CREATE UNIQUE INDEX "payments_provider_transaction_id_key" ON "payments" ("provider", "provider_transaction_id") WHERE (provider_transaction_id IS NOT NULL);
-- Create "payment_events" table
CREATE TABLE "payment_events" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "order_id" uuid NOT NULL,
  "payment_id" uuid NULL,
  "provider" character varying(30) NOT NULL,
  "provider_event_id" character varying(200) NOT NULL,
  "event_type" character varying(100) NOT NULL,
  "provider_payment_id" character varying(160) NOT NULL,
  "provider_transaction_id" character varying(160) NULL,
  "payload_sha256" character varying(64) NOT NULL,
  "status" character varying(30) NOT NULL DEFAULT 'received',
  "error_message" character varying(500) NULL,
  "received_at" timestamptz NOT NULL DEFAULT now(),
  "processed_at" timestamptz NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "payment_events_order_id_fkey" FOREIGN KEY ("order_id") REFERENCES "orders" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "payment_events_payment_id_fkey" FOREIGN KEY ("payment_id") REFERENCES "payments" ("id") ON UPDATE NO ACTION ON DELETE SET NULL,
  CONSTRAINT "payment_events_event_type_not_blank" CHECK (length(TRIM(BOTH FROM event_type)) > 0),
  CONSTRAINT "payment_events_payload_sha256_valid" CHECK (length((payload_sha256)::text) = 64),
  CONSTRAINT "payment_events_provider_event_id_not_blank" CHECK (length(TRIM(BOTH FROM provider_event_id)) > 0),
  CONSTRAINT "payment_events_provider_payment_id_not_blank" CHECK (length(TRIM(BOTH FROM provider_payment_id)) > 0),
  CONSTRAINT "payment_events_provider_valid" CHECK ((provider)::text = ANY ((ARRAY['bkash'::character varying, 'nagad'::character varying])::text[])),
  CONSTRAINT "payment_events_status_valid" CHECK ((status)::text = ANY ((ARRAY['received'::character varying, 'processed'::character varying, 'ignored'::character varying, 'failed'::character varying])::text[]))
);
-- Create index "idx_payment_events_order_received" to table: "payment_events"
CREATE INDEX "idx_payment_events_order_received" ON "payment_events" ("order_id", "received_at");
-- Create index "idx_payment_events_status_received" to table: "payment_events"
CREATE INDEX "idx_payment_events_status_received" ON "payment_events" ("status", "received_at");
-- Create index "payment_events_provider_event_id_key" to table: "payment_events"
CREATE UNIQUE INDEX "payment_events_provider_event_id_key" ON "payment_events" ("provider", "provider_event_id");
