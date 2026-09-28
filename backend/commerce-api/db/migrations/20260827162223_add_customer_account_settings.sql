-- Create "customer_notification_preferences" table
CREATE TABLE "customer_notification_preferences" (
  "customer_id" uuid NOT NULL,
  "order_updates_sms" boolean NOT NULL DEFAULT true,
  "order_updates_email" boolean NOT NULL DEFAULT true,
  "delivery_updates_sms" boolean NOT NULL DEFAULT true,
  "delivery_updates_email" boolean NOT NULL DEFAULT true,
  "support_updates_sms" boolean NOT NULL DEFAULT true,
  "support_updates_email" boolean NOT NULL DEFAULT true,
  "promotions_sms" boolean NOT NULL DEFAULT false,
  "promotions_email" boolean NOT NULL DEFAULT false,
  "recommendations_email" boolean NOT NULL DEFAULT false,
  "push_enabled" boolean NOT NULL DEFAULT false,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("customer_id"),
  CONSTRAINT "customer_notification_preferences_customer_id_fkey" FOREIGN KEY ("customer_id") REFERENCES "customers" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- Create "customer_preferences" table
CREATE TABLE "customer_preferences" (
  "customer_id" uuid NOT NULL,
  "locale" character varying(10) NOT NULL DEFAULT 'en',
  "assistant_language" character varying(20) NOT NULL DEFAULT 'auto',
  "currency" character varying(3) NOT NULL DEFAULT 'BDT',
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("customer_id"),
  CONSTRAINT "customer_preferences_customer_id_fkey" FOREIGN KEY ("customer_id") REFERENCES "customers" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "customer_preferences_assistant_language_valid" CHECK ((assistant_language)::text = ANY ((ARRAY['auto'::character varying, 'en'::character varying, 'bn'::character varying, 'banglish'::character varying])::text[])),
  CONSTRAINT "customer_preferences_currency_valid" CHECK ((length((currency)::text) = 3) AND ((currency)::text = upper((currency)::text))),
  CONSTRAINT "customer_preferences_locale_valid" CHECK ((locale)::text = ANY ((ARRAY['en'::character varying, 'bn'::character varying])::text[]))
);
-- Create "customer_privacy_settings" table
CREATE TABLE "customer_privacy_settings" (
  "customer_id" uuid NOT NULL,
  "personalization_enabled" boolean NOT NULL DEFAULT true,
  "search_history_enabled" boolean NOT NULL DEFAULT true,
  "recently_viewed_enabled" boolean NOT NULL DEFAULT true,
  "save_tryon_media" boolean NOT NULL DEFAULT false,
  "use_tryon_for_personalization" boolean NOT NULL DEFAULT false,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("customer_id"),
  CONSTRAINT "customer_privacy_settings_customer_id_fkey" FOREIGN KEY ("customer_id") REFERENCES "customers" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- Create "customer_saved_sizes" table
CREATE TABLE "customer_saved_sizes" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "customer_id" uuid NOT NULL,
  "category_key" character varying(80) NOT NULL,
  "brand" character varying(160) NULL,
  "size_system" character varying(20) NULL,
  "size_label" character varying(40) NOT NULL,
  "fit_preference" character varying(30) NULL,
  "notes" character varying(255) NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "customer_saved_sizes_customer_id_fkey" FOREIGN KEY ("customer_id") REFERENCES "customers" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "customer_saved_sizes_brand_not_blank" CHECK ((brand IS NULL) OR (length(TRIM(BOTH FROM brand)) > 0)),
  CONSTRAINT "customer_saved_sizes_category_not_blank" CHECK (length(TRIM(BOTH FROM category_key)) > 0),
  CONSTRAINT "customer_saved_sizes_fit_not_blank" CHECK ((fit_preference IS NULL) OR (length(TRIM(BOTH FROM fit_preference)) > 0)),
  CONSTRAINT "customer_saved_sizes_label_not_blank" CHECK (length(TRIM(BOTH FROM size_label)) > 0),
  CONSTRAINT "customer_saved_sizes_notes_not_blank" CHECK ((notes IS NULL) OR (length(TRIM(BOTH FROM notes)) > 0)),
  CONSTRAINT "customer_saved_sizes_system_not_blank" CHECK ((size_system IS NULL) OR (length(TRIM(BOTH FROM size_system)) > 0))
);
-- Create index "customer_saved_sizes_brand_key" to table: "customer_saved_sizes"
CREATE UNIQUE INDEX "customer_saved_sizes_brand_key" ON "customer_saved_sizes" ("customer_id", "category_key", "brand") WHERE (brand IS NOT NULL);
-- Create index "customer_saved_sizes_generic_key" to table: "customer_saved_sizes"
CREATE UNIQUE INDEX "customer_saved_sizes_generic_key" ON "customer_saved_sizes" ("customer_id", "category_key") WHERE (brand IS NULL);
-- Create index "idx_customer_saved_sizes_customer" to table: "customer_saved_sizes"
CREATE INDEX "idx_customer_saved_sizes_customer" ON "customer_saved_sizes" ("customer_id", "created_at");
-- Create "customer_style_profiles" table
CREATE TABLE "customer_style_profiles" (
  "customer_id" uuid NOT NULL,
  "preferred_colors" jsonb NOT NULL DEFAULT '[]',
  "preferred_styles" jsonb NOT NULL DEFAULT '[]',
  "preferred_categories" jsonb NOT NULL DEFAULT '[]',
  "fit_preference" character varying(30) NULL,
  "budget_min_amount" bigint NULL,
  "budget_max_amount" bigint NULL,
  "currency" character varying(3) NOT NULL DEFAULT 'BDT',
  "notes" character varying(500) NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("customer_id"),
  CONSTRAINT "customer_style_profiles_customer_id_fkey" FOREIGN KEY ("customer_id") REFERENCES "customers" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "customer_style_profiles_budget_max_nonnegative" CHECK ((budget_max_amount IS NULL) OR (budget_max_amount >= 0)),
  CONSTRAINT "customer_style_profiles_budget_min_nonnegative" CHECK ((budget_min_amount IS NULL) OR (budget_min_amount >= 0)),
  CONSTRAINT "customer_style_profiles_budget_range_valid" CHECK ((budget_min_amount IS NULL) OR (budget_max_amount IS NULL) OR (budget_max_amount >= budget_min_amount)),
  CONSTRAINT "customer_style_profiles_categories_array" CHECK (jsonb_typeof(preferred_categories) = 'array'::text),
  CONSTRAINT "customer_style_profiles_colors_array" CHECK (jsonb_typeof(preferred_colors) = 'array'::text),
  CONSTRAINT "customer_style_profiles_currency_valid" CHECK ((length((currency)::text) = 3) AND ((currency)::text = upper((currency)::text))),
  CONSTRAINT "customer_style_profiles_fit_not_blank" CHECK ((fit_preference IS NULL) OR (length(TRIM(BOTH FROM fit_preference)) > 0)),
  CONSTRAINT "customer_style_profiles_notes_not_blank" CHECK ((notes IS NULL) OR (length(TRIM(BOTH FROM notes)) > 0)),
  CONSTRAINT "customer_style_profiles_styles_array" CHECK (jsonb_typeof(preferred_styles) = 'array'::text)
);
