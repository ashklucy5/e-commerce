export type AdminCustomerStatus = "active" | "disabled";

export type AdminPaginationMeta = {
  page: number;
  limit: number;
  total: number;
  total_pages: number;
  has_next: boolean;
  has_previous: boolean;
};

export type AdminCustomerListItem = {
  id: string;
  phone: string;
  email?: string;
  full_name: string;
  status: AdminCustomerStatus;
  order_count: number;
  review_count: number;
  active_session_count: number;
  last_order_at?: string;
  created_at: string;
  updated_at: string;
};

export type AdminCustomerAddress = {
  id: string;
  label: string;
  recipient_name: string;
  phone: string;
  address_line1: string;
  address_line2?: string;
  city: string;
  area: string;
  postal_code?: string;
  is_default: boolean;
  created_at: string;
  updated_at: string;
};

export type AdminCustomerOrderSummary = {
  id: string;
  order_number: string;
  status: string;
  payment_status: string;
  total_amount: number;
  currency: string;
  created_at: string;
};

export type AdminCustomerDetail = AdminCustomerListItem & {
  delivered_or_completed_orders: number;
  addresses: AdminCustomerAddress[];
  recent_orders: AdminCustomerOrderSummary[];
};

export type AdminCustomersResponse = {
  data: AdminCustomerListItem[];
  meta: AdminPaginationMeta;
};

export type AdminCustomerResponse = {
  data: AdminCustomerDetail;
};
