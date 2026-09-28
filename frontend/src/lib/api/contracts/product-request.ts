// Location: src/lib/api/contracts/product-request.ts

import type { Order } from "@/lib/api/contracts/commerce";

export type ProductRequestStatus =
  | "pending_review"
  | "on_hold"
  | "accepted"
  | "negotiating"
  | "agreed"
  | "converted_to_order"
  | "cancelled";

export type ProductRequest = {
  id: string;
  request_number: string;

  requested_product_name: string;
  description: string;
  requested_quantity: number;

  customer_requirements?: Record<string, unknown>;
  external_url?: string;
  attachments?: unknown[];

  status: ProductRequestStatus | string;
  status_reason?: string;

  can_message: boolean;
  last_message_at: string;
  reviewed_at?: string;

  created_at: string;
  updated_at: string;
};

export type ProductRequestMessage = {
  id: string;

  author_type: "customer" | "support" | string;
  visibility: "customer" | string;
  body: string;

  support_actor_code?: string;
  support_actor_type?: string;
  support_actor_name?: string;

  created_at: string;
  attachments?: unknown[];
};

export type ProductRequestListMeta = {
  limit: number;
  offset: number;
};

export type ProductRequestDataResponse<T> = {
  data: T;
};

export type ProductRequestListResponse = {
  data: ProductRequest[];
  meta: ProductRequestListMeta;
};

export type ProductRequestMessagesResponse = {
  data: ProductRequestMessage[];
  meta?: ProductRequestListMeta;
};

export type CreateProductRequestInput = {
  requested_product_name: string;
  description: string;
  requested_quantity: number;

  customer_requirements?: Record<string, unknown>;
  external_url?: string;
  attachments?: unknown[];
};

export type AddProductRequestMessageInput = {
  message: string;
  attachments?: unknown[];
};

export type SourcingOfferStatus =
  | "draft"
  | "sent"
  | "accepted"
  | "rejected"
  | "customer_accepted"
  | "customer_rejected"
  | "superseded"
  | "expired"
  | "finalized";

export type SourcingOffer = {
  id: string;
  request_id: string;
  status: SourcingOfferStatus | string;

  product_name: string;
  description: string;

  attachments?: unknown[];
  offered_specifications?: Record<string, unknown>;

  unit_price: number;
  shipping_price: number;
  currency: string;

  quoted_quantity?: number;
  minimum_order_quantity?: number;

  expires_at?: string;
  sent_at?: string;
  customer_responded_at?: string;
  finalized_at?: string;

  created_at: string;
  updated_at: string;
};

export type SourcingOffersResponse = {
  data: SourcingOffer[];
};

export type SourcingOfferResponse = {
  data: SourcingOffer;
};

export type SourcingConfirmation = {
  id: string;
  request_id: string;
  offer_id: string;

  quantity: number;
  minimum_order_quantity: number;

  accepted_product_name: string;
  accepted_specifications?: Record<string, unknown>;

  unit_price_snapshot: number;
  shipping_price_snapshot: number;
  currency: string;
  total_amount: number;

  status: "confirmed" | "order_created" | "cancelled" | string;

  created_product_id?: string;
  created_variant_id?: string;
  created_order_id?: string;

  created_at: string;
  updated_at: string;
};

export type SourcingConfirmationResponse = {
  data: SourcingConfirmation;
};

/*
 * The normal frontend Order contract currently assumes checkout_id
 * and cart_id always exist.
 *
 * Sourcing orders are deliberately cartless, so keep the sourcing
 * response accurate here until the shared Order contract is updated
 * when we wire sourcing into My Orders.
 */
export type SourcingOrder =
  Order & {
    order_type:
      | "sourcing"
      | string;
  };

export type PlaceSourcingOrderInput = {
  customer_name: string;
  customer_phone: string;
  customer_email?: string;

  shipping_address_line1: string;
  shipping_address_line2?: string;
  shipping_city: string;
  shipping_area: string;
  shipping_postal_code?: string;

  payment_method: string;
};

export type PlaceSourcingOrderResult = {
  order: SourcingOrder;
  created: boolean;
};

export type PlaceSourcingOrderResponse = {
  data: PlaceSourcingOrderResult;
};