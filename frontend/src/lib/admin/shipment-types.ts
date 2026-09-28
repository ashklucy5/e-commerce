import type { AdminPaginationMeta } from "./order-types";

export type AdminShipmentQueueItem = {
  shipment_id?: string;
  order_id: string;
  order_number: string;
  order_status: string;
  payment_status: string;
  payment_method: string;
  customer_name: string;
  customer_phone: string;
  customer_email?: string;
  currency: string;
  total_amount: number;
  shipping_city: string;
  shipping_area: string;
  delivery_method: string;
  delivery_mode?: string;
  provider_code?: string;
  provider_shipment_id?: string;
  provider_status?: string;
  courier_name?: string;
  courier_reference?: string;
  rider_reference?: string;
  tracking_number?: string;
  tracking_url?: string;
  shipment_status: string;
  warehouse_id?: string;
  warehouse_code?: string;
  warehouse_name?: string;
  latest_event_code?: string;
  latest_event_status?: string;
  latest_event_message?: string;
  latest_event_at?: string;
  shipped_at?: string;
  awaiting_confirmation_at?: string;
  delivered_at?: string;
  created_at: string;
  updated_at: string;
};

export type AdminShipmentQueueResponse = {
  data: AdminShipmentQueueItem[];
  meta: AdminPaginationMeta;
};

export type AdminShipmentSummary = {
  needs_shipment: number;
  pending: number;
  shipped: number;
  awaiting_confirmation: number;
  delivered: number;
  cancelled: number;
};

export type AdminShipmentSummaryResponse = {
  data: AdminShipmentSummary;
};
