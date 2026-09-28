import type {
  AdminMfaEnrollment,
  AdminPrincipal,
} from "./types";

export type AdminSecurityStepUpMethod =
  | "totp"
  | "recovery_code";

export type AdminSelfSecurityMfa = {
  enabled: boolean;
  label: string;
  verified_at: string | null;
  last_used_at: string | null;
  recovery_codes_remaining: number;
};

export type AdminSelfSecuritySession = {
  id: string;
  current: boolean;
  authenticated_at: string;
  mfa_verified_at: string;
  last_used_at: string | null;
  last_rotated_at: string | null;
  created_ip: string;
  last_ip: string;
  user_agent: string;
  refresh_expires_at: string;
};

export type AdminSelfSecurityOverview = {
  mfa: AdminSelfSecurityMfa;
  sessions: AdminSelfSecuritySession[];
};

export type AdminSelfSecurityOverviewResponse = {
  data: AdminSelfSecurityOverview;
};

export type AdminBeginMfaRotationResponse = {
  data: {
    rotation_token: string;
    rotation_expires_at: string;
    enrollment: AdminMfaEnrollment;
  };
};

export type AdminSelfSecurityMutationResponse = {
  data: {
    principal: AdminPrincipal;
    csrf_token: string;
    recovery_codes: string[];
  };
};

export type AdminChangePasswordResponse = {
  data: {
    principal: AdminPrincipal;
    csrf_token: string;
  };
};

export type AdminRevokeOtherSessionsResponse = {
  data: {
    revoked_sessions: number;
  };
};
