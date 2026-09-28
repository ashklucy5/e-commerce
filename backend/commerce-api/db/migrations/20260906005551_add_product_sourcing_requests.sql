-- Modify "crm_cases" table
ALTER TABLE "crm_cases" DROP CONSTRAINT "crm_cases_type_valid", ADD CONSTRAINT "crm_cases_type_valid" CHECK ((case_type)::text = ANY ((ARRAY['bulk_stock_request'::character varying, 'product_request'::character varying, 'product_question'::character varying, 'order_issue'::character varying, 'payment_issue'::character varying, 'return_issue'::character varying, 'delivery_issue'::character varying, 'complaint'::character varying, 'general_question'::character varying, 'other'::character varying])::text[]));
-- Create "product_sourcing_requests" table
CREATE TABLE "product_sourcing_requests" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "case_id" uuid NOT NULL,
  "request_number" character varying(60) NOT NULL,
  "requested_product_name" character varying(180) NOT NULL,
  "description" text NOT NULL,
  "external_url" text NULL,
  "attachments" jsonb NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "product_sourcing_requests_case_id_fkey" FOREIGN KEY ("case_id") REFERENCES "crm_cases" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "product_sourcing_requests_attachments_is_array" CHECK ((attachments IS NULL) OR (jsonb_typeof(attachments) = 'array'::text)),
  CONSTRAINT "product_sourcing_requests_description_not_blank" CHECK (length(TRIM(BOTH FROM description)) > 0),
  CONSTRAINT "product_sourcing_requests_name_not_blank" CHECK (length(TRIM(BOTH FROM requested_product_name)) > 0)
);
-- Create index "product_sourcing_requests_case_key" to table: "product_sourcing_requests"
CREATE UNIQUE INDEX "product_sourcing_requests_case_key" ON "product_sourcing_requests" ("case_id");
-- Create index "product_sourcing_requests_request_number_key" to table: "product_sourcing_requests"
CREATE UNIQUE INDEX "product_sourcing_requests_request_number_key" ON "product_sourcing_requests" ("request_number");
