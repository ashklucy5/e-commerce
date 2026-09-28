-- Modify "customers" table
ALTER TABLE "customers" DROP CONSTRAINT "customers_status_valid", ADD CONSTRAINT "customers_status_valid" CHECK ((status)::text = ANY ((ARRAY['active'::character varying, 'disabled'::character varying, 'banned'::character varying])::text[]));
-- Create "account_bans" table
CREATE TABLE "account_bans" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "target_type" character varying(20) NOT NULL,
  "staff_account_id" uuid NULL,
  "customer_id" uuid NULL,
  "scope" character varying(40) NOT NULL,
  "ban_type" character varying(20) NOT NULL,
  "reason" character varying(1000) NOT NULL,
  "previous_account_status" character varying(20) NULL,
  "starts_at" timestamptz NOT NULL DEFAULT now(),
  "expires_at" timestamptz NULL,
  "issued_by_staff_id" uuid NOT NULL,
  "revoked_at" timestamptz NULL,
  "revoked_by_staff_id" uuid NULL,
  "revocation_reason" character varying(1000) NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "account_bans_customer_id_fkey" FOREIGN KEY ("customer_id") REFERENCES "customers" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "account_bans_issued_by_staff_id_fkey" FOREIGN KEY ("issued_by_staff_id") REFERENCES "staff_accounts" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "account_bans_revoked_by_staff_id_fkey" FOREIGN KEY ("revoked_by_staff_id") REFERENCES "staff_accounts" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "account_bans_staff_account_id_fkey" FOREIGN KEY ("staff_account_id") REFERENCES "staff_accounts" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "account_bans_duration_consistent" CHECK ((((ban_type)::text = 'permanent'::text) AND (expires_at IS NULL)) OR (((ban_type)::text = 'temporary'::text) AND (expires_at IS NOT NULL) AND (expires_at > starts_at))),
  CONSTRAINT "account_bans_previous_status_required" CHECK ((((scope)::text = 'full_account'::text) AND (previous_account_status IS NOT NULL)) OR (((scope)::text <> 'full_account'::text) AND (previous_account_status IS NULL))),
  CONSTRAINT "account_bans_previous_status_valid" CHECK ((previous_account_status IS NULL) OR (((target_type)::text = 'staff'::text) AND ((previous_account_status)::text = ANY ((ARRAY['active'::character varying, 'suspended'::character varying, 'disabled'::character varying])::text[]))) OR (((target_type)::text = 'customer'::text) AND ((previous_account_status)::text = ANY ((ARRAY['active'::character varying, 'disabled'::character varying])::text[])))),
  CONSTRAINT "account_bans_reason_not_blank" CHECK (length(TRIM(BOTH FROM reason)) > 0),
  CONSTRAINT "account_bans_revocation_consistent" CHECK (((revoked_at IS NULL) AND (revoked_by_staff_id IS NULL) AND (revocation_reason IS NULL)) OR ((revoked_at IS NOT NULL) AND (revoked_by_staff_id IS NOT NULL) AND (revocation_reason IS NOT NULL) AND (length(TRIM(BOTH FROM revocation_reason)) > 0))),
  CONSTRAINT "account_bans_revocation_time_valid" CHECK ((revoked_at IS NULL) OR (revoked_at >= created_at)),
  CONSTRAINT "account_bans_scope_valid" CHECK ((scope)::text = ANY ((ARRAY['full_account'::character varying, 'purchasing'::character varying, 'product_requests'::character varying, 'support_messages'::character varying, 'reviews'::character varying, 'returns'::character varying, 'promotions'::character varying])::text[])),
  CONSTRAINT "account_bans_staff_scope_valid" CHECK (((target_type)::text <> 'staff'::text) OR ((scope)::text = 'full_account'::text)),
  CONSTRAINT "account_bans_target_consistent" CHECK ((((target_type)::text = 'staff'::text) AND (staff_account_id IS NOT NULL) AND (customer_id IS NULL)) OR (((target_type)::text = 'customer'::text) AND (customer_id IS NOT NULL) AND (staff_account_id IS NULL))),
  CONSTRAINT "account_bans_target_type_valid" CHECK ((target_type)::text = ANY ((ARRAY['staff'::character varying, 'customer'::character varying])::text[])),
  CONSTRAINT "account_bans_type_valid" CHECK ((ban_type)::text = ANY ((ARRAY['temporary'::character varying, 'permanent'::character varying])::text[]))
);
-- Create index "idx_account_bans_customer_scope_created" to table: "account_bans"
CREATE INDEX "idx_account_bans_customer_scope_created" ON "account_bans" ("customer_id", "scope", "created_at") WHERE (customer_id IS NOT NULL);
-- Create index "idx_account_bans_issued_by_created" to table: "account_bans"
CREATE INDEX "idx_account_bans_issued_by_created" ON "account_bans" ("issued_by_staff_id", "created_at");
-- Create index "idx_account_bans_revoked_by" to table: "account_bans"
CREATE INDEX "idx_account_bans_revoked_by" ON "account_bans" ("revoked_by_staff_id", "revoked_at") WHERE (revoked_by_staff_id IS NOT NULL);
-- Create index "idx_account_bans_staff_scope_created" to table: "account_bans"
CREATE INDEX "idx_account_bans_staff_scope_created" ON "account_bans" ("staff_account_id", "scope", "created_at") WHERE (staff_account_id IS NOT NULL);
-- Create index "idx_account_bans_unrevoked_window" to table: "account_bans"
CREATE INDEX "idx_account_bans_unrevoked_window" ON "account_bans" ("target_type", "scope", "starts_at", "expires_at") WHERE (revoked_at IS NULL);
