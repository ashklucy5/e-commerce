export type CustomerPreferences = {
  customer_id: string;
  locale: "en" | "bn" | string;
  assistant_language: "auto" | "en" | "bn" | "banglish" | string;
  currency: string;
};

export type CustomerNotificationPreferences = {
  customer_id: string;
  order_updates_sms: boolean;
  order_updates_email: boolean;
  delivery_updates_sms: boolean;
  delivery_updates_email: boolean;
  support_updates_sms: boolean;
  support_updates_email: boolean;
  promotions_sms: boolean;
  promotions_email: boolean;
  recommendations_email: boolean;
  push_enabled: boolean;
};

export type CustomerPrivacySettings = {
  customer_id: string;
  personalization_enabled: boolean;
  search_history_enabled: boolean;
  recently_viewed_enabled: boolean;
  save_tryon_media: boolean;
  use_tryon_for_personalization: boolean;
};
