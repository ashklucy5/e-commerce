-- Rename a column from "variants" to "offered_specifications"
ALTER TABLE "product_sourcing_offers" RENAME COLUMN "variants" TO "offered_specifications";
-- Modify "product_sourcing_offers" table
ALTER TABLE "product_sourcing_offers" DROP CONSTRAINT "product_sourcing_offers_variants_is_object", ADD CONSTRAINT "product_sourcing_offers_offered_specifications_is_object" CHECK ((offered_specifications IS NULL) OR (jsonb_typeof(offered_specifications) = 'object'::text));
-- Modify "product_sourcing_requests" table
ALTER TABLE "product_sourcing_requests" ADD CONSTRAINT "product_sourcing_requests_customer_requirements_is_object" CHECK ((customer_requirements IS NULL) OR (jsonb_typeof(customer_requirements) = 'object'::text)), ADD COLUMN "customer_requirements" jsonb NULL;
