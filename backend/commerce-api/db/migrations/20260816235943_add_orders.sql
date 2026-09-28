-- Create "orders" table
CREATE TABLE "orders" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "order_number" character varying(50) NOT NULL,
  "checkout_id" uuid NOT NULL,
  "cart_id" uuid NOT NULL,
  "status" character varying(30) NOT NULL DEFAULT 'pending_payment',
  "payment_status" character varying(30) NOT NULL DEFAULT 'pending',
  "payment_method" character varying(80) NOT NULL,
  "currency" character varying(3) NOT NULL,
  "subtotal_amount" bigint NOT NULL,
  "discount_amount" bigint NOT NULL DEFAULT 0,
  "shipping_amount" bigint NOT NULL DEFAULT 0,
  "total_amount" bigint NOT NULL,
  "customer_name" character varying(160) NOT NULL,
  "customer_phone" character varying(40) NOT NULL,
  "customer_email" character varying(255) NULL,
  "shipping_address_line1" character varying(255) NOT NULL,
  "shipping_address_line2" character varying(255) NULL,
  "shipping_city" character varying(120) NOT NULL,
  "shipping_area" character varying(120) NOT NULL,
  "shipping_postal_code" character varying(30) NULL,
  "delivery_method" character varying(80) NOT NULL,
  "payment_due_at" timestamptz NULL,
  "paid_at" timestamptz NULL,
  "confirmed_at" timestamptz NULL,
  "cancelled_at" timestamptz NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "orders_cart_id_fkey" FOREIGN KEY ("cart_id") REFERENCES "carts" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "orders_checkout_id_fkey" FOREIGN KEY ("checkout_id") REFERENCES "checkout_sessions" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "orders_currency_valid" CHECK ((length((currency)::text) = 3) AND ((currency)::text = upper((currency)::text))),
  CONSTRAINT "orders_customer_name_not_blank" CHECK (length(TRIM(BOTH FROM customer_name)) > 0),
  CONSTRAINT "orders_customer_phone_not_blank" CHECK (length(TRIM(BOTH FROM customer_phone)) > 0),
  CONSTRAINT "orders_delivery_method_not_blank" CHECK (length(TRIM(BOTH FROM delivery_method)) > 0),
  CONSTRAINT "orders_discount_nonnegative" CHECK (discount_amount >= 0),
  CONSTRAINT "orders_discount_not_above_subtotal" CHECK (discount_amount <= subtotal_amount),
  CONSTRAINT "orders_order_number_not_blank" CHECK (length(TRIM(BOTH FROM order_number)) > 0),
  CONSTRAINT "orders_payment_method_not_blank" CHECK (length(TRIM(BOTH FROM payment_method)) > 0),
  CONSTRAINT "orders_payment_status_valid" CHECK ((payment_status)::text = ANY ((ARRAY['pending'::character varying, 'paid'::character varying, 'failed'::character varying, 'expired'::character varying, 'cod_pending'::character varying, 'cod_collected'::character varying, 'refunded'::character varying])::text[])),
  CONSTRAINT "orders_pending_payment_has_deadline" CHECK (((status)::text <> 'pending_payment'::text) OR (payment_due_at IS NOT NULL)),
  CONSTRAINT "orders_shipping_address_line1_not_blank" CHECK (length(TRIM(BOTH FROM shipping_address_line1)) > 0),
  CONSTRAINT "orders_shipping_area_not_blank" CHECK (length(TRIM(BOTH FROM shipping_area)) > 0),
  CONSTRAINT "orders_shipping_city_not_blank" CHECK (length(TRIM(BOTH FROM shipping_city)) > 0),
  CONSTRAINT "orders_shipping_nonnegative" CHECK (shipping_amount >= 0),
  CONSTRAINT "orders_status_valid" CHECK ((status)::text = ANY ((ARRAY['pending_payment'::character varying, 'confirmed'::character varying, 'processing'::character varying, 'shipped'::character varying, 'delivered'::character varying, 'completed'::character varying, 'payment_expired'::character varying, 'cancelled'::character varying])::text[])),
  CONSTRAINT "orders_subtotal_nonnegative" CHECK (subtotal_amount >= 0),
  CONSTRAINT "orders_total_consistent" CHECK (total_amount = ((subtotal_amount - discount_amount) + shipping_amount)),
  CONSTRAINT "orders_total_nonnegative" CHECK (total_amount >= 0)
);
-- Create index "idx_orders_customer_phone_created" to table: "orders"
CREATE INDEX "idx_orders_customer_phone_created" ON "orders" ("customer_phone", "created_at");
-- Create index "idx_orders_payment_status_created" to table: "orders"
CREATE INDEX "idx_orders_payment_status_created" ON "orders" ("payment_status", "created_at");
-- Create index "idx_orders_pending_payment_due" to table: "orders"
CREATE INDEX "idx_orders_pending_payment_due" ON "orders" ("payment_due_at") WHERE ((status)::text = 'pending_payment'::text);
-- Create index "idx_orders_status_created" to table: "orders"
CREATE INDEX "idx_orders_status_created" ON "orders" ("status", "created_at");
-- Create index "orders_checkout_id_key" to table: "orders"
CREATE UNIQUE INDEX "orders_checkout_id_key" ON "orders" ("checkout_id");
-- Create index "orders_order_number_key" to table: "orders"
CREATE UNIQUE INDEX "orders_order_number_key" ON "orders" ("order_number");
-- Create "order_items" table
CREATE TABLE "order_items" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "order_id" uuid NOT NULL,
  "variant_id" uuid NOT NULL,
  "sku" character varying(100) NOT NULL,
  "product_name" character varying(180) NOT NULL,
  "quantity" integer NOT NULL,
  "minimum_order_quantity" integer NOT NULL,
  "unit_price_amount" bigint NOT NULL,
  "line_total_amount" bigint NOT NULL,
  "currency" character varying(3) NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "order_items_order_id_fkey" FOREIGN KEY ("order_id") REFERENCES "orders" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "order_items_variant_id_fkey" FOREIGN KEY ("variant_id") REFERENCES "product_variants" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "order_items_currency_valid" CHECK ((length((currency)::text) = 3) AND ((currency)::text = upper((currency)::text))),
  CONSTRAINT "order_items_line_total_consistent" CHECK (line_total_amount = (unit_price_amount * quantity)),
  CONSTRAINT "order_items_line_total_nonnegative" CHECK (line_total_amount >= 0),
  CONSTRAINT "order_items_minimum_order_quantity_positive" CHECK (minimum_order_quantity > 0),
  CONSTRAINT "order_items_product_name_not_blank" CHECK (length(TRIM(BOTH FROM product_name)) > 0),
  CONSTRAINT "order_items_quantity_meets_minimum" CHECK (quantity >= minimum_order_quantity),
  CONSTRAINT "order_items_quantity_positive" CHECK (quantity > 0),
  CONSTRAINT "order_items_sku_not_blank" CHECK (length(TRIM(BOTH FROM sku)) > 0),
  CONSTRAINT "order_items_unit_price_nonnegative" CHECK (unit_price_amount >= 0)
);
-- Create index "idx_order_items_variant" to table: "order_items"
CREATE INDEX "idx_order_items_variant" ON "order_items" ("variant_id");
-- Create index "order_items_order_variant_key" to table: "order_items"
CREATE UNIQUE INDEX "order_items_order_variant_key" ON "order_items" ("order_id", "variant_id");
