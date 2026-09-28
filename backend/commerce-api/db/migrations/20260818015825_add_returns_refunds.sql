-- Create "returns" table
CREATE TABLE "returns" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "return_number" character varying(50) NOT NULL,
  "order_id" uuid NOT NULL,
  "status" character varying(30) NOT NULL DEFAULT 'requested',
  "customer_note" character varying(1000) NULL,
  "requested_by" character varying(100) NULL,
  "approved_by" character varying(100) NULL,
  "rejected_by" character varying(100) NULL,
  "rejection_reason" character varying(500) NULL,
  "requested_at" timestamptz NOT NULL DEFAULT now(),
  "approved_at" timestamptz NULL,
  "rejected_at" timestamptz NULL,
  "received_at" timestamptz NULL,
  "inspected_at" timestamptz NULL,
  "completed_at" timestamptz NULL,
  "cancelled_at" timestamptz NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "returns_order_id_fkey" FOREIGN KEY ("order_id") REFERENCES "orders" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "returns_return_number_not_blank" CHECK (length(TRIM(BOTH FROM return_number)) > 0),
  CONSTRAINT "returns_status_valid" CHECK ((status)::text = ANY ((ARRAY['requested'::character varying, 'approved'::character varying, 'rejected'::character varying, 'received'::character varying, 'inspected'::character varying, 'completed'::character varying, 'cancelled'::character varying])::text[]))
);
-- Create index "idx_returns_order_created" to table: "returns"
CREATE INDEX "idx_returns_order_created" ON "returns" ("order_id", "created_at");
-- Create index "idx_returns_status_created" to table: "returns"
CREATE INDEX "idx_returns_status_created" ON "returns" ("status", "created_at");
-- Create index "returns_return_number_key" to table: "returns"
CREATE UNIQUE INDEX "returns_return_number_key" ON "returns" ("return_number");
-- Create "refunds" table
CREATE TABLE "refunds" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "refund_number" character varying(50) NOT NULL,
  "order_id" uuid NOT NULL,
  "return_id" uuid NULL,
  "payment_id" uuid NULL,
  "source_type" character varying(30) NOT NULL,
  "status" character varying(30) NOT NULL DEFAULT 'requested',
  "amount" bigint NOT NULL,
  "currency" character varying(3) NOT NULL,
  "provider" character varying(50) NULL,
  "provider_refund_id" character varying(160) NULL,
  "reason" character varying(500) NOT NULL,
  "requested_by" character varying(100) NULL,
  "approved_by" character varying(100) NULL,
  "failure_code" character varying(100) NULL,
  "failure_message" character varying(500) NULL,
  "requested_at" timestamptz NOT NULL DEFAULT now(),
  "approved_at" timestamptz NULL,
  "processing_at" timestamptz NULL,
  "succeeded_at" timestamptz NULL,
  "failed_at" timestamptz NULL,
  "cancelled_at" timestamptz NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "refunds_order_id_fkey" FOREIGN KEY ("order_id") REFERENCES "orders" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "refunds_payment_id_fkey" FOREIGN KEY ("payment_id") REFERENCES "payments" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "refunds_return_id_fkey" FOREIGN KEY ("return_id") REFERENCES "returns" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "refunds_amount_positive" CHECK (amount > 0),
  CONSTRAINT "refunds_currency_valid" CHECK ((length((currency)::text) = 3) AND ((currency)::text = upper((currency)::text))),
  CONSTRAINT "refunds_provider_valid" CHECK ((provider IS NULL) OR ((provider)::text = ANY ((ARRAY['bkash'::character varying, 'nagad'::character varying, 'rocket'::character varying, 'bank_transfer'::character varying, 'manual'::character varying])::text[]))),
  CONSTRAINT "refunds_reason_not_blank" CHECK (length(TRIM(BOTH FROM reason)) > 0),
  CONSTRAINT "refunds_refund_number_not_blank" CHECK (length(TRIM(BOTH FROM refund_number)) > 0),
  CONSTRAINT "refunds_source_type_valid" CHECK ((source_type)::text = ANY ((ARRAY['cancellation'::character varying, 'return'::character varying, 'manual'::character varying])::text[])),
  CONSTRAINT "refunds_status_valid" CHECK ((status)::text = ANY ((ARRAY['requested'::character varying, 'approved'::character varying, 'processing'::character varying, 'succeeded'::character varying, 'failed'::character varying, 'cancelled'::character varying])::text[]))
);
-- Create index "idx_refunds_order_created" to table: "refunds"
CREATE INDEX "idx_refunds_order_created" ON "refunds" ("order_id", "created_at");
-- Create index "idx_refunds_payment" to table: "refunds"
CREATE INDEX "idx_refunds_payment" ON "refunds" ("payment_id");
-- Create index "idx_refunds_return" to table: "refunds"
CREATE INDEX "idx_refunds_return" ON "refunds" ("return_id");
-- Create index "idx_refunds_status_created" to table: "refunds"
CREATE INDEX "idx_refunds_status_created" ON "refunds" ("status", "created_at");
-- Create index "refunds_provider_refund_id_key" to table: "refunds"
CREATE UNIQUE INDEX "refunds_provider_refund_id_key" ON "refunds" ("provider", "provider_refund_id") WHERE (provider_refund_id IS NOT NULL);
-- Create index "refunds_refund_number_key" to table: "refunds"
CREATE UNIQUE INDEX "refunds_refund_number_key" ON "refunds" ("refund_number");
-- Create "refund_events" table
CREATE TABLE "refund_events" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "refund_id" uuid NOT NULL,
  "event_type" character varying(60) NOT NULL,
  "from_status" character varying(30) NULL,
  "to_status" character varying(30) NULL,
  "message" character varying(500) NULL,
  "actor_type" character varying(40) NULL,
  "actor_id" character varying(100) NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "refund_events_refund_id_fkey" FOREIGN KEY ("refund_id") REFERENCES "refunds" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "refund_events_event_type_not_blank" CHECK (length(TRIM(BOTH FROM event_type)) > 0)
);
-- Create index "idx_refund_events_refund_created" to table: "refund_events"
CREATE INDEX "idx_refund_events_refund_created" ON "refund_events" ("refund_id", "created_at");
-- Create "return_events" table
CREATE TABLE "return_events" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "return_id" uuid NOT NULL,
  "event_type" character varying(60) NOT NULL,
  "from_status" character varying(30) NULL,
  "to_status" character varying(30) NULL,
  "message" character varying(500) NULL,
  "actor_type" character varying(40) NULL,
  "actor_id" character varying(100) NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "return_events_return_id_fkey" FOREIGN KEY ("return_id") REFERENCES "returns" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "return_events_event_type_not_blank" CHECK (length(TRIM(BOTH FROM event_type)) > 0)
);
-- Create index "idx_return_events_return_created" to table: "return_events"
CREATE INDEX "idx_return_events_return_created" ON "return_events" ("return_id", "created_at");
-- Create "return_items" table
CREATE TABLE "return_items" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "return_id" uuid NOT NULL,
  "order_item_id" uuid NOT NULL,
  "quantity" integer NOT NULL,
  "reason_code" character varying(80) NOT NULL,
  "reason_note" character varying(500) NULL,
  "received_quantity" integer NOT NULL DEFAULT 0,
  "restock_quantity" integer NOT NULL DEFAULT 0,
  "inspection_status" character varying(30) NOT NULL DEFAULT 'pending',
  "inspection_note" character varying(500) NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "return_items_order_item_id_fkey" FOREIGN KEY ("order_item_id") REFERENCES "order_items" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "return_items_return_id_fkey" FOREIGN KEY ("return_id") REFERENCES "returns" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "return_items_inspection_status_valid" CHECK ((inspection_status)::text = ANY ((ARRAY['pending'::character varying, 'restockable'::character varying, 'damaged'::character varying, 'non_restockable'::character varying])::text[])),
  CONSTRAINT "return_items_quantity_positive" CHECK (quantity > 0),
  CONSTRAINT "return_items_reason_code_not_blank" CHECK (length(TRIM(BOTH FROM reason_code)) > 0),
  CONSTRAINT "return_items_received_quantity_valid" CHECK ((received_quantity >= 0) AND (received_quantity <= quantity)),
  CONSTRAINT "return_items_restock_quantity_valid" CHECK ((restock_quantity >= 0) AND (restock_quantity <= received_quantity))
);
-- Create index "idx_return_items_order_item" to table: "return_items"
CREATE INDEX "idx_return_items_order_item" ON "return_items" ("order_item_id");
-- Create index "return_items_return_order_item_key" to table: "return_items"
CREATE UNIQUE INDEX "return_items_return_order_item_key" ON "return_items" ("return_id", "order_item_id");
