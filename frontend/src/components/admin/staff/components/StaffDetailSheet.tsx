"use client";

import {
  type FormEvent,
  useEffect,
  useMemo,
  useState,
} from "react";

import {
  AdminRequestError,
  adminFetch,
} from "@/lib/admin/api";
import type {
  AdminRoleListItem,
  AdminStaffBanResponse,
  AdminStaffBansResponse,
  AdminStaffDetail,
  AdminStaffInvitation,
  AdminStaffInvitationResponse,
  AdminReissueStaffInvitationResponse,
  AdminStaffDetailResponse,
} from "@/lib/admin/staff-types";

import styles from "../css/AdminStaff.module.css";

const PROTECTED_ROLE_CODES = new Set([
  "admin_superuser",
  "admin_administrator",
  "admin_security",
]);

type Props = {
  portal: string;
  staff: AdminStaffDetail | null;
  roles: AdminRoleListItem[];
  currentStaffId: string;
  isSuperAdmin: boolean;
  canManage: boolean;
  canAssignRoles: boolean;
  canSecurityManage: boolean;
  canDelete: boolean;
  canBan: boolean;
  onClose: () => void;
  onChanged: (detail?: AdminStaffDetail) => Promise<void> | void;
};

type ActionMode =
  | ""
  | "roles"
  | "ban"
  | "unban"
  | "password"
  | "delete";

function errorMessage(value: unknown): string {
  if (value instanceof AdminRequestError) return value.message;
  if (value instanceof Error) return value.message;
  return "Unable to complete the staff operation.";
}

function formatDate(value?: string): string {
  if (!value) return "—";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "—";
  return new Intl.DateTimeFormat("en", {
    month: "short",
    day: "numeric",
    year: "numeric",
    hour: "numeric",
    minute: "2-digit",
  }).format(date);
}

function initials(name: string): string {
  return (
    name
      .split(/\s+/)
      .filter(Boolean)
      .slice(0, 2)
      .map((part) => part[0])
      .join("")
      .toUpperCase() || "ST"
  );
}

function titleCase(value: string): string {
  return value.replace(/_/g, " ").replace(/\b\w/g, (character) => character.toUpperCase());
}

function activationUrl(portal: string, email: string, token: string): string {
  if (typeof window === "undefined") return "";
  const url = new URL(`/${portal}/activate`, window.location.origin);
  url.searchParams.set("email", email);
  url.hash = `token=${encodeURIComponent(token)}`;
  return url.toString();
}

