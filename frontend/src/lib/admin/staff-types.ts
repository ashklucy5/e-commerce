export type AdminStaffStatus =
  | "pending_activation"
  | "active"
  | "suspended"
  | "disabled"
  | "deleted"
  | "banned";

export type AdminStaffMfaStatus =
  | "not_enrolled"
  | "pending"
  | "active"
  | string;

export type AdminPaginationMeta = {
  page: number;
  limit: number;
  total: number;
  total_pages: number;
  has_next: boolean;
  has_previous: boolean;
};

export type AdminStaffListItem = {
  id: string;
  staff_code: string;
  full_name: string;
  email: string;
  phone?: string;
  status: AdminStaffStatus;
  roles: string[];
  has_admin_panel_access: boolean;
  admin_mfa_status: AdminStaffMfaStatus;
  active_staff_sessions: number;
  active_admin_sessions: number;
  last_login_at?: string;
  created_at: string;
  updated_at: string;
};

export type AdminStaffSupportActor = {
  id: string;
  actor_code: string;
  status: string;
  presence: string;
};

export type AdminStaffDetail = AdminStaffListItem & {
  permissions: string[];
  support_actor?: AdminStaffSupportActor;
};

export type AdminStaffListResponse = {
  data: AdminStaffListItem[];
  meta: AdminPaginationMeta;
};

export type AdminStaffDetailResponse = {
  data: AdminStaffDetail;
};

export type AdminStaffInvitationStatus =
  | "pending"
  | "expired"
  | "accepted"
  | "cancelled"
  | string;

export type AdminStaffInvitation = {
  id: string;
  email: string;
  delivery_mode: string;
  delivery_status: string;
  status: AdminStaffInvitationStatus;
  expires_at: string;
  password_set_at?: string;
  accepted_at?: string;
  cancelled_at?: string;
  delivered_at?: string;
  delivery_error?: string;
  created_by_staff_id: string;
  created_at: string;
  updated_at: string;
};

export type AdminStaffInvitationResponse = {
  data: AdminStaffInvitation;
};

export type AdminStaffInvitationDelivery = {
  id: string;
  delivery_mode: string;
  activation_token: string;
  expires_at: string;
};

export type AdminCreateStaffResult = {
  staff: AdminStaffDetail;
  invitation: AdminStaffInvitationDelivery;
};

export type AdminCreateStaffResponse = {
  data: AdminCreateStaffResult;
};

export type AdminReissueStaffInvitationResult = {
  invitation: AdminStaffInvitation;
  activation_token: string;
};

export type AdminReissueStaffInvitationResponse = {
  data: AdminReissueStaffInvitationResult;
};

export type AdminAccountBan = {
  id: string;
  target_type: string;
  staff_account_id?: string;
  customer_id?: string;
  scope: string;
  ban_type: string;
  reason: string;
  previous_account_status?: string;
  starts_at: string;
  expires_at?: string;
  issued_by_staff_id: string;
  revoked_at?: string;
  revoked_by_staff_id?: string;
  revocation_reason?: string;
  created_at: string;
  updated_at: string;
  active: boolean;
};

export type AdminStaffBansResponse = {
  data: AdminAccountBan[];
};

export type AdminStaffBanResponse = {
  data: AdminAccountBan;
};

export type AdminPermission = {
  id: string;
  code: string;
  description?: string;
};

export type AdminRoleListItem = {
  id: string;
  code: string;
  name: string;
  description?: string;
  is_system_role: boolean;
  permission_count: number;
  assigned_staff: number;
  created_at: string;
  updated_at: string;
};

export type AdminRoleDetail = AdminRoleListItem & {
  permissions: AdminPermission[];
};

export type AdminRolesResponse = {
  data: AdminRoleListItem[];
};

export type AdminRoleResponse = {
  data: AdminRoleDetail;
};

export type AdminPermissionsResponse = {
  data: AdminPermission[];
};

export type AdminStaffActivationPasswordResult = {
  staff_account_id: string;
  email: string;
  password_set_at: string;
  next: string;
};

export type AdminStaffActivationPasswordResponse = {
  data: AdminStaffActivationPasswordResult;
};

export type AdminStaffActivationMfaEnrollment = {
  credential_id: string;
  label: string;
  secret: string;
  enrollment_uri: string;
  algorithm: string;
  digits: number;
  period_seconds: number;
};

export type AdminStaffActivationMfaEnrollResponse = {
  data: {
    staff_account_id: string;
    email: string;
    enrollment: AdminStaffActivationMfaEnrollment;
  };
};

export type AdminStaffActivationCompleteResponse = {
  data: {
    activated: boolean;
    staff_account_id: string;
    email: string;
    activated_at: string;
    recovery_codes: string[];
    next: string;
  };
};
