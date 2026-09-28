export type AdminCRMCaseStatus =
  | "waiting_support"
  | "waiting_customer"
  | "resolved"
  | "closed";

export type AdminCRMCasePriority =
  | "low"
  | "normal"
  | "high"
  | "urgent";

export type AdminCRMQueue = {
  id: string;
  code: string;
  name: string;
  description?: string;
  membership_role?: string;
};

export type AdminCRMAttachment = {
  id: string;
  kind: "image" | "link";
  type?: "image" | "link";
  name: string;
  url: string;
  mime_type?: string;
  size?: number;
};

export type AdminCRMTicketActor = {
  id: string;
  actor_code: string;
  actor_type: string;
  display_name: string;
};

export type AdminCRMTicketAssignment = {
  id: string;
  assignment_type: string;
  assigned_actor?: AdminCRMTicketActor;
  assigned_at: string;
  accepted_at?: string;
};

export type AdminCRMTicketQueue = {
  id: string;
  code: string;
  name: string;
  description?: string;
};

export type AdminCRMCase = {
  id: string;
  case_number: string;
  customer_id: string;
  case_type: string;
  subject: string;
  status: AdminCRMCaseStatus;
  priority: AdminCRMCasePriority;
  product_id?: string;
  variant_id?: string;
  order_id?: string;
  requested_quantity?: number;
  available_quantity_snapshot?: number;
  context_snapshot?: unknown;
  queue: AdminCRMTicketQueue;
  assignment: AdminCRMTicketAssignment;
  last_message_at: string;
  last_customer_message_at?: string;
  last_support_message_at?: string;
  resolved_at?: string;
  closed_at?: string;
  created_at: string;
  updated_at: string;
};

export type AdminCRMMessage = {
  id: string;
  case_id: string;
  author_type: string;
  visibility: "customer" | "internal" | string;
  body: string;
  support_actor?: AdminCRMTicketActor;
  attachments?: unknown;
  created_at: string;
};

export type AdminCRMQueuesResponse = {
  data: AdminCRMQueue[];
};

export type AdminCRMCasesResponse = {
  data: AdminCRMCase[];
  meta: {
    limit: number;
    offset: number;
  };
};

export type AdminCRMCaseResponse = {
  data: AdminCRMCase;
};

export type AdminCRMMessagesResponse = {
  data: AdminCRMMessage[];
  meta: {
    limit: number;
    offset: number;
  };
};

export type AdminCRMMessageResponse = {
  data: AdminCRMMessage;
};

export type AdminCRMEscalationResponse = {
  data: {
    case_id: string;
    status: AdminCRMCaseStatus;
    queue_code: string;
  };
};