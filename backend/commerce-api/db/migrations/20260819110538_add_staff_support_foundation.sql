-- Create "staff_accounts" table
CREATE TABLE "staff_accounts" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "staff_code" character varying(40) NOT NULL,
  "full_name" character varying(160) NOT NULL,
  "email" character varying(255) NOT NULL,
  "phone" character varying(40) NULL,
  "password_hash" text NOT NULL,
  "status" character varying(20) NOT NULL DEFAULT 'active',
  "last_login_at" timestamptz NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "staff_accounts_email_not_blank" CHECK (length(TRIM(BOTH FROM email)) > 0),
  CONSTRAINT "staff_accounts_full_name_not_blank" CHECK (length(TRIM(BOTH FROM full_name)) > 0),
  CONSTRAINT "staff_accounts_password_hash_not_blank" CHECK (length(TRIM(BOTH FROM password_hash)) > 0),
  CONSTRAINT "staff_accounts_phone_not_blank" CHECK ((phone IS NULL) OR (length(TRIM(BOTH FROM phone)) > 0)),
  CONSTRAINT "staff_accounts_staff_code_not_blank" CHECK (length(TRIM(BOTH FROM staff_code)) > 0),
  CONSTRAINT "staff_accounts_status_valid" CHECK ((status)::text = ANY ((ARRAY['active'::character varying, 'suspended'::character varying, 'disabled'::character varying])::text[]))
);
-- Create index "idx_staff_accounts_status_created" to table: "staff_accounts"
CREATE INDEX "idx_staff_accounts_status_created" ON "staff_accounts" ("status", "created_at");
-- Create index "staff_accounts_email_key" to table: "staff_accounts"
CREATE UNIQUE INDEX "staff_accounts_email_key" ON "staff_accounts" ("email");
-- Create index "staff_accounts_phone_key" to table: "staff_accounts"
CREATE UNIQUE INDEX "staff_accounts_phone_key" ON "staff_accounts" ("phone") WHERE (phone IS NOT NULL);
-- Create index "staff_accounts_staff_code_key" to table: "staff_accounts"
CREATE UNIQUE INDEX "staff_accounts_staff_code_key" ON "staff_accounts" ("staff_code");
-- Create "staff_roles" table
CREATE TABLE "staff_roles" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "code" character varying(80) NOT NULL,
  "name" character varying(120) NOT NULL,
  "description" character varying(500) NULL,
  "is_system_role" boolean NOT NULL DEFAULT true,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "staff_roles_code_not_blank" CHECK (length(TRIM(BOTH FROM code)) > 0),
  CONSTRAINT "staff_roles_name_not_blank" CHECK (length(TRIM(BOTH FROM name)) > 0)
);
-- Create index "staff_roles_code_key" to table: "staff_roles"
CREATE UNIQUE INDEX "staff_roles_code_key" ON "staff_roles" ("code");
-- Create "staff_account_roles" table
CREATE TABLE "staff_account_roles" (
  "staff_account_id" uuid NOT NULL,
  "role_id" uuid NOT NULL,
  "assigned_by_staff_id" uuid NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("staff_account_id", "role_id"),
  CONSTRAINT "staff_account_roles_assigned_by_staff_id_fkey" FOREIGN KEY ("assigned_by_staff_id") REFERENCES "staff_accounts" ("id") ON UPDATE NO ACTION ON DELETE SET NULL,
  CONSTRAINT "staff_account_roles_role_id_fkey" FOREIGN KEY ("role_id") REFERENCES "staff_roles" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "staff_account_roles_staff_account_id_fkey" FOREIGN KEY ("staff_account_id") REFERENCES "staff_accounts" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- Create index "idx_staff_account_roles_role" to table: "staff_account_roles"
CREATE INDEX "idx_staff_account_roles_role" ON "staff_account_roles" ("role_id");
-- Create "staff_permissions" table
CREATE TABLE "staff_permissions" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "code" character varying(120) NOT NULL,
  "description" character varying(500) NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "staff_permissions_code_not_blank" CHECK (length(TRIM(BOTH FROM code)) > 0)
);
-- Create index "staff_permissions_code_key" to table: "staff_permissions"
CREATE UNIQUE INDEX "staff_permissions_code_key" ON "staff_permissions" ("code");
-- Create "staff_role_permissions" table
CREATE TABLE "staff_role_permissions" (
  "role_id" uuid NOT NULL,
  "permission_id" uuid NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("role_id", "permission_id"),
  CONSTRAINT "staff_role_permissions_permission_id_fkey" FOREIGN KEY ("permission_id") REFERENCES "staff_permissions" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "staff_role_permissions_role_id_fkey" FOREIGN KEY ("role_id") REFERENCES "staff_roles" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- Create index "idx_staff_role_permissions_permission" to table: "staff_role_permissions"
CREATE INDEX "idx_staff_role_permissions_permission" ON "staff_role_permissions" ("permission_id");
-- Create "staff_sessions" table
CREATE TABLE "staff_sessions" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "staff_account_id" uuid NOT NULL,
  "access_token_hash" character varying(64) NOT NULL,
  "refresh_token_hash" character varying(64) NOT NULL,
  "access_expires_at" timestamptz NOT NULL,
  "refresh_expires_at" timestamptz NOT NULL,
  "last_used_at" timestamptz NULL,
  "revoked_at" timestamptz NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "staff_sessions_staff_account_id_fkey" FOREIGN KEY ("staff_account_id") REFERENCES "staff_accounts" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "staff_sessions_access_token_hash_valid" CHECK (length((access_token_hash)::text) = 64),
  CONSTRAINT "staff_sessions_expiry_valid" CHECK (refresh_expires_at > access_expires_at),
  CONSTRAINT "staff_sessions_refresh_token_hash_valid" CHECK (length((refresh_token_hash)::text) = 64)
);
-- Create index "idx_staff_sessions_account_created" to table: "staff_sessions"
CREATE INDEX "idx_staff_sessions_account_created" ON "staff_sessions" ("staff_account_id", "created_at");
-- Create index "idx_staff_sessions_refresh_expiry" to table: "staff_sessions"
CREATE INDEX "idx_staff_sessions_refresh_expiry" ON "staff_sessions" ("refresh_expires_at") WHERE (revoked_at IS NULL);
-- Create index "staff_sessions_access_token_hash_key" to table: "staff_sessions"
CREATE UNIQUE INDEX "staff_sessions_access_token_hash_key" ON "staff_sessions" ("access_token_hash");
-- Create index "staff_sessions_refresh_token_hash_key" to table: "staff_sessions"
CREATE UNIQUE INDEX "staff_sessions_refresh_token_hash_key" ON "staff_sessions" ("refresh_token_hash");
-- Create "support_actors" table
CREATE TABLE "support_actors" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "actor_code" character varying(40) NOT NULL,
  "actor_type" character varying(20) NOT NULL,
  "staff_account_id" uuid NULL,
  "display_name" character varying(160) NOT NULL,
  "status" character varying(20) NOT NULL DEFAULT 'active',
  "presence" character varying(20) NOT NULL DEFAULT 'offline',
  "max_active_cases" integer NOT NULL DEFAULT 10,
  "last_presence_at" timestamptz NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "support_actors_staff_account_id_fkey" FOREIGN KEY ("staff_account_id") REFERENCES "staff_accounts" ("id") ON UPDATE NO ACTION ON DELETE SET NULL,
  CONSTRAINT "support_actors_actor_code_not_blank" CHECK (length(TRIM(BOTH FROM actor_code)) > 0),
  CONSTRAINT "support_actors_actor_type_valid" CHECK ((actor_type)::text = ANY ((ARRAY['human'::character varying, 'ai'::character varying, 'system'::character varying])::text[])),
  CONSTRAINT "support_actors_display_name_not_blank" CHECK (length(TRIM(BOTH FROM display_name)) > 0),
  CONSTRAINT "support_actors_human_staff_consistent" CHECK ((((actor_type)::text = 'human'::text) AND (staff_account_id IS NOT NULL)) OR (((actor_type)::text <> 'human'::text) AND (staff_account_id IS NULL))),
  CONSTRAINT "support_actors_max_active_cases_positive" CHECK (max_active_cases > 0),
  CONSTRAINT "support_actors_presence_valid" CHECK ((presence)::text = ANY ((ARRAY['offline'::character varying, 'available'::character varying, 'busy'::character varying, 'away'::character varying])::text[])),
  CONSTRAINT "support_actors_status_valid" CHECK ((status)::text = ANY ((ARRAY['active'::character varying, 'disabled'::character varying])::text[]))
);
-- Create index "idx_support_actors_presence" to table: "support_actors"
CREATE INDEX "idx_support_actors_presence" ON "support_actors" ("status", "presence");
-- Create index "support_actors_actor_code_key" to table: "support_actors"
CREATE UNIQUE INDEX "support_actors_actor_code_key" ON "support_actors" ("actor_code");
-- Create index "support_actors_staff_account_id_key" to table: "support_actors"
CREATE UNIQUE INDEX "support_actors_staff_account_id_key" ON "support_actors" ("staff_account_id") WHERE (staff_account_id IS NOT NULL);
-- Create "support_queues" table
CREATE TABLE "support_queues" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "code" character varying(80) NOT NULL,
  "name" character varying(120) NOT NULL,
  "description" character varying(500) NULL,
  "status" character varying(20) NOT NULL DEFAULT 'active',
  "sort_order" integer NOT NULL DEFAULT 0,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "support_queues_code_not_blank" CHECK (length(TRIM(BOTH FROM code)) > 0),
  CONSTRAINT "support_queues_name_not_blank" CHECK (length(TRIM(BOTH FROM name)) > 0),
  CONSTRAINT "support_queues_sort_order_nonnegative" CHECK (sort_order >= 0),
  CONSTRAINT "support_queues_status_valid" CHECK ((status)::text = ANY ((ARRAY['active'::character varying, 'disabled'::character varying])::text[]))
);
-- Create index "idx_support_queues_status_sort" to table: "support_queues"
CREATE INDEX "idx_support_queues_status_sort" ON "support_queues" ("status", "sort_order");
-- Create index "support_queues_code_key" to table: "support_queues"
CREATE UNIQUE INDEX "support_queues_code_key" ON "support_queues" ("code");
-- Create "support_queue_members" table
CREATE TABLE "support_queue_members" (
  "queue_id" uuid NOT NULL,
  "support_actor_id" uuid NOT NULL,
  "membership_role" character varying(20) NOT NULL DEFAULT 'member',
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("queue_id", "support_actor_id"),
  CONSTRAINT "support_queue_members_queue_id_fkey" FOREIGN KEY ("queue_id") REFERENCES "support_queues" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "support_queue_members_support_actor_id_fkey" FOREIGN KEY ("support_actor_id") REFERENCES "support_actors" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "support_queue_members_role_valid" CHECK ((membership_role)::text = ANY ((ARRAY['member'::character varying, 'lead'::character varying])::text[]))
);
-- Create index "idx_support_queue_members_actor" to table: "support_queue_members"
CREATE INDEX "idx_support_queue_members_actor" ON "support_queue_members" ("support_actor_id");
