"use client";

import {
  type FormEvent,
  useState,
} from "react";

import AdminButton from "@/components/admin/ui/components/AdminButton";
import AdminField from "@/components/admin/ui/components/AdminField";
import AdminInput from "@/components/admin/ui/components/AdminInput";

import styles from "../css/AdminAuthForm.module.css";

type AdminLoginFormProps = {
  busy: boolean;
  error: string;
  onSubmit: (
    identifier: string,
    password: string,
  ) => Promise<void>;
};

export default function AdminLoginForm({
  busy,
  error,
  onSubmit,
}: AdminLoginFormProps) {
  const [identifier, setIdentifier] =
    useState("");

  const [password, setPassword] =
    useState("");

  async function handleSubmit(
    event: FormEvent<HTMLFormElement>,
  ) {
    event.preventDefault();

    if (busy) {
      return;
    }

    await onSubmit(
      identifier.trim(),
      password,
    );
  }

  return (
    <form
      className={styles.form}
      onSubmit={handleSubmit}
    >
      <AdminField
        label="Email or identifier"
        required
      >
        <AdminInput
          type="text"
          autoComplete="username"
          autoFocus
          value={identifier}
          onChange={(event) =>
            setIdentifier(
              event.target.value,
            )
          }
          disabled={busy}
          required
        />
      </AdminField>

      <AdminField
        label="Password"
        required
      >
        <AdminInput
          type="password"
          autoComplete="current-password"
          value={password}
          onChange={(event) =>
            setPassword(
              event.target.value,
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

      <AdminButton
        type="submit"
        fullWidth
        disabled={busy}
      >
        {busy
          ? "Authenticating…"
          : "Continue securely"}
      </AdminButton>
    </form>
  );
}