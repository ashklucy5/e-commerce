"use client";

import {
  type FormEvent,
  useCallback,
  useEffect,
  useMemo,
  useState,
} from "react";
import { useRouter } from "next/navigation";

import {
  adminFetch,
  AdminRequestError,
} from "@/lib/admin/api";
import type {
  AdminBeginMfaRotationResponse,
  AdminChangePasswordResponse,
  AdminRevokeOtherSessionsResponse,
  AdminSecurityStepUpMethod,
  AdminSelfSecurityMutationResponse,
  AdminSelfSecurityOverview,
  AdminSelfSecurityOverviewResponse,
  AdminSelfSecuritySession,
} from "@/lib/admin/account-types";
import type { AdminMfaEnrollment } from "@/lib/admin/types";
import { useAdminSession } from "@/components/admin/layout/context/AdminSessionContext";

import styles from "../css/AdminAccountSecurity.module.css";

type AdminAccountSecurityProps = {
  portal: string;
};

type StepUpState = {
  password: string;
  method: AdminSecurityStepUpMethod;
  code: string;
};

type RotationState = {
  token: string;
  expiresAt: string;
  enrollment: AdminMfaEnrollment;
} | null;

const emptyStepUp = (): StepUpState => ({
  password: "",
  method: "totp",
  code: "",
});

