-- Create "warehouses" table
CREATE TABLE "warehouses" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "code" character varying(40) NOT NULL,
  "name" character varying(160) NOT NULL,
  "country_code" character varying(2) NOT NULL DEFAULT 'BD',
  "city" character varying(120) NULL,
  "address_line1" character varying(255) NULL,
  "status" character varying(20) NOT NULL DEFAULT 'active',
  "is_default" boolean NOT NULL DEFAULT false,
  "allows_self_pickup" boolean NOT NULL DEFAULT false,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "warehouses_code_not_blank" CHECK (length(TRIM(BOTH FROM code)) > 0),
  CONSTRAINT "warehouses_country_code_valid" CHECK ((length((country_code)::text) = 2) AND ((country_code)::text = upper((country_code)::text))),
  CONSTRAINT "warehouses_name_not_blank" CHECK (length(TRIM(BOTH FROM name)) > 0),
  CONSTRAINT "warehouses_status_valid" CHECK ((status)::text = ANY ((ARRAY['active'::character varying, 'inactive'::character varying])::text[]))
);
-- Create index "idx_warehouses_status_created" to table: "warehouses"
CREATE INDEX "idx_warehouses_status_created" ON "warehouses" ("status", "created_at");
-- Create index "warehouses_code_key" to table: "warehouses"
CREATE UNIQUE INDEX "warehouses_code_key" ON "warehouses" ("code");
-- Create index "warehouses_one_default_key" to table: "warehouses"
CREATE UNIQUE INDEX "warehouses_one_default_key" ON "warehouses" ("is_default") WHERE (is_default = true);
-- Seed the initial Bangladesh fulfillment warehouse.
-- This is operational data and therefore belongs in the migration,
-- not in the declarative schema.pg.hcl file.
INSERT INTO "warehouses" (
  "code",
  "name",
  "country_code",
  "status",
  "is_default",
  "allows_self_pickup"
)
VALUES (
  'BD-MAIN',
  'Bangladesh Main Warehouse',
  'BD',
  'active',
  true,
  false
);
-- Modify "shipments" table
ALTER TABLE "shipments" DROP CONSTRAINT "shipments_courier_name_not_blank", ADD CONSTRAINT "shipments_courier_name_not_blank" CHECK ((courier_name IS NULL) OR (length(TRIM(BOTH FROM courier_name)) > 0)), DROP CONSTRAINT "shipments_status_valid", ADD CONSTRAINT "shipments_status_valid" CHECK ((status)::text = ANY ((ARRAY['pending'::character varying, 'shipped'::character varying, 'awaiting_confirmation'::character varying, 'delivered'::character varying, 'cancelled'::character varying])::text[])), ADD CONSTRAINT "shipments_awaiting_confirmation_requires_timestamp" CHECK (((status)::text <> 'awaiting_confirmation'::text) OR (awaiting_confirmation_at IS NOT NULL)), ADD CONSTRAINT "shipments_confirmation_fields_consistent" CHECK (((confirmed_received_at IS NULL) AND (confirmation_source IS NULL) AND (confirmed_by_actor_id IS NULL)) OR ((confirmed_received_at IS NOT NULL) AND (confirmation_source IS NOT NULL) AND (confirmed_by_actor_id IS NOT NULL))), ADD CONSTRAINT "shipments_confirmation_source_valid" CHECK ((confirmation_source IS NULL) OR ((confirmation_source)::text = ANY ((ARRAY['customer'::character varying, 'support'::character varying, 'admin'::character varying])::text[]))), ADD CONSTRAINT "shipments_delivery_mode_valid" CHECK ((delivery_mode)::text = ANY ((ARRAY['courier'::character varying, 'self_pickup'::character varying, 'community_rider'::character varying])::text[])), ADD CONSTRAINT "shipments_provider_code_not_blank" CHECK ((provider_code IS NULL) OR (length(TRIM(BOTH FROM provider_code)) > 0)), ADD CONSTRAINT "shipments_provider_shipment_id_not_blank" CHECK ((provider_shipment_id IS NULL) OR (length(TRIM(BOTH FROM provider_shipment_id)) > 0)), ADD CONSTRAINT "shipments_rider_reference_not_blank" CHECK ((rider_reference IS NULL) OR (length(TRIM(BOTH FROM rider_reference)) > 0)), ALTER COLUMN "courier_name" DROP NOT NULL, ADD COLUMN "origin_warehouse_id" uuid NULL, ADD COLUMN "delivery_mode" character varying(30) NOT NULL DEFAULT 'courier', ADD COLUMN "provider_code" character varying(60) NULL, ADD COLUMN "provider_shipment_id" character varying(160) NULL, ADD COLUMN "provider_status" character varying(80) NULL, ADD COLUMN "rider_reference" character varying(160) NULL, ADD COLUMN "provider_delivered_at" timestamptz NULL, ADD COLUMN "awaiting_confirmation_at" timestamptz NULL, ADD COLUMN "confirmed_received_at" timestamptz NULL, ADD COLUMN "confirmation_source" character varying(30) NULL, ADD COLUMN "confirmed_by_actor_id" character varying(160) NULL, ADD COLUMN "confirmation_note" character varying(1000) NULL, ADD COLUMN "last_provider_sync_at" timestamptz NULL, ADD CONSTRAINT "shipments_origin_warehouse_fkey" FOREIGN KEY ("origin_warehouse_id") REFERENCES "warehouses" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;
-- Backfill historical shipment confirmation information.
--
-- Shipments that were already marked delivered before Logistics v0
-- predate the explicit customer/support/admin receipt-confirmation
-- pipeline.
--
-- Preserve those historical deliveries as administratively confirmed.
-- Backfill historical shipment receipt-confirmation information.
--
-- Shipments already marked delivered before Logistics v0 predate the
-- explicit provider-delivered -> awaiting-confirmation -> customer/support
-- confirmation pipeline.
--
-- Treat those historical final deliveries as administratively confirmed,
-- but DO NOT manufacture a provider_delivered_at timestamp because no
-- provider/rider delivery event was recorded by the old system.
UPDATE "shipments"
SET
  "confirmed_received_at" = COALESCE(
    "confirmed_received_at",
    "delivered_at",
    "updated_at",
    "created_at"
  ),
  "confirmation_source" = COALESCE(
    "confirmation_source",
    'admin'
  ),
  "confirmed_by_actor_id" = COALESCE(
    "confirmed_by_actor_id",
    'historical-logistics-migration'
  ),
  "confirmation_note" = COALESCE(
    "confirmation_note",
    'Backfilled from a pre-Logistics-v0 delivered shipment.'
  )
