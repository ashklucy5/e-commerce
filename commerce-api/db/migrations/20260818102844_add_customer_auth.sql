-- Drop index "payments_one_succeeded_per_order" from table: "payments"
DROP INDEX "payments_one_succeeded_per_order";
-- Create index "payments_one_succeeded_per_order" to table: "payments"
CREATE UNIQUE INDEX "payments_one_succeeded_per_order" ON "payments" ("order_id") WHERE ((status)::text = ANY ((ARRAY['succeeded'::character varying, 'refunded'::character varying])::text[]));
-- Create "customers" table
CREATE TABLE "customers" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "phone" character varying(40) NOT NULL,
  "email" character varying(255) NULL,
  "password_hash" text NOT NULL,
  "full_name" character varying(160) NOT NULL,
  "status" character varying(20) NOT NULL DEFAULT 'active',
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "customers_email_not_blank" CHECK ((email IS NULL) OR (length(TRIM(BOTH FROM email)) > 0)),
  CONSTRAINT "customers_full_name_not_blank" CHECK (length(TRIM(BOTH FROM full_name)) > 0),
  CONSTRAINT "customers_password_hash_not_blank" CHECK (length(TRIM(BOTH FROM password_hash)) > 0),
  CONSTRAINT "customers_phone_not_blank" CHECK (length(TRIM(BOTH FROM phone)) > 0),
  CONSTRAINT "customers_status_valid" CHECK ((status)::text = ANY ((ARRAY['active'::character varying, 'disabled'::character varying])::text[]))
);
-- Create index "customers_email_key" to table: "customers"
CREATE UNIQUE INDEX "customers_email_key" ON "customers" ("email");
-- Create index "customers_phone_key" to table: "customers"
CREATE UNIQUE INDEX "customers_phone_key" ON "customers" ("phone");
-- Create index "idx_customers_status_created" to table: "customers"
CREATE INDEX "idx_customers_status_created" ON "customers" ("status", "created_at");
-- Create "auth_sessions" table
CREATE TABLE "auth_sessions" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "customer_id" uuid NOT NULL,
  "access_token_hash" character varying(64) NOT NULL,
  "refresh_token_hash" character varying(64) NOT NULL,
  "access_expires_at" timestamptz NOT NULL,
  "refresh_expires_at" timestamptz NOT NULL,
  "last_used_at" timestamptz NULL,
  "revoked_at" timestamptz NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "auth_sessions_customer_id_fkey" FOREIGN KEY ("customer_id") REFERENCES "customers" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "auth_sessions_access_token_hash_valid" CHECK (length((access_token_hash)::text) = 64),
  CONSTRAINT "auth_sessions_expiry_valid" CHECK (refresh_expires_at > access_expires_at),
  CONSTRAINT "auth_sessions_refresh_token_hash_valid" CHECK (length((refresh_token_hash)::text) = 64)
);
-- Create index "auth_sessions_access_token_hash_key" to table: "auth_sessions"
CREATE UNIQUE INDEX "auth_sessions_access_token_hash_key" ON "auth_sessions" ("access_token_hash");
-- Create index "auth_sessions_refresh_token_hash_key" to table: "auth_sessions"
CREATE UNIQUE INDEX "auth_sessions_refresh_token_hash_key" ON "auth_sessions" ("refresh_token_hash");
-- Create index "idx_auth_sessions_customer_created" to table: "auth_sessions"
CREATE INDEX "idx_auth_sessions_customer_created" ON "auth_sessions" ("customer_id", "created_at");
-- Create index "idx_auth_sessions_refresh_expiry" to table: "auth_sessions"
CREATE INDEX "idx_auth_sessions_refresh_expiry" ON "auth_sessions" ("refresh_expires_at") WHERE (revoked_at IS NULL);
-- Modify "checkout_sessions" table
ALTER TABLE "checkout_sessions" ADD COLUMN "customer_id" uuid NULL, ADD CONSTRAINT "checkout_sessions_customer_id_fkey" FOREIGN KEY ("customer_id") REFERENCES "customers" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;
-- Create index "idx_checkout_sessions_customer_created" to table: "checkout_sessions"
CREATE INDEX "idx_checkout_sessions_customer_created" ON "checkout_sessions" ("customer_id", "created_at");
-- Create "customer_addresses" table
CREATE TABLE "customer_addresses" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "customer_id" uuid NOT NULL,
  "label" character varying(80) NOT NULL DEFAULT 'Address',
  "recipient_name" character varying(160) NOT NULL,
  "phone" character varying(40) NOT NULL,
  "address_line1" character varying(255) NOT NULL,
  "address_line2" character varying(255) NULL,
  "city" character varying(120) NOT NULL,
  "area" character varying(120) NOT NULL,
  "postal_code" character varying(30) NULL,
  "is_default" boolean NOT NULL DEFAULT false,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "customer_addresses_customer_id_fkey" FOREIGN KEY ("customer_id") REFERENCES "customers" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "customer_addresses_address_line1_not_blank" CHECK (length(TRIM(BOTH FROM address_line1)) > 0),
  CONSTRAINT "customer_addresses_area_not_blank" CHECK (length(TRIM(BOTH FROM area)) > 0),
  CONSTRAINT "customer_addresses_city_not_blank" CHECK (length(TRIM(BOTH FROM city)) > 0),
  CONSTRAINT "customer_addresses_label_not_blank" CHECK (length(TRIM(BOTH FROM label)) > 0),
  CONSTRAINT "customer_addresses_phone_not_blank" CHECK (length(TRIM(BOTH FROM phone)) > 0),
  CONSTRAINT "customer_addresses_recipient_name_not_blank" CHECK (length(TRIM(BOTH FROM recipient_name)) > 0)
);
-- Create index "customer_addresses_one_default_per_customer" to table: "customer_addresses"
CREATE UNIQUE INDEX "customer_addresses_one_default_per_customer" ON "customer_addresses" ("customer_id") WHERE (is_default = true);
-- Create index "idx_customer_addresses_customer_created" to table: "customer_addresses"
CREATE INDEX "idx_customer_addresses_customer_created" ON "customer_addresses" ("customer_id", "created_at");
-- Modify "orders" table
ALTER TABLE "orders" ADD COLUMN "customer_id" uuid NULL, ADD CONSTRAINT "orders_customer_id_fkey" FOREIGN KEY ("customer_id") REFERENCES "customers" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;
-- Create index "idx_orders_customer_created" to table: "orders"
CREATE INDEX "idx_orders_customer_created" ON "orders" ("customer_id", "created_at");
