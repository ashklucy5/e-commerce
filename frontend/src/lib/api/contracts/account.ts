// Location: src/lib/api/contracts/account.ts

export type AccountCustomer = {
  id: string;
  phone: string;
  email?: string;
  full_name: string;
  status: string;
  created_at: string;
  updated_at: string;
  avatar_url?: string;
  avatar_url_expires_at?: string;
  avatar_updated_at?: string;
};

export type AccountAuthTokens = {
  access_token: string;
  refresh_token: string;
  token_type: string;
  access_expires_at: string;
  refresh_expires_at: string;
};

export type AccountAuthResult = {
  customer: AccountCustomer;
  tokens: AccountAuthTokens;
};

export type AccountAuthResponse = {
  data: AccountAuthResult;
};

export type AccountDataResponse<T> = {
  data: T;
};

export type AccountPaginationMeta = {
  page: number;
  limit: number;
  total: number;
  total_pages: number;
};

export type AccountAddress = {
  id: string;
  customer_id: string;
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

export type CreateAccountAddressRequest = {
  label: string;
  recipient_name: string;
  phone: string;
  address_line1: string;
  address_line2: string;
  city: string;
  area: string;
  postal_code: string;
  is_default: boolean;
};

export type UpdateAccountAddressRequest = Partial<CreateAccountAddressRequest>;

export type AccountOrderHistoryItem = {
  id: string;
  order_number: string;
  order_type?:
    | "standard"
    | "sourcing"
    | string;

  status: string;
  payment_status: string;
  payment_method: string;
  currency: string;
  subtotal_amount: number;
  discount_amount: number;
  shipping_amount: number;
  total_amount: number;
  delivery_method: string;
  item_count: number;
  quantity_total: number;
  first_product_name?: string;
  created_at: string;
  updated_at: string;
};

export type AccountOrderHistoryResponse = {
  data: AccountOrderHistoryItem[];
  meta: AccountPaginationMeta;
};

export type AccountWishlistItem = {
  id: string;
  product_id: string;
  name: string;
  slug?: string;
  product_slug?: string;
  primary_image_url?: string;
  image_url?: string;
  price_amount: number;
  currency: string;
  in_stock: boolean;
  is_available: boolean;
  created_at?: string;
  updated_at?: string;
};

export type AccountWishlistResponse = {
  data: AccountWishlistItem[];
  meta?: AccountPaginationMeta;
};

export type AccountReturnItem = {
  id: string;
  return_number?: string;
  order_id: string;
  order_number?: string;
  status: string;
  reason?: string;
  requested_quantity?: number;
  currency?: string;
  refund_amount?: number;
  requested_at?: string;
  created_at: string;
  updated_at?: string;
};

export type AccountReturnsResponse = {
  data: AccountReturnItem[];
  meta?: AccountPaginationMeta;
};

export type AccountSecurityOverview = {
  password_updated_at?: string;
  last_login_at?: string;
  active_session_count?: number;
  session_count?: number;
  phone_verified?: boolean;
  email_verified?: boolean;
  mfa_enabled?: boolean;
};

export type AccountSession = {
  id: string;
  is_current?: boolean;
  device_name?: string;
  user_agent?: string;
  ip_address?: string;
  created_at: string;
  last_seen_at?: string;
  expires_at?: string;
};

export type AccountSessionsResponse = {
  data: AccountSession[];
};

export type AccountNotificationPreferences = {
  order_updates_sms: boolean;
  order_updates_email: boolean;
  delivery_updates_sms: boolean;
  delivery_updates_email: boolean;
  support_updates_sms: boolean;
  support_updates_email: boolean;
  promotions_sms?: boolean;
  promotions_email?: boolean;
  recommendations_email?: boolean;
  push_enabled?: boolean;
};

export type AccountPrivacySettings = {
  personalization_enabled?: boolean;
  search_history_enabled: boolean;
  recently_viewed_enabled: boolean;
  save_tryon_media?: boolean;
  use_tryon_for_personalization?: boolean;
};

export type AccountPreferences = {
  locale: string;
  assistant_language?: string;
  currency: string;
};

export type AccountSupportCase = {
  id: string;
  case_number?: string;
  customer_id?: string;
  case_type?: string;
  type?: string;
  subject: string;
  status: string;
  priority?: string;
  product_id?: string;
  variant_id?: string;
  order_id?: string;
  requested_quantity?: number;
  available_quantity_snapshot?: number;
  last_message_at?: string;
  last_customer_message_at?: string;
  last_support_message_at?: string;
  resolved_at?: string;
  closed_at?: string;
  created_at: string;
  updated_at?: string;
};

export type AccountSupportCasesResponse = {
  data: AccountSupportCase[];
};

export type AccountSupportCaseResponse = {
  data: AccountSupportCase;
};

export type AccountSupportMessage = {
  id: string;
  case_id: string;

  author_type:
    | "customer"
    | "support"
    | string;

  visibility?: string;

  body: string;

  attachments?: unknown;

  created_at: string;
};

export type AccountSupportMessagesResponse = {
  data: AccountSupportMessage[];
};
