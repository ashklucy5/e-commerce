import type { ReactNode } from "react";

import styles from "../css/AdminField.module.css";

type AdminFieldProps = {
  label: string;
  children: ReactNode;
  hint?: string;
  error?: string;
  required?: boolean;
  className?: string;
};

export default function AdminField({
  label,
  children,
  hint,
  error,
  required = false,
  className = "",
}: AdminFieldProps) {
  return (
    <label
      className={[
        styles.field,
        className,
      ]
        .filter(Boolean)
        .join(" ")}
    >
      <span className={styles.label}>
        {label}

        {required ? (
          <span className={styles.required}>
            *
          </span>
        ) : null}
      </span>

      {children}

      {error ? (
        <span className={styles.error}>
          {error}
        </span>
      ) : hint ? (
        <span className={styles.hint}>
          {hint}
        </span>
      ) : null}
    </label>
  );
}