-- Modify "orders" table
ALTER TABLE "orders" ADD COLUMN "processing_at" timestamptz NULL, ADD COLUMN "shipped_at" timestamptz NULL, ADD COLUMN "delivered_at" timestamptz NULL, ADD COLUMN "completed_at" timestamptz NULL;
-- Create "shipments" table
CREATE TABLE "shipments" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "order_id" uuid NOT NULL,
  "courier_name" character varying(120) NOT NULL,
  "courier_reference" character varying(160) NULL,
  "tracking_number" character varying(160) NULL,
  "tracking_url" character varying(1000) NULL,
  "status" character varying(30) NOT NULL DEFAULT 'pending',
  "shipped_at" timestamptz NULL,
  "delivered_at" timestamptz NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "shipments_order_id_fkey" FOREIGN KEY ("order_id") REFERENCES "orders" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "shipments_courier_name_not_blank" CHECK (length(TRIM(BOTH FROM courier_name)) > 0),
  CONSTRAINT "shipments_status_valid" CHECK ((status)::text = ANY ((ARRAY['pending'::character varying, 'shipped'::character varying, 'delivered'::character varying, 'cancelled'::character varying])::text[]))
);
-- Create index "idx_shipments_status_created" to table: "shipments"
CREATE INDEX "idx_shipments_status_created" ON "shipments" ("status", "created_at");
-- Create index "idx_shipments_tracking_number" to table: "shipments"
CREATE INDEX "idx_shipments_tracking_number" ON "shipments" ("tracking_number");
-- Create index "shipments_order_id_key" to table: "shipments"
CREATE UNIQUE INDEX "shipments_order_id_key" ON "shipments" ("order_id");
