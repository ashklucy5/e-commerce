-- Create "product_360_frames" table
CREATE TABLE "product_360_frames" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "product_id" uuid NOT NULL,
  "variant_id" uuid NULL,
  "storage_key" text NOT NULL,
  "url" text NOT NULL,
  "frame_index" integer NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "product_360_frames_product_id_fkey" FOREIGN KEY ("product_id") REFERENCES "products" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "product_360_frames_variant_product_fkey" FOREIGN KEY ("variant_id", "product_id") REFERENCES "product_variants" ("id", "product_id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "product_360_frames_frame_index_nonnegative" CHECK (frame_index >= 0),
  CONSTRAINT "product_360_frames_storage_key_not_blank" CHECK (length(TRIM(BOTH FROM storage_key)) > 0),
  CONSTRAINT "product_360_frames_url_not_blank" CHECK (length(TRIM(BOTH FROM url)) > 0)
);
-- Create index "idx_product_360_frames_product" to table: "product_360_frames"
CREATE INDEX "idx_product_360_frames_product" ON "product_360_frames" ("product_id", "frame_index");
-- Create index "idx_product_360_frames_variant" to table: "product_360_frames"
CREATE INDEX "idx_product_360_frames_variant" ON "product_360_frames" ("variant_id", "frame_index");
-- Create index "product_360_frames_product_frame_key" to table: "product_360_frames"
CREATE UNIQUE INDEX "product_360_frames_product_frame_key" ON "product_360_frames" ("product_id", "frame_index") WHERE (variant_id IS NULL);
-- Create index "product_360_frames_storage_key_key" to table: "product_360_frames"
CREATE UNIQUE INDEX "product_360_frames_storage_key_key" ON "product_360_frames" ("storage_key");
-- Create index "product_360_frames_variant_frame_key" to table: "product_360_frames"
CREATE UNIQUE INDEX "product_360_frames_variant_frame_key" ON "product_360_frames" ("product_id", "variant_id", "frame_index") WHERE (variant_id IS NOT NULL);
-- Create "product_3d_models" table
CREATE TABLE "product_3d_models" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "product_id" uuid NOT NULL,
  "variant_id" uuid NULL,
  "storage_key" text NOT NULL,
  "url" text NOT NULL,
  "poster_url" text NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "product_3d_models_product_id_fkey" FOREIGN KEY ("product_id") REFERENCES "products" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "product_3d_models_variant_product_fkey" FOREIGN KEY ("variant_id", "product_id") REFERENCES "product_variants" ("id", "product_id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "product_3d_models_poster_url_not_blank" CHECK ((poster_url IS NULL) OR (length(TRIM(BOTH FROM poster_url)) > 0)),
  CONSTRAINT "product_3d_models_storage_key_not_blank" CHECK (length(TRIM(BOTH FROM storage_key)) > 0),
  CONSTRAINT "product_3d_models_url_not_blank" CHECK (length(TRIM(BOTH FROM url)) > 0)
);
-- Create index "idx_product_3d_models_variant" to table: "product_3d_models"
CREATE INDEX "idx_product_3d_models_variant" ON "product_3d_models" ("variant_id");
-- Create index "product_3d_models_product_key" to table: "product_3d_models"
CREATE UNIQUE INDEX "product_3d_models_product_key" ON "product_3d_models" ("product_id") WHERE (variant_id IS NULL);
-- Create index "product_3d_models_storage_key_key" to table: "product_3d_models"
CREATE UNIQUE INDEX "product_3d_models_storage_key_key" ON "product_3d_models" ("storage_key");
-- Create index "product_3d_models_variant_key" to table: "product_3d_models"
CREATE UNIQUE INDEX "product_3d_models_variant_key" ON "product_3d_models" ("product_id", "variant_id") WHERE (variant_id IS NOT NULL);
