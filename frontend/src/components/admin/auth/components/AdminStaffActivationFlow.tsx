"use client";

import {
  type FormEvent,
  useEffect,
  useState,
} from "react";
import { useRouter, useSearchParams } from "next/navigation";

import AdminButton from "@/components/admin/ui/components/AdminButton";
import AdminField from "@/components/admin/ui/components/AdminField";
import AdminInput from "@/components/admin/ui/components/AdminInput";
import {
  AdminRequestError,
  adminFetch,
} from "@/lib/admin/api";
import type {
  AdminStaffActivationCompleteResponse,
  AdminStaffActivationMfaEnrollResponse,
  AdminStaffActivationPasswordResponse,
} from "@/lib/admin/staff-types";
import type { AdminMfaEnrollment } from "@/lib/admin/types";

import AdminAuthFrame from "./AdminAuthFrame";
import AdminEnrollmentForm from "./AdminEnrollmentForm";
import AdminRecoveryCodes from "./AdminRecoveryCodes";

import styles from "../css/AdminStaffActivation.module.css";

type Props = {
  portal: string;
};

type Stage = "loading" | "password" | "mfa" | "recovery" | "invalid";

function errorMessage(value: unknown): string {
  if (value instanceof AdminRequestError) return value.message;
  if (value instanceof Error) return value.message;
  return "The activation request could not be completed.";
}

function activationStorageKey(portal: string, email: string): string {
  return `ene-admin-activation:${portal}:${email.toLowerCase()}`;
}

