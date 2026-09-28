-- Create "crm_cases" table
CREATE TABLE "crm_cases" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "case_number" character varying(60) NOT NULL,
  "customer_id" uuid NOT NULL,
  "case_type" character varying(40) NOT NULL,
  "subject" character varying(180) NOT NULL,
  "status" character varying(30) NOT NULL DEFAULT 'waiting_support',
  "priority" character varying(20) NOT NULL DEFAULT 'normal',
  "product_id" uuid NULL,
  "variant_id" uuid NULL,
  "order_id" uuid NULL,
  "requested_quantity" integer NULL,
  "available_quantity_snapshot" integer NULL,
  "context_snapshot" jsonb NOT NULL DEFAULT '{}',
  "last_message_at" timestamptz NOT NULL DEFAULT now(),
  "last_customer_message_at" timestamptz NULL,
  "last_support_message_at" timestamptz NULL,
  "resolved_at" timestamptz NULL,
  "closed_at" timestamptz NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "crm_cases_customer_id_fkey" FOREIGN KEY ("customer_id") REFERENCES "customers" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "crm_cases_order_id_fkey" FOREIGN KEY ("order_id") REFERENCES "orders" ("id") ON UPDATE NO ACTION ON DELETE SET NULL,
  CONSTRAINT "crm_cases_product_id_fkey" FOREIGN KEY ("product_id") REFERENCES "products" ("id") ON UPDATE NO ACTION ON DELETE SET NULL,
  CONSTRAINT "crm_cases_variant_id_fkey" FOREIGN KEY ("variant_id") REFERENCES "product_variants" ("id") ON UPDATE NO ACTION ON DELETE SET NULL,
  CONSTRAINT "crm_cases_available_quantity_nonnegative" CHECK ((available_quantity_snapshot IS NULL) OR (available_quantity_snapshot >= 0)),
  CONSTRAINT "crm_cases_bulk_context_valid" CHECK (((case_type)::text <> 'bulk_stock_request'::text) OR ((product_id IS NOT NULL) AND (variant_id IS NOT NULL) AND (requested_quantity IS NOT NULL) AND (available_quantity_snapshot IS NOT NULL))),
  CONSTRAINT "crm_cases_case_number_not_blank" CHECK (length(TRIM(BOTH FROM case_number)) > 0),
  CONSTRAINT "crm_cases_closed_timestamp_valid" CHECK (((status)::text <> 'closed'::text) OR (closed_at IS NOT NULL)),
  CONSTRAINT "crm_cases_priority_valid" CHECK ((priority)::text = ANY ((ARRAY['low'::character varying, 'normal'::character varying, 'high'::character varying, 'urgent'::character varying])::text[])),
  CONSTRAINT "crm_cases_requested_quantity_positive" CHECK ((requested_quantity IS NULL) OR (requested_quantity > 0)),
  CONSTRAINT "crm_cases_resolved_timestamp_valid" CHECK (((status)::text <> ALL ((ARRAY['resolved'::character varying, 'closed'::character varying])::text[])) OR (resolved_at IS NOT NULL)),
  CONSTRAINT "crm_cases_status_valid" CHECK ((status)::text = ANY ((ARRAY['waiting_support'::character varying, 'waiting_customer'::character varying, 'resolved'::character varying, 'closed'::character varying])::text[])),
  CONSTRAINT "crm_cases_subject_not_blank" CHECK (length(TRIM(BOTH FROM subject)) > 0),
  CONSTRAINT "crm_cases_type_valid" CHECK ((case_type)::text = ANY ((ARRAY['bulk_stock_request'::character varying, 'product_question'::character varying, 'order_issue'::character varying, 'payment_issue'::character varying, 'return_issue'::character varying, 'delivery_issue'::character varying, 'complaint'::character varying, 'general_question'::character varying, 'other'::character varying])::text[]))
);
-- Create index "crm_cases_case_number_key" to table: "crm_cases"
CREATE UNIQUE INDEX "crm_cases_case_number_key" ON "crm_cases" ("case_number");
-- Create index "idx_crm_cases_customer_status" to table: "crm_cases"
CREATE INDEX "idx_crm_cases_customer_status" ON "crm_cases" ("customer_id", "status", "updated_at");
-- Create index "idx_crm_cases_customer_updated" to table: "crm_cases"
CREATE INDEX "idx_crm_cases_customer_updated" ON "crm_cases" ("customer_id", "updated_at");
-- Create index "idx_crm_cases_order" to table: "crm_cases"
CREATE INDEX "idx_crm_cases_order" ON "crm_cases" ("order_id");
-- Create index "idx_crm_cases_product" to table: "crm_cases"
CREATE INDEX "idx_crm_cases_product" ON "crm_cases" ("product_id");
-- Create index "idx_crm_cases_support_inbox" to table: "crm_cases"
CREATE INDEX "idx_crm_cases_support_inbox" ON "crm_cases" ("status", "priority", "last_message_at");
-- Create index "idx_crm_cases_variant" to table: "crm_cases"
CREATE INDEX "idx_crm_cases_variant" ON "crm_cases" ("variant_id");
-- Create "crm_case_events" table
CREATE TABLE "crm_case_events" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "case_id" uuid NOT NULL,
  "event_type" character varying(80) NOT NULL,
  "actor_type" character varying(20) NOT NULL,
  "support_actor_id" uuid NULL,
  "from_status" character varying(30) NULL,
  "to_status" character varying(30) NULL,
  "payload" jsonb NOT NULL DEFAULT '{}',
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "crm_case_events_case_id_fkey" FOREIGN KEY ("case_id") REFERENCES "crm_cases" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "crm_case_events_support_actor_id_fkey" FOREIGN KEY ("support_actor_id") REFERENCES "support_actors" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "crm_case_events_actor_consistent" CHECK ((((actor_type)::text = 'customer'::text) AND (support_actor_id IS NULL)) OR (((actor_type)::text = 'support'::text) AND (support_actor_id IS NOT NULL))),
  CONSTRAINT "crm_case_events_actor_type_valid" CHECK ((actor_type)::text = ANY ((ARRAY['customer'::character varying, 'support'::character varying])::text[])),
  CONSTRAINT "crm_case_events_event_type_not_blank" CHECK (length(TRIM(BOTH FROM event_type)) > 0),
  CONSTRAINT "crm_case_events_from_status_valid" CHECK ((from_status IS NULL) OR ((from_status)::text = ANY ((ARRAY['waiting_support'::character varying, 'waiting_customer'::character varying, 'resolved'::character varying, 'closed'::character varying])::text[]))),
  CONSTRAINT "crm_case_events_to_status_valid" CHECK ((to_status IS NULL) OR ((to_status)::text = ANY ((ARRAY['waiting_support'::character varying, 'waiting_customer'::character varying, 'resolved'::character varying, 'closed'::character varying])::text[])))
);
-- Create index "idx_crm_case_events_case_created" to table: "crm_case_events"
CREATE INDEX "idx_crm_case_events_case_created" ON "crm_case_events" ("case_id", "created_at");
-- Create "crm_messages" table
CREATE TABLE "crm_messages" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "case_id" uuid NOT NULL,
  "author_type" character varying(20) NOT NULL,
  "support_actor_id" uuid NULL,
  "visibility" character varying(20) NOT NULL DEFAULT 'customer',
  "body" text NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "crm_messages_case_id_fkey" FOREIGN KEY ("case_id") REFERENCES "crm_cases" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "crm_messages_support_actor_id_fkey" FOREIGN KEY ("support_actor_id") REFERENCES "support_actors" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "crm_messages_author_consistent" CHECK ((((author_type)::text = 'customer'::text) AND (support_actor_id IS NULL) AND ((visibility)::text = 'customer'::text)) OR (((author_type)::text = 'support'::text) AND (support_actor_id IS NOT NULL))),
  CONSTRAINT "crm_messages_author_type_valid" CHECK ((author_type)::text = ANY ((ARRAY['customer'::character varying, 'support'::character varying])::text[])),
  CONSTRAINT "crm_messages_body_not_blank" CHECK (length(TRIM(BOTH FROM body)) > 0),
  CONSTRAINT "crm_messages_visibility_valid" CHECK ((visibility)::text = ANY ((ARRAY['customer'::character varying, 'internal'::character varying])::text[]))
);
-- Create index "idx_crm_messages_case_created" to table: "crm_messages"
CREATE INDEX "idx_crm_messages_case_created" ON "crm_messages" ("case_id", "created_at");
-- Create index "idx_crm_messages_support_actor" to table: "crm_messages"
CREATE INDEX "idx_crm_messages_support_actor" ON "crm_messages" ("support_actor_id");
-- Create "support_case_assignments" table
CREATE TABLE "support_case_assignments" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "case_id" uuid NOT NULL,
  "queue_id" uuid NOT NULL,
  "support_actor_id" uuid NULL,
  "assigned_by_actor_id" uuid NULL,
  "assignment_type" character varying(20) NOT NULL,
  "assigned_at" timestamptz NOT NULL DEFAULT now(),
  "accepted_at" timestamptz NULL,
  "released_at" timestamptz NULL,
  "release_reason" character varying(300) NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "support_case_assignments_assigned_by_actor_id_fkey" FOREIGN KEY ("assigned_by_actor_id") REFERENCES "support_actors" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "support_case_assignments_case_id_fkey" FOREIGN KEY ("case_id") REFERENCES "crm_cases" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "support_case_assignments_queue_id_fkey" FOREIGN KEY ("queue_id") REFERENCES "support_queues" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "support_case_assignments_support_actor_id_fkey" FOREIGN KEY ("support_actor_id") REFERENCES "support_actors" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "support_case_assignments_release_reason_valid" CHECK ((release_reason IS NULL) OR (length(TRIM(BOTH FROM release_reason)) > 0)),
  CONSTRAINT "support_case_assignments_times_valid" CHECK (((accepted_at IS NULL) OR (accepted_at >= assigned_at)) AND ((released_at IS NULL) OR (released_at >= assigned_at))),
  CONSTRAINT "support_case_assignments_type_valid" CHECK ((assignment_type)::text = ANY ((ARRAY['automatic'::character varying, 'manual'::character varying, 'claimed'::character varying, 'escalated'::character varying])::text[]))
);
-- Create index "idx_support_case_assignments_actor_active" to table: "support_case_assignments"
CREATE INDEX "idx_support_case_assignments_actor_active" ON "support_case_assignments" ("support_actor_id", "assigned_at") WHERE ((support_actor_id IS NOT NULL) AND (released_at IS NULL));
-- Create index "idx_support_case_assignments_case_history" to table: "support_case_assignments"
CREATE INDEX "idx_support_case_assignments_case_history" ON "support_case_assignments" ("case_id", "assigned_at");
-- Create index "idx_support_case_assignments_queue_active" to table: "support_case_assignments"
CREATE INDEX "idx_support_case_assignments_queue_active" ON "support_case_assignments" ("queue_id", "assigned_at") WHERE (released_at IS NULL);
-- Create index "support_case_assignments_active_case_key" to table: "support_case_assignments"
CREATE UNIQUE INDEX "support_case_assignments_active_case_key" ON "support_case_assignments" ("case_id") WHERE (released_at IS NULL);
