import type {
  SelectHTMLAttributes,
} from "react";

import styles from "../css/AdminSelect.module.css";

type AdminSelectProps =
  SelectHTMLAttributes<HTMLSelectElement> & {
    invalid?: boolean;
  };

export default function AdminSelect({
  invalid = false,
  className = "",
  children,
  ...props
}: AdminSelectProps) {
  return (
    <select
      {...props}
      className={[
        styles.select,
        invalid
          ? styles.invalid
          : "",
        className,
      ]
        .filter(Boolean)
        .join(" ")}
    >
      {children}
    </select>
  );
}