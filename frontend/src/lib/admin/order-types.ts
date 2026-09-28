export type AdminPaginationMeta = {
  page: number;
  limit: number;
  total: number;
  total_pages: number;
  has_next: boolean;
  has_previous: boolean;
};

export type AdminOrderListItem = {
  id: string;
  order_number: string;
  customer_id?: string;
  customer_name: string;
  customer_phone: string;
  status: string;
  payment_status: string;
  payment_method: string;
  currency: string;
  total_amount: number;
  delivery_method: string;
  created_at: string;
  updated_at: string;
};

export type AdminOrderItem = {
  id: string;
  variant_id: string;
  sku: string;
  product_name: string;
  quantity: number;
  minimum_order_quantity: number;
  unit_price_amount: number;
  line_total_amount: number;
  currency: string;
};

export type AdminOrderPaymentSummary = {
  id: string;
  provider: string;
  status: string;
  amount: number;
  currency: string;
  provider_payment_id: string;
  provider_transaction_id?: string;
  paid_at?: string;
  failed_at?: string;
  created_at: string;
};

export type AdminOrderReturnSummary = {
  id: string;
  return_number: string;
  status: string;
  requested_at: string;
};

export type AdminOrderRefundSummary = {
  id: string;
  refund_number: string;
  return_id?: string;
  payment_id?: string;
  source_type: string;
  status: string;
  amount: number;
  currency: string;
  provider?: string;
  provider_refund_id?: string;
  requested_at: string;
  succeeded_at?: string;
};

export type AdminOrderDetail = AdminOrderListItem & {
  order_type: string;
  checkout_id: string;
  cart_id: string;
  customer_email?: string;
  subtotal_amount: number;
  discount_amount: number;
  promotion_id?: string;
  promotion_code?: string;
  shipping_amount: number;
  shipping_address_line1: string;
  shipping_address_line2?: string;
  shipping_city: string;
  shipping_area: string;
  shipping_postal_code?: string;
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
  items: AdminOrderItem[];
  payments: AdminOrderPaymentSummary[];
  returns: AdminOrderReturnSummary[];
  refunds: AdminOrderRefundSummary[];
};

export type AdminOrdersResponse = {
  data: AdminOrderListItem[];
  meta: AdminPaginationMeta;
};

export type AdminOrderResponse = {
  data: AdminOrderDetail;
};

export type AdminWarehouseFulfillment = {
  id: string;
  order_id: string;
  order_item_id: string;
  order_item_sku?: string;
  product_name?: string;
  warehouse_id: string;
  warehouse_code?: string;
  warehouse_name?: string;
  inbound_shipment_id?: string;
  inbound_reference?: string;
  source: string;
  status: string;
  quantity: number;
  allocated_at: string;
  received_at?: string;
  picking_at?: string;
  packed_at?: string;
  ready_for_handoff_at?: string;
  handed_off_at?: string;
  created_at: string;
  updated_at: string;
};

export type AdminWarehouseFulfillmentsResponse = {
  data: AdminWarehouseFulfillment[];
};

export type AdminDeliveryShipment = {
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

export type AdminDeliveryTrackingEvent = {
  id: string;
  shipment_id: string;
  order_id: string;
  source: string;
  event_code: string;
  status?: string;
  message?: string;
  latitude?: number;
  longitude?: number;
  external_event_id?: string;
  occurred_at: string;
  created_at: string;
};

export type AdminDeliveryTrackingDetail = {
  shipment: AdminDeliveryShipment;
  events: AdminDeliveryTrackingEvent[];
};

export type AdminDeliveryTrackingResponse = {
  data: AdminDeliveryTrackingDetail;
};

export type AdminInvoiceItem = {
  sku: string;
  product_name: string;
  quantity: number;
  unit_price_amount: number;
  line_total_amount: number;
  currency: string;
};

export type AdminInvoice = {
  id: string;
  invoice_number: string;
  issued_at: string;
  order_id: string;
  order_number: string;
  currency: string;
  subtotal_amount: number;
  discount_amount: number;
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
  payment_method: string;
  payment_status_at_issue: string;
  items: AdminInvoiceItem[];
};

export type AdminInvoiceResponse = {
  data: AdminInvoice;
};
