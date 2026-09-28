-- Modify "staff_accounts" table
ALTER TABLE "staff_accounts" DROP CONSTRAINT "staff_accounts_status_valid", ADD CONSTRAINT "staff_accounts_status_valid" CHECK ((status)::text = ANY ((ARRAY['pending_activation'::character varying, 'active'::character varying, 'suspended'::character varying, 'disabled'::character varying, 'deleted'::character varying, 'banned'::character varying])::text[]));
-- Create "staff_invitations" table
CREATE TABLE "staff_invitations" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "staff_account_id" uuid NOT NULL,
  "email" character varying(255) NOT NULL,
  "token_hash" character varying(64) NOT NULL,
  "delivery_mode" character varying(20) NOT NULL DEFAULT 'manual',
  "delivery_status" character varying(20) NOT NULL DEFAULT 'not_requested',
  "status" character varying(20) NOT NULL DEFAULT 'pending',
  "expires_at" timestamptz NOT NULL,
  "password_set_at" timestamptz NULL,
  "accepted_at" timestamptz NULL,
  "cancelled_at" timestamptz NULL,
  "delivered_at" timestamptz NULL,
  "delivery_error" text NULL,
  "created_by_staff_id" uuid NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "staff_invitations_created_by_staff_id_fkey" FOREIGN KEY ("created_by_staff_id") REFERENCES "staff_accounts" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "staff_invitations_staff_account_id_fkey" FOREIGN KEY ("staff_account_id") REFERENCES "staff_accounts" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "staff_invitations_accepted_state_valid" CHECK (((status)::text <> 'accepted'::text) OR (accepted_at IS NOT NULL)),
  CONSTRAINT "staff_invitations_cancelled_state_valid" CHECK (((status)::text <> 'cancelled'::text) OR (cancelled_at IS NOT NULL)),
  CONSTRAINT "staff_invitations_delivery_error_valid" CHECK ((delivery_error IS NULL) OR (length(TRIM(BOTH FROM delivery_error)) > 0)),
  CONSTRAINT "staff_invitations_delivery_mode_valid" CHECK ((delivery_mode)::text = ANY ((ARRAY['manual'::character varying, 'email'::character varying])::text[])),
  CONSTRAINT "staff_invitations_delivery_status_valid" CHECK ((delivery_status)::text = ANY ((ARRAY['not_requested'::character varying, 'pending'::character varying, 'sent'::character varying, 'failed'::character varying])::text[])),
  CONSTRAINT "staff_invitations_email_not_blank" CHECK (length(TRIM(BOTH FROM email)) > 0),
  CONSTRAINT "staff_invitations_expiry_valid" CHECK (expires_at > created_at),
  CONSTRAINT "staff_invitations_manual_delivery_state_valid" CHECK (((delivery_mode)::text <> 'manual'::text) OR ((delivery_status)::text = 'not_requested'::text)),
  CONSTRAINT "staff_invitations_password_time_valid" CHECK ((password_set_at IS NULL) OR (password_set_at >= created_at)),
  CONSTRAINT "staff_invitations_status_valid" CHECK ((status)::text = ANY ((ARRAY['pending'::character varying, 'accepted'::character varying, 'cancelled'::character varying, 'expired'::character varying])::text[])),
  CONSTRAINT "staff_invitations_terminal_state_exclusive" CHECK ((accepted_at IS NULL) OR (cancelled_at IS NULL)),
  CONSTRAINT "staff_invitations_token_hash_valid" CHECK (length((token_hash)::text) = 64)
);
-- Create index "idx_staff_invitations_creator_created" to table: "staff_invitations"
CREATE INDEX "idx_staff_invitations_creator_created" ON "staff_invitations" ("created_by_staff_id", "created_at");
-- Create index "idx_staff_invitations_email_status" to table: "staff_invitations"
CREATE INDEX "idx_staff_invitations_email_status" ON "staff_invitations" ("email", "status");
-- Create index "idx_staff_invitations_expiry" to table: "staff_invitations"
CREATE INDEX "idx_staff_invitations_expiry" ON "staff_invitations" ("expires_at") WHERE ((status)::text = 'pending'::text);
-- Create index "idx_staff_invitations_staff_status_created" to table: "staff_invitations"
CREATE INDEX "idx_staff_invitations_staff_status_created" ON "staff_invitations" ("staff_account_id", "status", "created_at");
-- Create index "staff_invitations_token_hash_key" to table: "staff_invitations"
CREATE UNIQUE INDEX "staff_invitations_token_hash_key" ON "staff_invitations" ("token_hash");
