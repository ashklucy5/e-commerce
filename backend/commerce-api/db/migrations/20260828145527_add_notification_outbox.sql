-- Create "notification_outbox" table
CREATE TABLE "notification_outbox" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "dedupe_key" character varying(200) NOT NULL,
  "category" character varying(30) NOT NULL,
  "event_type" character varying(100) NOT NULL,
  "channel" character varying(20) NOT NULL,
  "customer_id" uuid NULL,
  "order_id" uuid NULL,
  "case_id" uuid NULL,
  "recipient" character varying(255) NULL,
  "template_key" character varying(120) NOT NULL,
  "payload" jsonb NOT NULL DEFAULT '{}',
  "status" character varying(20) NOT NULL DEFAULT 'pending',
  "attempt_count" integer NOT NULL DEFAULT 0,
  "max_attempts" integer NOT NULL DEFAULT 5,
  "available_at" timestamptz NOT NULL DEFAULT now(),
  "locked_at" timestamptz NULL,
  "processed_at" timestamptz NULL,
  "provider_message_id" character varying(255) NULL,
  "last_error" character varying(1000) NULL,
  "skip_reason" character varying(120) NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "notification_outbox_case_id_fkey" FOREIGN KEY ("case_id") REFERENCES "crm_cases" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "notification_outbox_customer_id_fkey" FOREIGN KEY ("customer_id") REFERENCES "customers" ("id") ON UPDATE NO ACTION ON DELETE SET NULL,
  CONSTRAINT "notification_outbox_order_id_fkey" FOREIGN KEY ("order_id") REFERENCES "orders" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "notification_outbox_attempt_count_valid" CHECK (attempt_count >= 0),
  CONSTRAINT "notification_outbox_attempts_consistent" CHECK (attempt_count <= max_attempts),
  CONSTRAINT "notification_outbox_category_valid" CHECK ((category)::text = ANY ((ARRAY['order'::character varying, 'payment'::character varying, 'delivery'::character varying, 'support'::character varying, 'promotion'::character varying, 'recommendation'::character varying, 'security'::character varying, 'system'::character varying])::text[])),
  CONSTRAINT "notification_outbox_channel_valid" CHECK ((channel)::text = ANY ((ARRAY['sms'::character varying, 'email'::character varying, 'push'::character varying, 'messenger'::character varying])::text[])),
  CONSTRAINT "notification_outbox_dedupe_key_not_blank" CHECK (length(TRIM(BOTH FROM dedupe_key)) > 0),
  CONSTRAINT "notification_outbox_event_type_not_blank" CHECK (length(TRIM(BOTH FROM event_type)) > 0),
  CONSTRAINT "notification_outbox_last_error_not_blank" CHECK ((last_error IS NULL) OR (length(TRIM(BOTH FROM last_error)) > 0)),
  CONSTRAINT "notification_outbox_max_attempts_valid" CHECK ((max_attempts >= 1) AND (max_attempts <= 20)),
  CONSTRAINT "notification_outbox_nonterminal_timestamp_valid" CHECK (((status)::text <> ALL ((ARRAY['pending'::character varying, 'processing'::character varying])::text[])) OR (processed_at IS NULL)),
  CONSTRAINT "notification_outbox_processing_lock_valid" CHECK (((status)::text <> 'processing'::text) OR (locked_at IS NOT NULL)),
  CONSTRAINT "notification_outbox_provider_message_id_not_blank" CHECK ((provider_message_id IS NULL) OR (length(TRIM(BOTH FROM provider_message_id)) > 0)),
  CONSTRAINT "notification_outbox_recipient_not_blank" CHECK ((recipient IS NULL) OR (length(TRIM(BOTH FROM recipient)) > 0)),
  CONSTRAINT "notification_outbox_skip_reason_not_blank" CHECK ((skip_reason IS NULL) OR (length(TRIM(BOTH FROM skip_reason)) > 0)),
  CONSTRAINT "notification_outbox_status_valid" CHECK ((status)::text = ANY ((ARRAY['pending'::character varying, 'processing'::character varying, 'sent'::character varying, 'skipped'::character varying, 'dead'::character varying])::text[])),
  CONSTRAINT "notification_outbox_template_key_not_blank" CHECK (length(TRIM(BOTH FROM template_key)) > 0),
  CONSTRAINT "notification_outbox_terminal_timestamp_valid" CHECK (((status)::text <> ALL ((ARRAY['sent'::character varying, 'skipped'::character varying, 'dead'::character varying])::text[])) OR (processed_at IS NOT NULL))
);
-- Create index "idx_notification_outbox_case_created" to table: "notification_outbox"
CREATE INDEX "idx_notification_outbox_case_created" ON "notification_outbox" ("case_id", "created_at");
-- Create index "idx_notification_outbox_customer_created" to table: "notification_outbox"
CREATE INDEX "idx_notification_outbox_customer_created" ON "notification_outbox" ("customer_id", "created_at");
-- Create index "idx_notification_outbox_due" to table: "notification_outbox"
CREATE INDEX "idx_notification_outbox_due" ON "notification_outbox" ("status", "available_at", "created_at") WHERE ((status)::text = ANY ((ARRAY['pending'::character varying, 'processing'::character varying])::text[]));
-- Create index "idx_notification_outbox_order_created" to table: "notification_outbox"
CREATE INDEX "idx_notification_outbox_order_created" ON "notification_outbox" ("order_id", "created_at");
-- Create index "notification_outbox_dedupe_key_key" to table: "notification_outbox"
CREATE UNIQUE INDEX "notification_outbox_dedupe_key_key" ON "notification_outbox" ("dedupe_key");
