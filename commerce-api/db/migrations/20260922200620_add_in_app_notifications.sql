-- Create "customer_notifications" table
CREATE TABLE "customer_notifications" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "dedupe_key" character varying(200) NOT NULL,
  "customer_id" uuid NOT NULL,
  "category" character varying(30) NOT NULL,
  "event_type" character varying(100) NOT NULL,
  "title" character varying(160) NOT NULL,
  "message" character varying(1000) NOT NULL,
  "action_url" character varying(500) NULL,
  "order_id" uuid NULL,
  "case_id" uuid NULL,
  "metadata" jsonb NOT NULL DEFAULT '{}',
  "read_at" timestamptz NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "customer_notifications_case_id_fkey" FOREIGN KEY ("case_id") REFERENCES "crm_cases" ("id") ON UPDATE NO ACTION ON DELETE SET NULL,
  CONSTRAINT "customer_notifications_customer_id_fkey" FOREIGN KEY ("customer_id") REFERENCES "customers" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "customer_notifications_order_id_fkey" FOREIGN KEY ("order_id") REFERENCES "orders" ("id") ON UPDATE NO ACTION ON DELETE SET NULL,
  CONSTRAINT "customer_notifications_action_url_valid" CHECK ((action_url IS NULL) OR ((length(TRIM(BOTH FROM action_url)) > 0) AND ((action_url)::text ~~ '/%'::text) AND ((action_url)::text !~~ '//%'::text))),
  CONSTRAINT "customer_notifications_category_valid" CHECK ((category)::text = ANY ((ARRAY['order'::character varying, 'payment'::character varying, 'delivery'::character varying, 'sourcing'::character varying, 'support'::character varying, 'promotion'::character varying, 'recommendation'::character varying, 'security'::character varying, 'system'::character varying])::text[])),
  CONSTRAINT "customer_notifications_dedupe_key_not_blank" CHECK (length(TRIM(BOTH FROM dedupe_key)) > 0),
  CONSTRAINT "customer_notifications_event_type_not_blank" CHECK (length(TRIM(BOTH FROM event_type)) > 0),
  CONSTRAINT "customer_notifications_message_not_blank" CHECK (length(TRIM(BOTH FROM message)) > 0),
  CONSTRAINT "customer_notifications_title_not_blank" CHECK (length(TRIM(BOTH FROM title)) > 0)
);
-- Create index "customer_notifications_dedupe_key_key" to table: "customer_notifications"
CREATE UNIQUE INDEX "customer_notifications_dedupe_key_key" ON "customer_notifications" ("dedupe_key");
-- Create index "idx_customer_notifications_customer_created" to table: "customer_notifications"
CREATE INDEX "idx_customer_notifications_customer_created" ON "customer_notifications" ("customer_id", "created_at");
-- Create index "idx_customer_notifications_customer_unread" to table: "customer_notifications"
CREATE INDEX "idx_customer_notifications_customer_unread" ON "customer_notifications" ("customer_id", "created_at") WHERE (read_at IS NULL);
-- Create "staff_notifications" table
CREATE TABLE "staff_notifications" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "dedupe_key" character varying(200) NOT NULL,
  "category" character varying(30) NOT NULL,
  "event_type" character varying(100) NOT NULL,
  "priority" character varying(20) NOT NULL DEFAULT 'info',
  "title" character varying(160) NOT NULL,
  "message" character varying(1000) NOT NULL,
  "action_url" character varying(500) NULL,
  "entity_type" character varying(50) NULL,
  "entity_id" character varying(160) NULL,
  "metadata" jsonb NOT NULL DEFAULT '{}',
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "staff_notifications_action_url_valid" CHECK ((action_url IS NULL) OR ((length(TRIM(BOTH FROM action_url)) > 0) AND ((action_url)::text ~~ '/%'::text) AND ((action_url)::text !~~ '//%'::text))),
  CONSTRAINT "staff_notifications_category_valid" CHECK ((category)::text = ANY ((ARRAY['order'::character varying, 'inventory'::character varying, 'sourcing'::character varying, 'support'::character varying, 'return'::character varying, 'delivery'::character varying, 'security'::character varying, 'system'::character varying])::text[])),
  CONSTRAINT "staff_notifications_dedupe_key_not_blank" CHECK (length(TRIM(BOTH FROM dedupe_key)) > 0),
  CONSTRAINT "staff_notifications_entity_pair" CHECK (((entity_type IS NULL) AND (entity_id IS NULL)) OR ((entity_type IS NOT NULL) AND (entity_id IS NOT NULL))),
  CONSTRAINT "staff_notifications_event_type_not_blank" CHECK (length(TRIM(BOTH FROM event_type)) > 0),
  CONSTRAINT "staff_notifications_message_not_blank" CHECK (length(TRIM(BOTH FROM message)) > 0),
  CONSTRAINT "staff_notifications_priority_valid" CHECK ((priority)::text = ANY ((ARRAY['info'::character varying, 'attention'::character varying, 'critical'::character varying])::text[])),
  CONSTRAINT "staff_notifications_title_not_blank" CHECK (length(TRIM(BOTH FROM title)) > 0)
);
-- Create index "idx_staff_notifications_category_created" to table: "staff_notifications"
CREATE INDEX "idx_staff_notifications_category_created" ON "staff_notifications" ("category", "created_at");
-- Create index "idx_staff_notifications_entity_created" to table: "staff_notifications"
CREATE INDEX "idx_staff_notifications_entity_created" ON "staff_notifications" ("entity_type", "entity_id", "created_at") WHERE ((entity_type IS NOT NULL) AND (entity_id IS NOT NULL));
-- Create index "staff_notifications_dedupe_key_key" to table: "staff_notifications"
CREATE UNIQUE INDEX "staff_notifications_dedupe_key_key" ON "staff_notifications" ("dedupe_key");
-- Create "staff_notification_recipients" table
CREATE TABLE "staff_notification_recipients" (
  "notification_id" uuid NOT NULL,
  "staff_account_id" uuid NOT NULL,
  "read_at" timestamptz NULL,
  "dismissed_at" timestamptz NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("notification_id", "staff_account_id"),
  CONSTRAINT "staff_notification_recipients_notification_fkey" FOREIGN KEY ("notification_id") REFERENCES "staff_notifications" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "staff_notification_recipients_staff_fkey" FOREIGN KEY ("staff_account_id") REFERENCES "staff_accounts" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- Create index "idx_staff_notification_recipients_staff_created" to table: "staff_notification_recipients"
CREATE INDEX "idx_staff_notification_recipients_staff_created" ON "staff_notification_recipients" ("staff_account_id", "created_at");
-- Create index "idx_staff_notification_recipients_staff_unread" to table: "staff_notification_recipients"
CREATE INDEX "idx_staff_notification_recipients_staff_unread" ON "staff_notification_recipients" ("staff_account_id", "created_at") WHERE ((read_at IS NULL) AND (dismissed_at IS NULL));