export default function StaffDetailSheet({
  portal,
  staff,
  roles,
  currentStaffId,
  isSuperAdmin,
  canManage,
  canAssignRoles,
  canSecurityManage,
  canDelete,
  canBan,
  onClose,
  onChanged,
}: Props) {
  const [invitation, setInvitation] = useState<AdminStaffInvitation | null>(null);
  const [bans, setBans] = useState<AdminStaffBansResponse["data"]>([]);
  const [loadingMeta, setLoadingMeta] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [mode, setMode] = useState<ActionMode>("");
  const [roleCodes, setRoleCodes] = useState<string[]>([]);
  const [reason, setReason] = useState("");
  const [password, setPassword] = useState("");
  const [reissuedToken, setReissuedToken] = useState("");
  const [copied, setCopied] = useState(false);

  const isSelf = staff?.id === currentStaffId;
  const activeBan = useMemo(() => bans.find((ban) => ban.active) ?? null, [bans]);
  const targetProtected = useMemo(
    () => staff?.roles.some((role) => PROTECTED_ROLE_CODES.has(role)) ?? false,
    [staff],
  );

  useEffect(() => {
    setMode("");
    setReason("");
    setPassword("");
    setError("");
    setReissuedToken("");
    setInvitation(null);
    setBans([]);
  }, [staff?.id]);

  useEffect(() => {
    setRoleCodes(staff?.roles ?? []);
  }, [staff?.roles]);

  useEffect(() => {
    if (!staff) {
      setInvitation(null);
      setBans([]);
      return;
    }

    const selectedStaff = staff;
    let cancelled = false;

    async function load() {
      setLoadingMeta(true);
      try {
        const banRequest = adminFetch<AdminStaffBansResponse>(`/staff/${selectedStaff.id}/bans`);
        const inviteRequest = selectedStaff.status === "pending_activation"
          ? adminFetch<AdminStaffInvitationResponse>(`/staff/${selectedStaff.id}/invitation`)
          : Promise.resolve(null);

        const [banResponse, inviteResponse] = await Promise.all([banRequest, inviteRequest]);
        if (cancelled) return;
        setBans(banResponse.data ?? []);
        setInvitation(inviteResponse?.data ?? null);
      } catch (value: unknown) {
        if (!cancelled) setError(errorMessage(value));
      } finally {
        if (!cancelled) setLoadingMeta(false);
      }
    }

    void load();
    return () => {
      cancelled = true;
    };
  }, [staff?.id, staff?.status]);

  if (!staff) return null;

  const selectedStaff = staff;

  const canMutateTarget = !isSelf && selectedStaff.status !== "deleted" && (isSuperAdmin || !targetProtected);
  const canRoleEdit =
    canManage &&
    canAssignRoles &&
    canMutateTarget &&
    selectedStaff.status !== "banned" &&
    (isSuperAdmin || !targetProtected);

  const assignableRoles = roles.filter(
    (role) => isSuperAdmin || (role.is_system_role && !PROTECTED_ROLE_CODES.has(role.code)),
  );

  async function refreshDetail() {
    const response = await adminFetch<AdminStaffDetailResponse>(`/staff/${selectedStaff.id}`);
    await onChanged(response.data);
  }

  async function mutate(action: () => Promise<void>) {
    if (busy) return;
    setBusy(true);
    setError("");
    try {
      await action();
      await refreshDetail();
    } catch (value: unknown) {
      setError(errorMessage(value));
    } finally {
      setBusy(false);
    }
  }

  async function setStatus(status: "active" | "suspended" | "disabled") {
    await mutate(async () => {
      await adminFetch(`/staff/${selectedStaff.id}/status`, {
        method: "PATCH",
        body: JSON.stringify({ status }),
      });
      setMode("");
    });
  }

  async function saveRoles(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    await mutate(async () => {
      await adminFetch(`/staff/${selectedStaff.id}/roles`, {
        method: "PUT",
        body: JSON.stringify({ role_codes: roleCodes }),
      });
      setMode("");
    });
  }

  async function reissueInvitation() {
    await mutate(async () => {
      const response = await adminFetch<AdminReissueStaffInvitationResponse>(
        `/staff/${selectedStaff.id}/invitation/reissue`,
        { method: "POST" },
      );
      setInvitation(response.data.invitation);
      setReissuedToken(response.data.activation_token);
    });
  }

  async function cancelInvitation() {
    await mutate(async () => {
      const response = await adminFetch<AdminStaffInvitationResponse>(
        `/staff/${selectedStaff.id}/invitation/cancel`,
        { method: "POST" },
      );
      setInvitation(response.data);
      setReissuedToken("");
    });
  }

  async function createBan(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!reason.trim()) return;
    await mutate(async () => {
      const response = await adminFetch<AdminStaffBanResponse>(`/staff/${selectedStaff.id}/bans`, {
        method: "POST",
        body: JSON.stringify({ ban_type: "permanent", reason: reason.trim() }),
      });
      setBans((current) => [response.data, ...current]);
      setReason("");
      setMode("");
    });
  }

  async function revokeBan(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!activeBan || !reason.trim()) return;
    await mutate(async () => {
      const response = await adminFetch<AdminStaffBanResponse>(
        `/staff/${selectedStaff.id}/bans/${activeBan.id}/revoke`,
        {
          method: "POST",
          body: JSON.stringify({ reason: reason.trim() }),
        },
      );
      setBans((current) =>
        current.map((ban) => (ban.id === response.data.id ? response.data : ban)),
      );
      setReason("");
      setMode("");
    });
  }

  async function resetPassword(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!password) return;
    await mutate(async () => {
      await adminFetch(`/staff/${selectedStaff.id}/password`, {
        method: "PUT",
        body: JSON.stringify({ password }),
      });
      setPassword("");
      setMode("");
    });
  }

  async function resetMfa() {
    await mutate(async () => {
      await adminFetch(`/staff/${selectedStaff.id}/admin-mfa/reset`, { method: "POST" });
    });
  }

  async function deleteStaff() {
    await mutate(async () => {
      await adminFetch(`/staff/${selectedStaff.id}`, { method: "DELETE" });
      setMode("");
    });
  }

  function toggleRole(code: string) {
    setRoleCodes((current) =>
      current.includes(code)
        ? current.filter((value) => value !== code)
        : [...current, code],
    );
  }

  const invitationLink = reissuedToken
    ? activationUrl(portal, selectedStaff.email, reissuedToken)
    : "";

  return (
    <div className={styles.sheetBackdrop} role="presentation" onMouseDown={onClose}>
      <section
        className={styles.detailSheet}
        role="dialog"
        aria-modal="true"
        aria-labelledby="staff-detail-title"
        onMouseDown={(event) => event.stopPropagation()}
      >
        <div className={styles.sheetHandle} aria-hidden="true" />

        <header className={styles.detailHeader}>
          <div className={styles.personHeader}>
            <div className={styles.avatar}>{initials(selectedStaff.full_name)}</div>
            <div>
              <div className={styles.personTitleLine}>
                <h2 id="staff-detail-title">{selectedStaff.full_name}</h2>
                <span className={styles.statusBadge} data-tone={selectedStaff.status}>{titleCase(selectedStaff.status)}</span>
              </div>
              <p>{selectedStaff.staff_code} · {selectedStaff.email}</p>
            </div>
          </div>

          <button type="button" className={styles.iconButton} onClick={onClose} aria-label="Close">×</button>
        </header>

        <div className={styles.detailScroll}>
          {error ? <div className={styles.errorBanner} role="alert">{error}</div> : null}

          {reissuedToken ? (
            <section className={styles.activationCard}>
              <div className={styles.activationHeader}>
                <div>
                  <span className={styles.metaLabel}>Fresh activation link</span>
                  <strong>Shown once — deliver securely</strong>
                </div>
                <span className={styles.statusBadge} data-tone="warning">New token</span>
              </div>
              <div className={styles.copyField}>
                <code>{invitationLink}</code>
                <button
                  type="button"
                  onClick={() => {
                    void navigator.clipboard.writeText(invitationLink);
                    setCopied(true);
                    window.setTimeout(() => setCopied(false), 1600);
                  }}
                >
                  {copied ? "Copied" : "Copy link"}
                </button>
              </div>
            </section>
          ) : null}

          <section className={styles.detailSection}>
            <div className={styles.sectionHeading}>
              <div>
                <span className={styles.metaLabel}>Identity & access</span>
                <h3>Staff profile</h3>
              </div>
              {isSelf ? <span className={styles.selfBadge}>Your account</span> : null}
            </div>

            <div className={styles.factGrid}>
              <div><span>Email</span><strong>{selectedStaff.email}</strong></div>
              <div><span>Phone</span><strong>{selectedStaff.phone || "—"}</strong></div>
              <div><span>Admin MFA</span><strong>{titleCase(selectedStaff.admin_mfa_status)}</strong></div>
              <div><span>Panel access</span><strong>{selectedStaff.has_admin_panel_access ? "Enabled" : "No access"}</strong></div>
              <div><span>Admin sessions</span><strong>{selectedStaff.active_admin_sessions}</strong></div>
              <div><span>Staff sessions</span><strong>{selectedStaff.active_staff_sessions}</strong></div>
              <div><span>Last login</span><strong>{formatDate(selectedStaff.last_login_at)}</strong></div>
              <div><span>Created</span><strong>{formatDate(selectedStaff.created_at)}</strong></div>
            </div>

            {selectedStaff.support_actor ? (
              <div className={styles.inlineInfo}>
                <span>Support actor</span>
                <strong>{selectedStaff.support_actor.actor_code}</strong>
                <small>{titleCase(selectedStaff.support_actor.status)} · {titleCase(selectedStaff.support_actor.presence)}</small>
              </div>
            ) : null}
          </section>

          {selectedStaff.status === "pending_activation" ? (
            <section className={styles.detailSection}>
              <div className={styles.sectionHeading}>
                <div>
                  <span className={styles.metaLabel}>Onboarding</span>
                  <h3>Invitation</h3>
                </div>
                {loadingMeta ? <span>Loading…</span> : invitation ? (
                  <span className={styles.statusBadge} data-tone={invitation.status}>{titleCase(invitation.status)}</span>
                ) : null}
              </div>

              {invitation ? (
                <div className={styles.invitationPanel}>
                  <div className={styles.factGrid}>
                    <div><span>Delivery</span><strong>{titleCase(invitation.delivery_mode)}</strong></div>
                    <div><span>Expires</span><strong>{formatDate(invitation.expires_at)}</strong></div>
                    <div><span>Password</span><strong>{invitation.password_set_at ? "Set" : "Not set"}</strong></div>
                    <div><span>Created</span><strong>{formatDate(invitation.created_at)}</strong></div>
                  </div>

                  {canManage && canMutateTarget ? (
                    <div className={styles.inlineActions}>
                      <button type="button" className={styles.secondaryButton} onClick={() => void reissueInvitation()} disabled={busy}>
                        Reissue invitation
                      </button>
                      {invitation.status === "pending" ? (
                        <button type="button" className={styles.ghostDangerButton} onClick={() => void cancelInvitation()} disabled={busy}>
                          Cancel invitation
                        </button>
                      ) : null}
                    </div>
                  ) : null}
                </div>
              ) : (
                <p className={styles.muted}>Invitation state is unavailable.</p>
              )}
            </section>
          ) : null}

          <section className={styles.detailSection}>
            <div className={styles.sectionHeading}>
              <div>
                <span className={styles.metaLabel}>Authorization</span>
                <h3>Roles</h3>
              </div>
              {canRoleEdit ? (
                <button type="button" className={styles.textButton} onClick={() => setMode(mode === "roles" ? "" : "roles")}>
                  {mode === "roles" ? "Close" : "Edit roles"}
                </button>
              ) : null}
            </div>

            <div className={styles.chipRow}>
              {selectedStaff.roles.map((role) => <span className={styles.roleChip} key={role}>{role}</span>)}
            </div>

            {mode === "roles" ? (
              <form className={styles.actionPanel} onSubmit={saveRoles}>
                <div className={styles.roleChoiceGrid}>
                  {assignableRoles.map((role) => {
                    const selected = roleCodes.includes(role.code);
                    return (
                      <button
                        key={role.id}
                        type="button"
                        className={`${styles.roleChoice} ${selected ? styles.roleChoiceSelected : ""}`}
                        onClick={() => toggleRole(role.code)}
                        aria-pressed={selected}
                      >
                        <span className={styles.roleChoiceMark}>{selected ? "✓" : "+"}</span>
                        <span><strong>{role.name}</strong><small>{role.code}</small></span>
                      </button>
                    );
                  })}
                </div>
                <div className={styles.inlineActions}>
                  <button type="button" className={styles.secondaryButton} onClick={() => { setRoleCodes(selectedStaff.roles); setMode(""); }} disabled={busy}>Cancel</button>
                  <button type="submit" className={styles.primaryButton} disabled={busy}>{busy ? "Saving…" : "Save roles"}</button>
                </div>
              </form>
            ) : null}
          </section>

          {canManage && canMutateTarget && !["pending_activation", "deleted", "banned"].includes(selectedStaff.status) ? (
            <section className={styles.detailSection}>
              <div className={styles.sectionHeading}>
                <div>
                  <span className={styles.metaLabel}>Lifecycle</span>
                  <h3>Account status</h3>
                </div>
              </div>
              <div className={styles.segmentedActions}>
                {(["active", "suspended", "disabled"] as const).map((status) => (
                  <button
                    key={status}
                    type="button"
                    className={selectedStaff.status === status ? styles.segmentActive : ""}
                    onClick={() => void setStatus(status)}
                    disabled={busy || selectedStaff.status === status}
                  >
                    {titleCase(status)}
                  </button>
                ))}
              </div>
            </section>
          ) : null}

          {activeBan ? (
            <section className={`${styles.detailSection} ${styles.dangerSection}`}>
              <div className={styles.sectionHeading}>
                <div>
                  <span className={styles.metaLabel}>Security restriction</span>
                  <h3>Permanent staff ban</h3>
                </div>
                <span className={styles.statusBadge} data-tone="banned">Active</span>
              </div>
              <p className={styles.reasonText}>{activeBan.reason}</p>
              <small className={styles.muted}>Issued {formatDate(activeBan.starts_at)}</small>

              {canBan && !isSelf ? (
                mode === "unban" ? (
                  <form className={styles.actionPanel} onSubmit={revokeBan}>
                    <label className={styles.field}>
                      <span>Revocation reason</span>
                      <textarea value={reason} onChange={(event) => setReason(event.target.value)} maxLength={1000} required />
                    </label>
                    <div className={styles.inlineActions}>
                      <button type="button" className={styles.secondaryButton} onClick={() => { setMode(""); setReason(""); }}>Cancel</button>
                      <button type="submit" className={styles.primaryButton} disabled={busy || !reason.trim()}>Revoke ban</button>
                    </div>
                  </form>
                ) : (
                  <button type="button" className={styles.secondaryButton} onClick={() => setMode("unban")}>Revoke ban</button>
                )
              ) : null}
            </section>
          ) : canBan && canMutateTarget && selectedStaff.status !== "pending_activation" && selectedStaff.status !== "deleted" ? (
            <section className={`${styles.detailSection} ${styles.dangerSection}`}>
              <div className={styles.sectionHeading}>
                <div>
                  <span className={styles.metaLabel}>Security restriction</span>
                  <h3>Staff ban</h3>
                </div>
              </div>
              <p className={styles.muted}>Staff bans are permanent until explicitly revoked. Active sessions are revoked by the backend.</p>
              {mode === "ban" ? (
                <form className={styles.actionPanel} onSubmit={createBan}>
                  <label className={styles.field}>
                    <span>Reason</span>
                    <textarea value={reason} onChange={(event) => setReason(event.target.value)} maxLength={1000} required />
                  </label>
                  <div className={styles.inlineActions}>
                    <button type="button" className={styles.secondaryButton} onClick={() => { setMode(""); setReason(""); }}>Cancel</button>
                    <button type="submit" className={styles.dangerButton} disabled={busy || !reason.trim()}>Ban staff account</button>
                  </div>
                </form>
              ) : (
                <button type="button" className={styles.ghostDangerButton} onClick={() => setMode("ban")}>Ban staff account</button>
              )}
            </section>
          ) : null}

          {canSecurityManage && canMutateTarget && !["pending_activation", "deleted", "banned"].includes(selectedStaff.status) ? (
            <section className={styles.detailSection}>
              <div className={styles.sectionHeading}>
                <div>
                  <span className={styles.metaLabel}>Recovery</span>
                  <h3>Security recovery</h3>
                </div>
              </div>
              <div className={styles.securityActions}>
                <div>
                  <strong>Reset Admin MFA</strong>
                  <p>Remove the registered authenticator and recovery codes. The staff member will enroll MFA again at sign-in.</p>
                  <button type="button" className={styles.secondaryButton} onClick={() => void resetMfa()} disabled={busy}>Reset MFA</button>
                </div>
                <div>
                  <strong>Reset password</strong>
                  <p>Set a new password through the privileged staff-security endpoint.</p>
                  {mode === "password" ? (
                    <form className={styles.actionPanel} onSubmit={resetPassword}>
                      <label className={styles.field}>
                        <span>New password</span>
                        <input type="password" value={password} onChange={(event) => setPassword(event.target.value)} autoComplete="new-password" required />
                      </label>
                      <div className={styles.inlineActions}>
                        <button type="button" className={styles.secondaryButton} onClick={() => { setMode(""); setPassword(""); }}>Cancel</button>
                        <button type="submit" className={styles.primaryButton} disabled={busy || !password}>Reset password</button>
                      </div>
                    </form>
                  ) : (
                    <button type="button" className={styles.secondaryButton} onClick={() => setMode("password")}>Reset password</button>
                  )}
                </div>
              </div>
            </section>
          ) : null}

          {canDelete && canMutateTarget ? (
            <section className={`${styles.detailSection} ${styles.dangerSection}`}>
              <div className={styles.sectionHeading}>
                <div>
                  <span className={styles.metaLabel}>Permanent access removal</span>
                  <h3>Delete staff access</h3>
                </div>
              </div>
              <p className={styles.muted}>This preserves the historical staff identity for audit and business records, but permanently removes access.</p>
              {mode === "delete" ? (
                <div className={styles.confirmPanel}>
                  <strong>Delete access for {selectedStaff.full_name}?</strong>
                  <p>This cannot be reversed through ordinary staff lifecycle controls.</p>
                  <div className={styles.inlineActions}>
                    <button type="button" className={styles.secondaryButton} onClick={() => setMode("")}>Cancel</button>
                    <button type="button" className={styles.dangerButton} onClick={() => void deleteStaff()} disabled={busy}>Delete staff access</button>
                  </div>
                </div>
              ) : (
                <button type="button" className={styles.ghostDangerButton} onClick={() => setMode("delete")}>Delete staff access</button>
              )}
            </section>
          ) : null}

          {bans.length > 0 ? (
            <section className={styles.detailSection}>
              <div className={styles.sectionHeading}>
                <div>
                  <span className={styles.metaLabel}>History</span>
                  <h3>Ban history</h3>
                </div>
                <span>{bans.length}</span>
              </div>
              <div className={styles.historyList}>
                {bans.map((ban) => (
                  <div className={styles.historyItem} key={ban.id}>
                    <div>
                      <strong>{ban.active ? "Active permanent ban" : "Revoked ban"}</strong>
                      <span>{ban.reason}</span>
                    </div>
                    <small>{formatDate(ban.starts_at)}</small>
                  </div>
                ))}
              </div>
            </section>
          ) : null}
        </div>
      </section>
    </div>
  );
}
