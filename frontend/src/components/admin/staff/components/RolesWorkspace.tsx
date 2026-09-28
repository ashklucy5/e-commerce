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
  AdminPermission,
  AdminRoleDetail,
  AdminRoleListItem,
  AdminRoleResponse,
  AdminStaffListItem,
  AdminStaffListResponse,
} from "@/lib/admin/staff-types";

import styles from "../css/AdminStaff.module.css";

type Props = {
  roles: AdminRoleListItem[];
  permissions: AdminPermission[];
  canManage: boolean;
  canReadStaff: boolean;
  canCreateStaff: boolean;
  staffRefreshKey: number;
  onRolesChanged: () => Promise<void> | void;
  onCreateStaff: () => void;
  onOpenStaff: (staffId: string) => void;
  onBrowseStaffForRole: (roleCode: string) => void;
};

function errorMessage(value: unknown): string {
  if (value instanceof AdminRequestError) return value.message;
  if (value instanceof Error) return value.message;
  return "Unable to complete the role operation.";
}

function permissionGroup(code: string): string {
  const parts = code.split(".");
  return parts.length >= 3 ? parts[1] : "general";
}

function titleCase(value: string): string {
  return value
    .replace(/[_-]/g, " ")
    .replace(/\b\w/g, (character) => character.toUpperCase());
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

export default function RolesWorkspace({
  roles,
  permissions,
  canManage,
  canReadStaff,
  canCreateStaff,
  staffRefreshKey,
  onRolesChanged,
  onCreateStaff,
  onOpenStaff,
  onBrowseStaffForRole,
}: Props) {
  const [selectedId, setSelectedId] = useState<string | null>(roles[0]?.id ?? null);
  const [detail, setDetail] = useState<AdminRoleDetail | null>(null);
  const [loading, setLoading] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [creating, setCreating] = useState(false);
  const [editing, setEditing] = useState(false);
  const [code, setCode] = useState("");
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [permissionCodes, setPermissionCodes] = useState<string[]>([]);
  const [permissionQuery, setPermissionQuery] = useState("");
  const [assignedStaff, setAssignedStaff] = useState<AdminStaffListItem[]>([]);
  const [assignedStaffTotal, setAssignedStaffTotal] = useState(0);
  const [assignedStaffLoading, setAssignedStaffLoading] = useState(false);
  const [assignedStaffError, setAssignedStaffError] = useState("");

  useEffect(() => {
    if (creating) return;
    if (!selectedId && roles.length > 0) setSelectedId(roles[0].id);
  }, [creating, roles, selectedId]);

  useEffect(() => {
    if (creating || !selectedId) {
      if (creating) setDetail(null);
      return;
    }

    let cancelled = false;
    setLoading(true);
    setError("");

    void adminFetch<AdminRoleResponse>(`/roles/${selectedId}`)
      .then((response) => {
        if (cancelled) return;
        setDetail(response.data);
        setName(response.data.name);
        setDescription(response.data.description ?? "");
        setPermissionCodes(response.data.permissions.map((permission) => permission.code));
        setEditing(false);
      })
      .catch((value: unknown) => {
        if (!cancelled) setError(errorMessage(value));
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });

    return () => {
      cancelled = true;
    };
  }, [creating, selectedId]);

  useEffect(() => {
    if (!canReadStaff || !detail || creating || editing) {
      setAssignedStaff([]);
      setAssignedStaffTotal(0);
      setAssignedStaffError("");
      return;
    }

    let cancelled = false;
    setAssignedStaffLoading(true);
    setAssignedStaffError("");

    const params = new URLSearchParams({
      page: "1",
      limit: "8",
      role: detail.code,
    });

    void adminFetch<AdminStaffListResponse>(`/staff?${params.toString()}`)
      .then((response) => {
        if (cancelled) return;
        setAssignedStaff(response.data ?? []);
        setAssignedStaffTotal(response.meta.total);
      })
      .catch((value: unknown) => {
        if (cancelled) return;
        setAssignedStaff([]);
        setAssignedStaffTotal(0);
        setAssignedStaffError(errorMessage(value));
      })
      .finally(() => {
        if (!cancelled) setAssignedStaffLoading(false);
      });

    return () => {
      cancelled = true;
    };
  }, [canReadStaff, creating, detail, editing, staffRefreshKey]);

  const permissionGroups = useMemo(() => {
    const query = permissionQuery.trim().toLowerCase();
    const grouped = new Map<string, AdminPermission[]>();

    for (const permission of permissions) {
      if (
        query &&
        !permission.code.toLowerCase().includes(query) &&
        !(permission.description ?? "").toLowerCase().includes(query)
      ) {
        continue;
      }

      const group = permissionGroup(permission.code);
      const current = grouped.get(group) ?? [];
      current.push(permission);
      grouped.set(group, current);
    }

    return Array.from(grouped.entries()).sort(([left], [right]) => left.localeCompare(right));
  }, [permissionQuery, permissions]);

  function beginCreate() {
    setCreating(true);
    setEditing(true);
    setSelectedId(null);
    setDetail(null);
    setCode("");
    setName("");
    setDescription("");
    setPermissionCodes([]);
    setPermissionQuery("");
    setError("");
  }

  function cancelEditor() {
    if (creating) {
      setCreating(false);
      setSelectedId(roles[0]?.id ?? null);
      return;
    }

    if (detail) {
      setName(detail.name);
      setDescription(detail.description ?? "");
      setPermissionCodes(detail.permissions.map((permission) => permission.code));
    }
    setEditing(false);
  }

  function togglePermission(permissionCode: string) {
    setPermissionCodes((current) =>
      current.includes(permissionCode)
        ? current.filter((value) => value !== permissionCode)
        : [...current, permissionCode],
    );
  }

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!canManage || busy) return;

    setBusy(true);
    setError("");

    try {
      if (creating) {
        const response = await adminFetch<AdminRoleResponse>("/roles", {
          method: "POST",
          body: JSON.stringify({
            code: code.trim().toLowerCase(),
            name: name.trim(),
            description: description.trim(),
            permission_codes: permissionCodes,
          }),
        });

        setCreating(false);
        setSelectedId(response.data.id);
        setDetail(response.data);
        setEditing(false);
        await onRolesChanged();
        return;
      }

      if (!detail || detail.is_system_role) return;

      await adminFetch<AdminRoleResponse>(`/roles/${detail.id}`, {
        method: "PATCH",
        body: JSON.stringify({
          name: name.trim(),
          description: description.trim(),
        }),
      });

      const permissionsResponse = await adminFetch<AdminRoleResponse>(
        `/roles/${detail.id}/permissions`,
        {
          method: "PUT",
          body: JSON.stringify({ permission_codes: permissionCodes }),
        },
      );

      setDetail(permissionsResponse.data);
      setEditing(false);
      await onRolesChanged();
    } catch (value: unknown) {
      setError(errorMessage(value));
    } finally {
      setBusy(false);
    }
  }

  const editorOpen = creating || editing;
  const editorLocked = detail?.is_system_role ?? false;

  return (
    <div className={styles.rolesWorkspace}>
      <section className={styles.rolesRail}>
        <div className={styles.rolesRailHeader}>
          <div>
            <span className={styles.metaLabel}>Authorization</span>
            <h2>Roles</h2>
          </div>
          {canManage ? (
            <button type="button" className={styles.smallPrimaryButton} onClick={beginCreate}>
              + New role
            </button>
          ) : null}
        </div>

        <div className={styles.rolesList}>
          {roles.map((role) => (
            <button
              key={role.id}
              type="button"
              className={`${styles.roleListItem} ${
                selectedId === role.id && !creating ? styles.roleListItemActive : ""
              }`}
              onClick={() => {
                setCreating(false);
                setSelectedId(role.id);
                setError("");
              }}
            >
              <span className={styles.roleListIcon}>{role.is_system_role ? "◆" : "◇"}</span>
              <span className={styles.roleListCopy}>
                <strong>{role.name}</strong>
                <small>{role.code}</small>
              </span>
              <span className={styles.roleCount}>{role.assigned_staff}</span>
            </button>
          ))}
        </div>
      </section>

      <section className={styles.roleDetailPanel}>
        {error ? (
          <div className={styles.errorBanner} role="alert">
            {error}
          </div>
        ) : null}

        {creating ? (
          <RoleEditor
            title="Create custom role"
            code={code}
            setCode={setCode}
            name={name}
            setName={setName}
            description={description}
            setDescription={setDescription}
            permissionCodes={permissionCodes}
            togglePermission={togglePermission}
            permissionGroups={permissionGroups}
            permissionQuery={permissionQuery}
            setPermissionQuery={setPermissionQuery}
            busy={busy}
            creating
            onCancel={cancelEditor}
            onSubmit={submit}
          />
        ) : loading ? (
          <div className={styles.roleEmpty}>Loading role…</div>
        ) : detail ? (
          editorOpen && !editorLocked ? (
            <RoleEditor
              title={`Edit ${detail.name}`}
              code={detail.code}
              setCode={() => undefined}
              name={name}
              setName={setName}
              description={description}
              setDescription={setDescription}
              permissionCodes={permissionCodes}
              togglePermission={togglePermission}
              permissionGroups={permissionGroups}
              permissionQuery={permissionQuery}
              setPermissionQuery={setPermissionQuery}
              busy={busy}
              creating={false}
              onCancel={cancelEditor}
              onSubmit={submit}
            />
          ) : (
            <div className={styles.roleReadView}>
              <header className={styles.roleReadHeader}>
                <div>
                  <div className={styles.roleTitleLine}>
                    <h2>{detail.name}</h2>
                    {detail.is_system_role ? (
                      <span className={styles.systemBadge}>System role</span>
                    ) : null}
                  </div>
                  <code>{detail.code}</code>
                  <p>{detail.description || "No description."}</p>
                </div>
                {canManage && !detail.is_system_role ? (
                  <button
                    type="button"
                    className={styles.secondaryButton}
                    onClick={() => setEditing(true)}
                  >
                    Edit role
                  </button>
                ) : null}
              </header>

              <div className={styles.roleStats}>
                <div>
                  <span>Assigned staff</span>
                  <strong>{detail.assigned_staff}</strong>
                </div>
                <div>
                  <span>Permissions</span>
                  <strong>{detail.permission_count}</strong>
                </div>
                <div>
                  <span>Type</span>
                  <strong>{detail.is_system_role ? "Managed by system" : "Custom"}</strong>
                </div>
              </div>

              {canReadStaff ? (
                <section className={styles.roleStaffSection}>
                  <div className={styles.roleSectionHeader}>
                    <div>
                      <span className={styles.metaLabel}>Account control</span>
                      <h3>Staff assigned to this role</h3>
                      <p>
                        Open a staff account to change role assignments, lifecycle status,
                        MFA/security controls, invitations or access restrictions.
                      </p>
                    </div>
                    <div className={styles.roleSectionActions}>
                      <button
                        type="button"
                        className={styles.secondaryButton}
                        onClick={() => onBrowseStaffForRole(detail.code)}
                      >
                        View all staff
                      </button>
                      {canCreateStaff ? (
                        <button
                          type="button"
                          className={styles.smallPrimaryButton}
                          onClick={onCreateStaff}
                        >
                          + Add staff
                        </button>
                      ) : null}
                    </div>
                  </div>

                  {assignedStaffError ? (
                    <div className={styles.inlineError}>{assignedStaffError}</div>
                  ) : assignedStaffLoading ? (
                    <div className={styles.roleStaffLoading}>Loading assigned staff…</div>
                  ) : assignedStaff.length > 0 ? (
                    <div className={styles.roleStaffList}>
                      {assignedStaff.map((staff) => (
                        <button
                          key={staff.id}
                          type="button"
                          className={styles.roleStaffItem}
                          onClick={() => onOpenStaff(staff.id)}
                        >
                          <span className={styles.smallAvatar}>{initials(staff.full_name)}</span>
                          <span className={styles.roleStaffIdentity}>
                            <strong>{staff.full_name}</strong>
                            <small>{staff.staff_code} · {staff.email}</small>
                          </span>
                          <span className={styles.roleStaffSecurity}>
                            <span className={styles.statusBadge} data-tone={staff.status}>
                              {titleCase(staff.status)}
                            </span>
                            <small>
                              {staff.admin_mfa_status === "active"
                                ? "MFA active"
                                : titleCase(staff.admin_mfa_status)}
                            </small>
                          </span>
                          <span className={styles.roleStaffOpen}>Manage ›</span>
                        </button>
                      ))}
                    </div>
                  ) : (
                    <div className={styles.roleStaffEmpty}>
                      <strong>No staff are assigned to this role.</strong>
                      <p>
                        Create a staff account or open Staff accounts to assign this role to an
                        existing member.
                      </p>
                      {canCreateStaff ? (
                        <button
                          type="button"
                          className={styles.secondaryButton}
                          onClick={onCreateStaff}
                        >
                          Add staff account
                        </button>
                      ) : null}
                    </div>
                  )}

                  {assignedStaffTotal > assignedStaff.length ? (
                    <button
                      type="button"
                      className={styles.roleStaffMore}
                      onClick={() => onBrowseStaffForRole(detail.code)}
                    >
                      View all {assignedStaffTotal} assigned staff
                    </button>
                  ) : null}
                </section>
              ) : null}

              <section className={styles.rolePermissionsSection}>
                <div className={styles.roleSectionHeader}>
                  <div>
                    <span className={styles.metaLabel}>Authorization bundle</span>
                    <h3>Permissions</h3>
                    <p>Effective permission codes granted by this role.</p>
                  </div>
                </div>

                <div className={styles.permissionReadList}>
                  {detail.permissions.map((permission) => (
                    <div className={styles.permissionReadItem} key={permission.id}>
                      <code>{permission.code}</code>
                      <p>{permission.description || "No description."}</p>
                    </div>
                  ))}
                </div>
              </section>
            </div>
          )
        ) : (
          <div className={styles.roleEmpty}>Select a role to inspect its permissions.</div>
        )}
      </section>
    </div>
  );
}

