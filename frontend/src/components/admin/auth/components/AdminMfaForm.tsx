"use client";

import {
  type FormEvent,
  useState,
} from "react";

import AdminButton from "@/components/admin/ui/components/AdminButton";
import AdminField from "@/components/admin/ui/components/AdminField";
import AdminInput from "@/components/admin/ui/components/AdminInput";

import styles from "../css/AdminAuthForm.module.css";

type AdminMfaFormProps = {
  busy: boolean;
  error: string;
  onSubmit: (
    code: string,
  ) => Promise<void>;
  onBack: () => void;
};

export default function AdminMfaForm({
  busy,
  error,
  onSubmit,
  onBack,
}: AdminMfaFormProps) {
  const [code, setCode] =
    useState("");

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
      <AdminField
        label="Authenticator code"
        hint="Enter the current 6-digit code from your authenticator."
        required
      >
        <AdminInput
          className={styles.codeInput}
          type="text"
          inputMode="numeric"
          autoComplete="one-time-code"
          pattern="[0-9]{6}"
          maxLength={6}
          autoFocus
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
            ? "Verifying…"
            : "Verify MFA"}
        </AdminButton>
      </div>
    </form>
  );
}