export default function AdminStaffActivationFlow({ portal }: Props) {
  const router = useRouter();
  const searchParams = useSearchParams();

  const [stage, setStage] = useState<Stage>("loading");
  const [email, setEmail] = useState("");
  const [token, setToken] = useState("");
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [enrollment, setEnrollment] = useState<AdminMfaEnrollment | null>(null);
  const [recoveryCodes, setRecoveryCodes] = useState<string[]>([]);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    const inviteEmail = searchParams.get("email")?.trim() ?? "";
    const hashParams = new URLSearchParams(window.location.hash.replace(/^#/, ""));
    const hashToken = hashParams.get("token")?.trim() ?? "";
    const storageKey = inviteEmail ? activationStorageKey(portal, inviteEmail) : "";
    const storedToken = storageKey ? window.sessionStorage.getItem(storageKey)?.trim() ?? "" : "";
    const activationToken = hashToken || storedToken;

    if (!inviteEmail || !activationToken) {
      setStage("invalid");
      return;
    }

    if (hashToken && storageKey) {
      window.sessionStorage.setItem(storageKey, hashToken);
    }

    setEmail(inviteEmail);
    setToken(activationToken);
    setStage("password");

    // Keep the token client-side for refresh recovery, but remove it from the visible URL.
    const cleanUrl = new URL(window.location.href);
    cleanUrl.hash = "";
    window.history.replaceState(null, "", cleanUrl.toString());
  }, [searchParams]);

  async function beginMfa() {
    const response = await adminFetch<AdminStaffActivationMfaEnrollResponse>(
      "/auth/activate/mfa/enroll",
      {
        method: "POST",
        body: JSON.stringify({
          email,
          activation_token: token,
          label: "Ene dei Operations",
        }),
      },
    );

    setEnrollment(response.data.enrollment);
    setStage("mfa");
  }

  async function submitPassword(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();

    if (busy || !password || password !== confirmPassword) return;

    setBusy(true);
    setError("");

    try {
      await adminFetch<AdminStaffActivationPasswordResponse>(
        "/auth/activate/password",
        {
          method: "POST",
          body: JSON.stringify({
            email,
            activation_token: token,
            password,
          }),
        },
      );

      setPassword("");
      setConfirmPassword("");
      await beginMfa();
    } catch (value: unknown) {
      if (
        value instanceof AdminRequestError &&
        value.code === "STAFF_ACTIVATION_PASSWORD_ALREADY_SET"
      ) {
        try {
          await beginMfa();
          return;
        } catch (nested: unknown) {
          setError(errorMessage(nested));
          return;
        }
      }
      setError(errorMessage(value));
    } finally {
      setBusy(false);
    }
  }

  async function confirmMfa(code: string) {
    setBusy(true);
    setError("");

    try {
      const response = await adminFetch<AdminStaffActivationCompleteResponse>(
        "/auth/activate/mfa/confirm",
        {
          method: "POST",
          body: JSON.stringify({
            email,
            activation_token: token,
            code,
          }),
        },
      );

      setRecoveryCodes(response.data.recovery_codes ?? []);
      window.sessionStorage.removeItem(activationStorageKey(portal, email));
      setStage("recovery");
    } catch (value: unknown) {
      setError(errorMessage(value));
    } finally {
      setBusy(false);
    }
  }

  function goToLogin() {
    if (email) {
      window.sessionStorage.removeItem(activationStorageKey(portal, email));
    }
    router.replace(`/${portal}/login`);
    router.refresh();
  }

  if (stage === "loading") {
    return (
      <AdminAuthFrame
        eyebrow="Staff activation"
        title="Preparing your account."
        description="Validating the one-time activation package."
      >
        <div className={styles.loading}>Preparing secure activation…</div>
      </AdminAuthFrame>
    );
  }

  if (stage === "invalid") {
    return (
      <AdminAuthFrame
        eyebrow="Staff activation"
        title="Activation link required."
        description="Open the complete activation link supplied by your administrator."
      >
        <div className={styles.invalidPanel}>
          <p>The activation package is incomplete. Ask the administrator who created your staff account to reissue the invitation.</p>
          <AdminButton type="button" variant="secondary" onClick={goToLogin}>
            Back to sign in
          </AdminButton>
        </div>
      </AdminAuthFrame>
    );
  }

  if (stage === "mfa" && enrollment) {
    return (
      <AdminAuthFrame
        eyebrow="Mandatory security"
        title="Set up your authenticator."
        description={`Password created for ${email}. Add Ene dei Operations to your authenticator, then confirm the current code.`}
      >
        <AdminEnrollmentForm
          enrollment={enrollment}
          busy={busy}
          error={error}
          onSubmit={confirmMfa}
          onBack={() => setStage("password")}
        />
      </AdminAuthFrame>
    );
  }

  if (stage === "recovery") {
    return (
      <AdminAuthFrame
        eyebrow="Activation complete"
        title="Save your recovery codes."
        description="Your staff account is active. Store these one-time recovery codes before signing in."
      >
        <AdminRecoveryCodes codes={recoveryCodes} onContinue={goToLogin} />
      </AdminAuthFrame>
    );
  }

  return (
    <AdminAuthFrame
      eyebrow="Staff activation"
      title="Create your password."
      description={`This invitation is bound to ${email}. Choose your own password; your administrator cannot retrieve it.`}
    >
      <form className={styles.form} onSubmit={submitPassword}>
        <div className={styles.identityBox}>
          <span>Staff email</span>
          <strong>{email}</strong>
        </div>

        <AdminField label="New password" required>
          <AdminInput
            type="password"
            autoComplete="new-password"
            value={password}
            onChange={(event) => setPassword(event.target.value)}
            disabled={busy}
            required
          />
        </AdminField>

        <AdminField label="Confirm password" required>
          <AdminInput
            type="password"
            autoComplete="new-password"
            value={confirmPassword}
            onChange={(event) => setConfirmPassword(event.target.value)}
            disabled={busy}
            required
          />
        </AdminField>

        {confirmPassword && password !== confirmPassword ? (
          <div className={styles.inlineError}>Passwords do not match.</div>
        ) : null}

        {error ? <div className={styles.error} role="alert">{error}</div> : null}

        <div className={styles.actions}>
          <AdminButton
            type="submit"
            disabled={busy || !password || password !== confirmPassword}
          >
            {busy ? "Securing account…" : "Continue to MFA"}
          </AdminButton>
        </div>
      </form>
    </AdminAuthFrame>
  );
}
