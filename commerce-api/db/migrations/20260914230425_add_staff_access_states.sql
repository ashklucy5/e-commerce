-- Modify "staff_accounts" table
ALTER TABLE "staff_accounts" DROP CONSTRAINT "staff_accounts_status_valid", ADD CONSTRAINT "staff_accounts_status_valid" CHECK ((status)::text = ANY ((ARRAY['active'::character varying, 'suspended'::character varying, 'disabled'::character varying, 'deleted'::character varying, 'banned'::character varying])::text[]));
