export type AdminNotification = {
  id: string;
  category: string;
  event_type: string;
  priority: "info" | "attention" | "critical" | string;
  title: string;
  message: string;
  action_url?: string;
  entity_type?: string;
  entity_id?: string;
  metadata?: Record<string, unknown>;
  read_at?: string | null;
  created_at: string;
};

export type AdminNotificationListResponse = {
  data: AdminNotification[];
  meta: {
    limit: number;
    offset: number;
  };
};

export type AdminNotificationSummaryResponse = {
  data: {
    unread_count: number;
  };
};

export type AdminNotificationReadResponse = {
  data: {
    read: boolean;
  };
};

export type AdminNotificationReadAllResponse = {
  data: {
    updated: number;
  };
};