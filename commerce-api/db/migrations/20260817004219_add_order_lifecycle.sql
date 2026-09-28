-- Modify "orders" table
ALTER TABLE "orders" ADD COLUMN "cancellation_reason" character varying(500) NULL, ADD COLUMN "cancelled_by" character varying(100) NULL;
-- Create "order_events" table
CREATE TABLE "order_events" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "order_id" uuid NOT NULL,
  "event_type" character varying(60) NOT NULL,
  "from_status" character varying(30) NULL,
  "to_status" character varying(30) NULL,
  "message" character varying(500) NULL,
  "actor_type" character varying(40) NULL,
  "actor_id" character varying(100) NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "order_events_order_id_fkey" FOREIGN KEY ("order_id") REFERENCES "orders" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "order_events_event_type_not_blank" CHECK (length(TRIM(BOTH FROM event_type)) > 0)
);
-- Create index "idx_order_events_order_created" to table: "order_events"
CREATE INDEX "idx_order_events_order_created" ON "order_events" ("order_id", "created_at");