type RoleEditorProps = {
  title: string;
  code: string;
  setCode: (value: string) => void;
  name: string;
  setName: (value: string) => void;
  description: string;
  setDescription: (value: string) => void;
  permissionCodes: string[];
  togglePermission: (code: string) => void;
  permissionGroups: Array<[string, AdminPermission[]]>;
  permissionQuery: string;
  setPermissionQuery: (value: string) => void;
  busy: boolean;
  creating: boolean;
  onCancel: () => void;
  onSubmit: (event: FormEvent<HTMLFormElement>) => Promise<void>;
};

function RoleEditor({
  title,
  code,
  setCode,
  name,
  setName,
  description,
  setDescription,
  permissionCodes,
  togglePermission,
  permissionGroups,
  permissionQuery,
  setPermissionQuery,
  busy,
  creating,
  onCancel,
  onSubmit,
}: RoleEditorProps) {
  return (
    <form className={styles.roleEditor} onSubmit={onSubmit}>
      <div className={styles.roleEditorHeader}>
        <div>
          <span className={styles.metaLabel}>
            {creating ? "Custom authorization" : "Role definition"}
          </span>
          <h2>{title}</h2>
        </div>
      </div>

      <div className={styles.formGrid}>
        <label className={styles.field}>
          <span>Role code</span>
          <input
            value={code}
            onChange={(event) => setCode(event.target.value)}
            placeholder="ops_custom_role"
            maxLength={80}
            readOnly={!creating}
            required
          />
        </label>
        <label className={styles.field}>
          <span>Name</span>
          <input
            value={name}
            onChange={(event) => setName(event.target.value)}
            maxLength={120}
            required
          />
        </label>
        <label className={`${styles.field} ${styles.fieldFull}`}>
          <span>Description</span>
          <textarea
            value={description}
            onChange={(event) => setDescription(event.target.value)}
            maxLength={500}
          />
        </label>
      </div>

      <div className={styles.permissionPickerHeader}>
        <div>
          <span className={styles.metaLabel}>Permission bundle</span>
          <h3>{permissionCodes.length} permissions selected</h3>
        </div>
        <input
          className={styles.permissionSearch}
          value={permissionQuery}
          onChange={(event) => setPermissionQuery(event.target.value)}
          placeholder="Filter permissions…"
        />
      </div>

      <div className={styles.permissionGroups}>
        {permissionGroups.map(([group, items]) => (
          <section className={styles.permissionGroup} key={group}>
            <h4>{titleCase(group)}</h4>
            <div className={styles.permissionChoiceList}>
              {items.map((permission) => {
                const selected = permissionCodes.includes(permission.code);
                return (
                  <label className={styles.permissionChoice} key={permission.id}>
                    <input
                      type="checkbox"
                      checked={selected}
                      onChange={() => togglePermission(permission.code)}
                    />
                    <span className={styles.checkVisual}>{selected ? "✓" : ""}</span>
                    <span>
                      <code>{permission.code}</code>
                      <small>{permission.description || "No description."}</small>
                    </span>
                  </label>
                );
              })}
            </div>
          </section>
        ))}
      </div>

      <div className={styles.editorFooter}>
        <button
          type="button"
          className={styles.secondaryButton}
          onClick={onCancel}
          disabled={busy}
        >
          Cancel
        </button>
        <button
          type="submit"
          className={styles.primaryButton}
          disabled={busy || !name.trim() || (creating && !code.trim())}
        >
          {busy ? "Saving…" : creating ? "Create role" : "Save role"}
        </button>
      </div>
    </form>
  );
}
