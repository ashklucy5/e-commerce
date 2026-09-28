schema "public" {}


table "categories" {
  schema = schema.public

  column "id" {
    type    = uuid
    null    = false
    default = sql("gen_random_uuid()")
  }

  column "parent_id" {
    type = uuid
    null = true
  }

  column "name" {
    type = varchar(120)
    null = false
  }

  column "slug" {
    type = varchar(140)
    null = false
  }

  column "description" {
    type = text
    null = true
  }

  column "image_url" {
    type = text
    null = true
  }

  column "icon_url" {
    type = text
    null = true
  }

  column "sort_order" {
    type    = integer
    null    = false
    default = 0
  }

  column "is_active" {
    type    = boolean
    null    = false
    default = true
  }

  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  column "updated_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  foreign_key "categories_parent_id_fkey" {
    columns     = [column.parent_id]
    ref_columns = [column.id]
    on_update   = NO_ACTION
    on_delete   = SET_NULL
  }

  index "categories_slug_key" {
    unique  = true
    columns = [column.slug]
  }

  index "idx_categories_parent_id" {
    columns = [column.parent_id]
  }

  index "idx_categories_active_sort" {
    columns = [
      column.is_active,
      column.sort_order,
    ]
  }

  check "categories_name_not_blank" {
    expr = "length(trim(name)) > 0"
  }

  check "categories_slug_not_blank" {
    expr = "length(trim(slug)) > 0"
  }

  check "categories_not_own_parent" {
    expr = "parent_id IS NULL OR parent_id <> id"
  }
}


table "products" {
  schema = schema.public

  column "id" {
    type    = uuid
    null    = false
    default = sql("gen_random_uuid()")
  }

  column "category_id" {
    type = uuid
    null = false
  }

  column "product_code" {
    type = varchar(100)
    null = false
  }

  column "name" {
    type = varchar(180)
    null = false
  }

  column "slug" {
    type = varchar(200)
    null = false
  }

  column "brand" {
    type = varchar(120)
    null = true
  }

  column "short_description" {
    type = varchar(500)
    null = true
  }

  column "description" {
    type = text
    null = true
  }

  column "status" {
    type    = varchar(20)
    null    = false
    default = "draft"
  }

  column "is_featured" {
    type    = boolean
    null    = false
    default = false
  }

  column "published_at" {
    type = timestamptz
    null = true
  }

  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  column "updated_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  foreign_key "products_category_id_fkey" {
    columns     = [column.category_id]
    ref_columns = [table.categories.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }

  index "products_product_code_key" {
    unique  = true
    columns = [column.product_code]
  }

  index "products_slug_key" {
    unique  = true
    columns = [column.slug]
  }

  index "idx_products_category_status" {
    columns = [
      column.category_id,
      column.status,
    ]
  }

  index "idx_products_featured_status" {
    columns = [
      column.is_featured,
      column.status,
    ]
  }

  check "products_product_code_not_blank" {
    expr = "length(trim(product_code)) > 0"
  }

  check "products_name_not_blank" {
    expr = "length(trim(name)) > 0"
  }

  check "products_slug_not_blank" {
    expr = "length(trim(slug)) > 0"
  }

  check "products_status_valid" {
    expr = "status IN ('draft', 'active', 'archived')"
  }
}


table "product_variants" {
  schema = schema.public

  column "id" {
    type    = uuid
    null    = false
    default = sql("gen_random_uuid()")
  }

  column "product_id" {
    type = uuid
    null = false
  }

  column "sku" {
    type = varchar(100)
    null = false
  }

  column "color_name" {
    type = varchar(80)
    null = true
  }

  column "color_hex" {
    type = varchar(7)
    null = true
  }

  column "size" {
    type = varchar(40)
    null = true
  }

  column "minimum_order_quantity" {
    type    = integer
    null    = false
    default = 1
  }

  column "order_increment" {
    type    = integer
    null    = false
    default = 1
  }

  column "price_amount" {
    type = bigint
    null = false
  }

  column "compare_at_price_amount" {
    type = bigint
    null = true
  }

  column "cost_amount" {
    type = bigint
    null = true
  }

  column "currency" {
    type    = varchar(3)
    null    = false
    default = "BDT"
  }

  column "barcode" {
    type = varchar(100)
    null = true
  }

  column "weight_grams" {
    type = integer
    null = true
  }

  column "is_active" {
    type    = boolean
    null    = false
    default = true
  }

  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  column "updated_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  foreign_key "product_variants_product_id_fkey" {
    columns     = [column.product_id]
    ref_columns = [table.products.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }

  index "product_variants_sku_key" {
    unique  = true
    columns = [column.sku]
  }

  index "product_variants_barcode_key" {
    unique  = true
    columns = [column.barcode]
  }

  index "product_variants_id_product_id_key" {
    unique = true
    columns = [
      column.id,
      column.product_id,
    ]
  }

  index "idx_product_variants_product_active" {
    columns = [
      column.product_id,
      column.is_active,
    ]
  }

  check "product_variants_sku_not_blank" {
    expr = "length(trim(sku)) > 0"
  }

  check "product_variants_minimum_order_quantity_positive" {
    expr = "minimum_order_quantity > 0"
  }

  check "product_variants_order_increment_valid" {
    expr = "order_increment = 1"
  }

  check "product_variants_price_nonnegative" {
    expr = "price_amount >= 0"
  }

  check "product_variants_compare_at_price_valid" {
    expr = "compare_at_price_amount IS NULL OR compare_at_price_amount >= price_amount"
  }

  check "product_variants_cost_nonnegative" {
    expr = "cost_amount IS NULL OR cost_amount >= 0"
  }

  check "product_variants_currency_valid" {
    expr = "length(currency) = 3 AND currency = upper(currency)"
  }

  check "product_variants_weight_nonnegative" {
    expr = "weight_grams IS NULL OR weight_grams >= 0"
  }

  check "product_variants_color_not_blank" {
    expr = "color_name IS NULL OR length(trim(color_name)) > 0"
  }

  check "product_variants_size_not_blank" {
    expr = "size IS NULL OR length(trim(size)) > 0"
  }
}


table "product_variant_price_tiers" {
  schema = schema.public

  column "id" {
    type    = uuid
    null    = false
    default = sql("gen_random_uuid()")
  }

  column "variant_id" {
    type = uuid
    null = false
  }

  column "min_quantity" {
    type = integer
    null = false
  }

  column "unit_price_amount" {
    type = bigint
    null = false
  }

  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  column "updated_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  foreign_key "product_variant_price_tiers_variant_id_fkey" {
    columns     = [column.variant_id]
    ref_columns = [table.product_variants.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }

  index "product_variant_price_tiers_variant_quantity_key" {
    unique = true
    columns = [
      column.variant_id,
      column.min_quantity,
    ]
  }

  check "product_variant_price_tiers_min_quantity_positive" {
    expr = "min_quantity > 0"
  }

  check "product_variant_price_tiers_price_nonnegative" {
    expr = "unit_price_amount >= 0"
  }
}


table "product_images" {
  schema = schema.public

  column "id" {
    type    = uuid
    null    = false
    default = sql("gen_random_uuid()")
  }

  column "product_id" {
    type = uuid
    null = false
  }

  column "variant_id" {
    type = uuid
    null = true
  }

  column "url" {
    type = text
    null = false
  }

  column "alt_text" {
    type = varchar(255)
    null = true
  }

  column "sort_order" {
    type    = integer
    null    = false
    default = 0
  }

  column "is_primary" {
    type    = boolean
    null    = false
    default = false
  }

  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  foreign_key "product_images_product_id_fkey" {
    columns     = [column.product_id]
    ref_columns = [table.products.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }

  foreign_key "product_images_variant_product_fkey" {
    columns = [
      column.variant_id,
      column.product_id,
    ]

    ref_columns = [
      table.product_variants.column.id,
      table.product_variants.column.product_id,
    ]

    on_update = NO_ACTION
    on_delete = CASCADE
  }
  index "idx_product_images_product_sort" {
    columns = [
      column.product_id,
      column.sort_order,
    ]
  }

  index "idx_product_images_variant" {
    columns = [column.variant_id]
  }

  index "product_images_one_primary_per_product" {
    unique  = true
    columns = [column.product_id]
    where   = "is_primary = true"
  }

  check "product_images_url_not_blank" {
    expr = "length(trim(url)) > 0"
  }

  check "product_images_sort_order_nonnegative" {
    expr = "sort_order >= 0"
  }
}


table "product_360_frames" {
  schema = schema.public

  column "id" {
    type    = uuid
    null    = false
    default = sql("gen_random_uuid()")
  }

  column "product_id" {
    type = uuid
    null = false
  }

  column "variant_id" {
    type = uuid
    null = true
  }

  column "storage_key" {
    type = text
    null = false
  }

  column "url" {
    type = text
    null = false
  }

  column "frame_index" {
    type = integer
    null = false
  }

  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  foreign_key "product_360_frames_product_id_fkey" {
    columns     = [column.product_id]
    ref_columns = [table.products.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }

  foreign_key "product_360_frames_variant_product_fkey" {
    columns = [
      column.variant_id,
      column.product_id,
    ]

    ref_columns = [
      table.product_variants.column.id,
      table.product_variants.column.product_id,
    ]

    on_update = NO_ACTION
    on_delete = CASCADE
  }

  index "product_360_frames_storage_key_key" {
    unique  = true
    columns = [column.storage_key]
  }

  index "product_360_frames_product_frame_key" {
    unique = true

    columns = [
      column.product_id,
      column.frame_index,
    ]

    where = "variant_id IS NULL"
  }

  index "product_360_frames_variant_frame_key" {
    unique = true

    columns = [
      column.product_id,
      column.variant_id,
      column.frame_index,
    ]

    where = "variant_id IS NOT NULL"
  }

  index "idx_product_360_frames_product" {
    columns = [
      column.product_id,
      column.frame_index,
    ]
  }

  index "idx_product_360_frames_variant" {
    columns = [
      column.variant_id,
      column.frame_index,
    ]
  }

  check "product_360_frames_storage_key_not_blank" {
    expr = "length(trim(storage_key)) > 0"
  }

  check "product_360_frames_url_not_blank" {
    expr = "length(trim(url)) > 0"
  }

  check "product_360_frames_frame_index_nonnegative" {
    expr = "frame_index >= 0"
  }
}


table "product_3d_models" {
  schema = schema.public

  column "id" {
    type    = uuid
    null    = false
    default = sql("gen_random_uuid()")
  }

  column "product_id" {
    type = uuid
    null = false
  }

  column "variant_id" {
    type = uuid
    null = true
  }

  column "storage_key" {
    type = text
    null = false
  }

  column "url" {
    type = text
    null = false
  }

  column "poster_url" {
    type = text
    null = true
  }

  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  column "updated_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  foreign_key "product_3d_models_product_id_fkey" {
    columns     = [column.product_id]
    ref_columns = [table.products.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }

  foreign_key "product_3d_models_variant_product_fkey" {
    columns = [
      column.variant_id,
      column.product_id,
    ]

    ref_columns = [
      table.product_variants.column.id,
      table.product_variants.column.product_id,
    ]

    on_update = NO_ACTION
    on_delete = CASCADE
  }

  index "product_3d_models_storage_key_key" {
    unique  = true
    columns = [column.storage_key]
  }

  index "product_3d_models_product_key" {
    unique  = true
    columns = [column.product_id]
    where   = "variant_id IS NULL"
  }

  index "product_3d_models_variant_key" {
    unique = true

    columns = [
      column.product_id,
      column.variant_id,
    ]

    where = "variant_id IS NOT NULL"
  }

  index "idx_product_3d_models_variant" {
    columns = [column.variant_id]
  }

  check "product_3d_models_storage_key_not_blank" {
    expr = "length(trim(storage_key)) > 0"
  }

  check "product_3d_models_url_not_blank" {
    expr = "length(trim(url)) > 0"
  }

  check "product_3d_models_poster_url_not_blank" {
    expr = "poster_url IS NULL OR length(trim(poster_url)) > 0"
  }
}


table "inventory" {
  schema = schema.public

  column "variant_id" {
    type = uuid
    null = false
  }

  column "quantity_on_hand" {
    type    = integer
    null    = false
    default = 0
  }

  column "quantity_reserved" {
    type    = integer
    null    = false
    default = 0
  }

  column "reorder_level" {
    type    = integer
    null    = false
    default = 0
  }

  column "updated_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  primary_key {
    columns = [column.variant_id]
  }

  foreign_key "inventory_variant_id_fkey" {
    columns     = [column.variant_id]
    ref_columns = [table.product_variants.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }

  check "inventory_quantity_on_hand_nonnegative" {
    expr = "quantity_on_hand >= 0"
  }

  check "inventory_quantity_reserved_nonnegative" {
    expr = "quantity_reserved >= 0"
  }

  check "inventory_reserved_not_above_on_hand" {
    expr = "quantity_reserved <= quantity_on_hand"
  }

  check "inventory_reorder_level_nonnegative" {
    expr = "reorder_level >= 0"
  }
}

table "product_code_namespaces" {
  schema = schema.public

  column "category_id" {
    type = uuid
    null = false
  }

  column "prefix" {
    type = varchar(7)
    null = false
  }

  column "next_number" {
    type    = bigint
    null    = false
    default = 1
  }

  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  column "updated_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  primary_key {
    columns = [
      column.category_id,
    ]
  }

  foreign_key "product_code_namespaces_category_id_fkey" {
    columns = [
      column.category_id,
    ]

    ref_columns = [
      table.categories.column.id,
    ]

    on_update = NO_ACTION
    on_delete = RESTRICT
  }

  index "product_code_namespaces_prefix_key" {
    unique = true

    columns = [
      column.prefix,
    ]
  }

  check "product_code_namespaces_prefix_format" {
    expr = "prefix ~ '^[A-Z0-9]{3}-[A-Z0-9]{3}$'"
  }

  check "product_code_namespaces_next_number_positive" {
    expr = "next_number > 0"
  }

  check "product_code_namespaces_next_number_limit" {
    expr = "next_number <= 1000000"
  }
}

table "catalog_import_batches" {
  schema = schema.public

  column "id" {
    type    = uuid
    null    = false
    default = sql("gen_random_uuid()")
  }

  column "source_filename" {
    type = varchar(255)
    null = false
  }

  column "file_storage_key" {
    type = text
    null = false
  }

  column "image_archive_storage_key" {
    type = text
    null = true
  }

  column "file_checksum_sha256" {
    type = varchar(64)
    null = true
  }

  column "status" {
    type    = varchar(30)
    null    = false
    default = "uploaded"
  }

  column "total_rows" {
    type    = integer
    null    = false
    default = 0
  }

  column "valid_rows" {
    type    = integer
    null    = false
    default = 0
  }

  column "failed_rows" {
    type    = integer
    null    = false
    default = 0
  }

  column "created_products" {
    type    = integer
    null    = false
    default = 0
  }

  column "updated_products" {
    type    = integer
    null    = false
    default = 0
  }

  column "created_variants" {
    type    = integer
    null    = false
    default = 0
  }

  column "updated_variants" {
    type    = integer
    null    = false
    default = 0
  }

  column "created_categories" {
    type    = integer
    null    = false
    default = 0
  }

  column "last_error" {
    type = text
    null = true
  }

  column "started_at" {
    type = timestamptz
    null = true
  }

  column "completed_at" {
    type = timestamptz
    null = true
  }

  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  column "updated_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  index "idx_catalog_import_batches_status_created" {
    columns = [
      column.status,
      column.created_at,
    ]
  }

  check "catalog_import_batches_status_valid" {
    expr = "status IN ('uploaded', 'parsing', 'validating', 'ready', 'applying', 'completed', 'failed')"
  }

  check "catalog_import_batches_counts_nonnegative" {
    expr = "total_rows >= 0 AND valid_rows >= 0 AND failed_rows >= 0 AND created_products >= 0 AND updated_products >= 0 AND created_variants >= 0 AND updated_variants >= 0 AND created_categories >= 0"
  }

  check "catalog_import_batches_filename_not_blank" {
    expr = "length(trim(source_filename)) > 0"
  }

  check "catalog_import_batches_storage_key_not_blank" {
    expr = "length(trim(file_storage_key)) > 0"
  }
}


table "catalog_import_rows" {
  schema = schema.public

  column "id" {
    type    = uuid
    null    = false
    default = sql("gen_random_uuid()")
  }

  column "batch_id" {
    type = uuid
    null = false
  }
  column "sheet_name" {
    type = varchar(100)
    null = false
  }

  column "row_number" {
    type = integer
    null = false
  }

  column "product_code" {
    type = varchar(100)
    null = true
  }

  column "sku" {
    type = varchar(100)
    null = true
  }

  column "action" {
    type = varchar(20)
    null = true
  }

  column "status" {
    type    = varchar(20)
    null    = false
    default = "pending"
  }

  column "raw_data" {
    type = jsonb
    null = false
  }

  column "error_details" {
    type = jsonb
    null = true
  }

  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  column "updated_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  foreign_key "catalog_import_rows_batch_id_fkey" {
    columns     = [column.batch_id]
    ref_columns = [table.catalog_import_batches.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }

  index "catalog_import_rows_batch_sheet_row_key" {
    unique = true
    columns = [
      column.batch_id,
      column.sheet_name,
      column.row_number,
    ]
  }

  index "idx_catalog_import_rows_batch_status" {
    columns = [
      column.batch_id,
      column.status,
    ]
  }

  check "catalog_import_rows_row_number_positive" {
    expr = "row_number > 0"
  }

  check "catalog_import_rows_sheet_name_not_blank" {
    expr = "length(trim(sheet_name)) > 0"
  }

  check "catalog_import_rows_status_valid" {
    expr = "status IN ('pending', 'valid', 'invalid', 'applied', 'skipped', 'failed')"
  }

  check "catalog_import_rows_action_valid" {
    expr = "action IS NULL OR action IN ('create', 'update', 'skip')"
  }
}
table "inventory_reservations" {
  schema = schema.public

  column "id" {
    type    = uuid
    null    = false
    default = sql("gen_random_uuid()")
  }

  column "variant_id" {
    type = uuid
    null = false
  }

  column "reference_type" {
    type = varchar(50)
    null = false
  }

  column "reference_id" {
    type = varchar(160)
    null = false
  }

  column "quantity" {
    type = integer
    null = false
  }

  column "status" {
    type    = varchar(20)
    null    = false
    default = "active"
  }

  column "expires_at" {
    type = timestamptz
    null = true
  }

  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  column "updated_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  foreign_key "inventory_reservations_variant_id_fkey" {
    columns     = [column.variant_id]
    ref_columns = [table.product_variants.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }

  index "inventory_reservations_reference_variant_key" {
    unique = true
    columns = [
      column.reference_type,
      column.reference_id,
      column.variant_id,
    ]
  }

  index "idx_inventory_reservations_variant_status" {
    columns = [
      column.variant_id,
      column.status,
    ]
  }

  index "idx_inventory_reservations_active_expiry" {
    columns = [
      column.status,
      column.expires_at,
    ]

    where = "status = 'active'"
  }

  index "idx_inventory_reservations_reference" {
    columns = [
      column.reference_type,
      column.reference_id,
    ]
  }

  check "inventory_reservations_reference_type_not_blank" {
    expr = "length(trim(reference_type)) > 0"
  }

  check "inventory_reservations_reference_id_not_blank" {
    expr = "length(trim(reference_id)) > 0"
  }

  check "inventory_reservations_quantity_positive" {
    expr = "quantity > 0"
  }

  check "inventory_reservations_status_valid" {
    expr = "status IN ('active', 'released', 'committed', 'expired')"
  }
}


table "inventory_movements" {
  schema = schema.public

  column "id" {
    type    = uuid
    null    = false
    default = sql("gen_random_uuid()")
  }

  column "variant_id" {
    type = uuid
    null = false
  }

  column "reservation_id" {
    type = uuid
    null = true
  }

  column "movement_type" {
    type = varchar(30)
    null = false
  }

  column "quantity_on_hand_delta" {
    type    = integer
    null    = false
    default = 0
  }

  column "quantity_reserved_delta" {
    type    = integer
    null    = false
    default = 0
  }

  column "quantity_on_hand_after" {
    type = integer
    null = false
  }

  column "quantity_reserved_after" {
    type = integer
    null = false
  }

  column "reference_type" {
    type = varchar(50)
    null = true
  }

  column "reference_id" {
    type = varchar(160)
    null = true
  }

  column "reason" {
    type = varchar(120)
    null = true
  }

  column "note" {
    type = text
    null = true
  }

  column "actor_type" {
    type = varchar(30)
    null = true
  }

  column "actor_id" {
    type = varchar(160)
    null = true
  }

  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  foreign_key "inventory_movements_variant_id_fkey" {
    columns     = [column.variant_id]
    ref_columns = [table.product_variants.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }

  foreign_key "inventory_movements_reservation_id_fkey" {
    columns     = [column.reservation_id]
    ref_columns = [table.inventory_reservations.column.id]
    on_update   = NO_ACTION
    on_delete   = SET_NULL
  }

  index "idx_inventory_movements_variant_created" {
    columns = [
      column.variant_id,
      column.created_at,
    ]
  }

  index "idx_inventory_movements_reservation" {
    columns = [column.reservation_id]
  }

  index "idx_inventory_movements_reference" {
    columns = [
      column.reference_type,
      column.reference_id,
    ]
  }

  check "inventory_movements_type_valid" {
    expr = "movement_type IN ('adjustment', 'import_sync', 'reserve', 'reservation_change', 'release', 'commit', 'expire')"
  }

  check "inventory_movements_has_delta" {
    expr = "quantity_on_hand_delta <> 0 OR quantity_reserved_delta <> 0"
  }

  check "inventory_movements_on_hand_after_nonnegative" {
    expr = "quantity_on_hand_after >= 0"
  }

  check "inventory_movements_reserved_after_nonnegative" {
    expr = "quantity_reserved_after >= 0"
  }

  check "inventory_movements_reserved_after_not_above_on_hand" {
    expr = "quantity_reserved_after <= quantity_on_hand_after"
  }

  check "inventory_movements_reference_pair" {
    expr = "(reference_type IS NULL AND reference_id IS NULL) OR (reference_type IS NOT NULL AND reference_id IS NOT NULL)"
  }

  check "inventory_movements_actor_pair" {
    expr = "(actor_type IS NULL AND actor_id IS NULL) OR (actor_type IS NOT NULL AND actor_id IS NOT NULL)"
  }
}
table "carts" {
  schema = schema.public

  column "id" {
    type    = uuid
    null    = false
    default = sql("gen_random_uuid()")
  }

  column "cart_key" {
    type = varchar(160)
    null = false
  }

  column "status" {
    type    = varchar(20)
    null    = false
    default = "active"
  }

  column "currency" {
    type    = varchar(3)
    null    = false
    default = "BDT"
  }

  column "expires_at" {
    type = timestamptz
    null = true
  }

  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  column "updated_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  index "carts_cart_key_key" {
    unique  = true
    columns = [column.cart_key]
  }

  index "idx_carts_status_updated" {
    columns = [
      column.status,
      column.updated_at,
    ]
  }

  index "idx_carts_active_expiry" {
    columns = [
      column.status,
      column.expires_at,
    ]

    where = "status = 'active'"
  }

  check "carts_cart_key_not_blank" {
    expr = "length(trim(cart_key)) > 0"
  }

  check "carts_status_valid" {
    expr = "status IN ('active', 'converted', 'abandoned', 'expired')"
  }

  check "carts_currency_valid" {
    expr = "length(currency) = 3 AND currency = upper(currency)"
  }
}
table "cart_items" {
  schema = schema.public

  column "id" {
    type    = uuid
    null    = false
    default = sql("gen_random_uuid()")
  }

  column "cart_id" {
    type = uuid
    null = false
  }

  column "variant_id" {
    type = uuid
    null = false
  }

  column "quantity" {
    type = integer
    null = false
  }

  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  column "updated_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  foreign_key "cart_items_cart_id_fkey" {
    columns     = [column.cart_id]
    ref_columns = [table.carts.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }

  foreign_key "cart_items_variant_id_fkey" {
    columns     = [column.variant_id]
    ref_columns = [table.product_variants.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }

  index "cart_items_cart_variant_key" {
    unique = true
    columns = [
      column.cart_id,
      column.variant_id,
    ]
  }

  index "idx_cart_items_cart" {
    columns = [column.cart_id]
  }

  index "idx_cart_items_variant" {
    columns = [column.variant_id]
  }

  check "cart_items_quantity_positive" {
    expr = "quantity > 0"
  }
}

table "promotions" {
  schema = schema.public

  column "id" {
    type    = uuid
    null    = false
    default = sql("gen_random_uuid()")
  }

  column "name" {
    type = varchar(160)
    null = false
  }

  column "code" {
    type = varchar(64)
    null = true
  }

  column "discount_type" {
    type = varchar(20)
    null = false
  }

  column "percentage_bps" {
    type = integer
    null = true
  }

  column "fixed_amount" {
    type = bigint
    null = true
  }

  column "minimum_subtotal_amount" {
    type    = bigint
    null    = false
    default = 0
  }

  column "maximum_discount_amount" {
    type = bigint
    null = true
  }

  column "currency" {
    type    = varchar(3)
    null    = false
    default = "BDT"
  }

  column "status" {
    type    = varchar(20)
    null    = false
    default = "draft"
  }

  column "starts_at" {
    type = timestamptz
    null = true
  }

  column "ends_at" {
    type = timestamptz
    null = true
  }

  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  column "updated_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  column "scope" {
    type    = varchar(20)
    null    = false
    default = "order"
  }

  column "campaign_type" {
    type    = varchar(20)
    null    = false
    default = "standard"
  }

  primary_key {
    columns = [column.id]
  }

  index "promotions_code_key" {
    unique  = true
    columns = [column.code]
  }

  index "idx_promotions_status_window" {
    columns = [
      column.status,
      column.starts_at,
      column.ends_at,
    ]
  }

  index "idx_promotions_scope_campaign_status_window" {
    columns = [
      column.scope,
      column.campaign_type,
      column.status,
      column.starts_at,
      column.ends_at,
    ]
  }

  check "promotions_name_not_blank" {
    expr = "length(trim(name)) > 0"
  }

  check "promotions_code_valid" {
    expr = "code IS NULL OR (code = trim(code) AND length(code) > 0 AND code = upper(code))"
  }

  check "promotions_scope_valid" {
    expr = "scope IN ('order', 'product')"
  }

  check "promotions_campaign_type_valid" {
    expr = "campaign_type IN ('standard', 'flash_sale')"
  }

  check "promotions_campaign_scope_valid" {
    expr = "campaign_type <> 'flash_sale' OR scope = 'product'"
  }

  check "promotions_flash_sale_configuration_valid" {
    expr = "campaign_type <> 'flash_sale' OR (code IS NULL AND starts_at IS NOT NULL AND ends_at IS NOT NULL AND minimum_subtotal_amount = 0 AND maximum_discount_amount IS NULL)"
  }
  check "promotions_discount_type_valid" {
    expr = "discount_type IN ('percentage', 'fixed')"
  }

  check "promotions_discount_configuration_valid" {
    expr = "(discount_type = 'percentage' AND percentage_bps IS NOT NULL AND percentage_bps BETWEEN 1 AND 10000 AND fixed_amount IS NULL) OR (discount_type = 'fixed' AND fixed_amount IS NOT NULL AND fixed_amount > 0 AND percentage_bps IS NULL)"
  }

  check "promotions_minimum_subtotal_nonnegative" {
    expr = "minimum_subtotal_amount >= 0"
  }

  check "promotions_maximum_discount_positive" {
    expr = "maximum_discount_amount IS NULL OR maximum_discount_amount > 0"
  }

  check "promotions_currency_valid" {
    expr = "length(currency) = 3 AND currency = upper(currency)"
  }

  check "promotions_status_valid" {
    expr = "status IN ('draft', 'active', 'disabled')"
  }

  check "promotions_window_valid" {
    expr = "starts_at IS NULL OR ends_at IS NULL OR ends_at > starts_at"
  }
}


