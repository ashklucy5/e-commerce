export type AdminReturnStatus =
  | "requested"
  | "approved"
  | "rejected"
  | "received"
  | "inspected"
  | "completed"
  | "cancelled";

export type AdminReturnInspectionStatus =
  | "pending"
  | "restockable"
  | "damaged"
  | "non_restockable";

export type AdminRefundStatus =
  | "requested"
  | "approved"
  | "processing"
  | "succeeded"
  | "failed"
  | "cancelled";

export type AdminRefundSourceType =
  | "cancellation"
  | "return"
  | "manual";

export type AdminRefundProvider =
  | "bkash"
  | "nagad"
  | "rocket"
  | "bank_transfer"
  | "manual";

export type AdminPaginationMeta = {
  page: number;
  limit: number;
  total: number;
  total_pages: number;
  has_next: boolean;
  has_previous: boolean;
};

export type AdminReturnListItem = {
  id: string;
  return_number: string;

  order_id: string;
  order_number: string;

  status: AdminReturnStatus | string;

  customer_name: string;
  customer_phone: string;

  requested_at: string;
  updated_at: string;
};

export type AdminReturnListResponse = {
  data: AdminReturnListItem[];
  meta: AdminPaginationMeta;
};

export type AdminReturnItem = {
  id: string;

  order_item_id: string;
  variant_id: string;

  sku: string;
  product_name: string;

  quantity: number;

  reason_code: string;
  reason_note?: string;

  received_quantity: number;
  restock_quantity: number;

  inspection_status:
    | AdminReturnInspectionStatus
    | string;

  inspection_note?: string;

  unit_price_amount: number;
  currency: string;
};

export type AdminRefundSummary = {
  id: string;
  refund_number: string;

  return_id?: string;
  payment_id?: string;

  source_type:
    | AdminRefundSourceType
    | string;

  status:
    | AdminRefundStatus
    | string;

  amount: number;
  currency: string;

  provider?:
    | AdminRefundProvider
    | string;

  provider_refund_id?: string;

  requested_at: string;
  succeeded_at?: string;
};

export type AdminReturnDetail =
  AdminReturnListItem & {
    customer_note?: string;

    requested_by?: string;
    approved_by?: string;
    rejected_by?: string;

    rejection_reason?: string;

    approved_at?: string;
    rejected_at?: string;
    received_at?: string;
    inspected_at?: string;
    completed_at?: string;
    cancelled_at?: string;

    items: AdminReturnItem[];
    refunds: AdminRefundSummary[];
  };

export type AdminReturnDetailResponse = {
  data: AdminReturnDetail;
};

/*
 * Return mutation responses come from the
 * returns domain itself. They do not contain
 * the Admin read-model additions such as
 * order_number/customer_name/refunds.
 */
export type AdminReturnMutationItem = {
  id: string;
  return_id: string;

  order_item_id: string;
  variant_id: string;

  sku: string;
  product_name: string;

  quantity: number;

  reason_code: string;
  reason_note?: string;

  received_quantity: number;
  restock_quantity: number;

  inspection_status:
    | AdminReturnInspectionStatus
    | string;

  inspection_note?: string;

  unit_price_amount: number;
  currency: string;

  created_at: string;
  updated_at: string;
};

export type AdminReturnMutationResult = {
  id: string;
  return_number: string;

  order_id: string;

  status:
    | AdminReturnStatus
    | string;

  customer_note?: string;

  requested_by?: string;
  approved_by?: string;
  rejected_by?: string;

  rejection_reason?: string;

  requested_at: string;
  approved_at?: string;
  rejected_at?: string;
  received_at?: string;
  inspected_at?: string;
  completed_at?: string;
  cancelled_at?: string;

  created_at: string;
  updated_at: string;

  items: AdminReturnMutationItem[];
};

export type AdminReturnMutationResponse = {
  data: AdminReturnMutationResult;
};

export type AdminRejectReturnRequest = {
  reason: string;
};

export type AdminReceiveReturnItemRequest = {
  order_item_id: string;
  received_quantity: number;
};

export type AdminReceiveReturnRequest = {
  items: AdminReceiveReturnItemRequest[];
};

export type AdminInspectReturnItemRequest = {
  order_item_id: string;

  inspection_status:
    | Exclude<
        AdminReturnInspectionStatus,
        "pending"
      >
    | string;

  inspection_note: string;

  restock_quantity: number;
};

export type AdminInspectReturnRequest = {
  items: AdminInspectReturnItemRequest[];
};

/* ---------------------------------
 * Refund write model
 * --------------------------------- */

export type AdminRefund = {
  id: string;
  refund_number: string;

  order_id: string;
  return_id?: string;
  payment_id?: string;

  source_type:
    | AdminRefundSourceType
    | string;

  status:
    | AdminRefundStatus
    | string;

  amount: number;
  currency: string;

  provider?:
    | AdminRefundProvider
    | string;

  provider_refund_id?: string;

  reason: string;

  requested_by?: string;
  approved_by?: string;

  failure_code?: string;
  failure_message?: string;

  requested_at: string;
  approved_at?: string;
  processing_at?: string;
  succeeded_at?: string;
  failed_at?: string;
  cancelled_at?: string;

  created_at: string;
  updated_at: string;
};

export type AdminRefundResponse = {
  data: AdminRefund;
};

export type AdminCreateReturnRefundRequest = {
  reason: string;
};

export type AdminRefundSuccessRequest = {
  provider_refund_id: string;
};

export type AdminRefundFailureRequest = {
  failure_code: string;
  failure_message: string;
};