-- Modify "crm_messages" table
ALTER TABLE "crm_messages" ADD CONSTRAINT "crm_messages_attachments_is_array" CHECK ((attachments IS NULL) OR (jsonb_typeof(attachments) = 'array'::text)), ADD COLUMN "attachments" jsonb NULL;