WHERE "status" = 'delivered';
-- Create index "idx_shipments_confirmation" to table: "shipments"
CREATE INDEX "idx_shipments_confirmation" ON "shipments" ("status", "awaiting_confirmation_at");
-- Create index "idx_shipments_order_created" to table: "shipments"
CREATE INDEX "idx_shipments_order_created" ON "shipments" ("order_id", "created_at");
-- Create index "idx_shipments_origin_warehouse_created" to table: "shipments"
CREATE INDEX "idx_shipments_origin_warehouse_created" ON "shipments" ("origin_warehouse_id", "created_at");
-- Create index "shipments_provider_shipment_key" to table: "shipments"
CREATE UNIQUE INDEX "shipments_provider_shipment_key" ON "shipments" ("provider_code", "provider_shipment_id") WHERE ((provider_code IS NOT NULL) AND (provider_shipment_id IS NOT NULL));
-- Create "delivery_tracking_events" table
CREATE TABLE "delivery_tracking_events" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "shipment_id" uuid NOT NULL,
  "order_id" uuid NOT NULL,
  "source" character varying(30) NOT NULL,
  "event_code" character varying(80) NOT NULL,
  "status" character varying(80) NULL,
  "message" character varying(1000) NULL,
  "latitude" double precision NULL,
  "longitude" double precision NULL,
  "external_event_id" character varying(160) NULL,
  "metadata" jsonb NULL,
  "occurred_at" timestamptz NOT NULL DEFAULT now(),
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "delivery_tracking_events_order_fkey" FOREIGN KEY ("order_id") REFERENCES "orders" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "delivery_tracking_events_shipment_fkey" FOREIGN KEY ("shipment_id") REFERENCES "shipments" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "delivery_tracking_events_event_code_not_blank" CHECK (length(TRIM(BOTH FROM event_code)) > 0),
  CONSTRAINT "delivery_tracking_events_latitude_valid" CHECK ((latitude IS NULL) OR ((latitude >= ('-90'::integer)::double precision) AND (latitude <= (90)::double precision))),
  CONSTRAINT "delivery_tracking_events_longitude_valid" CHECK ((longitude IS NULL) OR ((longitude >= ('-180'::integer)::double precision) AND (longitude <= (180)::double precision))),
  CONSTRAINT "delivery_tracking_events_source_valid" CHECK ((source)::text = ANY ((ARRAY['manual'::character varying, 'provider'::character varying, 'rider'::character varying, 'customer'::character varying, 'support'::character varying, 'admin'::character varying, 'system'::character varying])::text[]))
);
-- Create index "delivery_tracking_events_external_key" to table: "delivery_tracking_events"
CREATE UNIQUE INDEX "delivery_tracking_events_external_key" ON "delivery_tracking_events" ("shipment_id", "external_event_id") WHERE (external_event_id IS NOT NULL);
-- Create index "idx_delivery_tracking_events_order_time" to table: "delivery_tracking_events"
CREATE INDEX "idx_delivery_tracking_events_order_time" ON "delivery_tracking_events" ("order_id", "occurred_at", "created_at");
-- Create index "idx_delivery_tracking_events_shipment_time" to table: "delivery_tracking_events"
CREATE INDEX "idx_delivery_tracking_events_shipment_time" ON "delivery_tracking_events" ("shipment_id", "occurred_at", "created_at");
-- Create "inbound_shipments" table
CREATE TABLE "inbound_shipments" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "reference_code" character varying(80) NOT NULL,
  "origin_country" character varying(2) NOT NULL DEFAULT 'CN',
  "destination_warehouse_id" uuid NOT NULL,
  "carrier_name" character varying(160) NULL,
  "external_reference" character varying(160) NULL,
  "tracking_number" character varying(160) NULL,
  "status" character varying(40) NOT NULL DEFAULT 'created',
  "eta" timestamptz NULL,
  "departed_at" timestamptz NULL,
  "arrived_bangladesh_at" timestamptz NULL,
  "customs_released_at" timestamptz NULL,
  "received_at" timestamptz NULL,
  "notes" text NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "inbound_shipments_destination_warehouse_fkey" FOREIGN KEY ("destination_warehouse_id") REFERENCES "warehouses" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "inbound_shipments_origin_country_valid" CHECK ((length((origin_country)::text) = 2) AND ((origin_country)::text = upper((origin_country)::text))),
  CONSTRAINT "inbound_shipments_reference_not_blank" CHECK (length(TRIM(BOTH FROM reference_code)) > 0),
  CONSTRAINT "inbound_shipments_status_valid" CHECK ((status)::text = ANY ((ARRAY['created'::character varying, 'supplier_ready'::character varying, 'picked_up_in_china'::character varying, 'departed_china'::character varying, 'in_international_transit'::character varying, 'arrived_bangladesh'::character varying, 'customs_processing'::character varying, 'customs_released'::character varying, 'received_at_warehouse'::character varying, 'cancelled'::character varying])::text[]))
);
-- Create index "idx_inbound_shipments_destination_created" to table: "inbound_shipments"
CREATE INDEX "idx_inbound_shipments_destination_created" ON "inbound_shipments" ("destination_warehouse_id", "created_at");
-- Create index "idx_inbound_shipments_status_eta" to table: "inbound_shipments"
CREATE INDEX "idx_inbound_shipments_status_eta" ON "inbound_shipments" ("status", "eta");
-- Create index "inbound_shipments_reference_code_key" to table: "inbound_shipments"
CREATE UNIQUE INDEX "inbound_shipments_reference_code_key" ON "inbound_shipments" ("reference_code");
-- Create "inbound_shipment_events" table
CREATE TABLE "inbound_shipment_events" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "inbound_shipment_id" uuid NOT NULL,
  "status" character varying(40) NOT NULL,
  "message" character varying(1000) NULL,
  "actor_type" character varying(40) NULL,
  "actor_id" character varying(160) NULL,
  "occurred_at" timestamptz NOT NULL DEFAULT now(),
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "inbound_shipment_events_shipment_fkey" FOREIGN KEY ("inbound_shipment_id") REFERENCES "inbound_shipments" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "inbound_shipment_events_actor_pair" CHECK (((actor_type IS NULL) AND (actor_id IS NULL)) OR ((actor_type IS NOT NULL) AND (actor_id IS NOT NULL))),
  CONSTRAINT "inbound_shipment_events_status_valid" CHECK ((status)::text = ANY ((ARRAY['created'::character varying, 'supplier_ready'::character varying, 'picked_up_in_china'::character varying, 'departed_china'::character varying, 'in_international_transit'::character varying, 'arrived_bangladesh'::character varying, 'customs_processing'::character varying, 'customs_released'::character varying, 'received_at_warehouse'::character varying, 'cancelled'::character varying])::text[]))
);
-- Create index "idx_inbound_shipment_events_shipment_time" to table: "inbound_shipment_events"
CREATE INDEX "idx_inbound_shipment_events_shipment_time" ON "inbound_shipment_events" ("inbound_shipment_id", "occurred_at", "created_at");
-- Create "warehouse_fulfillments" table
CREATE TABLE "warehouse_fulfillments" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "order_id" uuid NOT NULL,
  "order_item_id" uuid NOT NULL,
  "warehouse_id" uuid NOT NULL,
  "inbound_shipment_id" uuid NULL,
  "source" character varying(30) NOT NULL,
  "status" character varying(40) NOT NULL,
  "quantity" integer NOT NULL,
  "allocated_at" timestamptz NOT NULL DEFAULT now(),
  "received_at" timestamptz NULL,
  "picking_at" timestamptz NULL,
  "packed_at" timestamptz NULL,
  "ready_for_handoff_at" timestamptz NULL,
  "handed_off_at" timestamptz NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "warehouse_fulfillments_inbound_shipment_fkey" FOREIGN KEY ("inbound_shipment_id") REFERENCES "inbound_shipments" ("id") ON UPDATE NO ACTION ON DELETE SET NULL,
  CONSTRAINT "warehouse_fulfillments_order_fkey" FOREIGN KEY ("order_id") REFERENCES "orders" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "warehouse_fulfillments_order_item_fkey" FOREIGN KEY ("order_item_id") REFERENCES "order_items" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "warehouse_fulfillments_warehouse_fkey" FOREIGN KEY ("warehouse_id") REFERENCES "warehouses" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "warehouse_fulfillments_local_has_no_inbound" CHECK (((source)::text <> 'bangladesh_stock'::text) OR (inbound_shipment_id IS NULL)),
  CONSTRAINT "warehouse_fulfillments_quantity_positive" CHECK (quantity > 0),
  CONSTRAINT "warehouse_fulfillments_source_valid" CHECK ((source)::text = ANY ((ARRAY['bangladesh_stock'::character varying, 'china_inbound'::character varying])::text[])),
  CONSTRAINT "warehouse_fulfillments_status_valid" CHECK ((status)::text = ANY ((ARRAY['allocated'::character varying, 'waiting_inbound'::character varying, 'received'::character varying, 'picking'::character varying, 'packed'::character varying, 'ready_for_handoff'::character varying, 'handed_off'::character varying, 'cancelled'::character varying])::text[])),
  CONSTRAINT "warehouse_fulfillments_waiting_inbound_source" CHECK (((status)::text <> 'waiting_inbound'::text) OR ((source)::text = 'china_inbound'::text))
);
-- Create index "idx_warehouse_fulfillments_inbound_status" to table: "warehouse_fulfillments"
CREATE INDEX "idx_warehouse_fulfillments_inbound_status" ON "warehouse_fulfillments" ("inbound_shipment_id", "status");
-- Create index "idx_warehouse_fulfillments_order_item" to table: "warehouse_fulfillments"
CREATE INDEX "idx_warehouse_fulfillments_order_item" ON "warehouse_fulfillments" ("order_item_id");
-- Create index "idx_warehouse_fulfillments_order_status" to table: "warehouse_fulfillments"
CREATE INDEX "idx_warehouse_fulfillments_order_status" ON "warehouse_fulfillments" ("order_id", "status");
-- Create index "idx_warehouse_fulfillments_warehouse_status" to table: "warehouse_fulfillments"
CREATE INDEX "idx_warehouse_fulfillments_warehouse_status" ON "warehouse_fulfillments" ("warehouse_id", "status");
-- Create "shipment_fulfillments" table
CREATE TABLE "shipment_fulfillments" (
  "shipment_id" uuid NOT NULL,
  "fulfillment_id" uuid NOT NULL,
  "quantity" integer NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("shipment_id", "fulfillment_id"),
  CONSTRAINT "shipment_fulfillments_fulfillment_fkey" FOREIGN KEY ("fulfillment_id") REFERENCES "warehouse_fulfillments" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "shipment_fulfillments_shipment_fkey" FOREIGN KEY ("shipment_id") REFERENCES "shipments" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "shipment_fulfillments_quantity_positive" CHECK (quantity > 0)
);
-- Create index "idx_shipment_fulfillments_fulfillment" to table: "shipment_fulfillments"
CREATE INDEX "idx_shipment_fulfillments_fulfillment" ON "shipment_fulfillments" ("fulfillment_id");
-- Create "warehouse_fulfillment_events" table
CREATE TABLE "warehouse_fulfillment_events" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "fulfillment_id" uuid NOT NULL,
  "event_type" character varying(60) NOT NULL,
  "from_status" character varying(40) NULL,
  "to_status" character varying(40) NULL,
  "message" character varying(1000) NULL,
  "actor_type" character varying(40) NULL,
  "actor_id" character varying(160) NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "warehouse_fulfillment_events_fulfillment_fkey" FOREIGN KEY ("fulfillment_id") REFERENCES "warehouse_fulfillments" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "warehouse_fulfillment_events_actor_pair" CHECK (((actor_type IS NULL) AND (actor_id IS NULL)) OR ((actor_type IS NOT NULL) AND (actor_id IS NOT NULL))),
  CONSTRAINT "warehouse_fulfillment_events_type_not_blank" CHECK (length(TRIM(BOTH FROM event_type)) > 0)
);
-- Create index "idx_warehouse_fulfillment_events_fulfillment_time" to table: "warehouse_fulfillment_events"
CREATE INDEX "idx_warehouse_fulfillment_events_fulfillment_time" ON "warehouse_fulfillment_events" ("fulfillment_id", "created_at");
-- Create "warehouse_handoffs" table
CREATE TABLE "warehouse_handoffs" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "shipment_id" uuid NOT NULL,
  "warehouse_id" uuid NOT NULL,
  "handoff_type" character varying(30) NOT NULL,
  "reference" character varying(160) NULL,
  "handed_off_by_actor_type" character varying(40) NULL,
  "handed_off_by_actor_id" character varying(160) NULL,
  "handed_off_at" timestamptz NOT NULL DEFAULT now(),
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "warehouse_handoffs_shipment_fkey" FOREIGN KEY ("shipment_id") REFERENCES "shipments" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "warehouse_handoffs_warehouse_fkey" FOREIGN KEY ("warehouse_id") REFERENCES "warehouses" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "warehouse_handoffs_actor_pair" CHECK (((handed_off_by_actor_type IS NULL) AND (handed_off_by_actor_id IS NULL)) OR ((handed_off_by_actor_type IS NOT NULL) AND (handed_off_by_actor_id IS NOT NULL))),
  CONSTRAINT "warehouse_handoffs_type_valid" CHECK ((handoff_type)::text = ANY ((ARRAY['courier'::character varying, 'self_pickup'::character varying, 'community_rider'::character varying])::text[]))
);
-- Create index "idx_warehouse_handoffs_shipment_time" to table: "warehouse_handoffs"
CREATE INDEX "idx_warehouse_handoffs_shipment_time" ON "warehouse_handoffs" ("shipment_id", "handed_off_at");
-- Create index "idx_warehouse_handoffs_warehouse_time" to table: "warehouse_handoffs"
CREATE INDEX "idx_warehouse_handoffs_warehouse_time" ON "warehouse_handoffs" ("warehouse_id", "handed_off_at");
