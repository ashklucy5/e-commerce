import type {
  InputHTMLAttributes,
} from "react";

import styles from "../css/AdminInput.module.css";

type AdminInputProps =
  InputHTMLAttributes<HTMLInputElement> & {
    invalid?: boolean;
  };

export default function AdminInput({
  invalid = false,
  className = "",
  ...props
}: AdminInputProps) {
  return (
    <input
      {...props}
      className={[
        styles.input,
        invalid
          ? styles.invalid
          : "",
        className,
      ]
        .filter(Boolean)
        .join(" ")}
    />
  );
}