table "promotion_targets" {
  schema = schema.public

  column "id" {
    type    = uuid
    null    = false
    default = sql("gen_random_uuid()")
  }

  column "promotion_id" {
    type = uuid
    null = false
  }

  column "product_id" {
    type = uuid
    null = true
  }

  column "variant_id" {
    type = uuid
    null = true
  }

  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  foreign_key "promotion_targets_promotion_id_fkey" {
    columns     = [column.promotion_id]
    ref_columns = [table.promotions.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }

  foreign_key "promotion_targets_product_id_fkey" {
    columns     = [column.product_id]
    ref_columns = [table.products.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }

  foreign_key "promotion_targets_variant_id_fkey" {
    columns     = [column.variant_id]
    ref_columns = [table.product_variants.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }

  index "promotion_targets_promotion_product_key" {
    unique = true

    columns = [
      column.promotion_id,
      column.product_id,
    ]

    where = "product_id IS NOT NULL"
  }

  index "promotion_targets_promotion_variant_key" {
    unique = true

    columns = [
      column.promotion_id,
      column.variant_id,
    ]

    where = "variant_id IS NOT NULL"
  }

  index "idx_promotion_targets_product" {
    columns = [column.product_id]
    where   = "product_id IS NOT NULL"
  }

  index "idx_promotion_targets_variant" {
    columns = [column.variant_id]
    where   = "variant_id IS NOT NULL"
  }

  check "promotion_targets_exactly_one_target" {
    expr = "(product_id IS NOT NULL AND variant_id IS NULL) OR (product_id IS NULL AND variant_id IS NOT NULL)"
  }
}

table "checkout_sessions" {
  schema = schema.public

  column "id" {
    type    = uuid
    null    = false
    default = sql("gen_random_uuid()")
  }

  column "customer_id" {
    type = uuid
    null = true
  }
  column "checkout_key" {
    type = varchar(160)
    null = false
  }

  column "cart_id" {
    type = uuid
    null = false
  }

  column "status" {
    type    = varchar(20)
    null    = false
    default = "active"
  }

  column "currency" {
    type    = varchar(3)
    null    = false
    default = "BDT"
  }

  column "subtotal_amount" {
    type    = bigint
    null    = false
    default = 0
  }

  column "discount_amount" {
    type    = bigint
    null    = false
    default = 0
  }
  column "promotion_id" {
    type = uuid
    null = true
  }

  column "promotion_code" {
    type = varchar(64)
    null = true
  }
  column "shipping_amount" {
    type    = bigint
    null    = false
    default = 0
  }

  column "total_amount" {
    type    = bigint
    null    = false
    default = 0
  }

  column "customer_name" {
    type = varchar(160)
    null = true
  }

  column "customer_phone" {
    type = varchar(40)
    null = true
  }

  column "customer_email" {
    type = varchar(255)
    null = true
  }

  column "shipping_address_line1" {
    type = varchar(255)
    null = true
  }

  column "shipping_address_line2" {
    type = varchar(255)
    null = true
  }

  column "shipping_city" {
    type = varchar(120)
    null = true
  }

  column "shipping_area" {
    type = varchar(120)
    null = true
  }

  column "shipping_postal_code" {
    type = varchar(30)
    null = true
  }

  column "delivery_method" {
    type = varchar(80)
    null = true
  }

  column "payment_method" {
    type = varchar(80)
    null = true
  }

  column "expires_at" {
    type = timestamptz
    null = false
  }

  column "completed_at" {
    type = timestamptz
    null = true
  }

  column "cancelled_at" {
    type = timestamptz
    null = true
  }

  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  column "updated_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  foreign_key "checkout_sessions_cart_id_fkey" {
    columns     = [column.cart_id]
    ref_columns = [table.carts.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }

  foreign_key "checkout_sessions_customer_id_fkey" {
  columns     = [column.customer_id]
  ref_columns = [table.customers.column.id]
  on_update   = NO_ACTION
  on_delete   = SET_NULL
}
  foreign_key "checkout_sessions_promotion_id_fkey" {
    columns     = [column.promotion_id]
    ref_columns = [table.promotions.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }

  index "idx_checkout_sessions_promotion" {
    columns = [column.promotion_id]
  }

  check "checkout_sessions_promotion_code_valid" {
     expr = "promotion_code IS NULL OR (promotion_id IS NOT NULL AND promotion_code = trim(promotion_code) AND length(promotion_code) > 0 AND promotion_code = upper(promotion_code))"
}

  index "checkout_sessions_checkout_key_key" {
    unique  = true
    columns = [column.checkout_key]
  }

  index "idx_checkout_sessions_customer_created" {
  columns = [
    column.customer_id,
    column.created_at,
  ]
}

  index "idx_checkout_sessions_cart" {
    columns = [column.cart_id]
  }

  index "idx_checkout_sessions_status_expiry" {
    columns = [
      column.status,
      column.expires_at,
    ]
  }

  index "checkout_sessions_one_active_per_cart" {
    unique  = true
    columns = [column.cart_id]
    where   = "status = 'active'"
  }

  check "checkout_sessions_checkout_key_not_blank" {
    expr = "length(trim(checkout_key)) > 0"
  }

  check "checkout_sessions_status_valid" {
    expr = "status IN ('active', 'completed', 'cancelled', 'expired')"
  }

  check "checkout_sessions_currency_valid" {
    expr = "length(currency) = 3 AND currency = upper(currency)"
  }

  check "checkout_sessions_subtotal_nonnegative" {
    expr = "subtotal_amount >= 0"
  }

  check "checkout_sessions_discount_nonnegative" {
    expr = "discount_amount >= 0"
  }

  check "checkout_sessions_shipping_nonnegative" {
    expr = "shipping_amount >= 0"
  }

  check "checkout_sessions_total_nonnegative" {
    expr = "total_amount >= 0"
  }

  check "checkout_sessions_total_consistent" {
    expr = "total_amount = subtotal_amount - discount_amount + shipping_amount"
  }

  check "checkout_sessions_discount_not_above_subtotal" {
    expr = "discount_amount <= subtotal_amount"
  }
}
table "checkout_items" {
  schema = schema.public

  column "id" {
    type    = uuid
    null    = false
    default = sql("gen_random_uuid()")
  }

  column "checkout_id" {
    type = uuid
    null = false
  }

  column "variant_id" {
    type = uuid
    null = false
  }

  column "sku" {
    type = varchar(100)
    null = false
  }

  column "product_name" {
    type = varchar(180)
    null = false
  }

  column "quantity" {
    type = integer
    null = false
  }

  column "minimum_order_quantity" {
    type = integer
    null = false
  }

  column "unit_price_amount" {
    type = bigint
    null = false
  }

  column "line_total_amount" {
    type = bigint
    null = false
  }

  column "currency" {
    type = varchar(3)
    null = false
  }

  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  foreign_key "checkout_items_checkout_id_fkey" {
    columns     = [column.checkout_id]
    ref_columns = [table.checkout_sessions.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }

  foreign_key "checkout_items_variant_id_fkey" {
    columns     = [column.variant_id]
    ref_columns = [table.product_variants.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }

  index "checkout_items_checkout_variant_key" {
    unique = true
    columns = [
      column.checkout_id,
      column.variant_id,
    ]
  }

  index "idx_checkout_items_variant" {
    columns = [column.variant_id]
  }

  check "checkout_items_sku_not_blank" {
    expr = "length(trim(sku)) > 0"
  }

  check "checkout_items_product_name_not_blank" {
    expr = "length(trim(product_name)) > 0"
  }

  check "checkout_items_quantity_positive" {
    expr = "quantity > 0"
  }

  check "checkout_items_minimum_order_quantity_positive" {
    expr = "minimum_order_quantity > 0"
  }

  check "checkout_items_quantity_meets_minimum" {
    expr = "quantity >= minimum_order_quantity"
  }

  check "checkout_items_unit_price_nonnegative" {
    expr = "unit_price_amount >= 0"
  }

  check "checkout_items_line_total_nonnegative" {
    expr = "line_total_amount >= 0"
  }

  check "checkout_items_line_total_consistent" {
    expr = "line_total_amount = unit_price_amount * quantity"
  }

  check "checkout_items_currency_valid" {
    expr = "length(currency) = 3 AND currency = upper(currency)"
  }
}
table "orders" {
  schema = schema.public

  column "id" {
    type    = uuid
    null    = false
    default = sql("gen_random_uuid()")
  }

  column "customer_id" {
    type = uuid
    null = true
}

  column "order_number" {
    type = varchar(50)
    null = false
  }

  column "order_type" {
    type    = varchar(20)
    null    = false
    default = "standard"
  }

  column "checkout_id" {
    type = uuid
    null = true
  }

  column "cart_id" {
    type = uuid
    null = true
  }

  column "status" {
    type    = varchar(30)
    null    = false
    default = "pending_payment"
  }

  column "payment_status" {
    type    = varchar(30)
    null    = false
    default = "pending"
  }

  column "payment_method" {
    type = varchar(80)
    null = false
  }

  column "currency" {
    type = varchar(3)
    null = false
  }

  column "subtotal_amount" {
    type = bigint
    null = false
  }

  column "discount_amount" {
    type    = bigint
    null    = false
    default = 0
  }

    column "promotion_id" {
    type = uuid
    null = true
  }

  column "promotion_code" {
    type = varchar(64)
    null = true
  }

  column "shipping_amount" {
    type    = bigint
    null    = false
    default = 0
  }

  column "total_amount" {
    type = bigint
    null = false
  }

  column "customer_name" {
    type = varchar(160)
    null = false
  }

  column "customer_phone" {
    type = varchar(40)
    null = false
  }

  column "customer_email" {
    type = varchar(255)
    null = true
  }

  column "shipping_address_line1" {
    type = varchar(255)
    null = false
  }

  column "shipping_address_line2" {
    type = varchar(255)
    null = true
  }

  column "shipping_city" {
    type = varchar(120)
    null = false
  }

  column "shipping_area" {
    type = varchar(120)
    null = false
  }

  column "shipping_postal_code" {
    type = varchar(30)
    null = true
  }

  column "delivery_method" {
    type = varchar(80)
    null = false
  }

  column "payment_due_at" {
    type = timestamptz
    null = true
  }

  column "paid_at" {
    type = timestamptz
    null = true
  }

  column "confirmed_at" {
    type = timestamptz
    null = true
  }
column "processing_at" {
  type = timestamptz
  null = true
}

column "shipped_at" {
  type = timestamptz
  null = true
}

column "delivered_at" {
  type = timestamptz
  null = true
}

column "completed_at" {
  type = timestamptz
  null = true
}
  column "cancelled_at" {
    type = timestamptz
    null = true
  }
    column "cancellation_reason" {
    type = varchar(500)
    null = true
  }

  column "cancelled_by" {
    type = varchar(100)
    null = true
  }

  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  column "updated_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  foreign_key "orders_checkout_id_fkey" {
    columns     = [column.checkout_id]
    ref_columns = [table.checkout_sessions.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }

  foreign_key "orders_customer_id_fkey" {
  columns     = [column.customer_id]
  ref_columns = [table.customers.column.id]
  on_update   = NO_ACTION
  on_delete   = SET_NULL
}

  foreign_key "orders_cart_id_fkey" {
    columns     = [column.cart_id]
    ref_columns = [table.carts.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }

    foreign_key "orders_promotion_id_fkey" {
    columns     = [column.promotion_id]
    ref_columns = [table.promotions.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }

  index "idx_orders_promotion" {
    columns = [column.promotion_id]
  }

  check "orders_promotion_code_valid" {
     expr = "promotion_code IS NULL OR (promotion_id IS NOT NULL AND promotion_code = trim(promotion_code) AND length(promotion_code) > 0 AND promotion_code = upper(promotion_code))"
  }

  index "orders_order_number_key" {
    unique  = true
    columns = [column.order_number]
  }

  index "idx_orders_type_created" {
    columns = [
      column.order_type,
      column.created_at,
    ]
  }

  index "orders_checkout_id_key" {
    unique  = true
    columns = [column.checkout_id]
  }

  index "idx_orders_status_created" {
    columns = [
      column.status,
      column.created_at,
    ]
  }

  index "idx_orders_payment_status_created" {
    columns = [
      column.payment_status,
      column.created_at,
    ]
  }

  index "idx_orders_customer_created" {
  columns = [
    column.customer_id,
    column.created_at,
  ]
}

  index "idx_orders_pending_payment_due" {
    columns = [
      column.payment_due_at,
    ]

    where = "status = 'pending_payment'"
  }

  index "idx_orders_customer_phone_created" {
    columns = [
      column.customer_phone,
      column.created_at,
    ]
  }

  check "orders_order_number_not_blank" {
    expr = "length(trim(order_number)) > 0"
  }

  check "orders_status_valid" {
    expr = "status IN ('pending_payment', 'awaiting_procurement', 'confirmed', 'processing', 'shipped', 'delivered', 'completed', 'payment_expired', 'cancelled')"
  }

  check "orders_type_valid" {
    expr = "order_type IN ('standard', 'sourcing')"
  }

  check "orders_origin_valid" {
    expr = "(order_type = 'standard' AND checkout_id IS NOT NULL AND cart_id IS NOT NULL) OR (order_type = 'sourcing' AND checkout_id IS NULL AND cart_id IS NULL AND customer_id IS NOT NULL)"
}

  check "orders_payment_status_valid" {
    expr = "payment_status IN ('pending', 'paid', 'failed', 'expired', 'cod_pending', 'cod_collected', 'refunded')"
  }

  check "orders_payment_method_not_blank" {
    expr = "length(trim(payment_method)) > 0"
  }

  check "orders_currency_valid" {
    expr = "length(currency) = 3 AND currency = upper(currency)"
  }

  check "orders_subtotal_nonnegative" {
    expr = "subtotal_amount >= 0"
  }

  check "orders_discount_nonnegative" {
    expr = "discount_amount >= 0"
  }

  check "orders_shipping_nonnegative" {
    expr = "shipping_amount >= 0"
  }

  check "orders_discount_not_above_subtotal" {
    expr = "discount_amount <= subtotal_amount"
  }

  check "orders_total_nonnegative" {
    expr = "total_amount >= 0"
  }

  check "orders_total_consistent" {
    expr = "total_amount = subtotal_amount - discount_amount + shipping_amount"
  }

  check "orders_customer_name_not_blank" {
    expr = "length(trim(customer_name)) > 0"
  }

  check "orders_customer_phone_not_blank" {
    expr = "length(trim(customer_phone)) > 0"
  }

  check "orders_shipping_address_line1_not_blank" {
    expr = "length(trim(shipping_address_line1)) > 0"
  }

  check "orders_shipping_city_not_blank" {
    expr = "length(trim(shipping_city)) > 0"
  }

  check "orders_shipping_area_not_blank" {
    expr = "length(trim(shipping_area)) > 0"
  }

  check "orders_delivery_method_not_blank" {
    expr = "length(trim(delivery_method)) > 0"
  }

  check "orders_pending_payment_has_deadline" {
    expr = "status <> 'pending_payment' OR payment_due_at IS NOT NULL"
  }
}


table "order_items" {
  schema = schema.public

  column "id" {
    type    = uuid
    null    = false
    default = sql("gen_random_uuid()")
  }

  column "order_id" {
    type = uuid
    null = false
  }

  column "variant_id" {
    type = uuid
    null = false
  }

  column "sku" {
    type = varchar(100)
    null = false
  }

  column "product_name" {
    type = varchar(180)
    null = false
  }

  column "quantity" {
    type = integer
    null = false
  }

  column "minimum_order_quantity" {
    type = integer
    null = false
  }

  column "unit_price_amount" {
    type = bigint
    null = false
  }

  column "line_total_amount" {
    type = bigint
    null = false
  }

  column "currency" {
    type = varchar(3)
    null = false
  }

  # Immutable cost snapshot captured when a NEW order is placed.
  #
  # Existing historical orders deliberately remain NULL because using
  # today's variant cost would create false historical profit figures.
  column "unit_cost_amount" {
    type = bigint
    null = true
  }

  column "line_cost_amount" {
    type = bigint
    null = true
  }

  column "cost_currency" {
    type = varchar(3)
    null = true
  }

  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  foreign_key "order_items_order_id_fkey" {
    columns     = [column.order_id]
    ref_columns = [table.orders.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }

  foreign_key "order_items_variant_id_fkey" {
    columns     = [column.variant_id]
    ref_columns = [table.product_variants.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }

  index "order_items_order_variant_key" {
    unique = true

    columns = [
      column.order_id,
      column.variant_id,
    ]
  }

  index "idx_order_items_variant" {
    columns = [column.variant_id]
  }

  check "order_items_sku_not_blank" {
    expr = "length(trim(sku)) > 0"
  }

  check "order_items_product_name_not_blank" {
    expr = "length(trim(product_name)) > 0"
  }

  check "order_items_quantity_positive" {
    expr = "quantity > 0"
  }

  check "order_items_minimum_order_quantity_positive" {
    expr = "minimum_order_quantity > 0"
  }

  check "order_items_quantity_meets_minimum" {
    expr = "quantity >= minimum_order_quantity"
  }

  check "order_items_unit_price_nonnegative" {
    expr = "unit_price_amount >= 0"
  }

  check "order_items_line_total_nonnegative" {
    expr = "line_total_amount >= 0"
  }

  check "order_items_line_total_consistent" {
    expr = "line_total_amount = unit_price_amount * quantity"
  }

  check "order_items_currency_valid" {
    expr = "length(currency) = 3 AND currency = upper(currency)"
  }

  check "order_items_unit_cost_nonnegative" {
    expr = "unit_cost_amount IS NULL OR unit_cost_amount >= 0"
  }

  check "order_items_line_cost_nonnegative" {
    expr = "line_cost_amount IS NULL OR line_cost_amount >= 0"
  }

  check "order_items_cost_currency_valid" {
    expr = "cost_currency IS NULL OR (length(cost_currency) = 3 AND cost_currency = upper(cost_currency))"
  }

  check "order_items_cost_snapshot_consistent" {
    expr = "(unit_cost_amount IS NULL AND line_cost_amount IS NULL AND cost_currency IS NULL) OR (unit_cost_amount IS NOT NULL AND line_cost_amount IS NOT NULL AND cost_currency IS NOT NULL AND line_cost_amount::numeric = unit_cost_amount::numeric * quantity::numeric)"
  }
}

table "order_events" {
  schema = schema.public

  column "id" {
    type    = uuid
    null    = false
    default = sql("gen_random_uuid()")
  }

  column "order_id" {
    type = uuid
    null = false
  }

  column "event_type" {
    type = varchar(60)
    null = false
  }

  column "from_status" {
    type = varchar(30)
    null = true
  }

  column "to_status" {
    type = varchar(30)
    null = true
  }

  column "message" {
    type = varchar(500)
    null = true
  }

  column "actor_type" {
    type = varchar(40)
    null = true
  }

  column "actor_id" {
    type = varchar(100)
    null = true
  }

  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  foreign_key "order_events_order_id_fkey" {
    columns     = [column.order_id]
    ref_columns = [table.orders.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }

  index "idx_order_events_order_created" {
    columns = [
      column.order_id,
      column.created_at,
    ]
  }

  check "order_events_event_type_not_blank" {
    expr = "length(trim(event_type)) > 0"
  }
}

table "shipments" {
  schema = schema.public

  column "id" {
    type    = uuid
    null    = false
    default = sql("gen_random_uuid()")
  }

  column "order_id" {
    type = uuid
    null = false
  }

  column "origin_warehouse_id" {
    type = uuid
    null = true
  }

  column "delivery_mode" {
    type    = varchar(30)
    null    = false
    default = "courier"
  }

  column "provider_code" {
    type = varchar(60)
    null = true
  }

  column "provider_shipment_id" {
    type = varchar(160)
    null = true
  }

  column "provider_status" {
    type = varchar(80)
    null = true
  }

  column "courier_name" {
    type = varchar(120)
    null = true
  }

  column "courier_reference" {
    type = varchar(160)
    null = true
  }

  column "rider_reference" {
    type = varchar(160)
    null = true
  }

  column "tracking_number" {
    type = varchar(160)
    null = true
  }

  column "tracking_url" {
    type = varchar(1000)
    null = true
  }

  column "status" {
    type    = varchar(30)
    null    = false
    default = "pending"
  }

  column "shipped_at" {
    type = timestamptz
    null = true
  }

  column "provider_delivered_at" {
    type = timestamptz
    null = true
  }

  column "awaiting_confirmation_at" {
    type = timestamptz
    null = true
  }

  column "confirmed_received_at" {
    type = timestamptz
    null = true
  }

  column "confirmation_source" {
    type = varchar(30)
    null = true
  }

  column "confirmed_by_actor_id" {
    type = varchar(160)
    null = true
  }

  column "confirmation_note" {
    type = varchar(1000)
    null = true
  }

  column "last_provider_sync_at" {
    type = timestamptz
    null = true
  }

  column "delivered_at" {
    type = timestamptz
    null = true
  }

  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  column "updated_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  foreign_key "shipments_order_id_fkey" {
    columns     = [column.order_id]
    ref_columns = [table.orders.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }

  foreign_key "shipments_origin_warehouse_fkey" {
    columns     = [column.origin_warehouse_id]
    ref_columns = [table.warehouses.column.id]
    on_update   = NO_ACTION
    on_delete   = SET_NULL
  }

  # TEMPORARILY keep one shipment per order.
  #
  # The current Go Order and Delivery repositories still operate on a
  # single shipment per order. Remove this only when the application
  # model is explicitly refactored for multiple shipments.
  index "shipments_order_id_key" {
    unique  = true
    columns = [column.order_id]
  }

  index "idx_shipments_order_created" {
    columns = [
      column.order_id,
      column.created_at,
    ]
  }

  index "idx_shipments_status_created" {
    columns = [
      column.status,
      column.created_at,
    ]
  }

  index "idx_shipments_tracking_number" {
    columns = [column.tracking_number]
  }

  index "shipments_provider_shipment_key" {
    unique = true

    columns = [
      column.provider_code,
      column.provider_shipment_id,
    ]

    where = "provider_code IS NOT NULL AND provider_shipment_id IS NOT NULL"
  }

  index "idx_shipments_confirmation" {
    columns = [
      column.status,
      column.awaiting_confirmation_at,
    ]
  }

  index "idx_shipments_origin_warehouse_created" {
    columns = [
      column.origin_warehouse_id,
      column.created_at,
    ]
  }

  check "shipments_courier_name_not_blank" {
    expr = "courier_name IS NULL OR length(trim(courier_name)) > 0"
  }

  check "shipments_provider_code_not_blank" {
    expr = "provider_code IS NULL OR length(trim(provider_code)) > 0"
  }

  check "shipments_provider_shipment_id_not_blank" {
    expr = "provider_shipment_id IS NULL OR length(trim(provider_shipment_id)) > 0"
  }

  check "shipments_rider_reference_not_blank" {
    expr = "rider_reference IS NULL OR length(trim(rider_reference)) > 0"
  }

  check "shipments_delivery_mode_valid" {
    expr = "delivery_mode IN ('courier', 'self_pickup', 'community_rider')"
  }

  check "shipments_status_valid" {
    expr = "status IN ('pending', 'shipped', 'awaiting_confirmation', 'delivered', 'cancelled')"
  }

  check "shipments_confirmation_source_valid" {
    expr = "confirmation_source IS NULL OR confirmation_source IN ('customer', 'support', 'admin')"
  }

  check "shipments_confirmation_fields_consistent" {
    expr = "(confirmed_received_at IS NULL AND confirmation_source IS NULL AND confirmed_by_actor_id IS NULL) OR (confirmed_received_at IS NOT NULL AND confirmation_source IS NOT NULL AND confirmed_by_actor_id IS NOT NULL)"
  }

  check "shipments_awaiting_confirmation_requires_timestamp" {
    expr = "status <> 'awaiting_confirmation' OR awaiting_confirmation_at IS NOT NULL"
  }

  # Final Logistics-v0 receipt integrity:
  #
  # A shipment cannot be persisted as delivered unless receipt was
  # explicitly confirmed by customer/support/admin and delivered_at was
  # recorded.
  check "shipments_delivered_requires_confirmation" {
    expr = "status <> 'delivered' OR (delivered_at IS NOT NULL AND confirmed_received_at IS NOT NULL AND confirmation_source IS NOT NULL AND confirmed_by_actor_id IS NOT NULL)"
  }

  # Confirmation data belongs only to a final delivered shipment.
  check "shipments_confirmation_requires_delivered_status" {
    expr = "confirmed_received_at IS NULL OR status = 'delivered'"
  }

  # delivered_at is also a final-state timestamp and must not appear on
  # pending/shipped/awaiting-confirmation/cancelled shipments.
  check "shipments_delivered_at_requires_delivered_status" {
    expr = "delivered_at IS NULL OR status = 'delivered'"
  }
}


table "shipment_fulfillments" {
  schema = schema.public

  column "shipment_id" {
    type = uuid
    null = false
  }

  column "fulfillment_id" {
    type = uuid
    null = false
  }

  column "quantity" {
    type = integer
    null = false
  }

  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  primary_key {
    columns = [
      column.shipment_id,
      column.fulfillment_id,
    ]
  }

  foreign_key "shipment_fulfillments_shipment_fkey" {
    columns     = [column.shipment_id]
    ref_columns = [table.shipments.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }

  foreign_key "shipment_fulfillments_fulfillment_fkey" {
    columns     = [column.fulfillment_id]
    ref_columns = [table.warehouse_fulfillments.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }

  index "idx_shipment_fulfillments_fulfillment" {
    columns = [column.fulfillment_id]
  }

  check "shipment_fulfillments_quantity_positive" {
    expr = "quantity > 0"
  }
}


table "warehouse_handoffs" {
  schema = schema.public

  column "id" {
    type    = uuid
    null    = false
    default = sql("gen_random_uuid()")
  }

  column "shipment_id" {
    type = uuid
    null = false
  }

  column "warehouse_id" {
    type = uuid
    null = false
  }

  column "handoff_type" {
    type = varchar(30)
    null = false
  }

  column "reference" {
    type = varchar(160)
    null = true
  }

  column "handed_off_by_actor_type" {
    type = varchar(40)
    null = true
  }

  column "handed_off_by_actor_id" {
    type = varchar(160)
    null = true
  }

  column "handed_off_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  foreign_key "warehouse_handoffs_shipment_fkey" {
    columns     = [column.shipment_id]
    ref_columns = [table.shipments.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }

  foreign_key "warehouse_handoffs_warehouse_fkey" {
    columns     = [column.warehouse_id]
    ref_columns = [table.warehouses.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }

  # Handoffs are history/audit records.
  #
  # Do NOT make shipment_id unique.
  #
  # A shipment may later be handed to Pathao, fail pickup,
  # then be reassigned and handed to another courier/rider.
  index "idx_warehouse_handoffs_shipment_time" {
    columns = [
      column.shipment_id,
      column.handed_off_at,
    ]
  }

  index "idx_warehouse_handoffs_warehouse_time" {
    columns = [
      column.warehouse_id,
      column.handed_off_at,
    ]
  }

  check "warehouse_handoffs_type_valid" {
    expr = "handoff_type IN ('courier', 'self_pickup', 'community_rider')"
  }

  check "warehouse_handoffs_actor_pair" {
    expr = "(handed_off_by_actor_type IS NULL AND handed_off_by_actor_id IS NULL) OR (handed_off_by_actor_type IS NOT NULL AND handed_off_by_actor_id IS NOT NULL)"
  }
}

table "delivery_tracking_events" {
  schema = schema.public

  column "id" {
    type    = uuid
    null    = false
    default = sql("gen_random_uuid()")
  }

  column "shipment_id" {
    type = uuid
    null = false
  }

  column "order_id" {
    type = uuid
    null = false
  }

  column "source" {
    type = varchar(30)
    null = false
  }

  column "event_code" {
    type = varchar(80)
    null = false
  }

  column "status" {
    type = varchar(80)
    null = true
  }

  column "message" {
    type = varchar(1000)
    null = true
  }

  column "latitude" {
    type = double_precision
    null = true
  }

  column "longitude" {
    type = double_precision
    null = true
  }

    column "public_message" {
    type = varchar(500)
    null = true
  }

  column "country_code" {
    type = varchar(2)
    null = true
  }

  column "city" {
    type = varchar(120)
    null = true
  }

  column "location_name" {
    type = varchar(160)
    null = true
  }

  column "customer_visible" {
    type    = boolean
    null    = false
    default = true
  }

  column "external_event_id" {
    type = varchar(160)
    null = true
  }

  column "metadata" {
    type = jsonb
    null = true
  }

  column "occurred_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  foreign_key "delivery_tracking_events_shipment_fkey" {
    columns     = [column.shipment_id]
    ref_columns = [table.shipments.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }

  foreign_key "delivery_tracking_events_order_fkey" {
    columns     = [column.order_id]
    ref_columns = [table.orders.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }

  index "idx_delivery_tracking_events_order_time" {
    columns = [
      column.order_id,
      column.occurred_at,
      column.created_at,
    ]
  }

  index "idx_delivery_tracking_events_shipment_time" {
    columns = [
      column.shipment_id,
      column.occurred_at,
      column.created_at,
    ]
  }

  index "delivery_tracking_events_external_key" {
    unique = true

    columns = [
      column.shipment_id,
      column.external_event_id,
    ]

    where = "external_event_id IS NOT NULL"
  }

  check "delivery_tracking_events_source_valid" {
    expr = "source IN ('manual', 'provider', 'rider', 'customer', 'support', 'admin', 'system')"
  }

  check "delivery_tracking_events_event_code_not_blank" {
    expr = "length(trim(event_code)) > 0"
  }

  check "delivery_tracking_events_latitude_valid" {
    expr = "latitude IS NULL OR (latitude >= -90 AND latitude <= 90)"
  }

  check "delivery_tracking_events_longitude_valid" {
    expr = "longitude IS NULL OR (longitude >= -180 AND longitude <= 180)"
  }

    check "delivery_tracking_events_country_valid" {
    expr = "country_code IS NULL OR (length(country_code) = 2 AND country_code = upper(country_code))"
  }

  check "delivery_tracking_events_coordinates_pair" {
    expr = "(latitude IS NULL AND longitude IS NULL) OR (latitude IS NOT NULL AND longitude IS NOT NULL)"
  }
}


table "delivery_otp_challenges" {
  schema = schema.public

  column "id" {
    type    = uuid
    null    = false
    default = sql("gen_random_uuid()")
  }

  column "shipment_id" {
    type = uuid
    null = false
  }

  column "purpose" {
    type = varchar(40)
    null = false
  }

  column "channel" {
    type    = varchar(20)
    null    = false
    default = "sms"
  }

  column "recipient" {
    type = varchar(255)
    null = false
  }

  column "code_hash" {
    type = varchar(255)
    null = false
  }

  column "status" {
    type    = varchar(20)
    null    = false
    default = "pending"
  }

  column "expires_at" {
    type = timestamptz
    null = false
  }

  column "attempt_count" {
    type    = integer
    null    = false
    default = 0
  }

  column "max_attempts" {
    type    = integer
    null    = false
    default = 5
  }

  column "verified_at" {
    type = timestamptz
    null = true
  }

  column "consumed_at" {
    type = timestamptz
    null = true
  }

  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  column "updated_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  foreign_key "delivery_otp_challenges_shipment_fkey" {
    columns     = [column.shipment_id]
    ref_columns = [table.shipments.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }

  index "idx_delivery_otp_challenges_shipment_status" {
    columns = [
      column.shipment_id,
      column.status,
      column.created_at,
    ]
  }

  index "idx_delivery_otp_challenges_expiry" {
    columns = [
      column.status,
      column.expires_at,
    ]
  }

  index "delivery_otp_challenges_one_active" {
  unique = true

  columns = [
    column.shipment_id,
    column.purpose,
    column.channel,
    column.recipient,
  ]

  where = "(status)::text = ANY ((ARRAY['pending'::character varying, 'verified'::character varying])::text[])"
}

  check "delivery_otp_challenges_purpose_valid" {
    expr = "purpose IN ('delivery_confirmation', 'self_pickup', 'rider_pickup', 'rider_delivery')"
  }

  check "delivery_otp_challenges_channel_valid" {
    expr = "channel IN ('sms')"
  }

  check "delivery_otp_challenges_status_valid" {
    expr = "status IN ('pending', 'verified', 'consumed', 'expired', 'locked', 'cancelled')"
  }

  check "delivery_otp_challenges_recipient_not_blank" {
    expr = "length(trim(recipient)) > 0"
  }

  check "delivery_otp_challenges_code_hash_not_blank" {
    expr = "length(trim(code_hash)) > 0"
  }

  check "delivery_otp_challenges_attempts_valid" {
    expr = "attempt_count >= 0 AND max_attempts >= 1 AND max_attempts <= 10 AND attempt_count <= max_attempts"
  }

  check "delivery_otp_challenges_expiry_valid" {
    expr = "expires_at > created_at"
  }

  check "delivery_otp_challenges_verified_timestamp" {
    expr = "status NOT IN ('verified', 'consumed') OR verified_at IS NOT NULL"
  }

  check "delivery_otp_challenges_consumed_timestamp" {
    expr = "status <> 'consumed' OR (verified_at IS NOT NULL AND consumed_at IS NOT NULL)"
  }
}


table "delivery_proofs" {
  schema = schema.public

  column "id" {
    type    = uuid
    null    = false
    default = sql("gen_random_uuid()")
  }

  column "shipment_id" {
    type = uuid
    null = false
  }

  column "otp_challenge_id" {
    type = uuid
    null = true
  }

  column "purpose" {
    type = varchar(40)
    null = false
  }

  column "proof_type" {
    type = varchar(40)
    null = false
  }

  column "source" {
    type = varchar(30)
    null = false
  }

  column "actor_id" {
    type = varchar(160)
    null = true
  }

  column "storage_key" {
    type = varchar(1000)
    null = true
  }

  column "external_reference" {
    type = varchar(160)
    null = true
  }

  column "verification_status" {
    type    = varchar(20)
    null    = false
    default = "recorded"
  }

  column "metadata" {
    type = jsonb
    null = true
  }

  column "occurred_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  foreign_key "delivery_proofs_shipment_fkey" {
    columns     = [column.shipment_id]
    ref_columns = [table.shipments.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }

  foreign_key "delivery_proofs_otp_challenge_fkey" {
    columns     = [column.otp_challenge_id]
    ref_columns = [table.delivery_otp_challenges.column.id]
    on_update   = NO_ACTION
    on_delete   = SET_NULL
  }

  index "idx_delivery_proofs_shipment_time" {
    columns = [
      column.shipment_id,
      column.occurred_at,
      column.created_at,
    ]
  }

  index "delivery_proofs_otp_challenge_key" {
    unique  = true
    columns = [column.otp_challenge_id]
    where   = "otp_challenge_id IS NOT NULL"
  }

  check "delivery_proofs_purpose_valid" {
    expr = "purpose IN ('delivery_confirmation', 'self_pickup', 'rider_pickup', 'rider_delivery')"
  }

  check "delivery_proofs_type_valid" {
    expr = "proof_type IN ('otp', 'photo', 'signature', 'provider_reference', 'receipt_confirmation', 'pickup_confirmation', 'staff_confirmation')"
  }

  check "delivery_proofs_source_valid" {
    expr = "source IN ('manual', 'provider', 'rider', 'customer', 'support', 'admin', 'system')"
  }

  check "delivery_proofs_verification_status_valid" {
    expr = "verification_status IN ('recorded', 'pending', 'verified', 'rejected')"
  }

  check "delivery_proofs_actor_not_blank" {
    expr = "actor_id IS NULL OR length(trim(actor_id)) > 0"
  }

  check "delivery_proofs_storage_key_not_blank" {
    expr = "storage_key IS NULL OR length(trim(storage_key)) > 0"
  }

  check "delivery_proofs_external_reference_not_blank" {
    expr = "external_reference IS NULL OR length(trim(external_reference)) > 0"
  }
}

table "payments" {
  schema = schema.public

  column "id" {
    type    = uuid
    null    = false
    default = sql("gen_random_uuid()")
  }

  column "order_id" {
    type = uuid
    null = false
  }

  column "provider" {
    type = varchar(30)
    null = false
  }

  column "status" {
    type    = varchar(30)
    null    = false
    default = "pending"
  }

  column "amount" {
    type = bigint
    null = false
  }

  column "currency" {
    type = varchar(3)
    null = false
  }

  # Local idempotent payment-attempt reference.
  #
  # New provider payment initiations receive this before any
  # external API call is made. Older payment rows may remain NULL.
  column "attempt_key" {
    type = varchar(100)
    null = true
  }

  # NULL is valid while a local payment attempt exists but the
  # provider has not yet acknowledged/created its payment.
  column "provider_payment_id" {
    type = varchar(160)
    null = true
  }

  column "provider_transaction_id" {
    type = varchar(160)
    null = true
  }

  # Provider/customer handoff URL returned after successful
  # initiation. Never required for COD.
  column "handoff_url" {
    type = text
    null = true
  }

  # Provider-side deadline if one is supplied.
  #
  # The Order payment_due_at remains the commerce-system
  # authoritative deadline.
  column "provider_expires_at" {
    type = timestamptz
    null = true
  }

  column "paid_at" {
    type = timestamptz
    null = true
  }

  column "failed_at" {
    type = timestamptz
    null = true
  }

  column "failure_code" {
    type = varchar(100)
    null = true
  }

  column "failure_message" {
    type = varchar(500)
    null = true
  }

  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  column "updated_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  foreign_key "payments_order_id_fkey" {
    columns     = [column.order_id]
    ref_columns = [table.orders.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }

  index "idx_payments_order_created" {
    columns = [
      column.order_id,
      column.created_at,
    ]
  }

  index "idx_payments_status_created" {
    columns = [
      column.status,
      column.created_at,
    ]
  }

  index "payments_attempt_key_key" {
    unique = true

    columns = [
      column.attempt_key,
    ]

    where = "attempt_key IS NOT NULL"
  }

  # Concurrent HTTP retries for the same Order must converge on the
  # same pending payment attempt rather than creating two provider
  # payments.
  index "payments_one_pending_per_order" {
    unique = true

    columns = [
      column.order_id,
    ]

    where = "status = 'pending'"
  }

  index "payments_one_succeeded_per_order" {
    unique = true

    columns = [
      column.order_id,
    ]

    where = "status = 'succeeded'"
  }

  index "payments_provider_payment_id_key" {
    unique = true

    columns = [
      column.provider,
      column.provider_payment_id,
    ]
  }

  index "payments_provider_transaction_id_key" {
    unique = true

    columns = [
      column.provider,
      column.provider_transaction_id,
    ]

    where = "provider_transaction_id IS NOT NULL"
  }

  check "payments_provider_valid" {
    expr = "provider IN ('bkash', 'nagad', 'rocket', 'bank_transfer')"
  }

  check "payments_status_valid" {
    expr = "status IN ('pending', 'succeeded', 'failed', 'expired', 'refunded')"
  }

  check "payments_amount_nonnegative" {
    expr = "amount >= 0"
  }

  check "payments_currency_valid" {
    expr = "length(currency) = 3 AND currency = upper(currency)"
  }

  check "payments_attempt_key_not_blank" {
    expr = "attempt_key IS NULL OR length(trim(attempt_key)) > 0"
  }

  check "payments_provider_payment_id_not_blank" {
    expr = "provider_payment_id IS NULL OR length(trim(provider_payment_id)) > 0"
  }

  check "payments_provider_transaction_id_not_blank" {
    expr = "provider_transaction_id IS NULL OR length(trim(provider_transaction_id)) > 0"
  }

  check "payments_handoff_url_not_blank" {
    expr = "handoff_url IS NULL OR length(trim(handoff_url)) > 0"
  }

  check "payments_provider_expiry_valid" {
    expr = "provider_expires_at IS NULL OR provider_expires_at > created_at"
  }
}


table "payment_events" {
  schema = schema.public

  column "id" {
    type    = uuid
    null    = false
    default = sql("gen_random_uuid()")
  }

  column "order_id" {
    type = uuid
    null = false
  }

  column "payment_id" {
    type = uuid
    null = true
  }

  column "provider" {
    type = varchar(30)
    null = false
  }

  column "provider_event_id" {
    type = varchar(200)
    null = false
  }

  column "event_type" {
    type = varchar(100)
    null = false
  }

  column "provider_payment_id" {
    type = varchar(160)
    null = false
  }

  column "provider_transaction_id" {
    type = varchar(160)
    null = true
  }

  column "payload_sha256" {
    type = varchar(64)
    null = false
  }

  column "status" {
    type    = varchar(30)
    null    = false
    default = "received"
  }

  column "error_message" {
    type = varchar(500)
    null = true
  }

  column "received_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  column "processed_at" {
    type = timestamptz
    null = true
  }

  primary_key {
    columns = [column.id]
  }

  foreign_key "payment_events_order_id_fkey" {
    columns     = [column.order_id]
    ref_columns = [table.orders.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }

  foreign_key "payment_events_payment_id_fkey" {
    columns     = [column.payment_id]
    ref_columns = [table.payments.column.id]
    on_update   = NO_ACTION
    on_delete   = SET_NULL
  }

  index "payment_events_provider_event_id_key" {
    unique = true

    columns = [
      column.provider,
      column.provider_event_id,
    ]
  }

  index "idx_payment_events_order_received" {
    columns = [
      column.order_id,
      column.received_at,
    ]
  }

  index "idx_payment_events_status_received" {
    columns = [
      column.status,
      column.received_at,
    ]
  }

  check "payment_events_provider_valid" {
    expr = "provider IN ('bkash', 'nagad', 'rocket', 'bank_transfer')"
  }

  check "payment_events_provider_event_id_not_blank" {
    expr = "length(trim(provider_event_id)) > 0"
  }

  check "payment_events_event_type_not_blank" {
    expr = "length(trim(event_type)) > 0"
  }

  check "payment_events_provider_payment_id_not_blank" {
    expr = "length(trim(provider_payment_id)) > 0"
  }

  check "payment_events_payload_sha256_valid" {
    expr = "length(payload_sha256) = 64"
  }

  check "payment_events_status_valid" {
    expr = "status IN ('received', 'processed', 'ignored', 'failed')"
  }
}
table "returns" {
  schema = schema.public

  column "id" {
    type    = uuid
    null    = false
    default = sql("gen_random_uuid()")
  }

  column "return_number" {
    type = varchar(50)
    null = false
  }

  column "order_id" {
    type = uuid
    null = false
  }

  column "status" {
    type    = varchar(30)
    null    = false
    default = "requested"
  }

  column "customer_note" {
    type = varchar(1000)
    null = true
  }

  column "requested_by" {
    type = varchar(100)
    null = true
  }

  column "approved_by" {
    type = varchar(100)
    null = true
  }

  column "rejected_by" {
    type = varchar(100)
    null = true
  }

  column "rejection_reason" {
    type = varchar(500)
    null = true
  }

  column "requested_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  column "approved_at" {
    type = timestamptz
    null = true
  }

  column "rejected_at" {
    type = timestamptz
    null = true
  }

  column "received_at" {
    type = timestamptz
    null = true
  }

  column "inspected_at" {
    type = timestamptz
    null = true
  }

  column "completed_at" {
    type = timestamptz
    null = true
  }

  column "cancelled_at" {
    type = timestamptz
    null = true
  }

  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  column "updated_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  foreign_key "returns_order_id_fkey" {
    columns     = [column.order_id]
    ref_columns = [table.orders.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }

  index "returns_return_number_key" {
    unique  = true
    columns = [column.return_number]
  }

  index "idx_returns_order_created" {
    columns = [
      column.order_id,
      column.created_at,
    ]
  }

  index "idx_returns_status_created" {
    columns = [
      column.status,
      column.created_at,
    ]
  }

  check "returns_return_number_not_blank" {
    expr = "length(trim(return_number)) > 0"
  }

  check "returns_status_valid" {
    expr = "status IN ('requested', 'approved', 'rejected', 'received', 'inspected', 'completed', 'cancelled')"
  }
}

table "return_items" {
  schema = schema.public

  column "id" {
    type    = uuid
    null    = false
    default = sql("gen_random_uuid()")
  }

  column "return_id" {
    type = uuid
    null = false
  }

  column "order_item_id" {
    type = uuid
    null = false
  }

  column "quantity" {
    type = integer
    null = false
  }

  column "reason_code" {
    type = varchar(80)
    null = false
  }

  column "reason_note" {
    type = varchar(500)
    null = true
  }

  column "received_quantity" {
    type    = integer
    null    = false
    default = 0
  }

  column "restock_quantity" {
    type    = integer
    null    = false
    default = 0
  }

  column "inspection_status" {
    type    = varchar(30)
    null    = false
    default = "pending"
  }

  column "inspection_note" {
    type = varchar(500)
    null = true
  }

  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  column "updated_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  foreign_key "return_items_return_id_fkey" {
    columns     = [column.return_id]
    ref_columns = [table.returns.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }

  foreign_key "return_items_order_item_id_fkey" {
    columns     = [column.order_item_id]
    ref_columns = [table.order_items.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }

  index "return_items_return_order_item_key" {
    unique = true
    columns = [
      column.return_id,
      column.order_item_id,
    ]
  }

  index "idx_return_items_order_item" {
    columns = [column.order_item_id]
  }

  check "return_items_quantity_positive" {
    expr = "quantity > 0"
  }

  check "return_items_reason_code_not_blank" {
    expr = "length(trim(reason_code)) > 0"
  }

  check "return_items_received_quantity_valid" {
    expr = "received_quantity >= 0 AND received_quantity <= quantity"
  }

  check "return_items_restock_quantity_valid" {
    expr = "restock_quantity >= 0 AND restock_quantity <= received_quantity"
  }

  check "return_items_inspection_status_valid" {
    expr = "inspection_status IN ('pending', 'restockable', 'damaged', 'non_restockable')"
  }
}

table "return_events" {
  schema = schema.public

  column "id" {
    type    = uuid
    null    = false
    default = sql("gen_random_uuid()")
  }

  column "return_id" {
    type = uuid
    null = false
  }

  column "event_type" {
    type = varchar(60)
    null = false
  }

  column "from_status" {
    type = varchar(30)
    null = true
  }

  column "to_status" {
    type = varchar(30)
    null = true
  }

  column "message" {
    type = varchar(500)
    null = true
  }

  column "actor_type" {
    type = varchar(40)
    null = true
  }

  column "actor_id" {
    type = varchar(100)
    null = true
  }

  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  foreign_key "return_events_return_id_fkey" {
    columns     = [column.return_id]
    ref_columns = [table.returns.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }

  index "idx_return_events_return_created" {
    columns = [
      column.return_id,
      column.created_at,
    ]
  }

  check "return_events_event_type_not_blank" {
    expr = "length(trim(event_type)) > 0"
  }
}


table "refunds" {
  schema = schema.public

  column "id" {
    type    = uuid
    null    = false
    default = sql("gen_random_uuid()")
  }

  column "refund_number" {
    type = varchar(50)
    null = false
  }

  column "order_id" {
    type = uuid
    null = false
  }

  column "return_id" {
    type = uuid
    null = true
  }

  column "payment_id" {
    type = uuid
    null = true
  }

  column "source_type" {
    type = varchar(30)
    null = false
  }

  column "status" {
    type    = varchar(30)
    null    = false
    default = "requested"
  }

  column "amount" {
    type = bigint
    null = false
  }

  column "currency" {
    type = varchar(3)
    null = false
  }

  column "provider" {
    type = varchar(50)
    null = true
  }

  column "provider_refund_id" {
    type = varchar(160)
    null = true
  }

  column "reason" {
    type = varchar(500)
    null = false
  }

  column "requested_by" {
    type = varchar(100)
    null = true
  }

  column "approved_by" {
    type = varchar(100)
    null = true
  }

  column "failure_code" {
    type = varchar(100)
    null = true
  }

  column "failure_message" {
    type = varchar(500)
    null = true
  }

  column "requested_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  column "approved_at" {
    type = timestamptz
    null = true
  }

  column "processing_at" {
    type = timestamptz
    null = true
  }

  column "succeeded_at" {
    type = timestamptz
    null = true
  }

  column "failed_at" {
    type = timestamptz
    null = true
  }

  column "cancelled_at" {
    type = timestamptz
    null = true
  }

  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  column "updated_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  foreign_key "refunds_order_id_fkey" {
    columns     = [column.order_id]
    ref_columns = [table.orders.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }

  foreign_key "refunds_return_id_fkey" {
    columns     = [column.return_id]
    ref_columns = [table.returns.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }

  foreign_key "refunds_payment_id_fkey" {
    columns     = [column.payment_id]
    ref_columns = [table.payments.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }

  index "refunds_refund_number_key" {
    unique  = true
    columns = [column.refund_number]
  }

  index "idx_refunds_order_created" {
    columns = [
      column.order_id,
      column.created_at,
    ]
  }

  index "idx_refunds_return" {
    columns = [column.return_id]
  }

  index "idx_refunds_payment" {
    columns = [column.payment_id]
  }

  index "idx_refunds_status_created" {
    columns = [
      column.status,
      column.created_at,
    ]
  }

  index "refunds_provider_refund_id_key" {
    unique  = true
    columns = [
      column.provider,
      column.provider_refund_id,
    ]
    where = "provider_refund_id IS NOT NULL"
  }

  check "refunds_refund_number_not_blank" {
    expr = "length(trim(refund_number)) > 0"
  }

  check "refunds_source_type_valid" {
    expr = "source_type IN ('cancellation', 'return', 'manual')"
  }

  check "refunds_status_valid" {
    expr = "status IN ('requested', 'approved', 'processing', 'succeeded', 'failed', 'cancelled')"
  }

  check "refunds_amount_positive" {
    expr = "amount > 0"
  }

  check "refunds_currency_valid" {
    expr = "length(currency) = 3 AND currency = upper(currency)"
  }

  check "refunds_provider_valid" {
    expr = "provider IS NULL OR provider IN ('bkash', 'nagad', 'rocket', 'bank_transfer', 'manual')"
  }

  check "refunds_reason_not_blank" {
    expr = "length(trim(reason)) > 0"
  }
}


table "refund_events" {
  schema = schema.public

  column "id" {
    type    = uuid
    null    = false
    default = sql("gen_random_uuid()")
  }

  column "refund_id" {
    type = uuid
    null = false
  }

  column "event_type" {
    type = varchar(60)
    null = false
  }

  column "from_status" {
    type = varchar(30)
    null = true
  }

  column "to_status" {
    type = varchar(30)
    null = true
  }

  column "message" {
    type = varchar(500)
    null = true
  }

  column "actor_type" {
    type = varchar(40)
    null = true
  }

  column "actor_id" {
    type = varchar(100)
    null = true
  }

  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  foreign_key "refund_events_refund_id_fkey" {
    columns     = [column.refund_id]
    ref_columns = [table.refunds.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }

  index "idx_refund_events_refund_created" {
    columns = [
      column.refund_id,
      column.created_at,
    ]
  }

  check "refund_events_event_type_not_blank" {
    expr = "length(trim(event_type)) > 0"
  }
}

table "customers" {
  schema = schema.public

  column "id" {
    type    = uuid
    null    = false
    default = sql("gen_random_uuid()")
  }

  column "avatar_storage_key" {
  type = text
  null = true
}

column "avatar_updated_at" {
  type = timestamptz
  null = true
}

  column "phone" {
    type = varchar(40)
    null = false
  }

  column "email" {
    type = varchar(255)
    null = true
  }

  column "password_hash" {
    type = text
    null = false
  }

  column "full_name" {
    type = varchar(160)
    null = false
  }

  column "status" {
    type    = varchar(20)
    null    = false
    default = "active"
  }

  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  column "updated_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  index "customers_phone_key" {
    unique  = true
    columns = [column.phone]
  }

  index "customers_email_key" {
    unique  = true
    columns = [column.email]
  }

  index "idx_customers_status_created" {
    columns = [
      column.status,
      column.created_at,
    ]
  }

  check "customers_phone_not_blank" {
    expr = "length(trim(phone)) > 0"
  }

  check "customers_email_not_blank" {
    expr = "email IS NULL OR length(trim(email)) > 0"
  }

  check "customers_password_hash_not_blank" {
    expr = "length(trim(password_hash)) > 0"
  }

  check "customers_full_name_not_blank" {
    expr = "length(trim(full_name)) > 0"
  }

  check "customers_status_valid" {
    expr = "status IN ('active', 'disabled','banned')"
  }
  check "customers_avatar_storage_key_not_blank" {
  expr = "avatar_storage_key IS NULL OR length(trim(avatar_storage_key)) > 0"
}

check "customers_avatar_consistent" {
  expr = "(avatar_storage_key IS NULL AND avatar_updated_at IS NULL) OR (avatar_storage_key IS NOT NULL AND avatar_updated_at IS NOT NULL)"
}
}

table "customer_addresses" {
  schema = schema.public

  column "id" {
    type    = uuid
    null    = false
    default = sql("gen_random_uuid()")
  }

  column "customer_id" {
    type = uuid
    null = false
  }

  column "label" {
    type    = varchar(80)
    null    = false
    default = "Address"
  }

  column "recipient_name" {
    type = varchar(160)
    null = false
  }

  column "phone" {
    type = varchar(40)
    null = false
  }

  column "address_line1" {
    type = varchar(255)
    null = false
  }

  column "address_line2" {
    type = varchar(255)
    null = true
  }

  column "city" {
    type = varchar(120)
    null = false
  }

  column "area" {
    type = varchar(120)
    null = false
  }

  column "postal_code" {
    type = varchar(30)
    null = true
  }

  column "is_default" {
    type    = boolean
    null    = false
    default = false
  }

  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  column "updated_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  foreign_key "customer_addresses_customer_id_fkey" {
    columns     = [column.customer_id]
    ref_columns = [table.customers.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }

  index "idx_customer_addresses_customer_created" {
    columns = [
      column.customer_id,
      column.created_at,
    ]
  }

  index "customer_addresses_one_default_per_customer" {
    unique  = true
    columns = [column.customer_id]
    where   = "is_default = true"
  }

  check "customer_addresses_label_not_blank" {
    expr = "length(trim(label)) > 0"
  }

  check "customer_addresses_recipient_name_not_blank" {
    expr = "length(trim(recipient_name)) > 0"
  }

  check "customer_addresses_phone_not_blank" {
    expr = "length(trim(phone)) > 0"
  }

  check "customer_addresses_address_line1_not_blank" {
    expr = "length(trim(address_line1)) > 0"
  }

  check "customer_addresses_city_not_blank" {
    expr = "length(trim(city)) > 0"
  }

  check "customer_addresses_area_not_blank" {
    expr = "length(trim(area)) > 0"
  }
}

table "customer_wishlist_items" {
  schema = schema.public

  column "id" {
    type    = uuid
    null    = false
    default = sql("gen_random_uuid()")
  }

  column "customer_id" {
    type = uuid
    null = false
  }

  column "product_id" {
    type = uuid
    null = false
  }

  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  primary_key {
    columns = [
      column.id,
    ]
  }

  foreign_key "customer_wishlist_items_customer_id_fkey" {
    columns = [
      column.customer_id,
    ]

    ref_columns = [
      table.customers.column.id,
    ]

    on_update = NO_ACTION
    on_delete = CASCADE
  }

  foreign_key "customer_wishlist_items_product_id_fkey" {
    columns = [
      column.product_id,
    ]

    ref_columns = [
      table.products.column.id,
    ]

    on_update = NO_ACTION
    on_delete = CASCADE
  }

  index "customer_wishlist_items_customer_product_key" {
    unique = true

    columns = [
      column.customer_id,
      column.product_id,
    ]
  }

  index "idx_customer_wishlist_items_customer_created" {
    columns = [
      column.customer_id,
      column.created_at,
      column.id,
    ]
  }

  index "idx_customer_wishlist_items_product" {
    columns = [
      column.product_id,
    ]
  }
}

table "customer_preferences" {
  schema = schema.public

  column "customer_id" {
    type = uuid
    null = false
  }

  column "locale" {
    type    = varchar(10)
    null    = false
    default = "en"
  }

  column "assistant_language" {
    type    = varchar(20)
    null    = false
    default = "auto"
  }

  column "currency" {
    type    = varchar(3)
    null    = false
    default = "BDT"
  }

  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  column "updated_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  primary_key {
    columns = [column.customer_id]
  }

  foreign_key "customer_preferences_customer_id_fkey" {
    columns     = [column.customer_id]
    ref_columns = [table.customers.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }

  check "customer_preferences_locale_valid" {
    expr = "locale IN ('en', 'bn')"
  }

  check "customer_preferences_assistant_language_valid" {
    expr = "assistant_language IN ('auto', 'en', 'bn', 'banglish')"
  }

  check "customer_preferences_currency_valid" {
    expr = "length(currency) = 3 AND currency = upper(currency)"
  }
}

table "customer_notification_preferences" {
  schema = schema.public

  column "customer_id" {
    type = uuid
    null = false
  }

  column "order_updates_sms" {
    type    = boolean
    null    = false
    default = true
  }

  column "order_updates_email" {
    type    = boolean
    null    = false
    default = true
  }

  column "delivery_updates_sms" {
    type    = boolean
    null    = false
    default = true
  }

  column "delivery_updates_email" {
    type    = boolean
    null    = false
    default = true
  }

  column "support_updates_sms" {
    type    = boolean
    null    = false
    default = true
  }

  column "support_updates_email" {
    type    = boolean
    null    = false
    default = true
  }

  column "promotions_sms" {
    type    = boolean
    null    = false
    default = false
  }

  column "promotions_email" {
    type    = boolean
    null    = false
    default = false
  }

  column "recommendations_email" {
    type    = boolean
    null    = false
    default = false
  }

  column "push_enabled" {
    type    = boolean
    null    = false
    default = false
  }

  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  column "updated_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  primary_key {
    columns = [column.customer_id]
  }

  foreign_key "customer_notification_preferences_customer_id_fkey" {
    columns     = [column.customer_id]
    ref_columns = [table.customers.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }
}

table "customer_privacy_settings" {
  schema = schema.public

  column "customer_id" {
    type = uuid
    null = false
  }

  column "personalization_enabled" {
    type    = boolean
    null    = false
    default = true
  }

  column "search_history_enabled" {
    type    = boolean
    null    = false
    default = true
  }

  column "recently_viewed_enabled" {
    type    = boolean
    null    = false
    default = true
  }

  column "save_tryon_media" {
    type    = boolean
    null    = false
    default = false
  }

  column "use_tryon_for_personalization" {
    type    = boolean
    null    = false
    default = false
  }

  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  column "updated_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  primary_key {
    columns = [column.customer_id]
  }

  foreign_key "customer_privacy_settings_customer_id_fkey" {
    columns     = [column.customer_id]
    ref_columns = [table.customers.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }
}

table "customer_style_profiles" {
  schema = schema.public

  column "customer_id" {
    type = uuid
    null = false
  }

  column "preferred_colors" {
    type    = jsonb
    null    = false
    default = sql("'[]'::jsonb")
  }

  column "preferred_styles" {
    type    = jsonb
    null    = false
    default = sql("'[]'::jsonb")
  }

  column "preferred_categories" {
    type    = jsonb
    null    = false
    default = sql("'[]'::jsonb")
  }

  column "fit_preference" {
    type = varchar(30)
    null = true
  }

  column "budget_min_amount" {
    type = bigint
    null = true
  }

  column "budget_max_amount" {
    type = bigint
    null = true
  }

  column "currency" {
    type    = varchar(3)
    null    = false
    default = "BDT"
  }

  column "notes" {
    type = varchar(500)
    null = true
  }

  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  column "updated_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  primary_key {
    columns = [column.customer_id]
  }

  foreign_key "customer_style_profiles_customer_id_fkey" {
    columns     = [column.customer_id]
    ref_columns = [table.customers.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }

  check "customer_style_profiles_colors_array" {
    expr = "jsonb_typeof(preferred_colors) = 'array'"
  }

  check "customer_style_profiles_styles_array" {
    expr = "jsonb_typeof(preferred_styles) = 'array'"
  }

  check "customer_style_profiles_categories_array" {
    expr = "jsonb_typeof(preferred_categories) = 'array'"
  }

  check "customer_style_profiles_fit_not_blank" {
    expr = "fit_preference IS NULL OR length(trim(fit_preference)) > 0"
  }

  check "customer_style_profiles_budget_min_nonnegative" {
    expr = "budget_min_amount IS NULL OR budget_min_amount >= 0"
  }

  check "customer_style_profiles_budget_max_nonnegative" {
    expr = "budget_max_amount IS NULL OR budget_max_amount >= 0"
  }

  check "customer_style_profiles_budget_range_valid" {
    expr = "budget_min_amount IS NULL OR budget_max_amount IS NULL OR budget_max_amount >= budget_min_amount"
  }

  check "customer_style_profiles_currency_valid" {
    expr = "length(currency) = 3 AND currency = upper(currency)"
  }

  check "customer_style_profiles_notes_not_blank" {
    expr = "notes IS NULL OR length(trim(notes)) > 0"
  }
}

table "customer_saved_sizes" {
  schema = schema.public

  column "id" {
    type    = uuid
    null    = false
    default = sql("gen_random_uuid()")
  }

  column "customer_id" {
    type = uuid
    null = false
  }

  column "category_key" {
    type = varchar(80)
    null = false
  }

  column "brand" {
    type = varchar(160)
    null = true
  }

  column "size_system" {
    type = varchar(20)
    null = true
  }

  column "size_label" {
    type = varchar(40)
    null = false
  }

  column "fit_preference" {
    type = varchar(30)
    null = true
  }

  column "notes" {
    type = varchar(255)
    null = true
  }

  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  column "updated_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  foreign_key "customer_saved_sizes_customer_id_fkey" {
    columns     = [column.customer_id]
    ref_columns = [table.customers.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }

  index "idx_customer_saved_sizes_customer" {
    columns = [
      column.customer_id,
      column.created_at,
    ]
  }

  index "customer_saved_sizes_generic_key" {
    unique = true

    columns = [
      column.customer_id,
      column.category_key,
    ]

    where = "brand IS NULL"
  }

  index "customer_saved_sizes_brand_key" {
    unique = true

    columns = [
      column.customer_id,
      column.category_key,
      column.brand,
    ]

    where = "brand IS NOT NULL"
  }

  check "customer_saved_sizes_category_not_blank" {
    expr = "length(trim(category_key)) > 0"
  }

  check "customer_saved_sizes_brand_not_blank" {
    expr = "brand IS NULL OR length(trim(brand)) > 0"
  }

  check "customer_saved_sizes_system_not_blank" {
    expr = "size_system IS NULL OR length(trim(size_system)) > 0"
  }

  check "customer_saved_sizes_label_not_blank" {
    expr = "length(trim(size_label)) > 0"
  }

  check "customer_saved_sizes_fit_not_blank" {
    expr = "fit_preference IS NULL OR length(trim(fit_preference)) > 0"
  }

  check "customer_saved_sizes_notes_not_blank" {
    expr = "notes IS NULL OR length(trim(notes)) > 0"
  }
}


table "auth_sessions" {
  schema = schema.public

  column "id" {
    type    = uuid
    null    = false
    default = sql("gen_random_uuid()")
  }

  column "customer_id" {
    type = uuid
    null = false
  }

  column "access_token_hash" {
    type = varchar(64)
    null = false
  }

  column "refresh_token_hash" {
    type = varchar(64)
    null = false
  }

  column "access_expires_at" {
    type = timestamptz
    null = false
  }

  column "refresh_expires_at" {
    type = timestamptz
    null = false
  }

  column "last_used_at" {
    type = timestamptz
    null = true
  }

  column "revoked_at" {
    type = timestamptz
    null = true
  }

  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  column "updated_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  foreign_key "auth_sessions_customer_id_fkey" {
    columns     = [column.customer_id]
    ref_columns = [table.customers.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }

  index "auth_sessions_access_token_hash_key" {
    unique  = true
    columns = [column.access_token_hash]
  }

  index "auth_sessions_refresh_token_hash_key" {
    unique  = true
    columns = [column.refresh_token_hash]
  }

  index "idx_auth_sessions_customer_created" {
    columns = [
      column.customer_id,
      column.created_at,
    ]
  }

  index "idx_auth_sessions_refresh_expiry" {
    columns = [column.refresh_expires_at]
    where   = "revoked_at IS NULL"
  }

  check "auth_sessions_access_token_hash_valid" {
    expr = "length(access_token_hash) = 64"
  }

  check "auth_sessions_refresh_token_hash_valid" {
    expr = "length(refresh_token_hash) = 64"
  }

  check "auth_sessions_expiry_valid" {
    expr = "refresh_expires_at > access_expires_at"
  }
}

table "customer_search_history" {
  schema = schema.public

  column "id" {
    type    = uuid
    null    = false
    default = sql("gen_random_uuid()")
  }

  column "customer_id" {
    type = uuid
    null = false
  }

  column "query" {
    type = varchar(200)
    null = false
  }

  column "normalized_query" {
    type = varchar(200)
    null = false
  }

  column "result_count" {
    type    = bigint
    null    = false
    default = 0
  }

  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  column "expires_at" {
    type    = timestamptz
    null    = false
    default = sql("now() + interval '72 hours'")
  }

  primary_key {
    columns = [column.id]
  }

  foreign_key "customer_search_history_customer_id_fkey" {
    columns     = [column.customer_id]
    ref_columns = [table.customers.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }

  index "idx_customer_search_history_customer_expires" {
    columns = [
      column.customer_id,
      column.expires_at,
    ]
  }

  index "idx_customer_search_history_expires" {
    columns = [column.expires_at]
  }

  index "idx_customer_search_history_customer_created" {
    columns = [
      column.customer_id,
      column.created_at,
    ]
  }

  check "customer_search_history_query_not_blank" {
    expr = "length(trim(query)) > 0"
  }

  check "customer_search_history_normalized_query_not_blank" {
    expr = "length(trim(normalized_query)) > 0"
  }

  check "customer_search_history_result_count_nonnegative" {
    expr = "result_count >= 0"
  }

  check "customer_search_history_expiry_valid" {
    expr = "expires_at > created_at AND expires_at <= created_at + interval '72 hours'"
  }
}

table "customer_search_history_categories" {
  schema = schema.public

  column "search_history_id" {
    type = uuid
    null = false
  }

  column "category_id" {
    type = uuid
    null = false
  }

  column "relevance_weight" {
    type    = smallint
    null    = false
    default = 100
  }

  primary_key {
    columns = [
      column.search_history_id,
      column.category_id,
    ]
  }

  foreign_key "customer_search_history_categories_history_id_fkey" {
    columns     = [column.search_history_id]
    ref_columns = [table.customer_search_history.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }

  foreign_key "customer_search_history_categories_category_id_fkey" {
    columns     = [column.category_id]
    ref_columns = [table.categories.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }

  index "idx_customer_search_history_categories_category" {
    columns = [column.category_id]
  }

  check "customer_search_history_categories_weight_valid" {
    expr = "relevance_weight BETWEEN 1 AND 100"
  }
}

table "recommendation_category_relations" {
  schema = schema.public

  column "source_category_id" {
    type = uuid
    null = false
  }

  column "target_category_id" {
    type = uuid
    null = false
  }

  column "relation_type" {
    type = varchar(20)
    null = false
  }

  column "weight" {
    type = smallint
    null = false
  }

  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  column "updated_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  primary_key {
    columns = [
      column.source_category_id,
      column.target_category_id,
    ]
  }

  foreign_key "recommendation_category_relations_source_fkey" {
    columns     = [column.source_category_id]
    ref_columns = [table.categories.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }

  foreign_key "recommendation_category_relations_target_fkey" {
    columns     = [column.target_category_id]
    ref_columns = [table.categories.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }

  index "idx_recommendation_category_relations_target" {
    columns = [column.target_category_id]
  }

  check "recommendation_category_relations_not_self" {
    expr = "source_category_id <> target_category_id"
  }

  check "recommendation_category_relations_type_valid" {
    expr = "relation_type IN ('related', 'complementary')"
  }

  check "recommendation_category_relations_weight_valid" {
    expr = "weight BETWEEN 1 AND 100"
  }
}

table "recommendation_events" {
  schema = schema.public

  column "id" {
    type    = uuid
    null    = false
    default = sql("gen_random_uuid()")
  }

  # Signed-in customer when available.
  #
  # Anonymous recommendation activity is still recorded through session_id.
  column "customer_id" {
    type = uuid
    null = true
  }

  # Frontend-generated anonymous/session correlation ID.
  #
  # Used for recommendation analytics without requiring authentication.
  column "session_id" {
    type = varchar(128)
    null = true
  }

  column "product_id" {
    type = uuid
    null = false
  }

  # Where the recommendation was displayed.
  #
  # catalog
  # product_page
  # home
  # search
  # cart
  column "placement" {
    type = varchar(40)
    null = false
  }

  # User interaction with the recommendation.
  #
  # impression
  # click
  # add_to_cart
  column "event_type" {
    type = varchar(30)
    null = false
  }

  # Recommendation strategy responsible for the product being surfaced.
  #
  # personalized
  # category_related
  # related
  # complementary
  # fallback
  # merchandising
  column "strategy" {
    type = varchar(40)
    null = false
  }

  # Zero/one-based presentation position supplied by the recommendation
  # response/client. NULL when rank is unavailable.
  column "rank_position" {
    type = integer
    null = true
  }

  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  foreign_key "recommendation_events_customer_id_fkey" {
    columns     = [column.customer_id]
    ref_columns = [table.customers.column.id]
    on_update   = NO_ACTION
    on_delete   = SET_NULL
  }

  foreign_key "recommendation_events_product_id_fkey" {
    columns     = [column.product_id]
    ref_columns = [table.products.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }

  index "idx_recommendation_events_created" {
    columns = [column.created_at]
  }

  index "idx_recommendation_events_product_created" {
    columns = [
      column.product_id,
      column.created_at,
    ]
  }

  index "idx_recommendation_events_customer_created" {
    columns = [
      column.customer_id,
      column.created_at,
    ]
    where = "customer_id IS NOT NULL"
  }

  index "idx_recommendation_events_session_created" {
    columns = [
      column.session_id,
      column.created_at,
    ]
    where = "session_id IS NOT NULL"
  }

  index "idx_recommendation_events_type_created" {
    columns = [
      column.event_type,
      column.created_at,
    ]
  }

  index "idx_recommendation_events_placement_created" {
    columns = [
      column.placement,
      column.created_at,
    ]
  }

  index "idx_recommendation_events_strategy_created" {
    columns = [
      column.strategy,
      column.created_at,
    ]
  }

  check "recommendation_events_identity_present" {
    expr = "customer_id IS NOT NULL OR (session_id IS NOT NULL AND length(trim(session_id)) > 0)"
  }

  check "recommendation_events_session_id_valid" {
    expr = "session_id IS NULL OR length(trim(session_id)) > 0"
  }

  check "recommendation_events_placement_valid" {
    expr = "placement IN ('catalog', 'product_page', 'home', 'search', 'cart')"
  }

  check "recommendation_events_event_type_valid" {
    expr = "event_type IN ('impression', 'click', 'add_to_cart')"
  }

  check "recommendation_events_strategy_valid" {
    expr = "strategy IN ('personalized', 'category_related', 'related', 'complementary', 'fallback', 'merchandising')"
  }

  check "recommendation_events_rank_position_valid" {
    expr = "rank_position IS NULL OR rank_position >= 0"
  }
}

table "product_search_embeddings" {
  schema = schema.public

  column "product_id" {
    type = uuid
    null = false
  }

  column "document_text" {
    type = text
    null = false
  }

  column "document_hash" {
    type = varchar(64)
    null = false
  }

  column "embedding" {
    type = sql("real[]")
    null = false
  }

  column "embedding_model" {
    type = varchar(80)
    null = false
  }

  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  column "updated_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  primary_key {
    columns = [column.product_id]
  }

  foreign_key "product_search_embeddings_product_id_fkey" {
    columns     = [column.product_id]
    ref_columns = [table.products.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }

  index "idx_product_search_embeddings_model" {
    columns = [column.embedding_model]
  }

  check "product_search_embeddings_document_not_blank" {
    expr = "length(trim(document_text)) > 0"
  }

  check "product_search_embeddings_hash_valid" {
    expr = "length(document_hash) = 64"
  }

  check "product_search_embeddings_model_not_blank" {
    expr = "length(trim(embedding_model)) > 0"
  }

  check "product_search_embeddings_dimensions_valid" {
    expr = "cardinality(embedding) = 1024"
  }

  check "product_search_embeddings_no_null_values" {
    expr = "array_position(embedding, NULL) IS NULL"
  }
}



table "staff_accounts" {
  schema = schema.public

  column "id" {
    type    = uuid
    null    = false
    default = sql("gen_random_uuid()")
  }

  column "staff_code" {
    type = varchar(40)
    null = false
  }

  column "full_name" {
    type = varchar(160)
    null = false
  }

  column "email" {
    type = varchar(255)
    null = false
  }

  column "phone" {
    type = varchar(40)
    null = true
  }

  column "password_hash" {
    type = text
    null = false
  }

  column "status" {
    type    = varchar(20)
    null    = false
    default = "active"
  }

  column "last_login_at" {
    type = timestamptz
    null = true
  }

  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  column "updated_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  index "staff_accounts_staff_code_key" {
    unique  = true
    columns = [column.staff_code]
  }

  index "staff_accounts_email_key" {
    unique  = true
    columns = [column.email]
  }

  index "staff_accounts_phone_key" {
    unique  = true
    columns = [column.phone]
    where   = "phone IS NOT NULL"
  }

  index "idx_staff_accounts_status_created" {
    columns = [
      column.status,
      column.created_at,
    ]
  }

  check "staff_accounts_staff_code_not_blank" {
    expr = "length(trim(staff_code)) > 0"
  }

  check "staff_accounts_full_name_not_blank" {
    expr = "length(trim(full_name)) > 0"
  }

  check "staff_accounts_email_not_blank" {
    expr = "length(trim(email)) > 0"
  }

  check "staff_accounts_phone_not_blank" {
    expr = "phone IS NULL OR length(trim(phone)) > 0"
  }

  check "staff_accounts_password_hash_not_blank" {
    expr = "length(trim(password_hash)) > 0"
  }

  check "staff_accounts_status_valid" {
    expr = "status IN ('pending_activation', 'active', 'suspended', 'disabled', 'deleted', 'banned')"
  }
}

table "staff_invitations" {
  schema = schema.public

  column "id" {
    type    = uuid
    null    = false
    default = sql("gen_random_uuid()")
  }

  column "staff_account_id" {
    type = uuid
    null = false
  }

  # Snapshot of the bound staff email at invitation creation.
  # Activation must still require the one-time token; email alone
  # never proves ownership of the invitation.
  column "email" {
    type = varchar(255)
    null = false
  }

  # Plaintext token is returned only once for manual delivery.
  # Only the SHA-256 hash is persisted.
  column "token_hash" {
    type = varchar(64)
    null = false
  }

  # manual is enabled now.
  # email exists for future SMTP delivery but application code will
  # reject attempts to use it until email delivery is explicitly enabled.
  column "delivery_mode" {
    type    = varchar(20)
    null    = false
    default = "manual"
  }

  column "delivery_status" {
    type    = varchar(20)
    null    = false
    default = "not_requested"
  }

  column "status" {
    type    = varchar(20)
    null    = false
    default = "pending"
  }

  column "expires_at" {
    type = timestamptz
    null = false
  }

  column "password_set_at" {
    type = timestamptz
    null = true
  }

  column "accepted_at" {
    type = timestamptz
    null = true
  }

  column "cancelled_at" {
    type = timestamptz
    null = true
  }

  column "delivered_at" {
    type = timestamptz
    null = true
  }

  column "delivery_error" {
    type = text
    null = true
  }

  column "created_by_staff_id" {
    type = uuid
    null = false
  }

  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  column "updated_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  foreign_key "staff_invitations_staff_account_id_fkey" {
    columns     = [column.staff_account_id]
    ref_columns = [table.staff_accounts.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }

  foreign_key "staff_invitations_created_by_staff_id_fkey" {
    columns     = [column.created_by_staff_id]
    ref_columns = [table.staff_accounts.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }

  index "staff_invitations_token_hash_key" {
    unique  = true
    columns = [column.token_hash]
  }

  index "idx_staff_invitations_staff_status_created" {
    columns = [
      column.staff_account_id,
      column.status,
      column.created_at,
    ]
  }

  index "idx_staff_invitations_email_status" {
    columns = [
      column.email,
      column.status,
    ]
  }

  index "idx_staff_invitations_expiry" {
    columns = [column.expires_at]
    where   = "status = 'pending'"
  }

  index "idx_staff_invitations_creator_created" {
    columns = [
      column.created_by_staff_id,
      column.created_at,
    ]
  }

  check "staff_invitations_email_not_blank" {
    expr = "length(trim(email)) > 0"
  }

  check "staff_invitations_token_hash_valid" {
    expr = "length(token_hash) = 64"
  }

  check "staff_invitations_delivery_mode_valid" {
    expr = "delivery_mode IN ('manual', 'email')"
  }

  check "staff_invitations_delivery_status_valid" {
    expr = "delivery_status IN ('not_requested', 'pending', 'sent', 'failed')"
  }

  check "staff_invitations_status_valid" {
    expr = "status IN ('pending', 'accepted', 'cancelled', 'expired')"
  }

  check "staff_invitations_manual_delivery_state_valid" {
    expr = "delivery_mode <> 'manual' OR delivery_status = 'not_requested'"
  }

  check "staff_invitations_expiry_valid" {
    expr = "expires_at > created_at"
  }

  check "staff_invitations_password_time_valid" {
    expr = "password_set_at IS NULL OR password_set_at >= created_at"
  }

  check "staff_invitations_accepted_state_valid" {
    expr = "status <> 'accepted' OR accepted_at IS NOT NULL"
  }

  check "staff_invitations_cancelled_state_valid" {
    expr = "status <> 'cancelled' OR cancelled_at IS NOT NULL"
  }

  check "staff_invitations_terminal_state_exclusive" {
    expr = "accepted_at IS NULL OR cancelled_at IS NULL"
  }

  check "staff_invitations_delivery_error_valid" {
    expr = "delivery_error IS NULL OR length(trim(delivery_error)) > 0"
  }
}

table "account_bans" {
  schema = schema.public

  column "id" {
    type    = uuid
    null    = false
    default = sql("gen_random_uuid()")
  }

  # The ban system is shared by staff and customers while retaining
  # real foreign keys to both account tables.
  column "target_type" {
    type = varchar(20)
    null = false
  }

  column "staff_account_id" {
    type = uuid
    null = true
  }

  column "customer_id" {
    type = uuid
    null = true
  }

  # Staff bans currently support full-account enforcement only.
  #
  # Customer scopes allow narrower restrictions without disabling
  # the whole customer account.
  column "scope" {
    type = varchar(40)
    null = false
  }

  # Explicit rather than inferring permanence solely from expires_at.
  column "ban_type" {
    type = varchar(20)
    null = false
  }

  column "reason" {
    type = varchar(1000)
    null = false
  }

  # Stored only for full-account bans so an explicit unban or
  # temporary-ban expiry can restore the account to the state it
  # had immediately before the ban.
  column "previous_account_status" {
    type = varchar(20)
    null = true
  }

  column "starts_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  # Required for temporary bans and forbidden for permanent bans.
  column "expires_at" {
    type = timestamptz
    null = true
  }

  column "issued_by_staff_id" {
    type = uuid
    null = false
  }

  # Revocation is different from natural expiry.
  #
  # Expired temporary bans retain revoked_at = NULL. Enforcement
  # determines activity from starts_at/expires_at/revoked_at.
  column "revoked_at" {
    type = timestamptz
    null = true
  }

  column "revoked_by_staff_id" {
    type = uuid
    null = true
  }

  column "revocation_reason" {
    type = varchar(1000)
    null = true
  }

  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  column "updated_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  foreign_key "account_bans_staff_account_id_fkey" {
    columns     = [column.staff_account_id]
    ref_columns = [table.staff_accounts.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }

  foreign_key "account_bans_customer_id_fkey" {
    columns     = [column.customer_id]
    ref_columns = [table.customers.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }

  foreign_key "account_bans_issued_by_staff_id_fkey" {
    columns     = [column.issued_by_staff_id]
    ref_columns = [table.staff_accounts.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }

  foreign_key "account_bans_revoked_by_staff_id_fkey" {
    columns     = [column.revoked_by_staff_id]
    ref_columns = [table.staff_accounts.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }

  # Fast staff enforcement/history lookup.
  index "idx_account_bans_staff_scope_created" {
    columns = [
      column.staff_account_id,
      column.scope,
      column.created_at,
    ]

    where = "staff_account_id IS NOT NULL"
  }

  # Fast customer enforcement/history lookup.
  index "idx_account_bans_customer_scope_created" {
    columns = [
      column.customer_id,
      column.scope,
      column.created_at,
    ]

    where = "customer_id IS NOT NULL"
  }

  # Active-ban candidate lookup.
  #
  # expires_at is still evaluated by the application/query because
  # a PostgreSQL partial-index predicate should not depend on now().
  index "idx_account_bans_unrevoked_window" {
    columns = [
      column.target_type,
      column.scope,
      column.starts_at,
      column.expires_at,
    ]

    where = "revoked_at IS NULL"
  }

  index "idx_account_bans_issued_by_created" {
    columns = [
      column.issued_by_staff_id,
      column.created_at,
    ]
  }

  index "idx_account_bans_revoked_by" {
    columns = [
      column.revoked_by_staff_id,
      column.revoked_at,
    ]

    where = "revoked_by_staff_id IS NOT NULL"
  }

  check "account_bans_target_type_valid" {
    expr = "target_type IN ('staff', 'customer')"
  }

  # Exactly one real target must be present, and it must correspond
  # to target_type.
  check "account_bans_target_consistent" {
    expr = "(target_type = 'staff' AND staff_account_id IS NOT NULL AND customer_id IS NULL) OR (target_type = 'customer' AND customer_id IS NOT NULL AND staff_account_id IS NULL)"
  }

  check "account_bans_scope_valid" {
    expr = "scope IN ('full_account', 'purchasing', 'product_requests', 'support_messages', 'reviews', 'returns', 'promotions')"
  }

  # We are deliberately not creating partial staff-security bans.
  # Administrative staff bans are full-access security actions.
  check "account_bans_staff_scope_valid" {
    expr = "target_type <> 'staff' OR scope = 'full_account'"
  }

  check "account_bans_type_valid" {
    expr = "ban_type IN ('temporary', 'permanent')"
  }

  check "account_bans_reason_not_blank" {
    expr = "length(trim(reason)) > 0"
  }

  # Permanent bans never expire automatically.
  # Temporary bans always have an expiry later than their start.
  check "account_bans_duration_consistent" {
    expr = "(ban_type = 'permanent' AND expires_at IS NULL) OR (ban_type = 'temporary' AND expires_at IS NOT NULL AND expires_at > starts_at)"
  }

  # A previous account status is needed only when the ban changes
  # the account-level status to banned.
  check "account_bans_previous_status_required" {
    expr = "(scope = 'full_account' AND previous_account_status IS NOT NULL) OR (scope <> 'full_account' AND previous_account_status IS NULL)"
  }

  check "account_bans_previous_status_valid" {
    expr = "previous_account_status IS NULL OR (target_type = 'staff' AND previous_account_status IN ('active', 'suspended', 'disabled')) OR (target_type = 'customer' AND previous_account_status IN ('active', 'disabled'))"
  }

  # A revocation is complete and attributable or it does not exist.
  check "account_bans_revocation_consistent" {
    expr = "(revoked_at IS NULL AND revoked_by_staff_id IS NULL AND revocation_reason IS NULL) OR (revoked_at IS NOT NULL AND revoked_by_staff_id IS NOT NULL AND revocation_reason IS NOT NULL AND length(trim(revocation_reason)) > 0)"
  }

  check "account_bans_revocation_time_valid" {
    expr = "revoked_at IS NULL OR revoked_at >= created_at"
  }
}

table "staff_roles" {
  schema = schema.public

  column "id" {
    type    = uuid
    null    = false
    default = sql("gen_random_uuid()")
  }

  column "code" {
    type = varchar(80)
    null = false
  }

  column "name" {
    type = varchar(120)
    null = false
  }

  column "description" {
    type = varchar(500)
    null = true
  }

  column "is_system_role" {
    type    = boolean
    null    = false
    default = true
  }

  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  column "updated_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  index "staff_roles_code_key" {
    unique  = true
    columns = [column.code]
  }

  check "staff_roles_code_not_blank" {
    expr = "length(trim(code)) > 0"
  }

  check "staff_roles_name_not_blank" {
    expr = "length(trim(name)) > 0"
  }
}

table "staff_permissions" {
  schema = schema.public

  column "id" {
    type    = uuid
    null    = false
    default = sql("gen_random_uuid()")
  }

  column "code" {
    type = varchar(120)
    null = false
  }

  column "description" {
    type = varchar(500)
    null = true
  }

  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  index "staff_permissions_code_key" {
    unique  = true
    columns = [column.code]
  }

  check "staff_permissions_code_not_blank" {
    expr = "length(trim(code)) > 0"
  }
}

table "staff_account_roles" {
  schema = schema.public

  column "staff_account_id" {
    type = uuid
    null = false
  }

  column "role_id" {
    type = uuid
    null = false
  }

  column "assigned_by_staff_id" {
    type = uuid
    null = true
  }

  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  primary_key {
    columns = [
      column.staff_account_id,
      column.role_id,
    ]
  }

  foreign_key "staff_account_roles_staff_account_id_fkey" {
    columns     = [column.staff_account_id]
    ref_columns = [table.staff_accounts.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }

  foreign_key "staff_account_roles_role_id_fkey" {
    columns     = [column.role_id]
    ref_columns = [table.staff_roles.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }

  foreign_key "staff_account_roles_assigned_by_staff_id_fkey" {
    columns     = [column.assigned_by_staff_id]
    ref_columns = [table.staff_accounts.column.id]
    on_update   = NO_ACTION
    on_delete   = SET_NULL
  }

  index "idx_staff_account_roles_role" {
    columns = [column.role_id]
  }
}

table "staff_role_permissions" {
  schema = schema.public

  column "role_id" {
    type = uuid
    null = false
  }

  column "permission_id" {
    type = uuid
    null = false
  }

  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  primary_key {
    columns = [
      column.role_id,
      column.permission_id,
    ]
  }

  foreign_key "staff_role_permissions_role_id_fkey" {
    columns     = [column.role_id]
    ref_columns = [table.staff_roles.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }

  foreign_key "staff_role_permissions_permission_id_fkey" {
    columns     = [column.permission_id]
    ref_columns = [table.staff_permissions.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }

  index "idx_staff_role_permissions_permission" {
    columns = [column.permission_id]
  }
}

table "staff_sessions" {
  schema = schema.public

  column "id" {
    type    = uuid
    null    = false
    default = sql("gen_random_uuid()")
  }

  column "staff_account_id" {
    type = uuid
    null = false
  }

  column "access_token_hash" {
    type = varchar(64)
    null = false
  }

  column "refresh_token_hash" {
    type = varchar(64)
    null = false
  }

  column "access_expires_at" {
    type = timestamptz
    null = false
  }

  column "refresh_expires_at" {
    type = timestamptz
    null = false
  }

  column "last_used_at" {
    type = timestamptz
    null = true
  }

  column "revoked_at" {
    type = timestamptz
    null = true
  }

  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  column "updated_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  foreign_key "staff_sessions_staff_account_id_fkey" {
    columns     = [column.staff_account_id]
    ref_columns = [table.staff_accounts.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }

  index "staff_sessions_access_token_hash_key" {
    unique  = true
    columns = [column.access_token_hash]
  }

  index "staff_sessions_refresh_token_hash_key" {
    unique  = true
    columns = [column.refresh_token_hash]
  }

  index "idx_staff_sessions_account_created" {
    columns = [
      column.staff_account_id,
      column.created_at,
    ]
  }

  index "idx_staff_sessions_refresh_expiry" {
    columns = [column.refresh_expires_at]
    where   = "revoked_at IS NULL"
  }

  check "staff_sessions_access_token_hash_valid" {
    expr = "length(access_token_hash) = 64"
  }

  check "staff_sessions_refresh_token_hash_valid" {
    expr = "length(refresh_token_hash) = 64"
  }

  check "staff_sessions_expiry_valid" {
    expr = "refresh_expires_at > access_expires_at"
  }
}

table "admin_totp_credentials" {
  schema = schema.public

  column "id" {
    type    = uuid
    null    = false
    default = sql("gen_random_uuid()")
  }

  column "staff_account_id" {
    type = uuid
    null = false
  }

  column "label" {
    type    = varchar(120)
    null    = false
    default = "Authenticator"
  }

  # The TOTP secret must be recoverable by the API in order to verify
  # authenticator codes, so it is encrypted rather than hashed.
  column "secret_ciphertext" {
    type = text
    null = false
  }

  # Identifies which application encryption key produced the ciphertext.
  # This gives us a clean key-rotation path later.
  column "encryption_key_id" {
    type = varchar(80)
    null = false
  }

  column "algorithm" {
    type    = varchar(20)
    null    = false
    default = "SHA1"
  }

  column "digits" {
    type    = integer
    null    = false
    default = 6
  }

  column "period_seconds" {
    type    = integer
    null    = false
    default = 30
  }

  column "status" {
    type    = varchar(20)
    null    = false
    default = "pending"
  }

  # Prevents accepting the same TOTP time-step twice.
  column "last_accepted_step" {
    type = bigint
    null = true
  }

  column "verified_at" {
    type = timestamptz
    null = true
  }

  column "last_used_at" {
    type = timestamptz
    null = true
  }

  column "disabled_at" {
    type = timestamptz
    null = true
  }

  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  column "updated_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  foreign_key "admin_totp_credentials_staff_account_id_fkey" {
    columns     = [column.staff_account_id]
    ref_columns = [table.staff_accounts.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }

  index "admin_totp_credentials_staff_account_id_key" {
    unique  = true
    columns = [column.staff_account_id]
  }

  index "idx_admin_totp_credentials_status" {
    columns = [
      column.status,
      column.created_at,
    ]
  }

  check "admin_totp_credentials_label_not_blank" {
    expr = "length(trim(label)) > 0"
  }

  check "admin_totp_credentials_secret_not_blank" {
    expr = "length(trim(secret_ciphertext)) > 0"
  }

  check "admin_totp_credentials_key_id_not_blank" {
    expr = "length(trim(encryption_key_id)) > 0"
  }

  check "admin_totp_credentials_algorithm_valid" {
    expr = "algorithm IN ('SHA1', 'SHA256', 'SHA512')"
  }

  check "admin_totp_credentials_digits_valid" {
    expr = "digits IN (6, 8)"
  }

  check "admin_totp_credentials_period_valid" {
    expr = "period_seconds BETWEEN 15 AND 120"
  }

  check "admin_totp_credentials_status_valid" {
    expr = "status IN ('pending', 'active', 'disabled')"
  }

  check "admin_totp_credentials_active_requires_verification" {
    expr = "status <> 'active' OR verified_at IS NOT NULL"
  }

  check "admin_totp_credentials_disabled_requires_timestamp" {
    expr = "status <> 'disabled' OR disabled_at IS NOT NULL"
  }

  check "admin_totp_credentials_step_valid" {
    expr = "last_accepted_step IS NULL OR last_accepted_step >= 0"
  }
}

table "admin_mfa_recovery_codes" {
  schema = schema.public

  column "id" {
    type    = uuid
    null    = false
    default = sql("gen_random_uuid()")
  }

  column "staff_account_id" {
    type = uuid
    null = false
  }

  # Recovery codes are generated with high entropy, so unlike passwords
  # they can safely be stored as one-way SHA-256 lookup hashes.
  column "code_hash" {
    type = varchar(64)
    null = false
  }

  column "used_at" {
    type = timestamptz
    null = true
  }

  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  foreign_key "admin_mfa_recovery_codes_staff_account_id_fkey" {
    columns     = [column.staff_account_id]
    ref_columns = [table.staff_accounts.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }

  index "admin_mfa_recovery_codes_code_hash_key" {
    unique  = true
    columns = [column.code_hash]
  }

  index "idx_admin_mfa_recovery_codes_account_used" {
    columns = [
      column.staff_account_id,
      column.used_at,
    ]
  }

  check "admin_mfa_recovery_codes_hash_valid" {
    expr = "length(code_hash) = 64"
  }
}

table "admin_login_challenges" {
  schema = schema.public

  column "id" {
    type    = uuid
    null    = false
    default = sql("gen_random_uuid()")
  }

  column "staff_account_id" {
    type = uuid
    null = false
  }

  # The browser receives the plaintext challenge token.
  # Only its SHA-256 hash is persisted.
  column "challenge_token_hash" {
    type = varchar(64)
    null = false
  }

  column "purpose" {
    type    = varchar(20)
    null    = false
    default = "login"
  }

  column "status" {
    type    = varchar(20)
    null    = false
    default = "pending"
  }

  column "failed_attempts" {
    type    = integer
    null    = false
    default = 0
  }

  column "max_attempts" {
    type    = integer
    null    = false
    default = 5
  }

  column "expires_at" {
    type = timestamptz
    null = false
  }

  column "verified_at" {
    type = timestamptz
    null = true
  }

  column "consumed_at" {
    type = timestamptz
    null = true
  }

  column "client_ip" {
    type = varchar(64)
    null = true
  }

  column "user_agent" {
    type = varchar(500)
    null = true
  }

  column "request_id" {
    type = varchar(128)
    null = true
  }

  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  column "updated_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  foreign_key "admin_login_challenges_staff_account_id_fkey" {
    columns     = [column.staff_account_id]
    ref_columns = [table.staff_accounts.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }

  index "admin_login_challenges_token_hash_key" {
    unique  = true
    columns = [column.challenge_token_hash]
  }

  index "idx_admin_login_challenges_account_status" {
    columns = [
      column.staff_account_id,
      column.status,
      column.created_at,
    ]
  }

  index "idx_admin_login_challenges_expiry" {
    columns = [
      column.status,
      column.expires_at,
    ]
  }

  check "admin_login_challenges_hash_valid" {
    expr = "length(challenge_token_hash) = 64"
  }

  check "admin_login_challenges_purpose_valid" {
    expr = "purpose IN ('login', 'reauth')"
  }

  check "admin_login_challenges_status_valid" {
    expr = "status IN ('pending', 'verified', 'consumed', 'expired', 'locked', 'cancelled')"
  }

  check "admin_login_challenges_attempts_valid" {
    expr = "failed_attempts >= 0 AND max_attempts > 0 AND failed_attempts <= max_attempts"
  }

  check "admin_login_challenges_expiry_valid" {
    expr = "expires_at > created_at"
  }

  check "admin_login_challenges_verified_valid" {
    expr = "status <> 'verified' OR verified_at IS NOT NULL"
  }

  check "admin_login_challenges_consumed_valid" {
    expr = "status <> 'consumed' OR consumed_at IS NOT NULL"
  }
}

table "admin_sessions" {
  schema = schema.public

  column "id" {
    type    = uuid
    null    = false
    default = sql("gen_random_uuid()")
  }

  column "staff_account_id" {
    type = uuid
    null = false
  }

  column "login_challenge_id" {
    type = uuid
    null = true
  }

  column "access_token_hash" {
    type = varchar(64)
    null = false
  }

  column "refresh_token_hash" {
    type = varchar(64)
    null = false
  }

  column "csrf_token_hash" {
    type = varchar(64)
    null = false
  }

  column "refresh_generation" {
    type    = integer
    null    = false
    default = 0
  }

  column "access_expires_at" {
    type = timestamptz
    null = false
  }

  column "refresh_expires_at" {
    type = timestamptz
    null = false
  }

  column "authenticated_at" {
    type = timestamptz
    null = false
  }

  column "mfa_verified_at" {
    type = timestamptz
    null = false
  }

  column "last_used_at" {
    type = timestamptz
    null = true
  }

  column "last_rotated_at" {
    type = timestamptz
    null = true
  }

  column "created_ip" {
    type = varchar(64)
    null = true
  }

  column "last_ip" {
    type = varchar(64)
    null = true
  }

  column "user_agent" {
    type = varchar(500)
    null = true
  }

  column "revoked_at" {
    type = timestamptz
    null = true
  }

  column "revoke_reason" {
    type = varchar(250)
    null = true
  }

  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  column "updated_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  foreign_key "admin_sessions_staff_account_id_fkey" {
    columns     = [column.staff_account_id]
    ref_columns = [table.staff_accounts.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }

  foreign_key "admin_sessions_login_challenge_id_fkey" {
    columns     = [column.login_challenge_id]
    ref_columns = [table.admin_login_challenges.column.id]
    on_update   = NO_ACTION
    on_delete   = SET_NULL
  }

  index "admin_sessions_access_token_hash_key" {
    unique  = true
    columns = [column.access_token_hash]
  }

  index "admin_sessions_refresh_token_hash_key" {
    unique  = true
    columns = [column.refresh_token_hash]
  }

  index "admin_sessions_csrf_token_hash_key" {
    unique  = true
    columns = [column.csrf_token_hash]
  }

  # One completed login challenge may establish at most one Admin session.
  index "admin_sessions_login_challenge_id_key" {
    unique  = true
    columns = [column.login_challenge_id]
  }

  index "idx_admin_sessions_account_created" {
    columns = [
      column.staff_account_id,
      column.created_at,
    ]
  }

  index "idx_admin_sessions_refresh_expiry" {
    columns = [column.refresh_expires_at]
    where   = "revoked_at IS NULL"
  }

  check "admin_sessions_access_hash_valid" {
    expr = "length(access_token_hash) = 64"
  }

  check "admin_sessions_refresh_hash_valid" {
    expr = "length(refresh_token_hash) = 64"
  }

  check "admin_sessions_csrf_hash_valid" {
    expr = "length(csrf_token_hash) = 64"
  }

  check "admin_sessions_refresh_generation_valid" {
    expr = "refresh_generation >= 0"
  }

  check "admin_sessions_expiry_valid" {
    expr = "refresh_expires_at > access_expires_at"
  }

  check "admin_sessions_revoke_reason_valid" {
    expr = "revoke_reason IS NULL OR revoked_at IS NOT NULL"
  }
}


table "admin_security_events" {
  schema = schema.public

  column "id" {
    type    = uuid
    null    = false
    default = sql("gen_random_uuid()")
  }

  column "staff_account_id" {
    type = uuid
    null = true
  }

  column "admin_session_id" {
    type = uuid
    null = true
  }

  column "event_type" {
    type = varchar(80)
    null = false
  }

  column "outcome" {
    type = varchar(20)
    null = false
  }

  # Allows correlation of failed logins for unknown identifiers without
  # persisting the plaintext email/staff code.
  column "identifier_hash" {
    type = varchar(64)
    null = true
  }

  column "request_id" {
    type = varchar(128)
    null = true
  }

  column "ip_address" {
    type = varchar(64)
    null = true
  }

  column "user_agent" {
    type = varchar(500)
    null = true
  }

  column "details" {
    type = jsonb
    null = true
  }

  column "occurred_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  foreign_key "admin_security_events_staff_account_id_fkey" {
    columns     = [column.staff_account_id]
    ref_columns = [table.staff_accounts.column.id]
    on_update   = NO_ACTION
    on_delete   = SET_NULL
  }

  foreign_key "admin_security_events_admin_session_id_fkey" {
    columns     = [column.admin_session_id]
    ref_columns = [table.admin_sessions.column.id]
    on_update   = NO_ACTION
    on_delete   = SET_NULL
  }

  index "idx_admin_security_events_staff_time" {
    columns = [
      column.staff_account_id,
      column.occurred_at,
    ]
  }

  index "idx_admin_security_events_session_time" {
    columns = [
      column.admin_session_id,
      column.occurred_at,
    ]
  }

  index "idx_admin_security_events_type_time" {
    columns = [
      column.event_type,
      column.occurred_at,
    ]
  }

  check "admin_security_events_type_not_blank" {
    expr = "length(trim(event_type)) > 0"
  }

  check "admin_security_events_outcome_valid" {
    expr = "outcome IN ('success', 'failure', 'blocked', 'info')"
  }

  check "admin_security_events_identifier_hash_valid" {
    expr = "identifier_hash IS NULL OR length(identifier_hash) = 64"
  }
}



table "support_actors" {
  schema = schema.public

  column "id" {
    type    = uuid
    null    = false
    default = sql("gen_random_uuid()")
  }

  column "actor_code" {
    type = varchar(40)
    null = false
  }

  column "actor_type" {
    type = varchar(20)
    null = false
  }

  column "staff_account_id" {
    type = uuid
    null = true
  }

  column "display_name" {
    type = varchar(160)
    null = false
  }

  column "status" {
    type    = varchar(20)
    null    = false
    default = "active"
  }

  column "presence" {
    type    = varchar(20)
    null    = false
    default = "offline"
  }

  column "max_active_cases" {
    type    = integer
    null    = false
    default = 10
  }

  column "last_presence_at" {
    type = timestamptz
    null = true
  }

  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  column "updated_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  foreign_key "support_actors_staff_account_id_fkey" {
    columns     = [column.staff_account_id]
    ref_columns = [table.staff_accounts.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }

  index "support_actors_actor_code_key" {
    unique  = true
    columns = [column.actor_code]
  }

  index "support_actors_staff_account_id_key" {
    unique  = true
    columns = [column.staff_account_id]
    where   = "staff_account_id IS NOT NULL"
  }

  index "idx_support_actors_presence" {
    columns = [
      column.status,
      column.presence,
    ]
  }

  check "support_actors_actor_code_not_blank" {
    expr = "length(trim(actor_code)) > 0"
  }

  check "support_actors_display_name_not_blank" {
    expr = "length(trim(display_name)) > 0"
  }

  check "support_actors_actor_type_valid" {
    expr = "actor_type IN ('human', 'ai', 'system')"
  }

  check "support_actors_status_valid" {
    expr = "status IN ('active', 'disabled')"
  }

  check "support_actors_presence_valid" {
    expr = "presence IN ('offline', 'available', 'busy', 'away')"
  }

  check "support_actors_max_active_cases_positive" {
    expr = "max_active_cases > 0"
  }

  check "support_actors_human_staff_consistent" {
    expr = "(actor_type = 'human' AND staff_account_id IS NOT NULL) OR (actor_type <> 'human' AND staff_account_id IS NULL)"
  }
}

table "support_queues" {
  schema = schema.public

  column "id" {
    type    = uuid
    null    = false
    default = sql("gen_random_uuid()")
  }

  column "code" {
    type = varchar(80)
    null = false
  }

  column "name" {
    type = varchar(120)
    null = false
  }

  column "description" {
    type = varchar(500)
    null = true
  }

  column "status" {
    type    = varchar(20)
    null    = false
    default = "active"
  }

  column "sort_order" {
    type    = integer
    null    = false
    default = 0
  }

  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  column "updated_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  index "support_queues_code_key" {
    unique  = true
    columns = [column.code]
  }

  index "idx_support_queues_status_sort" {
    columns = [
      column.status,
      column.sort_order,
    ]
  }

  check "support_queues_code_not_blank" {
    expr = "length(trim(code)) > 0"
  }

  check "support_queues_name_not_blank" {
    expr = "length(trim(name)) > 0"
  }

  check "support_queues_status_valid" {
    expr = "status IN ('active', 'disabled')"
  }

  check "support_queues_sort_order_nonnegative" {
    expr = "sort_order >= 0"
  }
}

table "support_queue_members" {
  schema = schema.public

  column "queue_id" {
    type = uuid
    null = false
  }

  column "support_actor_id" {
    type = uuid
    null = false
  }

  column "membership_role" {
    type    = varchar(20)
    null    = false
    default = "member"
  }

  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  primary_key {
    columns = [
      column.queue_id,
      column.support_actor_id,
    ]
  }

  foreign_key "support_queue_members_queue_id_fkey" {
    columns     = [column.queue_id]
    ref_columns = [table.support_queues.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }

  foreign_key "support_queue_members_support_actor_id_fkey" {
    columns     = [column.support_actor_id]
    ref_columns = [table.support_actors.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }

  index "idx_support_queue_members_actor" {
    columns = [column.support_actor_id]
  }

  check "support_queue_members_role_valid" {
    expr = "membership_role IN ('member', 'lead')"
  }
}


table "crm_cases" {
  schema = schema.public

  column "id" {
    type    = uuid
    null    = false
    default = sql("gen_random_uuid()")
  }

  column "case_number" {
    type = varchar(60)
    null = false
  }

  column "customer_id" {
    type = uuid
    null = false
  }

  column "case_type" {
    type = varchar(40)
    null = false
  }

  column "subject" {
    type = varchar(180)
    null = false
  }

  column "status" {
    type    = varchar(30)
    null    = false
    default = "waiting_support"
  }

  column "priority" {
    type    = varchar(20)
    null    = false
    default = "normal"
  }

  column "product_id" {
    type = uuid
    null = true
  }

  column "variant_id" {
    type = uuid
    null = true
  }

  column "order_id" {
    type = uuid
    null = true
  }

  column "requested_quantity" {
    type = integer
    null = true
  }

  column "available_quantity_snapshot" {
    type = integer
    null = true
  }

  column "context_snapshot" {
    type    = jsonb
    null    = false
    default = sql("'{}'::jsonb")
  }

  column "last_message_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  column "last_customer_message_at" {
    type = timestamptz
    null = true
  }

  column "last_support_message_at" {
    type = timestamptz
    null = true
  }

  column "resolved_at" {
    type = timestamptz
    null = true
  }

  column "closed_at" {
    type = timestamptz
    null = true
  }

  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  column "updated_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  foreign_key "crm_cases_customer_id_fkey" {
    columns     = [column.customer_id]
    ref_columns = [table.customers.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }

  foreign_key "crm_cases_product_id_fkey" {
    columns     = [column.product_id]
    ref_columns = [table.products.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }

  foreign_key "crm_cases_variant_id_fkey" {
    columns     = [column.variant_id]
    ref_columns = [table.product_variants.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }

  foreign_key "crm_cases_order_id_fkey" {
    columns     = [column.order_id]
    ref_columns = [table.orders.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }

  index "crm_cases_case_number_key" {
    unique  = true
    columns = [column.case_number]
  }

  index "idx_crm_cases_customer_updated" {
    columns = [
      column.customer_id,
      column.updated_at,
    ]
  }

  index "idx_crm_cases_customer_status" {
    columns = [
      column.customer_id,
      column.status,
      column.updated_at,
    ]
  }

  index "idx_crm_cases_support_inbox" {
    columns = [
      column.status,
      column.priority,
      column.last_message_at,
    ]
  }

  index "idx_crm_cases_product" {
    columns = [column.product_id]
  }

  index "idx_crm_cases_variant" {
    columns = [column.variant_id]
  }

  index "idx_crm_cases_order" {
    columns = [column.order_id]
  }

  check "crm_cases_case_number_not_blank" {
    expr = "length(trim(case_number)) > 0"
  }

  check "crm_cases_subject_not_blank" {
    expr = "length(trim(subject)) > 0"
  }

  check "crm_cases_type_valid" {
    expr = "case_type IN ('bulk_stock_request', 'product_request', 'product_question', 'order_issue', 'payment_issue', 'return_issue', 'delivery_issue', 'complaint', 'general_question', 'other')"
  }

  check "crm_cases_status_valid" {
    expr = "status IN ('waiting_support', 'waiting_customer', 'resolved', 'closed')"
  }

  check "crm_cases_priority_valid" {
    expr = "priority IN ('low', 'normal', 'high', 'urgent')"
  }

  check "crm_cases_requested_quantity_positive" {
    expr = "requested_quantity IS NULL OR requested_quantity > 0"
  }

  check "crm_cases_available_quantity_nonnegative" {
    expr = "available_quantity_snapshot IS NULL OR available_quantity_snapshot >= 0"
  }

  check "crm_cases_bulk_context_valid" {
    expr = "case_type <> 'bulk_stock_request' OR (product_id IS NOT NULL AND variant_id IS NOT NULL AND requested_quantity IS NOT NULL AND available_quantity_snapshot IS NOT NULL)"
  }

  check "crm_cases_resolved_timestamp_valid" {
    expr = "status NOT IN ('resolved', 'closed') OR resolved_at IS NOT NULL"
  }

  check "crm_cases_closed_timestamp_valid" {
    expr = "status <> 'closed' OR closed_at IS NOT NULL"
  }
}

table "crm_messages" {
  schema = schema.public

  column "id" {
    type    = uuid
    null    = false
    default = sql("gen_random_uuid()")
  }

  column "case_id" {
    type = uuid
    null = false
  }

  column "author_type" {
    type = varchar(20)
    null = false
  }

  column "support_actor_id" {
    type = uuid
    null = true
  }

  column "visibility" {
    type    = varchar(20)
    null    = false
    default = "customer"
  }

  column "body" {
    type = text
    null = false
  }

  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  column "attachments" {
  type = jsonb
  null = true
}

  primary_key {
    columns = [column.id]
  }

  foreign_key "crm_messages_case_id_fkey" {
    columns     = [column.case_id]
    ref_columns = [table.crm_cases.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }

  foreign_key "crm_messages_support_actor_id_fkey" {
    columns     = [column.support_actor_id]
    ref_columns = [table.support_actors.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }

  index "idx_crm_messages_case_created" {
    columns = [
      column.case_id,
      column.created_at,
    ]
  }

  index "idx_crm_messages_support_actor" {
    columns = [column.support_actor_id]
  }
  check "crm_messages_attachments_is_array" {
  expr = "attachments IS NULL OR jsonb_typeof(attachments) = 'array'"
}

  check "crm_messages_author_type_valid" {
    expr = "author_type IN ('customer', 'support')"
  }

  check "crm_messages_visibility_valid" {
    expr = "visibility IN ('customer', 'internal')"
  }

  check "crm_messages_body_not_blank" {
    expr = "length(trim(body)) > 0"
  }

  check "crm_messages_author_consistent" {
    expr = "(author_type = 'customer' AND support_actor_id IS NULL AND visibility = 'customer') OR (author_type = 'support' AND support_actor_id IS NOT NULL)"
  }
}


table "crm_support_attachments" {
  schema = schema.public

  column "id" {
    type    = uuid
    null    = false
    default = sql("gen_random_uuid()")
  }

  column "case_id" {
    type = uuid
    null = true
  }

  column "message_id" {
    type = uuid
    null = true
  }

  column "uploader_type" {
    type = varchar(20)
    null = false
  }

  column "customer_id" {
    type = uuid
    null = true
  }

  column "support_actor_id" {
    type = uuid
    null = true
  }

  column "storage_key" {
    type = text
    null = false
  }

  column "original_filename" {
    type = varchar(255)
    null = false
  }

  column "mime_type" {
    type = varchar(100)
    null = false
  }

  column "byte_size" {
    type = bigint
    null = false
  }

  column "status" {
    type    = varchar(20)
    null    = false
    default = "pending"
  }

  column "uploaded_at" {
    type = timestamptz
    null = true
  }

  column "deleted_at" {
    type = timestamptz
    null = true
  }

  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  column "updated_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  foreign_key "crm_support_attachments_case_id_fkey" {
    columns     = [column.case_id]
    ref_columns = [table.crm_cases.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }

  foreign_key "crm_support_attachments_message_id_fkey" {
    columns     = [column.message_id]
    ref_columns = [table.crm_messages.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }

  foreign_key "crm_support_attachments_customer_id_fkey" {
    columns     = [column.customer_id]
    ref_columns = [table.customers.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }

  foreign_key "crm_support_attachments_support_actor_id_fkey" {
    columns     = [column.support_actor_id]
    ref_columns = [table.support_actors.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }

  index "crm_support_attachments_storage_key_key" {
    unique  = true
    columns = [column.storage_key]
  }

  index "idx_crm_support_attachments_case_status" {
    columns = [
      column.case_id,
      column.status,
    ]
  }

  index "idx_crm_support_attachments_message" {
    columns = [column.message_id]
  }

  index "idx_crm_support_attachments_status_created" {
    columns = [
      column.status,
      column.created_at,
    ]
  }

  check "crm_support_attachments_uploader_type_valid" {
    expr = "uploader_type IN ('customer', 'support')"
  }

  check "crm_support_attachments_uploader_consistent" {
    expr = "(uploader_type = 'customer' AND customer_id IS NOT NULL AND support_actor_id IS NULL) OR (uploader_type = 'support' AND customer_id IS NULL AND support_actor_id IS NOT NULL)"
  }

  check "crm_support_attachments_storage_key_valid" {
    expr = "storage_key LIKE 'private/support/%' AND length(storage_key) > length('private/support/')"
  }

  check "crm_support_attachments_original_filename_not_blank" {
    expr = "length(trim(original_filename)) > 0"
  }

  check "crm_support_attachments_mime_type_valid" {
    expr = "mime_type IN ('image/jpeg', 'image/png', 'image/webp', 'image/gif')"
  }

  check "crm_support_attachments_byte_size_positive" {
    expr = "byte_size > 0"
  }

  check "crm_support_attachments_status_valid" {
    expr = "status IN ('pending', 'ready', 'attached', 'deleted')"
  }

  check "crm_support_attachments_message_requires_case" {
    expr = "message_id IS NULL OR case_id IS NOT NULL"
  }

  check "crm_support_attachments_state_consistent" {
    expr = "(status = 'pending' AND message_id IS NULL AND uploaded_at IS NULL AND deleted_at IS NULL) OR (status = 'ready' AND message_id IS NULL AND uploaded_at IS NOT NULL AND deleted_at IS NULL) OR (status = 'attached' AND case_id IS NOT NULL AND message_id IS NOT NULL AND uploaded_at IS NOT NULL AND deleted_at IS NULL) OR (status = 'deleted' AND deleted_at IS NOT NULL)"
  }
}

table "crm_case_events" {
  schema = schema.public

  column "id" {
    type    = uuid
    null    = false
    default = sql("gen_random_uuid()")
  }

  column "case_id" {
    type = uuid
    null = false
  }

  column "event_type" {
    type = varchar(80)
    null = false
  }

  column "actor_type" {
    type = varchar(20)
    null = false
  }

  column "support_actor_id" {
    type = uuid
    null = true
  }

  column "from_status" {
    type = varchar(30)
    null = true
  }

  column "to_status" {
    type = varchar(30)
    null = true
  }

  column "payload" {
    type    = jsonb
    null    = false
    default = sql("'{}'::jsonb")
  }

  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  foreign_key "crm_case_events_case_id_fkey" {
    columns     = [column.case_id]
    ref_columns = [table.crm_cases.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }

  foreign_key "crm_case_events_support_actor_id_fkey" {
    columns     = [column.support_actor_id]
    ref_columns = [table.support_actors.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }

  index "idx_crm_case_events_case_created" {
    columns = [
      column.case_id,
      column.created_at,
    ]
  }

  check "crm_case_events_event_type_not_blank" {
    expr = "length(trim(event_type)) > 0"
  }

  check "crm_case_events_actor_type_valid" {
    expr = "actor_type IN ('customer', 'support')"
  }

  check "crm_case_events_actor_consistent" {
    expr = "(actor_type = 'customer' AND support_actor_id IS NULL) OR (actor_type = 'support' AND support_actor_id IS NOT NULL)"
  }

  check "crm_case_events_from_status_valid" {
    expr = "from_status IS NULL OR from_status IN ('waiting_support', 'waiting_customer', 'resolved', 'closed')"
  }

  check "crm_case_events_to_status_valid" {
    expr = "to_status IS NULL OR to_status IN ('waiting_support', 'waiting_customer', 'resolved', 'closed')"
  }
}


table "support_case_assignments" {
  schema = schema.public

  column "id" {
    type    = uuid
    null    = false
    default = sql("gen_random_uuid()")
  }

  column "case_id" {
    type = uuid
    null = false
  }

  column "queue_id" {
    type = uuid
    null = false
  }

  column "support_actor_id" {
    type = uuid
    null = true
  }

  column "assigned_by_actor_id" {
    type = uuid
    null = true
  }

  column "assignment_type" {
    type = varchar(20)
    null = false
  }

  column "assigned_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  column "accepted_at" {
    type = timestamptz
    null = true
  }

  column "released_at" {
    type = timestamptz
    null = true
  }

  column "release_reason" {
    type = varchar(300)
    null = true
  }

  primary_key {
    columns = [column.id]
  }

  foreign_key "support_case_assignments_case_id_fkey" {
    columns     = [column.case_id]
    ref_columns = [table.crm_cases.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }

  foreign_key "support_case_assignments_queue_id_fkey" {
    columns     = [column.queue_id]
    ref_columns = [table.support_queues.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }

  foreign_key "support_case_assignments_support_actor_id_fkey" {
    columns     = [column.support_actor_id]
    ref_columns = [table.support_actors.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }

  foreign_key "support_case_assignments_assigned_by_actor_id_fkey" {
    columns     = [column.assigned_by_actor_id]
    ref_columns = [table.support_actors.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }

  index "support_case_assignments_active_case_key" {
    unique  = true
    columns = [column.case_id]
    where   = "released_at IS NULL"
  }

  index "idx_support_case_assignments_queue_active" {
    columns = [
      column.queue_id,
      column.assigned_at,
    ]
    where = "released_at IS NULL"
  }

  index "idx_support_case_assignments_actor_active" {
    columns = [
      column.support_actor_id,
      column.assigned_at,
    ]
    where = "support_actor_id IS NOT NULL AND released_at IS NULL"
  }

  index "idx_support_case_assignments_case_history" {
    columns = [
      column.case_id,
      column.assigned_at,
    ]
  }

  check "support_case_assignments_type_valid" {
    expr = "assignment_type IN ('automatic', 'manual', 'claimed', 'escalated')"
  }

  check "support_case_assignments_release_reason_valid" {
    expr = "release_reason IS NULL OR length(trim(release_reason)) > 0"
  }

  check "support_case_assignments_times_valid" {
    expr = "(accepted_at IS NULL OR accepted_at >= assigned_at) AND (released_at IS NULL OR released_at >= assigned_at)"
  }
}

table "product_sourcing_requests" {
  schema = schema.public

  column "id" {
    type    = uuid
    null    = false
    default = sql("gen_random_uuid()")
  }

  column "case_id" {
    type = uuid
    null = false
  }

  column "request_number" {
    type = varchar(60)
    null = false
  }

  column "requested_product_name" {
    type = varchar(180)
    null = false
  }

  column "description" {
    type = text
    null = false
  }

  column "customer_requirements" {
    type = jsonb
    null = true
  }

  column "external_url" {
    type = text
    null = true
  }

  column "attachments" {
    type = jsonb
    null = true
  }

  column "status" {
    type    = varchar(30)
    null    = false
    default = "pending_review"
  }

  column "status_reason" {
    type = text
    null = true
  }

  column "reviewed_by" {
    type = uuid
    null = true
  }

  column "reviewed_at" {
    type = timestamptz
    null = true
  }

  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  column "updated_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  foreign_key "product_sourcing_requests_case_id_fkey" {
    columns = [column.case_id]

    ref_columns = [
      table.crm_cases.column.id
    ]

    on_update = NO_ACTION
    on_delete = CASCADE
  }

  foreign_key "product_sourcing_requests_reviewed_by_fkey" {
    columns = [column.reviewed_by]

    ref_columns = [
      table.support_actors.column.id
    ]

    on_update = NO_ACTION
    on_delete = SET_NULL
  }

  index "product_sourcing_requests_request_number_key" {
    unique = true
    columns = [column.request_number]
  }

  index "product_sourcing_requests_case_key" {
    unique = true
    columns = [column.case_id]
  }

  index "idx_product_sourcing_requests_status_updated" {
    columns = [
      column.status,
      column.updated_at,
    ]
  }

  index "idx_product_sourcing_requests_reviewed_by" {
    columns = [column.reviewed_by]
  }

  check "product_sourcing_requests_name_not_blank" {
    expr = "length(trim(requested_product_name)) > 0"
  }

  check "product_sourcing_requests_description_not_blank" {
    expr = "length(trim(description)) > 0"
  }

  check "product_sourcing_requests_customer_requirements_is_object" {
    expr = "customer_requirements IS NULL OR jsonb_typeof(customer_requirements) = 'object'"
  }

  check "product_sourcing_requests_attachments_is_array" {
    expr = "attachments IS NULL OR jsonb_typeof(attachments) = 'array'"
  }

  check "product_sourcing_requests_status_valid" {
  expr = "status IN ('pending_review', 'on_hold', 'accepted', 'negotiating', 'agreed', 'converted_to_order', 'cancelled')"
}

  check "product_sourcing_requests_status_reason_not_blank" {
  expr = "status_reason IS NULL OR length(trim(status_reason)) > 0"
}

check "product_sourcing_requests_status_reason_required" {
  expr = "status NOT IN ('on_hold', 'cancelled') OR status_reason IS NOT NULL"
}
}

table "product_sourcing_offers" {
  schema = schema.public

  column "id" {
    type    = uuid
    null    = false
    default = sql("gen_random_uuid()")
  }

  column "request_id" {
    type = uuid
    null = false
  }

  column "status" {
    type    = varchar(30)
    null    = false
    default = "draft"
  }

  column "product_name" {
    type = varchar(180)
    null = false
  }

  column "description" {
    type = text
    null = false
  }

  column "offered_specifications" {
    type = jsonb
    null = true
  }

  column "attachments" {
    type = jsonb
    null = true
  }

  column "quoted_quantity" {
    type = integer
    null = true
  }

  column "minimum_order_quantity" {
    type = integer
    null = true
  }

  column "unit_price" {
    type = bigint
    null = false
  }

  column "shipping_price" {
    type    = bigint
    null    = false
    default = 0
  }

  column "currency" {
    type = varchar(10)
    null = false
  }

  column "expires_at" {
    type = timestamptz
    null = true
  }

  column "sent_at" {
    type = timestamptz
    null = true
  }

  column "customer_responded_at" {
    type = timestamptz
    null = true
  }

  column "finalized_at" {
    type = timestamptz
    null = true
  }

  column "created_by" {
    type = uuid
    null = false
  }

  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  column "updated_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  foreign_key "product_sourcing_offers_request_id_fkey" {
    columns = [column.request_id]

    ref_columns = [
      table.product_sourcing_requests.column.id
    ]

    on_update = NO_ACTION
    on_delete = CASCADE
  }

  foreign_key "product_sourcing_offers_created_by_fkey" {
    columns = [column.created_by]

    ref_columns = [
      table.support_actors.column.id
    ]

    on_update = NO_ACTION
    on_delete = RESTRICT
  }

  index "idx_product_sourcing_offers_request_id" {
    columns = [
      column.request_id,
      column.created_at,
    ]
  }

  index "idx_product_sourcing_offers_status" {
    columns = [column.status]
  }

  index "product_sourcing_offers_one_finalized_per_request" {
    unique = true
    columns = [column.request_id]
    where = "status = 'finalized'"
  }

  check "product_sourcing_offers_status_valid" {
    expr = "status IN ('draft', 'sent', 'accepted', 'rejected', 'customer_accepted', 'customer_rejected', 'superseded', 'expired', 'finalized')"
  }

  check "product_sourcing_offers_offered_specifications_is_object" {
    expr = "offered_specifications IS NULL OR jsonb_typeof(offered_specifications) = 'object'"
  }

  check "product_sourcing_offers_attachments_is_array" {
    expr = "attachments IS NULL OR jsonb_typeof(attachments) = 'array'"
  }

  check "product_sourcing_offers_description_not_blank" {
    expr = "length(trim(description)) > 0"
  }

  check "product_sourcing_offers_product_name_not_blank" {
    expr = "length(trim(product_name)) > 0"
  }

  check "product_sourcing_offers_quoted_quantity_positive" {
    expr = "quoted_quantity IS NULL OR quoted_quantity > 0"
  }

  check "product_sourcing_offers_minimum_order_quantity_positive" {
    expr = "minimum_order_quantity IS NULL OR minimum_order_quantity > 0"
  }

  check "product_sourcing_offers_quantity_meets_minimum" {
    expr = "quoted_quantity IS NULL OR minimum_order_quantity IS NULL OR quoted_quantity >= minimum_order_quantity"
  }

  check "product_sourcing_offers_unit_price_nonnegative" {
    expr = "unit_price >= 0"
  }

  check "product_sourcing_offers_shipping_price_nonnegative" {
    expr = "shipping_price >= 0"
  }

  check "product_sourcing_offers_currency_valid" {
    expr = "length(currency) = 3 AND currency = upper(currency)"
  }

  check "product_sourcing_offers_expiry_valid" {
    expr = "expires_at IS NULL OR expires_at > created_at"
  }
}

table "product_sourcing_confirmations" {
  schema = schema.public

  column "id" {
    type    = uuid
    null    = false
    default = sql("gen_random_uuid()")
  }

  column "request_id" {
    type = uuid
    null = false
  }

  column "offer_id" {
    type = uuid
    null = false
  }

  column "quantity" {
    type = integer
    null = false
  }

  column "minimum_order_quantity" {
    type = integer
    null = false
  }

  column "accepted_product_name" {
    type = varchar(180)
    null = false
  }

  column "accepted_specifications" {
    type = jsonb
    null = true
  }

  column "unit_price_snapshot" {
    type = bigint
    null = false
  }

  column "shipping_price_snapshot" {
    type = bigint
    null = false
  }

  column "currency" {
    type = varchar(3)
    null = false
  }

  column "total_amount" {
    type = bigint
    null = false
  }

  column "status" {
    type    = varchar(30)
    null    = false
    default = "confirmed"
  }

  column "finalized_by" {
    type = uuid
    null = false
  }

  column "created_product_id" {
    type = uuid
    null = true
  }

  column "created_variant_id" {
    type = uuid
    null = true
  }

  column "created_order_id" {
    type = uuid
    null = true
  }

  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  column "updated_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  foreign_key "product_sourcing_confirmations_request_id_fkey" {
    columns = [column.request_id]

    ref_columns = [
      table.product_sourcing_requests.column.id
    ]

    on_update = NO_ACTION
    on_delete = RESTRICT
  }

  foreign_key "product_sourcing_confirmations_offer_id_fkey" {
    columns = [column.offer_id]

    ref_columns = [
      table.product_sourcing_offers.column.id
    ]

    on_update = NO_ACTION
    on_delete = RESTRICT
  }

  foreign_key "product_sourcing_confirmations_finalized_by_fkey" {
    columns = [column.finalized_by]

    ref_columns = [
      table.support_actors.column.id
    ]

    on_update = NO_ACTION
    on_delete = NO_ACTION
  }

  foreign_key "product_sourcing_confirmations_product_id_fkey" {
    columns = [column.created_product_id]

    ref_columns = [
      table.products.column.id
    ]

    on_update = NO_ACTION
    on_delete = NO_ACTION
  }

  foreign_key "product_sourcing_confirmations_variant_id_fkey" {
    columns = [column.created_variant_id]

    ref_columns = [
      table.product_variants.column.id
    ]

    on_update = NO_ACTION
    on_delete = NO_ACTION
  }

  foreign_key "product_sourcing_confirmations_order_id_fkey" {
    columns = [column.created_order_id]

    ref_columns = [
      table.orders.column.id
    ]

    on_update = NO_ACTION
    on_delete = SET_NULL
  }

  index "product_sourcing_confirmations_request_key" {
    unique = true
    columns = [column.request_id]
  }

  index "product_sourcing_confirmations_offer_key" {
    unique = true
    columns = [column.offer_id]
  }

  index "idx_product_sourcing_confirmations_product_id" {
    columns = [column.created_product_id]
  }

  index "idx_product_sourcing_confirmations_variant_id" {
    columns = [column.created_variant_id]
  }

  index "product_sourcing_confirmations_order_key" {
    unique = true
    columns = [column.created_order_id]
    where = "created_order_id IS NOT NULL"
  }

  check "product_sourcing_confirmations_quantity_positive" {
    expr = "quantity > 0"
  }

  check "product_sourcing_confirmations_minimum_order_quantity_positive" {
    expr = "minimum_order_quantity > 0"
  }

  check "product_sourcing_confirmations_quantity_meets_minimum" {
    expr = "quantity >= minimum_order_quantity"
  }

  check "product_sourcing_confirmations_product_name_not_blank" {
    expr = "length(trim(accepted_product_name)) > 0"
  }

  check "product_sourcing_confirmations_specifications_is_object" {
    expr = "accepted_specifications IS NULL OR jsonb_typeof(accepted_specifications) = 'object'"
  }

  check "product_sourcing_confirmations_unit_price_nonnegative" {
    expr = "unit_price_snapshot >= 0"
  }

  check "product_sourcing_confirmations_shipping_price_nonnegative" {
    expr = "shipping_price_snapshot >= 0"
  }

  check "product_sourcing_confirmations_total_nonnegative" {
    expr = "total_amount >= 0"
  }

  check "product_sourcing_confirmations_total_consistent" {
    expr = "total_amount = unit_price_snapshot * quantity + shipping_price_snapshot"
  }

  check "product_sourcing_confirmations_currency_valid" {
    expr = "length(currency) = 3 AND currency = upper(currency)"
  }

  check "product_sourcing_confirmations_status_valid" {
    expr = "status IN ('confirmed', 'order_created', 'cancelled')"
  }
}

table "reviews" {
  schema = schema.public

  column "id" {
    type    = uuid
    null    = false
    default = sql("gen_random_uuid()")
  }

  column "customer_id" {
    type = uuid
    null = false
  }

  column "order_item_id" {
    type = uuid
    null = false
  }

  column "rating" {
    type = integer
    null = false
  }

  column "title" {
    type = varchar(160)
    null = true
  }

  column "body" {
    type = text
    null = true
  }

  column "status" {
    type    = varchar(20)
    null    = false
    default = "published"
  }

  column "deleted_at" {
    type = timestamptz
    null = true
  }

  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  column "updated_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  foreign_key "reviews_customer_id_fkey" {
    columns     = [column.customer_id]
    ref_columns = [table.customers.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }

  foreign_key "reviews_order_item_id_fkey" {
    columns     = [column.order_item_id]
    ref_columns = [table.order_items.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }

  index "reviews_order_item_active_key" {
    unique  = true
    columns = [column.order_item_id]
    where   = "deleted_at IS NULL"
  }

  index "idx_reviews_customer_created" {
    columns = [
      column.customer_id,
      column.created_at,
    ]

    where = "deleted_at IS NULL"
  }

  index "idx_reviews_status_created" {
    columns = [
      column.status,
      column.created_at,
    ]

    where = "deleted_at IS NULL"
  }

  check "reviews_rating_valid" {
    expr = "rating >= 1 AND rating <= 5"
  }

  check "reviews_title_valid" {
    expr = "title IS NULL OR length(trim(title)) > 0"
  }

  check "reviews_body_valid" {
    expr = "body IS NULL OR (length(trim(body)) > 0 AND char_length(body) <= 5000)"
  }

  check "reviews_status_valid" {
    expr = "status IN ('published', 'hidden')"
  }
}


table "warehouses" {
  schema = schema.public

  column "id" {
    type    = uuid
    null    = false
    default = sql("gen_random_uuid()")
  }

  column "code" {
    type = varchar(40)
    null = false
  }

  column "name" {
    type = varchar(160)
    null = false
  }

  column "country_code" {
    type    = varchar(2)
    null    = false
    default = "BD"
  }

  column "city" {
    type = varchar(120)
    null = true
  }

  column "address_line1" {
    type = varchar(255)
    null = true
  }


    column "latitude" {
    type = double_precision
    null = true
  }

  column "longitude" {
    type = double_precision
    null = true
  }

  column "status" {
    type    = varchar(20)
    null    = false
    default = "active"
  }

  column "is_default" {
    type    = boolean
    null    = false
    default = false
  }

  column "allows_self_pickup" {
    type    = boolean
    null    = false
    default = false
  }

  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  column "updated_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  index "warehouses_code_key" {
    unique  = true
    columns = [column.code]
  }

  index "warehouses_one_default_key" {
    unique  = true
    columns = [column.is_default]
    where   = "is_default = true"
  }

  index "idx_warehouses_status_created" {
    columns = [
      column.status,
      column.created_at,
    ]
  }

  check "warehouses_code_not_blank" {
    expr = "length(trim(code)) > 0"
  }

  check "warehouses_name_not_blank" {
    expr = "length(trim(name)) > 0"
  }

  check "warehouses_country_code_valid" {
    expr = "length(country_code) = 2 AND country_code = upper(country_code)"
  }

    check "warehouses_latitude_valid" {
    expr = "latitude IS NULL OR (latitude >= -90 AND latitude <= 90)"
  }

  check "warehouses_longitude_valid" {
    expr = "longitude IS NULL OR (longitude >= -180 AND longitude <= 180)"
  }

  check "warehouses_coordinates_pair" {
    expr = "(latitude IS NULL AND longitude IS NULL) OR (latitude IS NOT NULL AND longitude IS NOT NULL)"
  }

  check "warehouses_status_valid" {
    expr = "status IN ('active', 'inactive')"
  }
}


table "inbound_shipments" {
  schema = schema.public

  column "id" {
    type    = uuid
    null    = false
    default = sql("gen_random_uuid()")
  }

  column "reference_code" {
    type = varchar(80)
    null = false
  }

  column "origin_country" {
    type    = varchar(2)
    null    = false
    default = "CN"
  }

  column "destination_warehouse_id" {
    type = uuid
    null = false
  }

  column "carrier_name" {
    type = varchar(160)
    null = true
  }

  column "external_reference" {
    type = varchar(160)
    null = true
  }

  column "tracking_number" {
    type = varchar(160)
    null = true
  }

  column "status" {
    type    = varchar(40)
    null    = false
    default = "created"
  }

  column "eta" {
    type = timestamptz
    null = true
  }

  column "departed_at" {
    type = timestamptz
    null = true
  }

  column "arrived_bangladesh_at" {
    type = timestamptz
    null = true
  }

  column "customs_released_at" {
    type = timestamptz
    null = true
  }

  column "received_at" {
    type = timestamptz
    null = true
  }

  column "notes" {
    type = text
    null = true
  }

  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  column "updated_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  foreign_key "inbound_shipments_destination_warehouse_fkey" {
    columns     = [column.destination_warehouse_id]
    ref_columns = [table.warehouses.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }

  index "inbound_shipments_reference_code_key" {
    unique  = true
    columns = [column.reference_code]
  }

  index "idx_inbound_shipments_status_eta" {
    columns = [
      column.status,
      column.eta,
    ]
  }

  index "idx_inbound_shipments_destination_created" {
    columns = [
      column.destination_warehouse_id,
      column.created_at,
    ]
  }

  check "inbound_shipments_reference_not_blank" {
    expr = "length(trim(reference_code)) > 0"
  }

  check "inbound_shipments_origin_country_valid" {
    expr = "length(origin_country) = 2 AND origin_country = upper(origin_country)"
  }

  check "inbound_shipments_status_valid" {
    expr = "status IN ('created', 'supplier_ready', 'packed_in_china', 'picked_up_in_china', 'departed_china', 'in_international_transit', 'arrived_bangladesh', 'customs_processing', 'customs_released', 'received_at_warehouse', 'cancelled')"
  }
}
table "inbound_shipment_events" {
  schema = schema.public

  column "id" {
    type    = uuid
    null    = false
    default = sql("gen_random_uuid()")
  }

  column "inbound_shipment_id" {
    type = uuid
    null = false
  }

  column "status" {
    type = varchar(40)
    null = false
  }

  column "message" {
    type = varchar(1000)
    null = true
  }

  column "public_message" {
    type = varchar(500)
    null = true
  }

  column "country_code" {
    type = varchar(2)
    null = true
  }

  column "city" {
    type = varchar(120)
    null = true
  }

  column "location_name" {
    type = varchar(160)
    null = true
  }

  column "latitude" {
    type = double_precision
    null = true
  }

  column "longitude" {
    type = double_precision
    null = true
  }

  column "customer_visible" {
    type    = boolean
    null    = false
    default = true
  }

  column "actor_type" {
    type = varchar(40)
    null = true
  }

  column "actor_id" {
    type = varchar(160)
    null = true
  }

  column "occurred_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  foreign_key "inbound_shipment_events_shipment_fkey" {
    columns     = [column.inbound_shipment_id]
    ref_columns = [table.inbound_shipments.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }

  index "idx_inbound_shipment_events_shipment_time" {
    columns = [
      column.inbound_shipment_id,
      column.occurred_at,
      column.created_at,
    ]
  }

  check "inbound_shipment_events_status_valid" {
    expr = "status IN ('created', 'supplier_ready', 'packed_in_china', 'picked_up_in_china', 'departed_china', 'in_international_transit', 'arrived_bangladesh', 'customs_processing', 'customs_released', 'received_at_warehouse', 'cancelled')"
  }

  check "inbound_shipment_events_actor_pair" {
    expr = "(actor_type IS NULL AND actor_id IS NULL) OR (actor_type IS NOT NULL AND actor_id IS NOT NULL)"
  }

  check "inbound_shipment_events_country_valid" {
    expr = "country_code IS NULL OR (length(country_code) = 2 AND country_code = upper(country_code))"
  }

  check "inbound_shipment_events_latitude_valid" {
    expr = "latitude IS NULL OR (latitude >= -90 AND latitude <= 90)"
  }

  check "inbound_shipment_events_longitude_valid" {
    expr = "longitude IS NULL OR (longitude >= -180 AND longitude <= 180)"
  }

  check "inbound_shipment_events_coordinates_pair" {
    expr = "(latitude IS NULL AND longitude IS NULL) OR (latitude IS NOT NULL AND longitude IS NOT NULL)"
  }
}

table "warehouse_fulfillments" {
  schema = schema.public

  column "id" {
    type    = uuid
    null    = false
    default = sql("gen_random_uuid()")
  }

  column "order_id" {
    type = uuid
    null = false
  }

  column "order_item_id" {
    type = uuid
    null = false
  }

  column "warehouse_id" {
    type = uuid
    null = false
  }

  column "inbound_shipment_id" {
    type = uuid
    null = true
  }

  column "source" {
    type = varchar(30)
    null = false
  }

  column "status" {
    type = varchar(40)
    null = false
  }

  column "quantity" {
    type = integer
    null = false
  }

  column "allocated_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  column "received_at" {
    type = timestamptz
    null = true
  }

  column "picking_at" {
    type = timestamptz
    null = true
  }

  column "packed_at" {
    type = timestamptz
    null = true
  }

  column "ready_for_handoff_at" {
    type = timestamptz
    null = true
  }

  column "handed_off_at" {
    type = timestamptz
    null = true
  }

  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  column "updated_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  foreign_key "warehouse_fulfillments_order_fkey" {
    columns     = [column.order_id]
    ref_columns = [table.orders.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }

  foreign_key "warehouse_fulfillments_order_item_fkey" {
    columns     = [column.order_item_id]
    ref_columns = [table.order_items.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }

  foreign_key "warehouse_fulfillments_warehouse_fkey" {
    columns     = [column.warehouse_id]
    ref_columns = [table.warehouses.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }

  foreign_key "warehouse_fulfillments_inbound_shipment_fkey" {
    columns     = [column.inbound_shipment_id]
    ref_columns = [table.inbound_shipments.column.id]
    on_update   = NO_ACTION
    on_delete   = SET_NULL
  }

  index "idx_warehouse_fulfillments_order_status" {
    columns = [
      column.order_id,
      column.status,
    ]
  }

  index "idx_warehouse_fulfillments_order_item" {
    columns = [column.order_item_id]
  }

  index "idx_warehouse_fulfillments_warehouse_status" {
    columns = [
      column.warehouse_id,
      column.status,
    ]
  }

  index "idx_warehouse_fulfillments_inbound_status" {
    columns = [
      column.inbound_shipment_id,
      column.status,
    ]
  }

  check "warehouse_fulfillments_source_valid" {
    expr = "source IN ('bangladesh_stock', 'china_inbound')"
  }

  check "warehouse_fulfillments_status_valid" {
    expr = "status IN ('allocated', 'waiting_inbound', 'received', 'picking', 'packed', 'ready_for_handoff', 'handed_off', 'cancelled')"
  }

  check "warehouse_fulfillments_quantity_positive" {
    expr = "quantity > 0"
  }

  check "warehouse_fulfillments_waiting_inbound_source" {
    expr = "status <> 'waiting_inbound' OR source = 'china_inbound'"
  }

  check "warehouse_fulfillments_local_has_no_inbound" {
    expr = "source <> 'bangladesh_stock' OR inbound_shipment_id IS NULL"
  }
}


table "warehouse_fulfillment_events" {
  schema = schema.public

  column "id" {
    type    = uuid
    null    = false
    default = sql("gen_random_uuid()")
  }

  column "fulfillment_id" {
    type = uuid
    null = false
  }

  column "event_type" {
    type = varchar(60)
    null = false
  }

  column "from_status" {
    type = varchar(40)
    null = true
  }

  column "to_status" {
    type = varchar(40)
    null = true
  }

  column "message" {
    type = varchar(1000)
    null = true
  }

  column "actor_type" {
    type = varchar(40)
    null = true
  }

  column "actor_id" {
    type = varchar(160)
    null = true
  }

  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  foreign_key "warehouse_fulfillment_events_fulfillment_fkey" {
    columns     = [column.fulfillment_id]
    ref_columns = [table.warehouse_fulfillments.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }

  index "idx_warehouse_fulfillment_events_fulfillment_time" {
    columns = [
      column.fulfillment_id,
      column.created_at,
    ]
  }

  check "warehouse_fulfillment_events_type_not_blank" {
    expr = "length(trim(event_type)) > 0"
  }

  check "warehouse_fulfillment_events_actor_pair" {
    expr = "(actor_type IS NULL AND actor_id IS NULL) OR (actor_type IS NOT NULL AND actor_id IS NOT NULL)"
  }
}

table "customer_notifications" {
  schema = schema.public

  column "id" {
    type    = uuid
    null    = false
    default = sql("gen_random_uuid()")
  }

  column "dedupe_key" {
    type = varchar(200)
    null = false
  }

  column "customer_id" {
    type = uuid
    null = false
  }

  column "category" {
    type = varchar(30)
    null = false
  }

  column "event_type" {
    type = varchar(100)
    null = false
  }

  column "title" {
    type = varchar(160)
    null = false
  }

  column "message" {
    type = varchar(1000)
    null = false
  }

  column "action_url" {
    type = varchar(500)
    null = true
  }

  column "order_id" {
    type = uuid
    null = true
  }

  column "case_id" {
    type = uuid
    null = true
  }

  column "metadata" {
    type    = jsonb
    null    = false
    default = sql("'{}'::jsonb")
  }

  column "read_at" {
    type = timestamptz
    null = true
  }

  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  foreign_key "customer_notifications_customer_id_fkey" {
    columns     = [column.customer_id]
    ref_columns = [table.customers.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }

  foreign_key "customer_notifications_order_id_fkey" {
    columns     = [column.order_id]
    ref_columns = [table.orders.column.id]
    on_update   = NO_ACTION
    on_delete   = SET_NULL
  }

  foreign_key "customer_notifications_case_id_fkey" {
    columns     = [column.case_id]
    ref_columns = [table.crm_cases.column.id]
    on_update   = NO_ACTION
    on_delete   = SET_NULL
  }

  index "customer_notifications_dedupe_key_key" {
    unique  = true
    columns = [column.dedupe_key]
  }

  index "idx_customer_notifications_customer_created" {
    columns = [
      column.customer_id,
      column.created_at,
    ]
  }

  index "idx_customer_notifications_customer_unread" {
    columns = [
      column.customer_id,
      column.created_at,
    ]
    where = "read_at IS NULL"
  }

  check "customer_notifications_dedupe_key_not_blank" {
    expr = "length(trim(dedupe_key)) > 0"
  }

  check "customer_notifications_category_valid" {
    expr = "category IN ('order', 'payment', 'delivery', 'sourcing', 'support', 'promotion', 'recommendation', 'security', 'system')"
  }

  check "customer_notifications_event_type_not_blank" {
    expr = "length(trim(event_type)) > 0"
  }

  check "customer_notifications_title_not_blank" {
    expr = "length(trim(title)) > 0"
  }

  check "customer_notifications_message_not_blank" {
    expr = "length(trim(message)) > 0"
  }

  check "customer_notifications_action_url_valid" {
    expr = "action_url IS NULL OR (length(trim(action_url)) > 0 AND action_url LIKE '/%' AND action_url NOT LIKE '//%')"
  }
}


table "staff_notifications" {
  schema = schema.public

  column "id" {
    type    = uuid
    null    = false
    default = sql("gen_random_uuid()")
  }

  column "dedupe_key" {
    type = varchar(200)
    null = false
  }

  column "category" {
    type = varchar(30)
    null = false
  }

  column "event_type" {
    type = varchar(100)
    null = false
  }

  column "priority" {
    type    = varchar(20)
    null    = false
    default = "info"
  }

  column "title" {
    type = varchar(160)
    null = false
  }

  column "message" {
    type = varchar(1000)
    null = false
  }

  column "action_url" {
    type = varchar(500)
    null = true
  }

  column "entity_type" {
    type = varchar(50)
    null = true
  }

  column "entity_id" {
    type = varchar(160)
    null = true
  }

  column "metadata" {
    type    = jsonb
    null    = false
    default = sql("'{}'::jsonb")
  }

  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  index "staff_notifications_dedupe_key_key" {
    unique  = true
    columns = [column.dedupe_key]
  }

  index "idx_staff_notifications_category_created" {
    columns = [
      column.category,
      column.created_at,
    ]
  }

  index "idx_staff_notifications_entity_created" {
    columns = [
      column.entity_type,
      column.entity_id,
      column.created_at,
    ]
    where = "entity_type IS NOT NULL AND entity_id IS NOT NULL"
  }

  check "staff_notifications_dedupe_key_not_blank" {
    expr = "length(trim(dedupe_key)) > 0"
  }

  check "staff_notifications_category_valid" {
    expr = "category IN ('order', 'inventory', 'sourcing', 'support', 'return', 'delivery', 'security', 'system')"
  }

  check "staff_notifications_event_type_not_blank" {
    expr = "length(trim(event_type)) > 0"
  }

  check "staff_notifications_priority_valid" {
    expr = "priority IN ('info', 'attention', 'critical')"
  }

  check "staff_notifications_title_not_blank" {
    expr = "length(trim(title)) > 0"
  }

  check "staff_notifications_message_not_blank" {
    expr = "length(trim(message)) > 0"
  }

  check "staff_notifications_action_url_valid" {
    expr = "action_url IS NULL OR (length(trim(action_url)) > 0 AND action_url LIKE '/%' AND action_url NOT LIKE '//%')"
  }

  check "staff_notifications_entity_pair" {
    expr = "(entity_type IS NULL AND entity_id IS NULL) OR (entity_type IS NOT NULL AND entity_id IS NOT NULL)"
  }
}


table "staff_notification_recipients" {
  schema = schema.public

  column "notification_id" {
    type = uuid
    null = false
  }

  column "staff_account_id" {
    type = uuid
    null = false
  }

  column "read_at" {
    type = timestamptz
    null = true
  }

  column "dismissed_at" {
    type = timestamptz
    null = true
  }

  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  primary_key {
    columns = [
      column.notification_id,
      column.staff_account_id,
    ]
  }

  foreign_key "staff_notification_recipients_notification_fkey" {
    columns     = [column.notification_id]
    ref_columns = [table.staff_notifications.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }

  foreign_key "staff_notification_recipients_staff_fkey" {
    columns     = [column.staff_account_id]
    ref_columns = [table.staff_accounts.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }

  index "idx_staff_notification_recipients_staff_created" {
    columns = [
      column.staff_account_id,
      column.created_at,
    ]
  }

  index "idx_staff_notification_recipients_staff_unread" {
    columns = [
      column.staff_account_id,
      column.created_at,
    ]
    where = "read_at IS NULL AND dismissed_at IS NULL"
  }
}

table "notification_outbox" {
  schema = schema.public

  column "id" {
    type    = uuid
    null    = false
    default = sql("gen_random_uuid()")
  }

  # Idempotency key generated by the domain transaction.
  #
  # Examples:
  # order:<order_id>:order_placed:sms
  # delivery:<order_id>:arrived_bangladesh:email
  # support:<case_id>:<message_id>:sms
  column "dedupe_key" {
    type = varchar(200)
    null = false
  }

  column "category" {
    type = varchar(30)
    null = false
  }

  column "event_type" {
    type = varchar(100)
    null = false
  }

  column "channel" {
    type = varchar(20)
    null = false
  }

  column "customer_id" {
    type = uuid
    null = true
  }

  column "order_id" {
    type = uuid
    null = true
  }

  column "case_id" {
    type = uuid
    null = true
  }

  # Snapshot the recipient at event creation time.
  #
  # This is important for guest orders and also prevents a later
  # profile edit from changing the destination of an already-created
  # transactional notification.
  column "recipient" {
    type = varchar(255)
    null = true
  }

  column "template_key" {
    type = varchar(120)
    null = false
  }

  # Variables needed by the renderer.
  #
  # Do not put passwords, access tokens, OTP hashes, provider secrets,
  # internal admin metadata, or other sensitive values in this field.
  column "payload" {
    type    = jsonb
    null    = false
    default = sql("'{}'::jsonb")
  }

  # pending
  # processing
  # sent
  # skipped
  # dead
  column "status" {
    type    = varchar(20)
    null    = false
    default = "pending"
  }

  column "attempt_count" {
    type    = integer
    null    = false
    default = 0
  }

  column "max_attempts" {
    type    = integer
    null    = false
    default = 5
  }

  # Also serves as next-attempt time after temporary failures.
  column "available_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  # Used for FOR UPDATE SKIP LOCKED / abandoned-worker recovery.
  column "locked_at" {
    type = timestamptz
    null = true
  }

  column "processed_at" {
    type = timestamptz
    null = true
  }

  column "provider_message_id" {
    type = varchar(255)
    null = true
  }

  column "last_error" {
    type = varchar(1000)
    null = true
  }

  column "skip_reason" {
    type = varchar(120)
    null = true
  }

  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  column "updated_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  foreign_key "notification_outbox_customer_id_fkey" {
    columns     = [column.customer_id]
    ref_columns = [table.customers.column.id]
    on_update   = NO_ACTION
    on_delete   = SET_NULL
  }

  foreign_key "notification_outbox_order_id_fkey" {
    columns     = [column.order_id]
    ref_columns = [table.orders.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }

  foreign_key "notification_outbox_case_id_fkey" {
    columns     = [column.case_id]
    ref_columns = [table.crm_cases.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }

  index "notification_outbox_dedupe_key_key" {
    unique  = true
    columns = [column.dedupe_key]
  }

  index "idx_notification_outbox_due" {
    columns = [
      column.status,
      column.available_at,
      column.created_at,
    ]

   where = "(status)::text = ANY ((ARRAY['pending'::character varying, 'processing'::character varying])::text[])"
  }

  index "idx_notification_outbox_customer_created" {
    columns = [
      column.customer_id,
      column.created_at,
    ]
  }

  index "idx_notification_outbox_order_created" {
    columns = [
      column.order_id,
      column.created_at,
    ]
  }

  index "idx_notification_outbox_case_created" {
    columns = [
      column.case_id,
      column.created_at,
    ]
  }

  check "notification_outbox_dedupe_key_not_blank" {
    expr = "length(trim(dedupe_key)) > 0"
  }

  check "notification_outbox_category_valid" {
    expr = "category IN ('order', 'payment', 'delivery', 'sourcing', 'support', 'promotion', 'recommendation', 'security', 'system')"
  }

  check "notification_outbox_event_type_not_blank" {
    expr = "length(trim(event_type)) > 0"
  }

  check "notification_outbox_channel_valid" {
    expr = "channel IN ('sms', 'email', 'push', 'messenger')"
  }

  check "notification_outbox_recipient_not_blank" {
    expr = "recipient IS NULL OR length(trim(recipient)) > 0"
  }

  check "notification_outbox_template_key_not_blank" {
    expr = "length(trim(template_key)) > 0"
  }

  check "notification_outbox_status_valid" {
    expr = "status IN ('pending', 'processing', 'sent', 'skipped', 'dead')"
  }

  check "notification_outbox_attempt_count_valid" {
    expr = "attempt_count >= 0"
  }

  check "notification_outbox_max_attempts_valid" {
    expr = "max_attempts >= 1 AND max_attempts <= 20"
  }

  check "notification_outbox_attempts_consistent" {
    expr = "attempt_count <= max_attempts"
  }

  check "notification_outbox_processing_lock_valid" {
    expr = "status <> 'processing' OR locked_at IS NOT NULL"
  }

  check "notification_outbox_terminal_timestamp_valid" {
    expr = "status NOT IN ('sent', 'skipped', 'dead') OR processed_at IS NOT NULL"
  }

  check "notification_outbox_nonterminal_timestamp_valid" {
    expr = "status NOT IN ('pending', 'processing') OR processed_at IS NULL"
  }

  check "notification_outbox_provider_message_id_not_blank" {
    expr = "provider_message_id IS NULL OR length(trim(provider_message_id)) > 0"
  }

  check "notification_outbox_last_error_not_blank" {
    expr = "last_error IS NULL OR length(trim(last_error)) > 0"
  }

  check "notification_outbox_skip_reason_not_blank" {
    expr = "skip_reason IS NULL OR length(trim(skip_reason)) > 0"
  }
}


table "invoices" {
  schema = schema.public

  column "id" {
    type    = uuid
    null    = false
    default = sql("gen_random_uuid()")
  }

  column "order_id" {
    type = uuid
    null = false
  }

  column "customer_id" {
    type = uuid
    null = true
  }

  column "invoice_number" {
    type = varchar(80)
    null = false
  }

  column "issued_at" {
    type = timestamptz
    null = false
  }

  column "currency" {
    type = varchar(3)
    null = false
  }

  column "subtotal_amount" {
    type = bigint
    null = false
  }

  column "discount_amount" {
    type    = bigint
    null    = false
    default = 0
  }

  column "shipping_amount" {
    type    = bigint
    null    = false
    default = 0
  }

  column "total_amount" {
    type = bigint
    null = false
  }

  column "customer_name" {
    type = varchar(160)
    null = false
  }

  column "customer_phone" {
    type = varchar(40)
    null = false
  }

  column "customer_email" {
    type = varchar(255)
    null = true
  }

  column "shipping_address_line1" {
    type = varchar(255)
    null = false
  }

  column "shipping_address_line2" {
    type = varchar(255)
    null = true
  }

  column "shipping_city" {
    type = varchar(120)
    null = false
  }

  column "shipping_area" {
    type = varchar(120)
    null = false
  }

  column "shipping_postal_code" {
    type = varchar(30)
    null = true
  }

  column "delivery_method" {
    type = varchar(80)
    null = false
  }

  column "payment_method" {
    type = varchar(80)
    null = false
  }

  # This is the payment state at invoice issuance time.
  # Current payment state can still be obtained from the order.
  column "payment_status_at_issue" {
    type = varchar(30)
    null = false
  }

  # Merchant/company information will be populated by Invoice v1
  # configuration. Keeping the snapshot on the invoice prevents a
  # later merchant-profile change from rewriting old invoices.
  column "merchant_snapshot" {
    type    = jsonb
    null    = false
    default = sql("'{}'::jsonb")
  }

  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  foreign_key "invoices_order_id_fkey" {
    columns     = [column.order_id]
    ref_columns = [table.orders.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }

  foreign_key "invoices_customer_id_fkey" {
    columns     = [column.customer_id]
    ref_columns = [table.customers.column.id]
    on_update   = NO_ACTION
    on_delete   = SET_NULL
  }

  index "invoices_order_id_key" {
    unique  = true
    columns = [column.order_id]
  }

  index "invoices_invoice_number_key" {
    unique  = true
    columns = [column.invoice_number]
  }

  index "idx_invoices_customer_issued" {
    columns = [
      column.customer_id,
      column.issued_at,
    ]
  }

  index "idx_invoices_issued" {
    columns = [
      column.issued_at,
      column.id,
    ]
  }

  check "invoices_invoice_number_not_blank" {
    expr = "length(trim(invoice_number)) > 0"
  }

  check "invoices_currency_valid" {
    expr = "length(currency) = 3 AND currency = upper(currency)"
  }

  check "invoices_subtotal_nonnegative" {
    expr = "subtotal_amount >= 0"
  }

  check "invoices_discount_nonnegative" {
    expr = "discount_amount >= 0"
  }

  check "invoices_shipping_nonnegative" {
    expr = "shipping_amount >= 0"
  }

  check "invoices_total_nonnegative" {
    expr = "total_amount >= 0"
  }

  check "invoices_discount_not_above_subtotal" {
    expr = "discount_amount <= subtotal_amount"
  }

  check "invoices_total_consistent" {
    expr = "total_amount = subtotal_amount - discount_amount + shipping_amount"
  }

  check "invoices_customer_name_not_blank" {
    expr = "length(trim(customer_name)) > 0"
  }

  check "invoices_customer_phone_not_blank" {
    expr = "length(trim(customer_phone)) > 0"
  }

  check "invoices_customer_email_not_blank" {
    expr = "customer_email IS NULL OR length(trim(customer_email)) > 0"
  }

  check "invoices_shipping_address_line1_not_blank" {
    expr = "length(trim(shipping_address_line1)) > 0"
  }

  check "invoices_shipping_city_not_blank" {
    expr = "length(trim(shipping_city)) > 0"
  }

  check "invoices_shipping_area_not_blank" {
    expr = "length(trim(shipping_area)) > 0"
  }

  check "invoices_delivery_method_not_blank" {
    expr = "length(trim(delivery_method)) > 0"
  }

  check "invoices_payment_method_not_blank" {
    expr = "length(trim(payment_method)) > 0"
  }

  check "invoices_payment_status_valid" {
    expr = "payment_status_at_issue IN ('pending', 'paid', 'failed', 'expired', 'cod_pending', 'cod_collected', 'refunded')"
  }

  check "invoices_merchant_snapshot_object" {
    expr = "jsonb_typeof(merchant_snapshot) = 'object'"
  }
}


table "invoice_items" {
  schema = schema.public

  column "id" {
    type    = uuid
    null    = false
    default = sql("gen_random_uuid()")
  }

  column "invoice_id" {
    type = uuid
    null = false
  }

  column "order_item_id" {
    type = uuid
    null = false
  }

  column "variant_id" {
    type = uuid
    null = false
  }

  column "sku" {
    type = varchar(100)
    null = false
  }

  column "product_name" {
    type = varchar(180)
    null = false
  }

  column "quantity" {
    type = integer
    null = false
  }

  column "unit_price_amount" {
    type = bigint
    null = false
  }

  column "line_total_amount" {
    type = bigint
    null = false
  }

  column "currency" {
    type = varchar(3)
    null = false
  }

  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  foreign_key "invoice_items_invoice_id_fkey" {
    columns     = [column.invoice_id]
    ref_columns = [table.invoices.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }

  foreign_key "invoice_items_order_item_id_fkey" {
    columns     = [column.order_item_id]
    ref_columns = [table.order_items.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }

  foreign_key "invoice_items_variant_id_fkey" {
    columns     = [column.variant_id]
    ref_columns = [table.product_variants.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }

  index "invoice_items_order_item_id_key" {
    unique  = true
    columns = [column.order_item_id]
  }

  index "idx_invoice_items_invoice" {
    columns = [
      column.invoice_id,
      column.created_at,
    ]
  }

  index "idx_invoice_items_variant" {
    columns = [column.variant_id]
  }

  check "invoice_items_sku_not_blank" {
    expr = "length(trim(sku)) > 0"
  }

  check "invoice_items_product_name_not_blank" {
    expr = "length(trim(product_name)) > 0"
  }

  check "invoice_items_quantity_positive" {
    expr = "quantity > 0"
  }

  check "invoice_items_unit_price_nonnegative" {
    expr = "unit_price_amount >= 0"
  }

  check "invoice_items_line_total_nonnegative" {
    expr = "line_total_amount >= 0"
  }

  check "invoice_items_line_total_consistent" {
    expr = "line_total_amount::numeric = unit_price_amount::numeric * quantity::numeric"
  }

  check "invoice_items_currency_valid" {
    expr = "length(currency) = 3 AND currency = upper(currency)"
  }
}

table "finance_expenses" {
  schema = schema.public

  column "id" {
    type    = uuid
    null    = false
    default = sql("gen_random_uuid()")
  }

  # Business expense category.
  #
  # Product purchase/buying cost is deliberately NOT stored here.
  # Buying cost belongs to finance_variant_cost_history and ultimately
  # COGS. This ledger is for costs deducted after gross profit.
  column "category" {
    type = varchar(40)
    null = false
  }

  # Positive amount in the explicitly recorded currency.
  #
  # No implicit FX conversion is performed.
  column "amount" {
    type = bigint
    null = false
  }

  column "currency" {
    type    = varchar(3)
    null    = false
    default = "BDT"
  }

  # Business date/time when the cost was incurred.
  #
  # This is intentionally independent from created_at so that an
  # expense entered later is still reported in the correct period.
  column "occurred_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  column "description" {
    type = varchar(500)
    null = false
  }

  # At most ONE business scope target may be supplied.
  #
  # No target   = business-wide expense
  # order_id    = order-specific expense
  # product_id  = product-specific expense
  # variant_id  = SKU/variant-specific expense
  # staff_account_id = staff-specific expense, such as salary
  # warehouse_id     = warehouse-specific expense
  column "order_id" {
    type = uuid
    null = true
  }

  column "product_id" {
    type = uuid
    null = true
  }

  column "variant_id" {
    type = uuid
    null = true
  }

  column "staff_account_id" {
    type = uuid
    null = true
  }

  column "warehouse_id" {
    type = uuid
    null = true
  }

  column "reference" {
    type = varchar(160)
    null = true
  }

  # Entries are never hard-deleted.
  #
  # Corrections use the existing void + replacement model so financial
  # history remains auditable.
  column "status" {
    type    = varchar(20)
    null    = false
    default = "active"
  }

  column "idempotency_key_hash" {
    type = varchar(64)
    null = false
  }

  column "request_fingerprint" {
    type = varchar(64)
    null = false
  }

  column "created_by_staff_id" {
    type = uuid
    null = false
  }

  column "voided_by_staff_id" {
    type = uuid
    null = true
  }

  column "void_reason" {
    type = varchar(500)
    null = true
  }

  column "voided_at" {
    type = timestamptz
    null = true
  }

  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  column "updated_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  foreign_key "finance_expenses_order_id_fkey" {
    columns     = [column.order_id]
    ref_columns = [table.orders.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }

  foreign_key "finance_expenses_product_id_fkey" {
    columns     = [column.product_id]
    ref_columns = [table.products.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }

  foreign_key "finance_expenses_variant_id_fkey" {
    columns     = [column.variant_id]
    ref_columns = [table.product_variants.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }

  foreign_key "finance_expenses_staff_account_id_fkey" {
    columns     = [column.staff_account_id]
    ref_columns = [table.staff_accounts.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }

  foreign_key "finance_expenses_warehouse_id_fkey" {
    columns     = [column.warehouse_id]
    ref_columns = [table.warehouses.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }

  foreign_key "finance_expenses_created_by_staff_id_fkey" {
    columns     = [column.created_by_staff_id]
    ref_columns = [table.staff_accounts.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }

  foreign_key "finance_expenses_voided_by_staff_id_fkey" {
    columns     = [column.voided_by_staff_id]
    ref_columns = [table.staff_accounts.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }

  index "finance_expenses_idempotency_key_hash_key" {
    unique  = true
    columns = [column.idempotency_key_hash]
  }

  index "idx_finance_expenses_currency_occurred" {
    columns = [
      column.currency,
      column.occurred_at,
    ]
  }

  index "idx_finance_expenses_status_occurred" {
    columns = [
      column.status,
      column.occurred_at,
    ]
  }

  index "idx_finance_expenses_category_occurred" {
    columns = [
      column.category,
      column.occurred_at,
    ]
  }

  index "idx_finance_expenses_order" {
    columns = [
      column.order_id,
      column.occurred_at,
    ]
    where = "order_id IS NOT NULL"
  }

  index "idx_finance_expenses_product" {
    columns = [
      column.product_id,
      column.occurred_at,
    ]
    where = "product_id IS NOT NULL"
  }

  index "idx_finance_expenses_variant" {
    columns = [
      column.variant_id,
      column.occurred_at,
    ]
    where = "variant_id IS NOT NULL"
  }

  index "idx_finance_expenses_staff" {
    columns = [
      column.staff_account_id,
      column.occurred_at,
    ]
    where = "staff_account_id IS NOT NULL"
  }

  index "idx_finance_expenses_warehouse" {
    columns = [
      column.warehouse_id,
      column.occurred_at,
    ]
    where = "warehouse_id IS NOT NULL"
  }

  index "idx_finance_expenses_created_by" {
    columns = [
      column.created_by_staff_id,
      column.created_at,
    ]
  }

  check "finance_expenses_category_valid" {
    expr = "category IN ('delivery', 'payment_fee', 'china_freight', 'customs', 'packaging', 'marketing', 'warehouse', 'salary', 'staff_benefit', 'rent', 'utilities', 'software', 'professional_service', 'bank_fee', 'insurance', 'tax_fee', 'office', 'travel', 'other')"
  }

  check "finance_expenses_amount_positive" {
    expr = "amount > 0"
  }

  check "finance_expenses_currency_valid" {
    expr = "length(currency) = 3 AND currency = upper(currency)"
  }

  check "finance_expenses_description_not_blank" {
    expr = "length(trim(description)) > 0"
  }

  check "finance_expenses_reference_not_blank" {
    expr = "reference IS NULL OR length(trim(reference)) > 0"
  }

  # An expense can be global or attached to exactly one scope.
  check "finance_expenses_single_scope_target" {
    expr = "num_nonnulls(order_id, product_id, variant_id, staff_account_id, warehouse_id) <= 1"
  }

  check "finance_expenses_status_valid" {
    expr = "status IN ('active', 'voided')"
  }

  check "finance_expenses_idempotency_hash_valid" {
    expr = "length(idempotency_key_hash) = 64"
  }

  check "finance_expenses_request_fingerprint_valid" {
    expr = "length(request_fingerprint) = 64"
  }

  check "finance_expenses_void_fields_consistent" {
    expr = "(status = 'active' AND voided_by_staff_id IS NULL AND void_reason IS NULL AND voided_at IS NULL) OR (status = 'voided' AND voided_by_staff_id IS NOT NULL AND void_reason IS NOT NULL AND length(trim(void_reason)) > 0 AND voided_at IS NOT NULL)"
  }

  check "finance_expenses_updated_at_valid" {
    expr = "updated_at >= created_at"
  }
}

table "finance_variant_cost_history" {
  schema = schema.public

  column "id" {
    type    = uuid
    null    = false
    default = sql("gen_random_uuid()")
  }

  column "variant_id" {
    type = uuid
    null = false
  }

  # Cost per unit effective from effective_at.
  #
  # This is internal financial cost, not the customer selling price.
  column "unit_cost_amount" {
    type = bigint
    null = false
  }

  # Previous live product_variants.cost_amount at the time this change
  # was applied. NULL means that no buying cost had been recorded.
  column "previous_unit_cost_amount" {
    type = bigint
    null = true
  }

  column "currency" {
    type = varchar(3)
    null = false
  }

  # Allows historically correct cost reconstruction when an old order
  # item has no immutable unit_cost_amount snapshot.
  column "effective_at" {
    type = timestamptz
    null = false
  }

  # manual | import
  column "source" {
    type = varchar(20)
    null = false
  }

  column "description" {
    type = varchar(500)
    null = true
  }

  column "reference" {
    type = varchar(160)
    null = true
  }

  column "created_by_staff_id" {
    type = uuid
    null = false
  }

  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  foreign_key "finance_variant_cost_history_variant_id_fkey" {
    columns     = [column.variant_id]
    ref_columns = [table.product_variants.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }

  foreign_key "finance_variant_cost_history_created_by_staff_id_fkey" {
    columns     = [column.created_by_staff_id]
    ref_columns = [table.staff_accounts.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }

  index "idx_finance_variant_cost_history_variant_effective" {
    columns = [
      column.variant_id,
      column.effective_at,
    ]
  }

  index "idx_finance_variant_cost_history_effective" {
    columns = [column.effective_at]
  }

  index "idx_finance_variant_cost_history_created_by" {
    columns = [
      column.created_by_staff_id,
      column.created_at,
    ]
  }

  check "finance_variant_cost_history_cost_nonnegative" {
    expr = "unit_cost_amount >= 0"
  }

  check "finance_variant_cost_history_previous_cost_nonnegative" {
    expr = "previous_unit_cost_amount IS NULL OR previous_unit_cost_amount >= 0"
  }

  check "finance_variant_cost_history_currency_valid" {
    expr = "length(currency) = 3 AND currency = upper(currency)"
  }

  check "finance_variant_cost_history_source_valid" {
    expr = "source IN ('manual', 'import')"
  }

  check "finance_variant_cost_history_description_not_blank" {
    expr = "description IS NULL OR length(trim(description)) > 0"
  }

  check "finance_variant_cost_history_reference_not_blank" {
    expr = "reference IS NULL OR length(trim(reference)) > 0"
  }
}

table "finance_import_batches" {
  schema = schema.public

  column "id" {
    type    = uuid
    null    = false
    default = sql("gen_random_uuid()")
  }

  column "source_filename" {
    type = varchar(255)
    null = false
  }

  # Prevent accidentally importing the exact same workbook twice.
  column "file_checksum_sha256" {
    type = varchar(64)
    null = false
  }

  column "template_version" {
    type    = integer
    null    = false
    default = 1
  }

  # validating
  # ready
  # applying
  # completed
  # failed
  column "status" {
    type    = varchar(30)
    null    = false
    default = "validating"
  }

  column "total_rows" {
    type    = integer
    null    = false
    default = 0
  }

  column "valid_rows" {
    type    = integer
    null    = false
    default = 0
  }

  column "invalid_rows" {
    type    = integer
    null    = false
    default = 0
  }

  column "applied_expenses" {
    type    = integer
    null    = false
    default = 0
  }

  column "applied_cost_updates" {
    type    = integer
    null    = false
    default = 0
  }

  column "skipped_rows" {
    type    = integer
    null    = false
    default = 0
  }

  column "created_by_staff_id" {
    type = uuid
    null = false
  }

  column "last_error" {
    type = text
    null = true
  }

  column "applied_at" {
    type = timestamptz
    null = true
  }

  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  column "updated_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  foreign_key "finance_import_batches_created_by_staff_id_fkey" {
    columns     = [column.created_by_staff_id]
    ref_columns = [table.staff_accounts.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }

  index "finance_import_batches_file_checksum_sha256_key" {
    unique  = true
    columns = [column.file_checksum_sha256]
  }

  index "idx_finance_import_batches_status_created" {
    columns = [
      column.status,
      column.created_at,
    ]
  }

  index "idx_finance_import_batches_created_by" {
    columns = [
      column.created_by_staff_id,
      column.created_at,
    ]
  }

  check "finance_import_batches_filename_not_blank" {
    expr = "length(trim(source_filename)) > 0"
  }

  check "finance_import_batches_checksum_valid" {
    expr = "length(file_checksum_sha256) = 64"
  }

  check "finance_import_batches_template_version_positive" {
    expr = "template_version > 0"
  }

  check "finance_import_batches_status_valid" {
    expr = "status IN ('validating', 'ready', 'applying', 'completed', 'failed')"
  }

  check "finance_import_batches_counts_nonnegative" {
    expr = "total_rows >= 0 AND valid_rows >= 0 AND invalid_rows >= 0 AND applied_expenses >= 0 AND applied_cost_updates >= 0 AND skipped_rows >= 0"
  }

  check "finance_import_batches_updated_at_valid" {
    expr = "updated_at >= created_at"
  }
}

table "finance_import_rows" {
  schema = schema.public

  column "id" {
    type    = uuid
    null    = false
    default = sql("gen_random_uuid()")
  }

  column "batch_id" {
    type = uuid
    null = false
  }

  column "row_number" {
    type = integer
    null = false
  }

  # expense | variant_cost
  column "record_type" {
    type = varchar(30)
    null = false
  }

  # valid | invalid | applied | skipped | failed
  column "status" {
    type = varchar(20)
    null = false
  }

  # Original normalized workbook cells for audit/debugging.
  column "raw_data" {
    type = jsonb
    null = false
  }

  # Validated backend representation ready for application.
  column "normalized_data" {
    type = jsonb
    null = true
  }

  column "error_details" {
    type = jsonb
    null = true
  }

  # Exactly one of these will be populated when an import row is
  # successfully applied.
  column "applied_expense_id" {
    type = uuid
    null = true
  }

  column "applied_cost_history_id" {
    type = uuid
    null = true
  }

  column "created_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  column "updated_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  foreign_key "finance_import_rows_batch_id_fkey" {
    columns     = [column.batch_id]
    ref_columns = [table.finance_import_batches.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }

  foreign_key "finance_import_rows_applied_expense_id_fkey" {
    columns     = [column.applied_expense_id]
    ref_columns = [table.finance_expenses.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }

  foreign_key "finance_import_rows_applied_cost_history_id_fkey" {
    columns     = [column.applied_cost_history_id]
    ref_columns = [table.finance_variant_cost_history.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }

  index "finance_import_rows_batch_row_key" {
    unique = true
    columns = [
      column.batch_id,
      column.row_number,
    ]
  }

  index "idx_finance_import_rows_batch_status" {
    columns = [
      column.batch_id,
      column.status,
    ]
  }

  index "idx_finance_import_rows_applied_expense" {
    columns = [column.applied_expense_id]
    where   = "applied_expense_id IS NOT NULL"
  }

  index "idx_finance_import_rows_applied_cost_history" {
    columns = [column.applied_cost_history_id]
    where   = "applied_cost_history_id IS NOT NULL"
  }

  check "finance_import_rows_row_number_positive" {
    expr = "row_number > 0"
  }

  check "finance_import_rows_record_type_valid" {
    expr = "record_type IN ('expense', 'variant_cost')"
  }

  check "finance_import_rows_status_valid" {
    expr = "status IN ('valid', 'invalid', 'applied', 'skipped', 'failed')"
  }

  check "finance_import_rows_single_applied_target" {
    expr = "num_nonnulls(applied_expense_id, applied_cost_history_id) <= 1"
  }

  check "finance_import_rows_updated_at_valid" {
    expr = "updated_at >= created_at"
  }
}