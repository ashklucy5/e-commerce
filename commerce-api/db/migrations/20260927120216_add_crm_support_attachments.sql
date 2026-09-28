-- Create "crm_support_attachments" table
CREATE TABLE "crm_support_attachments" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "case_id" uuid NULL,
  "message_id" uuid NULL,
  "uploader_type" character varying(20) NOT NULL,
  "customer_id" uuid NULL,
  "support_actor_id" uuid NULL,
  "storage_key" text NOT NULL,
  "original_filename" character varying(255) NOT NULL,
  "mime_type" character varying(100) NOT NULL,
  "byte_size" bigint NOT NULL,
  "status" character varying(20) NOT NULL DEFAULT 'pending',
  "uploaded_at" timestamptz NULL,
  "deleted_at" timestamptz NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "crm_support_attachments_case_id_fkey" FOREIGN KEY ("case_id") REFERENCES "crm_cases" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "crm_support_attachments_customer_id_fkey" FOREIGN KEY ("customer_id") REFERENCES "customers" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "crm_support_attachments_message_id_fkey" FOREIGN KEY ("message_id") REFERENCES "crm_messages" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "crm_support_attachments_support_actor_id_fkey" FOREIGN KEY ("support_actor_id") REFERENCES "support_actors" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "crm_support_attachments_byte_size_positive" CHECK (byte_size > 0),
  CONSTRAINT "crm_support_attachments_message_requires_case" CHECK ((message_id IS NULL) OR (case_id IS NOT NULL)),
  CONSTRAINT "crm_support_attachments_mime_type_valid" CHECK ((mime_type)::text = ANY ((ARRAY['image/jpeg'::character varying, 'image/png'::character varying, 'image/webp'::character varying, 'image/gif'::character varying])::text[])),
  CONSTRAINT "crm_support_attachments_original_filename_not_blank" CHECK (length(TRIM(BOTH FROM original_filename)) > 0),
  CONSTRAINT "crm_support_attachments_state_consistent" CHECK ((((status)::text = 'pending'::text) AND (message_id IS NULL) AND (uploaded_at IS NULL) AND (deleted_at IS NULL)) OR (((status)::text = 'ready'::text) AND (message_id IS NULL) AND (uploaded_at IS NOT NULL) AND (deleted_at IS NULL)) OR (((status)::text = 'attached'::text) AND (case_id IS NOT NULL) AND (message_id IS NOT NULL) AND (uploaded_at IS NOT NULL) AND (deleted_at IS NULL)) OR (((status)::text = 'deleted'::text) AND (deleted_at IS NOT NULL))),
  CONSTRAINT "crm_support_attachments_status_valid" CHECK ((status)::text = ANY ((ARRAY['pending'::character varying, 'ready'::character varying, 'attached'::character varying, 'deleted'::character varying])::text[])),
  CONSTRAINT "crm_support_attachments_storage_key_valid" CHECK ((storage_key ~~ 'private/support/%'::text) AND (length(storage_key) > length('private/support/'::text))),
  CONSTRAINT "crm_support_attachments_uploader_consistent" CHECK ((((uploader_type)::text = 'customer'::text) AND (customer_id IS NOT NULL) AND (support_actor_id IS NULL)) OR (((uploader_type)::text = 'support'::text) AND (customer_id IS NULL) AND (support_actor_id IS NOT NULL))),
  CONSTRAINT "crm_support_attachments_uploader_type_valid" CHECK ((uploader_type)::text = ANY ((ARRAY['customer'::character varying, 'support'::character varying])::text[]))
);
-- Create index "crm_support_attachments_storage_key_key" to table: "crm_support_attachments"
CREATE UNIQUE INDEX "crm_support_attachments_storage_key_key" ON "crm_support_attachments" ("storage_key");
-- Create index "idx_crm_support_attachments_case_status" to table: "crm_support_attachments"
CREATE INDEX "idx_crm_support_attachments_case_status" ON "crm_support_attachments" ("case_id", "status");
-- Create index "idx_crm_support_attachments_message" to table: "crm_support_attachments"
CREATE INDEX "idx_crm_support_attachments_message" ON "crm_support_attachments" ("message_id");
-- Create index "idx_crm_support_attachments_status_created" to table: "crm_support_attachments"
CREATE INDEX "idx_crm_support_attachments_status_created" ON "crm_support_attachments" ("status", "created_at");
