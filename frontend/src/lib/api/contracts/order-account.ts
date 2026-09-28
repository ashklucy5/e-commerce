// Location: src/lib/api/contracts/order-account.ts

export type AccountOrderItem = {
  id: string;
  order_id: string;
  variant_id: string;
  sku: string;
  product_name: string;
  /** Enriched by the storefront order-detail BFF from the public catalog. */
  image_url?: string;
  product_slug?: string;
  quantity: number;
  minimum_order_quantity: number;
  unit_price_amount: number;
  line_total_amount: number;
  currency: string;
  created_at: string;
};

export type AccountOrderShipment = {
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
  confirmation_note?: string;
  created_at: string;
  updated_at: string;
};

export type AccountOrder = {
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
  shipment?: AccountOrderShipment;
  created_at: string;
  updated_at: string;
  item_count: number;
  quantity_total: number;
  items: AccountOrderItem[];
};

export type AccountOrderResponse = {
  data: AccountOrder;
};

export type AccountOrderEvent = {
  id: string;
  order_id: string;
  event_type: string;
  from_status?: string;
  to_status?: string;
  message?: string;
  actor_type?: string;
  actor_id?: string;
  created_at: string;
};

export type AccountOrderTimeline = {
  order_id: string;
  order_number: string;
  status: string;
  events: AccountOrderEvent[];
};

export type AccountOrderTimelineResponse = {
  data: AccountOrderTimeline;
};

export type AccountOrderTracking = {
  order_id?: string;
  current_stage?: string;
  stage?: string;
  status?: string;
  shipment?: AccountOrderShipment;
  events?: unknown[];
  timeline?: unknown[];
  updates?: unknown[];
  tracking_events?: unknown[];
  [key: string]: unknown;
};

export type AccountOrderTrackingResponse = {
  data: AccountOrderTracking;
};