function formatDate(value: string | null): string {
  if (!value) {
    return "Not available";
  }

  const date = new Date(value);

  if (Number.isNaN(date.getTime())) {
    return "Not available";
  }

  return new Intl.DateTimeFormat(undefined, {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(date);
}

function formatRole(role: string): string {
  const known: Record<string, string> = {
    admin_superuser: "Super Admin",
    admin_administrator: "Administrator",
    admin_security: "Security",
    admin_sourcing: "Sourcing",
    admin_catalog: "Catalog",
    admin_inventory: "Inventory",
    admin_warehouse: "Warehouse",
    admin_fulfillment: "Fulfillment",
    admin_finance: "Finance",
    admin_returns: "Returns",
    admin_analytics: "Analytics",
    support_agent: "Support Agent",
    support_supervisor: "Support Supervisor",
  };

  if (known[role]) {
    return known[role];
  }

  return role
    .replace(/^admin_/, "")
    .replace(/_/g, " ")
    .replace(/\b\w/g, (letter) => letter.toUpperCase());
}

function cleanCode(
  value: string,
  method: AdminSecurityStepUpMethod,
): string {
  if (method === "totp") {
    return value.replace(/\D/g, "").slice(0, 6);
  }

  return value.trimStart().slice(0, 128);
}

function actionError(error: unknown): string {
  if (error instanceof AdminRequestError) {
    return error.message;
  }

  if (error instanceof Error) {
    return error.message;
  }

  return "The security operation could not be completed.";
}

function shortUserAgent(value: string): string {
  const clean = value.trim();

  if (!clean) {
    return "Unknown client";
  }

  if (clean.length <= 92) {
    return clean;
  }

  return `${clean.slice(0, 89)}…`;
}

export default function AdminAccountSecurity({
  portal,
}: AdminAccountSecurityProps) {
  const router = useRouter();
  const principal = useAdminSession();

  const [overview, setOverview] =
    useState<AdminSelfSecurityOverview | null>(null);
  const [loading, setLoading] = useState(true);
  const [pageError, setPageError] = useState("");
  const [notice, setNotice] = useState("");
  const [busy, setBusy] = useState<string | null>(null);
  const [recoveryCodes, setRecoveryCodes] =
    useState<string[] | null>(null);
  const [codesCopied, setCodesCopied] = useState(false);

  const [passwordStepUp, setPasswordStepUp] =
    useState<StepUpState>(emptyStepUp);
  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");

  const [recoveryStepUp, setRecoveryStepUp] =
    useState<StepUpState>(emptyStepUp);

  const [rotationStepUp, setRotationStepUp] =
    useState<StepUpState>(emptyStepUp);
  const [rotationLabel, setRotationLabel] =
    useState("Authenticator");
  const [rotation, setRotation] =
    useState<RotationState>(null);
  const [rotationCode, setRotationCode] = useState("");
  const [secretCopied, setSecretCopied] = useState(false);

  const loadOverview = useCallback(async (signal?: AbortSignal) => {
    try {
      const response =
        await adminFetch<AdminSelfSecurityOverviewResponse>(
          "/auth/security/overview",
          { signal },
        );

      if (signal?.aborted) {
        return;
      }

      setOverview(response.data);
      setPageError("");
    } catch (error) {
      if (signal?.aborted) {
        return;
      }

      setPageError(actionError(error));
    } finally {
      if (!signal?.aborted) {
        setLoading(false);
      }
    }
  }, []);

  useEffect(() => {
    const controller = new AbortController();
    void loadOverview(controller.signal);

    return () => controller.abort();
  }, [loadOverview]);

  const currentSession = useMemo(
    () => overview?.sessions.find((session) => session.current) ?? null,
    [overview],
  );

  const otherSessionCount = useMemo(
    () => overview?.sessions.filter((session) => !session.current).length ?? 0,
    [overview],
  );

  function beginAction(name: string) {
    setBusy(name);
    setPageError("");
    setNotice("");
  }

  function endAction() {
    setBusy(null);
  }

  async function handleChangePassword(
    event: FormEvent<HTMLFormElement>,
  ) {
    event.preventDefault();

    if (newPassword !== confirmPassword) {
      setPageError("The new passwords do not match.");
      return;
    }

    if (newPassword.length < 16) {
      setPageError("The new password must contain at least 16 characters.");
      return;
    }

    beginAction("password");

    try {
      await adminFetch<AdminChangePasswordResponse>(
        "/auth/security/password/change",
        {
          method: "POST",
          body: JSON.stringify({
            password: passwordStepUp.password,
            method: passwordStepUp.method,
            code: passwordStepUp.code,
            new_password: newPassword,
          }),
        },
      );

      setPasswordStepUp(emptyStepUp());
      setNewPassword("");
      setConfirmPassword("");
      setNotice(
        "Password changed. Other Admin and staff sessions were revoked and this session was rotated.",
      );

      await loadOverview();
    } catch (error) {
      setPageError(actionError(error));
    } finally {
      endAction();
    }
  }

  async function handleRegenerateRecoveryCodes(
    event: FormEvent<HTMLFormElement>,
  ) {
    event.preventDefault();
    beginAction("recovery");

    try {
      const response =
        await adminFetch<AdminSelfSecurityMutationResponse>(
          "/auth/security/recovery-codes/regenerate",
          {
            method: "POST",
            body: JSON.stringify({
              password: recoveryStepUp.password,
              method: recoveryStepUp.method,
              code: recoveryStepUp.code,
            }),
          },
        );

      setRecoveryStepUp(emptyStepUp());
      setRecoveryCodes(response.data.recovery_codes);
      setNotice(
        "New recovery codes generated. Previous recovery codes are no longer valid.",
      );

      await loadOverview();
    } catch (error) {
      setPageError(actionError(error));
    } finally {
      endAction();
    }
  }

  async function handleBeginRotation(
    event: FormEvent<HTMLFormElement>,
  ) {
    event.preventDefault();
    beginAction("rotation-begin");

    try {
      const response =
        await adminFetch<AdminBeginMfaRotationResponse>(
          "/auth/security/mfa/rotation/begin",
          {
            method: "POST",
            body: JSON.stringify({
              password: rotationStepUp.password,
              method: rotationStepUp.method,
              code: rotationStepUp.code,
              label: rotationLabel.trim() || "Authenticator",
            }),
          },
        );

      setRotation({
        token: response.data.rotation_token,
        expiresAt: response.data.rotation_expires_at,
        enrollment: response.data.enrollment,
      });
      setRotationCode("");
      setNotice(
        "New authenticator setup created. Your existing authenticator remains active until you confirm the new code.",
      );
    } catch (error) {
      setPageError(actionError(error));
    } finally {
      endAction();
    }
  }

  async function handleConfirmRotation(
    event: FormEvent<HTMLFormElement>,
  ) {
    event.preventDefault();

    if (!rotation) {
      return;
    }

    beginAction("rotation-confirm");

    try {
      const response =
        await adminFetch<AdminSelfSecurityMutationResponse>(
          "/auth/security/mfa/rotation/confirm",
          {
            method: "POST",
            body: JSON.stringify({
              rotation_token: rotation.token,
              code: rotationCode,
            }),
          },
        );

      setRecoveryCodes(response.data.recovery_codes);
      setRotation(null);
      setRotationStepUp(emptyStepUp());
      setRotationCode("");
      setNotice(
        "Authenticator changed successfully. Save the new recovery codes before leaving this page.",
      );

      await loadOverview();
    } catch (error) {
      setPageError(actionError(error));
    } finally {
      endAction();
    }
  }

  async function handleRevokeSession(
    session: AdminSelfSecuritySession,
  ) {
    if (session.current) {
      return;
    }

    beginAction(`session:${session.id}`);

    try {
      await adminFetch<void>(
        `/auth/security/sessions/${encodeURIComponent(session.id)}`,
        { method: "DELETE" },
      );

      setNotice("Session revoked.");
      await loadOverview();
    } catch (error) {
      setPageError(actionError(error));
    } finally {
      endAction();
    }
  }

  async function handleRevokeOthers() {
    beginAction("revoke-others");

    try {
      const response =
        await adminFetch<AdminRevokeOtherSessionsResponse>(
          "/auth/security/sessions/revoke-others",
          { method: "POST" },
        );

      setNotice(
        response.data.revoked_sessions === 1
          ? "1 other session revoked."
          : `${response.data.revoked_sessions} other sessions revoked.`,
      );

      await loadOverview();
    } catch (error) {
      setPageError(actionError(error));
    } finally {
      endAction();
    }
  }

  async function handleLogout() {
    beginAction("logout");

    try {
      await adminFetch<void>(
        "/auth/logout",
        { method: "POST" },
      );

      router.replace(`/${portal}/login`);
      router.refresh();
    } catch (error) {
      setPageError(actionError(error));
      endAction();
    }
  }

  async function copyRecoveryCodes() {
    if (!recoveryCodes) {
      return;
    }

    try {
      await navigator.clipboard.writeText(recoveryCodes.join("\n"));
      setCodesCopied(true);
      window.setTimeout(() => setCodesCopied(false), 1600);
    } catch {
      setPageError("Clipboard access is unavailable. Copy the recovery codes manually.");
    }
  }

  async function copySecret() {
    if (!rotation) {
      return;
    }

    try {
      await navigator.clipboard.writeText(rotation.enrollment.secret);
      setSecretCopied(true);
      window.setTimeout(() => setSecretCopied(false), 1600);
    } catch {
      setPageError("Clipboard access is unavailable. Copy the setup key manually.");
    }
  }

  return (
    <div className={styles.page}>
      <header className={styles.hero}>
        <div className={styles.heroCopy}>
          <p className={styles.eyebrow}>Personal security</p>
          <h1 className={styles.title}>Account & security</h1>
          <p className={styles.description}>
            Manage your password, authenticator, recovery codes and active Admin sessions.
          </p>
        </div>

        <div className={styles.heroStatus}>
          <span className={styles.heroStatusDot} aria-hidden="true" />
          <span>Protected session</span>
        </div>
      </header>

      {pageError ? (
        <div className={styles.errorBanner} role="alert">
          <strong>Security action needs attention.</strong>
          <span>{pageError}</span>
        </div>
      ) : null}

      {notice ? (
        <div className={styles.successBanner} role="status">
          <strong>Updated.</strong>
          <span>{notice}</span>
        </div>
      ) : null}

      <section className={styles.profileGrid}>
        <article className={styles.profileCard}>
          <div className={styles.avatar} aria-hidden="true">
            {principal.staff.full_name
              .split(/\s+/)
              .filter(Boolean)
              .slice(0, 2)
              .map((part) => part[0])
              .join("")
              .toUpperCase() || "ED"}
          </div>

          <div className={styles.profileCopy}>
            <p className={styles.cardLabel}>Signed in as</p>
            <h2>{principal.staff.full_name}</h2>
            <p>{principal.staff.email}</p>
          </div>

          <span className={styles.statusPill}>
            {principal.staff.status}
          </span>
        </article>

        <article className={styles.identityCard}>
          <div>
            <span className={styles.metricLabel}>Staff code</span>
            <strong>{principal.staff.staff_code}</strong>
          </div>
          <div>
            <span className={styles.metricLabel}>Roles</span>
            <strong>
              {principal.staff.roles.length > 0
                ? principal.staff.roles.map(formatRole).join(" · ")
                : "Staff"}
            </strong>
          </div>
          <div>
            <span className={styles.metricLabel}>Permissions</span>
            <strong>{principal.staff.permissions.length}</strong>
          </div>
        </article>
      </section>

      <section className={styles.securitySummary} aria-label="Security overview">
        <article className={styles.summaryCard}>
          <span className={styles.summaryIconPositive} aria-hidden="true">✓</span>
          <div>
            <p>Authenticator</p>
            <strong>
              {loading
                ? "Checking…"
                : overview?.mfa.enabled
                  ? "Enabled"
                  : "Not enabled"}
            </strong>
            <span>{overview?.mfa.label || "MFA credential"}</span>
          </div>
        </article>

        <article className={styles.summaryCard}>
          <span className={styles.summaryIconInfo} aria-hidden="true">#</span>
          <div>
            <p>Recovery codes</p>
            <strong>
              {loading
                ? "Checking…"
                : `${overview?.mfa.recovery_codes_remaining ?? 0} remaining`}
            </strong>
            <span>Single-use backup verification</span>
          </div>
        </article>

        <article className={styles.summaryCard}>
          <span className={styles.summaryIconViolet} aria-hidden="true">◎</span>
          <div>
            <p>Active sessions</p>
            <strong>
              {loading
                ? "Checking…"
                : `${overview?.sessions.length ?? 0} active`}
            </strong>
            <span>{otherSessionCount} on other devices</span>
          </div>
        </article>
      </section>

      {recoveryCodes ? (
        <section className={styles.recoveryReveal} aria-labelledby="recovery-codes-title">
          <div className={styles.sectionHeading}>
            <div>
              <p className={styles.sectionEyebrow}>One-time reveal</p>
              <h2 id="recovery-codes-title">Save your new recovery codes</h2>
              <p>
                These codes are only returned now. Store them somewhere private before dismissing this panel.
              </p>
            </div>

            <div className={styles.headingActions}>
              <button
                type="button"
                className={styles.secondaryButton}
                onClick={() => void copyRecoveryCodes()}
              >
                {codesCopied ? "Copied" : "Copy codes"}
              </button>
              <button
                type="button"
                className={styles.primaryButton}
                onClick={() => setRecoveryCodes(null)}
              >
                I saved them
              </button>
            </div>
          </div>

          <div className={styles.recoveryCodeGrid}>
            {recoveryCodes.map((code) => (
              <code key={code}>{code}</code>
            ))}
          </div>
        </section>
      ) : null}

      <section className={styles.section}>
        <div className={styles.sectionHeading}>
          <div>
            <p className={styles.sectionEyebrow}>Sessions</p>
            <h2>Where you are signed in</h2>
            <p>
              Review Admin sessions and revoke devices you no longer recognize or use.
            </p>
          </div>

          <button
            type="button"
            className={styles.secondaryButton}
            disabled={otherSessionCount === 0 || busy === "revoke-others"}
            onClick={() => void handleRevokeOthers()}
          >
            {busy === "revoke-others"
              ? "Revoking…"
              : "Revoke other sessions"}
          </button>
        </div>

        <div className={styles.sessionList}>
          {loading ? (
            <div className={styles.loadingCard}>Loading active sessions…</div>
          ) : overview?.sessions.length ? (
            overview.sessions.map((session) => (
              <article className={styles.sessionRow} key={session.id}>
                <div className={styles.sessionPrimary}>
                  <div className={styles.sessionTitleLine}>
                    <strong>{session.current ? "This device" : "Admin session"}</strong>
                    {session.current ? (
                      <span className={styles.currentPill}>Current</span>
                    ) : null}
                  </div>
                  <p title={session.user_agent}>
                    {shortUserAgent(session.user_agent)}
                  </p>
                </div>

                <div className={styles.sessionFact}>
                  <span>Last active</span>
                  <strong>{formatDate(session.last_used_at)}</strong>
                </div>

                <div className={styles.sessionFact}>
                  <span>Last IP</span>
                  <strong>{session.last_ip || "Unknown"}</strong>
                </div>

                <div className={styles.sessionFact}>
                  <span>Refresh expires</span>
                  <strong>{formatDate(session.refresh_expires_at)}</strong>
                </div>

                <div className={styles.sessionAction}>
                  {session.current ? (
                    <button
                      type="button"
                      className={styles.ghostButton}
                      disabled={busy === "logout"}
                      onClick={() => void handleLogout()}
                    >
                      {busy === "logout" ? "Signing out…" : "Sign out"}
                    </button>
                  ) : (
                    <button
                      type="button"
                      className={styles.dangerGhostButton}
                      disabled={busy === `session:${session.id}`}
                      onClick={() => void handleRevokeSession(session)}
                    >
                      {busy === `session:${session.id}` ? "Revoking…" : "Revoke"}
                    </button>
                  )}
                </div>
              </article>
            ))
          ) : (
            <div className={styles.loadingCard}>No active Admin sessions were returned.</div>
          )}
        </div>

        {currentSession ? (
          <p className={styles.sectionFootnote}>
            Current session authenticated {formatDate(currentSession.authenticated_at)} · MFA verified {formatDate(currentSession.mfa_verified_at)}.
          </p>
        ) : null}
      </section>

      <section className={styles.actionGrid}>
        <details className={styles.actionCard}>
          <summary>
            <div>
              <span className={styles.actionIconBlue} aria-hidden="true">↗</span>
              <div>
                <h2>Change password</h2>
                <p>Rotates this session and revokes every other staff/Admin session.</p>
              </div>
            </div>
            <span className={styles.disclosure} aria-hidden="true">+</span>
          </summary>

          <form className={styles.actionForm} onSubmit={handleChangePassword}>
            <StepUpFields
              state={passwordStepUp}
              setState={setPasswordStepUp}
              disabled={busy === "password"}
            />

            <label className={styles.field}>
              <span>New password</span>
              <input
                type="password"
                minLength={16}
                maxLength={256}
                autoComplete="new-password"
                value={newPassword}
                onChange={(event) => setNewPassword(event.target.value)}
                disabled={busy === "password"}
                required
              />
              <small>Use at least 16 characters and avoid common passwords.</small>
            </label>

            <label className={styles.field}>
              <span>Confirm new password</span>
              <input
                type="password"
                minLength={16}
                maxLength={256}
                autoComplete="new-password"
                value={confirmPassword}
                onChange={(event) => setConfirmPassword(event.target.value)}
                disabled={busy === "password"}
                required
              />
            </label>

            <button
              className={styles.primaryButton}
              type="submit"
              disabled={busy === "password"}
            >
              {busy === "password" ? "Changing password…" : "Change password"}
            </button>
          </form>
        </details>

        <details className={styles.actionCard}>
          <summary>
            <div>
              <span className={styles.actionIconViolet} aria-hidden="true">◉</span>
              <div>
                <h2>Change authenticator</h2>
                <p>Prepare a new authenticator without disabling the current one first.</p>
              </div>
            </div>
            <span className={styles.disclosure} aria-hidden="true">+</span>
          </summary>

          {!rotation ? (
            <form className={styles.actionForm} onSubmit={handleBeginRotation}>
              <StepUpFields
                state={rotationStepUp}
                setState={setRotationStepUp}
                disabled={busy === "rotation-begin"}
              />

              <label className={styles.field}>
                <span>New authenticator label</span>
                <input
                  type="text"
                  maxLength={120}
                  value={rotationLabel}
                  onChange={(event) => setRotationLabel(event.target.value)}
                  disabled={busy === "rotation-begin"}
                />
              </label>

              <p className={styles.formNote}>
                If you verify with a recovery code, that code is consumed when this setup begins.
              </p>

              <button
                className={styles.primaryButton}
                type="submit"
                disabled={busy === "rotation-begin"}
              >
                {busy === "rotation-begin" ? "Preparing…" : "Prepare new authenticator"}
              </button>
            </form>
          ) : (
            <form className={styles.actionForm} onSubmit={handleConfirmRotation}>
              <div className={styles.setupKeyBox}>
                <div>
                  <span>New setup key</span>
                  <code>{rotation.enrollment.secret}</code>
                </div>
                <button
                  type="button"
                  className={styles.secondaryButton}
                  onClick={() => void copySecret()}
                >
                  {secretCopied ? "Copied" : "Copy key"}
                </button>
              </div>

              <div className={styles.setupMeta}>
                <span>{rotation.enrollment.algorithm}</span>
                <span>{rotation.enrollment.digits} digits</span>
                <span>{rotation.enrollment.period_seconds}s period</span>
                <span>Expires {formatDate(rotation.expiresAt)}</span>
              </div>

              <label className={styles.field}>
                <span>Code from the new authenticator</span>
                <input
                  type="text"
                  inputMode="numeric"
                  autoComplete="one-time-code"
                  pattern="[0-9]{6}"
                  maxLength={6}
                  value={rotationCode}
                  onChange={(event) =>
                    setRotationCode(event.target.value.replace(/\D/g, "").slice(0, 6))
                  }
                  disabled={busy === "rotation-confirm"}
                  required
                />
              </label>

              <div className={styles.inlineActions}>
                <button
                  type="button"
                  className={styles.secondaryButton}
                  disabled={busy === "rotation-confirm"}
                  onClick={() => {
                    setRotation(null);
                    setRotationCode("");
                  }}
                >
                  Cancel setup
                </button>
                <button
                  type="submit"
                  className={styles.primaryButton}
                  disabled={busy === "rotation-confirm" || rotationCode.length !== 6}
                >
                  {busy === "rotation-confirm" ? "Confirming…" : "Activate new authenticator"}
                </button>
              </div>
            </form>
          )}
        </details>

        <details className={styles.actionCard}>
          <summary>
            <div>
              <span className={styles.actionIconGold} aria-hidden="true">#</span>
              <div>
                <h2>Regenerate recovery codes</h2>
                <p>Invalidates all existing recovery codes and returns a fresh set once.</p>
              </div>
            </div>
            <span className={styles.disclosure} aria-hidden="true">+</span>
          </summary>

          <form className={styles.actionForm} onSubmit={handleRegenerateRecoveryCodes}>
            <StepUpFields
              state={recoveryStepUp}
              setState={setRecoveryStepUp}
              disabled={busy === "recovery"}
            />

            <p className={styles.formNote}>
              Save the new codes immediately. Existing codes stop working as soon as regeneration succeeds.
            </p>

            <button
              className={styles.primaryButton}
              type="submit"
              disabled={busy === "recovery"}
            >
              {busy === "recovery" ? "Generating…" : "Generate new recovery codes"}
            </button>
          </form>
        </details>
      </section>

      <section className={styles.securityDetails}>
        <div>
          <span>MFA last used</span>
          <strong>{formatDate(overview?.mfa.last_used_at ?? null)}</strong>
        </div>
        <div>
          <span>MFA verified</span>
          <strong>{formatDate(overview?.mfa.verified_at ?? null)}</strong>
        </div>
        <div>
          <span>Current access expires</span>
          <strong>{formatDate(principal.access_expires_at)}</strong>
        </div>
      </section>
    </div>
  );
}

type StepUpFieldsProps = {
  state: StepUpState;
  setState: (value: StepUpState) => void;
  disabled: boolean;
};

function StepUpFields({
  state,
  setState,
  disabled,
}: StepUpFieldsProps) {
  return (
    <>
      <label className={styles.field}>
        <span>Current password</span>
        <input
          type="password"
          autoComplete="current-password"
          value={state.password}
          onChange={(event) =>
            setState({ ...state, password: event.target.value })
          }
          disabled={disabled}
          required
        />
      </label>

      <div className={styles.stepUpGrid}>
        <label className={styles.field}>
          <span>Verification method</span>
          <select
            value={state.method}
            onChange={(event) => {
              const method = event.target.value as AdminSecurityStepUpMethod;
              setState({
                ...state,
                method,
                code: "",
              });
            }}
            disabled={disabled}
          >
            <option value="totp">Authenticator code</option>
            <option value="recovery_code">Recovery code</option>
          </select>
        </label>

        <label className={styles.field}>
          <span>
            {state.method === "totp"
              ? "Current authenticator code"
              : "Unused recovery code"}
          </span>
          <input
            type="text"
            inputMode={state.method === "totp" ? "numeric" : "text"}
            autoComplete={state.method === "totp" ? "one-time-code" : "off"}
            pattern={state.method === "totp" ? "[0-9]{6}" : undefined}
            maxLength={state.method === "totp" ? 6 : 128}
            value={state.code}
            onChange={(event) =>
              setState({
                ...state,
                code: cleanCode(event.target.value, state.method),
              })
            }
            disabled={disabled}
            required
          />
        </label>
      </div>
    </>
  );
}
