export type AdminProductRequestStatus =
  | "pending_review"
  | "on_hold"
  | "accepted"
  | "negotiating"
  | "agreed"
  | "converted_to_order"
  | "cancelled";

export type AdminProductRequestCustomer = {
  id: string;
  full_name: string;
  phone: string;
  email?: string;
};

export type AdminProductRequestActor = {
  id: string;
  actor_code: string;
  actor_type: string;
  display_name: string;
};

export type AdminProductRequestAssignment = {
  queue_code: string;
  queue_name: string;
  support_actor?: AdminProductRequestActor;
};

export type AdminProductRequest = {
  id: string;
  request_number: string;
  case_id: string;
  case_number: string;
  requested_product_name: string;
  description: string;
  requested_quantity: number;
  customer_requirements?: unknown;
  external_url?: string;
  attachments?: unknown;
  status: AdminProductRequestStatus;
  status_reason?: string;
  crm_status: string;
  crm_priority: string;
  customer: AdminProductRequestCustomer;
  reviewed_by?: AdminProductRequestActor;
  assignment?: AdminProductRequestAssignment;
  last_message_at: string;
  last_customer_message_at?: string;
  last_support_message_at?: string;
  reviewed_at?: string;
  created_at: string;
  updated_at: string;
};

export type AdminProductRequestMessage = {
  id: string;
  author_type: string;
  visibility: "customer" | "internal" | string;
  body: string;
  support_actor?: AdminProductRequestActor;
  created_at: string;
  attachments?: unknown;
};

export type AdminSourcingOfferStatus =
  | "draft"
  | "sent"
  | "customer_accepted"
  | "customer_rejected"
  | "superseded"
  | "expired"
  | "finalized"
  | string;

export type AdminSourcingOffer = {
  id: string;
  request_id: string;
  status: AdminSourcingOfferStatus;
  product_name: string;
  description: string;
  attachments?: unknown;
  offered_specifications?: unknown;
  unit_price: number;
  shipping_price: number;
  currency: string;
  quoted_quantity?: number;
  minimum_order_quantity?: number;
  expires_at?: string;
  sent_at?: string;
  customer_responded_at?: string;
  finalized_at?: string;
  created_by?: AdminProductRequestActor;
  created_at: string;
  updated_at: string;
};

export type AdminSourcingConfirmation = {
  id: string;
  request_id: string;
  offer_id: string;
  quantity: number;
  minimum_order_quantity: number;
  accepted_product_name: string;
  accepted_specifications?: unknown;
  unit_price_snapshot: number;
  shipping_price_snapshot: number;
  currency: string;
  total_amount: number;
  status: string;
  finalized_by?: AdminProductRequestActor;
  created_product_id?: string;
  created_variant_id?: string;
  created_order_id?: string;
  created_at: string;
  updated_at: string;
};

export type AdminProductRequestListResponse = {
  data: AdminProductRequest[];
  meta: {
    limit: number;
    offset: number;
  };
};

export type AdminProductRequestResponse = {
  data: AdminProductRequest;
};

export type AdminProductRequestMessagesResponse = {
  data: AdminProductRequestMessage[];
  meta: {
    limit: number;
    offset: number;
  };
};

export type AdminSourcingOffersResponse = {
  data: AdminSourcingOffer[];
};

export type AdminSourcingOfferResponse = {
  data: AdminSourcingOffer;
};

export type AdminSourcingConfirmationResponse = {
  data: AdminSourcingConfirmation;
};

export type AdminSourcingOfferMutationPayload = {
  product_name: string;
  description: string;
  attachments: unknown[];
  offered_specifications: Record<string, string>;
  unit_price: number;
  shipping_price: number;
  currency: string;
  quoted_quantity: number;
  minimum_order_quantity: number;
  expires_at: string | null;
};
