-- Create "inventory_reservations" table
CREATE TABLE "inventory_reservations" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "variant_id" uuid NOT NULL,
  "reference_type" character varying(50) NOT NULL,
  "reference_id" character varying(160) NOT NULL,
  "quantity" integer NOT NULL,
  "status" character varying(20) NOT NULL DEFAULT 'active',
  "expires_at" timestamptz NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "inventory_reservations_variant_id_fkey" FOREIGN KEY ("variant_id") REFERENCES "product_variants" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "inventory_reservations_quantity_positive" CHECK (quantity > 0),
  CONSTRAINT "inventory_reservations_reference_id_not_blank" CHECK (length(TRIM(BOTH FROM reference_id)) > 0),
  CONSTRAINT "inventory_reservations_reference_type_not_blank" CHECK (length(TRIM(BOTH FROM reference_type)) > 0),
  CONSTRAINT "inventory_reservations_status_valid" CHECK ((status)::text = ANY ((ARRAY['active'::character varying, 'released'::character varying, 'committed'::character varying, 'expired'::character varying])::text[]))
);
-- Create index "idx_inventory_reservations_active_expiry" to table: "inventory_reservations"
CREATE INDEX "idx_inventory_reservations_active_expiry" ON "inventory_reservations" ("status", "expires_at") WHERE ((status)::text = 'active'::text);
-- Create index "idx_inventory_reservations_reference" to table: "inventory_reservations"
CREATE INDEX "idx_inventory_reservations_reference" ON "inventory_reservations" ("reference_type", "reference_id");
-- Create index "idx_inventory_reservations_variant_status" to table: "inventory_reservations"
CREATE INDEX "idx_inventory_reservations_variant_status" ON "inventory_reservations" ("variant_id", "status");
-- Create index "inventory_reservations_reference_variant_key" to table: "inventory_reservations"
CREATE UNIQUE INDEX "inventory_reservations_reference_variant_key" ON "inventory_reservations" ("reference_type", "reference_id", "variant_id");
-- Create "inventory_movements" table
CREATE TABLE "inventory_movements" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "variant_id" uuid NOT NULL,
  "reservation_id" uuid NULL,
  "movement_type" character varying(30) NOT NULL,
  "quantity_on_hand_delta" integer NOT NULL DEFAULT 0,
  "quantity_reserved_delta" integer NOT NULL DEFAULT 0,
  "quantity_on_hand_after" integer NOT NULL,
  "quantity_reserved_after" integer NOT NULL,
  "reference_type" character varying(50) NULL,
  "reference_id" character varying(160) NULL,
  "reason" character varying(120) NULL,
  "note" text NULL,
  "actor_type" character varying(30) NULL,
  "actor_id" character varying(160) NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "inventory_movements_reservation_id_fkey" FOREIGN KEY ("reservation_id") REFERENCES "inventory_reservations" ("id") ON UPDATE NO ACTION ON DELETE SET NULL,
  CONSTRAINT "inventory_movements_variant_id_fkey" FOREIGN KEY ("variant_id") REFERENCES "product_variants" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "inventory_movements_actor_pair" CHECK (((actor_type IS NULL) AND (actor_id IS NULL)) OR ((actor_type IS NOT NULL) AND (actor_id IS NOT NULL))),
  CONSTRAINT "inventory_movements_has_delta" CHECK ((quantity_on_hand_delta <> 0) OR (quantity_reserved_delta <> 0)),
  CONSTRAINT "inventory_movements_on_hand_after_nonnegative" CHECK (quantity_on_hand_after >= 0),
  CONSTRAINT "inventory_movements_reference_pair" CHECK (((reference_type IS NULL) AND (reference_id IS NULL)) OR ((reference_type IS NOT NULL) AND (reference_id IS NOT NULL))),
  CONSTRAINT "inventory_movements_reserved_after_nonnegative" CHECK (quantity_reserved_after >= 0),
  CONSTRAINT "inventory_movements_reserved_after_not_above_on_hand" CHECK (quantity_reserved_after <= quantity_on_hand_after),
  CONSTRAINT "inventory_movements_type_valid" CHECK ((movement_type)::text = ANY ((ARRAY['adjustment'::character varying, 'import_sync'::character varying, 'reserve'::character varying, 'reservation_change'::character varying, 'release'::character varying, 'commit'::character varying, 'expire'::character varying])::text[]))
);
-- Create index "idx_inventory_movements_reference" to table: "inventory_movements"
CREATE INDEX "idx_inventory_movements_reference" ON "inventory_movements" ("reference_type", "reference_id");
-- Create index "idx_inventory_movements_reservation" to table: "inventory_movements"
CREATE INDEX "idx_inventory_movements_reservation" ON "inventory_movements" ("reservation_id");
-- Create index "idx_inventory_movements_variant_created" to table: "inventory_movements"
CREATE INDEX "idx_inventory_movements_variant_created" ON "inventory_movements" ("variant_id", "created_at");
