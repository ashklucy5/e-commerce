"use client";

import {
  type FormEvent,
  useState,
} from "react";

import AdminButton from "@/components/admin/ui/components/AdminButton";
import AdminField from "@/components/admin/ui/components/AdminField";
import AdminInput from "@/components/admin/ui/components/AdminInput";

import type {
  AdminMfaEnrollment,
} from "@/lib/admin/types";

import styles from "../css/AdminAuthForm.module.css";

type AdminEnrollmentFormProps = {
  enrollment: AdminMfaEnrollment;
  busy: boolean;
  error: string;
  onSubmit: (
    code: string,
  ) => Promise<void>;
  onBack: () => void;
};

export default function AdminEnrollmentForm({
  enrollment,
  busy,
  error,
  onSubmit,
  onBack,
}: AdminEnrollmentFormProps) {
  const [code, setCode] =
    useState("");

  const [copied, setCopied] =
    useState(false);

  async function copySecret() {
    await navigator.clipboard.writeText(
      enrollment.secret,
    );

    setCopied(true);

    window.setTimeout(() => {
      setCopied(false);
    }, 1800);
  }

  async function handleSubmit(
    event: FormEvent<HTMLFormElement>,
  ) {
    event.preventDefault();

    if (
      busy ||
      code.length !== 6
    ) {
      return;
    }

    await onSubmit(code);
  }

  return (
    <form
      className={styles.form}
      onSubmit={handleSubmit}
    >
      <div className={styles.setupBox}>
        <div>
          <p className={styles.setupLabel}>
            Authenticator setup key
          </p>

          <p className={styles.secret}>
            {enrollment.secret}
          </p>
        </div>

        <AdminButton
          type="button"
          variant="secondary"
          onClick={() => {
            void copySecret();
          }}
        >
          {copied
            ? "Copied"
            : "Copy"}
        </AdminButton>
      </div>

      <div className={styles.setupMeta}>
        <span>
          {enrollment.algorithm}
        </span>

        <span>
          {enrollment.digits} digits
        </span>

        <span>
          {enrollment.period_seconds}s
        </span>
      </div>

      <p className={styles.helper}>
        Add the setup key to a TOTP
        authenticator, then enter the
        current code below.
      </p>

      <AdminField
        label="Current authenticator code"
        required
      >
        <AdminInput
          className={styles.codeInput}
          type="text"
          inputMode="numeric"
          autoComplete="one-time-code"
          pattern="[0-9]{6}"
          maxLength={6}
          value={code}
          onChange={(event) =>
            setCode(
              event.target.value
                .replace(/\D/g, "")
                .slice(0, 6),
            )
          }
          disabled={busy}
          required
        />
      </AdminField>

      {error ? (
        <div
          className={styles.error}
          role="alert"
        >
          {error}
        </div>
      ) : null}

      <div className={styles.actions}>
        <AdminButton
          type="button"
          variant="secondary"
          onClick={onBack}
          disabled={busy}
        >
          Back
        </AdminButton>

        <AdminButton
          type="submit"
          disabled={
            busy ||
            code.length !== 6
          }
        >
          {busy
            ? "Activating…"
            : "Activate MFA"}
        </AdminButton>
      </div>
    </form>
  );
}