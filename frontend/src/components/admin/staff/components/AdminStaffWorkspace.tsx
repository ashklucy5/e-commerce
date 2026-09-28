"use client";

import {
  type FormEvent,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";

import { useAdminSession } from "@/components/admin/layout/context/AdminSessionContext";
import {
  AdminRequestError,
  adminFetch,
} from "@/lib/admin/api";
import type {
  AdminCreateStaffResult,
  AdminPermission,
  AdminPermissionsResponse,
  AdminRoleListItem,
  AdminRolesResponse,
  AdminStaffDetail,
  AdminStaffDetailResponse,
  AdminStaffListItem,
  AdminStaffListResponse,
  AdminStaffStatus,
} from "@/lib/admin/staff-types";

import RolesWorkspace from "./RolesWorkspace";
import StaffCreateSheet from "./StaffCreateSheet";
import StaffDetailSheet from "./StaffDetailSheet";

import styles from "../css/AdminStaff.module.css";

type Props = {
  portal: string;
};

type Tab = "roles" | "staff";

type Summary = {
  total: number;
  pending: number;
  active: number;
  restricted: number;
};

const PAGE_SIZE = 30;

const STATUS_FILTERS: Array<{ value: "" | AdminStaffStatus; label: string }> = [
  { value: "", label: "All staff" },
  { value: "pending_activation", label: "Pending activation" },
  { value: "active", label: "Active" },
  { value: "suspended", label: "Suspended" },
  { value: "disabled", label: "Disabled" },
  { value: "banned", label: "Banned" },
  { value: "deleted", label: "Deleted" },
];

function errorMessage(value: unknown): string {
  if (value instanceof AdminRequestError) return value.message;
  if (value instanceof Error) return value.message;
  return "Unable to complete the staff request.";
}

function formatDate(value?: string): string {
  if (!value) return "Never";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "Never";
  return new Intl.DateTimeFormat("en", {
    month: "short",
    day: "numeric",
    year: "numeric",
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

export default function AdminStaffWorkspace({ portal }: Props) {
  const principal = useAdminSession();
  const isSuperAdmin = principal.staff.roles.includes("admin_superuser");
  const permissions = principal.staff.permissions;

  const hasPermission = useCallback(
    (permission: string) => isSuperAdmin || permissions.includes(permission),
    [isSuperAdmin, permissions],
  );

  const canReadStaff = hasPermission("admin.staff.read");
  const canManageStaff = hasPermission("admin.staff.manage");
  const canAssignRoles = hasPermission("admin.staff.role.assign");
  const canSecurityManage = hasPermission("admin.staff.security.manage");
  const canDeleteStaff = hasPermission("admin.staff.delete");
  const canBanStaff = hasPermission("admin.staff.ban");
  const canReadRoles = hasPermission("admin.role.read");
  const canManageRoles = hasPermission("admin.role.manage");

  const [tab, setTab] = useState<Tab>(canReadRoles ? "roles" : "staff");
  const [items, setItems] = useState<AdminStaffListItem[]>([]);
  const [roles, setRoles] = useState<AdminRoleListItem[]>([]);
  const [rolePermissions, setRolePermissions] = useState<AdminPermission[]>([]);
  const [meta, setMeta] = useState<AdminStaffListResponse["meta"] | null>(null);
  const [summary, setSummary] = useState<Summary>({ total: 0, pending: 0, active: 0, restricted: 0 });
  const [status, setStatus] = useState<"" | AdminStaffStatus>("");
  const [roleFilter, setRoleFilter] = useState("");
  const [queryInput, setQueryInput] = useState("");
  const [query, setQuery] = useState("");
  const [page, setPage] = useState(1);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [detail, setDetail] = useState<AdminStaffDetail | null>(null);
  const [createOpen, setCreateOpen] = useState(false);
  const [loading, setLoading] = useState(true);
  const [detailLoading, setDetailLoading] = useState(false);
  const [error, setError] = useState("");
  const [detailError, setDetailError] = useState("");
  const [staffRefreshKey, setStaffRefreshKey] = useState(0);

  const listAbortRef = useRef<AbortController | null>(null);

  const loadRoles = useCallback(async () => {
    if (!canReadRoles) return;
    try {
      const requests: [Promise<AdminRolesResponse>, Promise<AdminPermissionsResponse> | null] = [
        adminFetch<AdminRolesResponse>("/roles"),
        canManageRoles ? adminFetch<AdminPermissionsResponse>("/permissions") : null,
      ];

      const roleResponse = await requests[0];
      setRoles(roleResponse.data ?? []);

      if (requests[1]) {
        const permissionResponse = await requests[1];
        setRolePermissions(permissionResponse.data ?? []);
      }
    } catch (value: unknown) {
      setError(errorMessage(value));
    }
  }, [canManageRoles, canReadRoles]);

  const loadSummary = useCallback(async () => {
    if (!canReadStaff) return;
    try {
      const [all, pending, active, banned, disabled, suspended] = await Promise.all([
        adminFetch<AdminStaffListResponse>("/staff?page=1&limit=1"),
        adminFetch<AdminStaffListResponse>("/staff?page=1&limit=1&status=pending_activation"),
        adminFetch<AdminStaffListResponse>("/staff?page=1&limit=1&status=active"),
        adminFetch<AdminStaffListResponse>("/staff?page=1&limit=1&status=banned"),
        adminFetch<AdminStaffListResponse>("/staff?page=1&limit=1&status=disabled"),
        adminFetch<AdminStaffListResponse>("/staff?page=1&limit=1&status=suspended"),
      ]);

      setSummary({
        total: all.meta.total,
        pending: pending.meta.total,
        active: active.meta.total,
        restricted: banned.meta.total + disabled.meta.total + suspended.meta.total,
      });
    } catch {
      // The directory stays usable when summary counts are unavailable.
    }
  }, [canReadStaff]);

  const loadStaff = useCallback(async () => {
    if (!canReadStaff) {
      setLoading(false);
      return;
    }

    listAbortRef.current?.abort();
    const controller = new AbortController();
    listAbortRef.current = controller;
    setLoading(true);
    setError("");

    const params = new URLSearchParams({ page: String(page), limit: String(PAGE_SIZE) });
    if (status) params.set("status", status);
    if (roleFilter) params.set("role", roleFilter);
    if (query.trim()) params.set("q", query.trim());

    try {
      const response = await adminFetch<AdminStaffListResponse>(
        `/staff?${params.toString()}`,
        { signal: controller.signal },
      );
      if (controller.signal.aborted) return;
      setItems(response.data ?? []);
      setMeta(response.meta);
    } catch (value: unknown) {
      if (controller.signal.aborted) return;
      setItems([]);
      setMeta(null);
      setError(errorMessage(value));
    } finally {
      if (!controller.signal.aborted) setLoading(false);
    }
  }, [canReadStaff, page, query, roleFilter, status]);

  const loadDetail = useCallback(async (staffId: string) => {
    if (!canReadStaff) return;
    setDetailLoading(true);
    setDetailError("");
    try {
      const response = await adminFetch<AdminStaffDetailResponse>(`/staff/${staffId}`);
      setDetail(response.data);
    } catch (value: unknown) {
      setDetail(null);
      setDetailError(errorMessage(value));
    } finally {
      setDetailLoading(false);
    }
  }, [canReadStaff]);

  useEffect(() => {
    void loadRoles();
  }, [loadRoles]);

  useEffect(() => {
    void loadSummary();
  }, [loadSummary]);

  useEffect(() => {
    void loadStaff();
    return () => listAbortRef.current?.abort();
  }, [loadStaff]);

  useEffect(() => {
    if (!selectedId) {
      setDetail(null);
      setDetailError("");
      return;
    }
    void loadDetail(selectedId);
  }, [loadDetail, selectedId]);

  useEffect(() => {
    const timer = window.setTimeout(() => {
      setQuery(queryInput.trim());
      setPage(1);
    }, 300);
    return () => window.clearTimeout(timer);
  }, [queryInput]);

  const roleNameByCode = useMemo(
    () => new Map(roles.map((role) => [role.code, role.name])),
    [roles],
  );

  async function refreshAfterChange(nextDetail?: AdminStaffDetail) {
    if (nextDetail) setDetail(nextDetail);
    await Promise.all([loadStaff(), loadSummary(), loadRoles()]);
    setStaffRefreshKey((value) => value + 1);
  }

  function handleCreated(result: AdminCreateStaffResult) {
    setSelectedId(result.staff.id);
    setDetail(result.staff);
    void Promise.all([loadStaff(), loadSummary(), loadRoles()]);
    setStaffRefreshKey((value) => value + 1);
  }

  function submitSearch(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setQuery(queryInput.trim());
    setPage(1);
  }

  if (!canReadStaff && !canReadRoles) {
    return (
      <main className={styles.page}>
        <div className={styles.permissionDenied}>
          <span>Restricted area</span>
          <h1>Staff & roles</h1>
          <p>Your current role does not include staff-directory or role visibility.</p>
        </div>
      </main>
    );
  }

  return (
    <main className={styles.page}>
      <header className={styles.pageHeader}>
        <div>
          <p className={styles.eyebrow}>Authorization</p>
          <h1>Access control</h1>
          <p>Control roles, permission bundles and the staff accounts that receive operational access.</p>
        </div>

        {canManageStaff && canAssignRoles && canReadRoles && canReadStaff ? (
          <button type="button" className={styles.primaryButton} onClick={() => setCreateOpen(true)}>
            <span aria-hidden="true">＋</span>
            Add staff
          </button>
        ) : null}
      </header>

      <div className={styles.tabBar} role="tablist" aria-label="Access control views">
        {canReadRoles ? (
          <button
            type="button"
            role="tab"
            aria-selected={tab === "roles"}
            className={tab === "roles" ? styles.tabActive : ""}
            onClick={() => setTab("roles")}
          >
            Role control
          </button>
        ) : null}
        {canReadStaff ? (
          <button
            type="button"
            role="tab"
            aria-selected={tab === "staff"}
            className={tab === "staff" ? styles.tabActive : ""}
            onClick={() => setTab("staff")}
          >
            Staff accounts
          </button>
        ) : null}
      </div>

      {tab === "roles" && canReadRoles ? (
        <RolesWorkspace
          roles={roles}
          permissions={rolePermissions}
          canManage={canManageRoles}
          canReadStaff={canReadStaff}
          canCreateStaff={canManageStaff && canAssignRoles && canReadRoles && canReadStaff}
          staffRefreshKey={staffRefreshKey}
          onRolesChanged={loadRoles}
          onCreateStaff={() => setCreateOpen(true)}
          onOpenStaff={(staffId) => setSelectedId(staffId)}
          onBrowseStaffForRole={(roleCode) => {
            setRoleFilter(roleCode);
            setStatus("");
            setQueryInput("");
            setQuery("");
            setPage(1);
            setTab("staff");
          }}
        />
      ) : (
        <>
          <section className={styles.summaryGrid} aria-label="Staff summary">
            <div className={styles.summaryCard} data-tone="neutral">
              <span>Total staff</span>
              <strong>{summary.total}</strong>
              <small>Historical identities included</small>
            </div>
            <div className={styles.summaryCard} data-tone="warning">
              <span>Pending activation</span>
              <strong>{summary.pending}</strong>
              <small>Waiting for password + MFA</small>
            </div>
            <div className={styles.summaryCard} data-tone="positive">
              <span>Active</span>
              <strong>{summary.active}</strong>
              <small>Operational access available</small>
            </div>
            <div className={styles.summaryCard} data-tone="danger">
              <span>Restricted</span>
              <strong>{summary.restricted}</strong>
              <small>Suspended, disabled or banned</small>
            </div>
          </section>

          <section className={styles.directoryCard}>
            <div className={styles.directoryToolbar}>
              <form className={styles.searchForm} onSubmit={submitSearch}>
                <span className={styles.searchIcon} aria-hidden="true">⌕</span>
                <input
                  value={queryInput}
                  onChange={(event) => setQueryInput(event.target.value)}
                  placeholder="Staff code, email, phone or UUID…"
                  aria-label="Search staff"
                />
                {queryInput ? (
                  <button type="button" onClick={() => { setQueryInput(""); setQuery(""); setPage(1); }} aria-label="Clear search">×</button>
                ) : null}
              </form>

              <div className={styles.filters}>
                <select
                  value={status}
                  onChange={(event) => { setStatus(event.target.value as "" | AdminStaffStatus); setPage(1); }}
                  aria-label="Filter by status"
                >
                  {STATUS_FILTERS.map((option) => <option value={option.value} key={option.value || "all"}>{option.label}</option>)}
                </select>

                <select
                  value={roleFilter}
                  onChange={(event) => { setRoleFilter(event.target.value); setPage(1); }}
                  aria-label="Filter by role"
                >
                  <option value="">All roles</option>
                  {roles.map((role) => <option value={role.code} key={role.id}>{role.name}</option>)}
                </select>

                <button type="button" className={styles.refreshButton} onClick={() => void Promise.all([loadStaff(), loadSummary()])} disabled={loading}>
                  ↻ <span>Refresh</span>
                </button>
              </div>
            </div>

            {error ? <div className={styles.errorBanner} role="alert">{error}</div> : null}

            <div className={styles.tableWrap}>
              <table className={styles.staffTable}>
                <thead>
                  <tr>
                    <th>Staff member</th>
                    <th>Role</th>
                    <th>Status</th>
                    <th>Security</th>
                    <th>Last login</th>
                    <th aria-label="Open" />
                  </tr>
                </thead>
                <tbody>
                  {loading ? (
                    Array.from({ length: 6 }).map((_, index) => (
                      <tr key={`skeleton-${index}`} className={styles.skeletonRow}>
                        <td colSpan={6}><span /></td>
                      </tr>
                    ))
                  ) : items.length === 0 ? (
                    <tr>
                      <td colSpan={6}>
                        <div className={styles.emptyState}>
                          <span>◎</span>
                          <strong>No staff found</strong>
                          <p>Adjust the search or filters to widen the directory.</p>
                        </div>
                      </td>
                    </tr>
                  ) : (
                    items.map((staff) => (
                      <tr key={staff.id} onClick={() => setSelectedId(staff.id)} className={styles.staffRow}>
                        <td>
                          <div className={styles.staffIdentity}>
                            <div className={styles.smallAvatar}>{initials(staff.full_name)}</div>
                            <div>
                              <strong>{staff.full_name}</strong>
                              <span>{staff.staff_code} · {staff.email}</span>
                            </div>
                          </div>
                        </td>
                        <td>
                          <div className={styles.tableRoleStack}>
                            {staff.roles.slice(0, 2).map((role) => (
                              <span key={role}>{roleNameByCode.get(role) ?? role}</span>
                            ))}
                            {staff.roles.length > 2 ? <small>+{staff.roles.length - 2} more</small> : null}
                          </div>
                        </td>
                        <td><span className={styles.statusBadge} data-tone={staff.status}>{titleCase(staff.status)}</span></td>
                        <td>
                          <div className={styles.securityCell}>
                            <strong>{staff.admin_mfa_status === "active" ? "MFA active" : titleCase(staff.admin_mfa_status)}</strong>
                            <span>{staff.active_admin_sessions} admin session{staff.active_admin_sessions === 1 ? "" : "s"}</span>
                          </div>
                        </td>
                        <td>{formatDate(staff.last_login_at)}</td>
                        <td><button type="button" className={styles.rowOpenButton} aria-label={`Open ${staff.full_name}`}>›</button></td>
                      </tr>
                    ))
                  )}
                </tbody>
              </table>
            </div>

            <div className={styles.mobileStaffList}>
              {!loading && items.map((staff) => (
                <button type="button" className={styles.mobileStaffCard} key={staff.id} onClick={() => setSelectedId(staff.id)}>
                  <div className={styles.mobileStaffTop}>
                    <div className={styles.staffIdentity}>
                      <div className={styles.smallAvatar}>{initials(staff.full_name)}</div>
                      <div>
                        <strong>{staff.full_name}</strong>
                        <span>{staff.staff_code}</span>
                      </div>
                    </div>
                    <span className={styles.statusBadge} data-tone={staff.status}>{titleCase(staff.status)}</span>
                  </div>
                  <div className={styles.mobileStaffMeta}>
                    <span>{staff.roles.map((role) => roleNameByCode.get(role) ?? role).join(" · ") || "No role"}</span>
                    <span>{staff.admin_mfa_status === "active" ? "MFA active" : titleCase(staff.admin_mfa_status)}</span>
                  </div>
                </button>
              ))}
            </div>

            {meta ? (
              <footer className={styles.pagination}>
                <span>{meta.total} staff · page {meta.page} of {Math.max(meta.total_pages, 1)}</span>
                <div>
                  <button type="button" onClick={() => setPage((value) => Math.max(1, value - 1))} disabled={!meta.has_previous || loading}>Previous</button>
                  <button type="button" onClick={() => setPage((value) => value + 1)} disabled={!meta.has_next || loading}>Next</button>
                </div>
              </footer>
            ) : null}
          </section>
        </>
      )}

      <StaffCreateSheet
        open={createOpen}
        portal={portal}
        roles={roles}
        canAssignProtected={isSuperAdmin}
        onClose={() => setCreateOpen(false)}
        onCreated={handleCreated}
      />

      {detailLoading && selectedId && !detail ? (
        <div className={styles.sheetBackdrop} role="presentation" onMouseDown={() => setSelectedId(null)}>
          <div className={styles.loadingSheet} onMouseDown={(event) => event.stopPropagation()}>Loading staff record…</div>
        </div>
      ) : null}

      {detailError && selectedId && !detail ? (
        <div className={styles.sheetBackdrop} role="presentation" onMouseDown={() => setSelectedId(null)}>
          <div className={styles.loadingSheet} onMouseDown={(event) => event.stopPropagation()}>
            <strong>Unable to open staff record</strong>
            <p>{detailError}</p>
            <button type="button" className={styles.secondaryButton} onClick={() => setSelectedId(null)}>Close</button>
          </div>
        </div>
      ) : null}

      <StaffDetailSheet
        portal={portal}
        staff={detail}
        roles={roles}
        currentStaffId={principal.staff.id}
        isSuperAdmin={isSuperAdmin}
        canManage={canManageStaff}
        canAssignRoles={canAssignRoles && canReadRoles}
        canSecurityManage={canSecurityManage}
        canDelete={canDeleteStaff}
        canBan={canBanStaff}
        onClose={() => { setSelectedId(null); setDetail(null); }}
        onChanged={refreshAfterChange}
      />
    </main>
  );
}
