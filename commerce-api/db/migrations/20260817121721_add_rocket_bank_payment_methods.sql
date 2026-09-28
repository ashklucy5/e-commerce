-- Modify "payment_events" table
ALTER TABLE "payment_events" DROP CONSTRAINT "payment_events_provider_valid", ADD CONSTRAINT "payment_events_provider_valid" CHECK ((provider)::text = ANY ((ARRAY['bkash'::character varying, 'nagad'::character varying, 'rocket'::character varying, 'bank_transfer'::character varying])::text[]));
-- Modify "payments" table
ALTER TABLE "payments" DROP CONSTRAINT "payments_provider_valid", ADD CONSTRAINT "payments_provider_valid" CHECK ((provider)::text = ANY ((ARRAY['bkash'::character varying, 'nagad'::character varying, 'rocket'::character varying, 'bank_transfer'::character varying])::text[]));
