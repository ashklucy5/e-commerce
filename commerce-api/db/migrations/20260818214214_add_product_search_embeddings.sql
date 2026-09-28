-- Create "product_search_embeddings" table
CREATE TABLE "product_search_embeddings" (
  "product_id" uuid NOT NULL,
  "document_text" text NOT NULL,
  "document_hash" character varying(64) NOT NULL,
  "embedding" real[] NOT NULL,
  "embedding_model" character varying(80) NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("product_id"),
  CONSTRAINT "product_search_embeddings_product_id_fkey" FOREIGN KEY ("product_id") REFERENCES "products" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "product_search_embeddings_dimensions_valid" CHECK (cardinality(embedding) = 1024),
  CONSTRAINT "product_search_embeddings_document_not_blank" CHECK (length(TRIM(BOTH FROM document_text)) > 0),
  CONSTRAINT "product_search_embeddings_hash_valid" CHECK (length((document_hash)::text) = 64),
  CONSTRAINT "product_search_embeddings_model_not_blank" CHECK (length(TRIM(BOTH FROM embedding_model)) > 0),
  CONSTRAINT "product_search_embeddings_no_null_values" CHECK (array_position(embedding, NULL::real) IS NULL)
);
-- Create index "idx_product_search_embeddings_model" to table: "product_search_embeddings"
CREATE INDEX "idx_product_search_embeddings_model" ON "product_search_embeddings" ("embedding_model");
