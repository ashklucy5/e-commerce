"use client";

import {
  type FormEvent,
  useMemo,
  useState,
} from "react";

import {
  AdminRequestError,
  adminFetch,
} from "@/lib/admin/api";
import type {
  AdminCreateStaffResponse,
  AdminCreateStaffResult,
  AdminRoleListItem,
} from "@/lib/admin/staff-types";

import styles from "../css/AdminStaff.module.css";

const PROTECTED_ROLE_CODES = new Set([
  "admin_superuser",
  "admin_administrator",
  "admin_security",
]);

type Props = {
  open: boolean;
  portal: string;
  roles: AdminRoleListItem[];
  canAssignProtected: boolean;
  onClose: () => void;
  onCreated: (result: AdminCreateStaffResult) => void;
};

function errorMessage(value: unknown): string {
  if (value instanceof AdminRequestError) return value.message;
  if (value instanceof Error) return value.message;
  return "Unable to create the staff account.";
}

function activationUrl(
  portal: string,
  email: string,
  token: string,
): string {
  if (typeof window === "undefined") return "";

  const url = new URL(`/${portal}/activate`, window.location.origin);
  url.searchParams.set("email", email);
  url.hash = `token=${encodeURIComponent(token)}`;
  return url.toString();
}

export default function StaffCreateSheet({
  open,
  portal,
  roles,
  canAssignProtected,
  onClose,
  onCreated,
}: Props) {
  const [fullName, setFullName] = useState("");
  const [email, setEmail] = useState("");
  const [phone, setPhone] = useState("");
  const [roleCodes, setRoleCodes] = useState<string[]>([]);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [created, setCreated] = useState<AdminCreateStaffResult | null>(null);
  const [copied, setCopied] = useState<"" | "link" | "token">("");

  const assignableRoles = useMemo(
    () =>
      roles.filter(
        (role) =>
          canAssignProtected || (role.is_system_role && !PROTECTED_ROLE_CODES.has(role.code)),
      ),
    [canAssignProtected, roles],
  );

  if (!open) return null;

  const inviteUrl = created
    ? activationUrl(
        portal,
        created.staff.email,
        created.invitation.activation_token,
      )
    : "";

  function close() {
    if (busy) return;
    setFullName("");
    setEmail("");
    setPhone("");
    setRoleCodes([]);
    setCreated(null);
    setError("");
    setCopied("");
    onClose();
  }

  function toggleRole(code: string) {
    setRoleCodes((current) =>
      current.includes(code)
        ? current.filter((value) => value !== code)
        : [...current, code],
    );
  }

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();

    if (busy || roleCodes.length === 0) return;

    setBusy(true);
    setError("");

    try {
      const response = await adminFetch<AdminCreateStaffResponse>(
        "/staff",
        {
          method: "POST",
          body: JSON.stringify({
            full_name: fullName.trim(),
            email: email.trim(),
            phone: phone.trim(),
            delivery_mode: "manual",
            role_codes: roleCodes,
          }),
        },
      );

      setCreated(response.data);
      onCreated(response.data);
    } catch (value: unknown) {
      setError(errorMessage(value));
    } finally {
      setBusy(false);
    }
  }

  async function copy(value: string, kind: "link" | "token") {
    if (!value) return;
    await navigator.clipboard.writeText(value);
    setCopied(kind);
    window.setTimeout(() => setCopied(""), 1600);
  }

  return (
    <div className={styles.sheetBackdrop} role="presentation" onMouseDown={close}>
      <section
        className={styles.createSheet}
        role="dialog"
        aria-modal="true"
        aria-labelledby="create-staff-title"
        onMouseDown={(event) => event.stopPropagation()}
      >
        <div className={styles.sheetHandle} aria-hidden="true" />

        <header className={styles.sheetHeader}>
          <div>
            <p className={styles.eyebrow}>Staff onboarding</p>
            <h2 id="create-staff-title">
              {created ? "Activation package" : "Add staff member"}
            </h2>
            <p>
              {created
                ? "This activation token is shown once. Deliver it securely to the staff member."
                : "Create a pending account, assign its starting role, then deliver the one-time activation link manually."}
            </p>
          </div>

          <button type="button" className={styles.iconButton} onClick={close} aria-label="Close">
            ×
          </button>
        </header>

        {created ? (
          <div className={styles.sheetBody}>
            <div className={styles.successPanel}>
              <span className={styles.successIcon}>✓</span>
              <div>
                <strong>{created.staff.full_name}</strong>
                <span>{created.staff.email}</span>
              </div>
            </div>

            <div className={styles.activationCard}>
              <div className={styles.activationHeader}>
                <div>
                  <span className={styles.metaLabel}>Activation link</span>
                  <strong>Expires {new Date(created.invitation.expires_at).toLocaleString()}</strong>
                </div>
                <span className={styles.statusBadge} data-tone="warning">Pending activation</span>
              </div>

              <div className={styles.copyField}>
                <code>{inviteUrl}</code>
                <button type="button" onClick={() => void copy(inviteUrl, "link")}>
                  {copied === "link" ? "Copied" : "Copy link"}
                </button>
              </div>

              <div className={styles.copyField}>
                <code>{created.invitation.activation_token}</code>
                <button
                  type="button"
                  onClick={() => void copy(created.invitation.activation_token, "token")}
                >
                  {copied === "token" ? "Copied" : "Copy token"}
                </button>
              </div>

              <p className={styles.securityNote}>
                The token is not stored in plaintext and cannot be retrieved later. If it is lost or expires, reissue the invitation from the staff record.
              </p>
            </div>

            <div className={styles.sheetFooter}>
              <button type="button" className={styles.primaryButton} onClick={close}>
                Done
              </button>
            </div>
          </div>
        ) : (
          <form className={styles.sheetBody} onSubmit={submit}>
            <div className={styles.formGrid}>
              <label className={styles.field}>
                <span>Full name</span>
                <input
                  value={fullName}
                  onChange={(event) => setFullName(event.target.value)}
                  maxLength={160}
                  autoComplete="name"
                  required
                />
              </label>

              <label className={styles.field}>
                <span>Work email</span>
                <input
                  type="email"
                  value={email}
                  onChange={(event) => setEmail(event.target.value)}
                  maxLength={255}
                  autoComplete="email"
                  required
                />
              </label>

              <label className={styles.field}>
                <span>Phone <small>optional</small></span>
                <input
                  value={phone}
                  onChange={(event) => setPhone(event.target.value)}
                  maxLength={40}
                  autoComplete="tel"
                />
              </label>
            </div>

            <div className={styles.formSection}>
              <div className={styles.sectionHeading}>
                <div>
                  <span className={styles.metaLabel}>Starting access</span>
                  <h3>Assign at least one role</h3>
                </div>
                <span>{roleCodes.length} selected</span>
              </div>

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
                      <span>
                        <strong>{role.name}</strong>
                        <small>{role.description || role.code}</small>
                      </span>
                    </button>
                  );
                })}
              </div>
            </div>

            <div className={styles.deliveryNotice}>
              <span>Manual delivery</span>
              <p>
                Email invitation delivery is not configured yet. A one-time activation link will be generated after creation.
              </p>
            </div>

            {error ? <div className={styles.errorBanner} role="alert">{error}</div> : null}

            <div className={styles.sheetFooter}>
              <button type="button" className={styles.secondaryButton} onClick={close} disabled={busy}>
                Cancel
              </button>
              <button
                type="submit"
                className={styles.primaryButton}
                disabled={busy || !fullName.trim() || !email.trim() || roleCodes.length === 0}
              >
                {busy ? "Creating…" : "Create staff account"}
              </button>
            </div>
          </form>
        )}
      </section>
    </div>
  );
}
