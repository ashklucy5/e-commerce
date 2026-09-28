-- Create "delivery_otp_challenges" table
CREATE TABLE "delivery_otp_challenges" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "shipment_id" uuid NOT NULL,
  "purpose" character varying(40) NOT NULL,
  "channel" character varying(20) NOT NULL DEFAULT 'sms',
  "recipient" character varying(255) NOT NULL,
  "code_hash" character varying(255) NOT NULL,
  "status" character varying(20) NOT NULL DEFAULT 'pending',
  "expires_at" timestamptz NOT NULL,
  "attempt_count" integer NOT NULL DEFAULT 0,
  "max_attempts" integer NOT NULL DEFAULT 5,
  "verified_at" timestamptz NULL,
  "consumed_at" timestamptz NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "delivery_otp_challenges_shipment_fkey" FOREIGN KEY ("shipment_id") REFERENCES "shipments" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "delivery_otp_challenges_attempts_valid" CHECK ((attempt_count >= 0) AND (max_attempts >= 1) AND (max_attempts <= 10) AND (attempt_count <= max_attempts)),
  CONSTRAINT "delivery_otp_challenges_channel_valid" CHECK ((channel)::text = 'sms'::text),
  CONSTRAINT "delivery_otp_challenges_code_hash_not_blank" CHECK (length(TRIM(BOTH FROM code_hash)) > 0),
  CONSTRAINT "delivery_otp_challenges_consumed_timestamp" CHECK (((status)::text <> 'consumed'::text) OR ((verified_at IS NOT NULL) AND (consumed_at IS NOT NULL))),
  CONSTRAINT "delivery_otp_challenges_expiry_valid" CHECK (expires_at > created_at),
  CONSTRAINT "delivery_otp_challenges_purpose_valid" CHECK ((purpose)::text = ANY ((ARRAY['delivery_confirmation'::character varying, 'self_pickup'::character varying, 'rider_pickup'::character varying, 'rider_delivery'::character varying])::text[])),
  CONSTRAINT "delivery_otp_challenges_recipient_not_blank" CHECK (length(TRIM(BOTH FROM recipient)) > 0),
  CONSTRAINT "delivery_otp_challenges_status_valid" CHECK ((status)::text = ANY ((ARRAY['pending'::character varying, 'verified'::character varying, 'consumed'::character varying, 'expired'::character varying, 'locked'::character varying, 'cancelled'::character varying])::text[])),
  CONSTRAINT "delivery_otp_challenges_verified_timestamp" CHECK (((status)::text <> ALL ((ARRAY['verified'::character varying, 'consumed'::character varying])::text[])) OR (verified_at IS NOT NULL))
);
-- Create index "delivery_otp_challenges_one_active" to table: "delivery_otp_challenges"
CREATE UNIQUE INDEX "delivery_otp_challenges_one_active" ON "delivery_otp_challenges" ("shipment_id", "purpose", "channel", "recipient") WHERE ((status)::text = ANY ((ARRAY['pending'::character varying, 'verified'::character varying])::text[]));
-- Create index "idx_delivery_otp_challenges_expiry" to table: "delivery_otp_challenges"
CREATE INDEX "idx_delivery_otp_challenges_expiry" ON "delivery_otp_challenges" ("status", "expires_at");
-- Create index "idx_delivery_otp_challenges_shipment_status" to table: "delivery_otp_challenges"
CREATE INDEX "idx_delivery_otp_challenges_shipment_status" ON "delivery_otp_challenges" ("shipment_id", "status", "created_at");
-- Create "delivery_proofs" table
CREATE TABLE "delivery_proofs" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "shipment_id" uuid NOT NULL,
  "otp_challenge_id" uuid NULL,
  "purpose" character varying(40) NOT NULL,
  "proof_type" character varying(40) NOT NULL,
  "source" character varying(30) NOT NULL,
  "actor_id" character varying(160) NULL,
  "storage_key" character varying(1000) NULL,
  "external_reference" character varying(160) NULL,
  "verification_status" character varying(20) NOT NULL DEFAULT 'recorded',
  "metadata" jsonb NULL,
  "occurred_at" timestamptz NOT NULL DEFAULT now(),
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "delivery_proofs_otp_challenge_fkey" FOREIGN KEY ("otp_challenge_id") REFERENCES "delivery_otp_challenges" ("id") ON UPDATE NO ACTION ON DELETE SET NULL,
  CONSTRAINT "delivery_proofs_shipment_fkey" FOREIGN KEY ("shipment_id") REFERENCES "shipments" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "delivery_proofs_actor_not_blank" CHECK ((actor_id IS NULL) OR (length(TRIM(BOTH FROM actor_id)) > 0)),
  CONSTRAINT "delivery_proofs_external_reference_not_blank" CHECK ((external_reference IS NULL) OR (length(TRIM(BOTH FROM external_reference)) > 0)),
  CONSTRAINT "delivery_proofs_purpose_valid" CHECK ((purpose)::text = ANY ((ARRAY['delivery_confirmation'::character varying, 'self_pickup'::character varying, 'rider_pickup'::character varying, 'rider_delivery'::character varying])::text[])),
  CONSTRAINT "delivery_proofs_source_valid" CHECK ((source)::text = ANY ((ARRAY['manual'::character varying, 'provider'::character varying, 'rider'::character varying, 'customer'::character varying, 'support'::character varying, 'admin'::character varying, 'system'::character varying])::text[])),
  CONSTRAINT "delivery_proofs_storage_key_not_blank" CHECK ((storage_key IS NULL) OR (length(TRIM(BOTH FROM storage_key)) > 0)),
  CONSTRAINT "delivery_proofs_type_valid" CHECK ((proof_type)::text = ANY ((ARRAY['otp'::character varying, 'photo'::character varying, 'signature'::character varying, 'provider_reference'::character varying, 'receipt_confirmation'::character varying, 'pickup_confirmation'::character varying, 'staff_confirmation'::character varying])::text[])),
  CONSTRAINT "delivery_proofs_verification_status_valid" CHECK ((verification_status)::text = ANY ((ARRAY['recorded'::character varying, 'pending'::character varying, 'verified'::character varying, 'rejected'::character varying])::text[]))
);
-- Create index "delivery_proofs_otp_challenge_key" to table: "delivery_proofs"
CREATE UNIQUE INDEX "delivery_proofs_otp_challenge_key" ON "delivery_proofs" ("otp_challenge_id") WHERE (otp_challenge_id IS NOT NULL);
-- Create index "idx_delivery_proofs_shipment_time" to table: "delivery_proofs"
CREATE INDEX "idx_delivery_proofs_shipment_time" ON "delivery_proofs" ("shipment_id", "occurred_at", "created_at");
