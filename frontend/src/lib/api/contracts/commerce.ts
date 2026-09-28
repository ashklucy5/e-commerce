export type CartItem = {
  id: string;
  variant_id: string;
  product_id: string;
  sku: string;
  product_name: string;
  product_slug: string;
  color_name?: string;
  size?: string;
  image_url?: string;
  quantity: number;
  minimum_order_quantity: number;
  order_increment: number;
  available_quantity: number;
  unit_price_amount: number;
  line_total_amount: number;
  currency: string;
  is_available: boolean;
  created_at: string;
  updated_at: string;
};

export type CartTotals = {
  currency: string;
  item_count: number;
  quantity_total: number;
  subtotal_amount: number;
};

export type Cart = {
  id: string;
  cart_key: string;
  status: string;
  currency: string;
  expires_at?: string;
  created_at: string;
  updated_at: string;
  items: CartItem[];
  totals: CartTotals;
};

export type CheckoutItem = {
  id: string;
  checkout_id: string;
  variant_id: string;
  sku: string;
  product_name: string;
  quantity: number;
  minimum_order_quantity: number;
  unit_price_amount: number;
  line_total_amount: number;
  currency: string;
  created_at: string;
};

export type CheckoutSession = {
  id: string;
  checkout_key: string;
  cart_id: string;
  status: string;
  currency: string;
  subtotal_amount: number;
  discount_amount: number;
  promotion_id?: string;
  promotion_code?: string;
  shipping_amount: number;
  total_amount: number;
  customer_name?: string;
  customer_phone?: string;
  customer_email?: string;
  shipping_address_line1?: string;
  shipping_address_line2?: string;
  shipping_city?: string;
  shipping_area?: string;
  shipping_postal_code?: string;
  delivery_method?: string;
  payment_method?: string;
  expires_at: string;
  completed_at?: string;
  cancelled_at?: string;
  created_at: string;
  updated_at: string;
  item_count: number;
  quantity_total: number;
  items: CheckoutItem[];
};

export type DeliveryOption = {
  code: string;
  label: string;
  shipping_amount: number;
  currency: string;
  estimated_min_minutes: number;
  estimated_max_minutes: number;
  selected: boolean;
};

export type PaymentOption = {
  code: string;
  label: string;
  requires_immediate_payment: boolean;
  payable_amount: number;
  currency: string;
};

export type OrderItem = {
  id: string;
  order_id: string;
  variant_id: string;
  sku: string;
  product_name: string;
  quantity: number;
  minimum_order_quantity: number;
  unit_price_amount: number;
  line_total_amount: number;
  currency: string;
  created_at: string;
};

export type Shipment = {
  id: string;
  order_id: string;
  origin_warehouse_id?: string;
  delivery_mode: string;
  provider_code?: string;
  provider_shipment_id?: string;
  provider_status?: string;
  courier_name?: string;
  courier_reference?: string;
  rider_reference?: string;
  tracking_number?: string;
  tracking_url?: string;
  status: string;
  shipped_at?: string;
  provider_delivered_at?: string;
  awaiting_confirmation_at?: string;
  confirmed_received_at?: string;
  delivered_at?: string;
  last_provider_sync_at?: string;
  confirmation_source?: string;
  confirmed_by_actor_id?: string;
  confirmation_note?: string;
  created_at: string;
  updated_at: string;
};

export type Order = {
  id: string;
  order_number: string;
  order_type:
    | "standard"
    | "sourcing"
    | string;

  checkout_id?: string;
  cart_id?: string;
  status: string;
  payment_status: string;
  payment_method: string;
  currency: string;
  subtotal_amount: number;
  discount_amount: number;
  promotion_id?: string;
  promotion_code?: string;
  shipping_amount: number;
  total_amount: number;
  customer_name: string;
  customer_phone: string;
  customer_email?: string;
  shipping_address_line1: string;
  shipping_address_line2?: string;
  shipping_city: string;
  shipping_area: string;
  shipping_postal_code?: string;
  delivery_method: string;
  payment_due_at?: string;
  paid_at?: string;
  confirmed_at?: string;
  processing_at?: string;
  shipped_at?: string;
  delivered_at?: string;
  completed_at?: string;
  cancelled_at?: string;
  cancellation_reason?: string;
  cancelled_by?: string;
  shipment?: Shipment;
  created_at: string;
  updated_at: string;
  item_count: number;
  quantity_total: number;
  items: OrderItem[];
};

export type ApiData<T> = {
  data: T;
};

export type PlaceOrderResponse = {
  data: Order;
  meta: {
    created: boolean;
  };
};
