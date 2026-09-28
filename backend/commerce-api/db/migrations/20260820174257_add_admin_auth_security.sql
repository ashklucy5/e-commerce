-- Create "admin_login_challenges" table
CREATE TABLE "admin_login_challenges" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "staff_account_id" uuid NOT NULL,
  "challenge_token_hash" character varying(64) NOT NULL,
  "purpose" character varying(20) NOT NULL DEFAULT 'login',
  "status" character varying(20) NOT NULL DEFAULT 'pending',
  "failed_attempts" integer NOT NULL DEFAULT 0,
  "max_attempts" integer NOT NULL DEFAULT 5,
  "expires_at" timestamptz NOT NULL,
  "verified_at" timestamptz NULL,
  "consumed_at" timestamptz NULL,
  "client_ip" character varying(64) NULL,
  "user_agent" character varying(500) NULL,
  "request_id" character varying(128) NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "admin_login_challenges_staff_account_id_fkey" FOREIGN KEY ("staff_account_id") REFERENCES "staff_accounts" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "admin_login_challenges_attempts_valid" CHECK ((failed_attempts >= 0) AND (max_attempts > 0) AND (failed_attempts <= max_attempts)),
  CONSTRAINT "admin_login_challenges_consumed_valid" CHECK (((status)::text <> 'consumed'::text) OR (consumed_at IS NOT NULL)),
  CONSTRAINT "admin_login_challenges_expiry_valid" CHECK (expires_at > created_at),
  CONSTRAINT "admin_login_challenges_hash_valid" CHECK (length((challenge_token_hash)::text) = 64),
  CONSTRAINT "admin_login_challenges_purpose_valid" CHECK ((purpose)::text = ANY ((ARRAY['login'::character varying, 'reauth'::character varying])::text[])),
  CONSTRAINT "admin_login_challenges_status_valid" CHECK ((status)::text = ANY ((ARRAY['pending'::character varying, 'verified'::character varying, 'consumed'::character varying, 'expired'::character varying, 'locked'::character varying, 'cancelled'::character varying])::text[])),
  CONSTRAINT "admin_login_challenges_verified_valid" CHECK (((status)::text <> 'verified'::text) OR (verified_at IS NOT NULL))
);
-- Create index "admin_login_challenges_token_hash_key" to table: "admin_login_challenges"
CREATE UNIQUE INDEX "admin_login_challenges_token_hash_key" ON "admin_login_challenges" ("challenge_token_hash");
-- Create index "idx_admin_login_challenges_account_status" to table: "admin_login_challenges"
CREATE INDEX "idx_admin_login_challenges_account_status" ON "admin_login_challenges" ("staff_account_id", "status", "created_at");
-- Create index "idx_admin_login_challenges_expiry" to table: "admin_login_challenges"
CREATE INDEX "idx_admin_login_challenges_expiry" ON "admin_login_challenges" ("status", "expires_at");
-- Create "admin_mfa_recovery_codes" table
CREATE TABLE "admin_mfa_recovery_codes" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "staff_account_id" uuid NOT NULL,
  "code_hash" character varying(64) NOT NULL,
  "used_at" timestamptz NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "admin_mfa_recovery_codes_staff_account_id_fkey" FOREIGN KEY ("staff_account_id") REFERENCES "staff_accounts" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "admin_mfa_recovery_codes_hash_valid" CHECK (length((code_hash)::text) = 64)
);
-- Create index "admin_mfa_recovery_codes_code_hash_key" to table: "admin_mfa_recovery_codes"
CREATE UNIQUE INDEX "admin_mfa_recovery_codes_code_hash_key" ON "admin_mfa_recovery_codes" ("code_hash");
-- Create index "idx_admin_mfa_recovery_codes_account_used" to table: "admin_mfa_recovery_codes"
CREATE INDEX "idx_admin_mfa_recovery_codes_account_used" ON "admin_mfa_recovery_codes" ("staff_account_id", "used_at");
-- Create "admin_sessions" table
CREATE TABLE "admin_sessions" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "staff_account_id" uuid NOT NULL,
  "login_challenge_id" uuid NULL,
  "access_token_hash" character varying(64) NOT NULL,
  "refresh_token_hash" character varying(64) NOT NULL,
  "csrf_token_hash" character varying(64) NOT NULL,
  "refresh_generation" integer NOT NULL DEFAULT 0,
  "access_expires_at" timestamptz NOT NULL,
  "refresh_expires_at" timestamptz NOT NULL,
  "authenticated_at" timestamptz NOT NULL,
  "mfa_verified_at" timestamptz NOT NULL,
  "last_used_at" timestamptz NULL,
  "last_rotated_at" timestamptz NULL,
  "created_ip" character varying(64) NULL,
  "last_ip" character varying(64) NULL,
  "user_agent" character varying(500) NULL,
  "revoked_at" timestamptz NULL,
  "revoke_reason" character varying(250) NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "admin_sessions_login_challenge_id_fkey" FOREIGN KEY ("login_challenge_id") REFERENCES "admin_login_challenges" ("id") ON UPDATE NO ACTION ON DELETE SET NULL,
  CONSTRAINT "admin_sessions_staff_account_id_fkey" FOREIGN KEY ("staff_account_id") REFERENCES "staff_accounts" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "admin_sessions_access_hash_valid" CHECK (length((access_token_hash)::text) = 64),
  CONSTRAINT "admin_sessions_csrf_hash_valid" CHECK (length((csrf_token_hash)::text) = 64),
  CONSTRAINT "admin_sessions_expiry_valid" CHECK (refresh_expires_at > access_expires_at),
  CONSTRAINT "admin_sessions_refresh_generation_valid" CHECK (refresh_generation >= 0),
  CONSTRAINT "admin_sessions_refresh_hash_valid" CHECK (length((refresh_token_hash)::text) = 64),
  CONSTRAINT "admin_sessions_revoke_reason_valid" CHECK ((revoke_reason IS NULL) OR (revoked_at IS NOT NULL))
);
-- Create index "admin_sessions_access_token_hash_key" to table: "admin_sessions"
CREATE UNIQUE INDEX "admin_sessions_access_token_hash_key" ON "admin_sessions" ("access_token_hash");
-- Create index "admin_sessions_csrf_token_hash_key" to table: "admin_sessions"
CREATE UNIQUE INDEX "admin_sessions_csrf_token_hash_key" ON "admin_sessions" ("csrf_token_hash");
-- Create index "admin_sessions_login_challenge_id_key" to table: "admin_sessions"
CREATE UNIQUE INDEX "admin_sessions_login_challenge_id_key" ON "admin_sessions" ("login_challenge_id");
-- Create index "admin_sessions_refresh_token_hash_key" to table: "admin_sessions"
CREATE UNIQUE INDEX "admin_sessions_refresh_token_hash_key" ON "admin_sessions" ("refresh_token_hash");
-- Create index "idx_admin_sessions_account_created" to table: "admin_sessions"
CREATE INDEX "idx_admin_sessions_account_created" ON "admin_sessions" ("staff_account_id", "created_at");
-- Create index "idx_admin_sessions_refresh_expiry" to table: "admin_sessions"
CREATE INDEX "idx_admin_sessions_refresh_expiry" ON "admin_sessions" ("refresh_expires_at") WHERE (revoked_at IS NULL);
-- Create "admin_security_events" table
CREATE TABLE "admin_security_events" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "staff_account_id" uuid NULL,
  "admin_session_id" uuid NULL,
  "event_type" character varying(80) NOT NULL,
  "outcome" character varying(20) NOT NULL,
  "identifier_hash" character varying(64) NULL,
  "request_id" character varying(128) NULL,
  "ip_address" character varying(64) NULL,
  "user_agent" character varying(500) NULL,
  "details" jsonb NULL,
  "occurred_at" timestamptz NOT NULL DEFAULT now(),
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "admin_security_events_admin_session_id_fkey" FOREIGN KEY ("admin_session_id") REFERENCES "admin_sessions" ("id") ON UPDATE NO ACTION ON DELETE SET NULL,
  CONSTRAINT "admin_security_events_staff_account_id_fkey" FOREIGN KEY ("staff_account_id") REFERENCES "staff_accounts" ("id") ON UPDATE NO ACTION ON DELETE SET NULL,
  CONSTRAINT "admin_security_events_identifier_hash_valid" CHECK ((identifier_hash IS NULL) OR (length((identifier_hash)::text) = 64)),
  CONSTRAINT "admin_security_events_outcome_valid" CHECK ((outcome)::text = ANY ((ARRAY['success'::character varying, 'failure'::character varying, 'blocked'::character varying, 'info'::character varying])::text[])),
  CONSTRAINT "admin_security_events_type_not_blank" CHECK (length(TRIM(BOTH FROM event_type)) > 0)
);
-- Create index "idx_admin_security_events_session_time" to table: "admin_security_events"
CREATE INDEX "idx_admin_security_events_session_time" ON "admin_security_events" ("admin_session_id", "occurred_at");
-- Create index "idx_admin_security_events_staff_time" to table: "admin_security_events"
CREATE INDEX "idx_admin_security_events_staff_time" ON "admin_security_events" ("staff_account_id", "occurred_at");
-- Create index "idx_admin_security_events_type_time" to table: "admin_security_events"
CREATE INDEX "idx_admin_security_events_type_time" ON "admin_security_events" ("event_type", "occurred_at");
-- Create "admin_totp_credentials" table
CREATE TABLE "admin_totp_credentials" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "staff_account_id" uuid NOT NULL,
  "label" character varying(120) NOT NULL DEFAULT 'Authenticator',
  "secret_ciphertext" text NOT NULL,
  "encryption_key_id" character varying(80) NOT NULL,
  "algorithm" character varying(20) NOT NULL DEFAULT 'SHA1',
  "digits" integer NOT NULL DEFAULT 6,
  "period_seconds" integer NOT NULL DEFAULT 30,
  "status" character varying(20) NOT NULL DEFAULT 'pending',
  "last_accepted_step" bigint NULL,
  "verified_at" timestamptz NULL,
  "last_used_at" timestamptz NULL,
  "disabled_at" timestamptz NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "admin_totp_credentials_staff_account_id_fkey" FOREIGN KEY ("staff_account_id") REFERENCES "staff_accounts" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "admin_totp_credentials_active_requires_verification" CHECK (((status)::text <> 'active'::text) OR (verified_at IS NOT NULL)),
  CONSTRAINT "admin_totp_credentials_algorithm_valid" CHECK ((algorithm)::text = ANY ((ARRAY['SHA1'::character varying, 'SHA256'::character varying, 'SHA512'::character varying])::text[])),
  CONSTRAINT "admin_totp_credentials_digits_valid" CHECK (digits = ANY (ARRAY[6, 8])),
  CONSTRAINT "admin_totp_credentials_disabled_requires_timestamp" CHECK (((status)::text <> 'disabled'::text) OR (disabled_at IS NOT NULL)),
  CONSTRAINT "admin_totp_credentials_key_id_not_blank" CHECK (length(TRIM(BOTH FROM encryption_key_id)) > 0),
  CONSTRAINT "admin_totp_credentials_label_not_blank" CHECK (length(TRIM(BOTH FROM label)) > 0),
  CONSTRAINT "admin_totp_credentials_period_valid" CHECK ((period_seconds >= 15) AND (period_seconds <= 120)),
  CONSTRAINT "admin_totp_credentials_secret_not_blank" CHECK (length(TRIM(BOTH FROM secret_ciphertext)) > 0),
  CONSTRAINT "admin_totp_credentials_status_valid" CHECK ((status)::text = ANY ((ARRAY['pending'::character varying, 'active'::character varying, 'disabled'::character varying])::text[])),
  CONSTRAINT "admin_totp_credentials_step_valid" CHECK ((last_accepted_step IS NULL) OR (last_accepted_step >= 0))
);
-- Create index "admin_totp_credentials_staff_account_id_key" to table: "admin_totp_credentials"
CREATE UNIQUE INDEX "admin_totp_credentials_staff_account_id_key" ON "admin_totp_credentials" ("staff_account_id");
-- Create index "idx_admin_totp_credentials_status" to table: "admin_totp_credentials"
CREATE INDEX "idx_admin_totp_credentials_status" ON "admin_totp_credentials" ("status", "created_at");
